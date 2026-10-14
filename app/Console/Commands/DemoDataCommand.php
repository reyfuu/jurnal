<?php

namespace App\Console\Commands;

use App\Models\Client;
use App\Models\Document;
use App\Models\Journal;
use App\Models\TaxEntry;
use App\Models\TaxFiling;
use App\Models\User;
use App\Services\Accounting\ChartOfAccounts;
use App\Services\Accounting\JournalPoster;
use App\Services\Intake\DocumentExtractor;
use App\Support\SandiLemah;
use Illuminate\Console\Command;
use Illuminate\Support\Carbon;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Str;

/**
 * Data dummy untuk mencoba aplikasi: 12 perusahaan, 3 bulan jurnal (termasuk
 * PPN/PPh), dokumen menunggu review, dan 5 akun per peran.
 *
 * Aman di produksi karena SEMUA tertanda: slug klien `demo-…` dan email
 * `…@demo.jurnal.id`. `--hapus` membersihkannya tanpa menyentuh data asli.
 */
class DemoDataCommand extends Command
{
    protected $signature = 'jurnal:demo
        {--hapus : Hapus semua perusahaan & akun demo}
        {--sandi= : Sandi untuk akun demo (default: acak, dicetak sekali)}';

    protected $description = 'Isi (atau hapus) data dummy bertanda demo';

    private const EMAIL_DOMAIN = '@demo.jurnal.id';

    /**
     * [nama, jenis, omzet/bulan, akun pendapatan, kanal terima uang, rasio HPP, PKP, rezim, wajib pajak, dipotong PPh23, beban/bulan]
     * ponytail: profil di kode, bukan file fixture — cukup untuk data coba.
     */
    private const PERUSAHAAN = [
        ['Kopi Senja Nusantara', 'external', 68_000_000, '4101', '1103', 0.38, false, 'final_umkm', 'orang_pribadi', false, ['6101' => 14_000_000, '6104' => 6_500_000, '6105' => 2_100_000, '6102' => 1_800_000]],
        ['Batik Larasati', 'external', 145_000_000, '4101', '1104', 0.45, false, 'final_umkm', 'badan', false, ['6101' => 18_500_000, '6102' => 16_000_000, '6108' => 9_800_000, '6106' => 4_200_000]],
        ['Klinik Gigi Senyum Sehat', 'external', 210_000_000, '4102', '1102', 0.0, true, 'normal', 'badan', false, ['6101' => 62_000_000, '6104' => 15_000_000, '6105' => 4_800_000, '6112' => 3_500_000]],
        ['Toko Bangunan Sinar Jaya', 'external', 380_000_000, '4101', '1102', 0.72, true, 'normal', 'badan', false, ['6101' => 28_000_000, '6106' => 11_000_000, '6105' => 3_200_000, '6107' => 1_400_000]],
        ['Studio Kreatif Lensa', 'external', 96_000_000, '4102', '1102', 0.0, true, 'normal', 'badan', true, ['6101' => 31_000_000, '6116' => 8_500_000, '6114' => 3_100_000, '6110' => 4_600_000]],
        ['Roti Hangat Bakery', 'external', 82_000_000, '4101', '1101', 0.42, false, 'final_umkm', 'orang_pribadi', false, ['6101' => 16_000_000, '6104' => 7_000_000, '6105' => 3_900_000, '6106' => 1_600_000]],
        ['CV Logistik Cepat', 'external', 240_000_000, '4102', '1102', 0.0, true, 'normal', 'badan', true, ['6101' => 54_000_000, '6106' => 38_000_000, '6112' => 12_000_000, '6105' => 2_700_000]],
        ['Laundry Bersih Kilat', 'external', 46_000_000, '4102', '1103', 0.0, false, 'final_umkm', 'orang_pribadi', false, ['6101' => 12_500_000, '6105' => 5_400_000, '6107' => 2_300_000, '6104' => 4_000_000]],
        ['PT Agro Tani Makmur', 'external', 520_000_000, '4101', '1102', 0.66, true, 'normal', 'badan', false, ['6101' => 74_000_000, '6106' => 26_000_000, '6110' => 9_000_000, '6109' => 7_500_000]],
        ['RM Padang Rasa Bundo', 'external', 124_000_000, '4101', '1101', 0.48, false, 'final_umkm', 'badan', false, ['6101' => 22_000_000, '6104' => 9_500_000, '6105' => 4_100_000, '6111' => 1_200_000]],
        ['Konveksi Jahit Rapi', 'external', 175_000_000, '4101', '1102', 0.55, true, 'normal', 'badan', false, ['6101' => 41_000_000, '6105' => 6_200_000, '6112' => 2_800_000, '6106' => 5_300_000]],
        ['Glow Lab Skincare', 'internal', 290_000_000, '4101', '1104', 0.35, true, 'normal', 'badan', false, ['6101' => 38_000_000, '6102' => 42_000_000, '6103' => 19_000_000, '6108' => 21_000_000]],
    ];

    /** [nama, email, peran, indeks perusahaan] — patuh batas: supervisor 3 · akuntan 5 · anggota 10. */
    private const AKUN = [
        ['Dewi (Supervisor)', 'dewi', User::ROLE_SUPERVISOR, [0, 1, 2]],
        ['Rina (Akuntan)', 'rina', User::ROLE_AKUNTAN, [0, 1, 2, 3, 4]],
        ['Budi (Akuntan)', 'budi', User::ROLE_AKUNTAN, [5, 6, 7, 8, 9]],
        ['Sari (Anggota)', 'sari', User::ROLE_ANGGOTA, [0, 1, 2, 3, 4, 5, 6, 7, 8, 9]],
        ['Joko (Anggota)', 'joko', User::ROLE_ANGGOTA, [9, 10, 11]],
    ];

    public function handle(JournalPoster $poster, ChartOfAccounts $coa, DocumentExtractor $extractor): int
    {
        if ($this->option('hapus')) {
            return $this->hapus();
        }

        if (Client::where('slug', 'like', 'demo-%')->exists()) {
            $this->warn('Data demo sudah ada. Jalankan `php artisan jurnal:demo --hapus` dulu kalau mau mengulang.');

            return self::FAILURE;
        }

        $sandi = (string) ($this->option('sandi') ?: 'Coba-'.Str::random(12));
        if ($alasan = SandiLemah::alasanTolak($sandi)) {
            $this->error($alasan);

            return self::FAILURE;
        }

        DB::transaction(function () use ($poster, $coa, $extractor, $sandi) {
            $clients = [];
            foreach (self::PERUSAHAAN as $i => $p) {
                $client = Client::create([
                    'name' => $p[0], 'slug' => 'demo-'.Str::slug($p[0]), 'type' => $p[1], 'is_active' => true,
                    'legal_name' => str_starts_with($p[0], 'PT') || str_starts_with($p[0], 'CV') ? $p[0] : null,
                    'contact_name' => 'Pemilik '.Str::before($p[0], ' '), 'phone' => '0812'.str_pad((string) (55_000_000 + $i * 7_331), 8, '0', STR_PAD_LEFT),
                    'notes' => '[DEMO] Data dummy — hapus dengan `php artisan jurnal:demo --hapus`.',
                    'taxpayer_type' => $p[8], 'is_pkp' => $p[6], 'pph_regime' => $p[7],
                    'pkp_since' => $p[6] ? '2024-01-01' : null,
                    'tax_id' => sprintf('0%d.%03d.%03d.%d-%03d.000', $i % 9 + 1, 100 + $i * 37, 200 + $i * 53, $i % 10, 400 + $i * 11),
                ]);
                $coa->seedFor($client);
                $clients[] = $client;
            }

            $users = [];
            foreach (self::AKUN as [$name, $slug, $role, $idx]) {
                $users[$slug] = User::create(['name' => $name, 'email' => $slug.self::EMAIL_DOMAIN, 'password' => $sandi, 'role' => $role, 'is_active' => true]);
                $users[$slug]->clients()->attach(array_map(fn ($k) => $clients[$k]->id, $idx));
            }

            foreach ($clients as $i => $client) {
                $pembuat = $i < 5 ? $users['rina'] : $users['budi'];
                $this->isiBuku($poster, $client, self::PERUSAHAAN[$i], $pembuat->id);
                $this->isiDokumen($extractor, $client, self::PERUSAHAAN[$i], $i, ($i < 10 ? $users['sari'] : $users['joko'])->id);
            }
        });

        $this->info(count(self::PERUSAHAAN).' perusahaan demo dibuat (3 bulan jurnal + dokumen menunggu review).');
        $this->table(['Email', 'Peran', 'Perusahaan'], array_map(
            fn ($a) => [$a[1].self::EMAIL_DOMAIN, User::ROLES[$a[2]], count($a[3])], self::AKUN,
        ));
        $this->line("Sandi semua akun demo: <comment>{$sandi}</comment>");

        return self::SUCCESS;
    }

    /** Tiga bulan pembukuan: saldo awal, penjualan mingguan (+PPN/PPh23), HPP, pembelian, gaji (+PPh21), beban, pajak final. */
    private function isiBuku(JournalPoster $poster, Client $client, array $p, int $userId): void
    {
        [, , $omzet, $akunPendapatan, $kanal, $rasioHpp, $pkp, $rezim, $wp, $pph23, $beban] = $p;
        mt_srand(crc32($client->slug));
        $akun = $client->accounts()->pluck('id', 'code');
        $catat = function (string $date, string $desc, string $type, array $lines, array $pajak = []) use ($poster, $client, $akun, $userId) {
            if (Carbon::parse($date)->isFuture()) {
                return;
            }
            $poster->record(
                ['client_id' => $client->id, 'date' => $date, 'description' => $desc, 'type' => $type, 'created_by' => $userId],
                array_map(fn ($l) => ['account_id' => $akun[$l[0]], 'debit' => $l[1], 'credit' => $l[2], 'memo' => $l[3] ?? null], $lines),
                Journal::STATUS_POSTED,
                array_map(fn ($t) => ['account_id' => $akun[$t[0]], 'tax_code' => $t[1], 'direction' => $t[2], 'base' => $t[3], 'amount' => $t[4],
                    'withheld' => $t[5] ?? false, 'match_status' => 'manual'], $pajak),
            );
        };
        $acak = fn (float $n) => (int) (round($n * mt_rand(82, 118) / 100 / 1000) * 1000);
        $bank = '1102';
        $hpp = $rasioHpp > 0;
        $nama = collect(ChartOfAccounts::template())->pluck('name', 'code');

        $awal = now()->startOfMonth()->subMonths(2);
        $catat($awal->format('Y-m-01'), 'Saldo awal: modal & persediaan', 'adjustment', array_filter([
            [$bank, $omzet * 2, 0, 'Saldo rekening awal'],
            $hpp ? ['1301', (int) ($omzet * $rasioHpp * 1.2), 0, 'Persediaan awal'] : null,
            ['3101', 0, $omzet * 2 + ($hpp ? (int) ($omzet * $rasioHpp * 1.2) : 0), 'Modal pemilik'],
        ]));

        for ($m = 2; $m >= 0; $m--) {
            $bulan = now()->startOfMonth()->subMonths($m);
            $ym = $bulan->format('Y-m');
            $omzetBulan = 0;

            // Penjualan mingguan.
            foreach ([3, 10, 17, 24] as $minggu => $hari) {
                $dpp = $acak($omzet / 4);
                $ppn = $pkp ? (int) round($dpp * 0.11) : 0;
                $potong = $pph23 ? (int) round($dpp * 0.02) : 0;
                $omzetBulan += $dpp;
                $lines = [[$kanal, $dpp + $ppn - $potong, 0, 'Penerimaan']];
                $pajak = [];
                if ($potong) {
                    $lines[] = ['1403', $potong, 0, 'PPh 23 dipotong pelanggan'];
                    $pajak[] = ['1403', 'pph23', 'in', $dpp, $potong, true];
                }
                $lines[] = [$akunPendapatan, 0, $dpp, $akunPendapatan === '4102' ? 'Pendapatan jasa' : 'Penjualan'];
                if ($ppn) {
                    $lines[] = ['2106', 0, $ppn, 'PPN keluaran 11%'];
                    $pajak[] = ['2106', 'ppn', 'out', $dpp, $ppn];
                }
                $label = $akunPendapatan === '4102' ? 'Pendapatan jasa' : 'Penjualan';
                $catat("{$ym}-".sprintf('%02d', $hari), "{$label} minggu ".($minggu + 1), 'sale', $lines, $pajak);
                if ($hpp) {
                    $nilai = (int) round($dpp * $rasioHpp / 1000) * 1000;
                    $catat("{$ym}-".sprintf('%02d', $hari), 'HPP minggu '.($minggu + 1), 'sale', [['5101', $nilai, 0, 'HPP'], ['1301', 0, $nilai, 'Persediaan keluar']]);
                }
            }

            // Pembelian persediaan (tempo) + pelunasan sebagian.
            if ($hpp) {
                $beli = $acak($omzet * $rasioHpp);
                $ppnMasuk = $pkp ? (int) round($beli * 0.11) : 0;
                $catat("{$ym}-06", 'Pembelian persediaan dari supplier', 'purchase', array_filter([
                    ['1301', $beli, 0, 'Barang dagang/bahan'],
                    $ppnMasuk ? ['1402', $ppnMasuk, 0, 'PPN masukan'] : null,
                    ['2101', 0, $beli + $ppnMasuk, 'Hutang supplier tempo 30 hari'],
                ]), $ppnMasuk ? [['1402', 'ppn', 'in', $beli, $ppnMasuk]] : []);
                $bayar = (int) round(($beli + $ppnMasuk) * 0.8 / 1000) * 1000;
                $catat("{$ym}-20", 'Pelunasan hutang supplier', 'purchase', [['2101', $bayar, 0], [$bank, 0, $bayar, 'Transfer']]);
            }

            // Gaji + PPh 21 (badan memotong PPh 21).
            $gaji = $beban['6101'];
            $pph21 = $wp === 'badan' ? (int) round($gaji * 0.025) : 0;
            $catat("{$ym}-25", 'Gaji karyawan '.$bulan->translatedFormat('F'), 'expense', array_filter([
                ['6101', $gaji, 0, 'Gaji & tunjangan'],
                [$bank, 0, $gaji - $pph21, 'Transfer payroll'],
                $pph21 ? ['2107', 0, $pph21, 'PPh 21 dipotong'] : null,
            ]), $pph21 ? [['2107', 'pph21', 'out', $gaji, $pph21]] : []);
            if ($pph21 && $m < 2) {
                $catat("{$ym}-10", 'Setor PPh 21 bulan lalu', 'bank', [['2107', $pph21, 0], [$bank, 0, $pph21, 'Billing DJP']]);
            }

            // Beban operasional lain.
            foreach (array_slice($beban, 1, null, true) as $kode => $nilai) {
                $hari = sprintf('%02d', mt_rand(2, 27));
                $jumlah = $acak($nilai);
                $catat("{$ym}-{$hari}", $nama[$kode], 'expense', [[$kode, $jumlah, 0], [$nilai < 3_000_000 ? '1101' : $bank, 0, $jumlah]]);
            }

            $catat("{$ym}-28", 'Biaya administrasi bank', 'bank', [['7102', 85_000, 0], [$bank, 0, 85_000]]);

            // PPh Final UMKM 0,5% omzet.
            if ($rezim === 'final_umkm') {
                $final = (int) round($omzetBulan * 0.005);
                $catat("{$ym}-28", 'PPh Final UMKM 0,5% '.$bulan->translatedFormat('F'), 'adjustment',
                    [['7103', $final, 0], ['2110', 0, $final, 'Terutang']], [['2110', 'final_umkm', 'out', $omzetBulan, $final]]);
            }

            // Bulan tertua sudah setor & lapor, supaya kalender pajak punya dua keadaan.
            if ($m === 2) {
                foreach (array_filter(['pph21' => $pph21, 'ppn' => $pkp, 'final_umkm' => $rezim === 'final_umkm']) as $kodePajak => $_) {
                    TaxFiling::create([
                        'client_id' => $client->id, 'tax_code' => $kodePajak, 'period' => $ym, 'status' => 'lapor',
                        'paid_at' => $bulan->copy()->addMonth()->day(14), 'filed_at' => $bulan->copy()->addMonth()->day(19),
                        'ntpn' => strtoupper(substr(md5($client->slug.$kodePajak.$ym), 0, 16)), 'note' => '[DEMO]',
                    ]);
                }
            }
        }

        // Satu jurnal salah yang sudah di-void — supaya fitur void terlihat.
        $salah = Journal::where('client_id', $client->id)->where('type', 'expense')->latest('id')->first();
        $salah?->update(['status' => Journal::STATUS_VOID, 'description' => $salah->description.' (dobel input — di-void)']);
    }

    /** Dokumen: satu menunggu review (hasil baca AI), sebagian gagal dibaca. */
    private function isiDokumen(DocumentExtractor $extractor, Client $client, array $p, int $i, int $uploaderId): void
    {
        $tanggal = now()->subDays(2)->toDateString();
        $item = $p[5] > 0
            ? [['description' => 'Bahan/barang dagang', 'qty' => 50, 'unit' => 'pcs', 'unit_price' => 42_000, 'amount' => 2_100_000, 'account_code' => '1301', 'account_reason' => 'Barang untuk dijual kembali → persediaan.', 'confidence' => 0.91],
                ['description' => 'Ongkos kirim supplier', 'qty' => 1, 'unit' => 'kali', 'unit_price' => 85_000, 'amount' => 85_000, 'account_code' => '6106', 'account_reason' => 'Biaya angkut pembelian.', 'confidence' => 0.62]]
            : [['description' => 'Perlengkapan operasional', 'qty' => 12, 'unit' => 'pak', 'unit_price' => 37_500, 'amount' => 450_000, 'account_code' => '6107', 'account_reason' => 'Habis pakai untuk operasional.', 'confidence' => 0.88],
                ['description' => 'Servis AC ruang tunggu', 'qty' => 1, 'unit' => 'kali', 'unit_price' => 350_000, 'amount' => 350_000, 'account_code' => '6112', 'account_reason' => 'Pemeliharaan aset.', 'confidence' => 0.83]];
        $total = array_sum(array_column($item, 'amount'));

        $client->documents()->create([
            'kind' => Document::KIND_RECEIPT,
            'title' => 'Nota pembelian '.Carbon::parse($tanggal)->translatedFormat('j M'),
            'original_name' => 'nota-'.$client->slug.'.jpg',
            'mime_type' => 'image/jpeg',
            'raw_text' => "TOKO SUMBER REJEKI\n".Carbon::parse($tanggal)->format('d/m/Y')."\n".collect($item)->map(fn ($r) => "{$r['description']} {$r['amount']}")->join("\n")."\nTOTAL {$total}",
            'status' => Document::STATUS_EXTRACTED,
            'model_used' => 'demo',
            'extracted_at' => now(),
            'uploaded_by' => $uploaderId,
            'extraction' => $extractor->normalize([
                'doc_type' => 'receipt', 'flow' => 'expense', 'vendor' => 'Toko Sumber Rejeki',
                'document_date' => $tanggal, 'payment_method' => 'cash', 'currency' => 'IDR',
                'subtotal' => $total, 'discount' => 0, 'tax' => 0, 'total' => $total, 'items' => $item,
                'notes' => 'Nota tulisan tangan, cukup jelas.',
                'warnings' => ['Satu baris punya keyakinan rendah — cek akunnya sebelum posting.'],
            ], $client),
        ]);

        if ($i % 3 === 0) {
            $client->documents()->create([
                'kind' => Document::KIND_RECEIPT, 'title' => 'Foto struk parkir', 'original_name' => 'IMG_2201.jpg', 'mime_type' => 'image/jpeg',
                'status' => Document::STATUS_FAILED, 'uploaded_by' => $uploaderId,
                'error' => 'Foto terlalu buram — total tidak terbaca. Foto ulang dengan cahaya cukup.',
            ]);
        }
    }

    private function hapus(): int
    {
        $clientIds = Client::where('slug', 'like', 'demo-%')->pluck('id');
        $userIds = User::where('email', 'like', '%'.self::EMAIL_DOMAIN)->pluck('id');

        DB::transaction(function () use ($clientIds, $userIds) {
            // Urutan eksplisit: journal_lines.account_id tidak cascade.
            TaxEntry::whereIn('client_id', $clientIds)->delete();
            TaxFiling::whereIn('client_id', $clientIds)->delete();
            Journal::whereIn('client_id', $clientIds)->each(fn (Journal $j) => $j->lines()->delete());
            Journal::whereIn('client_id', $clientIds)->delete();
            Document::whereIn('client_id', $clientIds)->delete();
            DB::table('accounts')->whereIn('client_id', $clientIds)->delete();
            DB::table('client_user')->whereIn('client_id', $clientIds)->orWhereIn('user_id', $userIds)->delete();
            Client::whereIn('id', $clientIds)->delete();
            User::whereIn('id', $userIds)->delete();
        });

        $this->info("Dihapus: {$clientIds->count()} perusahaan demo, {$userIds->count()} akun demo.");

        return self::SUCCESS;
    }
}
