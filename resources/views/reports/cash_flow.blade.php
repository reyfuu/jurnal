@extends('layouts.app')
@section('title', 'Arus Kas')

@section('content')
    <div class="mb-4 flex flex-wrap items-end justify-between gap-3">
        <div>
            <h2 class="text-lg font-semibold tracking-tight text-ink-900">Laporan Arus Kas</h2>
            <p class="text-xs text-ink-500">{{ $client->name }} · periode {{ $period }} · metode langsung</p>
        </div>
        @include('partials.period-picker')
    </div>

    @php
        $R = fn ($n) => \App\Support\Rupiah::format($n);
        $sections = [
            'operating' => 'Arus kas dari aktivitas operasi',
            'investing' => 'Arus kas dari aktivitas investasi',
            'financing' => 'Arus kas dari aktivitas pendanaan',
        ];
    @endphp

    <div class="card max-w-3xl overflow-hidden">
        <table class="w-full">
            @foreach ($sections as $key => $label)
                <tbody>
                    <tr><td class="th" colspan="2">{{ $label }}</td></tr>
                    @forelse ($report['groups'][$key] as $row)
                        <tr>
                            <td class="td pl-6">
                                {{ $row['value'] >= 0 ? 'Penerimaan' : 'Pembayaran' }} —
                                <span class="font-mono text-xs text-ink-400">{{ $row['account']->code }}</span> {{ $row['account']->name }}
                            </td>
                            <td class="td num w-48">{{ $row['value'] < 0 ? '('.$R(abs($row['value'])).')' : $R($row['value']) }}</td>
                        </tr>
                    @empty
                        <tr><td class="td pl-6 text-sm text-ink-400" colspan="2">—</td></tr>
                    @endforelse
                    <tr class="font-semibold">
                        <td class="td pl-6">Kas bersih {{ ['operating' => 'operasi', 'investing' => 'investasi', 'financing' => 'pendanaan'][$key] }}</td>
                        <td class="td num {{ $report['totals'][$key] < 0 ? 'text-rose-700' : '' }}">
                            {{ $report['totals'][$key] < 0 ? '('.$R(abs($report['totals'][$key])).')' : $R($report['totals'][$key]) }}
                        </td>
                    </tr>
                </tbody>
            @endforeach
            <tbody>
                <tr class="font-semibold">
                    <td class="td">Kenaikan (penurunan) kas bersih</td>
                    <td class="td num">{{ $report['net'] < 0 ? '('.$R(abs($report['net'])).')' : $R($report['net']) }}</td>
                </tr>
                <tr>
                    <td class="td">Kas & setara kas awal periode</td>
                    <td class="td num">{{ $R($report['opening']) }}</td>
                </tr>
                <tr class="bg-ink-900 font-bold text-white">
                    <td class="td border-0">KAS & SETARA KAS AKHIR PERIODE</td>
                    <td class="td num border-0">Rp{{ $R($report['closing']) }}</td>
                </tr>
            </tbody>
        </table>
    </div>

    <p class="mt-3 text-xs {{ $report['balanced'] ? 'text-emerald-700' : 'text-rose-700' }}">
        {{ $report['balanced'] ? '✓ Kas awal + arus bersih = kas akhir di buku.' : '✗ Kas awal + arus bersih ≠ kas akhir — ada jurnal yang perlu diperiksa.' }}
    </p>
    <p class="mt-1 max-w-3xl text-xs text-ink-400">
        Kas = akun berjenis kas, bank, dan e-wallet. Aktivitas ditentukan dari akun lawan: aset tetap → investasi;
        pinjaman, modal, prive → pendanaan; selain itu → operasi. Transfer antar rekening sendiri tidak dihitung.
    </p>
@endsection
