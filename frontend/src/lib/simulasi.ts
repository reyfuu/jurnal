import { formatRp } from './format'

const BULAN = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des']

/** ["2026-07", ...] → "Jul 26, ..." */
export const labelPeriode = (ids: string[]) => ids.map((p) => `${BULAN[+p.slice(5, 7) - 1]} ${p.slice(2, 4)}`).join(', ')

/** Kalimat di bawah angka "Laba per bulan jadi". */
export function kalimatPerubahan(dasar: number, baru: number) {
  const selisih = baru - dasar
  if (selisih === 0) return 'Sama dengan kondisi sekarang. Geser slider atau pilih skenario di kiri.'
  const pct = dasar !== 0 ? Math.round((selisih / Math.abs(dasar)) * 100) : 0
  const arah = selisih > 0 ? 'Naik' : 'Turun'
  const tanda = selisih > 0 ? '+' : '−'
  return `${arah} ${formatRp(Math.abs(selisih))} (${tanda}${Math.abs(pct)}%) dibanding sekarang${baru < 0 ? ' — usaha jadi rugi' : ''}.`
}
