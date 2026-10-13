<?php

namespace App\Console\Commands;

use App\Models\User;
use App\Support\SandiLemah;
use Illuminate\Console\Command;

/**
 * Ganti sandi user tanpa menyentuh file apa pun.
 *
 * Dibuat karena satu-satunya cara mengganti sandi sebelumnya adalah mengedit
 * seeder atau .env lalu seed ulang — rawan salah dan tidak bisa dipakai saat
 * buku sudah berisi data.
 */
class GantiSandiCommand extends Command
{
    protected $signature = 'user:sandi {email? : Email user yang mau diganti sandinya}';

    protected $description = 'Ganti sandi user (ditanya interaktif, tidak tersimpan di riwayat terminal)';

    public function handle(): int
    {
        $email = $this->argument('email') ?: $this->pilihUser();
        if ($email === null) {
            return self::FAILURE;
        }

        $user = User::where('email', $email)->first();
        if (! $user) {
            $this->error("User dengan email {$email} tidak ditemukan.");

            return self::FAILURE;
        }

        // secret() tidak menampilkan ketikan dan tidak masuk riwayat terminal.
        $sandi = $this->secret('Sandi baru (minimal '.SandiLemah::PANJANG_MINIMAL.' karakter)');

        if ($alasan = SandiLemah::alasanTolak($sandi)) {
            $this->error($alasan.' Tidak ada yang diubah.');

            return self::FAILURE;
        }

        if ($sandi !== $this->secret('Ulangi sandi baru')) {
            $this->error('Dua sandi tidak sama. Tidak ada yang diubah.');

            return self::FAILURE;
        }

        $user->update(['password' => $sandi]);
        $this->info("Sandi untuk {$user->email} berhasil diganti.");

        return self::SUCCESS;
    }

    private function pilihUser(): ?string
    {
        $emails = User::orderBy('id')->pluck('email')->all();

        if ($emails === []) {
            $this->error('Belum ada user sama sekali. Jalankan: php artisan db:seed');

            return null;
        }

        return count($emails) === 1 ? $emails[0] : $this->choice('Ganti sandi siapa?', $emails);
    }
}
