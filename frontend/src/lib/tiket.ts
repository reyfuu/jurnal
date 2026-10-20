import type { TiketStatus } from './api'

// Label & warna status tiket, dipakai Lapor Bug (klien) dan Panel Master (tim).
export const STATUS_LABEL: Record<TiketStatus, string> = { baru: 'Baru', dikerjakan: 'Sedang dikerjakan', selesai: 'Selesai' }
export const STATUS_PILL: Record<TiketStatus, string> = { baru: 'pill-amber', dikerjakan: 'pill-status-muted', selesai: 'pill-green' }
