import { useEffect, useMemo, useState, type FormEvent } from 'react'
import { PageHeader } from '@/components/PageHeader'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  batalPembayaran, buatPembayaran, getInvoices, getOpsiCatat, getPembayaran, getTagihan, getUmurSaldo, kunciBaru, pelunasanSaldoAwal,
  type AkunCatat, type InvoiceRow, type Pembayaran, type UmurSaldo, type View,
} from '@/lib/api'
import { angka } from '@/lib/angka'
import { PiutangUtang } from '@/views/PiutangStok'
import { formatRp, formatTanggal } from '@/lib/format'

// Umur piutang & utang: sisa tagihan yang belum lunas (+ saldo awal) dikelompokkan menurut lama
// lewat jatuh temponya. Totalnya sama dengan piutang/utang di Neraca. Pembayaran (boleh sebagian,
// boleh ke beberapa tagihan sekaligus) dicatat di sini dengan tanggal bayar, jadi kas bergerak di
// bulan pembayaran.

type Baris = UmurSaldo['baris'][number]
const hariIni = () => new Date().toLocaleDateString('sv-SE', { timeZone: 'Asia/Jakarta' })
// Per kelompok: belum jatuh tempo, 1–30, 31–60, 61–90, > 90 hari lewat jatuh tempo.
const WARNA = ['var(--text-muted)', 'var(--text-warn)', 'var(--accent-amber)', 'var(--text-danger)', 'var(--text-danger)']
const umurTeks = (b: Baris) => (b.hari > 0 ? `lewat ${b.hari} hari` : b.hari === 0 ? 'jatuh tempo hari ini' : `${-b.hari} hari lagi`)

export function UmurPiutang({ bisaUbah, onChanged, navigate }: { bisaUbah: boolean; onChanged: () => void; navigate?: (v: View) => void }) {
  const [tab, setTab] = useState<'piutang' | 'utang' | 'manual'>('piutang')
  const jenis = tab === 'utang' ? 'utang' : 'piutang'
  const [per, setPer] = useState(hariIni)
  const [muatUlang, setMuatUlang] = useState(0)
  // Invoice yang belum jadi piutang (draft / belum dibukukan): dijelaskan supaya angka di sini
  // tidak terlihat "tidak sinkron" dengan menu Invoice. Tanpa akses Invoice, catatan disembunyikan.
  const [invoiceLuar, setInvoiceLuar] = useState<InvoiceRow[]>([])
  useEffect(() => {
    getInvoices().then((i) => setInvoiceLuar((i.summary.invoices ?? []).filter((r) => r.status === 'draft' || (!r.dibukukan && r.status !== 'lunas'))), () => {})
  }, [muatUlang])
  // Tab tagihan manual (lama) hanya untuk usaha yang masih punya datanya.
  const [adaManual, setAdaManual] = useState(false)
  useEffect(() => { getTagihan().then((t) => setAdaManual((t.items ?? []).length > 0), () => {}) }, [])
  const [hasil, setHasil] = useState<{ kunci: string; d?: UmurSaldo; error?: string } | null>(null)
  const [lunasi, setLunasi] = useState<Baris | null>(null)
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
        subtitle={jenis === 'piutang' ? 'Pantau & tagih: sisa piutang pelanggan dari invoice terkirim dan penjualan belum lunas, dikelompokkan menurut lama lewat jatuh tempo. Tagih yang paling lama dulu.'
          : 'Pantau & bayar: sisa utang ke supplier dari pembelian belum lunas, dikelompokkan menurut lama lewat jatuh tempo.'}>
        {tab === 'piutang' && navigate && <Button onClick={() => navigate('invoice')}>+ Buat Invoice</Button>}
      </PageHeader>}
      {tab === 'piutang' && invoiceLuar.length > 0 && (
        <div className="note-box warn" style={{ marginTop: 0, marginBottom: 12 }}>
          Belum termasuk di sini: {invoiceLuar.map((r) => `${r.nomor} (${r.status === 'draft' ? 'masih draft' : 'belum dibukukan'})`).join(', ')}.
          Piutang dihitung dari invoice yang sudah <strong>Terkirim</strong> dan dibukukan.{' '}
          {navigate && <button type="button" className="underline underline-offset-4" onClick={() => navigate('invoice')}>Buka Invoice</button>}
        </div>
      )}
      <div className="flex flex-wrap items-end gap-3" style={{ marginBottom: 12 }}>
        <div className="tabbar" role="tablist" aria-label="Jenis">
          <button role="tab" className="tab-btn" aria-selected={tab === 'piutang'} onClick={() => setTab('piutang')}>Piutang (pelanggan)</button>
          <button role="tab" className="tab-btn" aria-selected={tab === 'utang'} onClick={() => setTab('utang')}>Utang (supplier)</button>
          {adaManual && <button role="tab" className="tab-btn" aria-selected={tab === 'manual'} onClick={() => setTab('manual')}>Tagihan manual (lama)</button>}
        </div>
        {tab !== 'manual' && <div className="grid gap-1" style={{ marginLeft: 'auto' }}>
          <Label htmlFor="umur-per" className="text-xs">Per tanggal</Label>
          <Input id="umur-per" type="date" value={per} onChange={(e) => e.target.value && setPer(e.target.value)} />
        </div>}
      </div>
      {tab === 'manual' && <PiutangUtang />}
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
                {d.pihak.length === 0 && <tr><td colSpan={6} className="text-center text-muted-foreground" style={{ padding: 20 }}>Tidak ada {kata} yang belum lunas.</td></tr>}
              </tbody>
            </table>
          </div>
        </div>

        <div className="card" style={{ padding: 20 }}>
          <div className="chart-title">Rincian transaksi belum lunas</div>
          <div className="chart-subtitle" style={{ marginBottom: 10 }}>
            Sudah {jenis === 'piutang' ? 'dibayar pelanggan' : 'dibayar ke supplier'}? Klik <strong>Catat pembayaran</strong> — boleh sebagian, boleh beberapa tagihan sekaligus. Kas tercatat di tanggal bayar.
          </div>
          <div className="table-scroll" style={{ opacity: hasil?.kunci === kunci ? 1 : 0.6 }}>
            <table className="data-table tabular" style={{ minWidth: 820 }}>
              <thead><tr><th>Tanggal</th><th>Nama</th><th>Keterangan</th><th style={{ textAlign: 'right' }}>Jatuh tempo</th><th style={{ textAlign: 'right' }}>Sisa</th>{bisaUbah && <th />}</tr></thead>
              <tbody>
                {d.baris.map((b, i) => (
                  <tr key={`${b.transaksi_id ?? 'awal'}-${i}`}>
                    <td>{formatTanggal(b.tanggal)}</td>
                    <td>{b.pihak}</td>
                    <td className="text-sm" style={{ whiteSpace: 'normal', maxWidth: 280 }}>{b.keterangan}{b.jenis && <span className="text-xs text-muted-foreground"> · {b.jenis}</span>}</td>
                    <td style={{ textAlign: 'right' }}>
                      {b.jatuh_tempo && <div className="text-xs text-muted-foreground">{formatTanggal(b.jatuh_tempo)}</div>}
                      <span style={{ color: WARNA[b.kelompok], fontWeight: 600 }}>{umurTeks(b)}</span>
                    </td>
                    <td style={{ textAlign: 'right' }}>
                      {formatRp(b.nilai)}
                      {!!b.dibayar && <div className="text-xs text-muted-foreground">sudah dibayar {formatRp(b.dibayar)}</div>}
                    </td>
                    {bisaUbah && (
                      <td style={{ textAlign: 'right' }}>
                        <Button size="sm" variant="outline" onClick={() => setLunasi(b)}>Catat pembayaran</Button>
                      </td>
                    )}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </>}

      {d && tab !== 'manual' && (
        <RiwayatPembayaran jenis={jenis} versi={muatUlang} bisaUbah={bisaUbah} onChanged={() => { setMuatUlang((n) => n + 1); onChanged() }} />
      )}

      {lunasi && (lunasi.transaksi_id ? (
        <PembayaranDialog baris={lunasi} semua={d?.baris ?? []} piutang={jenis === 'piutang'} onClose={() => setLunasi(null)}
          onDone={() => { setLunasi(null); setMuatUlang((n) => n + 1); onChanged() }} />
      ) : (
        <PelunasanDialog baris={lunasi} piutang={jenis === 'piutang'} onClose={() => setLunasi(null)}
          onDone={() => { setLunasi(null); setMuatUlang((n) => n + 1); onChanged() }} />
      ))}
    </>
  )
}

/** Pelunasan piutang/utang saldo awal (tidak punya transaksi asal). */
function PelunasanDialog({ baris, piutang, onClose, onDone }: { baris: Baris; piutang: boolean; onClose: () => void; onDone: () => void }) {
  const [tanggal, setTanggal] = useState(hariIni)
  const [nominal, setNominal] = useState(baris.nilai)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  async function kirim(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      await pelunasanSaldoAwal({ jenis: piutang ? 'piutang' : 'utang', tanggal, nominal })
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
            <DialogTitle>Catat pembayaran saldo awal</DialogTitle>
            <DialogDescription>
              Pelunasan {piutang ? 'piutang' : 'utang'} dari saldo awal (sisa {formatRp(baris.nilai)}). Boleh sebagian. Kas {piutang ? 'masuk' : 'keluar'} dicatat pada tanggal bayar.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-1.5" style={{ maxWidth: 220 }}>
            <Label htmlFor="lunas-nominal">Nominal dibayar</Label>
            <Input id="lunas-nominal" className="tabular" {...angka(nominal, setNominal)} />
          </div>
          <div className="grid gap-1.5" style={{ maxWidth: 200 }}>
            <Label htmlFor="lunas-tgl">Tanggal bayar</Label>
            <Input id="lunas-tgl" type="date" min={baris.tanggal} required value={tanggal} onChange={(e) => setTanggal(e.target.value)} />
          </div>
          {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>Batal</Button>
            <Button type="submit" disabled={busy}>{busy ? 'Menyimpan…' : 'Simpan pembayaran'}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

/**
 * Pembayaran sebagian / beberapa tagihan sekaligus: tagihan terbuka pihak yang sama, jatuh tempo
 * terlama dulu. Satu Idempotency-Key per buka dialog, jadi klik ganda / retry tidak tercatat dua kali.
 */
function PembayaranDialog({ baris, semua, piutang, onClose, onDone }: { baris: Baris; semua: Baris[]; piutang: boolean; onClose: () => void; onDone: () => void }) {
  const kunci = useMemo(() => kunciBaru(), [])
  const tagihan = useMemo(() => {
    const sama = baris.pihak === '(tanpa nama)' ? [baris] : semua.filter((b) => b.transaksi_id && b.pihak === baris.pihak)
    return [...sama].sort((a, b) => (a.jatuh_tempo ?? a.tanggal).localeCompare(b.jatuh_tempo ?? b.tanggal))
  }, [semua, baris])
  const [nilai, setNilai] = useState<Record<number, number>>({ [baris.transaksi_id!]: baris.nilai })
  const [tanggal, setTanggal] = useState(hariIni)
  const [lawan, setLawan] = useState('')
  const [rekening, setRekening] = useState<AkunCatat[]>([])
  const [bayarTotal, setBayarTotal] = useState(baris.nilai)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  useEffect(() => { getOpsiCatat().then((o) => setRekening(o.akun.filter((a) => a.lawan === 'rekening')), () => {}) }, [])
  const total = Object.values(nilai).reduce((a, b) => a + (b || 0), 0)
  // Bagi otomatis: isi tagihan dengan jatuh tempo terlama dulu sampai uangnya habis.
  function bagi(n: number) {
    let sisa = n
    const out: Record<number, number> = {}
    for (const b of tagihan) {
      const ambil = Math.min(sisa, b.nilai)
      if (ambil > 0) out[b.transaksi_id!] = ambil
      sisa -= ambil
    }
    setNilai(out)
  }
  async function kirim(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      const alokasi = Object.entries(nilai).filter(([, v]) => v > 0).map(([id, v]) => ({ transaksi_id: Number(id), nominal: v }))
      await buatPembayaran({ tanggal, jenis: piutang ? 'piutang' : 'utang', lawan: lawan || undefined, alokasi }, kunci)
      onDone()
    } catch (err) {
      setError((err as Error).message)
      setBusy(false)
    }
  }
  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="sm:max-w-2xl">
        <form onSubmit={kirim} className="grid gap-3">
          <DialogHeader>
            <DialogTitle>Catat pembayaran {piutang ? 'dari pelanggan' : 'ke supplier'}</DialogTitle>
            <DialogDescription>
              {baris.pihak !== '(tanpa nama)' ? `${baris.pihak}. ` : ''}Boleh sebagian, boleh beberapa tagihan sekaligus. Kas {piutang ? 'masuk' : 'keluar'} dicatat pada tanggal bayar;
              pembayaran yang salah bisa dibatalkan dengan alasan selama bulannya belum ditutup.
            </DialogDescription>
          </DialogHeader>
          <div className="flex flex-wrap gap-3">
            <div className="grid gap-1.5" style={{ width: 180 }}>
              <Label htmlFor="bayar-tgl">Tanggal bayar</Label>
              <Input id="bayar-tgl" type="date" required value={tanggal} onChange={(e) => setTanggal(e.target.value)} />
            </div>
            <div className="grid gap-1.5" style={{ minWidth: 200, flex: 1 }}>
              <Label htmlFor="bayar-rek">Rekening kas / bank</Label>
              <select id="bayar-rek" className="input-field" value={lawan} onChange={(e) => setLawan(e.target.value)}>
                <option value="">Otomatis</option>
                {rekening.map((a) => <option key={a.kode} value={a.kode}>{a.kode} {a.nama}</option>)}
              </select>
            </div>
            {tagihan.length > 1 && (
              <div className="grid gap-1.5" style={{ width: 200 }}>
                <Label htmlFor="bayar-total">Bagi otomatis</Label>
                <div className="flex gap-2">
                  <Input id="bayar-total" className="tabular" {...angka(bayarTotal, setBayarTotal)} />
                  <Button type="button" variant="outline" onClick={() => bagi(bayarTotal)}>Bagi</Button>
                </div>
              </div>
            )}
          </div>
          <div className="table-scroll">
            <table className="data-table tabular" style={{ minWidth: 520 }}>
              <thead><tr><th>Tagihan</th><th style={{ textAlign: 'right' }}>Jatuh tempo</th><th style={{ textAlign: 'right' }}>Sisa</th><th style={{ textAlign: 'right', width: 170 }}>Dibayar sekarang</th></tr></thead>
              <tbody>
                {tagihan.map((b) => (
                  <tr key={b.transaksi_id}>
                    <td className="text-sm" style={{ whiteSpace: 'normal' }}>{formatTanggal(b.tanggal)} · {b.keterangan || b.jenis}</td>
                    <td style={{ textAlign: 'right' }}>{b.jatuh_tempo ? formatTanggal(b.jatuh_tempo) : '—'}</td>
                    <td style={{ textAlign: 'right' }}>{formatRp(b.nilai)}</td>
                    <td style={{ textAlign: 'right' }}>
                      <Input className="tabular" aria-label={`Dibayar untuk tagihan ${formatTanggal(b.tanggal)}`} style={{ textAlign: 'right' }}
                        {...angka(nilai[b.transaksi_id!] ?? 0, (n) => setNilai((x) => ({ ...x, [b.transaksi_id!]: n })))} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div className="text-sm" style={{ textAlign: 'right' }}>Total pembayaran: <strong className="tabular">{formatRp(total)}</strong></div>
          {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>Batal</Button>
            <Button type="submit" disabled={busy || total <= 0}>{busy ? 'Menyimpan…' : 'Simpan pembayaran'}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

/** Riwayat dokumen pembayaran; yang salah dibatalkan dengan alasan (sekali, bulan belum ditutup). */
function RiwayatPembayaran({ jenis, versi, bisaUbah, onChanged }: { jenis: 'piutang' | 'utang'; versi: number; bisaUbah: boolean; onChanged: () => void }) {
  const [daftar, setDaftar] = useState<Pembayaran[]>([])
  const [batal, setBatal] = useState<Pembayaran | null>(null)
  useEffect(() => { getPembayaran(jenis).then(setDaftar, () => setDaftar([])) }, [jenis, versi])
  if (daftar.length === 0) return null
  return (
    <div className="card" style={{ padding: 20, marginTop: 16 }}>
      <div className="chart-title">Riwayat pembayaran</div>
      <div className="chart-subtitle" style={{ marginBottom: 10 }}>Pembayaran yang dibatalkan tetap tercatat beserta alasannya.</div>
      <div className="table-scroll">
        <table className="data-table tabular" style={{ minWidth: 720 }}>
          <thead><tr><th>No.</th><th>Tanggal</th><th>Nama</th><th>Tagihan</th><th style={{ textAlign: 'right' }}>Total</th><th>Status</th>{bisaUbah && <th />}</tr></thead>
          <tbody>
            {daftar.map((p) => (
              <tr key={p.id} style={p.batal ? { opacity: 0.6 } : undefined}>
                <td>PB-{p.id}</td>
                <td>{formatTanggal(p.tanggal)}</td>
                <td>{p.pihak || '—'}</td>
                <td className="text-xs">{p.alokasi.map((a) => `#${a.transaksi_id} ${formatRp(a.nominal)}`).join(', ')}{p.lawan ? ` · rek. ${p.lawan}` : ''}</td>
                <td style={{ textAlign: 'right' }}>{formatRp(p.total)}</td>
                <td className="text-sm" style={{ whiteSpace: 'normal', maxWidth: 240 }}>
                  {p.batal ? <span className="pill-status pill-red">Dibatalkan</span> : <span className="pill-status pill-green">Tercatat</span>}
                  {p.batal && <div className="text-xs text-muted-foreground">{p.batal.alasan}{p.batal.oleh ? ` · ${p.batal.oleh}` : ''}</div>}
                </td>
                {bisaUbah && <td style={{ textAlign: 'right' }}>{!p.batal && <Button size="sm" variant="outline" onClick={() => setBatal(p)}>Batalkan</Button>}</td>}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {batal && <BatalDialog p={batal} onClose={() => setBatal(null)} onDone={() => { setBatal(null); onChanged() }} />}
    </div>
  )
}

function BatalDialog({ p, onClose, onDone }: { p: Pembayaran; onClose: () => void; onDone: () => void }) {
  const [alasan, setAlasan] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  async function kirim(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      await batalPembayaran(p.id, alasan)
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
            <DialogTitle>Batalkan PB-{p.id}?</DialogTitle>
            <DialogDescription>
              Pembayaran {formatRp(p.total)} tanggal {formatTanggal(p.tanggal)} dicabut dari tagihannya dan kas dikembalikan. Dokumen tetap tercatat sebagai dibatalkan. Tidak bisa diulang.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-1.5">
            <Label htmlFor="batal-alasan">Alasan</Label>
            <Input id="batal-alasan" required maxLength={300} value={alasan} onChange={(e) => setAlasan(e.target.value)} placeholder="mis. salah nominal, transfer ditolak bank" />
          </div>
          {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>Kembali</Button>
            <Button type="submit" variant="destructive" disabled={busy || !alasan.trim()}>{busy ? 'Membatalkan…' : 'Batalkan pembayaran'}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
