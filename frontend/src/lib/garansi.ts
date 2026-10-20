import { getMe } from '@/lib/api'

// Ekspor data pertama menggugurkan garansi 7 hari uang kembali (server: exporting). Selagi
// garansi masih berlaku, minta konfirmasi dulu sebelum mengunduh.
export const PESAN_EKSPOR =
  'Mengunduh/ekspor data akan menggugurkan garansi 7 hari uang kembali Anda. Lanjutkan?'

export async function ekspor(url: string) {
  const berlaku = await getMe().then((m) => m.garansi.berlaku, () => false)
  if (berlaku && !window.confirm(PESAN_EKSPOR)) return
  window.location.assign(url)
}
