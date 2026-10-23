import { describe, expect, it } from 'vitest'
import { kolomHeader, labelSel } from './hp'

// Header Mutasi stok (KartuStok): dua baris, rowSpan & colSpan.
const MUTASI = [
  [{ teks: 'No', span: 1, rowSpan: 2 }, { teks: 'Barang', span: 1, rowSpan: 2 }, { teks: 'Awal', span: 1, rowSpan: 2 },
    { teks: 'Masuk', span: 2, rowSpan: 1 }, { teks: 'Keluar', span: 2, rowSpan: 1 },
    { teks: 'Opname', span: 1, rowSpan: 2 }, { teks: 'Akhir', span: 1, rowSpan: 2 }, { teks: 'Nilai akhir', span: 1, rowSpan: 2 }],
  [{ teks: 'Beli', span: 1, rowSpan: 1 }, { teks: 'Produksi', span: 1, rowSpan: 1 }, { teks: 'Terjual', span: 1, rowSpan: 1 }, { teks: 'Dipakai produksi', span: 1, rowSpan: 1 }],
]

describe('kolomHeader', () => {
  it('header satu baris', () => expect(kolomHeader([[{ teks: 'A', span: 1, rowSpan: 1 }, { teks: 'B', span: 2, rowSpan: 1 }]])).toEqual(['A', 'B', 'B']))
  it('header dua baris dengan rowSpan & colSpan', () => expect(kolomHeader(MUTASI)).toEqual(
    ['No', 'Barang', 'Awal', 'Masuk · Beli', 'Masuk · Produksi', 'Keluar · Terjual', 'Keluar · Dipakai produksi', 'Opname', 'Akhir', 'Nilai akhir']))
  it('label sel baris data Mutasi stok', () => {
    const kolom = kolomHeader(MUTASI).map((teks) => ({ teks, span: 1 }))
    expect(labelSel(kolom, Array(10).fill(1))[2]).toBe('Awal')
    expect(labelSel(kolom, [2, 1, 1, 1, 1, 1, 1, 1, 1])).toEqual(['No', 'Awal', 'Masuk · Beli', 'Masuk · Produksi', 'Keluar · Terjual', 'Keluar · Dipakai produksi', 'Opname', 'Akhir', 'Nilai akhir'])
  })
})

const H = [{ teks: 'Kode', span: 1 }, { teks: 'Akun', span: 1 }, { teks: 'Debit', span: 1 }, { teks: 'Kredit', span: 1 }]

describe('labelSel', () => {
  it('satu sel per kolom', () => expect(labelSel(H, [1, 1, 1, 1])).toEqual(['Kode', 'Akun', 'Debit', 'Kredit']))
  it('colSpan menggeser label', () => expect(labelSel(H, [2, 1, 1])).toEqual(['Kode', 'Debit', 'Kredit']))
  it('baris judul kelompok', () => expect(labelSel(H, [4])).toEqual(['']))
  it('header ber-colSpan', () => expect(labelSel([{ teks: 'Tanggal', span: 1 }, { teks: 'Nilai', span: 2 }], [1, 1, 1])).toEqual(['Tanggal', 'Nilai', 'Nilai']))
  it('sel lebih banyak dari header', () => expect(labelSel(H.slice(0, 2), [1, 1, 1])).toEqual(['Kode', 'Akun', '']))
})
