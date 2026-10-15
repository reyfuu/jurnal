<?php

namespace App\Services\Tax;

use App\Models\Account;
use App\Models\TaxRate;
use App\Services\Accounting\ChartOfAccounts;
use Illuminate\Support\Collection;
use InvalidArgumentException;

/**
 * Satu-satunya tempat rumus pajak. Semua uang integer rupiah, tanpa float:
 *   pajak = base × rate_bp × dpp_num / (10000 × dpp_den), dibulatkan sesuai tarif.
 * Tarif diambil dari tabel `tax_rates` sesuai tanggal — tidak ada angka tarif di kode.
 */
class TaxCalculator
{
    public const MAX_BASE = 10_000_000_000_000; // 10 triliun — batas aman integer 64-bit

    public static function codes(): array
    {
        return config('pajak.codes');
    }

    public static function isVat(string $code): bool
    {
        return (config("pajak.codes.{$code}.kind")) === 'vat';
    }

    /** Versi tarif yang berlaku di tanggal itu. Tarif khusus jenis WP menang atas tarif umum. */
    public function rateFor(string $code, string $date, ?string $taxpayerType = null): ?TaxRate
    {
        return TaxRate::where('tax_code', $code)
            ->whereDate('effective_from', '<=', $date)
            ->where(fn ($q) => $q->whereNull('effective_to')->orWhereDate('effective_to', '>=', $date))
            ->where(fn ($q) => $q->whereNull('applies_to')->when($taxpayerType, fn ($q) => $q->orWhere('applies_to', $taxpayerType)))
            ->orderByRaw('applies_to IS NULL') // spesifik dulu
            ->orderByDesc('effective_from')
            ->first();
    }

    public function compute(int $base, TaxRate $rate, bool $noTaxId = false, bool $applyDpp = true): int
    {
        if ($base < 0 || $base > self::MAX_BASE) {
            throw new InvalidArgumentException('Dasar pengenaan pajak di luar batas wajar.');
        }

        $bp = $noTaxId && $rate->rate_no_tax_id_bp ? $rate->rate_no_tax_id_bp : $rate->rate_bp;
        [$num, $den] = $applyDpp ? [$rate->dpp_numerator, $rate->dpp_denominator] : [1, 1];
        $raw = $base * $bp * $num;
        $div = 10000 * $den;

        return $rate->rounding === 'half_up' ? intdiv(2 * $raw + $div, 2 * $div) : intdiv($raw, $div);
    }

    /**
     * Cek angka pajak dokumen terhadap tarif berlaku. Nilai dokumen TIDAK pernah
     * diubah — hasilnya status + angka pembanding untuk warning.
     *
     * Dua bentuk umum yang diterima sebagai cocok: base sudah berupa DPP nilai
     * lain (Coretax mencetak "DPP Nilai Lain") — HANYA kalau $gross (harga jual
     * dari rincian) diketahui dan base ≈ gross × faktor DPP; tanpa itu PPN 12%
     * dari harga penuh tidak bisa dibedakan dan harus jadi selisih — dan tarif
     * tanpa NPWP.
     *
     * @return array{status:string,computed:?int,base:int,rate:?TaxRate,note:?string}
     */
    public function check(string $code, int $base, int $amount, string $date, ?string $taxpayerType = null, int $items = 1, ?int $gross = null): array
    {
        $rate = $base > 0 ? $this->rateFor($code, $date, $taxpayerType) : null;
        if (! $rate) {
            return ['status' => 'no_rate', 'computed' => null, 'base' => $base, 'rate' => null, 'note' => null];
        }

        $tolerance = max(1, $items) * (int) config('pajak.toleransi_rupiah', 1);
        $close = fn (int $c) => abs($c - $amount) <= $tolerance;

        $computed = $this->compute($base, $rate);
        if ($close($computed)) {
            return ['status' => 'match', 'computed' => $computed, 'base' => $base, 'rate' => $rate, 'note' => null];
        }

        $baseIsDpp = $gross !== null && abs(intdiv($gross * $rate->dpp_numerator, $rate->dpp_denominator) - $base) <= $tolerance;
        if ($baseIsDpp && $close($c = $this->compute($base, $rate, applyDpp: false))) {
            // Base yang tertulis sudah DPP nilai lain → simpan harga jualnya supaya rekap konsisten.
            return ['status' => 'match', 'computed' => $c, 'base' => $gross, 'rate' => $rate, 'note' => 'Dasar yang tertulis adalah DPP nilai lain.'];
        }

        if ($rate->rate_no_tax_id_bp && $close($c = $this->compute($base, $rate, noTaxId: true))) {
            return ['status' => 'match', 'computed' => $c, 'base' => $base, 'rate' => $rate, 'note' => 'Dihitung dengan tarif tanpa NPWP — pastikan lawan transaksi memang tidak ber-NPWP.'];
        }

        return ['status' => 'mismatch', 'computed' => $computed, 'base' => $base, 'rate' => $rate, 'note' => null];
    }

    /**
     * Akun pajak di COA klien untuk kode & arah ini, dicari lewat subtype.
     * Tidak ketemu → 1402 PPN Masukan / 2103 Hutang Pajak (skema Manufaktur/Jasa;
     * Dagang tidak punya 2103 → null, baris pajak dilewati & selisihnya jadi warning).
     *
     * @param  Collection<string,Account>  $accounts  akun aktif klien, key = kode
     */
    public function accountFor(Collection $accounts, string $code, string $direction): ?Account
    {
        $subtype = config("pajak.codes.{$code}.subtype_".$direction);
        $found = $subtype ? $accounts->first(fn (Account $a) => $a->subtype === $subtype) : null;

        return $found ?? $accounts->get($direction === 'in' ? ChartOfAccounts::DEFAULT_TAX_IN : '2103');
    }
}
