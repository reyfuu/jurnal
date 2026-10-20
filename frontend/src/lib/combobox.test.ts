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
  it('tidak cocok: kosong', () => {
    expect(filterOptions(kota, 'surabaya')).toEqual([])
  })
})
