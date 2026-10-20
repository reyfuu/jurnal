import { describe, expect, it } from 'vitest'
import { temaDari } from './theme'

describe('temaDari', () => {
  it('default gelap: pengguna lama tidak melihat perubahan', () => {
    expect(temaDari(null)).toBe('gelap')
    expect(temaDari('')).toBe('gelap')
    expect(temaDari('ngawur')).toBe('gelap')
  })
  it('terang hanya bila dipilih', () => {
    expect(temaDari('terang')).toBe('terang')
    expect(temaDari('gelap')).toBe('gelap')
  })
})
