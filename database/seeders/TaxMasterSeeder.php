<?php

namespace Database\Seeders;

use App\Models\TaxRate;
use Illuminate\Database\Seeder;

/**
 * Tarif awal dari docs/perpajakan/BRD.md §7 — BELUM diverifikasi konsultan
 * (verified_at null, tampil berlabel "belum diverifikasi"). Idempoten dan aman
 * di produksi: tidak ada data demo, versi yang sudah ada tidak disentuh.
 */
class TaxMasterSeeder extends Seeder
{
    public function run(): void
    {
        $from = '2025-01-01';
        $rows = [
            ['tax_code' => 'ppn', 'applies_to' => null, 'rate_bp' => 1200, 'dpp_numerator' => 11, 'dpp_denominator' => 12, 'source' => 'UU HPP; PMK 131/2024 (V1, V2)'],
            ['tax_code' => 'ppn_mewah', 'applies_to' => null, 'rate_bp' => 1200, 'source' => 'PMK 131/2024 (V3)'],
            ['tax_code' => 'pph23', 'applies_to' => null, 'rate_bp' => 200, 'rate_no_tax_id_bp' => 400, 'source' => 'UU PPh ps. 23 — jasa (V9)'],
            ['tax_code' => 'pph4_2', 'applies_to' => null, 'rate_bp' => 1000, 'source' => 'PP 34/2017 — sewa tanah/bangunan (V11)'],
            ['tax_code' => 'pph22', 'applies_to' => null, 'rate_bp' => 150, 'rate_no_tax_id_bp' => 300, 'source' => 'PMK 34/2017 — tarif umum, beragam per objek (V13)'],
            ['tax_code' => 'final_umkm', 'applies_to' => 'orang_pribadi', 'rate_bp' => 50, 'exempt_threshold' => 500_000_000, 'source' => 'PP 55/2022 (V5, V6)'],
            ['tax_code' => 'final_umkm', 'applies_to' => 'badan', 'rate_bp' => 50, 'source' => 'PP 55/2022 (V5)'],
        ];

        foreach ($rows as $row) {
            $exists = TaxRate::where('tax_code', $row['tax_code'])
                ->where('applies_to', $row['applies_to'])
                ->whereDate('effective_from', $from)
                ->exists();
            if (! $exists) {
                TaxRate::create($row + ['effective_from' => $from, 'rounding' => 'floor', 'notes' => 'Seed awal — belum diverifikasi konsultan pajak.']);
            }
        }
    }
}
