import { useEffect, useMemo, useState, type FormEvent } from 'react'
import { ChevronLeft, ChevronRight, FileSpreadsheet, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import {
  createTransaction, getGerakStok, getKartuBarang, getLaporanProduksi, getMutasiStok, hapusGerakStok, laporanProduksiXlsxUrl, mutasiStokXlsxUrl,
  stokAwal, stokOpname, stokProduksi,
  type BarisKartu, type GerakStok, type JenisKartu, type KartuBarang, type LaporanMutasi, type LaporanProduksi, type ProdukRow, type StokRow,
} from '@/lib/api'
import { angka } from '@/lib/angka'
import { formatRp, formatTanggal } from '@/lib/format'
import { ekspor } from '@/lib/garansi'
import { formatQty, parseQty, qtyKeTeks } from '@/lib/qty'

// Kartu stok: titik nol (stok awal), opname, beli bahan, kartu per barang, laporan mutasi,
// dan riwayat dokumen stok. Beli bahan dicatat sebagai transaksi "Beli Bahan / Stok" yang
// memilih barang, jadi kas/utang dan stok berasal dari satu catatan yang sama.

const hariIni = () => new Date().toLocaleDateString('sv-SE', { timeZone: 'Asia/Jakarta' })
const RED = 'var(--text-danger)'
const GREEN = 'var(--accent-green)'

const LABEL_KARTU: Record<JenisKartu, string> = {
  awal: 'Stok awal', beli: 'Beli', produksi_masuk: 'Hasil produksi', produksi_pakai: 'Dipakai produksi', jual: 'Terjual', opname: 'Stok opname',
}

function Galat({ teks }: { teks: string }) {
  return teks ? <p className="text-sm text-destructive" role="alert">{teks}</p> : null
}

/** Titik nol kartu stok: hitung fisik pertama + harga pokok per satuan. */
export function MulaiKartuDialog({ items, onClose, onDone }: { items: StokRow[]; onClose: () => void; onDone: () => void }) {
  const [tanggal, setTanggal] = useState(hariIni)
  const [isi, setIsi] = useState(() => items.map((it) => ({ id: it.id, fisik: qtyKeTeks(it.jumlah_mili), harga: it.harga_pokok })))
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const set = (i: number, p: Partial<(typeof isi)[number]>) => setIsi((x) => x.map((r, j) => (j === i ? { ...r, ...p } : r)))
  const total = isi.reduce((n, r) => n + Math.round((parseQty(r.fisik) * r.harga) / 1000), 0)

  async function kirim(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      await stokAwal({ tanggal, baris: isi.map((r) => ({ stok_id: r.id, fisik: parseQty(r.fisik), harga_unit: r.harga })) })
      onDone()
    } catch (err) {
      setError((err as Error).message)
      setBusy(false)
    }
  }
  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="sm:max-w-3xl">
        <form onSubmit={kirim} className="grid gap-3">
          <DialogHeader>
            <DialogTitle>Mulai kartu stok (titik nol)</DialogTitle>
            <DialogDescription>
              Hitung fisik semua barang di gudang pada tanggal ini, lalu isi jumlah & harga pokoknya. Mulai tanggal ini stok bergerak otomatis
              dari transaksi beli & jual, produksi, dan stok opname. Laporan sebelum tanggal ini tidak berubah.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-1.5" style={{ maxWidth: 200 }}>
            <Label htmlFor="ks-tgl">Tanggal hitung fisik</Label>
            <Input id="ks-tgl" type="date" required value={tanggal} onChange={(e) => setTanggal(e.target.value)} />
          </div>
          <div className="table-scroll" style={{ maxHeight: 360 }}>
            <table className="data-table tabular" style={{ minWidth: 560 }}>
              <thead><tr><th>Barang</th><th style={{ textAlign: 'right' }}>Jumlah fisik</th><th style={{ textAlign: 'right' }}>Harga pokok / satuan</th><th style={{ textAlign: 'right' }}>Nilai</th></tr></thead>
              <tbody>
                {items.map((it, i) => (
                  <tr key={it.id}>
                    <td><div className="font-semibold">{it.nama}</div><div className="text-xs text-muted-foreground">{it.sku || it.kategori}</div></td>
                    <td style={{ textAlign: 'right' }}>
                      <span className="inline-flex items-center gap-1.5">
                        <Input aria-label={`Jumlah fisik ${it.nama}`} inputMode="decimal" className="tabular" style={{ width: 110, textAlign: 'right' }}
                          value={isi[i].fisik} onChange={(e) => set(i, { fisik: e.target.value.replace(/[^\d.,]/g, '') })} />
                        <span className="text-xs text-muted-foreground" style={{ minWidth: 30 }}>{it.satuan}</span>
                      </span>
                    </td>
                    <td style={{ textAlign: 'right' }}>
                      <Input aria-label={`Harga pokok ${it.nama}`} className="tabular" style={{ width: 130, textAlign: 'right', marginLeft: 'auto' }}
                        {...angka(isi[i].harga, (n) => set(i, { harga: n }))} />
                    </td>
                    <td style={{ textAlign: 'right' }}>{formatRp(Math.round((parseQty(isi[i].fisik) * isi[i].harga) / 1000))}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <p className="text-sm">Nilai persediaan awal: <strong>{formatRp(total)}</strong></p>
          <Galat teks={error} />
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>Batal</Button>
            <Button type="submit" disabled={busy || items.length === 0}>{busy ? 'Menyimpan…' : 'Mulai kartu stok'}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

/** Stok opname: isi hasil hitung fisik; barang yang dikosongkan tidak ikut dihitung. */
export function OpnameDialog({ items, mulai, onClose, onDone }: { items: StokRow[]; mulai: string; onClose: () => void; onDone: () => void }) {
  const [tanggal, setTanggal] = useState(hariIni)
  const [ket, setKet] = useState('')
  const [fisik, setFisik] = useState<Record<number, string>>({})
  const [cari, setCari] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const tampil = items.filter((it) => (it.nama + ' ' + it.sku).toLowerCase().includes(cari.trim().toLowerCase()))
  const diisi = Object.entries(fisik).filter(([, v]) => v.trim() !== '')

  async function kirim(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      await stokOpname({ tanggal, keterangan: ket, baris: diisi.map(([id, v]) => ({ stok_id: Number(id), fisik: parseQty(v) })) })
      onDone()
    } catch (err) {
      setError((err as Error).message)
      setBusy(false)
    }
  }
  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="sm:max-w-3xl">
        <form onSubmit={kirim} className="grid gap-3">
          <DialogHeader>
            <DialogTitle>Stok opname</DialogTitle>
            <DialogDescription>Isi jumlah hasil hitung fisik. Selisih dengan stok sistem dicatat sebagai penyesuaian (barang hilang, rusak, atau salah catat). Barang yang tidak diisi tidak berubah.</DialogDescription>
          </DialogHeader>
          <div className="flex flex-wrap gap-3">
            <div className="grid gap-1.5"><Label htmlFor="op-tgl">Tanggal</Label>
              <Input id="op-tgl" type="date" min={mulai} required value={tanggal} onChange={(e) => setTanggal(e.target.value)} /></div>
            <div className="grid flex-1 gap-1.5" style={{ minWidth: 200 }}><Label htmlFor="op-ket">Keterangan</Label>
              <Input id="op-ket" placeholder="mis. Opname akhir bulan" value={ket} onChange={(e) => setKet(e.target.value)} /></div>
            <div className="grid gap-1.5"><Label htmlFor="op-cari">Cari barang</Label>
              <Input id="op-cari" value={cari} onChange={(e) => setCari(e.target.value)} /></div>
          </div>
          <div className="table-scroll" style={{ maxHeight: 360 }}>
            <table className="data-table tabular" style={{ minWidth: 560 }}>
              <thead><tr><th>Barang</th><th style={{ textAlign: 'right' }}>Stok sistem</th><th style={{ textAlign: 'right' }}>Hitung fisik</th><th style={{ textAlign: 'right' }}>Selisih</th></tr></thead>
              <tbody>
                {tampil.map((it) => {
                  const v = fisik[it.id] ?? ''
                  const selisih = v.trim() === '' ? null : parseQty(v) - it.jumlah_mili
                  return (
                    <tr key={it.id}>
                      <td><div className="font-semibold">{it.nama}</div><div className="text-xs text-muted-foreground">{it.sku || it.kategori}</div></td>
                      <td style={{ textAlign: 'right', color: it.minus ? RED : undefined }}>{formatQty(it.jumlah_mili)} {it.satuan}</td>
                      <td style={{ textAlign: 'right' }}>
                        <Input aria-label={`Hitung fisik ${it.nama}`} inputMode="decimal" className="tabular" style={{ width: 110, textAlign: 'right', marginLeft: 'auto' }}
                          value={v} placeholder="—" onChange={(e) => setFisik({ ...fisik, [it.id]: e.target.value.replace(/[^\d.,]/g, '') })} />
                      </td>
                      <td style={{ textAlign: 'right', color: selisih === null || selisih === 0 ? 'var(--text-muted)' : selisih < 0 ? RED : GREEN }}>
                        {selisih === null ? '' : `${selisih > 0 ? '+' : ''}${formatQty(selisih)}`}
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
          <Galat teks={error} />
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>Batal</Button>
            <Button type="submit" disabled={busy || diisi.length === 0}>{busy ? 'Menyimpan…' : `Simpan opname (${diisi.length} barang)`}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

/** Beli bahan/barang: transaksi "Beli Bahan / Stok" yang memilih barang → stok & HPP rata-rata bergerak. */
export function BeliBahanDialog({ items, onClose, onDone }: { items: StokRow[]; onClose: () => void; onDone: () => void }) {
  const [tanggal, setTanggal] = useState(hariIni)
  const [stokId, setStokId] = useState(items[0]?.id ?? 0)
  const [qty, setQty] = useState('')
  const [total, setTotal] = useState(0)
  const [pihak, setPihak] = useState('')
  const [lunas, setLunas] = useState(true)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const it = items.find((x) => x.id === stokId)
  const q = parseQty(qty)

  async function kirim(e: FormEvent) {
    e.preventDefault()
    if (!it) return
    setBusy(true)
    setError('')
    try {
      await createTransaction({
        tanggal, jenis: 'Beli Bahan / Stok', nominal: total, keterangan: `Beli ${it.nama}`, item: it.nama, qty: 0, channel: '',
        pihak, lunas, tanggal_lunas: '', stok_id: it.id, qty_stok: q,
      })
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
            <DialogTitle>Beli bahan / barang</DialogTitle>
            <DialogDescription>Menambah stok dan memperbarui harga pokok rata-rata. Tercatat juga sebagai transaksi pengeluaran (atau utang bila belum lunas).</DialogDescription>
          </DialogHeader>
          <div className="grid gap-1.5"><Label htmlFor="bb-barang">Barang</Label>
            <select id="bb-barang" className="input-field" value={stokId} onChange={(e) => setStokId(Number(e.target.value))}>
              {items.map((x) => <option key={x.id} value={x.id}>{x.nama}{x.sku ? ` (${x.sku})` : ''}</option>)}
            </select></div>
          <div className="grid grid-cols-2 gap-3">
            <div className="grid gap-1.5"><Label htmlFor="bb-qty">Jumlah{it ? ` (${it.satuan})` : ''}</Label>
              <Input id="bb-qty" required inputMode="decimal" className="tabular" placeholder="mis. 2,5" value={qty} onChange={(e) => setQty(e.target.value.replace(/[^\d.,]/g, ''))} /></div>
            <div className="grid gap-1.5"><Label htmlFor="bb-total">Total harga (Rp)</Label>
              <Input id="bb-total" required className="tabular" {...angka(total, setTotal)} /></div>
            <div className="grid gap-1.5"><Label htmlFor="bb-tgl">Tanggal</Label>
              <Input id="bb-tgl" type="date" required value={tanggal} onChange={(e) => setTanggal(e.target.value)} /></div>
            <div className="grid gap-1.5"><Label htmlFor="bb-pihak">Supplier</Label>
              <Input id="bb-pihak" placeholder="Opsional" value={pihak} onChange={(e) => setPihak(e.target.value)} /></div>
          </div>
          {q > 0 && total > 0 && it && <p className="text-sm text-muted-foreground">Harga per {it.satuan || 'satuan'}: <strong>{formatRp(Math.round((total * 1000) / q))}</strong></p>}
          <label className="flex items-center gap-3 text-sm"><Switch checked={lunas} onCheckedChange={setLunas} /> {lunas ? 'Sudah dibayar' : 'Belum dibayar (masuk utang)'}</label>
          <Galat teks={error} />
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>Batal</Button>
            <Button type="submit" disabled={busy || !it || q <= 0 || total <= 0}>{busy ? 'Menyimpan…' : 'Simpan pembelian'}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

const awalBulan = () => hariIni().slice(0, 8) + '01'

/** Kartu stok satu barang: setiap pergerakan beserta saldo jumlah & nilainya. */
export function KartuBarangDialog({ item, onClose }: { item: StokRow; onClose: () => void }) {
  const [dari, setDari] = useState(awalBulan)
  const [sampai, setSampai] = useState(hariIni)
  const [data, setData] = useState<{ kunci: string; k?: KartuBarang; error?: string } | null>(null)
  const kunci = `${dari}|${sampai}`
  useEffect(() => {
    let batal = false
    getKartuBarang(item.id, dari, sampai).then((k) => !batal && setData({ kunci, k }), (e: Error) => !batal && setData({ kunci, error: e.message }))
    return () => { batal = true }
  }, [item.id, dari, sampai, kunci])
  const k = data?.k
  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="sm:max-w-4xl">
        <DialogHeader>
          <DialogTitle>Kartu stok — {item.nama}</DialogTitle>
          <DialogDescription>Setiap barang masuk & keluar beserta saldonya. Nilai memakai harga pokok rata-rata bergerak.</DialogDescription>
        </DialogHeader>
        <div className="flex flex-wrap items-end gap-3">
          <div className="grid gap-1.5"><Label htmlFor="kb-dari">Dari</Label><Input id="kb-dari" type="date" value={dari} onChange={(e) => setDari(e.target.value)} /></div>
          <div className="grid gap-1.5"><Label htmlFor="kb-sampai">Sampai</Label><Input id="kb-sampai" type="date" value={sampai} onChange={(e) => setSampai(e.target.value)} /></div>
        </div>
        <Galat teks={data?.error ?? ''} />
        <div className="table-scroll" style={{ maxHeight: 420, opacity: data?.kunci === kunci ? 1 : 0.6 }}>
          <table className="data-table tabular" style={{ minWidth: 760 }}>
            <thead><tr><th>Tanggal</th><th>Keterangan</th><th style={{ textAlign: 'right' }}>Masuk</th><th style={{ textAlign: 'right' }}>Keluar</th><th style={{ textAlign: 'right' }}>Saldo</th><th style={{ textAlign: 'right' }}>Nilai</th><th style={{ textAlign: 'right' }}>Saldo nilai</th></tr></thead>
            <tbody>
              {k && (
                <tr className="font-semibold"><td>{formatTanggal(dari)}</td><td>Saldo awal periode</td><td /><td />
                  <td style={{ textAlign: 'right' }}>{formatQty(k.awal.qty)}</td><td /><td style={{ textAlign: 'right' }}>{formatRp(k.awal.nilai)}</td></tr>
              )}
              {k?.baris.map((b: BarisKartu, i) => (
                <tr key={i}>
                  <td>{formatTanggal(b.tanggal)}</td>
                  <td>{LABEL_KARTU[b.jenis]}{b.channel ? ` · ${b.channel}` : ''}{b.keterangan ? <div className="text-xs text-muted-foreground">{b.keterangan}</div> : null}</td>
                  <td style={{ textAlign: 'right', color: GREEN }}>{b.qty > 0 ? formatQty(b.qty) : ''}</td>
                  <td style={{ textAlign: 'right', color: RED }}>{b.qty < 0 ? formatQty(-b.qty) : ''}</td>
                  <td style={{ textAlign: 'right', color: b.minus ? RED : undefined, fontWeight: 600 }}>{formatQty(b.saldo_qty)}</td>
                  <td style={{ textAlign: 'right' }}>{formatRp(b.nilai)}{b.koreksi ? <div className="text-xs text-muted-foreground">koreksi {formatRp(b.koreksi)}</div> : null}</td>
                  <td style={{ textAlign: 'right' }}>{formatRp(b.saldo_nilai)}</td>
                </tr>
              ))}
              {k && k.baris.length === 0 && <tr><td colSpan={7} className="text-center text-muted-foreground" style={{ padding: 20 }}>Tidak ada pergerakan di rentang ini.</td></tr>}
            </tbody>
          </table>
        </div>
      </DialogContent>
    </Dialog>
  )
}

/** Laporan mutasi stok harian / bulanan: Awal + Masuk − Keluar ± Opname = Akhir. */
export function MutasiStokPanel({ mulai, bisaEkspor }: { mulai: string; bisaEkspor: boolean }) {
  const [mode, setMode] = useState<'harian' | 'bulanan'>('bulanan')
  const [tgl, setTgl] = useState(hariIni)
  const { dari, sampai } = useMemo(() => {
    if (mode === 'harian') return { dari: tgl, sampai: tgl }
    const [y, m] = tgl.split('-').map(Number)
    const akhir = new Date(Date.UTC(y, m, 0)).getUTCDate()
    return { dari: `${tgl.slice(0, 7)}-01`, sampai: `${tgl.slice(0, 7)}-${String(akhir).padStart(2, '0')}` }
  }, [mode, tgl])
  const [data, setData] = useState<{ kunci: string; l?: LaporanMutasi; error?: string } | null>(null)
  const kunci = `${dari}|${sampai}`
  useEffect(() => {
    let batal = false
    getMutasiStok(dari, sampai).then((l) => !batal && setData({ kunci, l }), (e: Error) => !batal && setData({ kunci, error: e.message }))
    return () => { batal = true }
  }, [dari, sampai, kunci])
  const geser = (n: number) => {
    const d = new Date(tgl + 'T00:00:00Z')
    if (mode === 'harian') d.setUTCDate(d.getUTCDate() + n)
    else d.setUTCMonth(d.getUTCMonth() + n, 1)
    setTgl(d.toISOString().slice(0, 10))
  }
  const l = data?.l
  const judul = mode === 'harian'
    ? new Date(tgl + 'T00:00:00').toLocaleDateString('id-ID', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' })
    : new Date(tgl + 'T00:00:00').toLocaleDateString('id-ID', { month: 'long', year: 'numeric' })
  const sel = (q: number, warna?: string) => <td style={{ textAlign: 'right', color: q === 0 ? 'var(--text-muted)' : warna }}>{q === 0 ? '·' : formatQty(q)}</td>

  return (
    <>
      <div className="card" style={{ padding: 16 }}>
        <div className="flex flex-wrap items-end gap-3">
          <div className="segmented" role="tablist" aria-label="Tampilan">
            <button role="tab" className="seg-btn" aria-selected={mode === 'harian'} onClick={() => setMode('harian')}>Harian</button>
            <button role="tab" className="seg-btn" aria-selected={mode === 'bulanan'} onClick={() => setMode('bulanan')}>Bulanan</button>
          </div>
          <div className="grid gap-1"><Label htmlFor="ms-tgl" className="text-xs">Periode</Label>
            {mode === 'harian'
              ? <Input id="ms-tgl" type="date" value={tgl} onChange={(e) => e.target.value && setTgl(e.target.value)} />
              : <Input id="ms-tgl" type="month" value={tgl.slice(0, 7)} onChange={(e) => e.target.value && setTgl(e.target.value + '-01')} />}
          </div>
          <Button variant="outline" size="icon" aria-label="Sebelumnya" onClick={() => geser(-1)}><ChevronLeft /></Button>
          <Button variant="outline" size="icon" aria-label="Berikutnya" onClick={() => geser(1)}><ChevronRight /></Button>
          {bisaEkspor && <Button variant="outline" onClick={() => void ekspor(mutasiStokXlsxUrl(dari, sampai))}><FileSpreadsheet /> Export Excel</Button>}
          <div style={{ marginLeft: 'auto', textAlign: 'right' }}>
            <div className="font-heading text-lg font-bold" style={{ textTransform: 'capitalize' }}>{judul}</div>
            {l && <div className="text-xs text-muted-foreground">Total {l.items.length} barang · HPP terjual {formatRp(l.hpp_jual)}</div>}
          </div>
        </div>
        <p className="mt-2 text-xs text-muted-foreground">Titik nol stok: <strong>{formatTanggal(mulai)}</strong>. Pergerakan sebelum tanggal ini tidak dihitung.</p>
      </div>
      <Galat teks={data?.error ?? ''} />
      <div className="card" style={{ padding: 0, opacity: data?.kunci === kunci ? 1 : 0.6 }}>
        <div className="table-scroll">
          <table className="data-table tabular" style={{ minWidth: 1060 }}>
            <thead>
              <tr>
                <th rowSpan={2}>No</th><th rowSpan={2}>Barang</th><th rowSpan={2} style={{ textAlign: 'right' }}>Awal</th>
                <th colSpan={2} style={{ textAlign: 'center' }}>Masuk</th><th colSpan={2} style={{ textAlign: 'center' }}>Keluar</th>
                <th rowSpan={2} style={{ textAlign: 'right' }}>Opname</th><th rowSpan={2} style={{ textAlign: 'right' }}>Akhir</th><th rowSpan={2} style={{ textAlign: 'right' }}>Nilai akhir</th>
              </tr>
              <tr>
                <th style={{ textAlign: 'right' }}>Beli</th><th style={{ textAlign: 'right' }}>Produksi</th>
                <th style={{ textAlign: 'right' }}>Terjual</th><th style={{ textAlign: 'right' }}>Dipakai produksi</th>
              </tr>
            </thead>
            <tbody>
              {l?.items.map((m, i) => (
                <tr key={m.stok_id}>
                  <td>{i + 1}</td>
                  <td><div className="font-semibold">{m.nama}{m.minus && <span className="pill-status pill-red" style={{ marginLeft: 6, fontSize: 10 }}>minus</span>}</div>
                    <div className="text-xs text-muted-foreground">{[m.sku, m.satuan].filter(Boolean).join(' · ')}</div></td>
                  {sel(m.awal_qty)}
                  {sel(m.beli_qty, GREEN)}{sel(m.produksi_qty, GREEN)}
                  {sel(m.jual_qty, RED)}{sel(m.pakai_qty, RED)}
                  {sel(m.opname_qty, m.opname_qty < 0 ? RED : GREEN)}
                  <td style={{ textAlign: 'right', fontWeight: 700, color: m.akhir_qty < 0 ? RED : undefined }}>{formatQty(m.akhir_qty)}</td>
                  <td style={{ textAlign: 'right' }}>{formatRp(m.akhir_nilai)}</td>
                </tr>
              ))}
              {l && l.items.length === 0 && <tr><td colSpan={10} className="text-center text-muted-foreground" style={{ padding: 24 }}>Belum ada barang.</td></tr>}
            </tbody>
            {l && (
              <tfoot>
                <tr className="font-semibold">
                  <td colSpan={2}>TOTAL NILAI</td><td style={{ textAlign: 'right' }}>{formatRp(l.total.awal_nilai)}</td>
                  <td style={{ textAlign: 'right' }}>{formatRp(l.total.beli_nilai)}</td><td style={{ textAlign: 'right' }}>{formatRp(l.total.produksi_nilai)}</td>
                  <td style={{ textAlign: 'right' }}>{formatRp(l.total.jual_nilai)}</td><td style={{ textAlign: 'right' }}>{formatRp(l.total.pakai_nilai)}</td>
                  <td style={{ textAlign: 'right' }}>{formatRp(l.total.opname_nilai)}</td><td />
                  <td style={{ textAlign: 'right' }}>{formatRp(l.total.akhir_nilai)}</td>
                </tr>
              </tfoot>
            )}
          </table>
        </div>
      </div>
      <p className="text-xs text-muted-foreground">
        Rumus: Akhir = Awal + Beli + Hasil produksi − Terjual − Dipakai produksi ± Opname. Terjual berasal dari transaksi Penjualan yang memilih barang;
        nilai memakai harga pokok rata-rata bergerak. Jumlah antarbarang tidak dijumlahkan karena satuannya berbeda.
      </p>
    </>
  )
}

const LABEL_GERAK: Record<GerakStok['jenis'], string> = { awal: 'Stok awal (titik nol)', opname: 'Stok opname', produksi: 'Produksi' }

/** Riwayat dokumen stok (stok awal, opname, produksi) yang bisa dihapus bila salah input. */
export function RiwayatGerakPanel({ items, bisaUbah, onChanged }: { items: StokRow[]; bisaUbah: boolean; onChanged: () => void }) {
  const [list, setList] = useState<GerakStok[] | null>(null)
  const [error, setError] = useState('')
  const muat = () => getGerakStok().then(setList, (e: Error) => setError(e.message))
  useEffect(() => { void muat() }, [])
  const nama = (id: number) => items.find((x) => x.id === id)?.nama ?? `#${id}`

  async function hapus(g: GerakStok) {
    if (!window.confirm(`Hapus ${LABEL_GERAK[g.jenis].toLowerCase()} ${formatTanggal(g.tanggal)}? Stok dihitung ulang tanpa dokumen ini.`)) return
    try {
      await hapusGerakStok(g.id)
      await muat()
      onChanged()
    } catch (e) {
      setError((e as Error).message)
    }
  }
  return (
    <div className="card" style={{ padding: 20 }}>
      <div className="chart-title">Riwayat stok awal, opname & produksi</div>
      <div className="chart-subtitle" style={{ marginBottom: 10 }}>Beli & jual barang tercatat di menu Transaksi. Hapus dokumen yang salah input, stok akan dihitung ulang.</div>
      <Galat teks={error} />
      <div className="table-scroll">
        <table className="data-table" style={{ minWidth: 720 }}>
          <thead><tr><th>Tanggal</th><th>Jenis</th><th>Rincian</th><th>Oleh</th>{bisaUbah && <th />}</tr></thead>
          <tbody>
            {(list ?? []).map((g) => (
              <tr key={g.id}>
                <td>{formatTanggal(g.tanggal)}</td>
                <td><span className={'pill-status ' + (g.jenis === 'awal' ? 'pill-brand' : g.jenis === 'opname' ? 'pill-amber' : 'pill-green')}>{LABEL_GERAK[g.jenis]}</span></td>
                <td className="text-sm" style={{ whiteSpace: 'normal', maxWidth: 420 }}>
                  {g.keterangan && <div>{g.keterangan}</div>}
                  <div className="text-xs text-muted-foreground">
                    {g.baris.slice(0, 4).map((b) => `${nama(b.stok_id)} ${g.jenis === 'produksi' ? (b.qty! > 0 ? '+' : '') + formatQty(b.qty ?? 0) : formatQty(b.fisik ?? 0)}`).join(' · ')}
                    {g.baris.length > 4 && ` · +${g.baris.length - 4} barang`}
                  </div>
                </td>
                <td className="text-sm">{g.oleh}</td>
                {bisaUbah && <td style={{ textAlign: 'right' }}><Button variant="ghost" size="icon-sm" aria-label="Hapus dokumen" onClick={() => void hapus(g)}><Trash2 /></Button></td>}
              </tr>
            ))}
            {list?.length === 0 && <tr><td colSpan={5} className="text-center text-muted-foreground" style={{ padding: 20 }}>Belum ada dokumen stok.</td></tr>}
          </tbody>
        </table>
      </div>
    </div>
  )
}

/** Catat produksi: bahan dari stok dipakai sesuai resep, barang jadi bertambah. */
export function ProduksiDialog({ produkList, awal, stok, mulai, onClose, onDone }: {
  produkList: ProdukRow[]; awal?: ProdukRow; stok: StokRow[]; mulai: string; onClose: () => void; onDone: (pesan: string) => void
}) {
  const [pilihId, setPilihId] = useState(awal?.id ?? produkList[0]?.id ?? 0)
  const produk = produkList.find((p) => p.id === pilihId) ?? produkList[0]
  const [tanggal, setTanggal] = useState(hariIni)
  const [batch, setBatch] = useState(1)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const bahan = (produk.bahan ?? []).filter((b) => b.stok_id > 0).map((b) => {
    const s = stok.find((x) => x.id === b.stok_id)
    const pakai = b.takaran * batch
    return { nama: s?.nama ?? b.nama, satuan: s?.satuan ?? b.satuan, pakai, sisa: (s?.jumlah_mili ?? 0) - pakai }
  })
  const manual = (produk.bahan ?? []).filter((b) => b.stok_id === 0).length
  const hasil = produk.hasil_per_batch * batch

  async function kirim(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      await stokProduksi({ tanggal, produk_id: produk.id, batch })
      onDone(`Produksi ${produk.nama}: +${hasil.toLocaleString('id-ID')} ${produk.satuan} masuk stok.`)
    } catch (err) {
      setError((err as Error).message)
      setBusy(false)
    }
  }
  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="sm:max-w-xl">
        <form onSubmit={kirim} className="grid gap-3">
          <DialogHeader>
            <DialogTitle>Catat produksi</DialogTitle>
            <DialogDescription>Pilih produk dan jumlah batch. Bahan dari stok berkurang sesuai resep dan produk jadi bertambah. Nilai produk jadi = nilai bahan dari stok yang terpakai.</DialogDescription>
          </DialogHeader>
          <div className="grid gap-1.5"><Label htmlFor="pr-produk">Produk</Label>
            <select id="pr-produk" className="input-field" value={produk.id} onChange={(e) => setPilihId(Number(e.target.value))}>
              {produkList.map((p) => <option key={p.id} value={p.id}>{p.nama}{p.sku ? ` (${p.sku})` : ''}</option>)}
            </select></div>
          <div className="grid grid-cols-2 gap-3">
            <div className="grid gap-1.5"><Label htmlFor="pr-tgl">Tanggal</Label>
              <Input id="pr-tgl" type="date" min={mulai} required value={tanggal} onChange={(e) => setTanggal(e.target.value)} /></div>
            <div className="grid gap-1.5"><Label htmlFor="pr-batch">Jumlah batch</Label>
              <Input id="pr-batch" type="number" min={1} max={100000} required value={batch} onChange={(e) => setBatch(Math.max(1, Number(e.target.value) || 1))} /></div>
          </div>
          <p className="text-sm">Hasil: <strong>{hasil.toLocaleString('id-ID')} {produk.satuan}</strong> ({produk.hasil_per_batch.toLocaleString('id-ID')} per batch)</p>
          <table className="data-table tabular">
            <thead><tr><th>Bahan dipakai</th><th style={{ textAlign: 'right' }}>Jumlah</th><th style={{ textAlign: 'right' }}>Sisa stok</th></tr></thead>
            <tbody>
              {bahan.map((b, i) => (
                <tr key={i}>
                  <td>{b.nama}</td>
                  <td style={{ textAlign: 'right' }}>{formatQty(b.pakai)} {b.satuan}</td>
                  <td style={{ textAlign: 'right', color: b.sisa < 0 ? RED : undefined, fontWeight: b.sisa < 0 ? 700 : undefined }}>
                    {formatQty(b.sisa)} {b.satuan}{b.sisa < 0 && ' (minus)'}
                  </td>
                </tr>
              ))}
              {bahan.length === 0 && <tr><td colSpan={3} className="text-muted-foreground">Resep ini belum memakai bahan dari Stok & Gudang.</td></tr>}
            </tbody>
          </table>
          {bahan.some((b) => b.sisa < 0) && <div className="note-box warn">Ada bahan yang jadi minus. Produksi tetap bisa dicatat, tapi cek apakah pembelian bahan belum dicatat.</div>}
          {manual > 0 && <p className="text-xs text-muted-foreground">{manual} bahan diisi manual (di luar daftar stok) tidak mengurangi stok. Tenaga kerja & overhead tetap dicatat sebagai biaya saat dibayar.</p>}
          <Galat teks={error} />
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>Batal</Button>
            <Button type="submit" disabled={busy}>{busy ? 'Menyimpan…' : 'Catat produksi'}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

/** Laporan produksi per bulan: apa yang diproduksi, berapa, bahan terpakai, dan nilainya. */
export function LaporanProduksiPanel({ muatUlang, bisaUbah, bisaEkspor, onChanged }: { muatUlang: number; bisaUbah: boolean; bisaEkspor: boolean; onChanged: () => void }) {
  const [bulan, setBulan] = useState(() => hariIni().slice(0, 7))
  const dari = `${bulan}-01`
  const [y, m] = bulan.split('-').map(Number)
  const sampai = `${bulan}-${String(new Date(Date.UTC(y, m, 0)).getUTCDate()).padStart(2, '0')}`
  const [data, setData] = useState<{ kunci: string; l?: LaporanProduksi; error?: string } | null>(null)
  const kunci = `${dari}|${muatUlang}`
  useEffect(() => {
    let batal = false
    getLaporanProduksi(dari, sampai).then((l) => !batal && setData({ kunci, l }), (e: Error) => !batal && setData({ kunci, error: e.message }))
    return () => { batal = true }
  }, [dari, sampai, kunci])
  const geser = (n: number) => {
    const d = new Date(Date.UTC(y, m - 1 + n, 1))
    setBulan(d.toISOString().slice(0, 7))
  }
  async function hapus(id: number) {
    if (!window.confirm('Hapus catatan produksi ini? Bahan & produk jadi dihitung ulang.')) return
    try {
      await hapusGerakStok(id)
      onChanged()
    } catch (e) {
      setData((d) => d && { ...d, error: (e as Error).message })
    }
  }
  const l = data?.l
  return (
    <div className="card" style={{ padding: 20 }}>
      <div className="flex flex-wrap items-end gap-3" style={{ marginBottom: 12 }}>
        <div>
          <div className="chart-title">Laporan produksi</div>
          <div className="chart-subtitle">Setiap produksi yang dicatat: hasil, bahan terpakai, dan nilai produk jadi.</div>
        </div>
        <div className="flex items-end gap-2" style={{ marginLeft: 'auto' }}>
          <Button variant="outline" size="icon" aria-label="Bulan sebelumnya" onClick={() => geser(-1)}><ChevronLeft /></Button>
          <Input aria-label="Bulan" type="month" value={bulan} onChange={(e) => e.target.value && setBulan(e.target.value)} style={{ width: 170 }} />
          <Button variant="outline" size="icon" aria-label="Bulan berikutnya" onClick={() => geser(1)}><ChevronRight /></Button>
          {bisaEkspor && <Button variant="outline" onClick={() => void ekspor(laporanProduksiXlsxUrl(dari, sampai))}><FileSpreadsheet /> Export Excel</Button>}
        </div>
      </div>
      <Galat teks={data?.error ?? ''} />
      <div className="table-scroll" style={{ opacity: data?.kunci === kunci ? 1 : 0.6 }}>
        <table className="data-table tabular" style={{ minWidth: 880 }}>
          <thead><tr><th>Tanggal</th><th>Produk</th><th style={{ textAlign: 'right' }}>Batch</th><th style={{ textAlign: 'right' }}>Hasil</th><th>Bahan terpakai</th><th style={{ textAlign: 'right' }}>Nilai</th><th style={{ textAlign: 'right' }}>Per unit</th><th>Oleh</th>{bisaUbah && <th />}</tr></thead>
          <tbody>
            {l?.baris.map((b) => (
              <tr key={b.id}>
                <td>{formatTanggal(b.tanggal)}</td>
                <td className="font-semibold">{b.produk}</td>
                <td style={{ textAlign: 'right' }}>{b.batch.toLocaleString('id-ID')}</td>
                <td style={{ textAlign: 'right', color: GREEN, fontWeight: 600 }}>+{formatQty(b.hasil_qty)} {b.satuan}</td>
                <td className="text-xs" style={{ whiteSpace: 'normal', maxWidth: 300 }}>{b.bahan.map((x) => `${x.nama} ${formatQty(x.qty)} ${x.satuan}`).join(' · ') || '—'}</td>
                <td style={{ textAlign: 'right' }}>{formatRp(b.nilai)}</td>
                <td style={{ textAlign: 'right' }}>{formatRp(b.per_unit)}</td>
                <td className="text-sm">{b.oleh}</td>
                {bisaUbah && <td style={{ textAlign: 'right' }}><Button variant="ghost" size="icon-sm" aria-label="Hapus catatan produksi" onClick={() => void hapus(b.id)}><Trash2 /></Button></td>}
              </tr>
            ))}
            {l && l.baris.length === 0 && <tr><td colSpan={9} className="text-center text-muted-foreground" style={{ padding: 24 }}>Belum ada produksi di bulan ini.</td></tr>}
          </tbody>
          {l && l.baris.length > 0 && (
            <tfoot><tr className="font-semibold"><td colSpan={5}>Total nilai produksi</td><td style={{ textAlign: 'right' }}>{formatRp(l.total_nilai)}</td><td colSpan={bisaUbah ? 3 : 2} /></tr></tfoot>
          )}
        </table>
      </div>
    </div>
  )
}
