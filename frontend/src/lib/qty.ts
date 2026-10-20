// Jumlah barang disimpan per seribu satuan (2500 = 2,5 kg), sama dengan takaran resep.

export const SKALA = 1000

/** 2500 → "2,5"; -1500 → "-1,5". */
export function formatQty(mili: number) {
  return (mili / SKALA).toLocaleString('id-ID', { maximumFractionDigits: 3 })
}

/** "2,5" atau "2.5" → 2500; isian kosong/salah → 0. */
export function parseQty(s: string) {
  const n = parseFloat(s.replace(/\./g, '').replace(',', '.'))
  return Number.isFinite(n) ? Math.round(n * SKALA) : 0
}

/** Untuk nilai awal input teks dari jumlah per seribu. */
export const qtyKeTeks = (mili: number) => (mili ? String(mili / SKALA).replace('.', ',') : '')
