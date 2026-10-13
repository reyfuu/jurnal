<?php

namespace Tests\Feature;

use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

/**
 * Pagar sebelum aplikasi dibuka ke internet lewat tunnel. Dijalankan sebelum tunnel;
 * kalau gagal, tunnel tidak dinyalakan sama sekali.
 */
class CekPublikTest extends TestCase
{
    use RefreshDatabase;

    private function user(string $sandi, string $email = 'admin@jurnal.test'): User
    {
        return User::create([
            'name' => 'Admin', 'email' => $email, 'password' => $sandi,
            'role' => User::ROLE_ADMIN, 'is_active' => true,
        ]);
    }

    public function test_sandi_bawaan_menggagalkan_pemeriksaan(): void
    {
        config(['app.debug' => false, 'app.key' => 'base64:'.base64_encode(random_bytes(32))]);
        $this->user('password123');

        $this->artisan('jurnal:cek-publik')
            ->expectsOutputToContain('BELUM AMAN')
            ->expectsOutputToContain('admin@jurnal.test')
            ->assertExitCode(1);
    }

    public function test_app_debug_menyala_menggagalkan_pemeriksaan(): void
    {
        config(['app.debug' => true, 'app.key' => 'base64:'.base64_encode(random_bytes(32))]);
        $this->user('KunciBukuAman2026');

        $this->artisan('jurnal:cek-publik')
            ->expectsOutputToContain('APP_DEBUG')
            ->assertExitCode(1);
    }

    public function test_app_key_kosong_menggagalkan_pemeriksaan(): void
    {
        config(['app.debug' => false, 'app.key' => null]);
        $this->user('KunciBukuAman2026');

        $this->artisan('jurnal:cek-publik')
            ->expectsOutputToContain('APP_KEY')
            ->assertExitCode(1);
    }

    public function test_semua_beres_lolos(): void
    {
        config(['app.debug' => false, 'app.key' => 'base64:'.base64_encode(random_bytes(32))]);
        $this->user('KunciBukuAman2026');

        $this->artisan('jurnal:cek-publik')
            ->expectsOutputToContain('Aman')
            ->assertExitCode(0);
    }

    public function test_satu_user_lemah_di_antara_yang_aman_tetap_terdeteksi(): void
    {
        config(['app.debug' => false, 'app.key' => 'base64:'.base64_encode(random_bytes(32))]);
        $this->user('KunciBukuAman2026', 'aman@skinku.id');
        $this->user('qwerty123', 'lemah@skinku.id');

        $this->artisan('jurnal:cek-publik')
            ->expectsOutputToContain('lemah@skinku.id')
            ->assertExitCode(1);
    }

    public function test_ganti_sandi_menolak_sandi_pendek(): void
    {
        $user = $this->user('KunciBukuAman2026', 'admin@skinku.id');

        $this->artisan('user:sandi', ['email' => 'admin@skinku.id'])
            ->expectsQuestion('Sandi baru (minimal 12 karakter)', 'pendek')
            ->expectsOutputToContain('minimal 12 karakter')
            ->assertExitCode(1);

        $this->assertTrue(\Illuminate\Support\Facades\Hash::check('KunciBukuAman2026', $user->fresh()->password));
    }

    public function test_ganti_sandi_menolak_kalau_ulangan_tidak_sama(): void
    {
        $user = $this->user('KunciBukuAman2026', 'admin@skinku.id');

        $this->artisan('user:sandi', ['email' => 'admin@skinku.id'])
            ->expectsQuestion('Sandi baru (minimal 12 karakter)', 'KunciBaruPanjang26')
            ->expectsQuestion('Ulangi sandi baru', 'KunciLainPanjang26')
            ->expectsOutputToContain('tidak sama')
            ->assertExitCode(1);

        $this->assertTrue(\Illuminate\Support\Facades\Hash::check('KunciBukuAman2026', $user->fresh()->password));
    }

    public function test_ganti_sandi_menolak_sandi_umum_walau_panjang(): void
    {
        $this->user('KunciBukuAman2026', 'admin@skinku.id');

        $this->artisan('user:sandi', ['email' => 'admin@skinku.id'])
            ->expectsQuestion('Sandi baru (minimal 12 karakter)', 'password1234')
            ->expectsOutputToContain('gampang ditebak')
            ->assertExitCode(1);
    }

    public function test_ganti_sandi_berhasil_dan_bisa_dipakai_login(): void
    {
        $this->user('password123', 'admin@skinku.id');

        $this->artisan('user:sandi', ['email' => 'admin@skinku.id'])
            ->expectsQuestion('Sandi baru (minimal 12 karakter)', 'KunciBaruKuat2026')
            ->expectsQuestion('Ulangi sandi baru', 'KunciBaruKuat2026')
            ->expectsOutputToContain('berhasil diganti')
            ->assertExitCode(0);

        $this->post(route('login'), ['email' => 'admin@skinku.id', 'password' => 'KunciBaruKuat2026'])
            ->assertRedirect(route('dashboard'));
        $this->assertAuthenticated();
    }

    public function test_ganti_sandi_email_tidak_ada_ditolak(): void
    {
        $this->user('KunciBukuAman2026', 'admin@skinku.id');

        $this->artisan('user:sandi', ['email' => 'bukan@siapa.id'])
            ->expectsOutputToContain('tidak ditemukan')
            ->assertExitCode(1);
    }
}
