<?php

namespace Tests\Feature;

use App\Models\Client;
use App\Models\Document;
use App\Models\Journal;
use App\Models\TaxEntry;
use App\Models\TaxRate;
use App\Models\User;
use App\Services\Accounting\ChartOfAccounts;
use App\Services\Accounting\JournalPoster;
use App\Services\Intake\DocumentExtractor;
use App\Services\Intake\JournalDrafter;
use App\Services\Tax\TaxCalculator;
use App\Services\Tax\TaxRecap;
use Database\Seeders\TaxMasterSeeder;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

/**
 * Perpajakan MVP (docs/perpajakan). Yang dijaga: rumus pajak tanpa float, nilai
 * dokumen tidak pernah ditimpa, arah jurnal PPN/PPh benar per profil klien,
 * catatan pajak = buku, dan isolasi klien.
 */
class TaxTest extends TestCase
{
    use RefreshDatabase;

    private Client $client;

    private User $admin;

    protected function setUp(): void
    {
        parent::setUp();
        $this->seed(TaxMasterSeeder::class);
        $this->client = $this->klien('PT Uji', ['is_pkp' => true, 'taxpayer_type' => 'badan']);
        $this->admin = User::create(['name' => 'Admin', 'email' => 'a@test.id', 'password' => 'secret123', 'role' => User::ROLE_ADMIN, 'is_active' => true]);
    }

    private function klien(string $name, array $profile = []): Client
    {
        $client = Client::create(['name' => $name, 'type' => Client::TYPE_EXTERNAL] + $profile);
        app(ChartOfAccounts::class)->seedFor($client);

        return $client;
    }

    private function akun(string $code, ?Client $client = null): int
    {
        return ($client ?? $this->client)->accounts()->where('code', $code)->value('id');
    }

    private function draft(array $extraction, ?Client $client = null): array
    {
        $doc = ($client ?? $this->client)->documents()->create([
            'kind' => Document::KIND_INVOICE, 'status' => Document::STATUS_EXTRACTED, 'extraction' => $extraction,
        ]);

        return app(JournalDrafter::class)->draft($doc)[0];
    }

    /** @return array<string,array{0:float,1:float}> kode => [debit, kredit] */
    private function sisi(array $draft): array
    {
        $out = [];
        foreach ($draft['lines'] as $l) {
            $out[$l['account_code']][0] = ($out[$l['account_code']][0] ?? 0) + $l['debit'];
            $out[$l['account_code']][1] = ($out[$l['account_code']][1] ?? 0) + $l['credit'];
        }

        return $out;
    }

    private function assertBalance(array $draft): void
    {
        $this->assertEqualsWithDelta(
            array_sum(array_column($draft['lines'], 'debit')),
            array_sum(array_column($draft['lines'], 'credit')), 0.001, 'Draft wajib balance.',
        );
    }

    private function invoice(array $taxes, string $flow = 'expense', int $subtotal = 1_000_000): array
    {
        return [
            'flow' => $flow, 'vendor' => 'CV Mitra', 'document_date' => '2026-10-05', 'payment_method' => 'bank',
            'items' => [['description' => 'Jasa', 'amount' => $subtotal, 'account_code' => $flow === 'income' ? '4102' : '6109', 'confidence' => 0.9]],
            'taxes' => $taxes,
        ];
    }

    // ── Kalkulator ──────────────────────────────────────────────────────────

    public function test_ppn_dpp_nilai_lain_tanpa_float(): void
    {
        $calc = app(TaxCalculator::class);
        $rate = $calc->rateFor('ppn', '2026-10-05');

        $this->assertSame(110000, $calc->compute(1_000_000, $rate));
        $this->assertSame('match', $calc->check('ppn', 1_000_000, 110000, '2026-10-05')['status']);
        $this->assertSame('mismatch', $calc->check('ppn', 1_000_000, 120000, '2026-10-05')['status']);

        // Dokumen Coretax mencantumkan DPP nilai lain → tetap cocok, base dikembalikan ke harga jual.
        $alt = $calc->check('ppn', 916_667, 110000, '2026-10-05', gross: 1_000_000);
        $this->assertSame('match', $alt['status']);
        $this->assertSame(1_000_000, $alt['base']);
        $this->assertSame('mismatch', $calc->check('ppn', 916_667, 110000, '2026-10-05')['status'], 'Tanpa harga jual, tafsiran DPP nilai lain tidak boleh diasumsikan.');

        // Tarif tanpa NPWP diterima, dengan catatan.
        $this->assertNotNull($calc->check('pph23', 5_000_000, 200000, '2026-10-05')['note']);
        $this->assertSame('no_rate', $calc->check('pph23', 5_000_000, 100000, '2020-01-01')['status']);
    }

    public function test_versi_tarif_dipilih_per_tanggal(): void
    {
        TaxRate::where('tax_code', 'pph23')->update(['effective_to' => '2026-06-30']);
        TaxRate::create(['tax_code' => 'pph23', 'rate_bp' => 300, 'effective_from' => '2026-07-01']);
        $calc = app(TaxCalculator::class);

        $this->assertSame(200, $calc->rateFor('pph23', '2026-06-30')->rate_bp);
        $this->assertSame(300, $calc->rateFor('pph23', '2026-07-01')->rate_bp);
    }

    public function test_jatuh_tempo_akhir_pekan_bergeser_ke_senin(): void
    {
        // 15 Nov 2026 = Minggu → Senin 16 Nov. Akhir Nov 2026 = Senin 30.
        $this->assertSame('2026-11-16', TaxRecap::dueDate('2026-10', 15));
        $this->assertSame('2026-11-30', TaxRecap::dueDate('2026-10', null));
    }

    // ── Ekstraksi ───────────────────────────────────────────────────────────

    public function test_pajak_hasil_ai_dicuci_dan_selisih_jadi_warning_tanpa_menimpa(): void
    {
        $out = app(DocumentExtractor::class)->normalize([
            'flow' => 'expense', 'document_date' => '2026-10-05', 'total' => '1.095.000',
            'items' => [['description' => 'Jasa', 'amount' => '1.000.000', 'account_code' => '6109']],
            'taxes' => [
                ['type' => 'ppn', 'base' => '1.000.000', 'amount' => 'Rp 115.000'],
                ['type' => 'pph23', 'amount' => '(20.000)'],
                ['type' => 'karangan', 'amount' => 5000, 'withheld' => false],
            ],
        ], $this->client);

        [$ppn, $pph, $lain] = $out['taxes'];
        $this->assertSame(115000, $ppn['amount'], 'Nilai dokumen tidak boleh diganti hitungan sistem.');
        $this->assertSame('mismatch', $ppn['match_status']);
        $this->assertSame(110000, $ppn['computed']);
        $this->assertTrue($pph['withheld'], 'PPh 23 default dipotong.');
        $this->assertSame(20000, $pph['amount']);
        $this->assertSame('other', $lain['type']);
        $this->assertTrue(collect($out['warnings'])->contains(fn ($w) => str_contains($w, 'PPN') && str_contains($w, '110.000')));
    }

    // ── Drafter ─────────────────────────────────────────────────────────────

    public function test_pkp_beli_ppn_dan_potong_pph23(): void
    {
        $draft = $this->draft($this->invoice([
            ['type' => 'ppn', 'amount' => 110000, 'withheld' => false, 'base' => 1_000_000],
            ['type' => 'pph23', 'amount' => 20000, 'withheld' => true, 'base' => 1_000_000],
        ]));
        $s = $this->sisi($draft);

        $this->assertBalance($draft);
        $this->assertSame(110000.0, $s['1402'][0], 'PPN → PPN Masukan (debit).');
        $this->assertSame(20000.0, $s['2108'][1], 'PPh 23 dipotong → Utang PPh 23 (kredit).');
        $this->assertSame(1090000.0, $s['1102'][1], 'Yang dibayar = 1.000.000 + PPN − PPh 23.');
        $this->assertCount(2, $draft['taxes']);
    }

    public function test_pkp_jual_ppn_keluaran_dan_dipotong_pelanggan(): void
    {
        $draft = $this->draft($this->invoice([
            ['type' => 'ppn', 'amount' => 110000, 'withheld' => false],
            ['type' => 'pph23', 'amount' => 20000, 'withheld' => true],
        ], 'income'));
        $s = $this->sisi($draft);

        $this->assertBalance($draft);
        $this->assertSame(110000.0, $s['2106'][1], 'PPN penjualan → PPN Keluaran.');
        $this->assertSame(20000.0, $s['1403'][0], 'Dipotong pelanggan → PPh Dibayar Dimuka.');
        $this->assertSame(1090000.0, $s['1102'][0]);
    }

    public function test_non_pkp_ppn_masuk_biaya(): void
    {
        $nonPkp = $this->klien('Toko Bu Sari', ['is_pkp' => false, 'pph_regime' => 'final_umkm', 'taxpayer_type' => 'orang_pribadi']);
        $draft = $this->draft($this->invoice([['type' => 'ppn', 'amount' => 110000, 'withheld' => false]]), $nonPkp);
        $s = $this->sisi($draft);

        $this->assertBalance($draft);
        $this->assertArrayNotHasKey('1402', $s, 'Non-PKP tidak boleh punya PPN Masukan.');
        $this->assertSame(1110000.0, $s['6109'][0]);
        $this->assertTrue($draft['taxes'][0]['noncreditable']);
    }

    public function test_profil_kosong_tetap_perilaku_lama(): void
    {
        $lama = $this->klien('Klien Lama');
        $draft = $this->draft(['flow' => 'expense', 'document_date' => '2026-10-05', 'payment_method' => 'cash', 'tax' => 110000,
            'items' => [['description' => 'Barang', 'amount' => 1_000_000, 'account_code' => '6190']]], $lama);

        $this->assertSame(110000.0, $this->sisi($draft)['1402'][0]);
    }

    // ── Posting ─────────────────────────────────────────────────────────────

    private function postPayload(array $draft, array $override = []): array
    {
        return ['drafts' => [array_replace_recursive([
            'post' => '1', 'date' => $draft['date'], 'description' => $draft['description'], 'type' => 'purchase',
            'fingerprint' => $draft['fingerprint'],
            'lines' => array_map(fn ($l) => ['account_id' => $l['account_id'], 'debit' => $l['debit'], 'credit' => $l['credit'], 'memo' => $l['memo']], $draft['lines']),
            'taxes' => array_map(fn ($t) => array_intersect_key($t, array_flip(['tax_code', 'account_id', 'direction', 'withheld', 'noncreditable', 'base', 'amount', 'document_number', 'counterparty_name'])), $draft['taxes']),
        ], $override)]];
    }

    public function test_posting_menyimpan_catatan_pajak_dari_baris_jurnal_bukan_dari_input(): void
    {
        $draft = $this->draft($this->invoice([['type' => 'ppn', 'amount' => 110000, 'withheld' => false, 'base' => 1_000_000]]));
        $doc = Document::latest('id')->first();
        // Hidden input dipalsukan jadi 999 — harus diabaikan.
        $payload = $this->postPayload($draft, ['taxes' => [['amount' => 999]]]);

        $this->actingAs($this->admin)->post(route('documents.post', [$this->client, $doc]), $payload)->assertRedirect();

        $entry = TaxEntry::sole();
        $this->assertSame(110000, (int) $entry->amount);
        $this->assertSame('match', $entry->match_status);
        $this->assertSame($this->client->id, $entry->client_id);
        $this->assertNotNull($entry->tax_rate_id);
    }

    public function test_catatan_pajak_tanpa_baris_jurnal_ditolak_dan_tidak_ada_yang_masuk(): void
    {
        $draft = $this->draft($this->invoice([['type' => 'ppn', 'amount' => 110000, 'withheld' => false]]));
        $doc = Document::latest('id')->first();
        // Akun pajak diarahkan ke akun yang tidak ada di baris jurnal.
        $payload = $this->postPayload($draft, ['taxes' => [['account_id' => $this->akun('2106'), 'direction' => 'out']]]);

        $this->actingAs($this->admin)->post(route('documents.post', [$this->client, $doc]), $payload)->assertSessionHasErrors('posting');

        $this->assertSame(0, Journal::count());
        $this->assertSame(0, TaxEntry::count());
    }

    // ── Rekap ───────────────────────────────────────────────────────────────

    private function jurnal(Client $client, string $date, array $lines, array $taxes = []): Journal
    {
        return app(JournalPoster::class)->record(['client_id' => $client->id, 'date' => $date], $lines, Journal::STATUS_POSTED, $taxes);
    }

    public function test_rekap_ppn_void_keluar_dan_cek_silang_buku(): void
    {
        $c = $this->client;
        $tax = fn ($amount, $code, $dir) => [['tax_code' => 'ppn', 'account_id' => $this->akun($code), 'direction' => $dir, 'base' => $amount * 10, 'amount' => $amount, 'match_status' => 'manual']];

        $this->jurnal($c, '2026-10-02', [['account_id' => $this->akun('1102'), 'debit' => 2_220_000], ['account_id' => $this->akun('4101'), 'credit' => 2_000_000], ['account_id' => $this->akun('2106'), 'credit' => 220_000]], $tax(220_000, '2106', 'out'));
        $this->jurnal($c, '2026-10-03', [['account_id' => $this->akun('6109'), 'debit' => 1_000_000], ['account_id' => $this->akun('1402'), 'debit' => 110_000], ['account_id' => $this->akun('1102'), 'credit' => 1_110_000]], $tax(110_000, '1402', 'in'));
        $void = $this->jurnal($c, '2026-10-04', [['account_id' => $this->akun('6109'), 'debit' => 500_000], ['account_id' => $this->akun('1402'), 'debit' => 55_000], ['account_id' => $this->akun('1102'), 'credit' => 555_000]], $tax(55_000, '1402', 'in'));
        app(JournalPoster::class)->void($void);

        $ppn = app(TaxRecap::class)->ppn($c, '2026-10');
        $this->assertSame(220_000, $ppn['keluaran']);
        $this->assertSame(110_000, $ppn['masukan'], 'Jurnal void tidak ikut rekap.');
        $this->assertSame(110_000, $ppn['net']);
        $this->assertSame([], app(TaxRecap::class)->reconcile($c, '2026-10'));

        // Jurnal manual menyentuh akun pajak tanpa catatan → terdeteksi.
        $this->jurnal($c, '2026-10-05', [['account_id' => $this->akun('1402'), 'debit' => 1_000], ['account_id' => $this->akun('1101'), 'credit' => 1_000]]);
        $this->assertSame('1402', app(TaxRecap::class)->reconcile($c, '2026-10')[0]['account']->code);
    }

    public function test_final_umkm_orang_pribadi_melewati_batas_bebas(): void
    {
        $op = $this->klien('Bu Sari', ['is_pkp' => false, 'pph_regime' => 'final_umkm', 'taxpayer_type' => 'orang_pribadi']);
        $jual = fn ($date, $amount) => $this->jurnal($op, $date, [['account_id' => $this->akun('1101', $op), 'debit' => $amount], ['account_id' => $this->akun('4101', $op), 'credit' => $amount]]);
        $jual('2026-01-10', 450_000_000);
        $jual('2026-02-10', 100_000_000);

        $feb = app(TaxRecap::class)->finalUmkm($op, '2026-02');
        $this->assertSame(50_000_000, $feb['exempt'], 'Sisa bebas tahun ini Rp50 jt.');
        $this->assertSame(50_000_000, $feb['taxable']);
        $this->assertSame(250_000, $feb['tax'], '0,5% × 50 jt.');

        $badan = $this->klien('CV Kecil', ['is_pkp' => false, 'pph_regime' => 'final_umkm', 'taxpayer_type' => 'badan']);
        $this->jurnal($badan, '2026-01-10', [['account_id' => $this->akun('1101', $badan), 'debit' => 10_000_000], ['account_id' => $this->akun('4101', $badan), 'credit' => 10_000_000]]);
        $this->assertSame(50_000, app(TaxRecap::class)->finalUmkm($badan, '2026-01')['tax'], 'Badan tanpa bagian bebas.');
    }

    // ── Akses, isolasi, ekspor ──────────────────────────────────────────────

    public function test_staff_tidak_bisa_ubah_profil_atau_tarif(): void
    {
        $staff = User::create(['name' => 'Staff', 'email' => 's@test.id', 'password' => 'secret123', 'role' => User::ROLE_AKUNTAN, 'is_active' => true]);
        $staff->clients()->attach($this->client);

        $this->actingAs($staff)->put(route('tax.profile.update', $this->client), ['is_pkp' => 0, 'pph_regime' => 'normal'])->assertForbidden();
        $this->actingAs($staff)->post(route('tax.rates.store'), [])->assertForbidden();
        $this->actingAs($staff)->get(route('tax.index', $this->client))->assertOk();
        $this->assertTrue($this->client->fresh()->is_pkp);
    }

    public function test_profil_npwp_terenkripsi_dan_tercatat_di_audit(): void
    {
        $this->actingAs($this->admin)->put(route('tax.profile.update', $this->client), [
            'taxpayer_type' => 'badan', 'tax_id' => '01.234.567.8-901.234', 'is_pkp' => 1, 'pph_regime' => 'normal',
        ])->assertRedirect(route('tax.index', $this->client));

        $this->assertSame('012345678901234', $this->client->fresh()->tax_id, 'Format lama 15 digit dinormalisasi ke angka saja.');
        $raw = \Illuminate\Support\Facades\DB::table('clients')->where('id', $this->client->id)->value('tax_id');
        $this->assertStringNotContainsString('0123456789', $raw, 'NPWP tidak boleh tersimpan polos.');
        $log = \App\Models\AuditLog::sole();
        $this->assertStringNotContainsString('0123456789', json_encode($log->after), 'Audit menyimpan NPWP termasking.');
    }

    public function test_rekap_tidak_bocor_antar_klien_dan_ekspor_aman(): void
    {
        $lain = $this->klien('Klien Lain', ['is_pkp' => true]);
        $this->jurnal($lain, '2026-10-02', [['account_id' => $this->akun('1102', $lain), 'debit' => 1_110_000], ['account_id' => $this->akun('4101', $lain), 'credit' => 1_000_000], ['account_id' => $this->akun('2106', $lain), 'credit' => 110_000]],
            [['tax_code' => 'ppn', 'account_id' => $this->akun('2106', $lain), 'direction' => 'out', 'base' => 1_000_000, 'amount' => 110_000, 'match_status' => 'manual', 'counterparty_name' => '=HYPERLINK("x")']]);

        $this->assertSame(0, app(TaxRecap::class)->ppn($this->client, '2026-10')['keluaran']);

        $csv = $this->actingAs($this->admin)->get(route('tax.export', [$lain, 'ppn']).'?period=2026-10')->assertOk()->streamedContent();
        $this->assertStringContainsString(config('pajak.disclaimer'), $csv);
        $this->assertStringContainsString("'=HYPERLINK", $csv, 'Formula injection wajib dinetralkan.');
        $this->assertSame("'=1+1", \App\Support\Csv::safe('=1+1'));
    }

    // ── Fase 2 ──────────────────────────────────────────────────────────────

    public function test_jurnal_manual_pph21_tercatat_otomatis_setoran_tidak(): void
    {
        $post = fn (array $lines, string $date = '2026-10-25') => $this->actingAs($this->admin)
            ->post(route('journals.store', $this->client), ['date' => $date, 'lines' => $lines])->assertRedirect();

        // Gaji 10 jt, PPh 21 dipotong 250 rb (nilai diisi manusia).
        $post([['account_id' => $this->akun('6101'), 'debit' => 10_000_000], ['account_id' => $this->akun('2107'), 'credit' => 250_000], ['account_id' => $this->akun('1102'), 'credit' => 9_750_000]]);
        // Setoran bulan yang sama: sisi berlawanan, bukan kejadian pajak.
        $post([['account_id' => $this->akun('2107'), 'debit' => 250_000], ['account_id' => $this->akun('1102'), 'credit' => 250_000]], '2026-10-30');

        $entry = TaxEntry::sole();
        $this->assertSame('pph21', $entry->tax_code);
        $this->assertTrue($entry->withheld);
        $this->assertSame('manual', $entry->match_status);
        $this->assertSame([], app(TaxRecap::class)->reconcile($this->client, '2026-10'), 'Setoran tidak boleh dianggap selisih.');
        $this->assertSame('pph21', collect(app(TaxRecap::class)->obligations($this->client, '2026-10'))->pluck('tax_code')->last());
    }

    public function test_draft_jurnal_dari_rekap_hanya_mengisi_form(): void
    {
        $op = $this->klien('Bu Sari', ['is_pkp' => false, 'pph_regime' => 'final_umkm', 'taxpayer_type' => 'badan']);
        $this->jurnal($op, '2026-10-10', [['account_id' => $this->akun('1101', $op), 'debit' => 20_000_000], ['account_id' => $this->akun('4101', $op), 'credit' => 20_000_000]]);

        $this->actingAs($this->admin)->get(route('tax.draft', [$op, 'final-akrual']).'?period=2026-10')
            ->assertRedirect(route('journals.create', $op))
            ->assertSessionHasInput('lines.0.debit', 100_000);
        $this->assertSame(1, Journal::count(), 'Draft tidak boleh langsung diposting.');

        $this->actingAs($this->admin)->get(route('tax.draft', [$op, 'ppn-setor']).'?period=2026-10')->assertNotFound();
        $this->actingAs($this->admin)->get(route('tax.draft', [$op, 'ngawur']))->assertNotFound();
    }

    public function test_dashboard_menampilkan_pajak_jatuh_tempo_lintas_klien(): void
    {
        $this->travelTo('2026-11-03');
        $this->actingAs($this->admin)->get(route('dashboard'))
            ->assertSee('Pajak jatuh tempo bulan ini')
            ->assertSee('PT Uji')
            ->assertSee('2026-10');
    }
}
