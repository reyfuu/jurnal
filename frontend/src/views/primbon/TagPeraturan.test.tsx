import { describe, expect, it } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import type { Peraturan } from '@/lib/api'
import { TagPeraturan } from './TagPeraturan'

const regulation: Peraturan = { uuid: 'test', kode: 'PMK contoh', judul: 'Contoh pengujian', jenis: 'PMK', jenis_label: 'Peraturan Menteri Keuangan', kategori: [], kategori_label: [], bookmark: false, ada_catatan: false }

describe('metadata peraturan dirender', () => {
  it('menampilkan nilai yang belum diketahui dan tidak mengklaim berlaku', () => {
    const html = renderToStaticMarkup(<TagPeraturan p={regulation} />)
    expect(html).toContain('Status belum diketahui')
    expect(html).toContain('Mulai berlaku belum diketahui')
    expect(html).toContain('Tanggal acuan status belum tersedia')
  })
  it('menampilkan tanggal acuan serta meng-escape isi tag', () => {
    const html = renderToStaticMarkup(<TagPeraturan p={{ ...regulation, status: 'Berlaku', perubahan: 'Diubah', status_pada: '2026-10-23', berlaku: '2026-01-01', tag: ['<script>alert(1)</script>'] }} />)
    expect(html).toContain('aria-label="Tag peraturan"')
    expect(html).toContain('Status per')
    expect(html).toContain('2026')
    expect(html).not.toContain('<script>')
    expect(html).toContain('&lt;script&gt;')
  })
})
