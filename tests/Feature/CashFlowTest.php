<?php

namespace Tests\Feature;

use App\Models\Client;
use App\Models\Journal;
use App\Models\User;
use App\Services\Accounting\ChartOfAccounts;
use App\Services\Accounting\JournalPoster;
use App\Services\Reporting\ReportService;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

/** Arus kas metode langsung: kas awal + arus bersih = kas akhir, dikelompokkan per aktivitas. */
class CashFlowTest extends TestCase
{
    use RefreshDatabase;

    public function test_arus_kas_dikelompokkan_dan_tutup_ke_saldo_kas(): void
    {
        $client = Client::create(['name' => 'Uji', 'type' => Client::TYPE_EXTERNAL]);
        app(ChartOfAccounts::class)->seedFor($client);
        $a = fn (string $code) => $client->accounts()->where('code', $code)->value('id');
        $post = fn (string $date, array $lines) => app(JournalPoster::class)->record(['client_id' => $client->id, 'date' => $date], $lines);

        $post('2026-09-01', [['account_id' => $a('1102'), 'debit' => 10_000_000], ['account_id' => $a('3101'), 'credit' => 10_000_000]]); // modal bulan lalu
        $post('2026-10-02', [['account_id' => $a('1102'), 'debit' => 5_000_000], ['account_id' => $a('4101'), 'credit' => 5_000_000]]);
        $post('2026-10-03', [['account_id' => $a('6104'), 'debit' => 2_000_000], ['account_id' => $a('1102'), 'credit' => 1_500_000], ['account_id' => $a('2101'), 'credit' => 500_000]]);
        $post('2026-10-04', [['account_id' => $a('1501'), 'debit' => 3_000_000], ['account_id' => $a('1102'), 'credit' => 3_000_000]]);
        $post('2026-10-05', [['account_id' => $a('1101'), 'debit' => 1_000_000], ['account_id' => $a('1102'), 'credit' => 1_000_000]]); // tarik tunai: netral
        $post('2026-10-06', [['account_id' => $a('2102'), 'debit' => 0.01], ['account_id' => $a('2102'), 'credit' => 0], ['account_id' => $a('1102'), 'credit' => 0.01]]);
        $void = $post('2026-10-07', [['account_id' => $a('1102'), 'debit' => 9_999_999], ['account_id' => $a('4101'), 'credit' => 9_999_999]]);
        app(JournalPoster::class)->void($void);

        $r = app(ReportService::class)->cashFlow($client->id, '2026-10');

        $this->assertSame(3_500_000.0, $r['totals']['operating'], 'Penjualan 5 jt − sewa dibayar 1,5 jt.');
        $this->assertSame(-3_000_000.0, $r['totals']['investing']);
        $this->assertSame(-0.01, $r['totals']['financing']);
        $this->assertSame(10_000_000.0, $r['opening']);
        $this->assertSame(10_499_999.99, $r['closing']);
        $this->assertTrue($r['balanced']);
    }

    public function test_halaman_arus_kas_bisa_dibuka(): void
    {
        $client = Client::create(['name' => 'Uji', 'type' => Client::TYPE_EXTERNAL]);
        $user = User::create(['name' => 'S', 'email' => 's@t.id', 'password' => 'secret123', 'role' => User::ROLE_STAFF, 'is_active' => true]);

        $this->actingAs($user)->get(route('reports.cash-flow', $client).'?period=2026-10')->assertOk()->assertSee('Arus Kas');
    }

    public function test_semua_laporan_bisa_diekspor_csv(): void
    {
        $client = Client::create(['name' => 'Uji', 'type' => Client::TYPE_EXTERNAL]);
        app(ChartOfAccounts::class)->seedFor($client);
        $a = fn (string $code) => $client->accounts()->where('code', $code)->value('id');
        app(JournalPoster::class)->record(['client_id' => $client->id, 'date' => '2026-10-02'], [['account_id' => $a('1102'), 'debit' => 5_000_000], ['account_id' => $a('4101'), 'credit' => 5_000_000]]);
        $user = User::create(['name' => 'S', 'email' => 's@t.id', 'password' => 'secret123', 'role' => User::ROLE_STAFF, 'is_active' => true]);

        foreach (['neraca-saldo' => 'Neraca Saldo', 'laba-rugi' => 'Laba Rugi', 'neraca' => 'Neraca', 'arus-kas' => 'Arus Kas'] as $jenis => $judul) {
            $csv = $this->actingAs($user)->get(route('reports.export', [$client, $jenis]).'?period=2026-10')->assertOk()->streamedContent();
            $this->assertStringStartsWith("\xEF\xBB\xBF{$judul};", $csv, $jenis);
            $this->assertStringContainsString('5000000', $csv, $jenis);
        }
        $this->actingAs($user)->get(route('reports.export', [$client, 'ngawur']))->assertNotFound();
    }
}
