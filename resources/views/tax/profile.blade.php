@extends('layouts.app')
@section('title', 'Profil pajak · '.$client->name)

@section('content')
    @php($admin = auth()->user()->isAdmin())
    <div class="mb-6">
        <p class="label mb-0.5">Pajak</p>
        <h2 class="text-xl font-semibold tracking-tight text-ink-900">Profil pajak</h2>
        <p class="mt-1 text-sm text-ink-500">Menentukan cara PPN dan PPh dijurnal untuk klien ini. Diisi manusia — AI tidak pernah menebak status PKP atau rezim.</p>
    </div>

    <form method="POST" action="{{ route('tax.profile.update', $client) }}" class="card max-w-2xl space-y-5 p-6">
        @csrf @method('PUT')
        <fieldset @disabled(! $admin) class="space-y-5">
            <div class="grid gap-4 sm:grid-cols-2">
                <div>
                    <label class="label" for="taxpayer_type">Jenis wajib pajak</label>
                    <select id="taxpayer_type" name="taxpayer_type" class="field">
                        <option value="">— belum diisi —</option>
                        @foreach (\App\Models\Client::TAXPAYER_TYPES as $k => $v)
                            <option value="{{ $k }}" @selected(old('taxpayer_type', $client->taxpayer_type) === $k)>{{ $v }}</option>
                        @endforeach
                    </select>
                </div>
                <div>
                    <label class="label" for="tax_id">NPWP / NIK (16 digit)</label>
                    <input id="tax_id" name="tax_id" inputmode="numeric" class="field font-mono" autocomplete="off"
                           value="{{ old('tax_id', $admin ? $client->tax_id : \App\Support\TaxId::mask($client->tax_id)) }}">
                    <p class="mt-1 text-xs text-ink-400">Disimpan terenkripsi. Ditampilkan termasking di laporan.</p>
                </div>
            </div>

            <div class="grid gap-4 sm:grid-cols-2">
                <div>
                    <span class="label">Status PPN</span>
                    @foreach (['1' => 'PKP — memungut & mengkreditkan PPN', '0' => 'Non-PKP — PPN pembelian masuk biaya'] as $v => $label)
                        <label class="flex items-center gap-2 py-1 text-sm">
                            <input type="radio" name="is_pkp" value="{{ $v }}" @checked((string) old('is_pkp', $client->is_pkp === null ? '' : (int) $client->is_pkp) === $v)> {{ $label }}
                        </label>
                    @endforeach
                </div>
                <div>
                    <label class="label" for="pkp_since">PKP sejak</label>
                    <input type="date" id="pkp_since" name="pkp_since" class="field" value="{{ old('pkp_since', $client->pkp_since?->toDateString()) }}">
                    <p class="mt-1 text-xs text-ink-400">Kosong = sejak awal. Transaksi sebelum tanggal ini diperlakukan non-PKP.</p>
                </div>
            </div>

            <div>
                <span class="label">Rezim PPh</span>
                @foreach (\App\Models\Client::REGIMES as $v => $label)
                    <label class="flex items-center gap-2 py-1 text-sm">
                        <input type="radio" name="pph_regime" value="{{ $v }}" @checked(old('pph_regime', $client->pph_regime) === $v)> {{ $label }}
                    </label>
                @endforeach
            </div>
        </fieldset>

        @if ($admin)
            <div class="flex gap-2 border-t border-ink-100 pt-4">
                <button class="btn-primary">Simpan profil</button>
                <a href="{{ route('tax.index', $client) }}" class="btn-ghost">Batal</a>
            </div>
        @else
            <p class="border-t border-ink-100 pt-4 text-sm text-ink-500">Hanya admin yang bisa mengubah profil pajak.</p>
        @endif
    </form>

    @include('tax._disclaimer')
@endsection
