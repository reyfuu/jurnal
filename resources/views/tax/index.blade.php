@extends('layouts.app')
@section('title', 'Pajak · '.$client->name)

@section('content')
    @php($R = fn ($n) => \App\Support\Rupiah::format((float) $n))

    <div class="mb-6 flex flex-wrap items-end justify-between gap-4">
        <div>
            <p class="label mb-0.5">Pajak masa {{ $period }}</p>
            <h2 class="text-xl font-semibold tracking-tight text-ink-900">Ringkasan pajak</h2>
            <p class="mt-1 flex flex-wrap gap-1.5 text-xs">
                <span class="{{ $client->isPkpOn() ? 'badge-info' : 'badge-mute' }}">{{ $client->isPkpOn() ? 'PKP' : 'Non-PKP' }}</span>
                <span class="badge-mute">{{ $client->isFinalUmkm() ? 'PPh Final UMKM' : 'PPh normal' }}</span>
                @if ($client->taxpayer_type)<span class="badge-mute">{{ \App\Models\Client::TAXPAYER_TYPES[$client->taxpayer_type] }}</span>@endif
                <a href="{{ route('tax.profile', $client) }}" class="text-brand-700 hover:underline">Profil pajak →</a>
            </p>
        </div>
        @include('partials.period-picker')
    </div>

    @unless ($client->hasTaxProfile())
        <div class="mb-6 rounded-[10px] border border-ai-500/30 bg-ai-50 px-4 py-3 text-sm text-ink-800">
            <p class="font-semibold text-ai-700">Profil pajak belum diisi.</p>
            <p>Sampai diisi, klien diperlakukan sebagai <b>PKP, rezim PPh normal</b> (perilaku lama). Isi di
                <a href="{{ route('tax.profile', $client) }}" class="font-medium underline">Profil pajak</a> supaya PPN klien non-PKP masuk biaya dan rekap Final UMKM muncul.</p>
        </div>
    @endunless

    @if ($mismatch)
        <div class="mb-6 rounded-[10px] border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-900">
            <p class="font-semibold">Rekap pajak tidak sama dengan buku untuk {{ count($mismatch) }} akun.</p>
            <ul class="mt-1 list-disc pl-5">
                @foreach ($mismatch as $m)
                    <li><span class="font-mono">{{ $m['account']->code }}</span> {{ $m['account']->name }}: di buku Rp{{ $R($m['booked']) }}, tercatat sebagai pajak Rp{{ $R($m['recorded']) }}.</li>
                @endforeach
            </ul>
            <p class="mt-1 text-xs">Biasanya karena jurnal manual memakai akun pajak umum (mis. 2103 Hutang Pajak) yang tidak bisa dipetakan otomatis ke jenis pajak. Rekap di bawah hanya menghitung yang tercatat.</p>
        </div>
    @endif

    {{-- ── Kewajiban & status ─────────────────────────────────────────────── --}}
    <section class="mb-8">
        <h3 class="mb-2 font-semibold text-ink-900">Kewajiban masa ini</h3>
        @if ($obligations === [])
            <p class="card p-5 text-sm text-ink-500">Tidak ada kewajiban terdeteksi untuk masa {{ $period }}.</p>
        @else
            <div class="card overflow-x-auto">
                <table class="w-full">
                    <thead><tr>
                        <th class="th">Pajak</th><th class="th">Setor sebelum</th><th class="th">Lapor sebelum</th><th class="th">Status</th><th class="th"></th>
                    </tr></thead>
                    <tbody>
                        @foreach ($obligations as $o)
                            <tr class="align-top">
                                <td class="td font-medium">{{ $o['name'] }}</td>
                                <td class="td num text-left">{{ \Illuminate\Support\Carbon::parse($o['pay_due'])->format('d/m/Y') }}</td>
                                <td class="td num text-left">{{ \Illuminate\Support\Carbon::parse($o['file_due'])->format('d/m/Y') }}</td>
                                <td class="td">
                                    <span class="{{ $o['overdue'] ? 'badge-bad' : (in_array($o['status'], ['lapor', 'nihil']) ? 'badge-ok' : ($o['status'] === 'setor' ? 'badge-info' : 'badge-mute')) }}">
                                        {{ $o['overdue'] ? 'Lewat jatuh tempo · ' : '' }}{{ \App\Models\TaxFiling::STATUSES[$o['status']] }}
                                    </span>
                                    @if ($o['filing']?->ntpn)<span class="block font-mono text-xs text-ink-500">NTPN {{ $o['filing']->ntpn }}</span>@endif
                                </td>
                                <td class="td">
                                    <details>
                                        <summary class="btn-ghost btn-sm list-none">Catat status</summary>
                                        <form method="POST" action="{{ route('tax.filing', $client) }}" class="mt-2 grid w-72 gap-2">
                                            @csrf
                                            <input type="hidden" name="tax_code" value="{{ $o['tax_code'] }}">
                                            <input type="hidden" name="period" value="{{ $period }}">
                                            <select name="status" class="field">
                                                @foreach (\App\Models\TaxFiling::STATUSES as $k => $v)
                                                    <option value="{{ $k }}" @selected($o['status'] === $k)>{{ $v }}</option>
                                                @endforeach
                                            </select>
                                            <input name="ntpn" class="field font-mono" placeholder="NTPN (wajib saat setor)" value="{{ $o['filing']?->ntpn }}">
                                            <input name="paid_amount" type="number" min="0" class="field num" placeholder="Jumlah setor" value="{{ $o['filing']?->paid_amount }}">
                                            <label class="text-xs text-ink-500">Tanggal setor <input type="date" name="paid_at" class="field" value="{{ $o['filing']?->paid_at?->toDateString() }}"></label>
                                            <label class="text-xs text-ink-500">Tanggal lapor <input type="date" name="filed_at" class="field" value="{{ $o['filing']?->filed_at?->toDateString() }}"></label>
                                            <input name="receipt_number" class="field font-mono" placeholder="No. BPE (bukti lapor)" value="{{ $o['filing']?->receipt_number }}">
                                            <button class="btn-primary btn-sm">Simpan</button>
                                        </form>
                                    </details>
                                </td>
                            </tr>
                        @endforeach
                    </tbody>
                </table>
            </div>
            <p class="mt-1 text-xs text-ink-400">Sabtu/Minggu digeser ke Senin. Hari libur nasional belum dihitung — cek kalender DJP.</p>
        @endif
    </section>

    {{-- ── PPN ────────────────────────────────────────────────────────────── --}}
    @if ($ppn)
        <section class="mb-8">
            <div class="mb-2 flex items-baseline justify-between gap-2">
                <h3 class="font-semibold text-ink-900">PPN</h3>
                <a href="{{ route('tax.export', [$client, 'ppn']) }}?period={{ $period }}" class="text-sm text-brand-700 hover:underline">Ekspor CSV</a>
            </div>
            <div class="card mb-3 grid divide-y sm:grid-cols-3 sm:divide-x sm:divide-y-0 divide-ink-200">
                <div class="p-4"><p class="label">PPN Keluaran</p><p class="num text-left text-lg font-semibold">{{ $R($ppn['keluaran']) }}</p></div>
                <div class="p-4"><p class="label">PPN Masukan</p><p class="num text-left text-lg font-semibold">{{ $R($ppn['masukan']) }}</p></div>
                <div class="p-4">
                    <p class="label">{{ $ppn['net'] >= 0 ? 'Kurang bayar' : 'Lebih bayar' }}</p>
                    <p class="num text-left text-lg font-semibold {{ $ppn['net'] > 0 ? 'text-ai-700' : '' }}">{{ $R(abs($ppn['net'])) }}</p>
                    @if ($ppn['net'] > 0)
                        <a href="{{ route('tax.draft', [$client, 'ppn-setor']) }}?period={{ $period }}" class="mt-1 inline-block text-xs text-brand-700 hover:underline">Buat draft jurnal setor →</a>
                    @elseif ($ppn['net'] < 0)
                        <p class="mt-1 text-xs text-ink-500">Lebih bayar — umumnya dikompensasi ke masa berikutnya.</p>
                    @endif
                </div>
            </div>
            @if ($ppn['rows']->isNotEmpty())
                <div class="card overflow-hidden">@include('tax._entries', ['rows' => $ppn['rows']])</div>
            @endif
        </section>
    @endif

    {{-- ── PPh potong/pungut ──────────────────────────────────────────────── --}}
    <section class="mb-8">
        <div class="mb-2 flex items-baseline justify-between gap-2">
            <h3 class="font-semibold text-ink-900">PPh potong / pungut</h3>
            @if ($pph['rows']->isNotEmpty())
                <a href="{{ route('tax.export', [$client, 'pph']) }}?period={{ $period }}" class="text-sm text-brand-700 hover:underline">Ekspor CSV</a>
            @endif
        </div>
        @if ($pph['groups']->isEmpty())
            <p class="card p-5 text-sm text-ink-500">Tidak ada PPh potong/pungut tercatat di masa ini.</p>
        @else
            <div class="card mb-3 overflow-x-auto">
                <table class="w-full">
                    <thead><tr><th class="th">Jenis</th><th class="th">Posisi</th><th class="th num">Transaksi</th><th class="th num">Dasar</th><th class="th num">PPh</th></tr></thead>
                    <tbody>
                        @foreach ($pph['groups'] as $g)
                            <tr>
                                <td class="td font-medium">{{ config("pajak.codes.{$g['tax_code']}.name") }}</td>
                                <td class="td">{{ $g['direction'] === 'out' ? 'Utang — wajib disetor klien' : 'Kredit pajak — minta bukti potong' }}</td>
                                <td class="td num">{{ $g['count'] }}</td>
                                <td class="td num">{{ $R($g['base']) }}</td>
                                <td class="td num font-semibold">
                                    {{ $R($g['amount']) }}
                                    @if ($g['direction'] === 'out')
                                        <a href="{{ route('tax.draft', [$client, 'pph-setor']) }}?period={{ $period }}&kode={{ $g['tax_code'] }}" class="block text-xs font-normal text-brand-700 hover:underline">draft setor →</a>
                                    @endif
                                </td>
                            </tr>
                        @endforeach
                    </tbody>
                </table>
            </div>
            <div class="card overflow-hidden">@include('tax._entries', ['rows' => $pph['rows']])</div>
        @endif
    </section>

    {{-- ── PPh Final UMKM ─────────────────────────────────────────────────── --}}
    @if ($final)
        <section class="mb-8">
            <div class="mb-2 flex items-baseline justify-between gap-2">
                <h3 class="font-semibold text-ink-900">PPh Final UMKM</h3>
                <a href="{{ route('tax.export', [$client, 'final']) }}?period={{ $period }}" class="text-sm text-brand-700 hover:underline">Ekspor CSV</a>
            </div>
            <div class="card max-w-xl p-5">
                <dl class="space-y-1.5 text-sm">
                    <div class="flex justify-between"><dt class="text-ink-500">Omzet bruto masa ini</dt><dd class="num">{{ $R($final['omzet']) }}</dd></div>
                    @if ($final['rate']?->exempt_threshold)
                        <div class="flex justify-between"><dt class="text-ink-500">Omzet Jan–bulan lalu</dt><dd class="num">{{ $R($final['omzet_ytd_before']) }}</dd></div>
                        <div class="flex justify-between"><dt class="text-ink-500">Bagian bebas (batas Rp{{ $R($final['rate']->exempt_threshold) }}/tahun)</dt><dd class="num">({{ $R($final['exempt']) }})</dd></div>
                    @endif
                    <div class="flex justify-between border-t border-ink-100 pt-1.5"><dt class="text-ink-500">Dasar pengenaan</dt><dd class="num">{{ $R($final['taxable']) }}</dd></div>
                    <div class="flex justify-between"><dt class="text-ink-500">Tarif</dt><dd>{{ $final['rate']?->label() ?? 'tidak ada tarif berlaku' }}</dd></div>
                    <div class="flex justify-between border-t border-ink-200 pt-1.5 font-semibold"><dt>PPh Final terutang</dt><dd class="num">{{ $final['tax'] === null ? '—' : 'Rp'.$R($final['tax']) }}</dd></div>
                </dl>
                @if ($final['tax'])
                    <div class="mt-4 flex flex-wrap gap-2">
                        <a href="{{ route('tax.draft', [$client, 'final-akrual']) }}?period={{ $period }}" class="btn-ghost btn-sm">Draft jurnal akrual</a>
                        <a href="{{ route('tax.draft', [$client, 'final-setor']) }}?period={{ $period }}" class="btn-ghost btn-sm">Draft jurnal setor</a>
                    </div>
                @endif
                <p class="mt-3 text-xs text-ink-400">Omzet = akun pendapatan usaha (Penjualan, Pendapatan Jasa) dari jurnal posted. Pendapatan lain-lain tidak dihitung.</p>
            </div>
        </section>
    @endif

    @include('tax._disclaimer')
@endsection
