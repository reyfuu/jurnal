<?php

namespace Tests\Feature;

use App\Models\Client;
use App\Models\User;
use App\Exceptions\AccountingException;
use App\Services\Accounting\ChartOfAccounts;
use App\Services\Accounting\JournalPoster;
use App\Services\Tax\TaxCalculator;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

/** Template COA per jenis usaha + COA tiap perusahaan tidak pernah bercampur. */
class CoaTemplateTest extends TestCase
{
    use RefreshDatabase;

    private function klien(string $nama, string $jenis): Client
    {
        $c = Client::create(['name' => $nama, 'type' => Client::TYPE_EXTERNAL, 'business_type' => $jenis]);
        app(ChartOfAccounts::class)->seedFor($c);

        return $c;
    }

    public function test_tiap_template_lengkap_untuk_mesin_dan_pajak(): void
    {
        $umum = ['1101', '1102', '1201', '1402', '1403', '2101', '2106', '2107', '2108', '2109', '2110', '4101', '6190'];
        $skema = [
            'perdagangan' => ['ada' => ['9103', '8101', '2116', '2117'], 'tidak' => ['2103', '7103', '4901']],
            'manufaktur' => ['ada' => ['7103', '4901', '2103'], 'tidak' => ['9103', '8101']],
            'jasa' => ['ada' => ['7103', '4901', '2103'], 'tidak' => ['9103', '8101']],
        ];
        $calc = app(TaxCalculator::class);

        foreach ($skema as $jenis => $harus) {
            $c = $this->klien("PT {$jenis}", $jenis);
            $akun = $c->accounts()->get();
            $kode = $akun->pluck('code');

            $this->assertSame($kode->count(), $kode->unique()->count(), "{$jenis}: kode dobel");
            $this->assertSame([], array_values(array_diff([...$umum, ...$harus['ada']], $kode->all())), "{$jenis}: akun wajib hilang");
            $this->assertSame([], array_values(array_intersect($harus['tidak'], $kode->all())), "{$jenis}: akun skema lain ikut");
            $this->assertNotNull($c->accounts()->whereIn('code', ChartOfAccounts::TAX_EXPENSE_CODES)->first(), "{$jenis}: beban pajak");

            // Tiap kode+arah pajak menemukan akunnya. Satu pengecualian yang disadari:
            // "pajak lain" yang dipungut di Dagang (tanpa 2103) → reviewer pilih akun sendiri.
            $aktif = $c->accounts()->active()->get()->keyBy('code');
            foreach (array_keys(config('pajak.codes')) as $code) {
                foreach (['in', 'out'] as $arah) {
                    if ($jenis === 'perdagangan' && $code === 'other' && $arah === 'out') {
                        $this->assertNull($calc->accountFor($aktif, $code, $arah));

                        continue;
                    }
                    if ($code === 'final_umkm' && $arah === 'in') {
                        continue; // PPh final tidak punya sisi kredit pajak
                    }
                    $this->assertNotNull($calc->accountFor($aktif, $code, $arah), "{$jenis}: akun {$code}/{$arah}");
                }
            }

            foreach ($akun->whereIn('subtype', ['contra_asset', 'contra_revenue', 'contra_cogs', 'drawing']) as $a) {
                $this->assertSame(in_array($a->type, ['asset', 'expense', 'equity']) && $a->subtype !== 'drawing' ? 'credit' : 'debit', $a->normal_balance, "{$jenis} {$a->code}");
            }
        }
    }

    /** Template Dagang = sheet "COA Baru" di Daftar COA.xlsx (72 akun) + 1407/2116/2117. */
    public function test_template_dagang_sesuai_daftar_akuntan(): void
    {
        $semua = collect([['pt', 'perpetual'], ['cv', 'periodik']])
            ->flatMap(fn ($o) => ChartOfAccounts::template('perdagangan', ['legal_form' => $o[0], 'inventory_method' => $o[1]]))
            ->keyBy('code');
        $this->assertCount(75, $semua);

        foreach ([
            ['1302', 'Perlengkapan', 'supplies', 'debit'], ['1406', 'PPN Lebih Bayar', 'vat_overpaid', 'debit'],
            ['1407', 'Piutang Karyawan', 'employee_receivable', 'debit'], ['2111', 'PPN Kurang Bayar', 'vat_underpaid', 'credit'],
            ['2113', 'Utang Karyawan', 'employee_payable', 'credit'], ['2201', 'Liabilitas Pajak Tangguhan', 'deferred_tax_liability', 'credit'],
            ['5105', 'Retur & Potongan Pembelian', 'contra_cogs', 'credit'], ['6117', 'Beban BPJS', 'opex', 'debit'],
            ['8101', 'Pendapatan Lain-lain', 'other_income', 'credit'], ['9104', 'Beban (Manfaat) Pajak Tangguhan', 'deferred_tax_expense', 'debit'],
        ] as [$code, $name, $subtype, $normal]) {
            $this->assertSame([$name, $subtype, $normal], [$semua[$code]['name'], $semua[$code]['subtype'], $semua[$code]['normal']], $code);
        }
    }

    public function test_kondisi_akun_dagang_mengikuti_bentuk_usaha_dan_metode_persediaan(): void
    {
        $kode = fn (Client $c) => $c->accounts()->pluck('code')->all();
        $pt = Client::create(['name' => 'PT Dagang', 'type' => Client::TYPE_EXTERNAL, 'legal_form' => 'pt', 'inventory_method' => 'perpetual']);
        $cv = Client::create(['name' => 'CV Dagang', 'type' => Client::TYPE_EXTERNAL, 'legal_form' => 'cv', 'inventory_method' => 'periodik']);
        foreach ([$pt, $cv] as $c) {
            app(ChartOfAccounts::class)->seedFor($c);
        }

        $this->assertContains('2115', $kode($pt));
        $this->assertNotContains('3102', $kode($pt));
        $this->assertContains('5101', $kode($pt));
        $this->assertEmpty(array_intersect(['5102', '5103', '5105'], $kode($pt)));

        $this->assertContains('3102', $kode($cv));
        $this->assertNotContains('2115', $kode($cv));
        $this->assertNotContains('5101', $kode($cv));
        $this->assertEmpty(array_diff(['5102', '5103', '5105'], $kode($cv)));

        // Gedung & kendaraan: ada tapi nonaktif sampai memang dimiliki.
        $this->assertSame([false, false, false, false], $pt->accounts()->whereIn('code', ['1503', '1504', '1505', '1506'])->pluck('is_active')->all());
    }

    /** Migrasi COA Dagang lama (sheet "Mapping Lama-Baru"): kode berpindah di tempat, jurnal tetap utuh. */
    public function test_revisi_dagang_memindah_kode_tanpa_merusak_jurnal(): void
    {
        $lama = Client::create(['name' => 'Toko Lama', 'type' => Client::TYPE_EXTERNAL]);
        foreach ([['7102', 'Beban Administrasi Bank', 'expense', 'non_operating'], ['4901', 'Pendapatan Lain-lain', 'revenue', 'other'],
            ['2103', 'Hutang Pajak', 'liability', 'tax_out'], ['1302', 'Persediaan Kemasan & Perlengkapan', 'asset', 'inventory'],
            ['5102', 'Pembelian Barang Dagang', 'expense', 'cogs'], ['1102', 'Bank', 'asset', 'bank']] as [$code, $name, $type, $subtype]) {
            $lama->accounts()->create(compact('code', 'name', 'type', 'subtype') + ['normal_balance' => ChartOfAccounts::normalBalanceFor($code, $type, $subtype), 'is_active' => true]);
        }
        $id = fn ($code) => $lama->accounts()->where('code', $code)->value('id');
        $adminBank = $id('7102');
        $jurnal = app(JournalPoster::class)->record(['client_id' => $lama->id, 'date' => '2026-10-02'], [
            ['account_id' => $adminBank, 'debit' => 6_500], ['account_id' => $id('2103'), 'debit' => 1_000], ['account_id' => $id('1102'), 'credit' => 7_500],
        ]);
        $pabrik = $this->klien('Pabrik', 'manufaktur');

        $log = app(ChartOfAccounts::class)->revisiDagang($lama);

        $this->assertSame('9102', $lama->accounts()->find($adminBank)->code); // akun yang sama, kode baru
        $this->assertSame($adminBank, $jurnal->lines()->where('debit', 6_500)->value('account_id'));
        $this->assertSame('other_income', $lama->accounts()->where('code', '8101')->value('subtype'));
        $this->assertFalse($lama->accounts()->where('code', '2103')->value('is_active')); // dipakai → nonaktif, tidak dihapus
        $this->assertSame('Perlengkapan', $lama->accounts()->where('code', '1302')->value('name'));
        $this->assertFalse($lama->accounts()->where('code', '5102')->value('is_active')); // perpetual, belum dipakai
        $this->assertTrue($lama->accounts()->where('code', '6117')->exists()); // akun baru ditambahkan
        $this->assertContains('7102 → 9102', $log);

        // Idempoten & klien Manufaktur tidak tersentuh.
        $jumlah = $lama->accounts()->count();
        app(ChartOfAccounts::class)->revisiDagang($lama);
        $this->assertSame($jumlah, $lama->accounts()->count());
        $this->assertTrue($pabrik->accounts()->where('code', '7102')->exists());
    }

    public function test_isi_template_berbeda_per_jenis_usaha(): void
    {
        $dagang = $this->klien('Toko', 'perdagangan')->accounts()->pluck('name', 'code');
        $pabrik = $this->klien('Pabrik', 'manufaktur')->accounts()->pluck('name', 'code');
        $jasa = $this->klien('Konsultan', 'jasa')->accounts();

        $this->assertSame('Persediaan Barang Dagang', $dagang['1301']);
        $this->assertSame('Persediaan Barang Dalam Proses', $pabrik['1303']);
        $this->assertArrayHasKey('5105', $pabrik->all()); // overhead pabrik
        $this->assertFalse($jasa->clone()->where('subtype', 'inventory')->exists());
        $this->assertSame('Pendapatan Jasa', $jasa->clone()->where('code', '4101')->value('name'));
        $this->assertSame('service', $jasa->clone()->where('code', '4101')->value('subtype')); // masuk omzet PPh Final
    }

    public function test_form_klien_memakai_template_pilihan(): void
    {
        $admin = User::create(['name' => 'A', 'email' => 'a@t.id', 'password' => 'x', 'role' => User::ROLE_ADMIN, 'is_active' => true]);
        $this->actingAs($admin)->post(route('clients.store'), ['name' => 'Bengkel Maju', 'type' => Client::TYPE_EXTERNAL, 'business_type' => 'jasa'])->assertRedirect();
        $c = Client::where('name', 'Bengkel Maju')->sole();

        $this->assertSame('jasa', $c->business_type);
        $this->assertSame(count(ChartOfAccounts::template('jasa')), $c->accounts()->count());
        $this->actingAs($admin)->get(route('accounts.index', $c))->assertOk()->assertSee('Template')->assertSee('Jasa');

        $this->actingAs($admin)->post(route('clients.store'), ['name' => 'X', 'type' => Client::TYPE_EXTERNAL, 'business_type' => 'tambang'])->assertSessionHasErrors('business_type');
    }

    public function test_coa_tidak_bercampur_antar_perusahaan(): void
    {
        $a = $this->klien('PT A', 'perdagangan');
        $b = $this->klien('PT B', 'perdagangan');
        $admin = User::create(['name' => 'A', 'email' => 'a@t.id', 'password' => 'x', 'role' => User::ROLE_ADMIN, 'is_active' => true]);

        // Kode sama, baris akun terpisah.
        $this->assertNotSame($a->accounts()->where('code', '1101')->value('id'), $b->accounts()->where('code', '1101')->value('id'));
        $this->assertSame([], array_intersect($a->accounts()->pluck('id')->all(), $b->accounts()->pluck('id')->all()));

        // Akun tambahan B tidak muncul di COA A.
        $this->actingAs($admin)->post(route('accounts.store', $b), ['code' => '6130', 'name' => 'Beban Riset Khusus B', 'type' => 'expense', 'is_active' => 1]);
        $this->actingAs($admin)->get(route('accounts.index', $a))->assertDontSee('Beban Riset Khusus B');
        $this->actingAs($admin)->get(route('accounts.index', $b))->assertSee('Beban Riset Khusus B');

        // Akun B tidak bisa diubah/dihapus lewat URL A.
        $akunB = $b->accounts()->where('code', '6130')->sole();
        $this->actingAs($admin)->put(route('accounts.update', [$a, $akunB]), ['code' => '6130', 'name' => 'Dibajak', 'type' => 'expense'])->assertNotFound();
        $this->actingAs($admin)->delete(route('accounts.destroy', [$a, $akunB]))->assertNotFound();
        $this->assertSame('Beban Riset Khusus B', $akunB->fresh()->name);

        // Jurnal A tidak bisa memakai akun B.
        $this->expectException(AccountingException::class);
        app(JournalPoster::class)->record(['client_id' => $a->id, 'date' => '2026-10-02'], [
            ['account_id' => $a->accounts()->where('code', '1101')->value('id'), 'debit' => 1000],
            ['account_id' => $akunB->id, 'credit' => 1000],
        ]);
    }
}
