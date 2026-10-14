@extends('layouts.app')
@section('title', 'COA')

@section('content')
    @php
        $canManage = auth()->user()->may('clients.manage');
        $flat = $accounts->flatten();
        // Validasi gagal → modal dibuka lagi dengan isian lama (field _dialog = 'new' atau id akun).
        $reopen = old('_dialog');
        $editing = $reopen && $reopen !== 'new' ? $flat->firstWhere('id', (int) $reopen) : null;
    @endphp

    <div class="mb-4 flex flex-wrap items-end justify-between gap-3">
        <div>
            <h2 class="text-lg font-semibold tracking-tight text-ink-900">Chart of Account</h2>
            <p class="text-xs text-ink-500">Template <span class="font-medium text-ink-700">{{ $client->businessTypeLabel() }}</span> · {{ $flat->count() }} akun · COA ini milik {{ $client->name }} saja — tidak tercampur perusahaan lain.</p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
            <a href="{{ route('accounts.export', $client) }}" class="btn-ghost btn-sm"><x-icon name="download"/> Ekspor Excel</a>
            @if ($canManage)
                <button type="button" class="btn-ghost btn-sm" data-dialog-open="import-dialog"><x-icon name="upload"/> Impor Excel</button>
                <form method="POST" action="{{ route('clients.coa.sync', $client) }}">
                    @csrf
                    <button class="btn-ghost btn-sm" title="Tambah akun template {{ $client->businessTypeLabel() }} yang belum ada. Akun yang sudah ada tidak diubah.">Sinkron dari template</button>
                </form>
                <button type="button" class="btn-primary btn-sm" data-account-new><x-icon name="plus"/> Akun baru</button>
            @endif
        </div>
    </div>

    @foreach ($types as $type => $typeLabel)
        @php $group = $accounts[$type] ?? collect(); @endphp
        @if ($group->isNotEmpty())
            <section class="card mb-4 overflow-hidden">
                <div class="overflow-x-auto">
                    <table class="w-full">
                        <thead>
                            <tr>
                                <th class="th w-24">{{ $typeLabel }} <span class="font-normal normal-case text-ink-400">({{ $group->count() }})</span></th>
                                <th class="th">Nama akun</th>
                                <th class="th w-40">Subtype</th>
                                <th class="th w-28">Saldo normal</th>
                                <th class="th w-24">Status</th>
                                @if ($canManage)<th class="th w-36"><span class="sr-only">Aksi</span></th>@endif
                            </tr>
                        </thead>
                        <tbody>
                            @foreach ($group as $account)
                                <tr class="hover:bg-ink-50 {{ $account->is_active ? '' : 'text-ink-400' }}">
                                    <td class="td font-mono text-xs">{{ $account->code }}</td>
                                    <td class="td">
                                        {{ $account->name }}
                                        @if ($account->used)<span class="ml-1 text-[11px] text-ink-400" title="Sudah dipakai jurnal — tipe terkunci, hapus = nonaktifkan">· dipakai</span>@endif
                                    </td>
                                    <td class="td font-mono text-xs text-ink-500">{{ $account->subtype ?? '—' }}</td>
                                    <td class="td text-xs text-ink-500">{{ $account->normal_balance === 'debit' ? 'Debit' : 'Kredit' }}</td>
                                    <td class="td"><span class="{{ $account->is_active ? 'badge-ok' : 'badge-mute' }}">{{ $account->is_active ? 'aktif' : 'nonaktif' }}</span></td>
                                    @if ($canManage)
                                        <td class="td whitespace-nowrap text-right">
                                            <button type="button" class="btn-ghost btn-sm" data-account-edit="{{ json_encode([
                                                'url' => route('accounts.update', [$client, $account]), 'id' => $account->id,
                                                'code' => $account->code, 'name' => $account->name, 'type' => $account->type,
                                                'subtype' => $account->subtype, 'normal_balance' => $account->normal_balance,
                                                'is_active' => $account->is_active, 'used' => (bool) $account->used,
                                            ]) }}">Ubah</button>
                                            <form method="POST" action="{{ route('accounts.destroy', [$client, $account]) }}" class="inline"
                                                  data-confirm="Hapus akun {{ $account->code }} {{ $account->name }}?{{ $account->used ? ' Akun ini sudah dipakai jurnal, jadi hanya dinonaktifkan.' : '' }}">
                                                @csrf @method('DELETE')
                                                <button class="btn-danger-quiet btn-sm">Hapus</button>
                                            </form>
                                        </td>
                                    @endif
                                </tr>
                            @endforeach
                        </tbody>
                    </table>
                </div>
            </section>
        @endif
    @endforeach

    @if ($canManage)
        {{-- Modal tambah/ubah akun — satu form, diisi JS sesuai tombol yang diklik. --}}
        <dialog id="account-dialog" class="m-auto w-full max-w-lg rounded-[10px] border border-ink-200 bg-white p-0 shadow-[0_1px_2px_rgb(0_0_0/.05),0_8px_24px_rgb(0_0_0/.08)] backdrop:bg-ink-900/30"
                data-store-url="{{ route('accounts.store', $client) }}" @if ($reopen) data-autopen @endif>
            <form method="POST" action="{{ $editing ? route('accounts.update', [$client, $editing]) : route('accounts.store', $client) }}" class="p-5" data-account-form>
                @csrf
                <input type="hidden" name="_method" value="PUT" @disabled(! $editing)>
                <input type="hidden" name="_dialog" value="{{ $reopen ?: 'new' }}">
                <div class="mb-4 flex items-start justify-between gap-3">
                    <h3 class="font-semibold text-ink-900" data-dialog-title>{{ $editing ? "Ubah akun {$editing->code}" : 'Akun baru' }}</h3>
                    <button type="button" class="nav-link -mr-2 -mt-1" data-dialog-close aria-label="Tutup"><x-icon name="x"/></button>
                </div>

                @if ($reopen && $errors->any())
                    <ul class="mb-4 space-y-0.5 rounded-md bg-rose-50 px-3 py-2 text-sm text-rose-800" data-dialog-errors>
                        @foreach ($errors->all() as $e)<li>{{ $e }}</li>@endforeach
                    </ul>
                @endif

                <div class="grid gap-4 sm:grid-cols-[8rem_1fr]">
                    <div>
                        <label class="label" for="acc-code">Kode</label>
                        <input id="acc-code" name="code" required maxlength="20" class="field font-mono" placeholder="6117" value="{{ old('code', $editing?->code) }}">
                    </div>
                    <div>
                        <label class="label" for="acc-name">Nama akun</label>
                        <input id="acc-name" name="name" required maxlength="160" class="field" placeholder="Beban Riset Produk" value="{{ old('name', $editing?->name) }}">
                    </div>
                    <div>
                        <label class="label" for="acc-type">Tipe</label>
                        <select id="acc-type" name="type" class="field" @disabled($editing?->used)>
                            @foreach ($types as $value => $label)
                                <option value="{{ $value }}" @selected(old('type', $editing?->type ?? 'expense') === $value)>{{ $label }}</option>
                            @endforeach
                        </select>
                        {{-- Tipe terkunci (akun sudah dipakai): select disabled tidak terkirim, jadi nilainya lewat hidden. --}}
                        <input type="hidden" name="type" value="{{ $editing?->type }}" data-type-locked @disabled(! $editing?->used)>
                    </div>
                    <div>
                        <label class="label" for="acc-subtype">Subtype <span class="font-normal normal-case text-ink-400">— opsional, dipakai mesin</span></label>
                        <input id="acc-subtype" name="subtype" list="acc-subtypes" maxlength="40" class="field font-mono text-xs" placeholder="opex" value="{{ old('subtype', $editing?->subtype) }}">
                        <datalist id="acc-subtypes">@foreach ($subtypes as $st)<option value="{{ $st }}">@endforeach</datalist>
                    </div>
                    <div>
                        <label class="label" for="acc-normal">Saldo normal</label>
                        <select id="acc-normal" name="normal_balance" class="field">
                            <option value="">Otomatis</option>
                            <option value="debit" @selected(old('normal_balance', $editing?->normal_balance) === 'debit')>Debit</option>
                            <option value="credit" @selected(old('normal_balance', $editing?->normal_balance) === 'credit')>Kredit</option>
                        </select>
                    </div>
                    <label class="flex items-center gap-2 self-end pb-2 text-sm">
                        <input type="checkbox" name="is_active" value="1" @checked(old('_dialog') ? old('is_active') : ($editing?->is_active ?? true))> Aktif
                    </label>
                </div>
                <p class="mt-3 text-xs text-amber-700 {{ $editing?->used ? '' : 'hidden' }}" data-used-note>Akun ini sudah dipakai jurnal — tipe tidak bisa diganti. Kalau salah tipe, buat akun baru lalu nonaktifkan yang ini.</p>

                <div class="mt-5 flex justify-end gap-2 border-t border-ink-100 pt-4">
                    <button type="button" class="btn-ghost" data-dialog-close>Batal</button>
                    <button class="btn-primary">Simpan</button>
                </div>
            </form>
        </dialog>

        {{-- Modal impor --}}
        <dialog id="import-dialog" class="m-auto w-full max-w-lg rounded-[10px] border border-ink-200 bg-white p-0 shadow-[0_1px_2px_rgb(0_0_0/.05),0_8px_24px_rgb(0_0_0/.08)] backdrop:bg-ink-900/30"
                @error('file') data-autopen @enderror>
            <form method="POST" action="{{ route('accounts.import', $client) }}" enctype="multipart/form-data" class="p-5">
                @csrf
                <div class="mb-3 flex items-start justify-between gap-3">
                    <h3 class="font-semibold text-ink-900">Impor COA dari Excel</h3>
                    <button type="button" class="nav-link -mr-2 -mt-1" data-dialog-close aria-label="Tutup"><x-icon name="x"/></button>
                </div>

                @error('file')
                    <ul class="mb-4 space-y-0.5 rounded-md bg-rose-50 px-3 py-2 text-sm text-rose-800">
                        @foreach ($errors->get('file') as $e)<li>{{ $e }}</li>@endforeach
                    </ul>
                @enderror

                <ol class="mb-4 list-decimal space-y-1 pl-5 text-sm text-ink-600">
                    <li>Unduh format lewat <a href="{{ route('accounts.export', $client) }}" class="text-brand-700 underline">Ekspor Excel</a> — kolom: <span class="font-mono text-xs">Kode · Nama akun · Tipe · Subtype · Saldo normal · Aktif</span>.</li>
                    <li>Kode yang sudah ada <b>diperbarui</b>, kode baru <b>ditambahkan</b>. Akun yang tidak ada di file tidak dihapus.</li>
                    <li>Kalau ada satu baris salah, seluruh impor dibatalkan dan baris yang salah ditunjukkan.</li>
                </ol>

                <label class="label" for="import-file">File .xlsx atau .csv (maks. 2 MB)</label>
                <input id="import-file" type="file" name="file" required accept=".xlsx,.csv" class="field file:mr-3 file:rounded file:border-0 file:bg-ink-100 file:px-2 file:py-1 file:text-sm">

                <div class="mt-5 flex justify-end gap-2 border-t border-ink-100 pt-4">
                    <button type="button" class="btn-ghost" data-dialog-close>Batal</button>
                    <button class="btn-primary">Impor</button>
                </div>
            </form>
        </dialog>
    @endif
@endsection
