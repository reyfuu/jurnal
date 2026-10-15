<?php

use App\Models\Client;
use App\Services\Accounting\ChartOfAccounts;
use Illuminate\Database\Migrations\Migration;

/**
 * COA Dagang yang sudah ada → revisi akuntan 15 Okt 2026 (ChartOfAccounts::revisiDagang):
 * 4901→8101, 7xxx→9xxx (kode diganti di tempat, jurnal tetap utuh), 2103 dihapus
 * atau dinonaktifkan kalau sudah dipakai, akun baru ditambahkan.
 */
return new class extends Migration
{
    public function up(): void
    {
        $coa = app(ChartOfAccounts::class);
        Client::where('business_type', 'perdagangan')->each(fn (Client $c) => $coa->revisiDagang($c));
    }

    // ponytail: maju saja — mengembalikan kode bisa bentrok dengan akun yang dibuat setelahnya.
    public function down(): void {}
};
