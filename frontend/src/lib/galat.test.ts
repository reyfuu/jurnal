import { describe, expect, it } from 'vitest'
import { pesanGalat } from './galat'

const cloudflare520 = `<!DOCTYPE html>
<!--[if lt IE 7]> <html class="no-js ie6 oldie" lang="en-US"> <![endif]-->
<head><title>pajak.aipreneur.co.id | 520: Web server is returning an unknown error</title></head>
<body>${'x'.repeat(5000)}</body></html>`

describe('pesan galat dari balasan server', () => {
  it('halaman HTML Cloudflare 520 jadi kalimat singkat, bukan kode HTML', () => {
    const m = pesanGalat(520, cloudflare520, '')
    expect(m).toBe('Server sedang tidak bisa dihubungi (kode 520). Coba muat ulang beberapa saat lagi.')
  })

  it('502/504 tanpa isi juga dianggap server tidak terhubung', () => {
    expect(pesanGalat(502, '', 'Bad Gateway')).toContain('kode 502')
    expect(pesanGalat(504, '  ', '')).toContain('tidak bisa dihubungi')
  })

  it('pesan teks dari aplikasi tetap ditampilkan apa adanya, termasuk 503', () => {
    expect(pesanGalat(409, 'bulan 2026-10 sudah tutup buku\n', 'Conflict')).toBe('bulan 2026-10 sudah tutup buku')
    expect(pesanGalat(503, 'gagal menyimpan data, coba lagi', '')).toBe('gagal menyimpan data, coba lagi')
  })

  it('HTML pada status lain tetap tidak ditampilkan mentah', () => {
    const m = pesanGalat(500, '<html><body><h1>Internal Server Error</h1></body></html>', '')
    expect(m).not.toContain('<')
    expect(m).toContain('kode 500')
  })

  it('isi kosong memakai statusText atau kode', () => {
    expect(pesanGalat(404, '', 'Not Found')).toBe('Not Found')
    expect(pesanGalat(418, '', '')).toBe('Permintaan gagal (kode 418).')
  })

  it('teks yang sangat panjang dipotong', () => {
    expect(pesanGalat(400, 'a'.repeat(1000), '').length).toBeLessThanOrEqual(301)
  })
})
