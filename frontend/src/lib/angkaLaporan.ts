/**
 * Cara menulis angka laporan laba rugi (sama dengan PDF, finance.angkaSifat):
 * - beban & HPP (pengurang): selalu hitam, tanpa minus; nilai disimpan negatif seperti di server
 * - akun kontra (potongan, penyusutan, amortisasi): merah dalam kurung
 * - lainnya: minus (rugi) merah dalam kurung
 */
export type Sifat = '' | 'pengurang' | 'kontra'

const rp = (n: number) => n.toLocaleString('id-ID')

export function angkaSifat(v: number, sifat: Sifat = ''): { teks: string; merah: boolean } {
  // beban & HPP selalu hitam; yang justru menambah laba (koreksi) cukup ditandai kurung
  if (sifat === 'pengurang') return v > 0 ? { teks: `(${rp(v)})`, merah: false } : { teks: rp(-v), merah: false }
  if (sifat === 'kontra') return v === 0 ? { teks: '0', merah: false } : { teks: `(${rp(Math.abs(v))})`, merah: true }
  return v < 0 ? { teks: `(${rp(-v)})`, merah: true } : { teks: rp(v), merah: false }
}
