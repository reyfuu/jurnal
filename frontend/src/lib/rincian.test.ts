import { describe, expect, it } from 'vitest'
import { PESAN_WAJIB_BARANG, bodyRincian, nilaiRincian, rincianDariTx, rincianSah, sisaNominal, wajibBarang } from './rincian'

describe('rincian barang transaksi', () => {
  it('sisa nominal = nominal dikurangi nilai barang', () => {
    expect(sisaNominal(100_000, [{ stok_id: 1, qty_stok: 1000, nilai: 60_000 }, { stok_id: 2, qty_stok: 1000, nilai: 30_000 }])).toBe(10_000)
    expect(sisaNominal(100_000, [])).toBe(100_000)
    expect(sisaNominal(50_000, [{ stok_id: 1, qty_stok: 1000 }])).toBe(50_000)
  })

  it('wajib: tanpa barang ditolak dengan pesan server, jasa boleh', () => {
    expect(PESAN_WAJIB_BARANG).toBe('pilih barang & jumlahnya, atau tandai Bukan barang (jasa)')
    expect(rincianSah({ rincian: [], bukan_barang: false }, true)).toBe(PESAN_WAJIB_BARANG)
    expect(rincianSah({ rincian: [{ stok_id: 0, qty_stok: 0 }], bukan_barang: false }, true)).toBe(PESAN_WAJIB_BARANG)
    expect(rincianSah({ rincian: [], bukan_barang: true }, true)).toBe('')
    expect(rincianSah({ rincian: [{ stok_id: 3, qty_stok: 2000 }], bukan_barang: false }, true)).toBe('')
  })

  it('tidak wajib: kosong boleh, tetapi barang tanpa jumlah tetap ditolak', () => {
    expect(rincianSah({ rincian: [], bukan_barang: false }, false)).toBe('')
    expect(rincianSah({ rincian: [{ stok_id: 0, qty_stok: 0 }], bukan_barang: false }, false)).toBe('')
    expect(rincianSah({ rincian: [{ stok_id: 3, qty_stok: 0 }], bukan_barang: false }, false)).toBe(PESAN_WAJIB_BARANG)
  })

  it('pembelian beberapa barang: nilai tiap barang wajib dan totalnya ≤ nominal', () => {
    const dua = [{ stok_id: 1, qty_stok: 1000, nilai: 60_000 }, { stok_id: 2, qty_stok: 1000 }]
    expect(rincianSah({ rincian: dua, bukan_barang: false }, true, { jenis: 'Beli Bahan / Stok', nominal: 100_000 })).toBe('isi nilai tiap barang pada pembelian beberapa barang')
    expect(rincianSah({ rincian: dua, bukan_barang: false }, true, { jenis: 'Penjualan', nominal: 100_000 })).toBe('')
    const lebih = [{ stok_id: 1, qty_stok: 1000, nilai: 80_000 }, { stok_id: 2, qty_stok: 1000, nilai: 40_000 }]
    expect(rincianSah({ rincian: lebih, bukan_barang: false }, true, { jenis: 'Beli Bahan / Stok', nominal: 100_000 })).toMatch(/melebihi nominal/)
  })

  it('wajib hanya perpetual, jenis berbarang, ada daftar barang, dan tanggal ≥ mulai kartu', () => {
    const dasar = { perpetual: true, jenis: 'Penjualan', tanggal: '2026-02-01', mulai: '2026-01-01', adaBarang: true }
    expect(wajibBarang(dasar)).toBe(true)
    expect(wajibBarang({ ...dasar, mulai: '' })).toBe(true) // kartu dimulai otomatis
    expect(wajibBarang({ ...dasar, tanggal: '2025-12-31' })).toBe(false)
    expect(wajibBarang({ ...dasar, perpetual: false })).toBe(false)
    expect(wajibBarang({ ...dasar, jenis: 'Gaji & Upah' })).toBe(false)
    expect(wajibBarang({ ...dasar, adaBarang: false })).toBe(false)
  })

  it('data lama stok_id tunggal jadi satu baris; payload membuang baris kosong', () => {
    expect(rincianDariTx({ stok_id: 3, qty_stok: 2000 })).toEqual({ rincian: [{ stok_id: 3, qty_stok: 2000 }], bukan_barang: false })
    expect(rincianDariTx({ rincian: [{ stok_id: 4, qty_stok: 1000, nilai: 5 }], bukan_barang: false })).toEqual({ rincian: [{ stok_id: 4, qty_stok: 1000, nilai: 5 }], bukan_barang: false })
    expect(rincianDariTx({ bukan_barang: true })).toEqual({ rincian: [], bukan_barang: true })
    expect(bodyRincian({ rincian: [{ stok_id: 3, qty_stok: 2000, nilai: 0 }, { stok_id: 0, qty_stok: 0 }], bukan_barang: false }, 'Penjualan'))
      .toEqual({ rincian: [{ stok_id: 3, qty_stok: 2000 }], bukan_barang: false, stok_id: undefined, qty_stok: undefined })
    expect(bodyRincian({ rincian: [{ stok_id: 3, qty_stok: 2000 }], bukan_barang: true }, 'Penjualan'))
      .toEqual({ rincian: undefined, bukan_barang: true, stok_id: undefined, qty_stok: undefined })
    expect(bodyRincian({ rincian: [{ stok_id: 3, qty_stok: 2000 }], bukan_barang: false }, 'Gaji & Upah'))
      .toEqual({ rincian: undefined, bukan_barang: undefined, stok_id: undefined, qty_stok: undefined })
    expect(nilaiRincian([{ stok_id: 1, qty_stok: 1, nilai: 7 }, { stok_id: 2, qty_stok: 1 }])).toBe(7)
  })
})
