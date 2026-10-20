import { useEffect, useState } from 'react'
import { Check } from 'lucide-react'
import { PageHeader } from '@/components/PageHeader'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { GARANSI_BELUM_BAYAR, checkout, getBilling, klaimGaransi, refreshBilling, type Billing, type GaransiStatus, type LanggananStatus } from '@/lib/api'
import { formatRp } from '@/lib/format'

declare global {
  interface Window {
    snap?: { pay: (token: string, cb: { onSuccess?: () => void; onPending?: () => void; onError?: () => void; onClose?: () => void }) => void }
  }
}

// loadSnap: snap.js Midtrans dimuat sekali, hanya saat pengguna menekan Bayar.
function loadSnap(src: string, clientKey: string): Promise<void> {
  if (window.snap) return Promise.resolve()
  return new Promise((resolve, reject) => {
    const s = document.createElement('script')
    s.src = src
    s.dataset.clientKey = clientKey
    s.onload = () => resolve()
    s.onerror = () => reject(new Error('Gagal memuat halaman pembayaran Midtrans.'))
    document.head.appendChild(s)
  })
}

const tanggal = (iso?: string) => (iso ? new Date(iso).toLocaleDateString('id-ID', { day: 'numeric', month: 'long', year: 'numeric' }) : '')

export function statusText(l: LanggananStatus) {
  if (l.status === 'aktif') return l.sampai ? `Langganan aktif sampai ${tanggal(l.sampai)}.` : 'Langganan aktif.'
  if (l.status === 'trial') return `Masa coba gratis sampai ${tanggal(l.sampai)}.`
  return 'Langganan belum aktif. Pilih paket untuk mulai memakai Laporan Bisnis.'
}

export const SYARAT_GARANSI =
  'Garansi 7 hari uang kembali untuk pembayaran pertama, selama Anda belum mengunduh/ekspor data (Excel atau PDF). Mengajukan garansi menghentikan akses saat itu juga; dana dikembalikan manual lewat Midtrans dalam 3–7 hari kerja.'

// GaransiCard: status garansi + tombol klaim. onDiklaim memuat ulang sesi (akses berhenti).
function GaransiCard({ g, onDiklaim }: { g: GaransiStatus; onDiklaim: () => void }) {
  const [alasan, setAlasan] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  if (g.alasan === GARANSI_BELUM_BAYAR) return null

  async function klaim() {
    setBusy(true)
    setError('')
    try {
      await klaimGaransi(alasan)
      onDiklaim()
    } catch (e) {
      setError((e as Error).message)
      setBusy(false)
    }
  }

  return (
    <div className="card" style={{ padding: 22, maxWidth: 880, marginBottom: 16 }}>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <div className="text-sm font-semibold">Garansi 7 hari uang kembali</div>
          <p className="mt-1 text-sm text-muted-foreground">
            {g.berlaku ? `Berlaku sampai ${tanggal(g.sampai)}. Mengunduh/ekspor data akan menggugurkannya.` : `Tidak berlaku: ${g.alasan}.`}
          </p>
        </div>
        {g.berlaku && (
          <Dialog>
            <DialogTrigger asChild><Button variant="outline" size="sm">Ajukan garansi</Button></DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Ajukan garansi uang kembali?</DialogTitle>
                <DialogDescription>Akses ke aplikasi berhenti saat ini juga. Dana dikembalikan manual lewat Midtrans dalam 3–7 hari kerja.</DialogDescription>
              </DialogHeader>
              <label className="grid gap-1.5 text-sm">
                Alasan (opsional, membantu kami memperbaiki produk)
                <textarea className="input-field" rows={3} maxLength={500} value={alasan} onChange={(e) => setAlasan(e.target.value)} />
              </label>
              {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
              <DialogFooter>
                <Button variant="destructive" disabled={busy} onClick={() => void klaim()}>{busy ? 'Mengirim…' : 'Ya, ajukan & hentikan akses'}</Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        )}
      </div>
    </div>
  )
}

export function Langganan({ onPaid }: { onPaid: () => void }) {
  const [b, setB] = useState<Billing | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState('')
  const [info, setInfo] = useState('')

  useEffect(() => {
    getBilling().then(setB, (e: Error) => setError(e.message))
  }, [])

  async function cekStatus() {
    setBusy('cek')
    try {
      const nb = await refreshBilling()
      setB(nb)
      if (nb.langganan.status === 'aktif') onPaid()
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setBusy('')
    }
  }

  async function bayar(paket: string) {
    if (!b) return
    setBusy(paket)
    setError('')
    setInfo('')
    try {
      const co = await checkout(paket)
      await loadSnap(b.snap_js, b.client_key)
      window.snap!.pay(co.token, {
        onSuccess: () => { setInfo('Pembayaran berhasil. Memperbarui status…'); void cekStatus() },
        onPending: () => setInfo('Pembayaran menunggu penyelesaian. Tekan "Cek status pembayaran" setelah Anda membayar.'),
        onError: () => setError('Pembayaran gagal. Silakan coba lagi.'),
        onClose: () => setBusy(''),
      })
    } catch (e) {
      setError((e as Error).message)
      setBusy('')
    }
  }

  if (!b) return error ? <p className="text-sm text-destructive" role="alert">{error}</p> : null
  const habis = b.langganan.status === 'habis'
  return (
    <>
      <PageHeader title="Langganan" subtitle="Kelola paket Laporan Bisnis untuk usaha Anda" />
      <div className="card" style={{ padding: 22, maxWidth: 880, marginBottom: 16, borderColor: habis ? 'var(--accent-red)' : undefined }}>
        <div className="text-sm font-semibold">Status: {b.langganan.status === 'trial' ? 'Masa coba' : b.langganan.status === 'aktif' ? 'Aktif' : 'Belum aktif'}</div>
        <p className="mt-1 text-sm text-muted-foreground">{statusText(b.langganan)}</p>
      </div>

      <GaransiCard g={b.garansi} onDiklaim={onPaid} />

      {!b.aktif && <p className="mb-4 text-sm text-muted-foreground">Pembayaran online belum diaktifkan oleh admin. Hubungi Master Pajak untuk aktivasi.</p>}
      <div className="grid gap-4 sm:grid-cols-2" style={{ maxWidth: 880 }}>
        {b.paket.map((p) => (
          <div key={p.id} className="card grid gap-3" style={{ padding: 22, justifyContent: 'stretch' }}>
            <div className="flex items-baseline justify-between gap-2">
              <div className="text-base font-bold">{p.nama}</div>
              {p.hemat && <span className="pill-status pill-green">{p.hemat}</span>}
            </div>
            <div><span className="text-2xl font-bold tabular">{formatRp(p.harga)}</span> <span className="text-sm text-muted-foreground">/ {p.hari === 365 ? 'tahun' : 'bulan'}</span></div>
            <ul className="grid gap-1.5 text-sm text-muted-foreground">
              {['Semua fitur laporan & pajak', 'Pengguna tanpa batas', 'Laporan AI & cek benchmark DJP'].map((x) => (
                <li key={x} className="flex items-center gap-2"><Check className="size-4 text-[var(--accent-green)]" aria-hidden="true" />{x}</li>
              ))}
            </ul>
            <Button className="rounded-full font-bold" disabled={!b.aktif || busy !== ''} onClick={() => void bayar(p.id)}>
              {busy === p.id ? 'Membuka pembayaran…' : `Bayar ${p.nama.toLowerCase()}`}
            </Button>
          </div>
        ))}
      </div>
      <p className="mt-3 text-xs text-muted-foreground" style={{ maxWidth: 880 }}>{SYARAT_GARANSI}</p>
      {info && <p className="mt-4 text-sm" style={{ color: 'var(--accent-green)' }} role="status">{info}</p>}
      {error && <p className="mt-4 text-sm text-destructive" role="alert">{error}</p>}

      {(b.orders ?? []).length > 0 && (
        <div className="card" style={{ padding: 22, maxWidth: 880, marginTop: 16 }}>
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div className="chart-title">Riwayat pembayaran</div>
            <Button variant="outline" size="sm" disabled={busy !== ''} onClick={() => void cekStatus()}>{busy === 'cek' ? 'Mengecek…' : 'Cek status pembayaran'}</Button>
          </div>
          <table className="mt-3 w-full text-sm">
            <thead><tr className="text-left text-muted-foreground"><th className="py-1.5 font-medium">Tanggal</th><th className="font-medium">Paket</th><th className="font-medium">Jumlah</th><th className="font-medium">Status</th></tr></thead>
            <tbody>
              {b.orders.map((o) => (
                <tr key={o.id} className="border-t border-border">
                  <td className="py-2">{tanggal(o.waktu)}</td>
                  <td className="capitalize">{o.paket}</td>
                  <td className="tabular">{formatRp(o.jumlah)}</td>
                  <td>{o.status === 'lunas' ? 'Lunas' : o.status === 'gagal' ? 'Gagal' : 'Menunggu'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </>
  )
}
