@extends('layouts.app')
@section('title', 'Audit log')

@section('content')
    <div class="mb-6 flex flex-wrap items-end justify-between gap-4">
        <div>
            <p class="label mb-0.5">Admin</p>
            <h1 class="text-2xl font-semibold tracking-tight text-ink-900">Audit log</h1>
            <p class="mt-1 text-sm text-ink-500">Perubahan profil pajak, tarif, status setor/lapor, dan ekspor. 200 terakhir.</p>
        </div>
        <form method="GET" class="flex items-end gap-2">
            <div>
                <label class="label" for="klien">Klien</label>
                <select id="klien" name="klien" class="field w-48">
                    <option value="">Semua</option>
                    @foreach ($clients as $id => $name)<option value="{{ $id }}" @selected(request('klien') == $id)>{{ $name }}</option>@endforeach
                </select>
            </div>
            <button class="btn-ghost">Saring</button>
        </form>
    </div>

    <div class="card overflow-x-auto">
        <table class="w-full">
            <thead><tr><th class="th">Waktu</th><th class="th">User</th><th class="th">Klien</th><th class="th">Aksi</th><th class="th">Perubahan</th></tr></thead>
            <tbody>
                @forelse ($logs as $log)
                    <tr class="align-top">
                        <td class="td whitespace-nowrap text-xs">{{ $log->created_at->format('d/m/Y H:i') }}</td>
                        <td class="td">{{ $log->user?->name ?? '—' }}</td>
                        <td class="td">{{ $clients[$log->client_id] ?? '—' }}</td>
                        <td class="td font-mono text-xs">{{ $log->action }}</td>
                        <td class="td font-mono text-xs text-ink-600">
                            @foreach ((array) $log->after as $k => $v)
                                @php($old = $log->before[$k] ?? null)
                                <div>{{ $k }}: @if ($log->before !== null && $old !== $v)<span class="text-ink-400 line-through">{{ is_scalar($old) ? var_export($old, true) : json_encode($old) }}</span> → @endif{{ is_scalar($v) ? var_export($v, true) : json_encode($v) }}</div>
                            @endforeach
                        </td>
                    </tr>
                @empty
                    <tr><td class="td text-ink-500" colspan="5">Belum ada catatan.</td></tr>
                @endforelse
            </tbody>
        </table>
    </div>
@endsection
