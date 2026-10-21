import { formatRp } from './format'

export const AMBER = 'var(--text-warn)'
export const RED = 'var(--text-danger)'
export const GREEN = 'var(--accent-green)'

/** 1500 → "15%", 175 → "1,75%". */
export const persen = (bps: number) =>
  (bps % 100 === 0 ? String(bps / 100) : (bps / 100).toFixed(2).replace(/0+$/, '').replace(/\.$/, '')).replace('.', ',') + '%'

/** "12,5" / "12.5" → 1250 bps; kosong → 0. */
export const keBps = (s: string) => Math.round(parseFloat((s || '0').replace(',', '.')) * 100) || 0

/** 10 ibukota provinsi pada norma PER-17/PJ/2015; kota lain tidak bisa ditebak dari nama saja. */
const IBUKOTA10 = ['medan', 'palembang', 'jakarta', 'bandung', 'semarang', 'surabaya', 'denpasar', 'manado', 'makassar', 'pontianak']
export const wilayahDariKota = (kota?: string) =>
  kota && IBUKOTA10.some((k) => kota.toLowerCase().includes(k)) ? 'ibukota10' : ''
export const SOROT = 'var(--text-brand)'

// Batas masukan (sama dengan internal/tax/validasi.go): margin laba = laba bersih ÷ omzet harus di
// antara -100% dan 100%. '' = sah; selain itu pesan
// yang meminta pengguna merevisi input sebelum menghitung.
const BATAS_MARGIN = 'Margin laba harus di antara -100% dan 100%'

export function cekLabaOmzet(laba: number, omzet: number) {
  if (laba <= omzet && laba >= -omzet) return ''
  const margin = omzet > 0 ? ` (margin ${persen(Math.round((laba * 10_000) / omzet))})` : ''
  return laba > omzet
    ? `Laba bersih ${formatRp(laba)} lebih besar dari omzet ${formatRp(omzet)}${margin}. ${BATAS_MARGIN}, jadi periksa lagi angka laba atau omzet.`
    : `Kerugian ${formatRp(-laba)} lebih besar dari omzet ${formatRp(omzet)}${margin}. ${BATAS_MARGIN}, jadi periksa lagi angka laba atau omzet.`
}

/** Margin dalam teks persen ("12,5", "-20"); kosong atau "-" saat mengetik dianggap sah. */
export const cekMargin = (teks: string) => (Math.abs(keBps(teks)) > 10_000 ? `${BATAS_MARGIN}.` : '')

