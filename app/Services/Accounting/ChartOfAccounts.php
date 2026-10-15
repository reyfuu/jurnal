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

    /** Beban pajak: 9103 (skema Dagang) atau 7103 (Manufaktur/Jasa) — pakai yang ada di COA klien. */
    public const TAX_EXPENSE_CODES = ['9103', '7103'];

    /** Bentuk badan usaha — menentukan Utang Dividen (PT) vs Prive (CV/perorangan) di template Dagang. */
    public const LEGAL_FORMS = ['pt' => 'PT', 'cv' => 'CV', 'firma' => 'Firma', 'perorangan' => 'Perorangan / UD'];

    public const INVENTORY_METHODS = [
        'perpetual' => 'Perpetual — tiap penjualan langsung dicatat HPP-nya',
        'periodik' => 'Periodik — HPP dihitung akhir periode dari stok opname',
    ];

    /**
     * Template Dagang dari review akuntan ("Daftar COA.xlsx", sheet COA Baru, 15 Okt 2026),
     * plus 1407/2116/2117 (keputusan user). Kolom: kode, nama, tipe, subtype, saldo normal
     * (D/K), kondisi: ada = dibuat nonaktif · pt · non_pt · perpetual · periodik.
     */
    private const DAGANG = [
        // ── Aset Lancar
        ['1101', 'Kas', 'asset', 'cash', 'D'],
        ['1102', 'Bank', 'asset', 'bank', 'D'],
        ['1103', 'E-Wallet (QRIS/OVO/GoPay/Dana)', 'asset', 'ewallet', 'D'],
        ['1104', 'Saldo Marketplace', 'asset', 'ewallet', 'D'],
        ['1201', 'Piutang Usaha', 'asset', 'receivable', 'D'],
        ['1202', 'Piutang Pihak Berelasi', 'asset', 'related_party_receivable', 'D'],
        ['1301', 'Persediaan Barang Dagang', 'asset', 'inventory', 'D'],
        ['1302', 'Perlengkapan', 'asset', 'supplies', 'D'],
        ['1401', 'Biaya Dibayar Dimuka', 'asset', 'prepaid', 'D'],
        ['1402', 'PPN Masukan', 'asset', 'tax_in', 'D'],
        ['1403', 'Pajak Dibayar Dimuka (PPh)', 'asset', 'prepaid_tax', 'D'],
        ['1404', 'Uang Muka Pembelian', 'asset', 'purchase_advance', 'D'],
        ['1405', 'Uang Muka Karyawan', 'asset', 'employee_advance', 'D'],
        ['1406', 'PPN Lebih Bayar', 'asset', 'vat_overpaid', 'D'],
        ['1407', 'Piutang Karyawan', 'asset', 'employee_receivable', 'D'], // + keputusan 15 Okt: talangan perusahaan (mis. BPJS) yang dipotong dari gaji
        // ── Aset Tidak Lancar
        ['1501', 'Peralatan & Inventaris', 'asset', 'fixed_asset', 'D'],
        ['1502', 'Akumulasi Penyusutan Peralatan', 'asset', 'contra_asset', 'K'],
        ['1503', 'Gedung', 'asset', 'fixed_asset', 'D', 'ada'],
        ['1504', 'Akumulasi Penyusutan Gedung', 'asset', 'contra_asset', 'K', 'ada'],
        ['1505', 'Kendaraan', 'asset', 'fixed_asset', 'D', 'ada'],
        ['1506', 'Akumulasi Penyusutan Kendaraan', 'asset', 'contra_asset', 'K', 'ada'],
        ['1601', 'Aset Pajak Tangguhan', 'asset', 'deferred_tax_asset', 'D'],
        // ── Liabilitas Jangka Pendek
        ['2101', 'Hutang Usaha', 'liability', 'payable', 'K'],
        ['2102', 'Hutang Bank', 'liability', 'loan', 'K'],
        ['2104', 'Beban Yang Masih Harus Dibayar', 'liability', 'accrued', 'K'],
        ['2105', 'Hutang Lain-lain', 'liability', 'payable', 'K'],
        ['2106', 'PPN Keluaran', 'liability', 'vat_out', 'K'],
        ['2107', 'Utang PPh 21', 'liability', 'wht_payable_pph21', 'K'],
        ['2108', 'Utang PPh 23', 'liability', 'wht_payable_pph23', 'K'],
        ['2109', 'Utang PPh 4(2)', 'liability', 'wht_payable_pph4_2', 'K'],
        ['2110', 'Utang PPh Final', 'liability', 'final_payable', 'K'],
        ['2111', 'PPN Kurang Bayar', 'liability', 'vat_underpaid', 'K'],
        ['2112', 'Utang BPJS', 'liability', 'bpjs_payable', 'K'],
        ['2113', 'Utang Karyawan', 'liability', 'employee_payable', 'K'],
        ['2114', 'Utang Pihak Berelasi', 'liability', 'related_party_payable', 'K'],
        ['2115', 'Utang Dividen', 'liability', 'dividend_payable', 'K', 'pt'],
        ['2116', 'Utang PPh 22', 'liability', 'wht_payable_pph22', 'K'], // + pengganti 2103 (bila pemungut)
        ['2117', 'Utang PPh Badan (Pasal 29)', 'liability', 'income_tax_payable', 'K'], // + pengganti 2103
        // ── Liabilitas Jangka Panjang
        ['2201', 'Liabilitas Pajak Tangguhan', 'liability', 'deferred_tax_liability', 'K'],
        // ── Ekuitas
        ['3101', 'Modal Pemilik', 'equity', 'capital', 'K'],
        ['3102', 'Prive / Pengambilan Pemilik', 'equity', 'drawing', 'D', 'non_pt'],
        ['3201', 'Laba Ditahan', 'equity', 'retained', 'K'],
        // ── Pendapatan Usaha
        ['4101', 'Penjualan', 'revenue', 'sales', 'K'],
        ['4102', 'Pendapatan Jasa', 'revenue', 'service', 'K'],
        ['4103', 'Diskon Penjualan', 'revenue', 'contra_revenue', 'D'],
        ['4104', 'Retur Penjualan', 'revenue', 'contra_revenue', 'D'],
        // ── Harga Pokok Penjualan
        ['5101', 'Harga Pokok Penjualan', 'expense', 'cogs', 'D', 'perpetual'],
        ['5102', 'Pembelian Barang Dagang', 'expense', 'cogs', 'D', 'periodik'],
        ['5103', 'Ongkos Angkut Pembelian', 'expense', 'cogs', 'D', 'periodik'],
        ['5105', 'Retur & Potongan Pembelian', 'expense', 'contra_cogs', 'K', 'periodik'],
        ['5104', 'Beban Kemasan', 'expense', 'cogs', 'D'],
        // ── Beban Operasional
        ['6101', 'Beban Gaji & Upah', 'expense', 'opex', 'D'],
        ['6117', 'Beban BPJS', 'expense', 'opex', 'D'],
        ['6102', 'Beban Iklan & Promosi', 'expense', 'opex', 'D'],
        ['6103', 'Beban Komisi Affiliate & KOL', 'expense', 'opex', 'D'],
        ['6104', 'Beban Sewa', 'expense', 'opex', 'D'],
        ['6105', 'Beban Listrik, Air & Internet', 'expense', 'opex', 'D'],
        ['6106', 'Beban Transportasi & Ongkir', 'expense', 'opex', 'D'],
        ['6107', 'Beban Perlengkapan Kantor', 'expense', 'opex', 'D'],
        ['6108', 'Beban Marketplace & Payment Gateway', 'expense', 'opex', 'D'],
        ['6109', 'Beban Jasa Profesional', 'expense', 'opex', 'D'],
        ['6110', 'Beban Perjalanan & Akomodasi', 'expense', 'opex', 'D'],
        ['6111', 'Beban Konsumsi & Entertainment', 'expense', 'opex', 'D'],
        ['6112', 'Beban Pemeliharaan & Perbaikan', 'expense', 'opex', 'D'],
        ['6113', 'Beban Penyusutan', 'expense', 'opex', 'D'],
        ['6114', 'Beban Langganan Software & AI', 'expense', 'opex', 'D'],
        ['6115', 'Beban Pelatihan & Pengembangan', 'expense', 'opex', 'D'],
        ['6116', 'Beban Produksi Konten', 'expense', 'opex', 'D'],
        ['6190', 'Beban Operasional Lain-lain', 'expense', 'opex', 'D'],
        // ── Pendapatan Lain-lain (di Luar Usaha)
        ['8101', 'Pendapatan Lain-lain', 'revenue', 'other_income', 'K'],
        // ── Beban Lain-lain (di Luar Usaha)
        ['9101', 'Beban Bunga', 'expense', 'non_operating', 'D'],
        ['9102', 'Beban Administrasi Bank', 'expense', 'non_operating', 'D'],
        ['9103', 'Beban Pajak', 'expense', 'non_operating', 'D'],
        ['9104', 'Beban (Manfaat) Pajak Tangguhan', 'expense', 'deferred_tax_expense', 'D'],
        ['9901', 'Beban Lain-lain', 'expense', 'non_operating', 'D'],
    ];

    /** Subtype akun kontra: saldo normalnya berlawanan dengan tipe induknya. */
    private const CONTRA_SUBTYPES = ['contra_asset', 'contra_revenue', 'contra_cogs', 'drawing'];

    /**
     * COA standar per jenis usaha. Dua skema kode:
     *  - Perdagangan: revisi akuntan 15 Okt 2026 (dagang()) — pendapatan di luar
     *    usaha 8xxx, beban di luar usaha 9xxx, tanpa 2103 Hutang Pajak.
     *  - Manufaktur & Jasa: skema lama — 4901 & 7xxx, ada 2103.
     * Akun yang dipakai mesin (kas 1101, bank 1102, piutang 1201, PPN 1402/2106,
     * PPh 2107–2110, hutang 2101, pendapatan default 4101, beban default 6190)
     * ada di SEMUA jenis. Beban pajak: TAX_EXPENSE_CODES.
     *
     * @return array<int,array{code:string,name:string,type:string,subtype:?string}>
     */
    public static function template(string $jenis = 'perdagangan', array $opsi = []): array
    {
        if ($jenis === 'perdagangan') {
            return self::dagang($opsi);
        }

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
            default => [],
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

    /**
     * @param  array{legal_form?:?string,inventory_method?:?string}  $opsi
     * @return array<int,array{code:string,name:string,type:string,subtype:?string,normal:string,active:bool}>
     */
    private static function dagang(array $opsi): array
    {
        $pt = ($opsi['legal_form'] ?? null) === 'pt';
        $periodik = ($opsi['inventory_method'] ?? 'perpetual') === 'periodik';

        $rows = [];
        foreach (self::DAGANG as $r) {
            [$code, $name, $type, $subtype, $normal] = $r;
            $kondisi = $r[5] ?? null;
            $berlaku = match ($kondisi) {
                'pt' => $pt,
                'non_pt' => ! $pt,
                'perpetual' => ! $periodik,
                'periodik' => $periodik,
                default => true,
            };
            if ($berlaku) {
                $rows[] = ['code' => $code, 'name' => $name, 'type' => $type, 'subtype' => $subtype,
                    'normal' => $normal === 'D' ? 'debit' : 'credit', 'active' => $kondisi !== 'ada'];
            }
        }

        return collect($rows)->sortBy('code')->values()->all();
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

        $opsi = ['legal_form' => $client->legal_form, 'inventory_method' => $client->inventory_method];
        foreach (self::template($client->business_type ?? 'perdagangan', $opsi) as $row) {
            if (isset($existing[$row['code']])) {
                continue;
            }

            $client->accounts()->create([
                'code' => $row['code'],
                'name' => $row['name'],
                'type' => $row['type'],
                'subtype' => $row['subtype'],
                'normal_balance' => $row['normal'] ?? self::normalBalanceFor($row['code'], $row['type'], $row['subtype']),
                'is_active' => $row['active'] ?? true,
            ]);
            $created++;
        }

        return $created;
    }

    /**
     * Bawa COA Dagang skema lama ke revisi akuntan (sheet "Mapping Lama-Baru").
     * Kode diganti DI TEMPAT — id akun tetap, jadi jurnal lama ikut pindah tanpa
     * disentuh. Tidak ada akun yang sudah dipakai jurnal yang dihapus. Idempoten.
     *
     * @return array<int,string> catatan perubahan
     */
    public function revisiDagang(Client $client): array
    {
        $log = [];
        $akun = fn (string $code) => $client->accounts()->where('code', $code)->first();

        foreach (['4901' => '8101', '7101' => '9101', '7102' => '9102', '7103' => '9103', '7901' => '9901'] as $lama => $baru) {
            if (! $a = $akun($lama)) {
                continue;
            }
            if ($akun($baru)) {
                $log[] = "{$lama} tidak dipindah: kode {$baru} sudah dipakai";

                continue;
            }
            // Key array numerik jadi int di PHP — bandingkan sebagai string.
            $a->update(['code' => (string) $baru] + ((string) $lama === '4901' ? ['subtype' => 'other_income'] : []));
            $log[] = "{$lama} → {$baru}";
        }

        if (($a = $akun('8101')) && $a->subtype === 'other') {
            $a->update(['subtype' => 'other_income']);
        }

        // Nama yang sudah diganti user tidak ditimpa.
        if (($a = $akun('1302')) && $a->name === 'Persediaan Kemasan & Perlengkapan') {
            $a->update(['name' => 'Perlengkapan', 'subtype' => 'supplies']);
            $log[] = '1302 → Perlengkapan';
        }

        if ($a = $akun('2103')) {
            if ($a->lines()->exists()) {
                $a->update(['is_active' => false]);
                $log[] = '2103 dinonaktifkan — sudah dipakai jurnal, saldonya perlu direklas ke akun utang pajak rinci';
            } else {
                $a->delete();
                $log[] = '2103 dihapus';
            }
        }

        // HPP perpetual ATAU pembelian periodik — set yang tidak dipakai dinonaktifkan kalau belum pernah dipakai.
        $lain = ($client->inventory_method ?? 'perpetual') === 'periodik' ? ['5101'] : ['5102', '5103', '5105'];
        foreach ($lain as $code) {
            if (($a = $akun($code)) && $a->is_active && ! $a->lines()->exists()) {
                $a->update(['is_active' => false]);
                $log[] = "{$code} dinonaktifkan (metode persediaan)";
            }
        }

        $log[] = $this->seedFor($client).' akun baru ditambahkan';

        return $log;
    }
}
