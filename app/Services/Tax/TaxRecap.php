<?php

namespace App\Services\Tax;

use App\Models\Account;
use App\Models\Client;
use App\Models\Journal;
use App\Models\TaxEntry;
use App\Models\TaxFiling;
use App\Services\Reporting\ReportService;
use Illuminate\Support\Carbon;
use Illuminate\Support\Collection;

/**
 * Rekap pajak per klien per masa (YYYY-MM). Satu sumber angka: tax_entries dari
 * jurnal POSTED — void otomatis keluar, sama seperti laporan keuangan.
 */
class TaxRecap
{
    public function __construct(private TaxCalculator $calc, private ReportService $reports) {}

    /** @return Collection<int,TaxEntry> */
    public function entries(Client $client, string $period): Collection
    {
        return TaxEntry::query()
            ->join('journals', 'journals.id', '=', 'tax_entries.journal_id')
            ->where('tax_entries.client_id', $client->id)
            ->where('journals.status', Journal::STATUS_POSTED)
            ->where('journals.period', $period)
            ->orderBy('journals.date')
            ->select('tax_entries.*', 'journals.date as journal_date', 'journals.description as journal_description')
            ->get();
    }

    /** PPN masa: Keluaran − Masukan. Positif = kurang bayar. */
    public function ppn(Client $client, string $period, ?Collection $entries = null): array
    {
        $rows = ($entries ?? $this->entries($client, $period))
            ->filter(fn (TaxEntry $e) => TaxCalculator::isVat($e->tax_code) && ! $e->noncreditable);
        $out = (int) $rows->where('direction', 'out')->sum('amount');
        $in = (int) $rows->where('direction', 'in')->sum('amount');

        return ['keluaran' => $out, 'masukan' => $in, 'net' => $out - $in, 'rows' => $rows->values()];
    }

    /** PPh potong/pungut per jenis: yang wajib kita setor vs yang jadi kredit pajak kita. */
    public function withholding(Client $client, string $period, ?Collection $entries = null): array
    {
        $rows = ($entries ?? $this->entries($client, $period))
            ->filter(fn (TaxEntry $e) => ! TaxCalculator::isVat($e->tax_code) && $e->tax_code !== 'final_umkm');

        $groups = $rows->groupBy(fn (TaxEntry $e) => $e->tax_code.'|'.$e->direction)
            ->map(fn ($g, $key) => [
                'tax_code' => explode('|', $key)[0],
                'direction' => explode('|', $key)[1],
                'base' => (int) $g->sum('base'),
                'amount' => (int) $g->sum('amount'),
                'count' => $g->count(),
            ])->values();

        return ['groups' => $groups, 'rows' => $rows->values()];
    }

    /**
     * PPh Final UMKM masa. Omzet = kredit − debit akun pendapatan usaha (sales &
     * service) di masa itu. Orang pribadi: omzet kumulatif tahun berjalan sampai
     * batas bebas (tarif `exempt_threshold`) tidak dikenai pajak.
     */
    public function finalUmkm(Client $client, string $period): ?array
    {
        if (! $client->isFinalUmkm()) {
            return null;
        }

        $date = Carbon::createFromFormat('Y-m', $period)->endOfMonth()->toDateString();
        $rate = $this->calc->rateFor('final_umkm', $date, $client->taxpayer_type);
        $omzetOf = fn (string $p) => $this->omzet($client, $p);

        $omzet = $omzetOf($period);
        $year = substr($period, 0, 4);
        $before = 0;
        for ($m = 1; $m < (int) substr($period, 5, 2); $m++) {
            $before += $omzetOf(sprintf('%s-%02d', $year, $m));
        }

        $remaining = $rate?->exempt_threshold ? max(0, $rate->exempt_threshold - $before) : 0;
        $exempt = min($omzet, $remaining);
        $taxable = $omzet - $exempt;

        return [
            'omzet' => $omzet,
            'omzet_ytd_before' => $before,
            'exempt' => $exempt,
            'taxable' => $taxable,
            'rate' => $rate,
            'tax' => $rate ? $this->calc->compute($taxable, $rate) : null,
        ];
    }

    private function omzet(Client $client, string $period): int
    {
        $ids = $client->accounts()->whereIn('subtype', ['sales', 'service'])->pluck('id');
        $moves = $this->reports->movements($client->id, $period)->whereIn('account_id', $ids->all());

        return max(0, (int) round($moves->sum(fn ($m) => (float) $m->credit - (float) $m->debit)));
    }

    /**
     * Cek silang dengan buku: mutasi akun pajak di masa ini vs catatan pajak.
     * Selisih biasanya = jurnal manual yang menyentuh akun pajak tanpa catatan.
     *
     * @return array<int,array{account:Account,booked:int,recorded:int}>
     */
    public function reconcile(Client $client, string $period, ?Collection $entries = null): array
    {
        $subtypes = collect(config('pajak.codes'))->flatMap(fn ($c) => [$c['subtype_in'], $c['subtype_out']])->filter()->unique();
        $accounts = $client->accounts()->whereIn('subtype', $subtypes)->get()->keyBy('id');
        $moves = $this->reports->movements($client->id, $period);
        $entries = ($entries ?? $this->entries($client, $period))->where('noncreditable', false);

        $out = [];
        foreach ($accounts as $id => $account) {
            $m = $moves->get($id);
            $booked = $m ? (int) round(abs((float) $m->debit - (float) $m->credit)) : 0;
            $byEntry = $entries->where('account_id', $id);
            $recorded = (int) abs($byEntry->where('direction', 'in')->sum('amount') - $byEntry->where('direction', 'out')->sum('amount'));
            if ($booked !== $recorded) {
                $out[] = ['account' => $account, 'booked' => $booked, 'recorded' => $recorded];
            }
        }

        return $out;
    }

    /**
     * Kewajiban setor/lapor masa ini, dihitung dari profil klien + catatan pajak.
     * Tidak disimpan — hanya statusnya (tax_filings) yang disimpan, diisi manusia.
     *
     * @return array<int,array<string,mixed>>
     */
    public function obligations(Client $client, string $period, ?Collection $entries = null): array
    {
        $entries ??= $this->entries($client, $period);
        $codes = [];
        if ($client->hasTaxProfile() && $client->isPkpOn(Carbon::createFromFormat('Y-m', $period)->endOfMonth()->toDateString())) {
            $codes[] = 'ppn';
        }
        if ($client->isFinalUmkm()) {
            $codes[] = 'final_umkm';
        }
        foreach ($entries->where('direction', 'out')->where('withheld', true)->pluck('tax_code')->unique() as $code) {
            $codes[] = $code;
        }

        $filings = TaxFiling::where('client_id', $client->id)->where('period', $period)->get()->keyBy('tax_code');
        $today = now()->toDateString();

        return collect(array_unique($codes))->map(function (string $code) use ($period, $filings, $today) {
            $cfg = config("pajak.codes.{$code}");
            $payDue = self::dueDate($period, $cfg['pay_day']);
            $fileDue = self::dueDate($period, $cfg['file_day']);
            $filing = $filings->get($code);
            $status = $filing?->status ?? 'belum';
            $done = in_array($status, ['lapor', 'nihil'], true);

            return [
                'tax_code' => $code,
                'name' => $cfg['name'],
                'pay_due' => $payDue,
                'file_due' => $fileDue,
                'filing' => $filing,
                'status' => $status,
                'overdue' => ! $done && (($status === 'belum' && $payDue < $today) || $fileDue < $today),
            ];
        })->values()->all();
    }

    /**
     * Jatuh tempo di bulan setelah masa. null = akhir bulan. Sabtu/Minggu
     * bergeser ke Senin.
     * ponytail: hari libur nasional belum dihitung — tambah tabel `holidays` kalau perlu (BRD V20).
     */
    public static function dueDate(string $period, ?int $day): string
    {
        $month = Carbon::createFromFormat('Y-m-d', $period.'-01')->addMonthNoOverflow();
        $date = $day ? $month->setDay($day) : $month->endOfMonth();
        while ($date->isWeekend()) {
            $date->addDay();
        }

        return $date->toDateString();
    }
}
