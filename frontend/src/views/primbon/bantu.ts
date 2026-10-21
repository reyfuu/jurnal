import type { Peraturan } from '@/lib/api'

export const MONO = { fontFamily: 'var(--font-mono)' }

/** Warna pill status: hijau berlaku, kuning diubah, merah dicabut/tidak berlaku. */
export function pillStatus(status?: string) {
  if (status === 'Berlaku') return 'pill-green'
  if (status === 'Diubah') return 'pill-amber'
  if (status && /cabut|tidak berlaku/i.test(status)) return 'pill-red'
  return 'pill-status-muted'
}

export const jenisSingkat = (p: Peraturan) => (p.jenis === 'LAIN' ? 'Lainnya' : p.jenis.replace('_', ' '))

export function keteranganNaskah(p: Peraturan) {
  if (p.format === 'pdf') return p.abstrak ? 'PDF abstrak' : 'PDF resmi'
  if (p.format === 'htm') return 'Naskah HTML resmi'
  return 'Naskah belum tersedia'
}

export function formatWaktu(iso: string) {
  return new Date(iso).toLocaleString('id-ID', {
    day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit', timeZone: 'Asia/Jakarta',
  })
}
