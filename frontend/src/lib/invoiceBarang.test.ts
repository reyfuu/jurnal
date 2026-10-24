import { describe, expect, it } from 'vitest'
import { barisTanpaBarang, galatBarangInvoice, isiDariBarang, perluCekBarang, wajibBarangInvoice } from './invoiceBarang'

const kain = { id: 7, sku: 'KN-01', nama: 'Kain katun', kategori: 'Bahan Baku', satuan: 'm', jumlah_mili: 10_000, harga_jual: 25_000 }

describe('barang per baris invoice', () => {
  it('wajib hanya di perpetual, bukan draft, dan sejak kartu dimulai', () => {
    expect(wajibBarangInvoice({ perpetual: true, status: 'terkirim', tanggal: '2026-01-10', mulai: '2026-01-01' })).toBe(true)
    expect(wajibBarangInvoice({ perpetual: true, status: 'terkirim', tanggal: '2026-01-10', mulai: '' })).toBe(true)
    expect(wajibBarangInvoice({ perpetual: true, status: 'draft', tanggal: '2026-01-10', mulai: '' })).toBe(false)
    expect(wajibBarangInvoice({ perpetual: true, status: 'lunas', tanggal: '2025-12-31', mulai: '2026-01-01' })).toBe(false)
    expect(wajibBarangInvoice({ perpetual: false, status: 'lunas', tanggal: '2026-01-10', mulai: '' })).toBe(false)
  })

  it('pesan per baris sama dengan server', () => {
    const items = [{ deskripsi: 'Kain', qty: 1, harga_satuan: 1, stok_id: 7 }, { deskripsi: 'Ongkir', qty: 1, harga_satuan: 1 }]
    expect(barisTanpaBarang(items)).toEqual([1])
    expect(galatBarangInvoice(items, true)).toBe('baris 2: pilih barang atau tandai Bukan barang (jasa)')
    expect(galatBarangInvoice(items, false)).toBe('')
    expect(galatBarangInvoice([items[0], { ...items[1], bukan_barang: true }], true)).toBe('')
    expect(galatBarangInvoice([{ ...items[0], bukan_barang: true }], false)).toBe('baris 1: pilih barang atau tandai Bukan barang (jasa), tidak keduanya')
  })

  it('invoice lama yang tidak diubah tidak dicek', () => {
    const asli = { status: 'terkirim', tanggal: '2026-01-10', items: [{ deskripsi: 'Kopi', qty: 2, harga_satuan: 1000 }] }
    expect(perluCekBarang(null, asli)).toBe(true)
    expect(perluCekBarang(asli, { ...asli, status: 'lunas', items: asli.items.map((i) => ({ ...i })) })).toBe(false)
    expect(perluCekBarang(asli, { ...asli, items: [{ ...asli.items[0], qty: 3 }] })).toBe(true)
    expect(perluCekBarang(asli, { ...asli, tanggal: '2026-01-11' })).toBe(true)
    expect(perluCekBarang({ ...asli, status: 'draft' }, { ...asli, status: 'terkirim' })).toBe(true)
  })

  it('memilih barang mengisi deskripsi, satuan, harga jual bila masih kosong', () => {
    expect(isiDariBarang({ deskripsi: '', qty: 1, satuan: '', harga_satuan: 0 }, kain))
      .toEqual({ stok_id: 7, bukan_barang: undefined, deskripsi: 'Kain katun', satuan: 'm', harga_satuan: 25_000 })
    expect(isiDariBarang({ deskripsi: 'Kain merah', qty: 1, satuan: 'roll', harga_satuan: 30_000, bukan_barang: true }, kain))
      .toEqual({ stok_id: 7, bukan_barang: undefined, deskripsi: 'Kain merah', satuan: 'roll', harga_satuan: 30_000 })
  })
})
