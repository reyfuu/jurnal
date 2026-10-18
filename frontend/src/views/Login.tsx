import { useState, type FormEvent } from 'react'
import { BrandMark } from '@/components/Sidebar'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { login, type Me } from '@/lib/api'

const DEMO_PASSWORD = 'senja2026'
const DEMO_ACCOUNTS = [
  { email: 'owner@kedaikopisenja.id', role: 'Owner' },
  { email: 'admin@kedaikopisenja.id', role: 'Admin Keuangan' },
  { email: 'kasir@kedaikopisenja.id', role: 'Kasir' },
]

export function Login({ onLogin }: { onLogin: (me: Me) => void }) {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  async function submit(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      onLogin(await login(email, password))
    } catch (err) {
      setError((err as Error).message)
      setBusy(false)
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center px-4 py-10">
      <div className="w-full max-w-sm">
        <div className="mb-6 flex items-center justify-center gap-2.5">
          <BrandMark size={28} />
          <div>
            <div className="text-base font-bold tracking-tight">Laporan Bisnis</div>
            <div className="brand-sub">by Akademi Pajak</div>
          </div>
        </div>

        <Card className="border-[var(--border-shell)] bg-[var(--bg-shell)]">
          <CardHeader>
            <CardTitle className="text-xl">Masuk</CardTitle>
            <CardDescription>Gunakan akun yang dibuat oleh Owner bisnis Anda.</CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={submit} className="grid gap-4">
              <div className="grid gap-1.5">
                <Label htmlFor="email">Email</Label>
                <Input id="email" type="email" autoComplete="username" required autoFocus value={email} onChange={(e) => setEmail(e.target.value)} />
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="password">Password</Label>
                <Input id="password" type="password" autoComplete="current-password" required value={password} onChange={(e) => setPassword(e.target.value)} />
              </div>
              {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
              <Button type="submit" size="lg" className="w-full rounded-full font-bold" disabled={busy}>
                {busy ? 'Memeriksa…' : 'Masuk'}
              </Button>
            </form>
          </CardContent>
        </Card>

        <div className="mt-5 rounded-xl border border-[var(--border-subtle)] p-3.5 text-xs text-muted-foreground">
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
        </div>
      </div>
    </main>
  )
}
