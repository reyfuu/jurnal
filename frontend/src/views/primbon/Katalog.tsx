import { useEffect, useRef, useState, type ReactNode } from 'react'
import { Loader2, Search, SlidersHorizontal, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { cariPrimbon, urutPrimbon, type FacetPrimbon, type FilterPrimbon, type HasilPrimbon, type OpsiPrimbon, type Peraturan, type UrutPrimbon } from '@/lib/api'
import { KartuPeraturan, KerangkaKartu } from './KartuPeraturan'
import { chipFilter, jumlahFilter, tanpaFilter } from './rute'

type Muat = { kunci: string; hasil?: HasilPrimbon; error?: string }

let terakhir: Muat | null = null

const JEDA_KETIK = 300
const LABEL_URUT: Record<UrutPrimbon, string> = { relevan: 'Paling relevan', terbaru: 'Terbaru', terlama: 'Terlama', abjad: 'Judul A–Z' }

/** Katalog: hasil langsung berubah saat mengetik atau mencentang filter (tanpa tombol Terapkan). */
export function Katalog({ filter, onFilter, onBuka }: {
  filter: FilterPrimbon
  onFilter: (f: FilterPrimbon) => void
  onBuka: (uuid: string) => void
}) {
  const kunci = JSON.stringify(filter)
  // Hasil terakhir langsung tampil saat kembali dari detail, supaya posisi gulir bisa dipulihkan.
  const [muat, setMuat] = useState<Muat | null>(() => (terakhir?.kunci === kunci ? terakhir : null))
  useEffect(() => { terakhir = muat }, [muat])
  const [coba, setCoba] = useState(0)
  const [sheet, setSheet] = useState(false)
  const atas = useRef<HTMLElement>(null)

  useEffect(() => {
    let batal = false // respons basi (filter sudah berganti) diabaikan
    const f = JSON.parse(kunci) as FilterPrimbon
    cariPrimbon(f).then(
      (hasil) => { if (!batal) setMuat({ kunci, hasil }) },
      (e: Error) => { if (!batal) setMuat((m) => ({ kunci, hasil: m?.hasil, error: e.message })) },
    )
    return () => { batal = true }
  }, [kunci, coba])

  const memuat = muat?.kunci !== kunci
  const hasil = muat?.hasil
  const chip = chipFilter(filter, hasil?.facet_total)
  const ubah = (f: Partial<FilterPrimbon>) => onFilter({ ...filter, hal: 1, ...f })
  const pindah = (hal: number) => {
    onFilter({ ...filter, hal })
    atas.current?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }
  const ubahItem = (p: Peraturan) =>
    setMuat((m) => m?.hasil ? { ...m, hasil: { ...m.hasil, items: m.hasil.items.map((x) => (x.uuid === p.uuid ? p : x)) } } : m)

  return (
    <div className="grid items-start gap-5 pt-4 lg:grid-cols-[260px_minmax(0,1fr)]">
      <aside className="card hidden justify-start gap-5 lg:sticky lg:top-4 lg:flex lg:max-h-[calc(100vh-2rem)] lg:overflow-y-auto" aria-label="Filter peraturan">
        <PanelFilter filter={filter} facet={hasil?.facet} onUbah={ubah} />
      </aside>

      <section ref={atas} className="flex min-w-0 scroll-mt-4 flex-col gap-3" aria-busy={memuat}>
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
          <KotakCari nilai={filter.q} onCari={(q) => ubah({ q })} />
          <div className="flex items-center gap-2">
            <Button type="button" variant="outline" className="lg:hidden" onClick={() => setSheet(true)}>
              <SlidersHorizontal /> Filter{jumlahFilter(filter) ? ` (${jumlahFilter(filter)})` : ''}
            </Button>
            <Select value={urutPrimbon(filter)} onValueChange={(u) => ubah({ urut: u as UrutPrimbon })}>
              <SelectTrigger className="h-10 w-full min-w-40 sm:w-auto" aria-label="Urutkan hasil"><SelectValue /></SelectTrigger>
              <SelectContent>
                {(filter.q.trim() ? ['relevan', 'terbaru', 'terlama', 'abjad'] as const : ['terbaru', 'terlama', 'abjad'] as const).map((u) => (
                  <SelectItem key={u} value={u}>{LABEL_URUT[u]}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>

        {chip.length > 0 && (
          <div className="flex flex-wrap items-center gap-1.5" aria-label="Filter aktif">
            {chip.map((c) => (
              <button key={c.kunci} type="button" onClick={() => onFilter(c.hapus(filter))} aria-label={`Hapus filter ${c.label}`}
                className="inline-flex min-h-8 items-center gap-1 rounded-full border border-[var(--border-strong)] bg-[var(--bg-card)] py-1 pr-2 pl-3 text-xs font-semibold hover:bg-[var(--bg-card-hover)]">
                {c.label} <X className="size-3.5" aria-hidden="true" />
              </button>
            ))}
            <button type="button" className="min-h-8 px-2 text-xs font-semibold text-[var(--link)] underline-offset-2 hover:underline" onClick={() => onFilter(tanpaFilter(filter))}>
              Hapus semua filter
            </button>
          </div>
        )}

        <div className="flex min-h-5 items-center gap-2 text-sm text-muted-foreground" aria-live="polite">
          {hasil && (
            <span>
              <strong className="tabular text-[var(--text-primary)]">{hasil.total.toLocaleString('id-ID')}</strong> peraturan
              {filter.q.trim() ? <> untuk "<span className="text-[var(--text-primary)]">{filter.q.trim()}</span>"</> : null}
            </span>
          )}
          {memuat && hasil && <span className="inline-flex items-center gap-1"><Loader2 className="size-3.5 motion-safe:animate-spin" aria-hidden="true" /> Memuat…</span>}
        </div>

        {muat?.error && (
          <div className="card flex-row flex-wrap items-center justify-between gap-3 text-sm" role="alert">
            <span className="text-[var(--text-danger)]">{muat.error}</span>
            <Button size="sm" variant="outline" onClick={() => setCoba((n) => n + 1)}>Coba lagi</Button>
          </div>
        )}
        {!hasil && !muat?.error && Array.from({ length: 6 }, (_, i) => <KerangkaKartu key={i} />)}
        {hasil && hasil.items.length === 0 && (
          <div className="card items-start justify-start gap-3 text-sm">
            <p className="font-semibold">Tidak ada peraturan yang cocok.</p>
            <p className="text-muted-foreground">
              {chip.length ? 'Coba lepas salah satu filter berikut, atau ganti kata kunci.' : 'Coba kata lain, mis. nomor peraturan (PMK 168 2023) atau topik (pph 21).'}
            </p>
            {chip.length > 0 && (
              <div className="flex flex-wrap gap-1.5">
                {chip.map((c) => <Button key={c.kunci} size="sm" variant="outline" onClick={() => onFilter(c.hapus(filter))}>Lepas {c.label}</Button>)}
              </div>
            )}
          </div>
        )}
        <div className={'flex flex-col gap-3 transition-opacity ' + (memuat && hasil ? 'opacity-60' : '')}>
          {hasil?.items.map((p) => <KartuPeraturan key={p.uuid} p={p} q={filter.q} onBuka={() => onBuka(p.uuid)} onUbah={ubahItem} />)}
        </div>
        {hasil && hasil.total_halaman > 1 && <Paginasi halaman={hasil.halaman} total={hasil.total_halaman} onPindah={pindah} />}
      </section>

      <Dialog open={sheet} onOpenChange={setSheet}>
        <DialogContent className="top-auto! bottom-0! left-0! flex! max-h-[85dvh]! w-full! max-w-none! translate-x-0! translate-y-0! flex-col gap-0 rounded-b-none! p-0! lg:hidden">
          <div className="border-b border-[var(--border-subtle)] px-5 pt-5 pb-3">
            <DialogTitle className="font-heading text-lg">Filter peraturan</DialogTitle>
            <DialogDescription className="sr-only">Hasil langsung berubah setiap kali filter diubah.</DialogDescription>
          </div>
          <div className="flex min-h-0 flex-1 flex-col gap-5 overflow-y-auto px-5 py-4">
            <PanelFilter filter={filter} facet={hasil?.facet} onUbah={ubah} />
          </div>
          <div className="flex gap-2 border-t border-[var(--border-subtle)] px-5 py-3">
            <Button variant="ghost" onClick={() => onFilter(tanpaFilter(filter))} disabled={!jumlahFilter(filter)}>Hapus filter</Button>
            <Button className="flex-1" onClick={() => setSheet(false)}>
              {hasil ? `Lihat ${hasil.total.toLocaleString('id-ID')} peraturan` : 'Lihat hasil'}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}

/** Kotak cari: hasil berubah 300 ms setelah berhenti mengetik; Enter langsung mencari. */
function KotakCari({ nilai, onCari }: { nilai: string; onCari: (q: string) => void }) {
  // Isian sendiri selama nilai dari luar tidak berubah; Back browser / hapus filter menimpanya.
  const [draf, setDraf] = useState({ teks: nilai, basis: nilai })
  const ketik = draf.basis === nilai ? draf.teks : nilai
  const terbaru = useRef({ nilai, onCari })
  useEffect(() => { terbaru.current = { nilai, onCari } })
  const kirim = (q: string) => {
    if (q.trim() !== terbaru.current.nilai.trim()) terbaru.current.onCari(q)
  }
  useEffect(() => {
    const t = setTimeout(() => {
      if (draf.basis === terbaru.current.nilai && draf.teks.trim() !== terbaru.current.nilai.trim()) terbaru.current.onCari(draf.teks)
    }, JEDA_KETIK)
    return () => clearTimeout(t)
  }, [draf])
  const setKetik = (teks: string) => setDraf({ teks, basis: nilai })
  return (
    <form role="search" className="relative min-w-0 flex-1" onSubmit={(e) => { e.preventDefault(); kirim(ketik) }}>
      <label htmlFor="primbon-cari" className="sr-only">Cari peraturan</label>
      <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
      <Input id="primbon-cari" type="search" className="h-10 pl-9" autoComplete="off" maxLength={200}
        placeholder="Cari nomor atau topik, mis. PMK 168 2023 atau pph 21"
        value={ketik} onChange={(e) => setKetik(e.target.value)} />
    </form>
  )
}

function PanelFilter({ filter, facet, onUbah }: {
  filter: FilterPrimbon
  facet?: FacetPrimbon
  onUbah: (f: Partial<FilterPrimbon>) => void
}) {
  return (
    <>
      <GrupCentang judul="Status" opsi={facet?.status} pilih={filter.status} onUbah={(status) => onUbah({ status })} />
      <GrupCentang judul="Kategori" opsi={facet?.kategori} pilih={filter.kategori} onUbah={(kategori) => onUbah({ kategori })} />
      <GrupCentang judul="Jenis peraturan" opsi={facet?.jenis} pilih={filter.jenis} onUbah={(jenis) => onUbah({ jenis })} />
      <RentangTahun dari={filter.dari} sampai={filter.sampai} min={facet?.tahun_min} max={facet?.tahun_max} onUbah={(dari, sampai) => onUbah({ dari, sampai })} />
    </>
  )
}

const MAKS_OPSI = 8

function GrupCentang({ judul, opsi, pilih, onUbah }: {
  judul: string
  opsi?: OpsiPrimbon[]
  pilih: string[]
  onUbah: (v: string[]) => void
}) {
  const [semua, setSemua] = useState(false)
  const daftar = opsi ?? []
  // pilihan yang dicentang selalu terlihat walau di luar 8 teratas
  const tampil = semua ? daftar : daftar.filter((o, i) => i < MAKS_OPSI || pilih.includes(o.kode))
  if (!opsi) return <Kerangka judul={judul} />
  return (
    <fieldset>
      <legend className="label-caps mb-2 text-muted-foreground">{judul}</legend>
      <div className="flex flex-col">
        {tampil.map((o) => {
          const aktif = pilih.includes(o.kode)
          const kosong = o.jumlah === 0 && !aktif
          return (
            <label key={o.kode}
              className={'flex min-h-9 cursor-pointer items-center gap-2.5 rounded-md px-1.5 text-sm hover:bg-[var(--bg-card-hover)] ' + (kosong ? 'opacity-50' : '')}>
              <input type="checkbox" className="size-4 shrink-0 cursor-pointer accent-[var(--primary)]"
                checked={aktif} disabled={kosong}
                onChange={(e) => onUbah(e.target.checked ? [...pilih, o.kode] : pilih.filter((k) => k !== o.kode))} />
              <span className="flex-1 leading-snug">{o.label}</span>
              <span className="tabular text-xs text-muted-foreground">{o.jumlah.toLocaleString('id-ID')}</span>
            </label>
          )
        })}
        {daftar.length > MAKS_OPSI && (
          <button type="button" className="mt-1 min-h-8 self-start px-1.5 text-xs font-semibold text-[var(--link)] underline-offset-2 hover:underline"
            aria-expanded={semua} onClick={() => setSemua(!semua)}>
            {semua ? 'Tampilkan lebih sedikit' : `Tampilkan semua (${daftar.length})`}
          </button>
        )}
        {daftar.length === 0 && <span className="px-1.5 text-xs text-muted-foreground">Tidak ada pilihan untuk hasil ini.</span>}
      </div>
    </fieldset>
  )
}

/** Tahun dari–sampai: diterapkan saat isian lengkap 4 angka (atau dikosongkan) dan masuk akal. */
function RentangTahun({ dari, sampai, min, max, onUbah }: {
  dari: string
  sampai: string
  min?: number
  max?: number
  onUbah: (dari: string, sampai: string) => void
}) {
  // Isian sendiri selama filter dari luar tidak berubah (pola yang sama dengan KotakCari).
  const [draf, setDraf] = useState({ d: dari, s: sampai, dari, sampai })
  const segar = draf.dari === dari && draf.sampai === sampai
  const d = segar ? draf.d : dari
  const s = segar ? draf.s : sampai
  const lengkap = (v: string) => v === '' || /^\d{4}$/.test(v)
  const salah = lengkap(d) && lengkap(s) && d && s && d > s ? 'Tahun "dari" lebih besar dari "sampai".' : ''
  const coba = (nd: string, ns: string) => {
    const terap = lengkap(nd) && lengkap(ns) && !(nd && ns && nd > ns) && (nd !== dari || ns !== sampai)
    setDraf(terap ? { d: nd, s: ns, dari: nd, sampai: ns } : { d: nd, s: ns, dari, sampai })
    if (terap) onUbah(nd, ns)
  }
  const bersih = (v: string) => v.replace(/\D/g, '').slice(0, 4)
  const contoh = min && max ? `${min}–${max}` : ''
  return (
    <fieldset>
      <legend className="label-caps mb-2 text-muted-foreground">Tahun</legend>
      <div className="flex items-center gap-2">
        <Input aria-label="Tahun dari" inputMode="numeric" placeholder="dari" value={d} aria-invalid={!!salah}
          onChange={(e) => coba(bersih(e.target.value), s)} />
        <span className="text-muted-foreground" aria-hidden="true">–</span>
        <Input aria-label="Tahun sampai" inputMode="numeric" placeholder="sampai" value={s} aria-invalid={!!salah}
          onChange={(e) => coba(d, bersih(e.target.value))} />
      </div>
      <p className={'mt-1.5 text-xs ' + (salah ? 'text-[var(--text-danger)]' : 'text-muted-foreground')} role={salah ? 'alert' : undefined}>
        {salah || (contoh && `Katalog mencakup ${contoh}.`)}
      </p>
    </fieldset>
  )
}

function Kerangka({ judul }: { judul: string }) {
  return (
    <div aria-hidden="true">
      <div className="label-caps mb-2 text-muted-foreground">{judul}</div>
      {Array.from({ length: 4 }, (_, i) => <div key={i} className="my-2 h-4 rounded bg-[color-mix(in_srgb,var(--text-primary)_8%,transparent)] motion-safe:animate-pulse" />)}
    </div>
  )
}

/** 1 2 3 … 76: halaman pertama, terakhir, dan di sekitar halaman aktif. */
function nomorHalaman(h: number, total: number): (number | '…')[] {
  const ada = new Set([1, total, h - 1, h, h + 1, h === 1 ? 3 : 0, h === total ? total - 2 : 0].filter((n) => n >= 1 && n <= total))
  const urut = [...ada].sort((a, b) => a - b)
  return urut.flatMap((n, i) => (i > 0 && n - urut[i - 1] > 1 ? ['…' as const, n] : [n]))
}

function Paginasi({ halaman, total, onPindah }: { halaman: number; total: number; onPindah: (h: number) => void }): ReactNode {
  return (
    <nav aria-label="Halaman katalog" className="mt-2 flex items-center justify-between gap-2 text-sm">
      <Button variant="outline" size="sm" disabled={halaman <= 1} onClick={() => onPindah(halaman - 1)}>← Sebelumnya</Button>
      <span className="tabular text-muted-foreground md:hidden">{halaman} / {total}</span>
      <div className="hidden items-center gap-1 md:flex">
        {nomorHalaman(halaman, total).map((n, i) => n === '…'
          ? <span key={'e' + i} className="px-1 text-muted-foreground" aria-hidden="true">…</span>
          : (
            <Button key={n} size="sm" variant={n === halaman ? 'outline' : 'ghost'} className="tabular min-w-8"
              aria-label={`Halaman ${n}`} aria-current={n === halaman ? 'page' : undefined} onClick={() => onPindah(n)}>
              {n}
            </Button>
          ))}
      </div>
      <Button variant="outline" size="sm" disabled={halaman >= total} onClick={() => onPindah(halaman + 1)}>Berikutnya →</Button>
    </nav>
  )
}
