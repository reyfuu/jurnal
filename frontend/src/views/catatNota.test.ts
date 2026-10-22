import { describe, expect, it } from 'vitest'
import { barisDariNota, rekeningBawaan } from './CatatJurnal'
import type { OpsiCatat, SaranNota } from '@/lib/api'

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
