import { useState, type FormEvent } from 'react'
import { BrandMark } from '@/components/Sidebar'
import { PageHeader } from '@/components/PageHeader'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { saveBusiness, type BusinessInfo, type BusinessProfile, type JenisUsaha, type BentukUsaha } from '@/lib/api'
import { DEFAULT_ANSWERS, QUESTIONS, answered, steps, summary, toProfile, type Answers } from '@/lib/onboarding'

const EMPTY: BusinessProfile = { nama_usaha: '', jenis_usaha: '' as JenisUsaha, bentuk_usaha: '' as BentukUsaha, ptkp: '', pkp: false, metode_persediaan: 'periodik' }

function ChoiceGrid({ name, options, value, onChange }: {
  name: string
  options: BusinessInfo['options']['jenis_usaha']
  value: string
  onChange: (v: string) => void
}) {
  return (
    <div role="radiogroup" aria-label={name} style={{ display: 'grid', gap: 8, gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))' }}>
      {options.map((o) => {
        const on = o.value === value
        return (
          <button
            key={o.value}
            type="button"
            role="radio"
            aria-checked={on}
            onClick={() => onChange(o.value)}
            className="card"
            style={{ textAlign: 'left', padding: 14, cursor: 'pointer', borderColor: on ? '#eab308' : undefined, background: on ? 'rgba(234,179,8,0.08)' : '#101318' }}
          >
            <div style={{ fontWeight: 700, fontSize: 14 }}>{o.label}</div>
            {o.hint && <div style={{ fontSize: 12, color: 'var(--text-muted)', marginTop: 4, lineHeight: 1.45 }}>{o.hint}</div>}
          </button>
        )
      })}
    </div>
  )
}

export function ProfilForm({ info, onSaved, submitLabel }: { info: BusinessInfo; onSaved: (p: BusinessProfile) => void; submitLabel: string }) {
  const [p, setP] = useState<BusinessProfile>(info.profile ?? EMPTY)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const set = (patch: Partial<BusinessProfile>) => setP((x) => ({ ...x, ...patch }))

  async function submit(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      onSaved(await saveBusiness(p))
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <form onSubmit={submit} className="grid gap-5">
      <div className="grid gap-1.5">
        <Label htmlFor="biz-nama">Nama usaha</Label>
        <Input id="biz-nama" required value={p.nama_usaha} onChange={(e) => set({ nama_usaha: e.target.value })} placeholder="mis. Kedai Kopi Senja" />
      </div>
      <div className="grid gap-1.5">
        <Label>Jenis usaha</Label>
        <ChoiceGrid name="Jenis usaha" options={info.options.jenis_usaha} value={p.jenis_usaha} onChange={(v) => set({ jenis_usaha: v as JenisUsaha })} />
        <p className="text-xs text-muted-foreground">Menentukan susunan akun (CoA) dan format laporan keuangan Anda.</p>
      </div>
      {(p.jenis_usaha === 'perdagangan' || p.jenis_usaha === 'manufaktur') && (
        <div className="grid gap-1.5">
          <Label>Metode pencatatan persediaan</Label>
          <ChoiceGrid
            name="Metode persediaan"
            options={info.options.metode_persediaan}
            value={p.metode_persediaan || 'periodik'}
            onChange={(v) => set({ metode_persediaan: v as BusinessProfile['metode_persediaan'] })}
          />
          <p className="text-xs text-muted-foreground">Belum yakin? Pilih Periodik, yang paling umum dipakai UMKM.</p>
        </div>
      )}
      <div className="grid gap-1.5">
        <Label>Bentuk usaha</Label>
        <ChoiceGrid name="Bentuk usaha" options={info.options.bentuk_usaha} value={p.bentuk_usaha} onChange={(v) => set({ bentuk_usaha: v as BentukUsaha })} />
      </div>
      {p.bentuk_usaha === 'op' && (
        <div className="grid gap-1.5">
          <Label htmlFor="biz-ptkp">Status PTKP pemilik</Label>
          <Select value={p.ptkp} onValueChange={(v) => set({ ptkp: v })}>
            <SelectTrigger id="biz-ptkp" className="w-48"><SelectValue placeholder="Pilih status" /></SelectTrigger>
            <SelectContent>
              {info.options.ptkp.map((c) => <SelectItem key={c} value={c}>{c}</SelectItem>)}
            </SelectContent>
          </Select>
          <p className="text-xs text-muted-foreground">TK = belum kawin, K = kawin, angka = jumlah tanggungan. K/I = penghasilan suami-istri digabung.</p>
        </div>
      )}
      <label className="flex items-center gap-3 text-sm" htmlFor="biz-pkp">
        <Switch id="biz-pkp" checked={p.pkp} onCheckedChange={(v) => set({ pkp: v })} />
        Sudah dikukuhkan sebagai PKP (memungut PPN)
      </label>
      {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
      <Button type="submit" size="lg" className="rounded-full font-bold" disabled={busy}>{busy ? 'Menyimpan…' : submitLabel}</Button>
    </form>
  )
}

// Wizard onboarding: satu pertanyaan bahasa awam per layar, ringkasan di akhir.
function OnboardingWizard({ onSaved }: { onSaved: (p: BusinessProfile) => void }) {
  const [a, setA] = useState<Answers>(DEFAULT_ANSWERS)
  const [i, setI] = useState(0)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const list = steps(a)
  const step = list[Math.min(i, list.length - 1)]
  const total = list.length - 1

  async function submit(e: FormEvent) {
    e.preventDefault()
    if (step !== 'ringkasan') {
      if (answered(step, a)) setI(i + 1)
      return
    }
    setBusy(true)
    setError('')
    try {
      onSaved(await saveBusiness(toProfile(a)))
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  const back = i > 0 && <Button type="button" variant="ghost" onClick={() => setI(i - 1)}>Kembali</Button>

  if (step === 'ringkasan') {
    return (
      <form onSubmit={submit} className="grid gap-4">
        <div>
          <h2 className="text-lg font-bold">Ini yang kami siapkan untuk {a.nama?.trim()}</h2>
          <p className="mt-1 text-sm text-muted-foreground">Semua bisa diubah nanti di menu Profil Usaha.</p>
        </div>
        <dl className="grid gap-3">
          {summary(toProfile(a)).map((r) => (
            <div key={r.judul} className="card" style={{ padding: 14, background: '#101318' }}>
              <dt className="text-xs font-bold uppercase tracking-wide text-muted-foreground">{r.judul}</dt>
              <dd className="mt-1 text-sm leading-relaxed">{r.isi}</dd>
            </div>
          ))}
        </dl>
        {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
        <div className="flex items-center justify-between gap-3">
          <Button type="button" variant="ghost" onClick={() => setI(0)}>Ubah jawaban</Button>
          <Button type="submit" size="lg" className="rounded-full font-bold" disabled={busy}>{busy ? 'Menyimpan…' : 'Simpan & mulai'}</Button>
        </div>
      </form>
    )
  }

  const q = QUESTIONS[step]
  return (
    <form onSubmit={submit} className="grid gap-4">
      <p className="text-xs font-semibold text-muted-foreground">Langkah {i + 1} dari {total}</p>
      <h2 className="text-lg font-bold">{q.judul}</h2>
      {q.options ? (
        <ChoiceGrid
          name={q.judul}
          options={q.options}
          value={a[step] ?? ''}
          onChange={(v) => { setA((x) => ({ ...x, [step]: v })); setI(i + 1) }}
        />
      ) : (
        <Input autoFocus required aria-label={q.judul} value={a[step] ?? ''} onChange={(e) => setA((x) => ({ ...x, [step]: e.target.value }))} placeholder="mis. Kedai Kopi Senja" />
      )}
      <p className="text-xs text-muted-foreground"><span className="font-semibold">Kenapa kami tanya ini?</span> {q.alasan}</p>
      <div className="flex items-center justify-between gap-3">
        <span>{back}</span>
        <Button type="submit" className="rounded-full font-bold" disabled={!answered(step, a)}>Lanjut</Button>
      </div>
    </form>
  )
}

// Onboarding: layar pertama Owner sebelum workspace bila profil usaha belum diisi.
export function Onboarding({ onSaved, canEdit, onLogout }: { onSaved: (p: BusinessProfile) => void; canEdit: boolean; onLogout: () => void }) {
  return (
    <main className="flex min-h-screen items-center justify-center px-4 py-10">
      <div className="w-full max-w-2xl">
        <div className="mb-6 flex items-center gap-2.5">
          <BrandMark size={28} />
          <div>
            <div className="text-base font-bold tracking-tight">Laporan Bisnis</div>
            <div className="brand-sub">by Akademi Pajak</div>
          </div>
        </div>
        <div className="card" style={{ padding: 24 }}>
          <h1 className="text-xl font-bold">Kenalkan usaha Anda dulu</h1>
          <p className="mt-1 mb-5 text-sm text-muted-foreground">
            Beberapa pertanyaan singkat, sekali saja. Tidak perlu paham akuntansi atau pajak: jawab sesuai keadaan usaha Anda, kami yang menyiapkan sisanya.
          </p>
          {canEdit ? (
            <OnboardingWizard onSaved={onSaved} />
          ) : (
            <p className="text-sm">Owner usaha perlu melengkapi profil usaha terlebih dahulu. Hubungi Owner Anda, lalu muat ulang halaman ini.</p>
          )}
        </div>
        <button type="button" onClick={onLogout} className="mt-4 text-xs text-muted-foreground underline">Keluar</button>
      </div>
    </main>
  )
}

export function ProfilUsaha({ info, onSaved }: { info: BusinessInfo; onSaved: (p: BusinessProfile) => void }) {
  const [saved, setSaved] = useState(false)
  return (
    <>
      <PageHeader title="Profil Usaha" subtitle="Jenis & bentuk usaha menentukan CoA, format laporan, dan simulasi pajak" />
      <div className="card" style={{ padding: 22, maxWidth: 760 }}>
        <ProfilForm info={info} onSaved={(p) => { setSaved(true); onSaved(p) }} submitLabel="Simpan perubahan" />
        {saved && <p className="mt-3 text-sm" style={{ color: '#10b981' }} role="status">Profil usaha tersimpan.</p>}
      </div>
    </>
  )
}
