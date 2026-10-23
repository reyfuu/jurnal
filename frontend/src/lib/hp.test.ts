import { describe, expect, it } from 'vitest'
import { labelSel } from './hp'

const H = [{ teks: 'Kode', span: 1 }, { teks: 'Akun', span: 1 }, { teks: 'Debit', span: 1 }, { teks: 'Kredit', span: 1 }]

describe('labelSel', () => {
  it('satu sel per kolom', () => expect(labelSel(H, [1, 1, 1, 1])).toEqual(['Kode', 'Akun', 'Debit', 'Kredit']))
  it('colSpan menggeser label', () => expect(labelSel(H, [2, 1, 1])).toEqual(['Kode', 'Debit', 'Kredit']))
  it('baris judul kelompok', () => expect(labelSel(H, [4])).toEqual(['']))
  it('header ber-colSpan', () => expect(labelSel([{ teks: 'Tanggal', span: 1 }, { teks: 'Nilai', span: 2 }], [1, 1, 1])).toEqual(['Tanggal', 'Nilai', 'Nilai']))
  it('sel lebih banyak dari header', () => expect(labelSel(H.slice(0, 2), [1, 1, 1])).toEqual(['Kode', 'Akun', '']))
})
