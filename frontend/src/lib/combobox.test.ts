import { describe, expect, it } from 'vitest'
import { filterOptions } from '@/lib/combobox'

const kota = ['Kabupaten Bandung', 'Kabupaten Bandung Barat', 'Kota Bandung', 'Kota Banjar', 'Kota Cimahi']

describe('filterOptions', () => {
  it('tanpa kata kunci: semua, dipotong batas', () => {
    expect(filterOptions(kota, '')).toEqual(kota)
    expect(filterOptions(kota, ' ', 2)).toEqual(kota.slice(0, 2))
  })
  it('tidak peka huruf besar/kecil; awal kata didahulukan', () => {
    expect(filterOptions(kota, 'BANDUNG')).toEqual(['Kabupaten Bandung', 'Kabupaten Bandung Barat', 'Kota Bandung'])
    expect(filterOptions(['Sukasari', 'Cisukaraja', 'Ngamprah'], 'suka')).toEqual(['Sukasari', 'Cisukaraja'])
  })
  it('pilihan di luar batas ditaruh paling atas; di dalam batas urutan tetap', () => {
    expect(filterOptions(kota, '', 2, 'Kota Cimahi')).toEqual(['Kota Cimahi', 'Kabupaten Bandung'])
    expect(filterOptions(kota, '', 2, 'Kabupaten Bandung Barat')).toEqual(kota.slice(0, 2))
    expect(filterOptions(kota, '', 2, 'tidak ada')).toEqual(kota.slice(0, 2))
  })
  it('pilihan AI jauh di bawah daftar tetap jadi opsi pertama, indeks 0 untuk fokus', () => {
    const klu = Array.from({ length: 150 }, (_, i) => `${100 + i} · Bidang ${i}`)
    const ai = klu[120]
    const shown = filterOptions(klu, '', 100, ai)
    expect(shown[0]).toBe(ai)
    expect(shown.indexOf(ai)).toBe(0)
    expect(shown).toHaveLength(100)
  })
  it('tidak cocok: kosong', () => {
    expect(filterOptions(kota, 'surabaya')).toEqual([])
  })
})
