import { useEffect, useState } from 'react'
import { PageHeader } from '@/components/PageHeader'
import { FILTER_PRIMBON_AWAL, listBookmarkPrimbon, listCatatanPrimbon, type Peraturan } from '@/lib/api'
import { Detail } from './Detail'
import { formatWaktu } from './bantu'
import { KartuPeraturan } from './KartuPeraturan'
import { Katalog } from './Katalog'

type Tab = 'katalog' | 'bookmark' | 'catatan'
const TAB: { id: Tab; label: string }[] = [
  { id: 'katalog', label: 'Katalog' },
  { id: 'bookmark', label: 'Bookmark' },
  { id: 'catatan', label: 'Catatan' },
]

/** Primbon Pajak: katalog peraturan resmi, bookmark, dan catatan pribadi (khusus pelanggan). */
export function PrimbonPajak() {
  const [tab, setTab] = useState<Tab>('katalog')
  // Filter & halaman disimpan di sini supaya kembali dari detail tetap di hasil yang sama.
  const [filter, setFilter] = useState(FILTER_PRIMBON_AWAL)
  const [buka, setBuka] = useState<string | null>(null)

  if (buka) {
    return <Detail key={buka} uuid={buka} kembali={TAB.find((t) => t.id === tab)!.label} onKembali={() => setBuka(null)} />
  }
  return (
    <>
      <PageHeader title="Primbon Pajak"
        subtitle="Katalog peraturan perpajakan resmi: cari, saring, baca naskah aslinya, lalu simpan bookmark dan catatan pribadi." />
      <div className="tabbar" role="tablist" aria-label="Primbon Pajak" style={{ marginBottom: 0 }}>
        {TAB.map((t) => (
          <button key={t.id} role="tab" className="tab-btn" aria-selected={tab === t.id} onClick={() => setTab(t.id)}>{t.label}</button>
        ))}
      </div>
      {tab === 'katalog'
        ? <Katalog filter={filter} onFilter={setFilter} onBuka={setBuka} />
        : <Simpanan key={tab} jenis={tab} onBuka={setBuka} />}
    </>
  )
}

function Simpanan({ jenis, onBuka }: { jenis: 'bookmark' | 'catatan'; onBuka: (uuid: string) => void }) {
  const [rows, setRows] = useState<Peraturan[] | null>(null)
  const [error, setError] = useState('')
  useEffect(() => {
    (jenis === 'bookmark' ? listBookmarkPrimbon() : listCatatanPrimbon()).then(setRows, (e: Error) => setError(e.message))
  }, [jenis])
  const ubah = (p: Peraturan) => setRows((r) => r && r.map((x) => (x.uuid === p.uuid ? p : x)))

  return (
    <div className="flex flex-col gap-3" style={{ maxWidth: 880 }}>
      {error && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start' }}>{error}</div>}
      {!rows && !error && <p className="workspace-subtitle">Memuat…</p>}
      {rows?.length === 0 && (
        <div className="card text-sm leading-relaxed text-muted-foreground" style={{ padding: 20 }}>
          {jenis === 'bookmark'
            ? 'Belum ada bookmark. Tekan ikon bookmark di katalog untuk menyimpan peraturan yang sering Anda buka.'
            : 'Belum ada catatan. Buka sebuah peraturan, lalu tulis di kotak "Catatan Anda".'}
        </div>
      )}
      {rows?.map((p) => (
        <KartuPeraturan key={p.uuid} p={p} onBuka={() => onBuka(p.uuid)} onUbah={ubah}>
          {jenis === 'catatan' && p.catatan && (
            <div className="mt-3">
              <p className="line-clamp-3 text-sm leading-relaxed whitespace-pre-wrap" style={{ borderLeft: '2px solid var(--border-strong)', paddingLeft: 12 }}>
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
