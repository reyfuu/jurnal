import { describe, expect, it } from 'vitest'
import { jepit } from './geser'

describe('jepit', () => {
  it('tombol tidak bisa digeser keluar layar', () => {
    const layar = { w: 1000, h: 800 }
    expect(jepit({ right: -50, bottom: -10 }, layar)).toEqual({ right: 8, bottom: 8 })
    expect(jepit({ right: 5000, bottom: 5000 }, layar)).toEqual({ right: 1000 - 180 - 8, bottom: 800 - 48 - 8 })
    expect(jepit({ right: 300, bottom: 200 }, layar)).toEqual({ right: 300, bottom: 200 })
  })
})
