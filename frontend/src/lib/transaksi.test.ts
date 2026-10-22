import { describe, expect, it } from 'vitest'
import type { JenisInfo, Tx } from './api'
import { bulanAwal, bulanWIB, filterTx, labelBulan, monthsOf, summarize } from './transaksi'

const JENIS: JenisInfo[] = [
  { nama: 'Penjualan', arah: 'masuk', pos: 'omzet' },
  { nama: 'Setor Modal', arah: 'masuk', pos: 'modal' },
  { nama: 'Beli Bahan / Stok', arah: 'keluar', pos: 'hpp' },
  { nama: 'Gaji & Upah', arah: 'keluar', pos: 'biaya' },
  { nama: 'Beli Alat / Aset', arah: 'keluar', pos: 'aset' },
]

const t = (id: number, tanggal: string, jenis: string, nominal: number, extra: Partial<Tx> = {}): Tx => ({
  id, tanggal, jenis, nominal, keterangan: '', item: '', qty: 0, channel: '', pihak: '', lunas: true, tanggal_lunas: '', ...extra,
})

const txs = [
  t(1, '2026-07-31', 'Penjualan', 1_000, { channel: 'GoFood' }),
  t(2, '2026-08-01', 'Gaji & Upah', 300, { keterangan: 'Gaji barista' }),
  t(3, '2026-08-01', 'Penjualan', 2_000, { pihak: 'PT Nusantara Kreatif' }),
  t(4, '2026-08-05', 'Setor Modal', 5_000),
  t(5, '2026-08-06', 'Beli Bahan / Stok', 400, { item: 'Susu' }),
  t(6, '2026-08-07', 'Beli Alat / Aset', 900),
]

describe('filterTx', () => {
  it('urut terbaru dulu, ID besar dulu di tanggal yang sama', () => {
    expect(filterTx(txs, {}).map((x) => x.id)).toEqual([6, 5, 4, 3, 2, 1])
  })
  it('filter bulan + jenis', () => {
    expect(filterTx(txs, { bulan: '2026-08', jenis: 'Penjualan' }).map((x) => x.id)).toEqual([3])
  })
  it('filter arah masuk/keluar', () => {
    expect(filterTx(txs, { arah: 'masuk' }, JENIS).map((x) => x.id)).toEqual([4, 3, 1])
    expect(filterTx(txs, { arah: 'keluar' }, JENIS).map((x) => x.id)).toEqual([6, 5, 2])
  })
  it('filter channel', () => {
    expect(filterTx(txs, { channel: 'GoFood' }).map((x) => x.id)).toEqual([1])
  })
  it('cari di keterangan, item, pihak, channel, dan jenis (tanpa beda huruf besar)', () => {
    expect(filterTx(txs, { cari: 'nusantara' }).map((x) => x.id)).toEqual([3])
    expect(filterTx(txs, { cari: 'gofood' }).map((x) => x.id)).toEqual([1])
    expect(filterTx(txs, { cari: 'SUSU' }).map((x) => x.id)).toEqual([5])
    expect(filterTx(txs, { cari: 'modal' }).map((x) => x.id)).toEqual([4])
  })
})

describe('summarize', () => {
  it('memisahkan penjualan dari uang masuk lain, dan biaya usaha dari keluar lain', () => {
    expect(summarize(txs, JENIS)).toEqual({ count: 6, masuk: 8_000, omzet: 3_000, keluar: 1_600, biayaUsaha: 700 })
  })
})

describe('monthsOf', () => {
  it('bulan unik, terbaru dulu, dengan label Indonesia', () => {
    expect(monthsOf(txs)).toEqual(['2026-08', '2026-07'])
    expect(labelBulan('2026-08')).toBe('Agustus 2026')
  })
})

describe('bulan awal daftar transaksi (checklist No 26)', () => {
  it('bulan WIB, bukan UTC: 1 Nov 00.30 WIB masih 31 Okt UTC', () => {
    expect(bulanWIB(new Date('2026-10-31T17:30:00Z'))).toBe('2026-11')
  })
  it('bawaan bulan berjalan; pilihan tersimpan menang, termasuk "semua bulan" (string kosong)', () => {
    expect(bulanAwal(null, '2026-10')).toBe('2026-10')
    expect(bulanAwal('2026-08', '2026-10')).toBe('2026-08')
    expect(bulanAwal('', '2026-10')).toBe('')
  })
  it('daftar bulan memuat bulan berjalan walau belum ada transaksinya', () => {
    expect(monthsOf([t(1, '2026-08-01', 'Penjualan', 1)], '2026-10')).toEqual(['2026-10', '2026-08'])
    expect(monthsOf([t(1, '2026-10-01', 'Penjualan', 1)], '2026-10')).toEqual(['2026-10'])
  })
})
