import { useEffect, useId, useRef, useState, type FormEvent } from 'react'
import { ArrowLeft, ArrowRight, Check } from 'lucide-react'
import { BrandMark } from '@/components/Sidebar'
import { CekAwalCard } from '@/components/CekAwalCard'
import { PageHeader } from '@/components/PageHeader'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { saveBusiness, type BusinessInfo, type BusinessProfile, type JenisUsaha, type BentukUsaha } from '@/lib/api'
import { formatRp } from '@/lib/format'
import { DEFAULT_ANSWERS, LOKASI_FIELDS, QUESTIONS, answered, parseRp, steps, summary, toProfile, type Answers } from '@/lib/onboarding'

const EMPTY: BusinessProfile = { nama_usaha: '', jenis_usaha: '' as JenisUsaha, bentuk_usaha: '' as BentukUsaha, ptkp: '', pkp: false, metode_persediaan: 'periodik' }

function ChoiceGrid({ name, options, value, onChange }: {
  name: string
  options: BusinessInfo['options']['jenis_usaha']
  value: string
  onChange: (v: string) => void
}) {
  const groupId = useId()
  return (
    <div role="radiogroup" aria-label={name} className="profile-choices">
      {options.map((o) => {
        const on = o.value === value
        return (
          <label key={o.value} className="profile-choice" data-selected={on}>
            <input
              type="radio"
              name={groupId}
              value={o.value}
              checked={on}
              onChange={() => onChange(o.value)}
              aria-labelledby={`${groupId}-${o.value}-label`}
              aria-describedby={o.hint ? `${groupId}-${o.value}-hint` : undefined}
            />
            <span>
              <span id={`${groupId}-${o.value}-label`} className="profile-choice-label">{o.label}</span>
              {o.hint && <span id={`${groupId}-${o.value}-hint`} className="profile-choice-hint">{o.hint}</span>}
            </span>
          </label>
        )
      })}
    </div>
  )
}

type Lokasi = Partial<Record<(typeof LOKASI_FIELDS)[number], string>>

// LokasiFields: alamat, kecamatan, kota, provinsi — dipakai wizard onboarding dan Profil Usaha.
function LokasiFields({ v, set, provinsi, required = false, big = false }: {
  v: Lokasi
  set: (patch: Lokasi) => void
  provinsi: string[]
  required?: boolean
  big?: boolean
}) {
  const cls = big ? 'h-12 px-4 md:text-base' : ''
  return (
    <div className="grid gap-3 sm:grid-cols-2">
      <div className="grid gap-1.5 sm:col-span-2">
        <Label htmlFor="lok-alamat">Alamat usaha</Label>
        <Input id="lok-alamat" className={cls} autoComplete="street-address" required={required} value={v.alamat ?? ''} onChange={(e) => set({ alamat: e.target.value })} placeholder="mis. Jl. Senopati Raya No. 45" />
      </div>
      <div className="grid gap-1.5">
        <Label htmlFor="lok-kecamatan">Kecamatan</Label>
        <Input id="lok-kecamatan" className={cls} required={required} value={v.kecamatan ?? ''} onChange={(e) => set({ kecamatan: e.target.value })} placeholder="mis. Kebayoran Baru" />
      </div>
      <div className="grid gap-1.5">
        <Label htmlFor="lok-kota">Kota / kabupaten</Label>
        <Input id="lok-kota" className={cls} autoComplete="address-level2" required={required} value={v.kota ?? ''} onChange={(e) => set({ kota: e.target.value })} placeholder="mis. Jakarta Selatan" />
      </div>
      <div className="grid gap-1.5 sm:col-span-2">
        <Label htmlFor="lok-provinsi">Provinsi</Label>
        <Select value={v.provinsi ?? ''} onValueChange={(x) => set({ provinsi: x })}>
          <SelectTrigger id="lok-provinsi" className={big ? 'h-12 w-full md:text-base' : 'w-full'}><SelectValue placeholder="Pilih provinsi" /></SelectTrigger>
          <SelectContent>
            {provinsi.map((x) => <SelectItem key={x} value={x}>{x}</SelectItem>)}
          </SelectContent>
        </Select>
      </div>
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
      <LokasiFields v={p} set={set} provinsi={info.options.provinsi} />
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
      <div className="grid gap-3 sm:grid-cols-2">
        <div className="grid gap-1.5">
          <Label htmlFor="biz-omzet">Omzet per tahun (perkiraan)</Label>
          <Input id="biz-omzet" className="tabular" inputMode="numeric" value={p.omzet_tahunan ? formatRp(p.omzet_tahunan) : ''} onChange={(e) => set({ omzet_tahunan: parseRp(e.target.value) || 0 })} placeholder="Rp0" />
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor="biz-laba">Laba sebelum pajak per tahun</Label>
          <Input id="biz-laba" className="tabular" value={tampilRp(String(p.laba_sebelum_pajak ?? ''))} onChange={(e) => set({ laba_sebelum_pajak: parseRp(e.target.value) || 0 })} placeholder="Rp0 (minus = rugi)" />
        </div>
        <p className="text-xs text-muted-foreground sm:col-span-2">Dipakai untuk cek benchmark DJP dan Perencanaan Pajak selama belum ada data transaksi.</p>
      </div>
      <label className="flex items-center gap-3 text-sm" htmlFor="biz-pkp">
        <Switch id="biz-pkp" checked={p.pkp} onCheckedChange={(v) => set({ pkp: v })} />
        Sudah dikukuhkan sebagai PKP (memungut PPN)
      </label>
      {p.pkp && (
        <div className="grid gap-1.5">
          <Label htmlFor="biz-ppn">Tarif PPN (%)</Label>
          <Input
            id="biz-ppn" type="number" step="0.5" min="0" max="100" className="w-32"
            value={((p.ppn_bps ?? 1100) / 100).toString()}
            onChange={(e) => set({ ppn_bps: Math.round(Number(e.target.value) * 100) })}
          />
          <p className="text-xs text-muted-foreground">Dipakai untuk faktur baru. Faktur yang sudah terbit menyimpan tarifnya sendiri, jadi tidak ikut berubah.</p>
        </div>
      )}

      <FakturFields p={p} set={set} />
      {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
      <Button type="submit" size="lg" className="rounded-full font-bold" disabled={busy}>{busy ? 'Menyimpan…' : submitLabel}</Button>
    </form>
  )
}

// tampilRp: tampilan input rupiah berformat ribuan; "-" sendirian dibiarkan supaya rugi bisa diketik.
const tampilRp = (s = '') => {
  const n = parseRp(s)
  return Number.isNaN(n) ? (s.trim().startsWith('-') ? '-' : '') : formatRp(n)
}

const INPUT_LABEL = { nama: 'Nama usaha', omzet: 'Omzet per tahun', laba: 'Laba sebelum pajak per tahun' } as const

// Wizard onboarding: satu pertanyaan bahasa awam per layar, ringkasan di akhir.
function OnboardingWizard({ onSaved, provinsi }: { onSaved: (p: BusinessProfile) => void; provinsi: string[] }) {
  const [a, setA] = useState<Answers>(DEFAULT_ANSWERS)
  const [i, setI] = useState(0)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const headingRef = useRef<HTMLHeadingElement>(null)
  const list = steps(a)
  const step = list[Math.min(i, list.length - 1)]
  const total = list.length - 1
  const isSummary = step === 'ringkasan'
  const q = isSummary ? null : QUESTIONS[step]

  useEffect(() => {
    headingRef.current?.focus()
  }, [step])

  function move(next: number) {
    setError('')
    setI(next)
  }

  async function submit(e: FormEvent) {
    e.preventDefault()
    if (busy) return
    if (step !== 'ringkasan') {
      if (answered(step, a)) move(i + 1)
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

  return (
    <form onSubmit={submit} className="onboarding-wizard" aria-labelledby="onboarding-question" aria-busy={busy}>
      <div className="onboarding-progress">
        <div className="onboarding-progress-label" id="onboarding-progress-label">
          <span>{isSummary ? 'Tinjau profil usaha' : `Pertanyaan ${i + 1} dari ${total}`}</span>
          {isSummary ? <Check size={16} aria-hidden="true" /> : <span>Profil usaha</span>}
        </div>
        <progress aria-label="Pertanyaan yang selesai" max={total} value={isSummary ? total : i} />
        {!isSummary && <p>Jumlah pertanyaan menyesuaikan jawaban Anda.</p>}
      </div>
      <div className="onboarding-question">
        <h2 id="onboarding-question" ref={headingRef} tabIndex={-1} aria-describedby="onboarding-progress-label">
          {isSummary ? `Ini yang kami siapkan untuk ${a.nama?.trim()}` : q!.judul}
        </h2>
        <p>{isSummary ? 'Periksa ringkasan ini sebelum mulai. Semua bisa diubah nanti di menu Profil Usaha.' : q!.options ? 'Pilih satu jawaban, lalu tekan Lanjut.' : step === 'lokasi' ? 'Isi sesuai alamat tempat usaha berjalan.' : step === 'nama' ? 'Gunakan nama yang ingin Anda tampilkan di laporan.' : 'Ketik angka dalam rupiah. Perkiraan kasar sudah cukup.'}</p>
      </div>
      {isSummary ? (
        <>
          <dl className="onboarding-summary">
            {summary(toProfile(a)).map((r) => (
              <div key={r.judul}>
                <dt>{r.judul}</dt>
                <dd>{r.isi}</dd>
              </div>
            ))}
          </dl>
          <CekAwalCard profile={toProfile(a)} />
        </>
      ) : (
        <>
          {step === 'lokasi' ? (
            <LokasiFields v={a} set={(patch) => setA((x) => ({ ...x, ...patch }))} provinsi={provinsi} required big />
          ) : q!.options ? (
            <ChoiceGrid
              key={step}
              name={q!.judul}
              options={q!.options}
              value={a[step] ?? ''}
              onChange={(v) => setA((x) => ({ ...x, [step]: v }))}
            />
          ) : (
            <div className="grid gap-2">
              <Label htmlFor="onboarding-input">{INPUT_LABEL[step as keyof typeof INPUT_LABEL]}</Label>
              {step === 'nama' ? (
                <Input key={step} id="onboarding-input" className="h-12 px-4 md:text-base" autoComplete="organization" required value={a[step] ?? ''} onChange={(e) => setA((x) => ({ ...x, [step]: e.target.value }))} placeholder="mis. Kedai Kopi Senja" aria-describedby="onboarding-reason" />
              ) : (
                <Input key={step} id="onboarding-input" className="h-12 px-4 md:text-base tabular" inputMode={step === 'laba' ? 'text' : 'numeric'} autoComplete="off" required value={tampilRp(a[step])} onChange={(e) => setA((x) => ({ ...x, [step]: e.target.value }))} placeholder={step === 'omzet' ? 'mis. Rp750.000.000' : 'mis. Rp90.000.000'} aria-describedby="onboarding-reason" />
              )}
            </div>
          )}
          <p className="onboarding-reason" id="onboarding-reason"><strong>Kenapa kami tanya ini?</strong> {q!.alasan}</p>
        </>
      )}
      {error && <p className="onboarding-error" role="alert">{error}</p>}
      <div className="onboarding-actions">
        {i > 0 && <Button type="button" variant="ghost" className="min-h-11 px-3" disabled={busy} onClick={() => move(isSummary ? 0 : i - 1)}><ArrowLeft aria-hidden="true" />{isSummary ? 'Ubah jawaban' : 'Kembali'}</Button>}
        <Button type="submit" className="onboarding-primary min-h-11 px-5 font-bold" disabled={busy || !answered(step, a)}>
          {busy ? 'Menyimpan…' : isSummary ? 'Simpan & mulai' : 'Lanjut'}
          {!busy && <ArrowRight aria-hidden="true" />}
        </Button>
      </div>
    </form>
  )
}

// Onboarding: layar pertama Owner sebelum workspace bila profil usaha belum diisi.
// OnboardingShell: layar penuh tanpa sidebar (onboarding & layar bayar langganan).
export function OnboardingShell({ onLogout, labelledBy, children }: { onLogout: () => void; labelledBy?: string; children: React.ReactNode }) {
  return (
    <main className="onboarding-page">
      <div className="onboarding-shell">
        <header className="onboarding-brand">
          <div className="flex items-center gap-2.5">
            <BrandMark size={28} />
            <div>
              <div className="text-base font-bold tracking-tight">Laporan Bisnis</div>
              <div className="brand-sub">by Akademi Pajak</div>
            </div>
          </div>
          <Button type="button" variant="ghost" className="min-h-11 text-muted-foreground" onClick={onLogout}>Keluar</Button>
        </header>
        <section className="onboarding-panel" aria-labelledby={labelledBy}>{children}</section>
      </div>
    </main>
  )
}

export function Onboarding({ onSaved, canEdit, onLogout, provinsi }: { onSaved: (p: BusinessProfile) => void; canEdit: boolean; onLogout: () => void; provinsi: string[] }) {
  return (
    <OnboardingShell onLogout={onLogout} labelledBy="onboarding-title">
      <div className="onboarding-intro">
        <h1 id="onboarding-title">Kenalkan usaha Anda dulu</h1>
        <p>Tidak perlu paham akuntansi atau pajak. Cukup jawab sesuai keadaan usaha Anda.</p>
      </div>
      {canEdit ? (
        <OnboardingWizard onSaved={onSaved} provinsi={provinsi} />
      ) : (
        <p className="text-sm">Owner usaha perlu melengkapi profil usaha terlebih dahulu. Hubungi Owner Anda, lalu muat ulang halaman ini.</p>
      )}
    </OnboardingShell>
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


// FakturFields: identitas penerbit yang dicetak di invoice. Dipisah dari pertanyaan pajak
// supaya form utama tetap pendek, dan semuanya opsional — usaha yang belum pernah menerbitkan
// faktur tidak perlu mengisinya untuk bisa memakai aplikasi.
function FakturFields({ p, set }: { p: BusinessProfile; set: (patch: Partial<BusinessProfile>) => void }) {
  const [buka, setBuka] = useState(false)
  const terisi = [p.npwp, p.telepon, p.bank_rekening, p.penandatangan_nama].filter(Boolean).length

  return (
    <div className="rounded-lg border border-border p-4">
      <button type="button" className="flex w-full items-center justify-between text-left" onClick={() => setBuka((v) => !v)} aria-expanded={buka}>
        <span className="text-sm font-semibold">Data untuk invoice</span>
        <span className="text-xs text-muted-foreground">{terisi}/4 terisi — {buka ? 'sembunyikan' : 'tampilkan'}</span>
      </button>
      <p className="mt-1 text-xs text-muted-foreground">NPWP, kontak, rekening, dan penandatangan yang dicetak di PDF faktur.</p>
      {buka && (
        <div className="mt-4 grid gap-3">
          {([
            ['npwp', 'NPWP', '01.234.567.8-012.000'],
            ['telepon', 'Telepon', '021-7123456'],
            ['email', 'Email', 'halo@usahaku.id'],
            ['bank_nama', 'Nama bank', 'BCA'],
            ['bank_rekening', 'Nomor rekening', '123-456-7890'],
            ['bank_atas_nama', 'Rekening atas nama', 'CV Kedai Kopi Senja'],
            ['penandatangan_nama', 'Nama penandatangan', 'Sari Wulandari'],
            ['penandatangan_jabatan', 'Jabatan penandatangan', 'Pemilik'],
          ] as const).map(([key, label, placeholder]) => (
            <div className="grid gap-1.5" key={key}>
              <Label htmlFor={`biz-${key}`}>{label}</Label>
              <Input
                id={`biz-${key}`} placeholder={placeholder} value={p[key] ?? ''}
                onChange={(e) => set({ [key]: e.target.value } as Partial<BusinessProfile>)}
              />
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
