import { useEffect, useMemo, useState, type FormEvent } from 'react'
import { ChevronLeft, ChevronRight, FileSpreadsheet, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import {
  createTransaction, getGerakStok, getKartuBarang, getLaporanProduksi, getMutasiStok, getPilihanBarang, hapusGerakStok, laporanProduksiXlsxUrl, mutasiStokXlsxUrl,
  type BarisKartu, type GerakStok, type JenisKartu, type KartuBarang, type LaporanMutasi, type LaporanProduksi, type PilihanBarang, type StokRow,
} from '@/lib/api'
import { PilihRincianBarang } from '@/components/PilihRincianBarang'
import { barisTerpilih, bodyRincian, rincianSah, type NilaiRincian } from '@/lib/rincian'
import { NAMA_METODE, NAMA_PENCATATAN, useMetodeStok } from '@/lib/metodeStok'
import { angka } from '@/lib/angka'
import { formatRp, formatTanggal } from '@/lib/format'
import { ekspor } from '@/lib/garansi'
import { SKALA, formatQty } from '@/lib/qty'

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



/** Beli bahan/barang: transaksi "Beli Bahan / Stok" dengan satu atau beberapa barang → stok & HPP
 *  (FIFO / rata-rata). Pembelian beberapa barang mengisi nilai tiap barang. */
export function BeliBahanDialog({ items, onClose, onDone }: { items: StokRow[]; onClose: () => void; onDone: () => void }) {
  const [tanggal, setTanggal] = useState(hariIni)
  const [rincian, setRincian] = useState<NilaiRincian>(() => ({ rincian: [{ stok_id: items[0]?.id ?? 0, qty_stok: 0 }], bukan_barang: false }))
  const [barang, setBarang] = useState<PilihanBarang[]>(() => items.map((x) => ({ id: x.id, sku: x.sku, nama: x.nama, kategori: x.kategori, satuan: x.satuan, jumlah_mili: x.jumlah * SKALA, harga_jual: x.harga_jual })))
  const [total, setTotal] = useState(0)
  const [pihak, setPihak] = useState('')
  const [lunas, setLunas] = useState(true)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  // jumlah menurut kartu stok (bukan daftar barang) bila kartu sudah dimulai
  useEffect(() => { getPilihanBarang().then((p) => setBarang(p.barang), () => {}) }, [])
  const terpilih = barisTerpilih(rincian.rincian)
  const satu = terpilih.length === 1 ? barang.find((x) => x.id === terpilih[0].stok_id) : undefined
  const q = terpilih.length === 1 ? terpilih[0].qty_stok : 0
  const nilaiSatu = terpilih.length === 1 ? terpilih[0].nilai || total : 0

  async function kirim(e: FormEvent) {
    e.preventDefault()
    const jenis = 'Beli Bahan / Stok'
    const salah = rincianSah(rincian, true, { jenis, nominal: total })
    if (salah) return setError(salah)
    const nama = terpilih.map((r) => barang.find((x) => x.id === r.stok_id)?.nama).filter(Boolean).join(', ')
    setBusy(true)
    setError('')
    try {
      await createTransaction({
        tanggal, jenis, nominal: total, keterangan: `Beli ${nama}`, item: nama, qty: 0, channel: '',
        pihak, lunas, tanggal_lunas: '', ...bodyRincian(rincian, jenis),
      })
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
            <DialogTitle>Beli bahan / barang</DialogTitle>
            <DialogDescription>Menambah stok dan memperbarui nilai stok. Tercatat juga sebagai transaksi pengeluaran (atau utang bila belum lunas).</DialogDescription>
          </DialogHeader>
          <div className="grid gap-3 sm:grid-cols-2">
            <div className="grid gap-1.5"><Label htmlFor="bb-total">Total harga (Rp)</Label>
              <Input id="bb-total" required className="tabular" {...angka(total, setTotal)} /></div>
            <div className="grid gap-1.5"><Label htmlFor="bb-tgl">Tanggal</Label>
              <Input id="bb-tgl" type="date" required value={tanggal} onChange={(e) => setTanggal(e.target.value)} /></div>
            <PilihRincianBarang id="bb-barang" jenis="Beli Bahan / Stok" nominal={total} value={rincian} onChange={setRincian}
              barang={barang} wajib bolehJasa={false} onBarangBaru={(b) => setBarang((xs) => [...xs, b])} />
            <div className="grid gap-1.5 sm:col-span-2"><Label htmlFor="bb-pihak">Supplier</Label>
              <Input id="bb-pihak" placeholder="Opsional" value={pihak} onChange={(e) => setPihak(e.target.value)} /></div>
          </div>
          {q > 0 && nilaiSatu > 0 && satu && <p className="text-sm text-muted-foreground">Harga per {satu.satuan || 'satuan'}: <strong>{formatRp(Math.round((nilaiSatu * SKALA) / q))}</strong></p>}
          <label className="flex items-center gap-3 text-sm"><Switch checked={lunas} onCheckedChange={setLunas} /> {lunas ? 'Sudah dibayar' : 'Belum dibayar (masuk utang)'}</label>
          <Galat teks={error} />
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>Batal</Button>
            <Button type="submit" disabled={busy || terpilih.length === 0 || total <= 0}>{busy ? 'Menyimpan…' : 'Simpan pembelian'}</Button>
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
  const metode = useMetodeStok()
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
          <DialogDescription>Setiap barang masuk & keluar beserta saldonya. Nilai memakai metode {metode ? NAMA_METODE[metode.metode] : 'nilai stok usaha'}.{metode?.pencatatan && <> Pencatatan {NAMA_PENCATATAN[metode.pencatatan]}.</>}</DialogDescription>
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
  const metode = useMetodeStok()
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
        nilai memakai metode {metode ? NAMA_METODE[metode.metode] : 'nilai stok usaha'}. Jumlah antarbarang tidak dijumlahkan karena satuannya berbeda.
      </p>
      {metode?.pencatatan && <p className="text-xs text-muted-foreground"><strong>Cara pencatatan stok ({metode.pencatatan}).</strong> {NAMA_PENCATATAN[metode.pencatatan]}. Bisa diubah di Profil Usaha.</p>}
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
                <td style={{ textAlign: 'right', color: GREEN, fontWeight: 600 }}>
                  +{formatQty(b.hasil_qty)} {b.satuan}
                  {b.hasil_standar > 0 && b.hasil_standar !== b.hasil_qty && <div className="text-xs" style={{ color: b.hasil_qty < b.hasil_standar ? RED : GREEN, fontWeight: 400 }}>resep {formatQty(b.hasil_standar)}</div>}
                </td>
                <td className="text-xs" style={{ whiteSpace: 'normal', maxWidth: 320 }}>
                  {b.bahan.map((x, i) => (
                    <span key={i}>{i > 0 && ' · '}{x.nama} {formatQty(x.qty)} {x.satuan}
                      {x.qty !== x.standar && <span style={{ color: x.qty > x.standar ? RED : GREEN, fontWeight: 600 }}> ({x.standar ? `resep ${formatQty(x.standar)}` : 'di luar resep'})</span>}
                    </span>
                  ))}
                  {b.bahan.length === 0 && '—'}
                  {b.keterangan && !b.keterangan.startsWith('Produksi ') && <div className="text-muted-foreground">Catatan: {b.keterangan}</div>}
                </td>
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
