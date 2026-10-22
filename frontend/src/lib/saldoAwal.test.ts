import { describe, expect, it } from 'vitest'
import { urutSaldo } from './saldoAwal'

const KAT = ['kas', 'piutang', 'persediaan', 'aset_lain', 'utang', 'kewajiban_lain', 'modal', 'laba_ditahan']

describe('urutSaldo (checklist No 29: akun baru ikut urutan nomor akun)', () => {
  it('aset baru bernomor 1xxx pindah dari bawah modal ke kelompok harta, urut kode', () => {
    const rows = [
      { kode: '1101', nama: 'Kas', kategori: 'kas' },
      { kode: '1102', nama: 'Bank', kategori: 'kas' },
      { kode: '2101', nama: 'Utang Usaha', kategori: 'utang' },
      { kode: '3101', nama: 'Modal Pemilik', kategori: 'modal' },
      { kode: '1105', nama: 'Bank BNI', kategori: 'kas' },
    ]
    expect(urutSaldo(rows, KAT).map((r) => r.kode)).toEqual(['1101', '1102', '1105', '2101', '3101'])
  })
  it('baris tanpa kategori tetap di paling bawah; tanpa kode di akhir kategorinya; urutan setara stabil', () => {
    const rows = [
      { kode: '', nama: 'baru', kategori: '' },
      { kode: '', nama: 'Bank X', kategori: 'kas' },
      { kode: '1101', nama: 'Kas', kategori: 'kas' },
      { kode: '', nama: 'Bank Y', kategori: 'kas' },
    ]
    expect(urutSaldo(rows, KAT).map((r) => r.nama)).toEqual(['Kas', 'Bank X', 'Bank Y', 'baru'])
  })
})
