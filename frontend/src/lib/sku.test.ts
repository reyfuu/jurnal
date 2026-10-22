import { describe, expect, it } from 'vitest'
import { buatSKU } from './sku'

describe('buatSKU', () => {
  it('dari nama, unik terhadap yang sudah dipakai', () => {
    expect(buatSKU('Singkong kupas', new Set())).toBe('SIN-KUP')
    expect(buatSKU('singkong kupas segar', new Set(['SIN-KUP']))).toBe('SIN-KUP-2')
    expect(buatSKU('Gula', new Set())).toBe('GUL')
    expect(buatSKU('  ', new Set())).toBe('')
  })
})
