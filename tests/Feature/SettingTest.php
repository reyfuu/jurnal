<?php

namespace Tests\Feature;

use App\Models\AuditLog;
use App\Models\Setting;
use App\Models\User;
use App\Services\Ai\AiProviderFactory;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Http;
use Tests\TestCase;

/** Key Gemini dari halaman Pengaturan: terenkripsi, khusus admin, menang atas .env. */
class SettingTest extends TestCase
{
    use RefreshDatabase;

    private function user(string $role): User
    {
        return User::create(['name' => $role, 'email' => $role.uniqid().'@t.id', 'password' => 'secret123', 'role' => $role, 'is_active' => true]);
    }

    public function test_admin_simpan_key_gemini_terenkripsi_dan_dipakai_factory(): void
    {
        config(["ai.openai.key" => ""]); // shell bisa meng-export OPENAI_API_KEY
        $this->assertFalse(AiProviderFactory::configured());

        $this->actingAs($this->user(User::ROLE_ADMIN))
            ->put(route('settings.update'), ['gemini_key' => 'AIzaRahasia1234', 'gemini_model' => 'gemini-2.5-flash'])
            ->assertRedirect(route('settings.edit'));

        $this->assertSame('AIzaRahasia1234', Setting::get('gemini_key'));
        $this->assertStringNotContainsString('AIzaRahasia1234', (string) DB::table('settings')->where('key', 'gemini_key')->value('value'));
        $this->assertTrue(AiProviderFactory::configured());
        $this->assertSame('gemini-2.5-flash', AiProviderFactory::make()->model());

        // Audit hanya menyimpan 4 karakter terakhir.
        $log = AuditLog::where('action', 'settings.ai')->firstOrFail();
        $this->assertSame('••••1234', $log->after['key']);

        // Halaman tidak menampilkan key utuh.
        $this->get(route('settings.edit'))->assertOk()->assertSee('••••1234')->assertDontSee('AIzaRahasia1234');
    }

    public function test_key_kosong_mempertahankan_key_lama_dan_hapus_mengosongkan(): void
    {
        Setting::put('gemini_key', 'AIzaLama9999');
        $admin = $this->user(User::ROLE_ADMIN);

        $this->actingAs($admin)->put(route('settings.update'), ['gemini_key' => '', 'gemini_model' => 'gemini-2.5-pro']);
        $this->assertSame('AIzaLama9999', Setting::get('gemini_key'));
        $this->assertSame('gemini-2.5-pro', AiProviderFactory::geminiModel());

        $this->actingAs($admin)->put(route('settings.update'), ['gemini_model' => 'gemini-2.5-pro', 'hapus_key' => '1']);
        $this->assertNull(Setting::get('gemini_key'));
    }

    public function test_tes_koneksi_memanggil_endpoint_gemini(): void
    {
        Setting::put('gemini_key', 'AIzaTes0000');
        Http::fake(['generativelanguage.googleapis.com/*' => Http::response(['choices' => [['message' => ['content' => 'OK']]]])]);

        $this->actingAs($this->user(User::ROLE_ADMIN))->post(route('settings.test'))
            ->assertSessionHas('status');

        Http::assertSent(fn ($r) => str_ends_with($r->url(), '/v1beta/openai/chat/completions')
            && $r->hasHeader('Authorization', 'Bearer AIzaTes0000'));
    }

    public function test_tes_koneksi_key_ditolak_tampil_sebagai_error(): void
    {
        Setting::put('gemini_key', 'AIzaSalah');
        Http::fake(['*' => Http::response(['error' => ['message' => 'API key not valid']], 400)]);

        $this->actingAs($this->user(User::ROLE_ADMIN))->post(route('settings.test'))
            ->assertSessionHasErrors('test');
    }

    public function test_selain_admin_tidak_bisa_membuka_pengaturan(): void
    {
        foreach ([User::ROLE_SUPERVISOR, User::ROLE_AKUNTAN, User::ROLE_ANGGOTA] as $role) {
            $user = $this->user($role);
            $this->actingAs($user)->get(route('settings.edit'))->assertForbidden();
            $this->actingAs($user)->put(route('settings.update'), ['gemini_key' => 'x', 'gemini_model' => 'y'])->assertForbidden();
        }
        $this->assertNull(Setting::get('gemini_key'));
    }
}
