import { FILTER_PRIMBON_AWAL, type FacetPrimbon, type FilterPrimbon, type UrutPrimbon } from '@/lib/api'

// Keadaan halaman Aturan Pajak di URL: tab, filter katalog, dan peraturan yang dibuka
// (#/primbon-pajak/p/<uuid>?q=pph). Back browser kembali ke hasil yang sama, tautan detail
// bisa dibagikan.

export type TabPrimbon = 'katalog' | 'bookmark' | 'catatan'
export interface RutePrimbon { tab: TabPrimbon; buka: string | null; filter: FilterPrimbon }

const TAB: TabPrimbon[] = ['katalog', 'bookmark', 'catatan']
const URUT: UrutPrimbon[] = ['relevan', 'terbaru', 'terlama', 'abjad']
const tahun = (v: string | null) => (v && /^\d{4}$/.test(v) ? v : '')

export function bacaRutePrimbon(sub: string, p: URLSearchParams): RutePrimbon {
  const tab = p.get('tab') as TabPrimbon
  const urut = p.get('urut') as UrutPrimbon
  const hal = Number(p.get('hal'))
  let buka: string | null = null
  if (sub.startsWith('p/') && sub.length > 2) {
    try {
      buka = decodeURIComponent(sub.slice(2))
    } catch {
      buka = null
    }
  }
  return {
    tab: TAB.includes(tab) ? tab : 'katalog',
    buka,
    filter: {
      q: (p.get('q') ?? '').slice(0, 200),
      kategori: p.getAll('kategori'),
      jenis: p.getAll('jenis'),
      status: p.getAll('status'),
      dari: tahun(p.get('dari')),
      sampai: tahun(p.get('sampai')),
      urut: URUT.includes(urut) ? urut : '',
      hal: Number.isInteger(hal) && hal > 1 ? hal : 1,
    },
  }
}

export function tulisRutePrimbon(r: RutePrimbon): { sub: string; params: URLSearchParams } {
  const p = new URLSearchParams()
  const f = r.filter
  if (r.tab !== 'katalog') p.set('tab', r.tab)
  if (f.q.trim()) p.set('q', f.q.trim())
  for (const k of ['kategori', 'jenis', 'status'] as const) f[k].forEach((v) => p.append(k, v))
  if (f.dari) p.set('dari', f.dari)
  if (f.sampai) p.set('sampai', f.sampai)
  if (f.urut) p.set('urut', f.urut)
  if (f.hal > 1) p.set('hal', String(f.hal))
  return { sub: r.buka ? `p/${encodeURIComponent(r.buka)}` : '', params: p }
}

export interface ChipFilter { kunci: string; label: string; hapus: (f: FilterPrimbon) => FilterPrimbon }

/** Filter yang sedang aktif sebagai chip yang bisa dihapus satu per satu. */
export function chipFilter(f: FilterPrimbon, facet?: FacetPrimbon): ChipFilter[] {
  const label = (grup: 'kategori' | 'jenis' | 'status', kode: string) => facet?.[grup].find((o) => o.kode === kode)?.label || kode
  const out: ChipFilter[] = []
  for (const grup of ['kategori', 'jenis', 'status'] as const) {
    for (const kode of f[grup]) {
      out.push({ kunci: `${grup}:${kode}`, label: label(grup, kode), hapus: (x) => ({ ...x, [grup]: x[grup].filter((k) => k !== kode), hal: 1 }) })
    }
  }
  if (f.dari || f.sampai) {
    const teks = f.dari && f.sampai ? (f.dari === f.sampai ? f.dari : `${f.dari}–${f.sampai}`) : f.dari ? `Sejak ${f.dari}` : `Sampai ${f.sampai}`
    out.push({ kunci: 'tahun', label: `Tahun ${teks}`, hapus: (x) => ({ ...x, dari: '', sampai: '', hal: 1 }) })
  }
  return out
}

/** Hapus semua filter, kata kunci tetap. */
export const tanpaFilter = (f: FilterPrimbon): FilterPrimbon => ({ ...FILTER_PRIMBON_AWAL, q: f.q, urut: f.urut })

export const jumlahFilter = (f: FilterPrimbon) => f.kategori.length + f.jenis.length + f.status.length + (f.dari || f.sampai ? 1 : 0)
