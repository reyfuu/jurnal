import type { TingkatCatatan } from './api'

export const TINGKAT: Record<TingkatCatatan, { label: string; pill: string; ket: string }> = {
  info: { label: 'Info', pill: 'pill-status-muted', ket: 'Sekadar informasi atau saran' },
  perbaiki: { label: 'Perlu diperbaiki', pill: 'pill-amber', ket: 'Ada data yang perlu dibetulkan klien' },
  penting: { label: 'Penting', pill: 'pill-red', ket: 'Berisiko pada pajak / laporan; segera ditindaklanjuti' },
}
