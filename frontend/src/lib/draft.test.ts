import { describe, expect, it } from 'vitest'
import { bacaDraft, hapusDraft, simpanDraft } from './draft'

// Lingkungan tes tanpa DOM: localStorage tiruan berbasis Map.
const isi = new Map<string, string>()
globalThis.localStorage = {
  getItem: (k: string) => isi.get(k) ?? null,
  setItem: (k: string, v: string) => { isi.set(k, v) },
  removeItem: (k: string) => { isi.delete(k) },
} as Storage

describe('draft', () => {
  it('menyimpan, membaca, dan menghapus isian', () => {
    simpanDraft('uji', { nama: 'Kas', nominal: 5 })
    expect(bacaDraft<{ nama: string }>('uji')?.nama).toBe('Kas')
    hapusDraft('uji')
    expect(bacaDraft('uji')).toBeNull()
  })
})
