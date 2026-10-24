import { describe, expect, it } from 'vitest'
import { barisDariNota, rekeningBawaan } from './CatatJurnal'
import type { OpsiCatat, PilihanBarang, SaranNota } from '@/lib/api'
import { barisBerbarang, cocokkanNota, rincianDariNota } from '@/lib/catatBarang'

const o: OpsiCatat = { akun: [], bawaan: { kas: '1101', bank: '1102', piutang: '1201', utang: '2101', 'Biaya Lain': '6190', Penjualan: '4101' } }
const nota = (x: Partial<SaranNota>): SaranNota => ({ jenis: 'Biaya Lain', nominal: 50_000, lunas: true, ...x } as SaranNota)

describe('rekening bawaan Bank', () => {
  it('struk lunas uang keluar & uang masuk: Bank, bukan Kas', () => {
    expect(barisDariNota(nota({}), o).find((x) => x.kredit)?.akun).toBe('1102')
    expect(barisDariNota(nota({ jenis: 'Penjualan' }), o).find((x) => x.debit)?.akun).toBe('1102')
  })
  it('tanpa akun bank: Kas; belum lunas: Utang', () => {
    expect(rekeningBawaan({ ...o, bawaan: { ...o.bawaan, bank: '' } })).toBe('1101')
    expect(barisDariNota(nota({ lunas: false }), o).find((x) => x.kredit)?.akun).toBe('2101')
  })
})

// Persediaan per akun tahap 2: barang dari foto nota & baris berbarang di Catat cepat.
const pb = (id: number, nama: string): PilihanBarang => ({ id, nama, sku: '', kategori: '', satuan: 'pcs', jumlah_mili: 0, harga_jual: 0 })
const daftar = [pb(1, 'Kain Katun'), pb(2, 'Benang')]

describe('cocokkanNota', () => {
  it('nama sama beda huruf besar & spasi → cocok', () => {
    expect(cocokkanNota({ nama: '  kain   KATUN ' }, daftar)?.id).toBe(1)
  })
  it('tidak ada di daftar / hanya mirip → kosong', () => {
    expect(cocokkanNota({ nama: 'Kancing' }, daftar)).toBeUndefined()
    expect(cocokkanNota({ nama: 'Kain' }, daftar)).toBeUndefined()
    expect(cocokkanNota({ nama: '' }, daftar)).toBeUndefined()
  })
})

describe('rincianDariNota', () => {
  const item = [{ nama: 'kain katun', qty: '2,5', harga: 30_000 }, { nama: 'Kancing', qty: '10', harga: 5_000 }, { nama: 'BENANG', qty: '3 pcs' }]
  it('pembelian: yang cocok jadi rincian (qty & nilai), yang tidak cocok dikembalikan', () => {
    expect(rincianDariNota(item, daftar, 'Beli Bahan / Stok')).toEqual({
      rincian: [{ stok_id: 1, qty_stok: 2500, nilai: 30_000 }, { stok_id: 2, qty_stok: 3000 }],
      tidakCocok: ['Kancing'],
    })
  })
  it('penjualan: tanpa nilai baris', () => {
    expect(rincianDariNota(item, daftar, 'Penjualan').rincian).toEqual([{ stok_id: 1, qty_stok: 2500 }, { stok_id: 2, qty_stok: 3000 }])
  })
})

describe('barisBerbarang', () => {
  const akun = new Map([
    ['1101', { lawan: 'rekening' }], ['1301', { beli_barang: true }], ['4101', { jual_barang: true }], ['6190', {}],
  ])
  it('persediaan di debit = beli, pendapatan di kredit = jual, lainnya bukan', () => {
    expect(barisBerbarang([{ akun: '1301', debit: 40_000, kredit: 0 }, { akun: '6190', debit: 5_000, kredit: 0 }, { akun: '1101', debit: 0, kredit: 45_000 }], akun))
      .toEqual([{ jenis: 'Beli Bahan / Stok', nominal: 40_000 }])
    expect(barisBerbarang([{ akun: '1101', debit: 100_000, kredit: 0 }, { akun: '4101', debit: 0, kredit: 100_000 }], akun))
      .toEqual([{ jenis: 'Penjualan', nominal: 100_000 }])
    expect(barisBerbarang([{ akun: '4101', debit: 100_000, kredit: 0 }], akun)).toEqual([])
  })
})

describe('barisDariNota satu baris barang', () => {
  it('nota berbarang: satu baris pokok supaya rinciannya menempel ke satu transaksi', () => {
    const n = nota({ jenis: 'Penjualan', nominal: 50_000, item: [{ nama: 'A', harga: 20_000 }, { nama: 'B', harga: 30_000 }] })
    expect(barisDariNota(n, o).filter((x) => x.kredit)).toHaveLength(2)
    expect(barisDariNota(n, o, true).filter((x) => x.kredit).map((x) => x.kredit)).toEqual([50_000])
  })
})
