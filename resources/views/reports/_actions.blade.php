<div class="flex items-end gap-2 print:hidden">
    <button type="button" onclick="window.print()" class="btn-ghost" title="Cetak, atau pilih 'Simpan sebagai PDF' di dialog cetak">
        <x-icon name="printer"/> Cetak / PDF
    </button>
    <a href="{{ route('reports.export', [$client, $jenis]) }}?period={{ $period }}" class="btn-ghost"><x-icon name="download"/> Excel (CSV)</a>
</div>
