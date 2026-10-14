{{-- Bubble "Tanya pajak" — hanya di halaman satu klien; jawaban AI berbekal data klien ini saja. --}}
<div class="print:hidden fixed bottom-4 right-4 z-30 flex flex-col items-end gap-3 [body:has([data-seam])_&]:bottom-24"
     data-chat data-chat-url="{{ route('tax.chat', $client) }}" data-chat-key="tanya-pajak-{{ $client->id }}">
    <section id="tax-chat-panel" class="hidden w-[min(24rem,calc(100vw-2rem))] flex-col overflow-hidden rounded-[10px] border border-ink-200 bg-white shadow-[0_1px_2px_rgb(0_0_0/.05),0_8px_24px_rgb(0_0_0/.08)] data-open:flex"
             data-chat-panel aria-label="Tanya pajak">
        <header class="flex items-start justify-between gap-3 border-b border-ink-200 px-4 py-3">
            <div class="min-w-0">
                <p class="font-semibold text-ink-900">Tanya pajak</p>
                <p class="truncate text-xs text-ink-500">{{ $client->name }} · {{ $aiReady ? 'dijawab AI dari data perusahaan ini' : 'AI belum terpasang' }}</p>
            </div>
            <button type="button" class="nav-link -mr-2" data-chat-toggle aria-label="Tutup"><x-icon name="x"/></button>
        </header>

        <div class="max-h-[min(26rem,60vh)] min-h-40 space-y-3 overflow-y-auto px-4 py-3 text-sm" data-chat-log aria-live="polite">
            <p class="text-ink-600">Halo! Tanyakan soal pajak {{ $client->name }} — jatuh tempo, PPN, PPh, atau cara menghitungnya.</p>
            <div class="flex flex-wrap gap-1.5" data-chat-suggest>
                @foreach (['Pajak apa saja yang harus disetor bulan ini?', 'Kapan batas setor & lapor terdekat?', 'Jelaskan PPh Final 0,5% untuk usaha saya'] as $q)
                    <button type="button" class="badge cursor-pointer bg-white text-left text-ink-700 ring-1 ring-ink-200 hover:bg-ink-100" data-chat-ask>{{ $q }}</button>
                @endforeach
            </div>
        </div>

        <form class="border-t border-ink-200 p-3" data-chat-form>
            @csrf
            <div class="flex items-end gap-2">
                <label for="tax-chat-q" class="sr-only">Pertanyaan</label>
                <textarea id="tax-chat-q" name="question" rows="1" maxlength="1000" required placeholder="Tulis pertanyaan…"
                          class="field max-h-32 min-h-10 resize-none py-2" @disabled(! $aiReady)></textarea>
                <button class="btn-primary h-10 px-3" aria-label="Kirim" @disabled(! $aiReady)><x-icon name="send"/></button>
            </div>
            <p class="mt-2 text-[11px] leading-snug text-ink-400">{{ config('pajak.disclaimer') }}</p>
        </form>
    </section>

    <button type="button" class="grid size-12 place-items-center rounded-full bg-brand-600 text-white shadow-[0_1px_2px_rgb(0_0_0/.1),0_8px_24px_rgb(0_0_0/.15)] transition-transform duration-150 ease-snappy hover:scale-105 hover:bg-brand-700 active:scale-95 cursor-pointer"
            data-chat-toggle aria-controls="tax-chat-panel" aria-expanded="false" aria-label="Tanya pajak">
        <x-icon name="chat" class="size-5"/>
    </button>
</div>
