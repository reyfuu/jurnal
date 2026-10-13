<?php

namespace App\Services\Intake;

use App\Models\Account;
use App\Models\Document;
use App\Services\Accounting\ChartOfAccounts;
use App\Services\Accounting\JournalPoster;
use App\Services\Tax\TaxCalculator as TaxCalculator;
use App\Support\Rupiah;

/**
 * Hasil baca AI → USULAN jurnal double-entry yang sudah balance. Keluarannya
 * bukan jurnal jadi, tapi bahan untuk layar review: user masih bisa ganti akun,
 * nominal, tanggal, dan akun pembayaran sebelum posting.
 *
 * Cara menyusun sisi jurnal (flow = expense):
 *     D  tiap baris rincian  → akun beban/HPP/aset pilihan AI
 *     D  PPN                 → PPN Masukan (klien PKP) / digabung ke biaya (non-PKP)
 *     K  PPh yang kita potong → Utang PPh xx (mengurangi yang dibayar)
 *     K  diskon              → akun beban dominan (kalau ada)
 *     K  sisanya             → akun pembayaran (kas/bank/e-wallet) atau Hutang Usaha
 * flow = income dibalik: rincian di kredit, pembayaran di debit.
 *
 * Sisi pembayaran dihitung sebagai SISA (total debit − kredit lain), jadi draft
 * SELALU balance apa pun angka yang dibaca model. Ketidakcocokan dengan total
 * di dokumen tidak disembunyikan — dilaporkan sebagai warning oleh extractor.
 */
class JournalDrafter
{
    /** Mode bayar dari AI → subtype akun yang dicari di COA. */
    private const PAYMENT_SUBTYPE = [
        'cash' => 'cash',
        'bank' => 'bank',
        'ewallet' => 'ewallet',
    ];

    /**
     * @return array<int,array<string,mixed>> daftar usulan jurnal
     */
    public function draft(Document $document): array
    {
        $extraction = $document->extraction ?? [];
        if ($extraction === []) {
            return [];
        }

        $accounts = $document->client->accounts()->active()->get()->keyBy('code');
        if ($accounts->isEmpty()) {
            return [];
        }

        $transactions = $extraction['transactions'] ?? [];

        return $transactions !== []
            ? $this->fromTransactions($extraction, $accounts, $document)
            : $this->fromItems($extraction, $accounts, $document);
    }

    /**
     * Struk / invoice / catatan teks → satu jurnal.
     *
     * @param  \Illuminate\Support\Collection<string,Account>  $accounts
     * @return array<int,array<string,mixed>>
     */
    private function fromItems(array $extraction, $accounts, Document $document): array
    {
        $items = $extraction['items'] ?? [];
        if ($items === []) {
            return [];
        }

        $income = ($extraction['flow'] ?? 'expense') === 'income';
        $discount = (float) ($extraction['discount'] ?? 0);
        $vendor = $extraction['vendor'] ?? null;

        $lines = [];
        $mainSide = $income ? 'credit' : 'debit';
        $otherSide = $income ? 'debit' : 'credit';

        foreach ($items as $item) {
            $account = $accounts->get($item['account_code']);
            if (! $account) {
                continue;
            }
            $lines[] = $this->line($account, $mainSide, (float) $item['amount'], $this->memo($item), $item['confidence'] ?? null, $item['account_reason'] ?? null);
        }

        if ($lines === []) {
            return [];
        }

        $date = $extraction['document_date'] ?? now()->toDateString();
        [$taxLines, $taxEntries] = $this->taxes($extraction, $accounts, $document, $lines, $items, $income, $date);
        array_push($lines, ...$taxLines);

        // Diskon mengurangi nilai transaksi → masuk sisi berlawanan, dibebankan
        // ke akun rincian terbesar supaya tidak bikin akun baru.
        if ($discount >= 0.005) {
            $dominant = $accounts->get($this->dominantCode($items));
            if ($dominant) {
                $lines[] = $this->line($dominant, $otherSide, $discount, 'Diskon');
            }
        }

        // Sisi pembayaran = sisa, supaya jurnal dijamin balance.
        $settlement = round(
            array_sum(array_column($lines, $mainSide)) - array_sum(array_column($lines, $otherSide)),
            2,
        );
        $payAccount = $this->paymentAccount($extraction, $accounts, $income);

        if ($settlement >= 0.005 && $payAccount) {
            $lines[] = $this->line($payAccount, $otherSide, $settlement, $this->settlementMemo($extraction, $income));
        }

        $reference = $extraction['document_number'] ?? null;
        $description = trim(implode(' — ', array_filter([
            $income ? 'Penjualan' : 'Pembelian/Beban',
            $vendor,
            $reference ? "No. {$reference}" : null,
        ])));

        return [[
            'date' => $date,
            'reference' => $reference,
            'description' => $description,
            'type' => $income ? 'sale' : ($document->kind === Document::KIND_INVOICE ? 'purchase' : 'expense'),
            'fingerprint' => JournalPoster::fingerprint(
                $document->client_id,
                $date,
                $settlement,
                implode('|', [$vendor, $reference, count($items)]),
            ),
            'lines' => $lines,
            'taxes' => $taxEntries,
        ]];
    }

    /**
     * Baris jurnal + catatan pajak per jenis (docs/perpajakan/TRD.md TR-09):
     *   PPN, klien PKP     → sisi rincian, akun PPN Masukan (beli) / PPN Keluaran (jual)
     *   PPN, klien non-PKP → ditambahkan ke baris rincian terbesar (tidak bisa dikreditkan)
     *   pajak dipotong     → sisi pembayaran: Utang PPh (kita potong) / PPh Dibayar Dimuka (dipotong pelanggan)
     *   pajak lain dipungut→ sisi rincian, akun aset pajak (beli) / liabilitas pajak (jual)
     * Pembayaran tetap dihitung sebagai sisa, jadi draft tetap balance.
     *
     * @param  array<int,array<string,mixed>>  $lines  baris rincian (diubah untuk PPN non-PKP)
     * @return array{0:array<int,array<string,mixed>>,1:array<int,array<string,mixed>>}
     */
    private function taxes(array $extraction, $accounts, Document $document, array &$lines, array $items, bool $income, string $date): array
    {
        $taxes = $extraction['taxes'] ?? null;
        if ($taxes === null) {
            // Hasil baca lama (sebelum fitur pajak): satu angka "tax" = PPN.
            $legacy = (float) ($extraction['tax'] ?? 0);
            $taxes = $legacy >= 0.005 ? [['type' => 'ppn', 'label' => 'Pajak', 'amount' => $legacy, 'withheld' => false, 'base' => 0]] : [];
        }

        $calc = app(TaxCalculator::class);
        $client = $document->client;
        $mainSide = $income ? 'credit' : 'debit';
        $otherSide = $income ? 'debit' : 'credit';
        $out = [];
        $entries = [];

        foreach ($taxes as $t) {
            $amount = (int) round((float) ($t['amount'] ?? 0));
            if ($amount <= 0) {
                continue;
            }
            $code = (string) ($t['type'] ?? 'other');
            $withheld = (bool) ($t['withheld'] ?? false);
            $entry = [
                'tax_code' => $code,
                'label' => $t['label'] ?? config("pajak.codes.{$code}.name", 'Pajak'),
                'base' => (int) ($t['base'] ?? 0),
                'amount' => $amount,
                'computed' => $t['computed'] ?? null,
                'match_status' => $t['match_status'] ?? 'manual',
                'withheld' => $withheld,
                'noncreditable' => false,
                'document_number' => $t['document_number'] ?? null,
                'counterparty_name' => $extraction['vendor'] ?? null,
                'counterparty_tax_id' => $t['counterparty_tax_id'] ?? null,
            ];

            if (TaxCalculator::isVat($code) && ! $income && ! $client->isPkpOn($date)) {
                // Non-PKP tidak bisa mengkreditkan PPN → jadi bagian biaya.
                // ponytail: ditambahkan ke baris terbesar, bukan dibagi proporsional per item.
                $i = $this->largestLine($lines, $mainSide);
                $lines[$i][$mainSide] = round($lines[$i][$mainSide] + $amount, 2);
                $lines[$i]['memo'] = mb_substr($lines[$i]['memo'].' (+PPN tidak dapat dikreditkan)', 0, 255);
                $entries[] = array_replace($entry, ['account_id' => $lines[$i]['account_id'], 'direction' => 'in', 'noncreditable' => true]);

                continue;
            }

            // Kita potong (beli) / dipungut dari kita (beli) → arah 'out' kalau dipotong, 'in' kalau dipungut.
            // Penjualan dibalik: dipotong pelanggan → 'in' (kredit pajak kita), dipungut kita → 'out'.
            $direction = $income ? ($withheld ? 'in' : 'out') : ($withheld ? 'out' : 'in');
            $account = $calc->accountFor($accounts, $code, $direction);
            if (! $account) {
                continue;
            }

            $memo = $entry['label'].($withheld ? ($income ? ' dipotong pelanggan' : ' dipotong') : '');
            $out[] = $this->line($account, $withheld ? $otherSide : $mainSide, $amount, mb_substr($memo, 0, 255));
            $entries[] = $entry + ['account_id' => $account->id, 'direction' => $direction];
        }

        return [$out, $entries];
    }

    private function largestLine(array $lines, string $side): int
    {
        $best = 0;
        foreach ($lines as $i => $line) {
            if ($line[$side] > $lines[$best][$side]) {
                $best = $i;
            }
        }

        return $best;
    }

    /**
     * Mutasi rekening → SATU JURNAL PER BARIS. Dipisah supaya tiap baris punya
     * sidik jari sendiri: impor ulang screenshot yang sama tidak menggandakan,
     * dan baris yang salah bisa di-void tanpa membatalkan sebulan mutasi.
     *
     * @return array<int,array<string,mixed>>
     */
    private function fromTransactions(array $extraction, $accounts, Document $document): array
    {
        // Akun kas sisi rekening: ikut mode bayar dokumen (bank / e-wallet).
        $bank = $this->paymentAccount($extraction, $accounts, false)
            ?? $accounts->get(ChartOfAccounts::DEFAULT_CASH);
        if (! $bank) {
            return [];
        }

        $drafts = [];
        foreach ($extraction['transactions'] as $row) {
            $counter = $accounts->get($row['account_code']);
            if (! $counter) {
                continue;
            }

            $amount = (float) $row['amount'];
            $date = $row['date'] ?? $extraction['document_date'] ?? now()->toDateString();
            $in = $row['direction'] === 'in';

            // Masuk: D rekening / K akun lawan. Keluar: D akun lawan / K rekening.
            $lines = $in
                ? [
                    $this->line($bank, 'debit', $amount, $row['description']),
                    $this->line($counter, 'credit', $amount, $row['description'], $row['confidence'] ?? null, $row['account_reason'] ?? null),
                ]
                : [
                    $this->line($counter, 'debit', $amount, $row['description'], $row['confidence'] ?? null, $row['account_reason'] ?? null),
                    $this->line($bank, 'credit', $amount, $row['description']),
                ];

            $drafts[] = [
                'date' => $date,
                'reference' => null,
                'description' => $row['description'],
                'type' => 'bank',
                'fingerprint' => JournalPoster::fingerprint(
                    $document->client_id,
                    $date,
                    $amount,
                    $bank->code.'|'.($in ? 'in' : 'out').'|'.$row['description'],
                ),
                'lines' => $lines,
            ];
        }

        return $drafts;
    }

    /**
     * Akun lawan untuk pembayaran. "credit" = belum dibayar → Hutang Usaha
     * (pembelian) / Piutang Usaha (penjualan). Selain itu cari akun dengan
     * subtype yang cocok; kalau tak ada, jatuh ke Kas.
     */
    private function paymentAccount(array $extraction, $accounts, bool $income): ?Account
    {
        $method = $extraction['payment_method'] ?? 'unknown';

        if ($method === 'credit') {
            $code = $income ? '1201' : ChartOfAccounts::DEFAULT_PAYABLE;
            if ($account = $accounts->get($code)) {
                return $account;
            }
        }

        if ($subtype = self::PAYMENT_SUBTYPE[$method] ?? null) {
            if ($account = $accounts->first(fn (Account $a) => $a->subtype === $subtype)) {
                return $account;
            }
        }

        return $accounts->get(ChartOfAccounts::DEFAULT_CASH)
            ?? $accounts->first(fn (Account $a) => in_array($a->subtype, Account::PAYMENT_SUBTYPES, true));
    }

    private function settlementMemo(array $extraction, bool $income): string
    {
        return match ($extraction['payment_method'] ?? 'unknown') {
            'credit' => $income ? 'Belum diterima (piutang)' : 'Belum dibayar (hutang)',
            'cash' => 'Dibayar tunai',
            'bank' => $income ? 'Masuk rekening bank' : 'Dibayar via transfer bank',
            'ewallet' => $income ? 'Masuk e-wallet' : 'Dibayar via e-wallet/QRIS',
            default => $income ? 'Penerimaan' : 'Pembayaran',
        };
    }

    /** Kode akun dengan nominal terbesar — sasaran pembebanan diskon. */
    private function dominantCode(array $items): string
    {
        $byCode = [];
        foreach ($items as $item) {
            $byCode[$item['account_code']] = ($byCode[$item['account_code']] ?? 0) + (float) $item['amount'];
        }
        arsort($byCode);

        return (string) array_key_first($byCode);
    }

    private function memo(array $item): string
    {
        $memo = (string) $item['description'];
        if (! empty($item['qty'])) {
            $memo .= ' ('.Rupiah::format((float) $item['qty']).($item['unit'] ? ' '.$item['unit'] : 'x').')';
        }

        return mb_substr($memo, 0, 255);
    }

    /** @return array<string,mixed> */
    private function line(Account $account, string $side, float $amount, string $memo, mixed $confidence = null, ?string $reason = null): array
    {
        return [
            'account_id' => $account->id,
            'account_code' => $account->code,
            'account_name' => $account->name,
            'debit' => $side === 'debit' ? round($amount, 2) : 0.0,
            'credit' => $side === 'credit' ? round($amount, 2) : 0.0,
            'memo' => $memo,
            'confidence' => $confidence === null ? null : (float) $confidence,
            'reason' => $reason,
        ];
    }
}
