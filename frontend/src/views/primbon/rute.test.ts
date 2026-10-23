import { describe, expect, it } from 'vitest'
import { FILTER_PRIMBON_AWAL, urutPrimbon, type FacetPrimbon } from '@/lib/api'
import { bacaRutePrimbon, chipFilter, jumlahFilter, tanpaFilter, tulisRutePrimbon, type RutePrimbon } from './rute'
import { pecahSorot } from './sorot'

const r = (s: string) => new URLSearchParams(s)

describe('rute Aturan Pajak', () => {
  it('filter, tab, dan detail bolak-balik lewat URL', () => {
    const awal: RutePrimbon = {
      tab: 'katalog', buka: 'a/1 b',
      filter: { q: 'pph 21', kategori: ['PPh', 'UMKM'], jenis: ['PMK'], status: [], dari: '2020', sampai: '2024', urut: 'abjad', hal: 3 },
    }
    const { sub, params } = tulisRutePrimbon(awal)
    expect(sub).toBe('p/a%2F1%20b')
    expect(bacaRutePrimbon(sub, params)).toEqual(awal)
  })
  it('keadaan awal tidak menulis parameter apa pun', () => {
    const { sub, params } = tulisRutePrimbon({ tab: 'katalog', buka: null, filter: FILTER_PRIMBON_AWAL })
    expect([sub, params.toString()]).toEqual(['', ''])
    expect(bacaRutePrimbon('', r(''))).toEqual({ tab: 'katalog', buka: null, filter: FILTER_PRIMBON_AWAL })
  })
  it('isian rusak diabaikan', () => {
    const x = bacaRutePrimbon('p/%E0%A4%A', r('tab=x&urut=acak&hal=-2&dari=20a4&sampai=99999'))
    expect(x.buka).toBeNull()
    expect([x.tab, x.filter.urut, x.filter.hal, x.filter.dari, x.filter.sampai]).toEqual(['katalog', '', 1, '', ''])
  })
})

describe('urut otomatis', () => {
  it('relevan bila ada kata kunci, terbaru bila tidak, pilihan pengguna menang', () => {
    expect(urutPrimbon({ ...FILTER_PRIMBON_AWAL, q: ' pph ' })).toBe('relevan')
    expect(urutPrimbon(FILTER_PRIMBON_AWAL)).toBe('terbaru')
    expect(urutPrimbon({ ...FILTER_PRIMBON_AWAL, q: 'pph', urut: 'terlama' })).toBe('terlama')
  })
})

describe('chip filter aktif', () => {
  const facet: FacetPrimbon = {
    kategori: [{ kode: 'PPh', label: 'Pajak Penghasilan', jumlah: 3 }], jenis: [], status: [], tahun_min: 1983, tahun_max: 2026,
  }
  const f = { ...FILTER_PRIMBON_AWAL, q: 'tarif', kategori: ['PPh', 'X'], status: ['Berlaku'], dari: '2020', hal: 4 }
  it('memakai label facet dan bisa dihapus satu per satu (kembali ke halaman 1)', () => {
    const chip = chipFilter(f, facet)
    expect(chip.map((c) => c.label)).toEqual(['Pajak Penghasilan', 'X', 'Berlaku', 'Tahun Sejak 2020'])
    const sisa = chip[0].hapus(f)
    expect([sisa.kategori, sisa.status, sisa.hal]).toEqual([['X'], ['Berlaku'], 1])
    expect(chip[3].hapus(f).dari).toBe('')
  })
  it('rentang tahun & hapus semua filter', () => {
    expect(chipFilter({ ...FILTER_PRIMBON_AWAL, dari: '2022', sampai: '2022' })[0].label).toBe('Tahun 2022')
    expect(chipFilter({ ...FILTER_PRIMBON_AWAL, sampai: '2010' })[0].label).toBe('Tahun Sampai 2010')
    expect(jumlahFilter(f)).toBe(4)
    expect(tanpaFilter(f)).toEqual({ ...FILTER_PRIMBON_AWAL, q: 'tarif' })
  })
})

describe('sorot kata kunci', () => {
  const ditandai = (teks: string, q: string) => pecahSorot(teks, q).filter((p) => p.cocok).map((p) => p.teks)
  it('tanpa beda huruf besar, semua kata', () => {
    expect(ditandai('Pajak Penghasilan Pasal 21', 'pasal PAJAK')).toEqual(['Pajak', 'Pasal'])
    expect(pecahSorot('PMK 10', 'pmk').map((p) => p.teks).join('')).toBe('PMK 10')
  })
  it('singkatan ikut menandai bentuk panjangnya', () => {
    expect(ditandai('Pajak Penghasilan Pasal 21', 'pph 21')).toEqual(['Pajak Penghasilan', '21'])
  })
  it('angka harus utuh', () => {
    expect(ditandai('PMK Nomor 10 Tahun 2010', '10')).toEqual(['10'])
  })
  it('karakter khusus regex aman & teks HTML tetap teks', () => {
    expect(ditandai('PER-03/PJ/2022 (lampiran)', '(lampiran) per-03/pj/2022')).toEqual(['PER-03/PJ/2022', '(lampiran)'])
    expect(pecahSorot('<b>x</b>', 'b')[0]).toEqual({ teks: '<', cocok: false })
  })
  it('tanpa kata kunci satu potongan utuh', () => {
    expect(pecahSorot('Judul', '  ')).toEqual([{ teks: 'Judul', cocok: false }])
  })
})
