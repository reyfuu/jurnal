// Penyaring opsi Combobox (components/ui/combobox.tsx).

export const MAX_SHOWN = 100 // daftar panjang (mis. 600+ kecamatan) dipotong; ketik untuk mempersempit

// filterOptions: cocokkan tanpa memandang huruf besar/kecil; awalan kata didahulukan.
// pinned: pilihan saat ini; bila di luar potongan `limit` ia ditaruh paling atas supaya tetap tampil.
export function filterOptions(options: string[], q: string, limit = MAX_SHOWN, pinned = "") {
  const s = q.trim().toLowerCase()
  if (!s) {
    const head = options.slice(0, limit)
    return pinned && options.includes(pinned) && !head.includes(pinned) ? [pinned, ...head.slice(0, limit - 1)] : head
  }
  const prefix: string[] = []
  const rest: string[] = []
  for (const o of options) {
    const l = o.toLowerCase()
    if (l.startsWith(s) || l.includes(" " + s)) prefix.push(o)
    else if (l.includes(s)) rest.push(o)
  }
  return [...prefix, ...rest].slice(0, limit)
}
