<?php

namespace Database\Seeders;

use App\Models\Client;
use App\Models\User;
use App\Services\Accounting\ChartOfAccounts;
use App\Support\SandiLemah;
use Illuminate\Database\Seeder;
use Illuminate\Support\Str;
use RuntimeException;

class DatabaseSeeder extends Seeder
{
    /**
     * Isi awal: satu akun admin, plus klien contoh saat pengembangan.
     *
     * Di PRODUKSI kredensial admin WAJIB diisi lewat .env. Seeder sengaja gagal
     * daripada membuat akun default di URL yang bisa dibuka siapa saja — buku
     * ini berisi data keuangan klien, dan sandi bawaan yang terlanjur hidup di
     * server tidak pernah sempat diganti.
     */
    public function run(): void
    {
        $produksi = app()->isProduction();
        [$email, $sandi] = $this->kredensialAdmin($produksi);

        $admin = User::firstOrCreate(
            ['email' => $email],
            [
                'name' => env('SEED_ADMIN_NAME', 'Admin'),
                'password' => $sandi,
                'role' => User::ROLE_ADMIN,
                'is_active' => true,
            ],
        );

        $this->command?->info($admin->wasRecentlyCreated
            ? "Admin dibuat: {$admin->email}"
            : "Admin sudah ada: {$admin->email} (sandi TIDAK diubah oleh seeder)");

        // Tarif pajak awal — aman di produksi (tanpa data demo, idempoten).
        $this->call(TaxMasterSeeder::class);

        // Klien contoh hanya untuk pengembangan. Di produksi buku dimulai kosong
        // supaya tidak ada entitas palsu nyangkut di laporan.
        if ($produksi && ! env('SEED_DEMO_CLIENTS', false)) {
            $this->command?->warn('Klien contoh dilewati (produksi). Buat klien sungguhan lewat menu Klien.');

            return;
        }

        $coa = app(ChartOfAccounts::class);

        foreach ([
            ['name' => 'SKINKU', 'type' => Client::TYPE_INTERNAL, 'notes' => 'Brand body care. Fee marketplace TikTok/Shopee ke akun 6108.'],
            ['name' => 'Klien Contoh', 'type' => Client::TYPE_EXTERNAL, 'notes' => 'Contoh klien jasa pembukuan — hapus kalau tidak dipakai.'],
        ] as $attributes) {
            $client = Client::firstOrCreate(
                ['slug' => Str::slug($attributes['name'])],
                $attributes + ['is_active' => true],
            );
            $coa->seedFor($client);
        }
    }

    /**
     * @return array{0:string,1:string} email & sandi admin
     *
     * @throws RuntimeException bila produksi tapi kredensialnya masih bawaan/lemah
     */
    private function kredensialAdmin(bool $produksi): array
    {
        $email = (string) env('SEED_ADMIN_EMAIL', 'admin@jurnal.test');
        $sandi = (string) env('SEED_ADMIN_PASSWORD', 'password123');

        if (! $produksi) {
            return [$email, $sandi];
        }

        $masalah = match (true) {
            env('SEED_ADMIN_EMAIL') === null => 'SEED_ADMIN_EMAIL belum diisi di .env.',
            env('SEED_ADMIN_PASSWORD') === null => 'SEED_ADMIN_PASSWORD belum diisi di .env.',
            SandiLemah::alasanTolak($sandi) !== null => 'SEED_ADMIN_PASSWORD ditolak — '.SandiLemah::alasanTolak($sandi),
            str_ends_with($email, '.test') => 'SEED_ADMIN_EMAIL masih memakai domain contoh (.test).',
            default => null,
        };

        if ($masalah !== null) {
            throw new RuntimeException(
                "Seeder dihentikan: {$masalah}\n".
                "Aplikasi ini menyimpan data keuangan klien dan sedang dipasang di server publik.\n".
                "Isi SEED_ADMIN_EMAIL dan SEED_ADMIN_PASSWORD di .env, lalu jalankan ulang."
            );
        }

        return [$email, $sandi];
    }
}
