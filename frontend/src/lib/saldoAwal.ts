// Urutan baris saldo awal: kelompok kategori (harta → kewajiban → modal, mengikuti urutan daftar
// kategori dari server), lalu kode akun. Baris tanpa kode di akhir kategorinya; baris yang belum
// berkategori paling bawah. Stabil: baris setara tidak bertukar tempat.
export function urutSaldo<T extends { kode?: string; kategori: string }>(rows: T[], urutKategori: string[]): T[] {
  const idx = (k: string) => { const i = urutKategori.indexOf(k); return i < 0 ? urutKategori.length : i }
  return rows
    .map((r, i) => ({ r, i }))
    .sort((a, b) => idx(a.r.kategori) - idx(b.r.kategori)
      || Number(!a.r.kode) - Number(!b.r.kode)
      || (a.r.kode ?? '').localeCompare(b.r.kode ?? '', 'id', { numeric: true })
      || a.i - b.i)
    .map((x) => x.r)
}

/**
 * Sisipkan baris aset tetap ke antara baris harta menurut kodenya (mis. 1503 sesudah 14xx, sebelum
 * 17xx). Baris harta lain tidak diurutkan ulang, jadi baris yang sedang diketik tidak melompat.
 */
export function sisipkanUrutKode<H extends { kode?: string }, A extends { kode?: string }>(harta: H[], aset: A[]): (H | A)[] {
  const out: (H | A)[] = [...harta]
  for (const a of aset) {
    const i = out.findIndex((r) => !!r.kode && !!a.kode && r.kode.localeCompare(a.kode, undefined, { numeric: true }) > 0)
    out.splice(i < 0 ? out.length : i, 0, a)
  }
  return out
}

/**
 * Baris persediaan form Saldo Awal dikunci (diisi per akun dari daftar barang) bila ada nilai
 * persediaan per akun dan isian belum memuat persediaan bernilai. Nilai yang sudah diketik /
 * disimpan pengguna tidak pernah ditimpa.
 */
export function persediaanTerkunci(persediaanAkun: { nominal: number }[] | undefined, rows: { kategori: string; nominal: number }[]): boolean {
  return !!persediaanAkun?.length && !rows.some((r) => r.kategori === 'persediaan' && !!r.nominal)
}
