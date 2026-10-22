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
