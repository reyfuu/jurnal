<?php

namespace App\Console\Commands;

use App\Models\User;
use App\Support\SandiLemah;
use Illuminate\Console\Command;
use Illuminate\Support\Facades\Hash;

/**
 * Periksa apakah aplikasi aman dibuka ke internet (lewat tunnel atau hosting).
 *
 * WAJIB dijalankan sebelum tunnel dinyalakan. Aplikasi ini
 * menyimpan pembukuan klien; begitu URL-nya publik, sandi bawaan dan halaman
 * error yang membocorkan isi .env berubah dari masalah sepele jadi serius.
 */
class CekPublikCommand extends Command
{
    protected $signature = 'jurnal:cek-publik';

    protected $description = 'Periksa kesiapan sebelum aplikasi dibuka ke internet';

    public function handle(): int
    {
        $masalah = array_values(array_filter([
            $this->cekAppKey(),
            $this->cekDebug(),
            $this->cekSandi(),
        ]));

        if ($masalah === []) {
            $this->info('Aman. Aplikasi siap dibuka lewat tunnel.');

            return self::SUCCESS;
        }

        $this->newLine();
        $this->error('BELUM AMAN untuk dibuka ke internet — '.count($masalah).' hal harus dibereskan:');
        foreach ($masalah as $i => [$judul, $caraBenerin]) {
            $this->newLine();
            $this->line('  '.($i + 1).'. <fg=yellow>'.$judul.'</>');
            $this->line('     Cara benerin: '.$caraBenerin);
        }
        $this->newLine();

        return self::FAILURE;
    }

    /** @return array{0:string,1:string}|null */
    private function cekAppKey(): ?array
    {
        return blank(config('app.key'))
            ? ['APP_KEY belum dibuat — sesi dan data terenkripsi tidak bisa dibuka.',
                'php artisan key:generate']
            : null;
    }

    /** @return array{0:string,1:string}|null */
    private function cekDebug(): ?array
    {
        // Halaman error Laravel dalam mode debug menampilkan seluruh isi .env,
        // termasuk kunci API dan sandi database.
        return config('app.debug')
            ? ['APP_DEBUG masih true — halaman error akan membocorkan isi .env (kunci API, sandi database).',
                'Buka .env, ubah jadi APP_DEBUG=false, lalu jalankan: php artisan config:clear']
            : null;
    }

    /** @return array{0:string,1:string}|null */
    private function cekSandi(): ?array
    {
        $lemah = User::all()->filter(
            fn (User $u) => collect(SandiLemah::BAWAAN)->contains(fn ($s) => Hash::check($s, $u->password)),
        );

        if ($lemah->isEmpty()) {
            return null;
        }

        return ['Masih ada user bersandi bawaan: '.$lemah->pluck('email')->implode(', '),
            'php artisan user:sandi '.$lemah->first()->email];
    }
}
