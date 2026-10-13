<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

/**
 * Peran admin · supervisor · akuntan · anggota + penugasan klien.
 * Akuntan & anggota hanya melihat klien yang ditugaskan (client_user);
 * admin & supervisor melihat semua. Peran lama `staff` = akuntan.
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::create('client_user', function (Blueprint $table) {
            $table->foreignId('client_id')->constrained()->cascadeOnDelete();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->primary(['client_id', 'user_id']);
        });

        Schema::table('users', function (Blueprint $table) {
            $table->string('role', 20)->default('anggota')->change();
        });

        // Staff lama boleh melihat semua klien — penugasan disalin supaya akses tidak hilang.
        $staff = DB::table('users')->where('role', 'staff')->pluck('id');
        $clients = DB::table('clients')->pluck('id');
        foreach ($staff as $userId) {
            foreach ($clients as $clientId) {
                DB::table('client_user')->insertOrIgnore(['client_id' => $clientId, 'user_id' => $userId]);
            }
        }
        DB::table('users')->where('role', 'staff')->update(['role' => 'akuntan']);
    }

    public function down(): void
    {
        DB::table('users')->whereIn('role', ['akuntan', 'anggota', 'supervisor'])->update(['role' => 'staff']);
        Schema::table('users', function (Blueprint $table) {
            $table->string('role', 20)->default('staff')->change();
        });
        Schema::dropIfExists('client_user');
    }
};
