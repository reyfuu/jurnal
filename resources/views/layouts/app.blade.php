<!DOCTYPE html>
<html lang="id">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>@yield('title', 'Dashboard') · {{ config('app.name') }}</title>
    @vite(['resources/css/app.css', 'resources/js/app.js'])
    @stack('head')
</head>
<body>
@php
    $client = $client ?? null;
    // ponytail: dua query kecil per halaman untuk sidebar. Pindah ke view composer
    // + cache kalau klien sudah ratusan.
    $navClients = \App\Models\Client::active()->orderBy('type')->orderBy('name')->get(['id', 'name', 'slug', 'type']);
    $navPending = $client?->exists
        ? $client->documents()->whereIn('status', [\App\Models\Document::STATUS_UPLOADED, \App\Models\Document::STATUS_EXTRACTED])->count()
        : 0;
    $aiReady = \App\Services\Ai\AiProviderFactory::configured();
    $initials = fn ($name) => \Illuminate\Support\Str::of($name)->explode(' ')->take(2)->map(fn ($w) => mb_substr($w, 0, 1))->implode('');
    $link = fn (bool $active) => $active ? 'nav-link-active' : 'nav-link';
@endphp

<div class="group/shell min-h-screen lg:grid lg:grid-cols-[16rem_1fr] print:block" data-shell>
    {{-- ── Sidebar ─────────────────────────────────────────────────────────── --}}
    <div class="fixed inset-0 z-30 bg-ink-900/30 hidden group-data-open/shell:block lg:hidden!" data-nav-close></div>
    <aside class="print:hidden fixed inset-y-0 left-0 z-40 w-64 -translate-x-full group-data-open/shell:translate-x-0 transition-transform duration-200 ease-snappy
                  lg:sticky lg:top-0 lg:h-screen lg:translate-x-0 flex flex-col border-r border-ink-200 bg-ink-50">
        <div class="flex h-14 items-center justify-between px-4">
            <a href="{{ route('dashboard') }}" class="flex items-center gap-2 font-semibold tracking-tight text-ink-900">
                <span class="grid size-7 place-items-center rounded-md bg-brand-600 text-white" aria-hidden="true">
                    <svg viewBox="0 0 16 16" class="size-3.5" fill="currentColor"><rect x="3" y="2" width="4" height="12" rx="1"/><rect x="9" y="2" width="4" height="12" rx="1" opacity=".55"/></svg>
                </span>
                {{ config('app.name') }}
            </a>
            <button type="button" class="nav-link lg:hidden" data-nav-close aria-label="Tutup menu"><x-icon name="x"/></button>
        </div>

        {{-- Client switcher: konteks klien harus selalu jelas — isolasi klien adalah aturan inti. --}}
        <details class="group/sw relative mx-3">
            <summary class="flex cursor-pointer list-none items-center gap-2.5 rounded-md border border-ink-200 bg-white px-2.5 py-2 text-sm hover:border-ink-300">
                @if ($client?->exists)
                    <span class="grid size-6 place-items-center rounded bg-brand-100 text-[11px] font-semibold text-brand-700">{{ $initials($client->name) }}</span>
                    <span class="min-w-0 grow">
                        <span class="block truncate font-medium text-ink-900">{{ $client->name }}</span>
                        <span class="block text-[11px] text-ink-500">{{ $client->type === \App\Models\Client::TYPE_INTERNAL ? 'Brand sendiri' : 'Klien' }}</span>
                    </span>
                @else
                    <span class="grow text-ink-500">Pilih klien…</span>
                @endif
                <x-icon name="updown" class="size-3.5 text-ink-400"/>
            </summary>
            <div class="absolute inset-x-0 top-full z-50 mt-1 max-h-72 overflow-auto rounded-md border border-ink-200 bg-white p-1 shadow-[0_1px_2px_rgb(0_0_0/.05),0_8px_24px_rgb(0_0_0/.08)]">
                @forelse ($navClients as $c)
                    <a href="{{ route('clients.show', $c) }}"
                       class="flex items-center gap-2.5 rounded px-2 py-1.5 text-sm hover:bg-ink-100 {{ $client?->is($c) ? 'bg-ink-100 font-medium' : '' }}">
                        <span class="grid size-5 place-items-center rounded bg-ink-100 text-[10px] font-semibold text-ink-600">{{ $initials($c->name) }}</span>
                        <span class="truncate">{{ $c->name }}</span>
                    </a>
                @empty
                    <p class="px-2 py-1.5 text-sm text-ink-500">Belum ada klien.</p>
                @endforelse
                @if (auth()->user()->isAdmin())
                    <a href="{{ route('clients.create') }}" class="mt-1 flex items-center gap-2 border-t border-ink-100 px-2 pt-2 pb-1.5 text-sm text-brand-700 hover:underline">
                        <x-icon name="plus" class="size-3.5"/> Klien baru
                    </a>
                @endif
            </div>
        </details>

        <nav class="mt-2 grow overflow-y-auto px-3 pb-4" aria-label="Navigasi utama">
            <a href="{{ route('dashboard') }}" class="{{ $link(request()->routeIs('dashboard')) }}"><x-icon name="dashboard"/> Dashboard</a>
            <a href="{{ route('clients.index') }}" class="{{ $link(request()->routeIs('clients.index', 'clients.create', 'clients.edit')) }}"><x-icon name="users"/> Semua klien</a>

            @if ($client?->exists)
                <p class="nav-heading">Buku {{ $client->name }}</p>
                <a href="{{ route('clients.show', $client) }}" class="{{ $link(request()->routeIs('clients.show')) }}"><x-icon name="home"/> Ringkasan</a>
                <a href="{{ route('documents.index', $client) }}" class="{{ $link(request()->routeIs('documents.*')) }}">
                    <x-icon name="file"/> Dokumen
                    @if ($navPending)<span class="badge-ai ml-auto">{{ $navPending }}</span>@endif
                </a>
                <a href="{{ route('journals.index', $client) }}" class="{{ $link(request()->routeIs('journals.*')) }}"><x-icon name="book"/> Jurnal</a>
                <a href="{{ route('accounts.index', $client) }}" class="{{ $link(request()->routeIs('accounts.*')) }}"><x-icon name="tree"/> Akun (COA)</a>

                <p class="nav-heading">Laporan</p>
                <a href="{{ route('reports.trial-balance', $client) }}" class="{{ $link(request()->routeIs('reports.trial-balance')) }}"><x-icon name="scale"/> Neraca Saldo</a>
                <a href="{{ route('reports.income-statement', $client) }}" class="{{ $link(request()->routeIs('reports.income-statement')) }}"><x-icon name="trend"/> Laba Rugi</a>
                <a href="{{ route('reports.balance-sheet', $client) }}" class="{{ $link(request()->routeIs('reports.balance-sheet')) }}"><x-icon name="landmark"/> Neraca</a>
                <a href="{{ route('reports.cash-flow', $client) }}" class="{{ $link(request()->routeIs('reports.cash-flow')) }}"><x-icon name="flow"/> Arus Kas</a>

                <p class="nav-heading">Pajak</p>
                <a href="{{ route('tax.index', $client) }}" class="{{ $link(request()->routeIs('tax.index')) }}"><x-icon name="receipt"/> Ringkasan pajak</a>
                <a href="{{ route('tax.profile', $client) }}" class="{{ $link(request()->routeIs('tax.profile')) }}">
                    <x-icon name="id"/> Profil pajak
                    @unless ($client->hasTaxProfile())<span class="badge-ai ml-auto">isi</span>@endunless
                </a>
            @endif

            @if (auth()->user()->isAdmin())
                <p class="nav-heading">Admin</p>
                <a href="{{ route('tax.rates') }}" class="{{ $link(request()->routeIs('tax.rates')) }}"><x-icon name="percent"/> Tarif pajak</a>
                <a href="{{ route('audit') }}" class="{{ $link(request()->routeIs('audit')) }}"><x-icon name="history"/> Audit log</a>
            @endif
        </nav>

        <div class="border-t border-ink-200 p-3 text-sm">
            <p class="flex items-center gap-2 px-2.5 pb-2 text-xs text-ink-500" title="{{ $aiReady ? 'Pembacaan dokumen otomatis aktif' : 'Isi API key AI di .env' }}">
                <span class="size-2 rounded-full {{ $aiReady ? 'bg-emerald-500' : 'bg-rose-500' }}"></span>
                {{ $aiReady ? 'AI tersambung' : 'AI belum terpasang' }}
            </p>
            <div class="flex items-center justify-between gap-2 px-2.5">
                <span class="min-w-0">
                    <span class="block truncate font-medium text-ink-800">{{ auth()->user()->name }}</span>
                    <span class="block text-[11px] text-ink-500">{{ auth()->user()->isAdmin() ? 'Admin' : 'Staff' }}</span>
                </span>
                <form method="POST" action="{{ route('logout') }}">@csrf
                    <button class="nav-link" title="Keluar" aria-label="Keluar"><x-icon name="logout"/></button>
                </form>
            </div>
        </div>
    </aside>

    {{-- ── Konten ─────────────────────────────────────────────────────────── --}}
    <div class="min-w-0">
        <header class="print:hidden sticky top-0 z-20 flex h-14 items-center gap-3 border-b border-ink-200 bg-ink-50/85 px-4 backdrop-blur lg:hidden">
            <button type="button" class="nav-link" data-nav-open aria-label="Buka menu"><x-icon name="menu" class="size-5"/></button>
            <span class="truncate font-medium text-ink-900">{{ $client?->exists ? $client->name : config('app.name') }}</span>
        </header>

        <main class="mx-auto max-w-7xl px-4 py-6 lg:px-8 lg:py-8 print:max-w-none print:p-0">
            @include('partials.client-nav', ['client' => $client])
            @include('partials.flash')
            @yield('content')
        </main>

        <footer class="print:hidden mx-auto max-w-7xl px-4 pb-8 text-xs text-ink-400 lg:px-8">
            Hasil baca AI selalu wajib direview sebelum masuk buku.
        </footer>
    </div>
</div>
</body>
</html>
