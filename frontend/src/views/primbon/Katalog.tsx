import { useEffect, useRef, useState, type FormEvent } from 'react'
import { Search, SlidersHorizontal } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { cariPrimbon, FILTER_PRIMBON_AWAL, type FilterPrimbon, type HasilPrimbon, type OpsiPrimbon, type Peraturan } from '@/lib/api'
import { KartuPeraturan } from './KartuPeraturan'

type Muat = { untuk: FilterPrimbon; hasil?: HasilPrimbon; error?: string }

/** Katalog: panel filter di kiri (diterapkan dengan tombol Terapkan), hasil di kanan. */
export function Katalog({ filter, onFilter, onBuka }: {
  filter: FilterPrimbon
  onFilter: (f: FilterPrimbon) => void
  onBuka: (uuid: string) => void
}) {
  const [muat, setMuat] = useState<Muat | null>(null)
  const [draf, setDraf] = useState(filter)
  const [panelTerbuka, setPanelTerbuka] = useState(false)
  const atas = useRef<HTMLElement>(null)

  useEffect(() => {
    let batal = false
    cariPrimbon(filter).then(
      (hasil) => { if (!batal) setMuat({ untuk: filter, hasil }) },
      (e: Error) => { if (!batal) setMuat((m) => ({ untuk: filter, hasil: m?.hasil, error: e.message })) },
    )
    return () => { batal = true }
  }, [filter])

  const memuat = muat?.untuk !== filter
  const hasil = muat?.hasil
  const facet = hasil?.facet
  const jumlahFilter = filter.kategori.length + filter.jenis.length + filter.status.length + (filter.dari ? 1 : 0) + (filter.sampai ? 1 : 0)

  function terapkan(e?: FormEvent) {
    e?.preventDefault()
    onFilter({ ...draf, hal: 1 })
    setPanelTerbuka(false)
  }
  function reset() {
    setDraf(FILTER_PRIMBON_AWAL)
    onFilter(FILTER_PRIMBON_AWAL)
  }
  function pindah(hal: number) {
    onFilter({ ...filter, hal })
    atas.current?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }
  const ubahItem = (p: Peraturan) =>
    setMuat((m) => m?.hasil ? { ...m, hasil: { ...m.hasil, items: m.hasil.items.map((x) => (x.uuid === p.uuid ? p : x)) } } : m)
  const tahun = (v: string) => v.replace(/\D/g, '').slice(0, 4)

  return (
    <div className="grid items-start gap-5 lg:grid-cols-[272px_minmax(0,1fr)]">
      <form onSubmit={terapkan} className="card lg:sticky lg:top-4 lg:max-h-[calc(100vh-2rem)] lg:overflow-y-auto"
        style={{ padding: 16, gap: 18, justifyContent: 'flex-start' }} aria-label="Filter peraturan">
        <div>
          <label htmlFor="primbon-cari" className="label-caps text-muted-foreground">Cari peraturan</label>
          <div className="relative mt-2">
            <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
            <Input id="primbon-cari" type="search" className="pl-9" placeholder="Nomor atau judul, mis. PMK 81 2024"
              value={draf.q} onChange={(e) => setDraf({ ...draf, q: e.target.value })} />
          </div>
        </div>
        <Button type="button" variant="outline" size="sm" className="lg:hidden" aria-expanded={panelTerbuka}
          onClick={() => setPanelTerbuka(!panelTerbuka)}>
          <SlidersHorizontal /> Filter{jumlahFilter ? ` (${jumlahFilter})` : ''}
        </Button>
        <div className={(panelTerbuka ? 'flex' : 'hidden') + ' flex-col gap-5 lg:flex'}>
          <GrupCentang judul="Kategori" opsi={facet?.kategori} pilih={draf.kategori} onUbah={(kategori) => setDraf({ ...draf, kategori })} />
          <GrupCentang judul="Jenis peraturan" opsi={facet?.jenis} pilih={draf.jenis} onUbah={(jenis) => setDraf({ ...draf, jenis })} />
          <GrupCentang judul="Status" opsi={facet?.status} pilih={draf.status} onUbah={(status) => setDraf({ ...draf, status })} />
          <fieldset>
            <legend className="label-caps mb-2 text-muted-foreground">Tahun</legend>
            <div className="flex items-center gap-2">
              <Input aria-label="Tahun dari" inputMode="numeric" placeholder="dari" value={draf.dari}
                onChange={(e) => setDraf({ ...draf, dari: tahun(e.target.value) })} />
              <span className="text-muted-foreground" aria-hidden="true">—</span>
              <Input aria-label="Tahun sampai" inputMode="numeric" placeholder="sampai" value={draf.sampai}
                onChange={(e) => setDraf({ ...draf, sampai: tahun(e.target.value) })} />
            </div>
          </fieldset>
          <div>
            <div id="primbon-urut" className="label-caps mb-2 text-muted-foreground">Urutkan</div>
            <Select value={draf.urut} onValueChange={(urut) => setDraf({ ...draf, urut: urut as FilterPrimbon['urut'] })}>
              <SelectTrigger className="w-full" aria-labelledby="primbon-urut"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="terbaru">Terbaru</SelectItem>
                <SelectItem value="terlama">Terlama</SelectItem>
                <SelectItem value="abjad">Abjad (A–Z)</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
        <div className="flex gap-2 lg:sticky lg:bottom-0 lg:-mx-4 lg:-mb-4 lg:px-4 lg:pt-3 lg:pb-4"
          style={{ background: 'var(--bg-card)', borderTop: '1px solid var(--border-subtle)' }}>
          <Button type="submit" className="flex-1">Terapkan</Button>
          <Button type="button" variant="ghost" onClick={reset}>Reset</Button>
        </div>
      </form>

      <section ref={atas} className="flex min-w-0 scroll-mt-4 flex-col gap-3" aria-busy={memuat}>
        <div className="flex flex-wrap items-end justify-between gap-2">
          <div>
            <div className="label-caps text-muted-foreground">Peraturan</div>
            <h2 className="font-heading text-xl font-bold tracking-tight">Jelajahi katalog</h2>
          </div>
          {hasil && (
            <div className="text-sm text-muted-foreground">
              <strong className="tabular" style={{ color: 'var(--text-primary)' }}>{hasil.total.toLocaleString('id-ID')}</strong> peraturan
            </div>
          )}
        </div>
        {muat?.error && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start' }}>{muat.error}</div>}
        {!hasil && !muat?.error && <p className="workspace-subtitle">Memuat katalog…</p>}
        {hasil && hasil.items.length === 0 && (
          <div className="card text-sm text-muted-foreground" style={{ padding: 20 }}>
            Tidak ada peraturan yang cocok. Coba kata lain atau kurangi filter.
          </div>
        )}
        <div className="flex flex-col gap-3" style={{ opacity: memuat && hasil ? 0.55 : 1, transition: 'opacity .15s' }}>
          {hasil?.items.map((p) => <KartuPeraturan key={p.uuid} p={p} onBuka={() => onBuka(p.uuid)} onUbah={ubahItem} />)}
        </div>
        {hasil && hasil.total_halaman > 1 && <Paginasi halaman={hasil.halaman} total={hasil.total_halaman} onPindah={pindah} />}
      </section>
    </div>
  )
}

function GrupCentang({ judul, opsi, pilih, onUbah }: {
  judul: string
  opsi?: OpsiPrimbon[]
  pilih: string[]
  onUbah: (v: string[]) => void
}) {
  return (
    <fieldset>
      <legend className="label-caps mb-2 text-muted-foreground">{judul}</legend>
      <div className="flex flex-col gap-0.5">
        {(opsi ?? []).map((o) => (
          <label key={o.kode} className="flex cursor-pointer items-start gap-2 rounded-md px-1.5 py-1 text-sm hover:bg-[var(--bg-card-hover)]">
            <input type="checkbox" className="mt-0.5 size-4 shrink-0 cursor-pointer" style={{ accentColor: 'var(--primary)' }}
              checked={pilih.includes(o.kode)}
              onChange={(e) => onUbah(e.target.checked ? [...pilih, o.kode] : pilih.filter((k) => k !== o.kode))} />
            <span className="flex-1 leading-snug">{o.label}</span>
            <span className="tabular text-xs text-muted-foreground">{o.jumlah.toLocaleString('id-ID')}</span>
          </label>
        ))}
      </div>
    </fieldset>
  )
}

/** 1 2 3 … 76: halaman pertama, terakhir, dan di sekitar halaman aktif. */
function nomorHalaman(h: number, total: number): (number | '…')[] {
  const ada = new Set([1, total, h - 1, h, h + 1, h === 1 ? 3 : 0, h === total ? total - 2 : 0].filter((n) => n >= 1 && n <= total))
  const urut = [...ada].sort((a, b) => a - b)
  return urut.flatMap((n, i) => (i > 0 && n - urut[i - 1] > 1 ? ['…' as const, n] : [n]))
}

function Paginasi({ halaman, total, onPindah }: { halaman: number; total: number; onPindah: (h: number) => void }) {
  return (
    <nav aria-label="Halaman katalog" className="mt-2 flex flex-wrap items-center justify-between gap-3 text-sm">
      <span className="tabular text-muted-foreground">Halaman {halaman} dari {total}</span>
      <div className="flex flex-wrap items-center gap-1">
        <Button variant="ghost" size="sm" disabled={halaman <= 1} onClick={() => onPindah(halaman - 1)}>← Sebelumnya</Button>
        {nomorHalaman(halaman, total).map((n, i) => n === '…'
          ? <span key={'e' + i} className="px-1 text-muted-foreground" aria-hidden="true">…</span>
          : (
            <Button key={n} size="sm" variant={n === halaman ? 'outline' : 'ghost'} className="tabular min-w-8"
              aria-current={n === halaman ? 'page' : undefined} onClick={() => onPindah(n)}>
              {n}
            </Button>
          ))}
        <Button variant="ghost" size="sm" disabled={halaman >= total} onClick={() => onPindah(halaman + 1)}>Berikutnya →</Button>
      </div>
    </nav>
  )
}
