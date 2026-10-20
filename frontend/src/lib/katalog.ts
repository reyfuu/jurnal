import type { Account, DiffAkun } from './api'

const sama = (x: Account, y: Account) =>
  x.nama === y.nama && x.subtype === y.subtype && x.normal === y.normal && x.kelompok === y.kelompok && x.kondisi === y.kondisi && (x.catatan ?? '') === (y.catatan ?? '')

/** Perubahan akun dari template lama ke baru (sama dengan katalog.Bandingkan di server). */
export function bandingkanAkun(lama: Account[], baru: Account[]): DiffAkun {
  const byKode = (xs: Account[]) => new Map(xs.map((x) => [x.kode, x]))
  const l = byKode(lama), b = byKode(baru)
  return {
    tambah: baru.filter((x) => !l.has(x.kode)),
    hapus: lama.filter((x) => !b.has(x.kode)),
    ubah: baru.filter((x) => l.has(x.kode) && !sama(l.get(x.kode)!, x)),
  }
}
