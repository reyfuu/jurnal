// Filter dan ringkasan daftar transaksi (dihitung di klien, seperti demo).
import type { JenisInfo, Tx } from './api'

const BULAN = ['Januari', 'Februari', 'Maret', 'April', 'Mei', 'Juni', 'Juli', 'Agustus', 'September', 'Oktober', 'November', 'Desember']
const BULAN_SINGKAT = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des']

export const labelBulan = (ym: string) => `${BULAN[+ym.slice(5, 7) - 1]} ${ym.slice(0, 4)}`
export const formatTanggal = (d: string) => `${+d.slice(8, 10)} ${BULAN_SINGKAT[+d.slice(5, 7) - 1]} ${d.slice(0, 4)}`

export const isMasuk = (pos: string) => ['omzet', 'lain', 'modal', 'pinjaman'].includes(pos)

export function filterTx(
  txs: Tx[],
  f: { bulan?: string; jenis?: string; cari?: string; arah?: string; channel?: string },
  jenisList?: JenisInfo[],
) {
  const q = (f.cari ?? '').trim().toLowerCase()
  const pos = jenisList ? Object.fromEntries(jenisList.map((j) => [j.nama, j.pos])) : {}
  return txs
    .filter((t) => {
      if (f.bulan && !t.tanggal.startsWith(f.bulan)) return false
      if (f.jenis && t.jenis !== f.jenis) return false
      if (f.channel && t.channel !== f.channel) return false
      if (f.arah && f.arah !== 'semua') {
        const p = pos[t.jenis] ?? ''
        const isM = isMasuk(p)
        if (f.arah === 'masuk' && !isM) return false
        if (f.arah === 'keluar' && isM) return false
      }
      if (q && ![t.keterangan, t.item, t.pihak, t.channel, t.jenis].some((x) => (x || '').toLowerCase().includes(q))) return false
      return true
    })
    .sort((a, b) => b.tanggal.localeCompare(a.tanggal) || b.id - a.id)
}

export function summarize(txs: Tx[], jenis: JenisInfo[]) {
  const pos = Object.fromEntries(jenis.map((j) => [j.nama, j.pos]))
  const sum = (ok: (p: string) => boolean) => txs.filter((t) => ok(pos[t.jenis])).reduce((s, t) => s + t.nominal, 0)
  return {
    count: txs.length,
    masuk: sum(isMasuk),
    omzet: sum((p) => p === 'omzet'),
    keluar: sum((p) => !isMasuk(p) && p !== 'pindah'), // pindah dana antar rekening bukan pengeluaran
    biayaUsaha: sum((p) => ['hpp', 'biaya', 'potongan'].includes(p)),
  }
}

// tambahan: bulan yang harus ada di pilihan walau belum/tidak ada transaksinya (bulan berjalan, pilihan tersimpan).
export const monthsOf = (txs: Tx[], ...tambahan: string[]) =>
  [...new Set([...txs.map((t) => t.tanggal.slice(0, 7)), ...tambahan])].sort().reverse()

/** Bulan YYYY-MM menurut WIB (toISOString memakai UTC, sehingga awal bulan jatuh ke bulan lalu). */
export const bulanWIB = (d = new Date()) => d.toLocaleDateString('sv-SE', { timeZone: 'Asia/Jakarta' }).slice(0, 7)

/** Filter bulan saat daftar dibuka: pilihan tersimpan ('' = semua bulan) atau bulan berjalan. */
export const bulanAwal = (tersimpan: string | null, sekarang: string) => tersimpan ?? sekarang
export const channelsOf = (txs: Tx[]) => [...new Set(txs.map((t) => t.channel).filter(Boolean))].sort()
