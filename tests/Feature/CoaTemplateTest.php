<?php

namespace Tests\Feature;

use App\Models\Client;
use App\Models\User;
use App\Exceptions\AccountingException;
use App\Services\Accounting\ChartOfAccounts;
use App\Services\Accounting\JournalPoster;
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
        $wajib = ['1101', '1102', '1201', '1402', '1403', '2101', '2103', '2106', '2107', '2108', '2109', '2110', '4101', '6190', '7103'];
        $subtypePajak = collect(config('pajak.codes'))->flatMap(fn ($c) => [$c['subtype_in'], $c['subtype_out']])->filter()->unique();

        foreach (array_keys(ChartOfAccounts::BUSINESS_TYPES) as $jenis) {
            $c = $this->klien("PT {$jenis}", $jenis);
            $akun = $c->accounts()->get();
            $kode = $akun->pluck('code');

            $this->assertCount(count(ChartOfAccounts::template($jenis)), $akun, $jenis);
            $this->assertSame($kode->count(), $kode->unique()->count(), "{$jenis}: kode dobel");
            $this->assertSame([], array_values(array_diff($wajib, $kode->all())), "{$jenis}: akun mesin hilang");
            foreach ($subtypePajak as $st) {
                $this->assertTrue($akun->contains('subtype', $st), "{$jenis}: akun pajak {$st} hilang");
            }
            // Akun kontra bersaldo normal berlawanan.
            foreach ($akun->whereIn('subtype', ['contra_asset', 'contra_revenue', 'contra_cogs', 'drawing']) as $a) {
                $this->assertSame(in_array($a->type, ['asset', 'expense', 'equity']) && $a->subtype !== 'drawing' ? 'credit' : 'debit', $a->normal_balance, "{$jenis} {$a->code}");
            }
        }
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
        $this->actingAs($admin)->post(route('accounts.store', $b), ['code' => '6117', 'name' => 'Beban Riset Khusus B', 'type' => 'expense', 'is_active' => 1]);
        $this->actingAs($admin)->get(route('accounts.index', $a))->assertDontSee('Beban Riset Khusus B');
        $this->actingAs($admin)->get(route('accounts.index', $b))->assertSee('Beban Riset Khusus B');

        // Akun B tidak bisa diubah/dihapus lewat URL A.
        $akunB = $b->accounts()->where('code', '6117')->sole();
        $this->actingAs($admin)->put(route('accounts.update', [$a, $akunB]), ['code' => '6117', 'name' => 'Dibajak', 'type' => 'expense'])->assertNotFound();
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
