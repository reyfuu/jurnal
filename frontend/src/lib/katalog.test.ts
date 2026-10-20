import { describe, expect, it } from 'vitest'
import type { Account } from './api'
import { bandingkanAkun } from './katalog'

const a = (kode: string, nama: string): Account => ({ kode, nama, subtype: 'opex', normal: 'debit', kelompok: 'Beban Operasional', kondisi: 'selalu' })

describe('bandingkanAkun', () => {
  it('memisahkan akun tambah, hapus, dan ubah', () => {
    const lama = [a('6101', 'Gaji'), a('6104', 'Sewa'), a('6116', 'Konten')]
    const baru = [a('6101', 'Gaji & Upah'), a('6104', 'Sewa'), a('6119', 'Asuransi')]
    const d = bandingkanAkun(lama, baru)
    expect(d.tambah.map((x) => x.kode)).toEqual(['6119'])
    expect(d.hapus.map((x) => x.kode)).toEqual(['6116'])
    expect(d.ubah.map((x) => x.kode)).toEqual(['6101'])
  })
})
