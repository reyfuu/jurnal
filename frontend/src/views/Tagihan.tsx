import { useState, type ReactNode } from 'react'
import type { View } from '@/lib/api'
import { DaftarPelanggan } from '@/views/Pelanggan'
import { RiwayatPihakDialog } from '@/views/RiwayatPihak'

// Satu halaman Tagihan (checklist tester No 13): Invoice untuk menerbitkan tagihan, Piutang &
// Utang untuk memantau & menagih, Pelanggan untuk data pelanggan. Invoice dan Piutang & Utang
// tetap view (izin & alamat) sendiri; tab Pelanggan hanya di halaman ini.

const TAB: { id: View; label: string }[] = [
  { id: 'invoice', label: 'Invoice' },
  { id: 'piutang-utang', label: 'Piutang & Utang' },
]

export function TagihanTabs({ view, boleh, onPilih, children }: { view: View; boleh: (v: View) => boolean; onPilih: (v: View) => void; children: ReactNode }) {
  const [pelanggan, setPelanggan] = useState(false)
  const [riwayat, setRiwayat] = useState<string | null>(null)
  const tabs = TAB.filter((t) => boleh(t.id))
  return (
    <>
      <div className="tabbar" role="tablist" aria-label="Tagihan" style={{ marginBottom: 14 }}>
        {tabs.map((t) => (
          <button key={t.id} role="tab" className="tab-btn" aria-selected={!pelanggan && t.id === view}
            onClick={() => { setPelanggan(false); if (t.id !== view) onPilih(t.id) }}>{t.label}</button>
        ))}
        {boleh('invoice') && <button role="tab" className="tab-btn" aria-selected={pelanggan} onClick={() => setPelanggan(true)}>Pelanggan</button>}
      </div>
      {pelanggan ? <DaftarPelanggan onRiwayat={(p) => setRiwayat(p.nama)} /> : children}
      <RiwayatPihakDialog nama={riwayat} onTutup={() => setRiwayat(null)} />
    </>
  )
}
