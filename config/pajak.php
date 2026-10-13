<?php

/*
 * Kode pajak yang dikenal sistem. TARIF tidak di sini — tarif ada di tabel
 * `tax_rates` berversi tanggal (lihat docs/perpajakan/TRD.md §2).
 *
 * subtype_in  = akun aset (PPN Masukan / pajak dibayar dimuka) di COA klien.
 * subtype_out = akun liabilitas (PPN Keluaran / utang PPh).
 * pay_day / file_day = tanggal batas setor/lapor di bulan berikutnya;
 *   null = akhir bulan berikutnya. Semua angka regulasi: BRD §7 V17–V19,
 *   WAJIB diverifikasi konsultan pajak.
 */

return [
    // ponytail: jatuh tempo di config, bukan tabel. Pindah ke tabel kalau admin
    // perlu mengubahnya tanpa deploy.
    'codes' => [
        'ppn' => ['name' => 'PPN', 'kind' => 'vat', 'subtype_in' => 'tax_in', 'subtype_out' => 'vat_out', 'pay_day' => null, 'file_day' => null],
        'ppn_mewah' => ['name' => 'PPN barang mewah', 'kind' => 'vat', 'subtype_in' => 'tax_in', 'subtype_out' => 'vat_out', 'pay_day' => null, 'file_day' => null],
        'pph21' => ['name' => 'PPh 21', 'kind' => 'withholding', 'subtype_in' => 'prepaid_tax', 'subtype_out' => 'wht_payable_pph21', 'pay_day' => 15, 'file_day' => 20],
        'pph22' => ['name' => 'PPh 22', 'kind' => 'withholding', 'subtype_in' => 'prepaid_tax', 'subtype_out' => 'tax_out', 'pay_day' => 15, 'file_day' => 20],
        'pph23' => ['name' => 'PPh 23', 'kind' => 'withholding', 'subtype_in' => 'prepaid_tax', 'subtype_out' => 'wht_payable_pph23', 'pay_day' => 15, 'file_day' => 20],
        'pph4_2' => ['name' => 'PPh 4(2)', 'kind' => 'withholding', 'subtype_in' => 'prepaid_tax', 'subtype_out' => 'wht_payable_pph4_2', 'pay_day' => 15, 'file_day' => 20],
        'final_umkm' => ['name' => 'PPh Final UMKM', 'kind' => 'final', 'subtype_in' => null, 'subtype_out' => 'final_payable', 'pay_day' => 15, 'file_day' => 15],
        'other' => ['name' => 'Pajak lain', 'kind' => 'other', 'subtype_in' => 'tax_in', 'subtype_out' => 'tax_out', 'pay_day' => null, 'file_day' => null],
    ],

    // Selisih pajak AI vs hitungan sistem yang masih dianggap cocok (rupiah per item).
    'toleransi_rupiah' => 1,

    'disclaimer' => 'Rekap ini alat bantu pembukuan, bukan nasihat pajak. Angka & tarif wajib dicek konsultan pajak sebelum setor/lapor.',
];
