import { describe, expect, it } from 'vitest'
import { angkaSifat, rapikanBaris } from './angkaLaporan'

describe('angkaSifat', () => {
  it('beban & HPP hitam tanpa minus, kontra & rugi merah dalam kurung', () => {
    expect(angkaSifat(-18_400_000, 'pengurang')).toEqual({ teks: '18.400.000', merah: false })
    expect(angkaSifat(500, 'pengurang')).toEqual({ teks: '(500)', merah: false })
    expect(angkaSifat(-625_000, 'kontra')).toEqual({ teks: '(625.000)', merah: false })
    expect(angkaSifat(-1_000)).toEqual({ teks: '(1.000)', merah: true })
    expect(angkaSifat(2_000)).toEqual({ teks: '2.000', merah: false })
  })
})

describe('rapikanBaris', () => {
  it('akun 0 disembunyikan, huruf & rumus disusun ulang', () => {
    const r = rapikanBaris([
      { kunci: 'omzet', label: 'Omzet', nilai: 100, gaya: 'baris', berhuruf: true, wajib: true },
      { kunci: 'potongan', label: 'Potongan', nilai: 0, gaya: 'baris', berhuruf: true },
      { kunci: 'bersih', label: 'Bersih', nilai: 100, gaya: 'total', berhuruf: true, suku: ['omzet', '-potongan'] },
      { kunci: 'judul', label: 'Beban', nilai: 0, gaya: 'judul' },
      { kunci: 'b1', label: 'Sewa', nilai: 0, gaya: 'baris' },
      { kunci: 'total', label: 'Total', nilai: 0, gaya: 'total', berhuruf: true, suku: ['*jumlah beban'] },
      { kunci: 'laba', label: 'Laba', nilai: 100, gaya: 'total', berhuruf: true, suku: ['bersih', '-total'] },
    ])
    expect(r.map((x) => x.kunci)).toEqual(['omzet', 'bersih', 'total', 'laba'])
    expect(r.map((x) => x.rumus)).toEqual([undefined, 'B = A', 'C = jumlah beban', 'D = B - C'])
  })
})
