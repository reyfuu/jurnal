import { useState, type FormEvent } from 'react'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

// Dialog bersama kelola data Panel Master: hapus permanen (ketik ulang nama/email) dan aksi
// yang wajib alasan (arsipkan).

export function HapusPermanenDialog({ open, onClose, judul, teks, kunci, labelKunci, onHapus }: {
  open: boolean; onClose: () => void; judul: string; teks: string
  /** Yang harus diketik ulang persis, mis. nama usaha atau email. */
  kunci: string; labelKunci: string
  onHapus: (konfirmasi: string) => Promise<void>
}) {
  const [isi, setIsi] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const cocok = isi.trim().toLowerCase() === kunci.trim().toLowerCase()
  async function kirim(e: FormEvent) {
    e.preventDefault()
    if (!cocok) return
    setBusy(true)
    setError('')
    try {
      await onHapus(isi)
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) { setIsi(''); setError(''); onClose() } }}>
      <DialogContent>
        <form onSubmit={kirim} className="grid gap-4">
          <DialogHeader>
            <DialogTitle>{judul}</DialogTitle>
            <DialogDescription>{teks} <strong>Tindakan ini tidak bisa dibatalkan.</strong></DialogDescription>
          </DialogHeader>
          <div className="grid gap-1.5">
            <Label htmlFor="hp-kunci">Ketik {labelKunci}: <span className="font-semibold">{kunci}</span></Label>
            <Input id="hp-kunci" autoComplete="off" value={isi} onChange={(e) => setIsi(e.target.value)} />
          </div>
          {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>Batal</Button>
            <Button type="submit" variant="destructive" disabled={busy || !cocok}>{busy ? 'Menghapus…' : 'Hapus permanen'}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

export function AlasanDialog({ open, onClose, judul, teks, tombol, onKirim }: {
  open: boolean; onClose: () => void; judul: string; teks: string; tombol: string
  onKirim: (alasan: string) => Promise<void>
}) {
  const [alasan, setAlasan] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  async function kirim(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      await onKirim(alasan)
      setAlasan('')
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent>
        <form onSubmit={kirim} className="grid gap-4">
          <DialogHeader>
            <DialogTitle>{judul}</DialogTitle>
            <DialogDescription>{teks}</DialogDescription>
          </DialogHeader>
          <div className="grid gap-1.5">
            <Label htmlFor="al-alasan">Alasan</Label>
            <Input id="al-alasan" required maxLength={300} value={alasan} onChange={(e) => setAlasan(e.target.value)} />
          </div>
          {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>Batal</Button>
            <Button type="submit" disabled={busy || !alasan.trim()}>{busy ? 'Menyimpan…' : tombol}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

/** Konfirmasi sederhana untuk aksi yang bisa dibatalkan, mis. nonaktifkan akun (soft delete). */
export function KonfirmasiDialog({ open, onClose, judul, teks, tombol, onYa }: {
  open: boolean; onClose: () => void; judul: string; teks: string; tombol: string
  onYa: () => Promise<void>
}) {
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  async function ya() {
    setBusy(true)
    setError('')
    try {
      await onYa()
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) { setError(''); onClose() } }}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{judul}</DialogTitle>
          <DialogDescription>{teks}</DialogDescription>
        </DialogHeader>
        {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>Batal</Button>
          <Button type="button" disabled={busy} onClick={() => void ya()}>{busy ? 'Menyimpan…' : tombol}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
