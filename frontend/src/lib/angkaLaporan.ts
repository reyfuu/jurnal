/**
 * Cara menulis angka laporan laba rugi (sama dengan PDF, finance.angkaSifat):
 * - beban, HPP, penyusutan & amortisasi (pengurang): selalu hitam, tanpa minus; nilai negatif seperti di server
 * - akun kontra (potongan, akumulasi penyusutan): hitam dalam kurung
 * - lainnya: minus (rugi) merah dalam kurung
 */
export type Sifat = '' | 'pengurang' | 'kontra'

const rp = (n: number) => n.toLocaleString('id-ID')

export function angkaSifat(v: number, sifat: Sifat = ''): { teks: string; merah: boolean } {
  // beban & HPP selalu hitam; yang justru menambah laba (koreksi) cukup ditandai kurung
  if (sifat === 'pengurang') return v > 0 ? { teks: `(${rp(v)})`, merah: false } : { teks: rp(-v), merah: false }
  if (sifat === 'kontra') return v === 0 ? { teks: '0', merah: false } : { teks: `(${rp(Math.abs(v))})`, merah: false }
  return v < 0 ? { teks: `(${rp(-v)})`, merah: true } : { teks: rp(v), merah: false }
}

/** Satu baris laporan sebelum dirapikan (sama dengan finance.BarisLaporan di server). */
export interface BarisSpec {
  kunci: string
  label: string
  nilai: number
  gaya: 'judul' | 'baris' | 'sub' | 'total'
  sifat?: Sifat
  /** diberi huruf A, B, C … */
  berhuruf?: boolean
  /** rumus baris jumlah: kunci baris (awalan "-" = dikurangi, "*" = teks apa adanya) */
  suku?: string[]
  /** tetap tampil walau 0 */
  wajib?: boolean
  huruf?: string
  rumus?: string
}

/**
 * rapikanBaris (= finance.rapikan): akun bernilai 0 tidak ditampilkan, judul yang jadi kosong ikut
 * hilang, lalu huruf & rumus disusun dari baris yang tersisa.
 */
export function rapikanBaris(rows: BarisSpec[]): BarisSpec[] {
  const isi = rows.filter((r) => !((r.gaya === 'baris' || r.gaya === 'sub') && r.nilai === 0 && !r.wajib))
  const out = isi.filter((r, i) => r.gaya !== 'judul' || (i + 1 < isi.length && (isi[i + 1].gaya === 'baris' || isi[i + 1].gaya === 'sub')))
  const huruf = new Map<string, string>()
  let h = 65
  const hasil = out.map((r) => {
    if (!r.berhuruf) return { ...r }
    const x = String.fromCharCode(h++)
    huruf.set(r.kunci, x)
    return { ...r, huruf: x }
  })
  for (const r of hasil) {
    if (!r.suku?.length) continue
    const bagian: string[] = []
    for (const s of r.suku) {
      const kurang = s.startsWith('-')
      const k = kurang ? s.slice(1) : s
      const t = k.startsWith('*') ? k.slice(1) : huruf.get(k)
      if (!t) continue
      bagian.push(bagian.length ? `${kurang ? '-' : '+'} ${t}` : t)
    }
    if (bagian.length) r.rumus = `${r.huruf} = ${bagian.join(' ')}`
  }
  return hasil
}
