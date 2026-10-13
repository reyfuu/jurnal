@extends('layouts.app')
@section('title', $user->exists ? 'Ubah pengguna' : 'Pengguna baru')

@section('content')
    <div class="mb-6">
        <p class="label mb-0.5">Admin · Pengguna</p>
        <h1 class="text-2xl font-semibold tracking-tight text-ink-900">{{ $user->exists ? 'Ubah '.$user->name : 'Pengguna baru' }}</h1>
    </div>

    <form method="POST" action="{{ $user->exists ? route('users.update', $user) : route('users.store') }}" class="card max-w-3xl space-y-6 p-6">
        @csrf
        @if ($user->exists) @method('PUT') @endif

        <div class="grid gap-4 sm:grid-cols-2">
            <div>
                <label class="label" for="name">Nama</label>
                <input id="name" name="name" class="field" required value="{{ old('name', $user->name) }}">
            </div>
            <div>
                <label class="label" for="email">Email (untuk login)</label>
                <input id="email" name="email" type="email" class="field" required autocomplete="off" value="{{ old('email', $user->email) }}">
            </div>
            <div class="sm:col-span-2">
                <label class="label" for="password">{{ $user->exists ? 'Password baru (kosongkan kalau tidak diganti)' : 'Password' }}</label>
                <div class="relative max-w-sm">
                    <input id="password" name="password" type="password" class="field pr-10" autocomplete="new-password" @required(! $user->exists)>
                    <button type="button" data-reveal="password" aria-pressed="false" aria-label="Tampilkan password"
                            class="absolute inset-y-0 right-0 grid w-10 place-items-center text-ink-400 hover:text-ink-800 cursor-pointer">
                        <x-icon name="eye"/><x-icon name="eye-off" class="size-4 hidden"/>
                    </button>
                </div>
                <p class="mt-1 text-xs text-ink-400">Minimal 12 karakter, tidak boleh mengandung kata umum seperti "password" atau "admin".</p>
            </div>
        </div>

        <fieldset>
            <legend class="label">Peran</legend>
            <div class="grid gap-2 sm:grid-cols-2">
                @foreach (\App\Models\User::ROLES as $role => $label)
                    <label class="flex cursor-pointer items-start gap-2 rounded-md border border-ink-200 p-3 text-sm hover:border-brand-500 has-checked:border-brand-600 has-checked:bg-brand-50">
                        <input type="radio" name="role" value="{{ $role }}" class="mt-0.5" data-role-radio
                               @checked(old('role', $user->role) === $role) @disabled($user->is(auth()->user()) && $role !== $user->role)>
                        <span><span class="font-medium text-ink-900">{{ $label }}</span>
                            <span class="block text-xs text-ink-500">{{ \App\Models\User::ROLE_HINTS[$role] }}</span></span>
                    </label>
                @endforeach
            </div>
        </fieldset>

        <fieldset data-client-picker>
            <legend class="label">Klien yang ditugaskan</legend>
            <p class="mb-2 text-xs text-ink-500">Berlaku untuk akuntan & anggota. Admin & supervisor otomatis melihat semua klien.</p>
            @php($picked = collect(old('clients', $user->clients?->pluck('id')->all() ?? []))->map(fn ($id) => (int) $id))
            <div class="grid gap-1.5 sm:grid-cols-2 lg:grid-cols-3">
                @forelse ($clients as $c)
                    <label class="flex items-center gap-2 text-sm {{ $c->is_active ? '' : 'text-ink-400' }}">
                        <input type="checkbox" name="clients[]" value="{{ $c->id }}" @checked($picked->contains($c->id))>
                        {{ $c->name }} @unless ($c->is_active)<span class="text-xs">(nonaktif)</span>@endunless
                    </label>
                @empty
                    <p class="text-sm text-ink-500">Belum ada klien.</p>
                @endforelse
            </div>
        </fieldset>

        <div class="flex gap-2 border-t border-ink-100 pt-4">
            <button class="btn-primary">{{ $user->exists ? 'Simpan' : 'Buat pengguna' }}</button>
            <a href="{{ route('users.index') }}" class="btn-ghost">Batal</a>
        </div>
    </form>
@endsection
