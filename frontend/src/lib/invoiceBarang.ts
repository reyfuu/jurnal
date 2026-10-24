import type { InvoiceItem, PilihanBarang } from '@/lib/api'

// Barang per baris invoice (persediaan tahap 2). Di perpetual, invoice yang dikirim / lunas
// bertanggal sejak kartu stok dimulai (atau kartu belum dimulai) wajib memilih barang atau
// menandai Bukan barang (jasa) di tiap baris; aturannya sama dengan server (cekBarangInvoiceLocked).

export const PESAN_BARIS_BARANG = 'pilih barang atau tandai Bukan barang (jasa)'

type InvoiceBagian = { status: string; tanggal: string; items: InvoiceItem[] }

/** Wajib barang untuk invoice ini. mulai = tanggal kartu stok dimulai ('' = belum). */
export function wajibBarangInvoice(p: { perpetual: boolean; status: string; tanggal: string; mulai: string }) {
  return p.perpetual && p.status !== 'draft' && (!p.mulai || p.tanggal >= p.mulai)
}

const samaBaris = (a: InvoiceItem, b: InvoiceItem) =>
  a.deskripsi === b.deskripsi && a.qty === b.qty && (a.satuan ?? '') === (b.satuan ?? '') && a.harga_satuan === b.harga_satuan &&
  (a.stok_id ?? 0) === (b.stok_id ?? 0) && !!a.bukan_barang === !!b.bukan_barang

/** Aturan diperiksa saat invoice baru, atau bila draft dikirim / baris / tanggal berubah. Invoice lama
 *  yang hanya dilunasi tanpa diubah tidak diblokir (sama dengan server). */
export function perluCekBarang(asli: InvoiceBagian | null | undefined, form: InvoiceBagian) {
  if (!asli) return true
  return asli.status === 'draft' || asli.tanggal !== form.tanggal || asli.items.length !== form.items.length ||
    asli.items.some((it, i) => !samaBaris(it, form.items[i]))
}

/** Baris (0-based) yang belum memilih barang maupun jasa. */
export const barisTanpaBarang = (items: InvoiceItem[]) =>
  items.flatMap((it, i) => (!it.stok_id && !it.bukan_barang ? [i] : []))

/** '' bila sah; selain itu pesan galat server untuk baris pertama yang bermasalah. */
export function galatBarangInvoice(items: InvoiceItem[], wajib: boolean): string {
  const dobel = items.findIndex((it) => !!it.stok_id && it.bukan_barang)
  if (dobel >= 0) return `baris ${dobel + 1}: ${PESAN_BARIS_BARANG}, tidak keduanya`
  if (!wajib) return ''
  const kosong = barisTanpaBarang(items)
  return kosong.length ? `baris ${kosong[0] + 1}: ${PESAN_BARIS_BARANG}` : ''
}

/** Patch baris saat barang dipilih: deskripsi, satuan, dan harga jual diisi bila masih kosong. */
export function isiDariBarang(it: InvoiceItem, b: PilihanBarang): Partial<InvoiceItem> {
  return {
    stok_id: b.id, bukan_barang: undefined,
    deskripsi: it.deskripsi.trim() ? it.deskripsi : b.nama,
    satuan: it.satuan?.trim() ? it.satuan : b.satuan,
    harga_satuan: it.harga_satuan ? it.harga_satuan : b.harga_jual,
  }
}
