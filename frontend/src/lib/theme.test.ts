import { describe, expect, it } from 'vitest'
import { temaDari } from './theme'

describe('temaDari', () => {
  it('default terang sesuai brand guideline (60% netral)', () => {
    expect(temaDari(null)).toBe('terang')
    expect(temaDari('')).toBe('terang')
    expect(temaDari('ngawur')).toBe('terang')
  })
  it('gelap hanya bila dipilih; pilihan lama tetap dihormati', () => {
    expect(temaDari('gelap')).toBe('gelap')
    expect(temaDari('terang')).toBe('terang')
  })
})
