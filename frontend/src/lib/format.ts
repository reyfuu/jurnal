export const formatRp = (n: number) => (n < 0 ? '-' : '') + 'Rp' + Math.abs(n).toLocaleString('id-ID')

export function formatJt(rp: number) {
  const jt = rp / 1_000_000
  return 'Rp' + (Number.isInteger(jt) ? jt : jt.toFixed(1).replace('.', ',')) + ' jt'
}

export const formatSignedJt = (rp: number) => (rp >= 0 ? '+' : '-') + formatJt(Math.abs(rp))

/** Basis poin (2608) → "26,1%". */
export const formatPct = (bps: number) => (bps / 100).toFixed(1).replace('.', ',') + '%'
