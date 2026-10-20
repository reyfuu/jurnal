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
