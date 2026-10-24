import { describe, expect, it } from 'vitest'
import { barangMutasiBody, butuhBarang, kurangBarang, nilaiBarisMutasi, ringkasBarang } from './mutasiBarang'

const jual = { pilih: true, jenis: 'Penjualan', tanggal: '2026-02-01', nominal: 100_000 }

describe('barang per baris mutasi', () => {
  it('ringkasan: jumlah barang atau jasa', () => {
    expect(ringkasBarang({ ...jual, rincian: [{ stok_id: 1, qty_stok: 1000 }, { stok_id: 2, qty_stok: 2000 }] })).toBe('2 barang')
    expect(ringkasBarang({ ...jual, rincian: [{ stok_id: 1, qty_stok: 1000 }, { stok_id: 0, qty_stok: 0 }] })).toBe('1 barang')
    expect(ringkasBarang({ ...jual, bukan_barang: true })).toBe('jasa')
    expect(ringkasBarang(jual)).toBe('')
  })

  it('hanya baris berbarang yang dibukukan sebagai transaksi baru yang butuh barang', () => {
    expect(butuhBarang(jual)).toBe(true)
    expect(butuhBarang({ ...jual, jenis: 'Beli Bahan / Stok' })).toBe(true)
    expect(butuhBarang({ ...jual, pilih: false })).toBe(false) // lewati
    expect(butuhBarang({ ...jual, lunasi: 5 })).toBe(false) // pelunasan
    expect(butuhBarang({ ...jual, jenis: 'Pendapatan Lain' })).toBe(false)
  })

  it('jumlah baris yang belum lengkap (perpetual, sejak kartu dimulai)', () => {
    const rows = [
      jual,
      { ...jual, bukan_barang: true },
      { ...jual, rincian: [{ stok_id: 1, qty_stok: 1000 }] },
      { ...jual, rincian: [{ stok_id: 1, qty_stok: 0 }] }, // jumlah 0 selalu ditolak
      { ...jual, tanggal: '2025-12-31' }, // sebelum kartu dimulai: tercakup stok awal
      { ...jual, pilih: false },
      { ...jual, jenis: 'Biaya Lain' },
    ]
    expect(kurangBarang(rows, { perpetual: true, mulai: '2026-01-01' })).toBe(2)
    expect(kurangBarang(rows, { perpetual: true, mulai: '' })).toBe(3) // kartu belum dimulai: semua wajib
    expect(kurangBarang(rows, { perpetual: false, mulai: '' })).toBe(1) // periodik: opsional
  })

  it('body commit: barang dirapikan, dibuang untuk baris bukan transaksi baru berbarang', () => {
    expect(barangMutasiBody({ ...jual, rincian: [{ stok_id: 1, qty_stok: 1000, nilai: 5 }, { stok_id: 0, qty_stok: 0 }] }))
      .toEqual({ rincian: [{ stok_id: 1, qty_stok: 1000 }], bukan_barang: false })
    expect(barangMutasiBody({ ...jual, jenis: 'Beli Bahan / Stok', rincian: [{ stok_id: 1, qty_stok: 1000, nilai: 40_000 }] }))
      .toEqual({ rincian: [{ stok_id: 1, qty_stok: 1000, nilai: 40_000 }], bukan_barang: false })
    expect(barangMutasiBody({ ...jual, bukan_barang: true })).toEqual({ rincian: undefined, bukan_barang: true })
    expect(barangMutasiBody({ ...jual, lunasi: 3, rincian: [{ stok_id: 1, qty_stok: 1000 }] })).toEqual({ rincian: undefined, bukan_barang: undefined })
    expect(barangMutasiBody({ ...jual, jenis: 'Biaya Lain', bukan_barang: true })).toEqual({ rincian: undefined, bukan_barang: undefined })
  })

  it('nilai pemilih dari baris', () => {
    expect(nilaiBarisMutasi(jual)).toEqual({ rincian: [], bukan_barang: false })
    expect(nilaiBarisMutasi({ ...jual, bukan_barang: true, rincian: [{ stok_id: 1, qty_stok: 1 }] })).toEqual({ rincian: [{ stok_id: 1, qty_stok: 1 }], bukan_barang: true })
  })
})
