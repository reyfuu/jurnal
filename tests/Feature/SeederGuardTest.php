<?php

namespace Tests\Feature;

use App\Models\Client;
use App\Models\User;
use Database\Seeders\DatabaseSeeder;
use Illuminate\Foundation\Testing\RefreshDatabase;
use RuntimeException;
use Tests\TestCase;

/**
 * Pagar kredensial saat dipasang di server publik. Tanpa ini, satu perintah
 * `db:seed` di produksi menghidupkan akun admin bawaan di URL yang bisa dibuka
 * siapa saja — padahal isinya pembukuan klien.
 */
class SeederGuardTest extends TestCase
{
    use RefreshDatabase;

    /**
     * Jalankan seeder langsung, bukan lewat `artisan db:seed` — di mode produksi
     * perintah itu meminta konfirmasi interaktif yang tidak ada di test.
     */
    private function seedProduksi(array $env): void
    {
        app()['env'] = 'production';
        foreach ($env as $k => $v) {
            if ($v === null) {
                putenv($k);
                unset($_ENV[$k]);

                continue;
            }
            putenv("{$k}={$v}");
            $_ENV[$k] = $v;
        }

        $seeder = new DatabaseSeeder;
        $seeder->setContainer(app());
        $seeder->run();
    }

    protected function tearDown(): void
    {
        foreach (['SEED_ADMIN_EMAIL', 'SEED_ADMIN_PASSWORD', 'SEED_DEMO_CLIENTS'] as $k) {
            putenv($k);
            unset($_ENV[$k]);
        }
        parent::tearDown();
    }

    public function test_produksi_tanpa_kredensial_ditolak(): void
    {
        $this->expectException(RuntimeException::class);
        $this->expectExceptionMessageMatches('/SEED_ADMIN_EMAIL belum diisi/');

        $this->seedProduksi(['SEED_ADMIN_EMAIL' => null, 'SEED_ADMIN_PASSWORD' => null]);
    }

    public function test_produksi_dengan_sandi_bawaan_ditolak(): void
    {
        $this->expectException(RuntimeException::class);
        $this->expectExceptionMessageMatches('/gampang ditebak/');

        $this->seedProduksi([
            'SEED_ADMIN_EMAIL' => 'freddie@skinku.id',
            // Panjangnya cukup; yang diuji aturan kandungan kata, bukan panjang.
            'SEED_ADMIN_PASSWORD' => 'password12345',
        ]);

        $this->assertSame(0, User::count());
    }

    public function test_produksi_dengan_sandi_pendek_ditolak(): void
    {
        $this->expectException(RuntimeException::class);
        $this->expectExceptionMessageMatches('/minimal 12 karakter/');

        $this->seedProduksi([
            'SEED_ADMIN_EMAIL' => 'freddie@skinku.id',
            'SEED_ADMIN_PASSWORD' => 'pendek1',
        ]);
    }

    public function test_produksi_dengan_email_domain_contoh_ditolak(): void
    {
        $this->expectException(RuntimeException::class);
        $this->expectExceptionMessageMatches('/domain contoh/');

        $this->seedProduksi([
            'SEED_ADMIN_EMAIL' => 'admin@jurnal.test',
            'SEED_ADMIN_PASSWORD' => 'KunciBukuAman2026',
        ]);
    }

    public function test_produksi_dengan_kredensial_benar_berhasil_tanpa_klien_contoh(): void
    {
        $this->seedProduksi([
            'SEED_ADMIN_EMAIL' => 'freddie@skinku.id',
            'SEED_ADMIN_PASSWORD' => 'KunciBukuAman2026',
        ]);

        $admin = User::sole();
        $this->assertSame('freddie@skinku.id', $admin->email);
        $this->assertTrue($admin->isAdmin());
        $this->assertTrue(\Illuminate\Support\Facades\Hash::check('KunciBukuAman2026', $admin->password));

        // Buku produksi mulai kosong — tidak ada entitas palsu di laporan.
        $this->assertSame(0, Client::count());
    }

    public function test_klien_contoh_bisa_diminta_eksplisit_di_produksi(): void
    {
        $this->seedProduksi([
            'SEED_ADMIN_EMAIL' => 'freddie@skinku.id',
            'SEED_ADMIN_PASSWORD' => 'KunciBukuAman2026',
            'SEED_DEMO_CLIENTS' => 'true',
        ]);

        $this->assertSame(2, Client::count());
        $this->assertGreaterThan(0, Client::where('slug', 'skinku')->sole()->accounts()->count());
    }

    public function test_lokal_tetap_boleh_pakai_kredensial_bawaan(): void
    {
        // Testing lokal tidak boleh ikut terhambat pagar produksi.
        (new DatabaseSeeder)->setContainer(app())->run();

        $this->assertSame('admin@jurnal.test', User::sole()->email);
        $this->assertSame(2, Client::count());
    }

    public function test_seeder_tidak_mengubah_sandi_admin_yang_sudah_ada(): void
    {
        // Menjalankan ulang seeder setelah deploy tidak boleh mereset sandi.
        (new DatabaseSeeder)->setContainer(app())->run();
        User::sole()->update(['password' => 'KunciDiubahManual26']);

        (new DatabaseSeeder)->setContainer(app())->run();

        $this->assertTrue(\Illuminate\Support\Facades\Hash::check('KunciDiubahManual26', User::sole()->password));
    }
}
