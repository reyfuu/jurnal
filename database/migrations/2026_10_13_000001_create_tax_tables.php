<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Crypt;
use Illuminate\Support\Facades\Schema;

/**
 * Perpajakan MVP (docs/perpajakan/TRD.md §2). Additive saja: tidak ada kolom
 * lama yang diubah/dihapus. `clients.npwp` dibiarkan; isinya disalin terenkripsi
 * ke `tax_id`.
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::table('clients', function (Blueprint $table) {
            $table->string('taxpayer_type', 20)->nullable();   // orang_pribadi | badan
            $table->text('tax_id')->nullable();                // NPWP/NIK, terenkripsi
            $table->boolean('is_pkp')->nullable();             // null = profil belum diisi (diperlakukan PKP, perilaku lama)
            $table->date('pkp_since')->nullable();
            $table->string('pph_regime', 20)->default('normal'); // normal | final_umkm
        });

        foreach (DB::table('clients')->whereNotNull('npwp')->where('npwp', '!=', '')->get(['id', 'npwp']) as $row) {
            DB::table('clients')->where('id', $row->id)->update(['tax_id' => Crypt::encryptString(preg_replace('/\D/', '', $row->npwp))]);
        }

        Schema::create('tax_rates', function (Blueprint $table) {
            $table->id();
            $table->string('tax_code', 20);
            $table->string('applies_to', 20)->nullable();      // orang_pribadi | badan | null = semua
            $table->unsignedInteger('rate_bp');                // basis poin: 12% = 1200
            $table->unsignedInteger('rate_no_tax_id_bp')->nullable();
            $table->unsignedInteger('dpp_numerator')->default(1);
            $table->unsignedInteger('dpp_denominator')->default(1);
            $table->unsignedBigInteger('exempt_threshold')->nullable();
            $table->string('rounding', 10)->default('floor');  // floor | half_up
            $table->date('effective_from');
            $table->date('effective_to')->nullable();
            $table->string('source')->nullable();
            $table->text('notes')->nullable();
            $table->timestamp('verified_at')->nullable();
            $table->foreignId('created_by')->nullable()->constrained('users')->nullOnDelete();
            $table->timestamps();

            $table->unique(['tax_code', 'applies_to', 'effective_from']);
        });

        Schema::create('tax_entries', function (Blueprint $table) {
            $table->id();
            $table->foreignId('client_id')->constrained()->cascadeOnDelete();
            $table->foreignId('journal_id')->constrained()->cascadeOnDelete();
            $table->foreignId('account_id')->constrained();
            $table->foreignId('document_id')->nullable()->constrained()->nullOnDelete();
            $table->foreignId('tax_rate_id')->nullable()->constrained()->nullOnDelete();
            $table->string('tax_code', 20);
            $table->string('direction', 4);                    // in = aset/kredit pajak · out = liabilitas
            $table->boolean('withheld')->default(false);
            $table->boolean('noncreditable')->default(false);  // PPN klien non-PKP → masuk biaya
            $table->unsignedBigInteger('base')->default(0);
            $table->unsignedBigInteger('amount');
            $table->unsignedBigInteger('computed_amount')->nullable();
            $table->unsignedInteger('rate_bp_snapshot')->nullable();
            $table->string('match_status', 10);                // match | mismatch | no_rate | manual
            $table->string('counterparty_name')->nullable();
            $table->text('counterparty_tax_id')->nullable();   // terenkripsi
            $table->string('document_number', 60)->nullable();
            $table->foreignId('created_by')->nullable()->constrained('users')->nullOnDelete();
            $table->timestamps();

            $table->index(['client_id', 'tax_code']);
        });

        Schema::create('tax_filings', function (Blueprint $table) {
            $table->id();
            $table->foreignId('client_id')->constrained()->cascadeOnDelete();
            $table->string('tax_code', 20);
            $table->string('period', 7);
            $table->string('status', 10)->default('belum');    // belum | setor | lapor | nihil
            $table->unsignedBigInteger('paid_amount')->nullable();
            $table->date('paid_at')->nullable();
            $table->string('ntpn', 16)->nullable();
            $table->date('filed_at')->nullable();
            $table->string('receipt_number', 60)->nullable();
            $table->text('note')->nullable();
            $table->foreignId('updated_by')->nullable()->constrained('users')->nullOnDelete();
            $table->timestamps();

            $table->unique(['client_id', 'tax_code', 'period']);
        });

        Schema::create('audit_logs', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->nullable()->constrained()->nullOnDelete();
            $table->foreignId('client_id')->nullable()->constrained()->nullOnDelete();
            $table->string('subject_type', 40);
            $table->unsignedBigInteger('subject_id')->nullable();
            $table->string('action', 40);
            $table->json('before')->nullable();
            $table->json('after')->nullable();
            $table->timestamp('created_at')->useCurrent();

            $table->index(['client_id', 'created_at']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('audit_logs');
        Schema::dropIfExists('tax_filings');
        Schema::dropIfExists('tax_entries');
        Schema::dropIfExists('tax_rates');
        Schema::table('clients', function (Blueprint $table) {
            $table->dropColumn(['taxpayer_type', 'tax_id', 'is_pkp', 'pkp_since', 'pph_regime']);
        });
    }
};
