// Halaman & periode aktif disimpan di hash URL (#/buku-besar?periode=2026-08), supaya muat ulang
// (F5) atau login ulang kembali ke halaman yang sama, dan tombol Back browser berpindah halaman.
// Jangkar biasa (#harga di landing) bukan rute.
//
// Halaman boleh menambah sub-alamat & parameternya sendiri (#/primbon-pajak/p/<id>?q=pph), mis.
// supaya detail bisa dibagikan; bacaRute tetap hanya mengembalikan halaman & periode.

export function tulisRute(view: string, periode: string, tambahan: { sub?: string; params?: URLSearchParams } = {}): string {
  const p = new URLSearchParams(tambahan.params)
  p.delete('periode')
  const q = new URLSearchParams(periode ? { periode } : {})
  p.forEach((v, k) => q.append(k, v))
  const s = q.toString()
  return `#/${view}${tambahan.sub ? `/${tambahan.sub}` : ''}${s ? `?${s}` : ''}`
}

export function bacaRute(hash: string): { view: string; periode: string } {
  const r = pecahRute(hash)
  return r ? { view: r.view, periode: r.periode } : { view: '', periode: '' }
}

/** Halaman, periode, sub-alamat, dan parameter lain di hash; null bila bukan rute. */
export function pecahRute(hash: string): { view: string; periode: string; sub: string; params: URLSearchParams } | null {
  const m = /^#\/([a-z0-9-]+)(?:\/([^?#]*))?(?:\?([^#]*))?$/.exec(hash)
  if (!m) return null
  const params = new URLSearchParams(m[3] ?? '')
  const periode = params.get('periode') ?? ''
  params.delete('periode')
  return { view: m[1], periode: /^\d{4}-\d{2}$/.test(periode) ? periode : '', sub: m[2] ?? '', params }
}
