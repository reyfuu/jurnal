// SKU otomatis dari nama barang: 3 huruf pertama tiap kata (maks 2 kata) + nomor urut bila sudah
// dipakai. "Singkong kupas" → "SIN-KUP", berikutnya "SIN-KUP-2". Pengguna tetap boleh mengetik sendiri.

export function buatSKU(nama: string, terpakai: Set<string>): string {
  const kata = nama.toUpperCase().normalize('NFKD').replace(/[^A-Z0-9 ]/g, ' ').split(/\s+/).filter(Boolean)
  if (!kata.length) return ''
  const dasar = kata.slice(0, 2).map((k) => k.slice(0, 3)).join('-')
  if (!terpakai.has(dasar)) return dasar
  for (let i = 2; ; i++) if (!terpakai.has(`${dasar}-${i}`)) return `${dasar}-${i}`
}
