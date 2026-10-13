<?php

namespace Tests\Feature;

use App\Models\AuditLog;
use App\Models\Client;
use App\Models\Document;
use App\Models\Journal;
use App\Models\User;
use App\Services\Accounting\ChartOfAccounts;
use App\Services\Accounting\JournalPoster;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

/**
 * Matriks peran: admin · supervisor · akuntan · anggota, plus isolasi per
 * klien (akuntan & anggota hanya klien yang ditugaskan).
 */
class RoleAccessTest extends TestCase
{
    use RefreshDatabase;

    private Client $a;

    private Client $b;

    protected function setUp(): void
    {
        parent::setUp();
        foreach (['a' => 'Klien A', 'b' => 'Klien B'] as $key => $name) {
            $this->$key = Client::create(['name' => $name, 'type' => Client::TYPE_EXTERNAL]);
            app(ChartOfAccounts::class)->seedFor($this->$key);
        }
    }

    private function user(string $role, array $clients = []): User
    {
        $user = User::create(['name' => ucfirst($role), 'email' => $role.uniqid().'@t.id', 'password' => 'secret123', 'role' => $role, 'is_active' => true]);
        $user->clients()->attach(array_map(fn (Client $c) => $c->id, $clients));

        return $user;
    }

    private function jurnal(Client $c): Journal
    {
        $a = fn ($code) => $c->accounts()->where('code', $code)->value('id');

        return app(JournalPoster::class)->record(['client_id' => $c->id, 'date' => '2026-10-02'], [['account_id' => $a('1101'), 'debit' => 1000], ['account_id' => $a('4101'), 'credit' => 1000]]);
    }

    public function test_akuntan_hanya_melihat_klien_yang_ditugaskan(): void
    {
        $akuntan = $this->user(User::ROLE_AKUNTAN, [$this->a]);

        $this->actingAs($akuntan)->get(route('clients.show', $this->a))->assertOk();
        $this->actingAs($akuntan)->get(route('reports.balance-sheet', $this->a))->assertOk();
        foreach (['clients.show', 'journals.index', 'documents.index', 'tax.index', 'reports.trial-balance'] as $route) {
            $this->actingAs($akuntan)->get(route($route, $this->b))->assertNotFound();
        }
        $this->actingAs($akuntan)->get(route('reports.export', [$this->b, 'neraca']))->assertNotFound();

        $this->actingAs($akuntan)->get(route('clients.index'))->assertSee('Klien A')->assertDontSee('Klien B');
        $this->actingAs($akuntan)->get(route('dashboard'))->assertSee('Klien A')->assertDontSee('Klien B');
    }

    public function test_anggota_hanya_upload_dan_melihat_dokumen(): void
    {
        $anggota = $this->user(User::ROLE_ANGGOTA, [$this->a]);
        $doc = $this->a->documents()->create(['kind' => Document::KIND_TEXT, 'status' => Document::STATUS_EXTRACTED, 'raw_text' => 'x',
            'extraction' => app(\App\Services\Intake\DocumentExtractor::class)->normalize(
                ['flow' => 'expense', 'document_date' => '2026-10-02', 'payment_method' => 'cash', 'items' => [['description' => 'X', 'amount' => 1000, 'account_code' => '6190']]],
                $this->a,
            )]);

        $this->actingAs($anggota)->get(route('dashboard'))->assertRedirect(route('documents.index', $this->a));
        $this->actingAs($anggota)->get(route('documents.create', $this->a))->assertOk();
        $this->actingAs($anggota)->get(route('documents.show', [$this->a, $doc]))->assertOk()
            ->assertSee('menunggu review akuntan')->assertDontSee('Posting jurnal yang dicentang');

        $this->actingAs($anggota)->post(route('documents.post', [$this->a, $doc]), ['drafts' => [['post' => 1]]])->assertForbidden();
        foreach (['journals.index', 'reports.income-statement', 'tax.index', 'clients.show', 'journals.create'] as $route) {
            $this->actingAs($anggota)->get(route($route, $this->a))->assertForbidden();
        }
        $this->actingAs($anggota)->get(route('documents.index', $this->b))->assertNotFound();
    }

    public function test_supervisor_semua_klien_dan_struktur_tapi_bukan_pengguna_tarif_hapus(): void
    {
        $sup = $this->user(User::ROLE_SUPERVISOR);
        $journal = $this->jurnal($this->b);

        $this->actingAs($sup)->get(route('clients.show', $this->b))->assertOk();
        $this->actingAs($sup)->get(route('clients.create'))->assertOk();
        $this->actingAs($sup)->put(route('tax.profile.update', $this->b), ['is_pkp' => 0, 'pph_regime' => 'normal'])->assertRedirect();
        $this->actingAs($sup)->post(route('journals.void', [$this->b, $journal]))->assertRedirect();

        $this->actingAs($sup)->get(route('users.index'))->assertForbidden();
        $this->actingAs($sup)->get(route('tax.rates'))->assertForbidden();
        $this->actingAs($sup)->delete(route('journals.destroy', [$this->b, $journal]))->assertForbidden();
        $this->assertFalse($this->b->fresh()->is_pkp);
    }

    public function test_admin_kelola_pengguna(): void
    {
        $admin = $this->user(User::ROLE_ADMIN);

        $this->actingAs($admin)->post(route('users.store'), [
            'name' => 'Rina', 'email' => 'rina@t.id', 'role' => User::ROLE_AKUNTAN, 'password' => 'password1234', 'clients' => [$this->a->id],
        ])->assertSessionHasErrors('password');

        $this->actingAs($admin)->post(route('users.store'), [
            'name' => 'Rina', 'email' => 'rina@t.id', 'role' => User::ROLE_AKUNTAN, 'password' => 'Kopi-Tubruk-Senja-27', 'clients' => [$this->a->id],
        ])->assertRedirect(route('users.index'));
        $rina = User::where('email', 'rina@t.id')->sole();
        $this->assertSame([$this->a->id], $rina->clients->pluck('id')->all());

        // Pindah tugas ke klien B tanpa ganti password.
        $hash = $rina->password;
        $this->actingAs($admin)->put(route('users.update', $rina), ['name' => 'Rina', 'email' => 'rina@t.id', 'role' => User::ROLE_AKUNTAN, 'clients' => [$this->b->id]])->assertRedirect();
        $this->assertSame([$this->b->id], $rina->fresh()->clients->pluck('id')->all());
        $this->assertSame($hash, $rina->fresh()->password);

        // Nonaktif → langsung keluar.
        $this->actingAs($admin)->post(route('users.toggle', $rina))->assertRedirect();
        $this->actingAs($rina->fresh())->get(route('clients.show', $this->b))->assertRedirect(route('login'));

        // Tidak bisa mengunci diri sendiri.
        $this->actingAs($admin)->post(route('users.toggle', $admin))->assertStatus(422);
        $this->actingAs($admin)->put(route('users.update', $admin), ['name' => 'A', 'email' => $admin->email, 'role' => User::ROLE_ANGGOTA])->assertSessionHasErrors('role');

        $this->assertSame(['user.create', 'user.update', 'user.deactivate'], AuditLog::orderBy('id')->pluck('action')->all());
    }

    public function test_hanya_admin_yang_boleh_kelola_pengguna(): void
    {
        foreach ([User::ROLE_SUPERVISOR, User::ROLE_AKUNTAN, User::ROLE_ANGGOTA] as $role) {
            $this->actingAs($this->user($role, [$this->a]))->post(route('users.store'), [])->assertForbidden();
        }
    }
}
