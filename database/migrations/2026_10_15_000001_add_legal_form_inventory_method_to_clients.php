<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/**
 * Isian awal perusahaan yang diminta akuntan (Daftar COA.xlsx, catatan #3 & #11):
 * bentuk badan usaha (PT → Utang Dividen, CV/perorangan → Prive) dan metode
 * persediaan (perpetual → HPP, periodik → Pembelian/Ongkos Angkut/Retur).
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::table('clients', function (Blueprint $table) {
            $table->string('legal_form', 20)->nullable()->after('business_type');
            $table->string('inventory_method', 20)->default('perpetual')->after('legal_form');
        });
    }

    public function down(): void
    {
        Schema::table('clients', function (Blueprint $table) {
            $table->dropColumn(['legal_form', 'inventory_method']);
        });
    }
};
