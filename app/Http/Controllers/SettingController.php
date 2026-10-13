<?php

namespace App\Http\Controllers;

use App\Models\Setting;
use App\Services\Ai\AiException;
use App\Services\Ai\AiProviderFactory;
use App\Support\Audit;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;

/** Pengaturan AI (khusus admin): key & model Gemini, disimpan terenkripsi di DB. */
class SettingController extends Controller
{
    public function edit()
    {
        return view('settings.edit', [
            'keyHint' => self::mask(Setting::get('gemini_key')),
            'model' => AiProviderFactory::geminiModel(),
            'aiReady' => AiProviderFactory::configured(),
        ]);
    }

    public function update(Request $request): RedirectResponse
    {
        $data = $request->validate([
            'gemini_key' => ['nullable', 'string', 'max:200', 'regex:/^\S+$/'],
            'gemini_model' => ['required', 'string', 'max:100', 'regex:/^[\w.\-\/]+$/'],
            'hapus_key' => ['nullable', 'boolean'],
        ], ['gemini_key.regex' => 'Key tidak boleh mengandung spasi.']);

        $before = ['key' => self::mask(Setting::get('gemini_key')), 'model' => AiProviderFactory::geminiModel()];

        // Kolom key kosong = key lama dipertahankan (tidak pernah ditampilkan ulang).
        if ($request->boolean('hapus_key')) {
            Setting::put('gemini_key', null);
        } elseif (filled($data['gemini_key'] ?? null)) {
            Setting::put('gemini_key', trim($data['gemini_key']));
        }
        Setting::put('gemini_model', $data['gemini_model'] === config('ai.gemini.model') ? null : $data['gemini_model']);

        $after = ['key' => self::mask(Setting::get('gemini_key')), 'model' => AiProviderFactory::geminiModel()];
        if ($before !== $after) {
            Audit::log('settings.ai', 'setting', null, null, $before, $after);
        }

        return redirect()->route('settings.edit')->with('status', 'Pengaturan AI disimpan.');
    }

    /** Satu panggilan kecil ke AI supaya key salah ketahuan sekarang, bukan saat dokumen dibaca. */
    public function test(): RedirectResponse
    {
        try {
            $turn = AiProviderFactory::make()->chat([['role' => 'user', 'content' => 'Balas satu kata: OK']], ['max_tokens' => 1000]);
        } catch (AiException $e) {
            return redirect()->route('settings.edit')->withErrors(['test' => 'Tes gagal: '.$e->getMessage()]);
        }

        return redirect()->route('settings.edit')->with('status', "Tes berhasil — model {$turn->model} merespons.");
    }

    private static function mask(?string $key): ?string
    {
        return $key ? '••••'.substr($key, -4) : null;
    }
}
