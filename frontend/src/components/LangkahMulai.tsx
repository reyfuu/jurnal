import { useEffect, useState } from 'react'
import { CheckCircle2, Circle, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { getLangkahMulai, type LangkahMulai as Langkah, type View } from '@/lib/api'

// Panduan "Langkah memulai" di Dashboard: urutan kerja untuk pengguna baru, dicentang otomatis
// dari datanya. Disembunyikan bila semua selesai atau pengguna menutupnya (per browser).

const KUNCI = 'langkah-mulai-ditutup'

function sudahDitutup() {
  try { return localStorage.getItem(KUNCI) === '1' } catch { return false }
}

export function LangkahMulaiCard({ onBuka }: { onBuka: (v: View, aksi?: string) => void }) {
  const [langkah, setLangkah] = useState<Langkah[] | null>(null)
  const [tutup, setTutup] = useState(sudahDitutup)
  useEffect(() => { getLangkahMulai().then(setLangkah, () => setLangkah([])) }, [])
  if (tutup || !langkah || langkah.length === 0) return null
  const selesai = langkah.filter((l) => l.selesai).length
  if (selesai === langkah.length) return null
  const berikut = langkah.find((l) => !l.selesai)
  return (
    <div className="card" style={{ padding: 20, marginBottom: 18, borderColor: 'var(--accent-blue)' }}>
      <div className="flex items-start gap-3">
        <div style={{ flex: 1 }}>
          <div className="chart-title">Langkah memulai</div>
          <div className="chart-subtitle">{selesai} dari {langkah.length} selesai. Ikuti urutannya supaya laporan & pajak langsung akurat.</div>
        </div>
        <Button variant="ghost" size="icon-sm" aria-label="Tutup panduan" onClick={() => { try { localStorage.setItem(KUNCI, '1') } catch { /* abaikan */ } setTutup(true) }}><X /></Button>
      </div>
      <div style={{ height: 6, borderRadius: 3, background: 'var(--bg-raised)', margin: '12px 0 14px' }}>
        <div style={{ height: 6, borderRadius: 3, width: `${(selesai / langkah.length) * 100}%`, background: 'var(--accent-green)' }} />
      </div>
      <ol className="grid gap-2" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(340px, 1fr))' }}>
        {langkah.map((l, i) => (
          <li key={l.kode} className="flex items-start gap-3 rounded-lg border px-3 py-2.5" style={{ borderColor: l === berikut ? 'var(--accent-blue)' : 'var(--border-shell)', opacity: l.selesai ? 0.7 : 1 }}>
            {l.selesai ? <CheckCircle2 size={20} style={{ color: 'var(--accent-green)', flexShrink: 0 }} aria-label="selesai" /> : <Circle size={20} style={{ color: 'var(--text-muted)', flexShrink: 0 }} aria-label="belum" />}
            <div style={{ flex: 1 }}>
              <div className="text-sm font-semibold">{i + 1}. {l.judul}</div>
              <div className="text-xs text-muted-foreground">{l.penjelasan}</div>
            </div>
            {!l.selesai && <Button size="sm" variant={l === berikut ? 'default' : 'outline'} onClick={() => onBuka(l.view, l.aksi)}>Kerjakan</Button>}
          </li>
        ))}
      </ol>
    </div>
  )
}
