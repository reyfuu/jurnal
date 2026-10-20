import { useState, type FormEvent } from 'react'
import { ArrowLeft, Eye, EyeOff } from 'lucide-react'
import { KebijakanPrivasi } from '@/components/KebijakanPrivasi'
import { BrandMark } from '@/components/Sidebar'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { login, register, type Me } from '@/lib/api'

// Akun demo hanya tampil di `npm run dev`; build produksi membuangnya (password prod dari env SEED_PASSWORD).
const DEMO_PASSWORD = 'senja2026'
const DEMO_ACCOUNTS = [
  { email: 'owner@kedaikopisenja.id', role: 'Owner' },
  { email: 'admin@kedaikopisenja.id', role: 'Admin Keuangan' },
  { email: 'kasir@kedaikopisenja.id', role: 'Kasir' },
]

export function Login({ onLogin, initialDaftar = false, onBack }: { onLogin: (me: Me) => void; initialDaftar?: boolean; onBack?: () => void }) {
  // daftar: buat usaha baru (kosong), lalu langsung masuk wizard onboarding.
  const [daftar, setDaftar] = useState(initialDaftar)
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [wa, setWa] = useState('')
  const [setujuPDP, setSetujuPDP] = useState(false)
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  async function submit(e: FormEvent) {
    e.preventDefault()
    setShowPassword(false)
    setBusy(true)
    setError('')
    try {
      onLogin(await (daftar ? register({ name, email, password, wa, setuju_pdp: setujuPDP }) : login(email, password)))
    } catch (err) {
      setError((err as Error).message)
      setBusy(false)
    }
  }

  return (
    <main className="login-page flex min-h-screen items-center justify-center px-4 py-10">
      <div className="w-full max-w-sm">
        {onBack && (
          <Button type="button" variant="ghost" className="mb-4 -ml-2 text-muted-foreground" onClick={onBack}>
            <ArrowLeft aria-hidden="true" /> Kembali ke beranda
          </Button>
        )}
        <div className="mb-6 flex items-center justify-center gap-2.5">
          <BrandMark size={28} />
          <div>
            <div className="text-base font-bold tracking-tight">Laporan Bisnis</div>
            <div className="brand-sub">by Master Pajak</div>
          </div>
        </div>

        <Card className="border-[var(--border-shell)] bg-[var(--bg-shell)]">
          <CardHeader>
            <CardTitle className="text-xl">{daftar ? 'Daftar' : 'Masuk'}</CardTitle>
            <CardDescription>
              {daftar ? 'Buat akun Owner untuk usaha baru Anda. Datanya mulai kosong.' : 'Gunakan akun yang dibuat oleh Owner bisnis Anda.'}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={submit} className="grid gap-4">
              {daftar && <div className="grid gap-1.5">
                <Label htmlFor="name">Nama</Label>
                <Input id="name" className="h-11" autoComplete="name" required autoFocus value={name} onChange={(e) => setName(e.target.value)} />
              </div>}
              <div className="grid gap-1.5">
                <Label htmlFor="email">Email</Label>
                <Input id="email" className="h-11" type="email" autoComplete="username" required autoFocus={!daftar} value={email} onChange={(e) => setEmail(e.target.value)} />
              </div>
              {daftar && <div className="grid gap-1.5">
                <Label htmlFor="wa">Nomor WhatsApp</Label>
                <Input id="wa" className="h-11" type="tel" inputMode="tel" autoComplete="tel" placeholder="0812-3456-7890" required value={wa} onChange={(e) => setWa(e.target.value)} />
              </div>}
              <div className="grid gap-1.5">
                <Label htmlFor="password">Password</Label>
                <div className="relative">
                  <Input id="password" className="h-11 pr-12" type={showPassword ? 'text' : 'password'} autoComplete={daftar ? 'new-password' : 'current-password'} required minLength={daftar ? 8 : undefined} value={password} onChange={(e) => setPassword(e.target.value)} />
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="absolute right-0 top-0 size-11 text-muted-foreground"
                    aria-label={showPassword ? 'Sembunyikan password' : 'Tampilkan password'}
                    aria-controls="password"
                    disabled={busy}
                    onClick={() => setShowPassword((visible) => !visible)}
                  >
                    {showPassword ? <EyeOff aria-hidden="true" /> : <Eye aria-hidden="true" />}
                  </Button>
                </div>
              </div>
              {daftar && <label className="flex items-start gap-2.5 text-sm text-muted-foreground">
                <input type="checkbox" className="mt-0.5 size-4 shrink-0 accent-[var(--primary)]" required checked={setujuPDP} onChange={(e) => setSetujuPDP(e.target.checked)} />
                <span>
                  Saya menyetujui pemrosesan data pribadi saya sesuai{' '}
                  <KebijakanPrivasi>
                    <button type="button" className="font-semibold text-foreground underline underline-offset-4">Kebijakan Privasi</button>
                  </KebijakanPrivasi>{' '}
                  (UU No. 27 Tahun 2022 tentang Pelindungan Data Pribadi).
                </span>
              </label>}
              {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
              <Button type="submit" size="lg" className="w-full rounded-full font-bold" disabled={busy || (daftar && !setujuPDP)}>
                {busy ? 'Memeriksa…' : daftar ? 'Daftar' : 'Masuk'}
              </Button>
            </form>
            <p className="mt-4 text-center text-sm text-muted-foreground">
              {daftar ? 'Sudah punya akun?' : 'Belum punya akun?'}{' '}
              <button type="button" className="font-semibold text-foreground underline-offset-4 hover:underline" onClick={() => { setDaftar(!daftar); setError('') }}>
                {daftar ? 'Masuk' : 'Daftar'}
              </button>
            </p>
          </CardContent>
        </Card>

        {import.meta.env.DEV && <div className="mt-5 rounded-xl border border-[var(--border-subtle)] p-3.5 text-xs text-muted-foreground">
          <div className="mb-2 font-semibold text-foreground">Akun demo · password <code className="font-mono">{DEMO_PASSWORD}</code></div>
          <div className="grid gap-1">
            {DEMO_ACCOUNTS.map((a) => (
              <button
                key={a.email}
                type="button"
                className="flex justify-between rounded-md px-2 py-1 text-left hover:bg-white/5"
                onClick={() => { setEmail(a.email); setPassword(DEMO_PASSWORD) }}
              >
                <span className="font-mono">{a.email}</span>
                <span>{a.role}</span>
              </button>
            ))}
          </div>
        </div>}
      </div>
    </main>
  )
}
