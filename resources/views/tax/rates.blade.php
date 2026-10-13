@extends('layouts.app')
@section('title', 'Tarif pajak')

@section('content')
    <div class="mb-6">
        <p class="label mb-0.5">Pajak · admin</p>
        <h1 class="text-2xl font-semibold tracking-tight text-ink-900">Tarif pajak</h1>
        <p class="mt-1 max-w-2xl text-sm text-ink-500">Tarif tidak pernah diedit. Perubahan regulasi = versi baru dengan tanggal berlaku; versi lama ditutup otomatis dan transaksi lama tetap memakai tarif di tanggalnya.</p>
    </div>

    <div class="card mb-8 overflow-x-auto">
        <table class="w-full">
            <thead><tr>
                <th class="th">Pajak</th><th class="th">Berlaku untuk</th><th class="th">Tarif</th><th class="th">Bebas/tahun</th>
                <th class="th">Berlaku</th><th class="th">Dasar hukum</th><th class="th">Verifikasi</th>
            </tr></thead>
            <tbody>
                @foreach ($rates as $r)
                    <tr class="{{ $r->effective_to ? 'text-ink-400' : '' }}">
                        <td class="td font-medium">{{ config("pajak.codes.{$r->tax_code}.name", $r->tax_code) }}</td>
                        <td class="td">{{ $r->applies_to ? \App\Models\Client::TAXPAYER_TYPES[$r->applies_to] : 'Semua' }}</td>
                        <td class="td font-mono text-xs">{{ $r->label() }}{{ $r->rounding === 'half_up' ? ' · bulat ½' : '' }}</td>
                        <td class="td num text-left">{{ $r->exempt_threshold ? \App\Support\Rupiah::format($r->exempt_threshold) : '—' }}</td>
                        <td class="td whitespace-nowrap text-xs">{{ $r->effective_from->format('d/m/Y') }} – {{ $r->effective_to?->format('d/m/Y') ?? 'sekarang' }}</td>
                        <td class="td text-xs">{{ $r->source }} @if ($used->has($r->id))<span class="badge-mute ml-1">terpakai</span>@endif</td>
                        <td class="td">
                            @if ($r->verified_at)
                                <span class="badge-ok">{{ $r->verified_at->format('d/m/Y') }}</span>
                            @else
                                <form method="POST" action="{{ route('tax.rates.verify', $r) }}" class="flex items-center gap-1.5">
                                    @csrf
                                    <span class="badge-warn">belum</span>
                                    <button class="text-xs text-brand-700 hover:underline cursor-pointer">tandai sudah dicek konsultan</button>
                                </form>
                            @endif
                        </td>
                    </tr>
                @endforeach
            </tbody>
        </table>
    </div>

    <h2 class="mb-2 font-semibold text-ink-900">Tambah versi tarif</h2>
    <form method="POST" action="{{ route('tax.rates.store') }}" class="card grid max-w-3xl gap-4 p-6 sm:grid-cols-3">
        @csrf
        <div>
            <label class="label" for="tax_code">Pajak</label>
            <select id="tax_code" name="tax_code" class="field">
                @foreach (config('pajak.codes') as $k => $c)
                    @continue($k === 'other')
                    <option value="{{ $k }}" @selected(old('tax_code') === $k)>{{ $c['name'] }}</option>
                @endforeach
            </select>
        </div>
        <div>
            <label class="label" for="applies_to">Berlaku untuk</label>
            <select id="applies_to" name="applies_to" class="field">
                <option value="">Semua</option>
                @foreach (\App\Models\Client::TAXPAYER_TYPES as $k => $v)<option value="{{ $k }}" @selected(old('applies_to') === $k)>{{ $v }}</option>@endforeach
            </select>
        </div>
        <div>
            <label class="label" for="effective_from">Berlaku mulai</label>
            <input type="date" id="effective_from" name="effective_from" class="field" required value="{{ old('effective_from') }}">
        </div>
        <div>
            <label class="label" for="rate_percent">Tarif (%)</label>
            <input id="rate_percent" name="rate_percent" class="field num" placeholder="12" required value="{{ old('rate_percent') }}">
        </div>
        <div>
            <label class="label" for="rate_no_tax_id_percent">Tarif tanpa NPWP (%)</label>
            <input id="rate_no_tax_id_percent" name="rate_no_tax_id_percent" class="field num" placeholder="opsional" value="{{ old('rate_no_tax_id_percent') }}">
        </div>
        <div>
            <span class="label">Faktor DPP</span>
            <div class="flex items-center gap-1">
                <input name="dpp_numerator" type="number" min="1" class="field num" value="{{ old('dpp_numerator', 1) }}" aria-label="Pembilang DPP">
                <span class="text-ink-400">/</span>
                <input name="dpp_denominator" type="number" min="1" class="field num" value="{{ old('dpp_denominator', 1) }}" aria-label="Penyebut DPP">
            </div>
        </div>
        <div>
            <label class="label" for="exempt_threshold">Omzet bebas/tahun (Rp)</label>
            <input id="exempt_threshold" name="exempt_threshold" type="number" min="0" class="field num" placeholder="opsional" value="{{ old('exempt_threshold') }}">
        </div>
        <div>
            <label class="label" for="rounding">Pembulatan</label>
            <select id="rounding" name="rounding" class="field">
                <option value="floor">Ke bawah</option>
                <option value="half_up" @selected(old('rounding') === 'half_up')>Setengah ke atas</option>
            </select>
        </div>
        <div>
            <label class="label" for="source">Dasar hukum</label>
            <input id="source" name="source" class="field" required placeholder="mis. PMK 131/2024" value="{{ old('source') }}">
        </div>
        <div class="sm:col-span-3"><button class="btn-primary">Simpan versi baru</button></div>
    </form>

    @include('tax._disclaimer')
@endsection
