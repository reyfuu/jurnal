{{-- Judul konteks klien. Menu per klien ada di sidebar (layouts/app). --}}
@if ($client?->exists)
    <div class="mb-6 flex flex-wrap items-end justify-between gap-3 border-b border-ink-200 pb-4">
        <div>
            <p class="label mb-0.5">{{ $client->type === \App\Models\Client::TYPE_INTERNAL ? 'Brand sendiri' : 'Klien' }}</p>
            <div class="flex items-center gap-2">
                <h1 class="text-xl font-semibold tracking-tight text-ink-900">{{ $client->name }}</h1>
                @unless ($client->is_active)<span class="badge-bad">Nonaktif</span>@endunless
                <span class="badge-mute">{{ $client->businessTypeLabel() }}</span>
                @if ($client->hasTaxProfile())
                    <span class="badge-mute">{{ $client->isPkpOn() ? 'PKP' : 'Non-PKP' }} · {{ $client->isFinalUmkm() ? 'Final 0,5%' : 'PPh normal' }}</span>
                @endif
            </div>
            @if ($client->legal_name)
                <p class="mt-0.5 text-xs text-ink-500">{{ $client->legal_name }}</p>
            @endif
        </div>
        <a href="{{ route('documents.create', $client) }}" class="btn-primary print:hidden"><x-icon name="plus"/> Dokumen baru</a>
    </div>
@endif
