import { describe, expect, it } from 'vitest'
import type { Mirip } from './api'
import { aksiMutasi, bolehPindah, patchAksi, sesuaikanRekening } from './mutasiAksi'
import { butuhBarang, sisaSesudahSimpan } from './mutasiBarang'

const kas: Mirip = { id: 7, tanggal: '2026-01-10', jenis: 'Biaya Lain', nominal: 500_000, belum_lunas: false, lawan: '1102' }

describe('aksi baris mutasi yang diduga sudah tercatat', () => {
  it('pindah akun bayar hanya untuk transaksi lunas biasa yang rekeningnya berbeda', () => {
    expect(bolehPindah(kas, '1111')).toBe(true)
    expect(bolehPindah(kas, '1102')).toBe(false) // sudah di rekening ini
    expect(bolehPindah(kas, '')).toBe(false) // rekening belum dimuat
    expect(bolehPindah({ ...kas, invoice_id: 3 }, '1111')).toBe(false)
    expect(bolehPindah({ ...kas, grup: 4 }, '1111')).toBe(true) // catatan Catat cepat
    expect(bolehPindah({ ...kas, grup: 4, produksi: true }, '1111')).toBe(false)
    expect(bolehPindah({ ...kas, khusus: true }, '1111')).toBe(false)
    expect(bolehPindah({ ...kas, belum_lunas: true }, '1111')).toBe(false) // itu pelunasan
    expect(bolehPindah({ ...kas, jenis: 'Pindah Dana' }, '1111')).toBe(false)
    expect(bolehPindah(undefined, '1111')).toBe(false)
  })

  it('aksi terbaca dari baris dan patch-nya saling meniadakan', () => {
    const r = { pilih: false, lunasi: 0, pindah: 0 }
    expect(aksiMutasi(r)).toBe('lewati')
    expect(aksiMutasi({ ...r, ...patchAksi('pindah', kas) })).toBe('pindah')
    expect(patchAksi('pindah', kas)).toEqual({ pilih: true, lunasi: 0, pindah: 7 })
    expect(patchAksi('lunasi', kas)).toEqual({ pilih: true, lunasi: 7, pindah: 0 })
    expect(patchAksi('baru', kas)).toEqual({ pilih: true, lunasi: 0, pindah: 0 })
    expect(patchAksi('lewati', kas)).toEqual({ pilih: false, lunasi: 0, pindah: 0 })
    expect(aksiMutasi({ pilih: true, lunasi: 7 })).toBe('lunasi')
    expect(aksiMutasi({ pilih: true })).toBe('baru')
  })

  it('ganti rekening: baris pindah yang tak berlaku lagi kembali dilewati', () => {
    const rows = [{ baris: 2, pilih: true, pindah: 7, mirip: kas }, { baris: 3, pilih: true, mirip: kas }]
    expect(sesuaikanRekening(rows, '1102')).toEqual([{ baris: 2, pilih: false, pindah: 0, mirip: kas }, rows[1]])
    expect(sesuaikanRekening(rows, '1112')).toBe(rows)
  })

  it('baris pindah bukan transaksi baru dan hilang sesudah tersimpan', () => {
    expect(butuhBarang({ pilih: true, pindah: 7, jenis: 'Penjualan', tanggal: '2026-01-10', nominal: 1 })).toBe(false)
    const rows = [{ baris: 2, pilih: true, pindah: 7 }, { baris: 3, pilih: true, pindah: 8 }]
    expect(sisaSesudahSimpan(rows, [{ baris: 3 }]).map((r) => r.baris)).toEqual([3])
  })
})
