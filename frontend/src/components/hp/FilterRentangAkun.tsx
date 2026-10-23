import { useState, type ReactNode } from 'react'
import { Button } from '@/components/ui/button'
import { LembarBawah } from '@/components/hp/LembarBawah'
import { pilihanCepat } from '@/lib/rentang'

export interface NilaiFilter { dari: string; sampai: string; akun?: string }

/**
 * Lembar filter HP untuk laporan per rentang tanggal & akun (Buku Besar, Jurnal Umum). Pilihan
 * baru berlaku setelah "Terapkan", supaya laporan tidak dimuat ulang tiap kali diketuk.
 */
export function FilterRentangAkun({ buka, onTutup, nilai, period, pilihan, labelSemuaAkun = 'Semua akun', onTerapkan, children }: {
  buka: boolean
  onTutup: () => void
  nilai: NilaiFilter
  period: string
  pilihan: { kode: string; nama: string }[]
  labelSemuaAkun?: string
  onTerapkan: (n: NilaiFilter) => void
  /** isian tambahan di bawah (mis. centang "tampilkan akun tanpa saldo") */
  children?: ReactNode
}) {
  const [draf, setDraf] = useState(nilai)
  const [bukaTerakhir, setBukaTerakhir] = useState(false)
  if (buka !== bukaTerakhir) { // dibuka ulang: mulai dari filter yang sedang berlaku
    setBukaTerakhir(buka)
    if (buka) setDraf(nilai)
  }
  const salah = draf.dari > draf.sampai
  return (
    <LembarBawah buka={buka} judul="Filter" onTutup={onTutup}>
      <div className="grid gap-3" style={{ paddingTop: 4 }}>
        <div className="grid gap-1.5">
          <span className="kpi-label">Rentang tanggal</span>
          <div className="flex flex-wrap gap-1.5">
            {pilihanCepat(period).map((p) => (
              <button key={p.label} type="button" className={'kepala-hp-chip' + (draf.dari === p.dari && draf.sampai === p.sampai ? ' aktif' : '')}
                aria-pressed={draf.dari === p.dari && draf.sampai === p.sampai} onClick={() => setDraf({ ...draf, dari: p.dari, sampai: p.sampai })}>{p.label}</button>
            ))}
          </div>
          <div className="grid grid-cols-2 gap-2">
            <label className="grid gap-1"><span className="text-xs text-muted-foreground">Dari</span>
              <input className="input-field" type="date" value={draf.dari} onChange={(e) => e.target.value && setDraf({ ...draf, dari: e.target.value })} /></label>
            <label className="grid gap-1"><span className="text-xs text-muted-foreground">Sampai</span>
              <input className="input-field" type="date" value={draf.sampai} onChange={(e) => e.target.value && setDraf({ ...draf, sampai: e.target.value })} /></label>
          </div>
          {salah && <span className="text-xs" style={{ color: 'var(--text-danger)' }}>Tanggal sampai tidak boleh sebelum tanggal dari.</span>}
        </div>
        <label className="grid gap-1.5"><span className="kpi-label">Akun</span>
          <select className="input-field" value={draf.akun ?? ''} onChange={(e) => setDraf({ ...draf, akun: e.target.value || undefined })}>
            <option value="">{labelSemuaAkun}</option>
            {pilihan.map((a) => <option key={a.kode} value={a.kode}>{a.kode} · {a.nama}</option>)}
          </select></label>
        {children}
        <Button disabled={salah} onClick={() => { onTerapkan(draf); onTutup() }}>Terapkan</Button>
      </div>
    </LembarBawah>
  )
}
