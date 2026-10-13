@extends('layouts.app')
@section('title', 'Pengguna')

@section('content')
    <div class="mb-6 flex flex-wrap items-end justify-between gap-4">
        <div>
            <p class="label mb-0.5">Admin</p>
            <h1 class="text-2xl font-semibold tracking-tight text-ink-900">Pengguna & peran</h1>
            <p class="mt-1 text-sm text-ink-500">Akuntan & anggota hanya melihat klien yang ditugaskan. Supervisor & admin melihat semua klien.</p>
        </div>
        <a href="{{ route('users.create') }}" class="btn-primary"><x-icon name="plus"/> Pengguna baru</a>
    </div>

    <div class="card mb-6 grid gap-px overflow-hidden bg-ink-200 sm:grid-cols-2 lg:grid-cols-4">
        @foreach (\App\Models\User::ROLES as $role => $label)
            <div class="bg-white p-4">
                <p class="font-medium text-ink-900">{{ $label }} <span class="font-mono text-xs text-ink-400">{{ $users->where('role', $role)->where('is_active', true)->count() }}</span></p>
                <p class="mt-1 text-xs text-ink-500">{{ \App\Models\User::ROLE_HINTS[$role] }}</p>
            </div>
        @endforeach
    </div>

    <div class="card overflow-x-auto">
        <table class="w-full">
            <thead><tr><th class="th">Nama</th><th class="th">Peran</th><th class="th">Klien</th><th class="th">Status</th><th class="th"></th></tr></thead>
            <tbody>
                @foreach ($users as $u)
                    <tr class="{{ $u->is_active ? '' : 'text-ink-400' }}">
                        <td class="td">
                            <span class="font-medium {{ $u->is_active ? 'text-ink-900' : '' }}">{{ $u->name }}</span>
                            @if ($u->is(auth()->user()))<span class="badge-mute ml-1">kamu</span>@endif
                            <span class="block text-xs text-ink-500">{{ $u->email }}</span>
                        </td>
                        <td class="td"><span class="{{ $u->role === 'admin' ? 'badge-info' : 'badge-mute' }}">{{ $u->roleLabel() }}</span></td>
                        <td class="td text-sm">
                            @if ($u->may('clients.all'))
                                <span class="text-ink-500">Semua klien</span>
                            @elseif ($u->clients->isEmpty())
                                <span class="badge-warn">belum ditugaskan</span>
                            @else
                                {{ $u->clients->pluck('name')->join(', ') }}
                            @endif
                        </td>
                        <td class="td"><span class="{{ $u->is_active ? 'badge-ok' : 'badge-bad' }}">{{ $u->is_active ? 'Aktif' : 'Nonaktif' }}</span></td>
                        <td class="td whitespace-nowrap text-right">
                            <a href="{{ route('users.edit', $u) }}" class="btn-ghost btn-sm">Ubah</a>
                            @unless ($u->is(auth()->user()))
                                <form method="POST" action="{{ route('users.toggle', $u) }}" class="inline"
                                      @if ($u->is_active) data-confirm="Nonaktifkan {{ $u->name }}? Sesinya langsung berakhir dan tidak bisa login." @endif>
                                    @csrf
                                    <button class="btn-ghost btn-sm">{{ $u->is_active ? 'Nonaktifkan' : 'Aktifkan' }}</button>
                                </form>
                            @endunless
                        </td>
                    </tr>
                @endforeach
            </tbody>
        </table>
    </div>
@endsection
