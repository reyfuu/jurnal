{{-- Panel pajak satu draft. Nominal yang masuk buku = baris jurnal di akun pajaknya;
     yang dikirim dari sini hanya pilihan manusia (dasar, nomor faktur). --}}
@if (! empty($draft['taxes']))
    @php($codeOf = $accounts->pluck('code', 'id'))
    <div class="mt-4 rounded-md border border-ink-200">
        <p class="border-b border-ink-200 bg-ink-50 px-3 py-1.5 text-[11px] font-semibold uppercase tracking-[0.08em] text-ink-500">Pajak di dokumen ini</p>
        <div class="overflow-x-auto">
            <table class="w-full text-sm">
                <thead><tr>
                    <th class="th w-14">Catat</th><th class="th">Jenis</th><th class="th num w-36">Dasar</th><th class="th num">Tertulis</th><th class="th num">Hitung sistem</th><th class="th">No. faktur / bukpot</th>
                </tr></thead>
                <tbody>
                    @foreach ($draft['taxes'] as $k => $t)
                        @php($n = "drafts[{$i}][taxes][{$k}]")
                        <tr class="align-top">
                            <td class="td">
                                <input type="hidden" name="{{ $n }}[record]" value="0">
                                <input type="checkbox" name="{{ $n }}[record]" value="1" checked aria-label="Catat {{ $t['label'] }} di rekap pajak">
                            </td>
                            <td class="td">
                                @foreach (['tax_code', 'account_id', 'direction', 'counterparty_name', 'counterparty_tax_id'] as $f)
                                    <input type="hidden" name="{{ $n }}[{{ $f }}]" value="{{ $t[$f] ?? '' }}">
                                @endforeach
                                <input type="hidden" name="{{ $n }}[withheld]" value="{{ $t['withheld'] ? 1 : 0 }}">
                                <input type="hidden" name="{{ $n }}[noncreditable]" value="{{ $t['noncreditable'] ? 1 : 0 }}">
                                <input type="hidden" name="{{ $n }}[amount]" value="{{ $t['amount'] }}">
                                <span class="font-medium">{{ $t['label'] }}</span>
                                <span class="block text-xs text-ink-500">
                                    @if ($t['noncreditable'])
                                        Klien non-PKP — digabung ke biaya
                                    @else
                                        {{ $t['direction'] === 'out' ? ($t['withheld'] ? 'Kita potong → utang, wajib setor' : 'Keluaran / dipungut') : ($t['withheld'] ? 'Dipotong pelanggan → minta bukti potong' : 'Masukan / dibayar dimuka') }}
                                        · akun <span class="font-mono">{{ $codeOf[$t['account_id']] ?? '?' }}</span>
                                    @endif
                                </span>
                            </td>
                            <td class="td"><input type="number" min="0" name="{{ $n }}[base]" value="{{ $t['base'] }}" class="field num" aria-label="Dasar pengenaan pajak"></td>
                            <td class="td num">{{ \App\Support\Rupiah::format($t['amount']) }}</td>
                            <td class="td num">
                                {{ $t['computed'] !== null ? \App\Support\Rupiah::format($t['computed']) : '—' }}
                                <span class="block">
                                    @switch($t['match_status'])
                                        @case('match') <span class="badge-ok">cocok</span> @break
                                        @case('mismatch') <span class="badge-bad">selisih</span> @break
                                        @case('no_rate') <span class="badge-mute">tanpa tarif</span> @break
                                        @default <span class="badge-mute">belum dicek</span>
                                    @endswitch
                                </span>
                            </td>
                            <td class="td"><input type="text" name="{{ $n }}[document_number]" value="{{ $t['document_number'] }}" class="field font-mono" maxlength="60"></td>
                        </tr>
                    @endforeach
                </tbody>
            </table>
        </div>
        <p class="px-3 py-2 text-xs text-ink-500">Nominal pajak yang dicatat mengikuti baris akun pajaknya di tabel atas. Status cocok/selisih dihitung ulang saat posting dengan tarif berlaku di tanggal jurnal.</p>
    </div>
@endif
