export const formatRp = (n: number) => (n < 0 ? '-' : '') + 'Rp' + Math.abs(n).toLocaleString('id-ID')

/** Harga paket ringkas: 199000 → "Rp199 rb", 1990000 → "Rp1,99 jt", 5000000 → "Rp5 jt". */
export function formatRpRingkas(n: number) {
  if (n < 1_000_000) return `Rp${Math.round(n / 1000)} rb`
  return `Rp${String(Math.round(n / 10_000) / 100).replace('.', ',')} jt`
}

export function formatJt(rp: number) {
  const jt = rp / 1_000_000
  return 'Rp' + (Number.isInteger(jt) ? jt : jt.toFixed(1).replace('.', ',')) + ' jt'
}

export const formatSignedJt = (rp: number) => (rp >= 0 ? '+' : '-') + formatJt(Math.abs(rp))

/** Rupiah penuh bertanda: +Rp1.250.000 / -Rp500.000 (laporan keuangan tidak dibulatkan ke juta). */
export const formatSignedRp = (rp: number) => (rp >= 0 ? '+' : '-') + formatRp(Math.abs(rp))

/** Basis poin (2608) → "26,1%". */
export const formatPct = (bps: number) => (bps / 100).toFixed(1).replace('.', ',') + '%'

/** Waktu relatif untuk "terakhir aktif": "3 jam lalu", "kemarin". Kosong = belum pernah. */
export function formatRelatif(iso: string | undefined, now = new Date()) {
  if (!iso) return 'belum pernah'
  const menit = Math.floor((now.getTime() - new Date(iso).getTime()) / 60_000)
  if (menit < 1) return 'baru saja'
  if (menit < 60) return `${menit} menit lalu`
  const jam = Math.floor(menit / 60)
  if (jam < 24) return `${jam} jam lalu`
  const hari = Math.floor(jam / 24)
  return hari === 1 ? 'kemarin' : `${hari} hari lalu`
}

/** "14 Nov 2026"; kosong = "—". Selalu zona Asia/Jakarta supaya sama di server & browser. */
export function formatTanggal(iso: string | undefined) {
  if (!iso) return '—'
  return new Date(iso).toLocaleDateString('id-ID', { day: 'numeric', month: 'short', year: 'numeric', timeZone: 'Asia/Jakarta' }).replace('.', '')
}
