import { describe, expect, it } from 'vitest'
import { akunDariKategori, akunIsianBarang } from './api'

const pilihan = [
  { kode: '1301', nama: 'Persediaan Bahan Baku' },
  { kode: '1302', nama: 'Persediaan Barang Setengah Jadi' },
  { kode: '1303', nama: 'Persediaan Barang Jadi' },
  { kode: '1304', nama: 'Persediaan Bahan Penolong' },
]

describe('akunIsianBarang', () => {
  it('kategori Produk Jadi tanpa akun -> Barang Jadi, bukan akun pertama', () => {
    expect(akunIsianBarang(pilihan, { kategori: 'Produk Jadi' })).toBe('1303')
    expect(akunIsianBarang(pilihan, { kategori: 'Barang Jadi' })).toBe('1303')
  })
  it('setengah jadi didahulukan dari jadi', () => {
    expect(akunDariKategori('Setengah Jadi', pilihan)).toBe('1302')
  })
  it('akun eksplisit menang', () => {
    expect(akunIsianBarang(pilihan, { akun: '1304', kategori: 'Barang Jadi' })).toBe('1304')
  })
  it('kategori tak dikenal -> kosong supaya server menebak', () => {
    expect(akunIsianBarang(pilihan, { kategori: 'Lain-lain' })).toBeUndefined()
  })
  it('tanpa kategori dan akun -> akun awal barang baru', () => {
    expect(akunIsianBarang(pilihan, {})).toBe('1301')
  })
})
