{{-- Daftar catatan pajak satu jenis. $rows: Collection<TaxEntry> --}}
<div class="overflow-x-auto">
    <table class="w-full">
        <thead>
            <tr>
                <th class="th">Tanggal</th>
                <th class="th">Jenis</th>
                <th class="th">Lawan · No. faktur/bukpot</th>
                <th class="th num">Dasar</th>
                <th class="th num">Pajak</th>
                <th class="th">Cek tarif</th>
            </tr>
        </thead>
        <tbody>
            @foreach ($rows as $e)
                <tr class="hover:bg-ink-50">
                    <td class="td whitespace-nowrap">{{ \Illuminate\Support\Carbon::parse($e->journal_date)->format('d/m/Y') }}</td>
                    <td class="td">
                        {{ $e->codeName() }}
                        <span class="block text-xs text-ink-500">
                            {{ $e->direction === 'out' ? ($e->withheld ? 'kita potong · wajib setor' : 'keluaran') : ($e->withheld ? 'dipotong pihak lain' : 'masukan') }}
                        </span>
                    </td>
                    <td class="td">
                        {{ $e->counterparty_name ?: '—' }}
                        <span class="block font-mono text-xs text-ink-500">
                            {{ $e->document_number ?: 'tanpa nomor' }} · NPWP {{ \App\Support\TaxId::mask($e->counterparty_tax_id) }}
                        </span>
                    </td>
                    <td class="td num">{{ \App\Support\Rupiah::format($e->base) }}</td>
                    <td class="td num font-medium">{{ \App\Support\Rupiah::format($e->amount) }}</td>
                    <td class="td">
                        @switch($e->match_status)
                            @case('match') <span class="badge-ok">cocok</span> @break
                            @case('mismatch') <span class="badge-bad" title="Hitungan sistem Rp{{ \App\Support\Rupiah::format($e->computed_amount) }}">selisih</span> @break
                            @case('no_rate') <span class="badge-mute">tanpa tarif</span> @break
                            @default <span class="badge-mute">manual</span>
                        @endswitch
                        <span class="ml-1 font-mono text-xs text-ink-400">#{{ $e->journal_id }}</span>
                    </td>
                </tr>
            @endforeach
        </tbody>
    </table>
</div>
