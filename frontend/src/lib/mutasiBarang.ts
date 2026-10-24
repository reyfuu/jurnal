import type { RincianBarang } from '@/lib/api'
import { barisTerpilih, berbarang, bodyRincian, rincianSah, wajibBarang, type NilaiRincian } from '@/lib/rincian'

// Barang per baris impor mutasi bank. Baris Penjualan / Beli Bahan / Stok yang dibukukan sebagai
// transaksi baru (bukan dilewati, bukan pelunasan / pindah akun bayar) membawa barang atau tanda jasa; di perpetual
// wajib (aturan sama dengan formulir transaksi & server cekBarangLocked).

export type BarisBarangMutasi = {
  pilih: boolean; lunasi?: number; pindah?: number; jenis: string; tanggal: string; nominal: number
  rincian?: RincianBarang[]; bukan_barang?: boolean
}

/** Baris yang akan jadi transaksi baru berjenis berbarang. */
export const butuhBarang = (r: BarisBarangMutasi) => r.pilih && !r.lunasi && !r.pindah && berbarang(r.jenis)

export const nilaiBarisMutasi = (r: BarisBarangMutasi): NilaiRincian => ({ rincian: r.rincian ?? [], bukan_barang: !!r.bukan_barang })

/** "2 barang", "jasa", atau '' bila belum diisi. */
export function ringkasBarang(r: BarisBarangMutasi) {
  if (r.bukan_barang) return 'jasa'
  const n = barisTerpilih(r.rincian ?? []).length
  return n ? `${n} barang` : ''
}

/** '' bila barang baris ini sah (atau tidak perlu); selain itu pesannya. */
export function salahBarang(r: BarisBarangMutasi, p: { perpetual: boolean; mulai: string }) {
  if (!butuhBarang(r)) return ''
  return rincianSah(nilaiBarisMutasi(r), wajibBarang({ ...p, jenis: r.jenis, tanggal: r.tanggal }), r)
}

/** Jumlah baris yang barangnya belum lengkap / belum sah. */
export const kurangBarang = (rows: BarisBarangMutasi[], p: { perpetual: boolean; mulai: string }) =>
  rows.filter((r) => salahBarang(r, p) !== '').length

/** Baris yang tersisa sesudah membukukan sebagian: baris terpilih yang tidak gagal sudah
 *  tersimpan (dilunasi / dipindah akun bayarnya) dan dibuang supaya tidak terbukukan dua kali saat dicoba ulang;
 *  baris gagal dan baris yang tidak dicentang tetap untuk diperbaiki. */
export function sisaSesudahSimpan<T extends { baris: number; pilih: boolean }>(rows: T[], errors: { baris: number }[]): T[] {
  const gagal = new Set(errors.map((e) => e.baris))
  return rows.filter((r) => !r.pilih || gagal.has(r.baris))
}

/** Bagian barang baris untuk dikirim saat membukukan. */
export function barangMutasiBody(r: BarisBarangMutasi): { rincian?: RincianBarang[]; bukan_barang?: boolean } {
  if (!butuhBarang(r)) return { rincian: undefined, bukan_barang: undefined }
  const { rincian, bukan_barang } = bodyRincian(nilaiBarisMutasi(r), r.jenis)
  return { rincian, bukan_barang }
}
