import { useEffect, useState, type FormEvent } from 'react'
import { PageHeader } from '@/components/PageHeader'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { getUmurSaldo, lunasiTransaksi, pelunasanSaldoAwal, type UmurSaldo } from '@/lib/api'
import { angka } from '@/lib/angka'
import { PiutangUtang } from '@/views/PiutangStok'
import { formatRp, formatTanggal } from '@/lib/format'

// Umur piutang & utang: transaksi yang belum lunas (+ saldo awal) dikelompokkan menurut lama
// tertunggaknya. Totalnya sama dengan piutang/utang di Neraca. Pelunasan dicatat di sini dengan
// tanggal bayar, jadi kas masuk/keluar di bulan pembayaran.

const hariIni = () => new Date().toLocaleDateString('sv-SE', { timeZone: 'Asia/Jakarta' })
const WARNA = ['var(--accent-green)', 'var(--text-warn)', 'var(--accent-amber)', 'var(--text-danger)']

export function UmurPiutang({ bisaUbah, onChanged }: { bisaUbah: boolean; onChanged: () => void }) {
  const [tab, setTab] = useState<'piutang' | 'utang' | 'manual'>('piutang')
  const jenis = tab === 'utang' ? 'utang' : 'piutang'
  const [per, setPer] = useState(hariIni)
  const [muatUlang, setMuatUlang] = useState(0)
  const [hasil, setHasil] = useState<{ kunci: string; d?: UmurSaldo; error?: string } | null>(null)
  const [lunasi, setLunasi] = useState<UmurSaldo['baris'][number] | null>(null)
  const kunci = `${jenis}|${per}|${muatUlang}`
  useEffect(() => {
    let batal = false
    getUmurSaldo(jenis, per).then((d) => !batal && setHasil({ kunci, d }), (e: Error) => !batal && setHasil({ kunci, error: e.message }))
    return () => { batal = true }
  }, [jenis, per, kunci])
  const d = hasil?.d
  const kata = jenis === 'piutang' ? 'piutang' : 'utang'

  return (
    <>
      {tab !== 'manual' && <PageHeader title="Piutang & Utang"
        subtitle={jenis === 'piutang' ? 'Tagihan ke pelanggan yang belum dibayar (dari transaksi belum lunas), dikelompokkan menurut lamanya. Tagih yang paling lama dulu.'
          : 'Kewajiban ke supplier yang belum dibayar (dari transaksi belum lunas), dikelompokkan menurut lamanya.'} />}
      <div className="flex flex-wrap items-end gap-3" style={{ marginBottom: 12 }}>
        <div className="tabbar" role="tablist" aria-label="Jenis">
          <button role="tab" className="tab-btn" aria-selected={tab === 'piutang'} onClick={() => setTab('piutang')}>Piutang (pelanggan)</button>
          <button role="tab" className="tab-btn" aria-selected={tab === 'utang'} onClick={() => setTab('utang')}>Utang (supplier)</button>
          <button role="tab" className="tab-btn" aria-selected={tab === 'manual'} onClick={() => setTab('manual')}>Tagihan manual (lama)</button>
        </div>
        {tab !== 'manual' && <div className="grid gap-1" style={{ marginLeft: 'auto' }}>
          <Label htmlFor="umur-per" className="text-xs">Per tanggal</Label>
          <Input id="umur-per" type="date" value={per} onChange={(e) => e.target.value && setPer(e.target.value)} />
        </div>}
      </div>
      {tab === 'manual' && <PiutangUtang manual />}
      {hasil?.error && <div className="pill-status pill-red" role="alert">{hasil.error}</div>}

      {d && tab !== 'manual' && <>
        <div className="kpi-grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))', marginBottom: 16 }}>
          <div className="card" style={{ padding: '14px 18px' }}>
            <div className="kpi-label">Total {kata} belum lunas</div>
            <div className="kpi-value tabular" style={{ fontSize: 20 }}>{formatRp(d.total)}</div>
            <div className="kpi-meta-row"><span>sama dengan {kata} di Neraca per {formatTanggal(d.per)}</span></div>
          </div>
          {d.kelompok.map((k, i) => (
            <div className="card" key={k.label} style={{ padding: '14px 18px' }}>
              <div className="kpi-label">{k.label}</div>
              <div className="kpi-value tabular" style={{ fontSize: 20, color: k.nilai ? WARNA[i] : undefined }}>{formatRp(k.nilai)}</div>
              <div className="kpi-meta-row"><span>{k.jumlah} transaksi{d.total ? ` · ${Math.round((k.nilai / d.total) * 100)}%` : ''}</span></div>
            </div>
          ))}
        </div>

        <div className="card" style={{ padding: 20, marginBottom: 16 }}>
          <div className="chart-title">Per {jenis === 'piutang' ? 'pelanggan' : 'supplier'}</div>
          <div className="chart-subtitle" style={{ marginBottom: 10 }}>Isi nama pelanggan/supplier di transaksi supaya rinciannya terbaca.</div>
          <div className="table-scroll">
            <table className="data-table tabular" style={{ minWidth: 760 }}>
              <thead><tr><th>Nama</th>{d.kelompok.map((k) => <th key={k.label} style={{ textAlign: 'right' }}>{k.label}</th>)}<th style={{ textAlign: 'right' }}>Total</th></tr></thead>
              <tbody>
                {d.pihak.map((p) => (
                  <tr key={p.nama}>
                    <td className="font-semibold">{p.nama}</td>
                    {p.nilai.map((v, i) => <td key={i} style={{ textAlign: 'right', color: v ? WARNA[i] : 'var(--text-muted)' }}>{v ? formatRp(v) : '·'}</td>)}
                    <td style={{ textAlign: 'right', fontWeight: 700 }}>{formatRp(p.total)}</td>
                  </tr>
                ))}
                {d.pihak.length === 0 && <tr><td colSpan={6} className="text-center text-muted-foreground" style={{ padding: 20 }}>Tidak ada {kata} yang belum lunas. 🎉</td></tr>}
              </tbody>
            </table>
          </div>
        </div>

        <div className="card" style={{ padding: 20 }}>
          <div className="chart-title">Rincian transaksi belum lunas</div>
          <div className="chart-subtitle" style={{ marginBottom: 10 }}>
            Sudah {jenis === 'piutang' ? 'dibayar pelanggan' : 'dibayar ke supplier'}? Klik <strong>Catat pelunasan</strong> dan isi tanggal bayarnya — kas tercatat di bulan itu.
          </div>
          <div className="table-scroll" style={{ opacity: hasil?.kunci === kunci ? 1 : 0.6 }}>
            <table className="data-table tabular" style={{ minWidth: 820 }}>
              <thead><tr><th>Tanggal</th><th>Nama</th><th>Keterangan</th><th style={{ textAlign: 'right' }}>Umur</th><th style={{ textAlign: 'right' }}>Nilai</th>{bisaUbah && <th />}</tr></thead>
              <tbody>
                {d.baris.map((b, i) => (
                  <tr key={`${b.transaksi_id ?? 'awal'}-${i}`}>
                    <td>{formatTanggal(b.tanggal)}</td>
                    <td>{b.pihak}</td>
                    <td className="text-sm" style={{ whiteSpace: 'normal', maxWidth: 280 }}>{b.keterangan}{b.jenis && <span className="text-xs text-muted-foreground"> · {b.jenis}</span>}</td>
                    <td style={{ textAlign: 'right', color: WARNA[b.kelompok], fontWeight: 600 }}>{b.hari} hari</td>
                    <td style={{ textAlign: 'right' }}>{formatRp(b.nilai)}</td>
                    {bisaUbah && (
                      <td style={{ textAlign: 'right' }}>
                        <Button size="sm" variant="outline" onClick={() => setLunasi(b)}>Catat pelunasan</Button>
                      </td>
                    )}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </>}

      {lunasi && (
        <PelunasanDialog baris={lunasi} piutang={jenis === 'piutang'} onClose={() => setLunasi(null)}
          onDone={() => { setLunasi(null); setMuatUlang((n) => n + 1); onChanged() }} />
      )}
    </>
  )
}

function PelunasanDialog({ baris, piutang, onClose, onDone }: { baris: UmurSaldo['baris'][number]; piutang: boolean; onClose: () => void; onDone: () => void }) {
  const [tanggal, setTanggal] = useState(hariIni)
  const saldoAwal = !baris.transaksi_id
  const [nominal, setNominal] = useState(baris.nilai)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  async function kirim(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      if (saldoAwal) await pelunasanSaldoAwal({ jenis: piutang ? 'piutang' : 'utang', tanggal, nominal })
      else await lunasiTransaksi(baris.transaksi_id!, tanggal)
      onDone()
    } catch (err) {
      setError((err as Error).message)
      setBusy(false)
    }
  }
  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent>
        <form onSubmit={kirim} className="grid gap-3">
          <DialogHeader>
            <DialogTitle>Catat pelunasan</DialogTitle>
            <DialogDescription>
              {saldoAwal
                ? `Pelunasan ${piutang ? 'piutang' : 'utang'} dari saldo awal (sisa ${formatRp(baris.nilai)}). Boleh sebagian.`
                : `${piutang ? 'Pelanggan' : 'Anda'} ${baris.pihak !== '(tanpa nama)' ? `(${baris.pihak}) ` : ''}melunasi ${formatRp(baris.nilai)} dari transaksi ${formatTanggal(baris.tanggal)}.`}
              {' '}Kas {piutang ? 'masuk' : 'keluar'} dicatat pada tanggal bayar.
            </DialogDescription>
          </DialogHeader>
          {saldoAwal && (
            <div className="grid gap-1.5" style={{ maxWidth: 220 }}>
              <Label htmlFor="lunas-nominal">Nominal dibayar</Label>
              <Input id="lunas-nominal" className="tabular" {...angka(nominal, setNominal)} />
            </div>
          )}
          <div className="grid gap-1.5" style={{ maxWidth: 200 }}>
            <Label htmlFor="lunas-tgl">Tanggal bayar</Label>
            <Input id="lunas-tgl" type="date" min={baris.tanggal} required value={tanggal} onChange={(e) => setTanggal(e.target.value)} />
          </div>
          {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>Batal</Button>
            <Button type="submit" disabled={busy}>{busy ? 'Menyimpan…' : 'Simpan pelunasan'}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
