import { describe, expect, it } from 'vitest'
import { pratinjau, rencana } from './aiForm'

describe('pratinjau isian AI', () => {
  it('transaksi: hanya kolom terisi, rupiah & status dalam bahasa pengguna', () => {
    expect(pratinjau({ form: 'transaksi', view: 'transaksi', isian: { tanggal: '2026-10-22', jenis: 'Penjualan', nominal: 2000000, keterangan: 'Penjualan kopi', lunas: true } }))
      .toEqual([
        { label: 'Tanggal', nilai: '22 Okt 2026' },
        { label: 'Jenis', nilai: 'Penjualan' },
        { label: 'Nominal', nilai: 'Rp2.000.000' },
        { label: 'Keterangan', nilai: 'Penjualan kopi' },
        { label: 'Status bayar', nilai: 'Lunas' },
      ])
  })

  it('transaksi belum lunas & barang dengan qty', () => {
    const rows = pratinjau({ form: 'transaksi', view: 'transaksi', isian: { tanggal: '2026-10-22', item: 'Kopi', qty: 3, lunas: false } })
    expect(rows).toContainEqual({ label: 'Barang', nilai: 'Kopi × 3' })
    expect(rows).toContainEqual({ label: 'Status bayar', nilai: 'Belum lunas' })
  })

  it('invoice: tiap item satu baris', () => {
    const rows = pratinjau({ form: 'invoice', view: 'invoice', isian: { tanggal: '2026-10-22', jatuh_tempo: '2026-11-21', pelanggan_nama: 'PT Maju', items: [{ deskripsi: 'Kaos', qty: 10, satuan: 'pcs', harga_satuan: 50000 }] } })
    expect(rows).toEqual([
      { label: 'Tanggal', nilai: '22 Okt 2026' },
      { label: 'Jatuh tempo', nilai: '21 Nov 2026' },
      { label: 'Pelanggan', nilai: 'PT Maju' },
      { label: 'Item 1', nilai: 'Kaos — 10 pcs × Rp50.000' },
    ])
  })

  it('stok: jumlah ditampilkan dengan satuannya', () => {
    expect(pratinjau({ form: 'stok', view: 'stok-gudang', isian: { nama: 'Kopi Arabika', satuan: 'gram', jumlah: 2500, harga_jual: 120000 } }))
      .toEqual([
        { label: 'Nama barang', nilai: 'Kopi Arabika' },
        { label: 'Satuan', nilai: 'gram' },
        { label: 'Stok awal', nilai: '2.500 gram' },
        { label: 'Harga jual', nilai: 'Rp120.000' },
      ])
  })

  it('pemasok: tempo dalam hari', () => {
    expect(pratinjau({ form: 'pemasok', view: 'produksi-supplier', isian: { nama: 'CV Sumber', telepon: '0812', tempo_hari: 14 } }))
      .toEqual([
        { label: 'Nama', nilai: 'CV Sumber' },
        { label: 'Telepon', nilai: '0812' },
        { label: 'Tempo bayar', nilai: '14 hari' },
      ])
  })
})

describe('rencana isi form', () => {
  it('buka form, isi kolom yang disebut, pengguna yang menyimpan', () => {
    expect(rencana({ form: 'transaksi', view: 'transaksi', isian: { tanggal: '2026-10-22', nominal: 2000000 } })).toEqual([
      'Buka Transaksi, form catat transaksi baru',
      'Isi 2 kolom: Tanggal, Nominal',
      'Anda periksa, lalu klik Simpan. Belum ada yang tersimpan sebelum itu.',
    ])
  })
})
