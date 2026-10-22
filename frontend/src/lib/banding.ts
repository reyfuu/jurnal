import { MAKS_BANDING } from './api'

// Pilihan cepat bulan di tab Banding. ada = id periode (YYYY-MM) yang punya data.

/** Tahun berjalan: Januari sampai bulan aktif di tahun yang sama. */
export const tahunBerjalan = (ada: string[], period: string) =>
  ada.filter((id) => id.slice(0, 4) === period.slice(0, 4) && id <= period).sort()

/** Bulan dari–sampai (urutan bebas); lebih dari 12 bulan → 12 bulan terakhir. */
export function bulanRentang(ada: string[], dari: string, sampai: string) {
  const [a, b] = dari <= sampai ? [dari, sampai] : [sampai, dari]
  return ada.filter((id) => id >= a && id <= b).sort().slice(-MAKS_BANDING)
}
