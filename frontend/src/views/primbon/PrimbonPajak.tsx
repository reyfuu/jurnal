import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState, type KeyboardEvent } from 'react'
import { Search } from 'lucide-react'
import { PageHeader } from '@/components/PageHeader'
import { Input } from '@/components/ui/input'
import { listBookmarkPrimbon, listCatatanPrimbon, type FilterPrimbon, type Peraturan } from '@/lib/api'
import { pecahRute, tulisRute } from '@/lib/rute'
import { Detail } from './Detail'
import { formatWaktu } from './bantu'
import { KartuPeraturan, KerangkaKartu } from './KartuPeraturan'
import { Katalog } from './Katalog'
import { bacaRutePrimbon, tulisRutePrimbon, type RutePrimbon, type TabPrimbon } from './rute'

const TAB: { id: TabPrimbon; label: string }[] = [
  { id: 'katalog', label: 'Katalog' },
  { id: 'bookmark', label: 'Bookmark' },
  { id: 'catatan', label: 'Catatan' },
]

function dariHash(): RutePrimbon {
  const r = pecahRute(window.location.hash)
  return bacaRutePrimbon(r?.view === 'primbon-pajak' ? r.sub : '', r?.view === 'primbon-pajak' ? r.params : new URLSearchParams())
}

/** Aturan Pajak: katalog peraturan resmi, bookmark, dan catatan pribadi (khusus pelanggan).
 *  Tab, filter, dan peraturan yang dibuka disimpan di URL: Back browser kembali ke hasil yang
 *  sama dan tautan detail bisa dibagikan. */
export function PrimbonPajak() {
  const [rute, setRute] = useState<RutePrimbon>(dariHash)
  const scrollDaftar = useRef(0)
  const kartuAsal = useRef<string | null>(null)

  useEffect(() => {
    const ubah = () => setRute(dariHash())
    window.addEventListener('popstate', ubah)
    return () => window.removeEventListener('popstate', ubah)
  }, [])

  /** push: langkah riwayat baru (buka detail, ganti tab); replace: ketikan & filter. */
  const pergi = useCallback((r: RutePrimbon, cara: 'push' | 'replace', state: object | null = null) => {
    const h = tulisRute('primbon-pajak', pecahRute(window.location.hash)?.periode ?? '', tulisRutePrimbon(r))
    if (h !== window.location.hash) {
      if (cara === 'push') history.pushState(state, '', h)
      else history.replaceState(history.state, '', h)
    }
    setRute(r)
  }, [])

  const buka = (uuid: string) => {
    scrollDaftar.current = window.scrollY
    kartuAsal.current = uuid
    pergi({ ...rute, buka: uuid }, 'push', { primbonDariDaftar: true })
  }
  const kembali = () => {
    if ((history.state as { primbonDariDaftar?: boolean } | null)?.primbonDariDaftar) history.back()
    else pergi({ ...rute, buka: null }, 'replace') // dibuka dari tautan: tidak ada daftar sebelumnya
  }

  // Kembali dari detail: posisi gulir & fokus kembali ke kartu yang tadi dibuka.
  const tadinyaDetail = useRef(!!rute.buka)
  useLayoutEffect(() => {
    if (rute.buka) {
      tadinyaDetail.current = true
      return
    }
    if (!tadinyaDetail.current) return
    tadinyaDetail.current = false
    window.scrollTo({ top: scrollDaftar.current })
    const id = kartuAsal.current
    if (id) requestAnimationFrame(() => document.querySelector<HTMLElement>(`[data-kartu="${CSS.escape(id)}"] [data-buka]`)?.focus({ preventScroll: true }))
  }, [rute.buka])

  if (rute.buka) {
    return <Detail key={rute.buka} uuid={rute.buka} kembali={rute.tab === 'katalog' ? 'Kembali ke hasil' : `Kembali ke ${TAB.find((t) => t.id === rute.tab)!.label}`} onKembali={kembali} />
  }

  const pilihTab = (tab: TabPrimbon) => pergi({ ...rute, tab }, 'push')
  // Panah kiri/kanan, Home, End berpindah tab (pola tablist WAI-ARIA).
  const keyTab = (e: KeyboardEvent<HTMLDivElement>) => {
    const i = TAB.findIndex((t) => t.id === rute.tab)
    const j = { ArrowRight: i + 1, ArrowLeft: i - 1, Home: 0, End: TAB.length - 1 }[e.key]
    if (j === undefined) return
    e.preventDefault()
    const t = TAB[(j + TAB.length) % TAB.length]
    pilihTab(t.id)
    requestAnimationFrame(() => document.getElementById(`primbon-tab-${t.id}`)?.focus())
  }

  return (
    <>
      <PageHeader title="Aturan Pajak"
        subtitle="Katalog peraturan perpajakan resmi: cari, saring, baca naskah aslinya, lalu simpan bookmark dan catatan pribadi." />
      <div className="tabbar mb-0" role="tablist" aria-label="Aturan Pajak" onKeyDown={keyTab}>
        {TAB.map((t) => (
          <button key={t.id} id={`primbon-tab-${t.id}`} role="tab" type="button" className="tab-btn"
            aria-selected={rute.tab === t.id} aria-controls={`primbon-panel-${t.id}`} tabIndex={rute.tab === t.id ? 0 : -1}
            onClick={() => pilihTab(t.id)}>
            {t.label}
          </button>
        ))}
      </div>
      <div id={`primbon-panel-${rute.tab}`} role="tabpanel" aria-labelledby={`primbon-tab-${rute.tab}`}>
        {rute.tab === 'katalog'
          ? <Katalog filter={rute.filter} onFilter={(filter: FilterPrimbon) => pergi({ ...rute, filter }, 'replace')} onBuka={buka} />
          : <Simpanan key={rute.tab} jenis={rute.tab} onBuka={buka} />}
      </div>
    </>
  )
}

type UrutSimpanan = 'terbaru' | 'abjad'

function Simpanan({ jenis, onBuka }: { jenis: 'bookmark' | 'catatan'; onBuka: (uuid: string) => void }) {
  const [rows, setRows] = useState<Peraturan[] | null>(null)
  const [error, setError] = useState('')
  const [coba, setCoba] = useState(0)
  const [saring, setSaring] = useState('')
  const [urut, setUrut] = useState<UrutSimpanan>('terbaru')
  useEffect(() => {
    let batal = false
    ;(jenis === 'bookmark' ? listBookmarkPrimbon() : listCatatanPrimbon()).then(
      (r) => { if (!batal) { setRows(r); setError('') } },
      (e: Error) => { if (!batal) setError(e.message) },
    )
    return () => { batal = true }
  }, [jenis, coba])
  const ubah = (p: Peraturan) => setRows((r) => r && r.map((x) => (x.uuid === p.uuid ? p : x)))

  const tampil = useMemo(() => {
    const kata = saring.toLowerCase().split(/\s+/).filter(Boolean)
    const cocok = (rows ?? []).filter((p) => {
      const teks = `${p.kode} ${p.judul} ${p.catatan?.isi ?? ''}`.toLowerCase()
      return kata.every((k) => teks.includes(k))
    })
    // urutan server sudah "terbaru disimpan / diubah"
    return urut === 'abjad' ? [...cocok].sort((a, b) => a.judul.localeCompare(b.judul, 'id')) : cocok
  }, [rows, saring, urut])

  return (
    <div className="flex max-w-[880px] flex-col gap-3 pt-4">
      {rows && rows.length > 0 && (
        <div className="flex flex-wrap items-center gap-2">
          <div className="relative min-w-0 flex-1 basis-60">
            <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
            <Input type="search" className="pl-9" aria-label={`Saring ${jenis}`} placeholder={jenis === 'bookmark' ? 'Saring bookmark…' : 'Saring catatan…'}
              value={saring} onChange={(e) => setSaring(e.target.value)} />
          </div>
          <div className="segmented" role="group" aria-label="Urutkan">
            {(['terbaru', 'abjad'] as const).map((u) => (
              <button key={u} type="button" aria-pressed={urut === u} className="seg-btn" onClick={() => setUrut(u)}>
                {u === 'terbaru' ? (jenis === 'bookmark' ? 'Terbaru disimpan' : 'Terbaru diubah') : 'A–Z'}
              </button>
            ))}
          </div>
        </div>
      )}
      {error && (
        <div className="card flex-row items-center justify-between gap-3 text-sm" role="alert">
          <span className="text-[var(--text-danger)]">{error}</span>
          <button type="button" className="font-semibold underline underline-offset-2" onClick={() => { setError(''); setCoba((n) => n + 1) }}>Coba lagi</button>
        </div>
      )}
      {!rows && !error && Array.from({ length: 3 }, (_, i) => <KerangkaKartu key={i} />)}
      {rows?.length === 0 && (
        <div className="card text-sm leading-relaxed text-muted-foreground">
          {jenis === 'bookmark'
            ? 'Belum ada bookmark. Tekan ikon bookmark di katalog untuk menyimpan peraturan yang sering Anda buka.'
            : 'Belum ada catatan. Buka sebuah peraturan, lalu tulis di kotak "Catatan Anda".'}
        </div>
      )}
      {rows && rows.length > 0 && tampil.length === 0 && (
        <div className="card text-sm text-muted-foreground">Tidak ada {jenis} yang cocok dengan "{saring.trim()}".</div>
      )}
      {tampil.map((p) => (
        <KartuPeraturan key={p.uuid} p={p} q={saring} onBuka={() => onBuka(p.uuid)} onUbah={ubah}>
          {jenis === 'catatan' && p.catatan && (
            <div className="mt-3">
              <p className="line-clamp-3 border-l-2 border-[var(--border-strong)] pl-3 text-sm leading-relaxed whitespace-pre-wrap">
                {p.catatan.isi}
              </p>
              <div className="mt-1 text-xs text-muted-foreground">Diubah {formatWaktu(p.catatan.diubah)}</div>
            </div>
          )}
        </KartuPeraturan>
      ))}
    </div>
  )
}
