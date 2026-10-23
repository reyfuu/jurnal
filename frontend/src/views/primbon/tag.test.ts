import { describe, expect, it } from 'vitest'
import { tagPeraturan } from './tag'

describe('tag peraturan', () => {
  it('tidak menganggap status kosong sebagai berlaku', () => {
    expect(tagPeraturan({ kategori_label: [] })).toEqual([{ label: 'Status belum diketahui', warna: 'pill-status-muted' }])
  })
  it('memisahkan status, perubahan, dan topik tanpa duplikasi', () => {
    const tags = tagPeraturan({ status: 'Berlaku', perubahan: 'Diubah', kategori_label: ['Pajak Penghasilan'], tag: ['Payroll', 'payroll', 'Pajak Penghasilan'] })
    expect(tags.map(t => t.label)).toEqual(['Berlaku', 'Diubah', 'Pajak Penghasilan', 'Payroll'])
    expect(tags[1].warna).toBe('pill-amber')
  })
  it('tidak menduplikasi status yang sama dengan perubahan', () => {
    expect(tagPeraturan({ status: 'Diubah', perubahan: 'Diubah', kategori_label: [] })).toHaveLength(1)
  })
})
