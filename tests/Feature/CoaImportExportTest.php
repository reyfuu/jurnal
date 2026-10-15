<?php

namespace Tests\Feature;

use App\Models\Client;
use App\Models\User;
use App\Services\Accounting\ChartOfAccounts;
use App\Services\Accounting\JournalPoster;
use App\Support\Xlsx;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Http\UploadedFile;
use Tests\TestCase;

/** COA: ekspor/impor Excel + CRUD modal memakai aturan yang sama. */
class CoaImportExportTest extends TestCase
{
    use RefreshDatabase;

    private Client $a;

    private Client $b;

    private User $admin;

    protected function setUp(): void
    {
        parent::setUp();
        $this->a = Client::create(['name' => 'PT A', 'type' => Client::TYPE_EXTERNAL, 'business_type' => 'jasa']);
        $this->b = Client::create(['name' => 'PT B', 'type' => Client::TYPE_EXTERNAL]);
        foreach ([$this->a, $this->b] as $c) {
            app(ChartOfAccounts::class)->seedFor($c);
        }
        $this->admin = User::create(['name' => 'A', 'email' => 'a@t.id', 'password' => 'x', 'role' => User::ROLE_ADMIN, 'is_active' => true]);
    }

    private function xlsx(array $rows, array $header = ['Kode', 'Nama akun', 'Tipe', 'Subtype', 'Saldo normal', 'Aktif']): UploadedFile
    {
        $path = tempnam(sys_get_temp_dir(), 'coa').'.xlsx';
        Xlsx::write($path, 'COA', $header, $rows);

        return new UploadedFile($path, 'coa.xlsx', null, null, true);
    }

    /** File dari Excel asli: teks di sharedStrings (termasuk teks kaya), sel kosong dilompati, sheet pertama ≠ sheet1.xml. */
    public function test_baca_xlsx_gaya_excel_asli(): void
    {
        $path = tempnam(sys_get_temp_dir(), 'xl').'.xlsx';
        $zip = new \ZipArchive;
        $zip->open($path, \ZipArchive::CREATE);
        $ns = 'http://schemas.openxmlformats.org';
        $zip->addFromString('xl/workbook.xml', "<workbook xmlns=\"{$ns}/spreadsheetml/2006/main\" xmlns:r=\"{$ns}/officeDocument/2006/relationships\"><sheets><sheet name=\"COA\" sheetId=\"1\" r:id=\"rId7\"/></sheets></workbook>");
        $zip->addFromString('xl/_rels/workbook.xml.rels', "<Relationships xmlns=\"{$ns}/package/2006/relationships\"><Relationship Id=\"rId7\" Type=\"x\" Target=\"worksheets/sheet3.xml\"/></Relationships>");
        $zip->addFromString('xl/sharedStrings.xml', "<sst xmlns=\"{$ns}/spreadsheetml/2006/main\"><si><t>Kode</t></si><si><t>Nama akun</t></si><si><r><t>Beban </t></r><r><rPr><b/></rPr><t>Ekspedisi</t></r></si><si><t>Beban</t></si></sst>");
        $zip->addFromString('xl/worksheets/sheet3.xml', "<worksheet xmlns=\"{$ns}/spreadsheetml/2006/main\"><sheetData>"
            .'<row r="1"><c r="A1" t="s"><v>0</v></c><c r="B1" t="s"><v>1</v></c><c r="C1" t="inlineStr"><is><t>Tipe</t></is></c></row>'
            .'<row r="2"><c r="A2"><v>6125</v></c><c r="C2" t="s"><v>3</v></c></row>'
            .'<row r="3"><c r="A3"><v>6126</v></c><c r="B3" t="s"><v>2</v></c><c r="C3" t="s"><v>3</v></c></row>'
            .'</sheetData></worksheet>');
        $zip->close();

        $this->assertSame([['Kode', 'Nama akun', 'Tipe'], ['6125', '', 'Beban'], ['6126', 'Beban Ekspedisi', 'Beban']], Xlsx::read($path, 'xlsx'));
    }

    public function test_ekspor_xlsx_bisa_dibaca_ulang_sama_persis(): void
    {
        $res = $this->actingAs($this->admin)->get(route('accounts.export', $this->a))->assertOk();
        $this->assertStringContainsString('spreadsheetml', $res->headers->get('Content-Type'));

        $rows = Xlsx::read($res->baseResponse->getFile()->getPathname(), 'xlsx');
        $this->assertSame(['Kode', 'Nama akun', 'Tipe', 'Subtype', 'Saldo normal', 'Aktif'], $rows[0]);
        $this->assertCount($this->a->accounts()->count() + 1, $rows);
        $this->assertContains(['4101', 'Pendapatan Jasa', 'Pendapatan', 'service', 'credit', 'ya'], $rows);
    }

    public function test_impor_menambah_dan_memperbarui_hanya_di_klien_ini(): void
    {
        $this->actingAs($this->admin)->post(route('accounts.import', $this->a), ['file' => $this->xlsx([
            ['6130', 'Beban Riset & Pengembangan', 'Beban', 'opex', '', 'ya'],
            ['4101', 'Pendapatan Jasa Konsultasi', 'pendapatan', 'service', 'kredit', 'ya'],
            ['1103', 'E-Wallet', 'Aset', 'ewallet', '', 'tidak'],
            ['', '', '', '', '', ''], // baris kosong dilewati
        ])])->assertSessionHasNoErrors()->assertSessionHas('status', 'Impor COA selesai: 1 akun baru, 2 diperbarui.');

        $akun = $this->a->accounts()->pluck('name', 'code');
        $this->assertSame('Beban Riset & Pengembangan', $akun['6130']);
        $this->assertSame('Pendapatan Jasa Konsultasi', $akun['4101']);
        $this->assertFalse($this->a->accounts()->where('code', '1103')->value('is_active'));
        $this->assertSame('debit', $this->a->accounts()->where('code', '6130')->value('normal_balance'));
        // Klien lain tidak tersentuh.
        $this->assertFalse($this->b->accounts()->where('code', '6130')->exists());
        $this->assertSame('Penjualan', $this->b->accounts()->where('code', '4101')->value('name'));
    }

    public function test_satu_baris_salah_seluruh_impor_batal(): void
    {
        $sebelum = $this->a->accounts()->count();
        $this->actingAs($this->admin)->post(route('accounts.import', $this->a), ['file' => $this->xlsx([
            ['6117', 'Beban Riset', 'Beban', '', '', ''],
            ['6118', '', 'Beban', '', '', ''],          // nama kosong
            ['6119', 'Beban X', 'Galaksi', '', '', ''], // tipe tidak dikenal
            ['6117', 'Dobel', 'Beban', '', '', ''],     // kode dobel di file
        ])])->assertSessionHasErrors('file');

        $pesan = implode(' ', session('errors')->get('file'));
        $this->assertStringContainsString('Baris 3', $pesan);
        $this->assertStringContainsString('Baris 4', $pesan);
        $this->assertStringContainsString('Baris 5', $pesan);
        $this->assertSame($sebelum, $this->a->accounts()->count());
    }

    public function test_impor_csv_dan_judul_kolom_wajib(): void
    {
        $csv = UploadedFile::fake()->createWithContent('coa.csv', "\xEF\xBB\xBFKode;Nama akun;Tipe\n6120;Beban Pameran;Beban\n");
        $this->actingAs($this->admin)->post(route('accounts.import', $this->a), ['file' => $csv])->assertSessionHasNoErrors();
        $this->assertSame('Beban Pameran', $this->a->accounts()->where('code', '6120')->value('name'));

        $this->actingAs($this->admin)->post(route('accounts.import', $this->a), ['file' => $this->xlsx([['6121', 'X']], ['Foo', 'Bar'])])
            ->assertSessionHasErrors('file');
        $this->actingAs($this->admin)->post(route('accounts.import', $this->a), ['file' => UploadedFile::fake()->createWithContent('rusak.xlsx', 'bukan zip')])
            ->assertSessionHasErrors('file');
    }

    public function test_tipe_akun_yang_sudah_dipakai_jurnal_terkunci(): void
    {
        $kas = $this->a->accounts()->where('code', '1101')->sole();
        app(JournalPoster::class)->record(['client_id' => $this->a->id, 'date' => '2026-10-02'], [
            ['account_id' => $kas->id, 'debit' => 1000],
            ['account_id' => $this->a->accounts()->where('code', '4101')->value('id'), 'credit' => 1000],
        ]);

        $this->actingAs($this->admin)->put(route('accounts.update', [$this->a, $kas]), ['code' => '1101', 'name' => 'Kas', 'type' => 'expense', '_dialog' => $kas->id])
            ->assertSessionHasErrors('type');
        $this->actingAs($this->admin)->post(route('accounts.import', $this->a), ['file' => $this->xlsx([['1101', 'Kas', 'Beban']])])
            ->assertSessionHasErrors('file');
        $this->assertSame('asset', $kas->fresh()->type);

        // Nama boleh diganti.
        $this->actingAs($this->admin)->put(route('accounts.update', [$this->a, $kas]), ['code' => '1101', 'name' => 'Kas Kecil', 'type' => 'asset', 'is_active' => 1])
            ->assertSessionHasNoErrors();
        $this->assertSame('Kas Kecil', $kas->fresh()->name);
    }

    public function test_halaman_coa_pakai_modal_dan_hak_akses_impor(): void
    {
        $this->actingAs($this->admin)->get(route('accounts.index', $this->a))->assertOk()
            ->assertSee('id="account-dialog"', false)->assertSee('data-account-edit', false)
            ->assertDontSee('form="acc-', false); // tidak ada lagi input edit langsung di baris

        // Validasi gagal → modal dibuka lagi.
        $this->actingAs($this->admin)->from(route('accounts.index', $this->a))
            ->post(route('accounts.store', $this->a), ['code' => '', 'name' => 'X', 'type' => 'expense', '_dialog' => 'new'])
            ->assertRedirect(route('accounts.index', $this->a));
        $this->actingAs($this->admin)->get(route('accounts.index', $this->a))->assertSee('data-autopen', false);

        $akuntan = User::create(['name' => 'R', 'email' => 'r@t.id', 'password' => 'x', 'role' => User::ROLE_AKUNTAN, 'is_active' => true]);
        $akuntan->clients()->attach($this->a);
        $this->actingAs($akuntan)->get(route('accounts.export', $this->a))->assertOk();
        $this->actingAs($akuntan)->post(route('accounts.import', $this->a), ['file' => $this->xlsx([['6117', 'X', 'Beban']])])->assertForbidden();
        $this->actingAs($akuntan)->get(route('accounts.export', $this->b))->assertNotFound();
    }
}
