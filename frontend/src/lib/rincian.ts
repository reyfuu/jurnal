import type { RincianBarang } from '@/lib/api'

// Rincian barang transaksi (Penjualan / Beli Bahan / Stok). Di perpetual barang wajib, atau
// transaksinya ditandai Bukan barang (jasa); aturannya sama dengan server (cekBarangLocked).

export const PESAN_WAJIB_BARANG = 'pilih barang & jumlahnya, atau tandai Bukan barang (jasa)'
export const JENIS_BERBARANG = ['Penjualan', 'Beli Bahan / Stok']
export const JENIS_BELI = 'Beli Bahan / Stok'

export type NilaiRincian = { rincian: RincianBarang[]; bukan_barang: boolean }

export const berbarang = (jenis: string) => JENIS_BERBARANG.includes(jenis)

/** Baris yang benar-benar memilih barang (baris kosong di formulir diabaikan). */
export const barisTerpilih = (rincian: RincianBarang[]) => rincian.filter((r) => r.stok_id > 0)

export const nilaiRincian = (rincian: RincianBarang[]) => rincian.reduce((n, r) => n + (r.nilai ?? 0), 0)

/** Bagian nominal yang tidak menjadi nilai barang (mis. ongkir), tetap ke akun bawaan. */
export const sisaNominal = (nominal: number, rincian: RincianBarang[]) => nominal - nilaiRincian(rincian)

/** Barang wajib: perpetual, jenis berbarang, usaha punya daftar barang, dan tanggal tidak sebelum
 *  kartu stok dimulai (kartu belum dimulai = dimulai otomatis oleh transaksi ini). */
export function wajibBarang(p: { perpetual: boolean; jenis: string; tanggal: string; mulai: string; adaBarang: boolean }) {
  return p.perpetual && berbarang(p.jenis) && p.adaBarang && (!p.mulai || p.tanggal >= p.mulai)
}

/** '' bila sah; selain itu pesan galat (sama dengan server). */
export function rincianSah(v: NilaiRincian, wajib: boolean, tx?: { jenis: string; nominal: number }): string {
  if (v.bukan_barang) return ''
  const baris = barisTerpilih(v.rincian)
  if (baris.some((r) => r.qty_stok <= 0)) return PESAN_WAJIB_BARANG
  if (!baris.length) return wajib ? PESAN_WAJIB_BARANG : ''
  if (tx?.jenis === JENIS_BELI) {
    if (baris.length >= 2 && baris.some((r) => !r.nilai)) return 'isi nilai tiap barang pada pembelian beberapa barang'
    if (nilaiRincian(baris) > tx.nominal) return 'jumlah nilai barang melebihi nominal transaksi'
  }
  return ''
}

/** Isian formulir dari transaksi tersimpan; data lama stok_id tunggal jadi satu baris. */
export function rincianDariTx(t: { stok_id?: number; qty_stok?: number; rincian?: RincianBarang[]; bukan_barang?: boolean }): NilaiRincian {
  if (t.rincian?.length) return { rincian: t.rincian.map((r) => ({ ...r })), bukan_barang: false }
  if (t.stok_id) return { rincian: [{ stok_id: t.stok_id, qty_stok: t.qty_stok ?? 0 }], bukan_barang: false }
  return { rincian: [], bukan_barang: !!t.bukan_barang }
}

/** Bagian body transaksi: rincian (tanpa baris kosong) menggantikan stok_id/qty_stok tunggal.
 *  Nilai baris hanya dikirim untuk pembelian (penjualan memakai HPP kartu stok). */
export function bodyRincian(v: NilaiRincian, jenis: string) {
  const kosong = { rincian: undefined, bukan_barang: undefined, stok_id: undefined, qty_stok: undefined }
  if (!berbarang(jenis)) return kosong
  if (v.bukan_barang) return { ...kosong, bukan_barang: true }
  const rincian = barisTerpilih(v.rincian).map((r) => (jenis === JENIS_BELI && r.nilai ? { stok_id: r.stok_id, qty_stok: r.qty_stok, nilai: r.nilai } : { stok_id: r.stok_id, qty_stok: r.qty_stok }))
  return { ...kosong, rincian: rincian.length ? rincian : undefined, bukan_barang: false }
}
