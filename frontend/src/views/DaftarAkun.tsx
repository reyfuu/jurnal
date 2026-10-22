import { useEffect, useState } from 'react'
import { Plus, Trash2 } from 'lucide-react'
import { PageHeader } from '@/components/PageHeader'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectGroup, SelectItem, SelectLabel, SelectTrigger, SelectValue } from '@/components/ui/select'
import { CATATAN_AKUN_TAMBAHAN, getCoA, hapusAkunCoA, perbaruiCoA, tambahAkunCoA, type Account, type CoAInfo } from '@/lib/api'

const KONDISI: Partial<Record<Account['kondisi'], string>> = {
  jika_ada: 'Opsional, bila dimiliki',
  jika_pt: 'Khusus PT',
  jika_non_pt: 'Khusus OP / CV',
  jika_perpetual: 'Metode perpetual',
  jika_periodik: 'Metode periodik',
}

/** bisaPerbarui: Owner boleh memindahkan usaha ke versi template COA terbaru. */
export function DaftarAkun({ bisaPerbarui }: { bisaPerbarui: boolean }) {
  const [c, setC] = useState<CoAInfo | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => { getCoA().then(setC, (e: Error) => setError(e.message)) }, [])
  async function perbarui() {
    setBusy(true)
    setError('')
    try {
      await perbaruiCoA()
      setC(await getCoA())
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setBusy(false)
    }
  }
  const p = c?.pembaruan
  const [nama, setNama] = useState('')
  const [induk, setInduk] = useState('')
  const [pesan, setPesan] = useState('')

  async function tambah() {
    setBusy(true)
    setError('')
    setPesan('')
    try {
      const a = await tambahAkunCoA(nama.trim(), induk)
      setC(await getCoA())
      setNama('')
      setPesan(`Akun ${a.kode} ${a.nama} ditambahkan.`)
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setBusy(false)
    }
  }
  async function hapus(a: Account) {
    if (!window.confirm(`Hapus akun ${a.kode} ${a.nama}?`)) return
    setBusy(true)
    setError('')
    setPesan('')
    try {
      await hapusAkunCoA(a.kode)
      setC(await getCoA())
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <>
      <PageHeader title="Daftar Akun (CoA)" subtitle="Susunan akun otomatis sesuai jenis usaha, bentuk badan, dan metode persediaan di Profil Usaha" />
      {error && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start', whiteSpace: 'normal' }}>{error}</div>}
      {p && (
        <div className="card" style={{ padding: 20, borderColor: 'var(--accent-amber)' }} role="status">
          <div className="chart-title">Versi baru daftar akun tersedia (v{p.versi})</div>
          <p className="text-sm text-muted-foreground" style={{ marginTop: 4 }}>
            Tim Master Pajak memperbarui template. Usaha Anda masih memakai v{c?.template?.versi} sampai Anda menerapkan versi baru.
          </p>
          <ul className="text-sm" style={{ margin: '10px 0', display: 'grid', gap: 4 }}>
            {p.diff.tambah.map((a) => <li key={'t' + a.kode}><span style={{ color: 'var(--accent-green)' }}>+ Akun baru</span> {a.kode} {a.nama}</li>)}
            {p.diff.ubah.map((a) => <li key={'u' + a.kode}><span style={{ color: 'var(--accent-amber)' }}>~ Berubah</span> {a.kode} {a.nama}</li>)}
            {p.diff.hapus.map((a) => <li key={'h' + a.kode}><span style={{ color: 'var(--accent-red)' }}>− Dihapus</span> {a.kode} {a.nama}</li>)}
          </ul>
          {bisaPerbarui
            ? <Button className="justify-self-start" style={{ width: 'fit-content' }} disabled={busy} onClick={() => void perbarui()}>{busy ? 'Menerapkan…' : `Terapkan v${p.versi}`}</Button>
            : <p className="text-xs text-muted-foreground">Hanya Owner yang bisa menerapkan versi baru.</p>}
        </div>
      )}
      {c && !c.supported && (
        <div className="card" style={{ padding: 22 }}>
          <div className="chart-title">CoA belum tersedia untuk jenis usaha ini</div>
          <p className="text-sm text-muted-foreground" style={{ marginTop: 6 }}>
            CoA untuk usaha jasa sedang disiapkan tim Master Pajak. Sementara itu laporan memakai format ringkas.
            Usaha perdagangan dan manufaktur sudah memakai CoA lengkap, masing-masing dengan daftar akunnya sendiri.
          </p>
        </div>
      )}
      {c?.supported && c.accounts && (
        <>
          <div className="card">
            <div className="chart-title">Pemetaan otomatis jenis transaksi ke akun</div>
            <p className="kpi-meta-row" style={{ marginTop: 4 }}>Setiap transaksi otomatis masuk ke akun ini. Akun bisa dikoreksi per transaksi di form Transaksi.</p>
            <table className="data-table tabular">
              <thead><tr><th>Jenis transaksi</th><th>Kode</th><th>Akun</th></tr></thead>
              <tbody>
                {c.mapping?.map((m) => (
                  <tr key={m.jenis}><td>{m.jenis}</td><td>{m.kode}</td><td>{m.nama}</td></tr>
                ))}
              </tbody>
            </table>
          </div>

          {bisaPerbarui && (
            <div className="card" style={{ padding: 20 }}>
              <div className="chart-title">Tambah akun</div>
              <p className="kpi-meta-row" style={{ marginTop: 4 }}>
                Pilih akun yang jenisnya sama. Akun baru mengikuti kelompok, saldo normal, dan posisinya di laporan; kodenya dibuat otomatis setelah akun itu.
              </p>
              <form className="grid gap-3 sm:grid-cols-[1fr_1fr_auto] sm:items-end" style={{ marginTop: 10 }}
                onSubmit={(e) => { e.preventDefault(); void tambah() }}>
                <div className="grid gap-1.5">
                  <Label htmlFor="akun-nama">Nama akun baru</Label>
                  <Input id="akun-nama" value={nama} onChange={(e) => setNama(e.target.value)} maxLength={80} placeholder="mis. Kas Cabang Bandung" />
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="akun-induk">Jenisnya sama dengan</Label>
                  <Select value={induk} onValueChange={setInduk}>
                    <SelectTrigger id="akun-induk"><SelectValue placeholder="Pilih akun" /></SelectTrigger>
                    <SelectContent>
                      {c.kelompok.map((k) => {
                        const rows = c.accounts!.filter((a) => a.kelompok === k)
                        return rows.length ? (
                          <SelectGroup key={k}>
                            <SelectLabel>{k}</SelectLabel>
                            {rows.map((a) => <SelectItem key={a.kode} value={a.kode}>{a.kode} {a.nama}</SelectItem>)}
                          </SelectGroup>
                        ) : null
                      })}
                    </SelectContent>
                  </Select>
                </div>
                <Button type="submit" disabled={busy || !nama.trim() || !induk}><Plus size={16} aria-hidden="true" /> Tambah akun</Button>
              </form>
              {pesan && <p className="text-sm" role="status" style={{ marginTop: 8, color: 'var(--accent-green)' }}>{pesan}</p>}
            </div>
          )}

          {c.kelompok.map((k) => {
            const rows = c.accounts!.filter((a) => a.kelompok === k)
            if (!rows.length) return null
            return (
              <div className="card" key={k}>
                <div className="chart-title">{k}</div>
                <table className="data-table tabular">
                  <thead><tr><th style={{ width: 80 }}>Kode</th><th>Nama akun</th><th style={{ width: 110 }}>Saldo normal</th><th>Keterangan</th></tr></thead>
                  <tbody>
                    {rows.map((a) => (
                      <tr key={a.kode}>
                        <td>{a.kode}</td>
                        <td>
                          {a.nama}
                          {KONDISI[a.kondisi] && <span className={'pill-status ' + (a.opsional ? 'pill-status-muted' : 'pill-amber')} style={{ marginLeft: 8 }}>{KONDISI[a.kondisi]}</span>}
                        </td>
                        <td style={{ textTransform: 'capitalize' }}>{a.normal}</td>
                        <td style={{ color: 'var(--text-muted)', whiteSpace: 'normal' }}>
                          {a.catatan}
                          {bisaPerbarui && a.catatan === CATATAN_AKUN_TAMBAHAN && (
                            <button type="button" className="ml-2 inline-flex align-middle text-destructive" aria-label={`Hapus akun ${a.kode} ${a.nama}`} disabled={busy} onClick={() => void hapus(a)}>
                              <Trash2 size={14} aria-hidden="true" />
                            </button>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )
          })}
          <p className="text-xs text-muted-foreground">Sumber: Daftar COA tim Master Pajak untuk jenis usaha {c.template?.jenis ?? 'Anda'}{c.template ? ` · versi ${c.template.versi}` : ''}. Daftar akun dipilih otomatis dari jenis usaha di profil.</p>
        </>
      )}
    </>
  )
}
