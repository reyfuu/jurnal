<?php

namespace App\Services\Accounting;

use App\Models\Account;
use App\Models\Client;

/**
 * Template COA standar UMKM Indonesia per jenis usaha (perdagangan · manufaktur ·
 * jasa). Dipasang otomatis saat klien baru dibuat, lalu bebas diedit per klien —
 * tiap klien punya salinan COA sendiri (accounts.client_id), tidak pernah berbagi. Nomor akun: 1xxx aset · 2xxx liabilitas ·
 * 3xxx ekuitas · 4xxx pendapatan · 5xxx HPP · 6xxx beban operasional ·
 * 7xxx non-operasional.
 *
 * `subtype` bukan hiasan — dipakai mesin:
 *   cash|bank|ewallet → kandidat akun pembayaran (sisi kredit saat belanja),
 *   payable           → lawan jurnal kalau belanja belum dibayar (hutang),
 *   tax_in            → PPN masukan dari invoice,
 *   prepaid_tax|vat_out|wht_payable_*|final_payable → akun pajak (config/pajak.php),
 *   opex|cogs         → sasaran default rincian biaya hasil baca AI.
 */
class ChartOfAccounts
{
    /** Akun default yang dipakai mesin kalau AI tidak yakin. */
    public const DEFAULT_EXPENSE = '6190';

    public const DEFAULT_CASH = '1101';

    public const DEFAULT_PAYABLE = '2101';

    public const DEFAULT_TAX_IN = '1402';

    public const DEFAULT_REVENUE = '4101';

    /** Jenis usaha → template COA. Dipilih saat klien dibuat (kolom clients.business_type). */
    public const BUSINESS_TYPES = [
        'perdagangan' => 'Perdagangan — jual-beli barang',
        'manufaktur' => 'Manufaktur — produksi barang sendiri',
        'jasa' => 'Jasa — menjual keahlian/layanan',
    ];

    /** Subtype akun kontra: saldo normalnya berlawanan dengan tipe induknya. */
    private const CONTRA_SUBTYPES = ['contra_asset', 'contra_revenue', 'contra_cogs', 'drawing'];

    /**
     * COA standar per jenis usaha. Akun yang dipakai mesin (kas 1101, bank 1102,
     * piutang 1201, PPN 1402/2106, PPh 2107–2110, hutang 2101/2103, pendapatan
     * default 4101, beban default 6190, beban pajak 7103) ada di SEMUA jenis —
     * yang berbeda hanya persediaan, pendapatan, dan harga pokok.
     *
     * @return array<int,array{code:string,name:string,type:string,subtype:?string}>
     */
    public static function template(string $jenis = 'perdagangan'): array
    {
        $a = fn (string $code, string $name, string $type, ?string $subtype) => compact('code', 'name', 'type', 'subtype');

        $khusus = match ($jenis) {
            'manufaktur' => [
                $a('1104', 'Saldo Marketplace', 'asset', 'ewallet'),
                $a('1301', 'Persediaan Barang Jadi', 'asset', 'inventory'),
                $a('1302', 'Persediaan Bahan Baku', 'asset', 'inventory'),
                $a('1303', 'Persediaan Barang Dalam Proses', 'asset', 'inventory'),
                $a('1304', 'Persediaan Bahan Penolong', 'asset', 'inventory'),
                $a('1503', 'Mesin & Peralatan Produksi', 'asset', 'fixed_asset'),
                $a('1504', 'Akumulasi Penyusutan Mesin', 'asset', 'contra_asset'),
                $a('4101', 'Penjualan', 'revenue', 'sales'),
                $a('4102', 'Pendapatan Jasa Maklon', 'revenue', 'service'),
                $a('4104', 'Retur Penjualan', 'revenue', 'contra_revenue'),
                $a('5101', 'Harga Pokok Penjualan', 'expense', 'cogs'),
                $a('5102', 'Pemakaian Bahan Baku', 'expense', 'cogs'),
                $a('5103', 'Upah Tenaga Kerja Langsung', 'expense', 'cogs'),
                $a('5104', 'Beban Kemasan', 'expense', 'cogs'),
                $a('5105', 'Overhead Pabrik — Listrik & Air', 'expense', 'cogs'),
                $a('5106', 'Overhead Pabrik — Penyusutan Mesin', 'expense', 'cogs'),
                $a('5107', 'Overhead Pabrik — Pemeliharaan Mesin', 'expense', 'cogs'),
                $a('5108', 'Beban Maklon / Subkon Produksi', 'expense', 'cogs'),
            ],
            'jasa' => [
                $a('2111', 'Pendapatan Diterima Dimuka (DP Pelanggan)', 'liability', 'unearned'),
                $a('4101', 'Pendapatan Jasa', 'revenue', 'service'),
                $a('4102', 'Pendapatan Proyek / Kontrak', 'revenue', 'service'),
                $a('5101', 'Beban Pokok Jasa', 'expense', 'cogs'),
                $a('5102', 'Beban Subkontraktor & Tenaga Ahli', 'expense', 'cogs'),
                $a('5103', 'Bahan Habis Pakai Langsung', 'expense', 'cogs'),
            ],
            default => [ // perdagangan
                $a('1104', 'Saldo Marketplace', 'asset', 'ewallet'),
                $a('1301', 'Persediaan Barang Dagang', 'asset', 'inventory'),
                $a('1302', 'Persediaan Kemasan & Perlengkapan', 'asset', 'inventory'),
                $a('4101', 'Penjualan', 'revenue', 'sales'),
                $a('4102', 'Pendapatan Jasa', 'revenue', 'service'),
                $a('4104', 'Retur Penjualan', 'revenue', 'contra_revenue'),
                $a('5101', 'Harga Pokok Penjualan', 'expense', 'cogs'),
                $a('5102', 'Pembelian Barang Dagang', 'expense', 'cogs'),
                $a('5103', 'Ongkos Angkut Pembelian', 'expense', 'cogs'),
                $a('5104', 'Beban Kemasan', 'expense', 'cogs'),
                $a('5105', 'Retur & Potongan Pembelian', 'expense', 'contra_cogs'),
            ],
        };

        $umum = [
            // ── 1xxx Aset ────────────────────────────────────────────────────
            $a('1101', 'Kas', 'asset', 'cash'),
            $a('1102', 'Bank', 'asset', 'bank'),
            $a('1103', 'E-Wallet (QRIS/OVO/GoPay/Dana)', 'asset', 'ewallet'),
            $a('1201', 'Piutang Usaha', 'asset', 'receivable'),
            $a('1401', 'Biaya Dibayar Dimuka', 'asset', 'prepaid'),
            $a('1402', 'PPN Masukan', 'asset', 'tax_in'),
            $a('1403', 'Pajak Dibayar Dimuka (PPh)', 'asset', 'prepaid_tax'),
            $a('1501', 'Peralatan & Inventaris', 'asset', 'fixed_asset'),
            $a('1502', 'Akumulasi Penyusutan Peralatan', 'asset', 'contra_asset'),

            // ── 2xxx Liabilitas ─────────────────────────────────────────────
            $a('2101', 'Hutang Usaha', 'liability', 'payable'),
            $a('2102', 'Hutang Bank', 'liability', 'loan'),
            $a('2103', 'Hutang Pajak', 'liability', 'tax_out'),
            $a('2104', 'Beban Yang Masih Harus Dibayar', 'liability', 'accrued'),
            $a('2105', 'Hutang Lain-lain', 'liability', 'payable'),
            $a('2106', 'PPN Keluaran', 'liability', 'vat_out'),
            $a('2107', 'Utang PPh 21', 'liability', 'wht_payable_pph21'),
            $a('2108', 'Utang PPh 23', 'liability', 'wht_payable_pph23'),
            $a('2109', 'Utang PPh 4(2)', 'liability', 'wht_payable_pph4_2'),
            $a('2110', 'Utang PPh Final', 'liability', 'final_payable'),

            // ── 3xxx Ekuitas ────────────────────────────────────────────────
            $a('3101', 'Modal Pemilik', 'equity', 'capital'),
            $a('3102', 'Prive / Pengambilan Pemilik', 'equity', 'drawing'),
            $a('3201', 'Laba Ditahan', 'equity', 'retained'),

            // ── 4xxx Pendapatan (selain yang khusus per jenis usaha) ────────
            $a('4103', 'Diskon Penjualan', 'revenue', 'contra_revenue'),
            $a('4901', 'Pendapatan Lain-lain', 'revenue', 'other'),

            // ── 6xxx Beban Operasional ──────────────────────────────────────
            $a('6101', 'Beban Gaji & Upah', 'expense', 'opex'),
            $a('6102', 'Beban Iklan & Promosi', 'expense', 'opex'),
            $a('6103', 'Beban Komisi Affiliate & KOL', 'expense', 'opex'),
            $a('6104', 'Beban Sewa', 'expense', 'opex'),
            $a('6105', 'Beban Listrik, Air & Internet', 'expense', 'opex'),
            $a('6106', 'Beban Transportasi & Ongkir', 'expense', 'opex'),
            $a('6107', 'Beban Perlengkapan Kantor', 'expense', 'opex'),
            $a('6108', 'Beban Marketplace & Payment Gateway', 'expense', 'opex'),
            $a('6109', 'Beban Jasa Profesional', 'expense', 'opex'),
            $a('6110', 'Beban Perjalanan & Akomodasi', 'expense', 'opex'),
            $a('6111', 'Beban Konsumsi & Entertainment', 'expense', 'opex'),
            $a('6112', 'Beban Pemeliharaan & Perbaikan', 'expense', 'opex'),
            $a('6113', 'Beban Penyusutan', 'expense', 'opex'),
            $a('6114', 'Beban Langganan Software & AI', 'expense', 'opex'),
            $a('6115', 'Beban Pelatihan & Pengembangan', 'expense', 'opex'),
            $a('6116', 'Beban Produksi Konten', 'expense', 'opex'),
            $a('6190', 'Beban Operasional Lain-lain', 'expense', 'opex'),

            // ── 7xxx Non-operasional ────────────────────────────────────────
            $a('7101', 'Beban Bunga', 'expense', 'non_operating'),
            $a('7102', 'Beban Administrasi Bank', 'expense', 'non_operating'),
            $a('7103', 'Beban Pajak', 'expense', 'non_operating'),
            $a('7901', 'Beban Lain-lain', 'expense', 'non_operating'),
        ];

        return collect([...$umum, ...$khusus])->sortBy('code')->values()->all();
    }

    /** Saldo normal diturunkan dari tipe, KECUALI akun kontra yang dibalik. */
    public static function normalBalanceFor(string $code, string $type, ?string $subtype = null): string
    {
        // Kontra-akun: tipenya ikut induk tapi saldo normalnya berlawanan. Dikenali
        // dari subtype; kode lama tetap dikenali untuk akun buatan manual tanpa subtype.
        $base = in_array($type, Account::DEBIT_TYPES, true) ? 'debit' : 'credit';
        $legacy = ['1502', '4103', '4104', '3102'];
        if (in_array($subtype, self::CONTRA_SUBTYPES, true) || ($subtype === null && in_array($code, $legacy, true))) {
            return $base === 'debit' ? 'credit' : 'debit';
        }

        return $base;
    }

    /**
     * Pasang COA template sesuai jenis usaha klien. Idempoten — akun yang kodenya sudah ada
     * dilewati, jadi aman dipanggil ulang pada klien lama (mis. setelah template
     * ditambah akun baru).
     *
     * @return int jumlah akun yang benar-benar baru dibuat
     */
    public function seedFor(Client $client): int
    {
        $existing = $client->accounts()->pluck('code')->all();
        $existing = array_flip($existing);
        $created = 0;

        foreach (self::template($client->business_type ?? 'perdagangan') as $row) {
            if (isset($existing[$row['code']])) {
                continue;
            }

            $client->accounts()->create([
                'code' => $row['code'],
                'name' => $row['name'],
                'type' => $row['type'],
                'subtype' => $row['subtype'],
                'normal_balance' => self::normalBalanceFor($row['code'], $row['type'], $row['subtype']),
                'is_active' => true,
            ]);
            $created++;
        }

        return $created;
    }
}
