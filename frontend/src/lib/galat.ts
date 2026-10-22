// Ubah balasan gagal dari server jadi kalimat yang layak ditampilkan. Halaman error proxy
// (Cloudflare 520, nginx 502) berupa HTML panjang dan tidak boleh muncul mentah di layar.

const GERBANG = new Set([502, 503, 504, 520, 521, 522, 523, 524, 525, 526, 527, 530])
const MAKS = 300

export function pesanGalat(status: number, body: string, statusText: string): string {
  const isi = body.trim()
  const html = /^<|<html[\s>]/i.test(isi.slice(0, 300))
  if (GERBANG.has(status) && (html || !isi)) {
    return `Server sedang tidak bisa dihubungi (kode ${status}). Coba muat ulang beberapa saat lagi.`
  }
  if (html) return `Terjadi gangguan di server (kode ${status}). Coba muat ulang beberapa saat lagi.`
  if (!isi) return statusText || `Permintaan gagal (kode ${status}).`
  return isi.length > MAKS ? isi.slice(0, MAKS) + '…' : isi
}
