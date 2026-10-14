<?php

namespace Tests\Feature;

use App\Models\Client;
use App\Models\Journal;
use App\Models\TaxEntry;
use App\Models\User;
use App\Services\Accounting\ChartOfAccounts;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

/** jurnal:demo mengisi data bertanda demo, dan --hapus membersihkannya tanpa menyentuh data asli. */
class DemoDataTest extends TestCase
{
    use RefreshDatabase;

    public function test_isi_lalu_hapus_data_demo(): void
    {
        $asli = Client::create(['name' => 'Klien Asli', 'type' => Client::TYPE_EXTERNAL]);
        app(ChartOfAccounts::class)->seedFor($asli);
        $admin = User::create(['name' => 'Admin', 'email' => 'admin@t.id', 'password' => 'x', 'role' => User::ROLE_ADMIN, 'is_active' => true]);

        $this->artisan('jurnal:demo', ['--sandi' => 'Kopi-Tubruk-Senja-27'])->assertSuccessful();
        $this->artisan('jurnal:demo')->assertFailed(); // tidak dobel

        $this->assertSame(12, Client::where('slug', 'like', 'demo-%')->count());
        $this->assertTrue(Journal::where('status', Journal::STATUS_POSTED)->count() > 200);
        $this->assertTrue(TaxEntry::count() > 0);
        foreach (User::where('email', 'like', '%@demo.jurnal.id')->get() as $u) {
            $this->assertLessThanOrEqual($u->clientLimit(), $u->clients()->count(), $u->email);
        }

        // Halaman utama tiap peran terbuka dengan data demo.
        $rina = User::where('email', 'rina@demo.jurnal.id')->sole();
        $klien = $rina->clients()->first();
        foreach (['clients.show', 'journals.index', 'reports.balance-sheet', 'reports.income-statement', 'reports.cash-flow', 'tax.index', 'documents.index'] as $route) {
            $this->actingAs($rina)->get(route($route, $klien))->assertOk();
        }
        $this->actingAs($admin)->get(route('dashboard'))->assertOk();
        $doc = $klien->documents()->where('status', 'extracted')->firstOrFail();
        $this->actingAs($rina)->get(route('documents.show', [$klien, $doc]))->assertOk();

        $this->artisan('jurnal:demo', ['--hapus' => true])->assertSuccessful();
        $this->assertSame(0, Client::where('slug', 'like', 'demo-%')->count());
        $this->assertSame(0, User::where('email', 'like', '%@demo.jurnal.id')->count());
        $this->assertTrue($asli->fresh()->accounts()->exists());
        $this->assertNotNull($admin->fresh());
    }
}
