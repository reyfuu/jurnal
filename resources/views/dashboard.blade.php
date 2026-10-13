@extends('layouts.app')
@section('title', 'Dashboard')

@section('content')
    <div class="mb-8 flex flex-wrap items-end justify-between gap-4">
        <div>
            <p class="label mb-0.5">Ringkasan {{ $period }}</p>
            <h1 class="text-2xl font-semibold tracking-tight text-ink-900">Dashboard</h1>
        </div>
    </div>

    @unless ($aiReady)
        <div class="mb-6 rounded-[10px] border border-ai-500/30 bg-ai-50 px-4 py-3 text-sm text-ink-800">
            <p class="font-semibold text-ai-700">Otak AI belum terpasang.</p>
            <p>Isi <code class="font-mono">OPENAI_API_KEY</code> (atau <code class="font-mono">ANTHROPIC_API_KEY</code> + set
            <code class="font-mono">AI_PROVIDER=anthropic</code>) di <code class="font-mono">.env</code>.
            Tanpa itu, upload dokumen tetap tersimpan tapi tidak bisa dibaca otomatis — jurnal manual tetap jalan.</p>
        </div>
    @endunless

    <div class="card mb-8 grid sm:grid-cols-[1.4fr_repeat(3,1fr)] divide-y sm:divide-y-0 sm:divide-x divide-ink-200">
        <div class="p-5 {{ $totals['pending'] ? 'bg-ai-50/60' : '' }}">
            <p class="label">Perlu direview</p>
            <p class="font-mono text-4xl font-semibold tabular-nums tracking-tight {{ $totals['pending'] ? 'text-ai-700' : 'text-ink-900' }}">{{ number_format($totals['pending'], 0, ',', '.') }}</p>
            <p class="mt-1 text-xs text-ink-500">{{ $totals['pending'] ? 'dokumen menunggu dicek manusia sebelum masuk buku' : 'semua dokumen sudah ditangani' }}</p>
        </div>
        @foreach ([
            ['Klien aktif', $totals['clients'], 'buku yang dikelola'],
            ['Dokumen masuk', $totals['documents'], 'struk, invoice, mutasi'],
            ['Jurnal posted', $totals['journals'], 'sudah masuk buku'],
        ] as [$label, $value, $hint])
            <div class="p-5">
                <p class="label">{{ $label }}</p>
                <p class="font-mono text-2xl font-semibold tabular-nums text-ink-900">{{ number_format($value, 0, ',', '.') }}</p>
                <p class="mt-1 text-xs text-ink-400">{{ $hint }}</p>
            </div>
        @endforeach
    </div>

    @if ($taxDue->isNotEmpty())
        <section class="mb-8">
            <h2 class="mb-2 font-semibold text-ink-900">Pajak jatuh tempo bulan ini</h2>
            <div class="card overflow-x-auto">
                <table class="w-full">
                    <thead><tr><th class="th">Klien</th><th class="th">Pajak</th><th class="th">Masa</th><th class="th">Setor sebelum</th><th class="th">Lapor sebelum</th><th class="th">Status</th></tr></thead>
                    <tbody>
                        @foreach ($taxDue as $o)
                            <tr class="hover:bg-ink-50">
                                <td class="td"><a href="{{ route('tax.index', $o['client']) }}?period={{ $o['period'] }}" class="font-medium text-brand-700 hover:underline">{{ $o['client']->name }}</a></td>
                                <td class="td">{{ $o['name'] }}</td>
                                <td class="td font-mono text-xs">{{ $o['period'] }}</td>
                                <td class="td num text-left">{{ \Illuminate\Support\Carbon::parse($o['pay_due'])->format('d/m') }}</td>
                                <td class="td num text-left">{{ \Illuminate\Support\Carbon::parse($o['file_due'])->format('d/m') }}</td>
                                <td class="td"><span class="{{ $o['overdue'] ? 'badge-bad' : ($o['status'] === 'setor' ? 'badge-info' : 'badge-mute') }}">{{ $o['overdue'] ? 'Lewat · ' : '' }}{{ \App\Models\TaxFiling::STATUSES[$o['status']] }}</span></td>
                            </tr>
                        @endforeach
                    </tbody>
                </table>
            </div>
        </section>
    @endif

    <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h2 class="font-semibold text-ink-900">Posisi tiap klien — {{ $period }}</h2>
        @if (auth()->user()->isAdmin())
            <a href="{{ route('clients.create') }}" class="btn-primary">+ Klien baru</a>
        @endif
    </div>

    @if ($rows->isEmpty())
        <div class="card p-10 text-center">
            <p class="font-semibold text-ink-800">Belum ada klien.</p>
            <p class="mt-1 text-sm text-ink-500">Buat klien dulu — tiap klien dapat COA standar dan buku jurnalnya sendiri.</p>
            @if (auth()->user()->isAdmin())
                <a href="{{ route('clients.create') }}" class="btn-primary mt-4">Buat klien pertama</a>
            @endif
        </div>
    @else
        <div class="card overflow-x-auto">
            <table class="w-full">
                <thead>
                    <tr>
                        <th class="th">Klien</th>
                        <th class="th num">Pendapatan</th>
                        <th class="th num">Beban</th>
                        <th class="th num">Laba/Rugi</th>
                        <th class="th num">Saldo Kas</th>
                        <th class="th num">Perlu review</th>
                    </tr>
                </thead>
                <tbody>
                    @foreach ($rows as $row)
                        @php $s = $row['snapshot']; @endphp
                        <tr class="hover:bg-ink-50">
                            <td class="td">
                                <a href="{{ route('clients.show', $row['client']) }}" class="font-semibold text-brand-700 hover:underline">
                                    {{ $row['client']->name }}
                                </a>
                                <span class="ml-1 {{ $row['client']->type === \App\Models\Client::TYPE_INTERNAL ? 'badge-info' : 'badge-mute' }}">
                                    {{ $row['client']->type === \App\Models\Client::TYPE_INTERNAL ? 'sendiri' : 'klien' }}
                                </span>
                            </td>
                            <td class="td num">{{ \App\Support\Rupiah::format($s['revenue']) }}</td>
                            <td class="td num">{{ \App\Support\Rupiah::format($s['expense']) }}</td>
                            <td class="td num font-semibold {{ $s['net_income'] >= 0 ? '' : 'text-rose-700' }}">
                                {{ \App\Support\Rupiah::format($s['net_income']) }}
                            </td>
                            <td class="td num">{{ \App\Support\Rupiah::format($s['cash']) }}</td>
                            <td class="td num">
                                @if ($row['pending'] > 0)
                                    <a href="{{ route('documents.index', $row['client']) }}" class="badge-warn">{{ $row['pending'] }} dokumen</a>
                                @else
                                    <span class="text-ink-300">—</span>
                                @endif
                            </td>
                        </tr>
                    @endforeach
                </tbody>
            </table>
        </div>
        <p class="mt-2 text-xs text-ink-400">Angka pendapatan/beban/laba = mutasi bulan {{ $period }}. Saldo kas = akumulasi sampai akhir bulan itu.</p>
    @endif
@endsection
