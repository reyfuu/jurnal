import type { PilihanBarang, RincianBarang } from '@/lib/api'
import { parseQty } from '@/lib/qty'
import { JENIS_BELI } from '@/lib/rincian'

// Barang di Catat cepat (persediaan per akun tahap 2): baris jurnal yang menjadi Penjualan
// (akun pendapatan usaha di kredit) atau Beli Bahan / Stok (akun persediaan/HPP di debit) membawa
// rincian barang. Server menempelkannya ke satu-satunya transaksi berbarang catatan itu.

export const PESAN_PECAH_CATAT = 'catatan ini berisi lebih dari satu penjualan/pembelian barang; pecah jadi beberapa catatan'

type AkunTanda = { lawan?: string; beli_barang?: boolean; jual_barang?: boolean }
type BarisJurnal = { akun: string; debit: number; kredit: number }

/** Baris jurnal yang membawa barang, dengan jenis transaksinya dan nominal baris itu. */
export function barisBerbarang(baris: BarisJurnal[], akun: Map<string, AkunTanda>): { jenis: string; nominal: number }[] {
  const out: { jenis: string; nominal: number }[] = []
  for (const b of baris) {
    const a = akun.get(b.akun)
    if (!a || a.lawan) continue
    if (b.debit > 0 && a.beli_barang) out.push({ jenis: JENIS_BELI, nominal: b.debit })
    else if (b.kredit > 0 && a.jual_barang) out.push({ jenis: 'Penjualan', nominal: b.kredit })
  }
  return out
}

const rapikan = (s: string) => s.trim().replace(/\s+/g, ' ').toLowerCase()

/** Barang di daftar yang namanya sama persis dengan nama barang di nota (tanpa beda huruf besar
 *  dan spasi berlebih); undefined bila tidak ada. */
export function cocokkanNota(item: { nama: string }, barang: PilihanBarang[]): PilihanBarang | undefined {
  const nama = rapikan(item.nama ?? '')
  return nama ? barang.find((b) => rapikan(b.nama) === nama) : undefined
}

/** Rincian dari barang terbaca di nota: yang cocok dengan daftar barang (jumlah dari teks qty,
 *  harga jadi nilai baris untuk pembelian); yang tidak cocok dikembalikan namanya. */
export function rincianDariNota(item: { nama: string; qty?: string; harga?: number }[], barang: PilihanBarang[], jenis: string) {
  const rincian: RincianBarang[] = []
  const tidakCocok: string[] = []
  for (const it of item) {
    const b = cocokkanNota(it, barang)
    if (!b) {
      if (it.nama?.trim()) tidakCocok.push(it.nama.trim())
      continue
    }
    const r: RincianBarang = { stok_id: b.id, qty_stok: parseQty(it.qty ?? '') }
    if (jenis === JENIS_BELI && (it.harga ?? 0) > 0) r.nilai = it.harga
    rincian.push(r)
  }
  return { rincian, tidakCocok }
}
