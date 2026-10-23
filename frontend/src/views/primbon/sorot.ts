// Penanda kata kunci di hasil pencarian. Hasilnya potongan teks biasa (dirender React sebagai
// teks), jadi isi judul tidak pernah ditafsirkan sebagai HTML. Aturannya sama dengan pencarian
// server: tanpa beda huruf besar, angka harus utuh ("10" tidak menandai "2010").

export interface Potongan { teks: string; cocok: boolean }

const escape = (s: string) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

/** Singkatan yang juga dikenali pencarian server (internal/primbon: singkatan). */
const SINGKATAN: Record<string, string> = {
  pph: 'pajak penghasilan',
  ppn: 'pajak pertambahan nilai',
  ppnbm: 'pajak penjualan atas barang mewah',
  pbb: 'pajak bumi dan bangunan',
  kup: 'ketentuan umum dan tata cara perpajakan',
  bphtb: 'bea perolehan hak atas tanah dan bangunan',
  npwp: 'nomor pokok wajib pajak',
  pkp: 'pengusaha kena pajak',
}

export function pecahSorot(teks: string, q: string): Potongan[] {
  const dasar = q.toLowerCase().split(/\s+/).filter(Boolean)
  const kata = [...new Set([...dasar, ...dasar.flatMap((k) => (SINGKATAN[k] ? [SINGKATAN[k]] : []))])].sort((a, b) => b.length - a.length)
  if (!teks || kata.length === 0) return [{ teks, cocok: false }]
  const pola = kata.map((k) => (/^\d+$/.test(k) ? `(?<!\\d)${k}(?!\\d)` : escape(k))).join('|')
  const out: Potongan[] = []
  let akhir = 0
  for (const m of teks.matchAll(new RegExp(pola, 'gi'))) {
    if (m.index > akhir) out.push({ teks: teks.slice(akhir, m.index), cocok: false })
    out.push({ teks: m[0], cocok: true })
    akhir = m.index + m[0].length
  }
  if (akhir < teks.length) out.push({ teks: teks.slice(akhir), cocok: false })
  return out
}
