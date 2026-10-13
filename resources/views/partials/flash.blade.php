@if (session('status'))
    {{-- Toast: konfirmasi singkat, hilang sendiri. Error tetap inline di bawah (harus dibaca). --}}
    <div data-toast role="status" aria-live="polite"
         class="fixed top-4 right-4 z-50 flex max-w-sm items-start gap-3 rounded-[10px] border border-ink-200 bg-white px-4 py-3 text-sm text-ink-800 shadow-[0_1px_2px_rgb(0_0_0/.05),0_8px_24px_rgb(0_0_0/.08)] transition-opacity duration-300 print:hidden">
        <span class="mt-1.5 size-2 shrink-0 rounded-full bg-emerald-500"></span>
        <span class="grow">{{ session('status') }}</span>
        <button type="button" data-toast-close class="text-ink-400 hover:text-ink-800 cursor-pointer" aria-label="Tutup"><x-icon name="x" class="size-3.5"/></button>
    </div>
@endif

@if ($errors->any())
    <div class="mb-4 rounded-lg border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-900">
        <p class="font-semibold mb-1">Ada yang perlu dibenahi:</p>
        <ul class="list-disc pl-5 space-y-0.5">
            @foreach ($errors->all() as $error)
                <li>{{ $error }}</li>
            @endforeach
        </ul>
    </div>
@endif
