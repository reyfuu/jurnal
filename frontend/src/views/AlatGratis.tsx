import { useEffect, useState } from 'react'
import { getKalkulator, type BusinessProfile, type KalkulatorMaster } from '@/lib/api'
import { FormCekPKP, FormPotong, FormPPhBadan } from '@/views/KalkulatorPajak'
import { PerencanaanPajak } from '@/views/PerencanaanPajak'

const TABS = [
  { id: 'pkp', label: 'Cek Ambang PKP' },
  { id: 'pph22', label: 'Kalkulator PPh 22' },
  { id: '31e', label: 'Simulasi Pasal 31E' },
  { id: 'bentuk', label: 'Bentuk Usaha Paling Hemat' },
] as const

type Tab = (typeof TABS)[number]['id']

/** Alat pajak gratis untuk usaha yang belum berlangganan (server: lockedWrite membolehkannya). */
export function AlatGratis({ profile, ptkpCodes, onLangganan }: { profile: BusinessProfile | null; ptkpCodes: string[]; onLangganan: () => void }) {
  const [m, setM] = useState<KalkulatorMaster | null>(null)
  const [tab, setTab] = useState<Tab>('pkp')
  const [error, setError] = useState('')

  useEffect(() => { getKalkulator().then(setM, (e) => setError((e as Error).message)) }, [])
  const pph22 = m?.jenis.find((j) => j.id === 'pph22')

  return (
    <div className="alat-gratis grid gap-4">
      <div>
        <h1 className="text-xl font-bold tracking-tight">Alat pajak gratis</h1>
        <p className="mt-1 text-sm text-muted-foreground">Hitung sendiri tanpa berlangganan. Hasilnya simulasi, bukan pengganti konsultan pajak.</p>
      </div>

      <div className="tabbar" role="tablist" aria-label="Alat pajak gratis">
        {TABS.map((t) => (
          <button key={t.id} role="tab" className="tab-btn" aria-selected={t.id === tab} onClick={() => setTab(t.id)}>{t.label}</button>
        ))}
      </div>

      {error && <p className="pill-status pill-red" role="alert">{error}</p>}
      {tab === 'pkp' && <FormCekPKP />}
      {tab === 'pph22' && pph22 && <FormPotong jenis={pph22} />}
      {tab === '31e' && m && <FormPPhBadan m={m} />}
      {tab === 'bentuk' && <PerencanaanPajak profile={profile} ptkpCodes={ptkpCodes} />}

      <div className="note-box flex flex-wrap items-center justify-between gap-3">
        <span>Mau pajak bulanan, laporan keuangan, dan pengingat setor terisi otomatis dari transaksi Anda?</span>
        <button type="button" className="btn-cta" style={{ padding: '8px 18px', fontSize: 13 }} onClick={onLangganan}>Pilih paket</button>
      </div>
    </div>
  )
}
