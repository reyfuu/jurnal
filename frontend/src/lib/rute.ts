// Halaman & periode aktif disimpan di hash URL (#/buku-besar?periode=2026-08), supaya muat ulang
// (F5) atau login ulang kembali ke halaman yang sama, dan tombol Back browser berpindah halaman.
// Jangkar biasa (#harga di landing) bukan rute.

export function tulisRute(view: string, periode: string): string {
  return `#/${view}${periode ? `?periode=${periode}` : ''}`
}

export function bacaRute(hash: string): { view: string; periode: string } {
  const m = /^#\/([a-z0-9-]+)(?:\?periode=(\d{4}-\d{2}))?$/.exec(hash)
  return m ? { view: m[1], periode: m[2] ?? '' } : { view: '', periode: '' }
}
