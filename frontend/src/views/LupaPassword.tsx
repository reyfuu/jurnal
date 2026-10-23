import { useState, type FormEvent } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Logo } from '@/components/Logo'
import { aturPassword, lupaPassword } from '@/lib/api'

// Lupa password (checklist tester No 15): minta tautan lewat email, lalu atur password baru.

/** Kartu "Lupa password" di dalam halaman Masuk. */
export function LupaPassword({ emailAwal, onKembali }: { emailAwal: string; onKembali: () => void }) {
  const [email, setEmail] = useState(emailAwal)
  const [pesan, setPesan] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  async function kirim(e: FormEvent) {
    e.preventDefault()
    setBusy(true); setError('')
    try { setPesan((await lupaPassword(email)).pesan) } catch (err) { setError((err as Error).message) } finally { setBusy(false) }
  }
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-xl">Lupa password</CardTitle>
        <CardDescription>Masukkan email akun Anda. Kami kirim tautan untuk membuat password baru.</CardDescription>
      </CardHeader>
      <CardContent>
        {pesan ? <p className="text-sm" role="status">{pesan}</p> : (
          <form onSubmit={kirim} className="grid gap-4">
            <div className="grid gap-2">
              <Label htmlFor="email-lupa">Email</Label>
              <Input id="email-lupa" className="h-11" type="email" autoComplete="email" required value={email} onChange={(e) => setEmail(e.target.value)} />
            </div>
            {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
            <Button type="submit" size="lg" className="w-full" disabled={busy}>{busy ? 'Mengirim…' : 'Kirim tautan'}</Button>
          </form>
        )}
        <p className="mt-4 text-center text-sm text-muted-foreground">
          <button type="button" className="font-semibold text-foreground underline-offset-4 hover:underline" onClick={onKembali}>Kembali ke Masuk</button>
        </p>
      </CardContent>
    </Card>
  )
}

/** Halaman dari tautan email: #/atur-password?token=… */
export function AturPassword({ token, onSelesai }: { token: string; onSelesai: () => void }) {
  const [pw, setPw] = useState('')
  const [ulang, setUlang] = useState('')
  const [error, setError] = useState('')
  const [selesai, setSelesai] = useState(false)
  const [busy, setBusy] = useState(false)
  async function simpan(e: FormEvent) {
    e.preventDefault()
    if (pw !== ulang) return setError('Kedua password belum sama.')
    setBusy(true); setError('')
    try { await aturPassword(token, pw); setSelesai(true) } catch (err) { setError((err as Error).message) } finally { setBusy(false) }
  }
  return (
    <main className="grid min-h-[100dvh] place-items-center bg-[var(--bg-shell)] p-4">
      <div className="w-full max-w-md">
        <div className="dark mb-6 flex justify-center"><Logo tinggi={32} /></div>
        <Card>
          <CardHeader>
            <CardTitle className="text-xl">Atur password baru</CardTitle>
            <CardDescription>Setelah disimpan, semua perangkat yang masih masuk akan keluar.</CardDescription>
          </CardHeader>
          <CardContent>
            {selesai ? (
              <div className="grid gap-4">
                <p className="text-sm" role="status">Password sudah diganti. Silakan masuk dengan password baru.</p>
                <Button size="lg" onClick={onSelesai}>Masuk</Button>
              </div>
            ) : (
              <form onSubmit={simpan} className="grid gap-4">
                <div className="grid gap-2">
                  <Label htmlFor="pw-baru">Password baru</Label>
                  <Input id="pw-baru" className="h-11" type="password" autoComplete="new-password" required minLength={8} value={pw} onChange={(e) => setPw(e.target.value)} />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="pw-ulang">Ulangi password baru</Label>
                  <Input id="pw-ulang" className="h-11" type="password" autoComplete="new-password" required minLength={8} value={ulang} onChange={(e) => setUlang(e.target.value)} />
                </div>
                {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
                <Button type="submit" size="lg" className="w-full" disabled={busy}>{busy ? 'Menyimpan…' : 'Simpan password'}</Button>
                {error && <button type="button" className="text-sm underline underline-offset-4" onClick={onSelesai}>Minta tautan baru dari halaman Masuk</button>}
              </form>
            )}
          </CardContent>
        </Card>
      </div>
    </main>
  )
}
