@extends('layouts.app')
@section('title', 'Pengaturan AI')

@section('content')
    <div class="mb-6">
        <p class="label mb-0.5">Admin · Pengaturan</p>
        <h1 class="text-2xl font-semibold tracking-tight text-ink-900">Otak AI — Google Gemini</h1>
    </div>

    <form method="POST" action="{{ route('settings.update') }}" class="card max-w-2xl space-y-5 p-6">
        @csrf
        @method('PUT')

        <div>
            <label class="label" for="gemini_key">API key Gemini</label>
            <div class="relative">
                <input id="gemini_key" name="gemini_key" type="password" class="field pr-10 font-mono" autocomplete="off" spellcheck="false"
                       placeholder="{{ $keyHint ? 'Tersimpan ('.$keyHint.') — kosongkan kalau tidak diganti' : 'AIza…' }}">
                <button type="button" data-reveal="gemini_key" aria-pressed="false" aria-label="Tampilkan key"
                        class="absolute inset-y-0 right-0 grid w-10 place-items-center text-ink-400 hover:text-ink-800 cursor-pointer">
                    <x-icon name="eye"/><x-icon name="eye-off" class="size-4 hidden"/>
                </button>
            </div>
            <p class="mt-1 text-xs text-ink-500">
                Buat di <a href="https://aistudio.google.com/apikey" target="_blank" rel="noopener" class="underline">aistudio.google.com/apikey</a>.
                Disimpan terenkripsi; tidak pernah ditampilkan ulang. Kalau terisi, key ini dipakai menggantikan setelan AI di <code class="font-mono">.env</code>.
            </p>
            @if ($keyHint)
                <label class="mt-2 flex items-center gap-2 text-sm text-ink-700">
                    <input type="checkbox" name="hapus_key" value="1"> Hapus key tersimpan (kembali ke <code class="font-mono">.env</code>)
                </label>
            @endif
        </div>

        <div>
            <label class="label" for="gemini_model">Model</label>
            <input id="gemini_model" name="gemini_model" class="field max-w-sm font-mono" required value="{{ old('gemini_model', $model) }}">
            <p class="mt-1 text-xs text-ink-500">Default <code class="font-mono">{{ config('ai.gemini.model') }}</code> — alias Google yang selalu menunjuk ke model Flash terbaru. Ganti hanya kalau perlu model tertentu.</p>
        </div>

        <div class="flex items-center gap-2 border-t border-ink-100 pt-4">
            <button class="btn-primary">Simpan</button>
            <span class="ml-auto flex items-center gap-2 text-xs text-ink-500">
                <span class="size-2 rounded-full {{ $aiReady ? 'bg-emerald-500' : 'bg-rose-500' }}"></span>
                {{ $aiReady ? 'Key AI terisi' : 'AI belum terpasang' }}
            </span>
        </div>
    </form>

    @if ($aiReady)
        <form method="POST" action="{{ route('settings.test') }}" class="mt-4 max-w-2xl">
            @csrf
            <button class="btn-ghost">Tes koneksi AI</button>
            <span class="text-xs text-ink-500">Satu panggilan kecil untuk memastikan key & model diterima.</span>
        </form>
    @endif
@endsection
