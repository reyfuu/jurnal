import { describe, expect, it } from 'vitest'
import { formatQty, parseQty, qtyKeTeks } from './qty'

describe('qty per seribu satuan', () => {
  it('membaca desimal koma & titik ribuan', () => {
    expect(parseQty('2,5')).toBe(2500)
    expect(parseQty('1.250')).toBe(1250000)
    expect(parseQty('0,018')).toBe(18)
    expect(parseQty('abc')).toBe(0)
  })
  it('menampilkan & mengisi ulang', () => {
    expect(formatQty(2500)).toBe('2,5')
    expect(formatQty(-1500)).toBe('-1,5')
    expect(qtyKeTeks(2500)).toBe('2,5')
    expect(qtyKeTeks(0)).toBe('')
  })
})
