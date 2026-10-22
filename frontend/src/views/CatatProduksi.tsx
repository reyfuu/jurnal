import { useEffect, useMemo, useState } from 'react'
import { ArrowLeft, Plus, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { catatProduksi, getOpsiCatat, type AkunCatat, type ProdukRow, type StokRow } from '@/lib/api'
import { angka } from '@/lib/angka'
import { formatRp } from '@/lib/format'
import { formatQty, parseQty, qtyKeTeks } from '@/lib/qty'

// Halaman Catat produksi: satu tanggal, beberapa produk sekaligus. Tiap produk: jumlah jadi,
// pemakaian bahan (bisa diisi dari resep lalu diubah), dan biaya lain (tenaga harian, gas, ongkir)
// yang dibayar dari Kas/Bank dan ikut menjadi nilai produk. HPP/pcs di sini perkiraan dari harga
// pokok di daftar barang; nilai pastinya dihitung kartu stok (FIFO / rata-rata).

type BarisBahan = { _k: number; stokId: number; qty: string }
type BarisBiaya = { _k: number; ket: string; nominal: number }
type Blok = { _k: number; produkId: number; hasil: string; bahan: BarisBahan[]; biaya: BarisBiaya[] }

let urut = 0
const k = () => ++urut
const hariIni = () => new Date().toLocaleDateString('sv-SE', { timeZone: 'Asia/Jakarta' })
const RED = 'var(--text-danger)'

/** Bahan dari resep untuk jumlah jadi tertentu (per seribu satuan). */
function dariResep(p: ProdukRow | undefined, hasilMili: number): BarisBahan[] {
  if (!p) return []
  const kali = p.hasil_per_batch > 0 ? hasilMili / (p.hasil_per_batch * 1000) : 1
  return (p.bahan ?? []).filter((b) => b.stok_id > 0).map((b) => ({ _k: k(), stokId: b.stok_id, qty: qtyKeTeks(Math.round(b.takaran * kali)) }))
}

function blokBaru(p: ProdukRow | undefined): Blok {
  const hasil = (p?.hasil_per_batch ?? 1) * 1000
  return { _k: k(), produkId: p?.id ?? 0, hasil: qtyKeTeks(hasil), bahan: dariResep(p, hasil), biaya: [{ _k: k(), ket: '', nominal: 0 }] }
}

export function CatatProduksi({ produkList, stok, stokMulai, awal, onKembali, onTersimpan }: {
  produkList: ProdukRow[]; stok: StokRow[]; stokMulai: string; awal?: ProdukRow
  onKembali: () => void; onTersimpan: (pesan: string) => void
}) {
  const [tanggal, setTanggal] = useState(hariIni)
  const [rekening, setRekening] = useState<AkunCatat[]>([])
  const [bayarDari, setBayarDari] = useState('')
  const [blok, setBlok] = useState<Blok[]>(() => [blokBaru(awal ?? produkList[0])])
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    getOpsiCatat().then((o) => {
      const r = o.akun.filter((a) => a.lawan === 'rekening')
      setRekening(r)
      setBayarDari((x) => x || o.bawaan.bank || o.bawaan.kas || r[0]?.kode || '')
    }, () => {})
  }, [])

  const stokPeta = useMemo(() => new Map(stok.map((s) => [s.id, s])), [stok])
  const produkPeta = useMemo(() => new Map(produkList.map((p) => [p.id, p])), [produkList])
  // Bahan yang bisa dipakai: semua barang kecuali produk jadi dari resep (dicocokkan SKU / nama).
  const namaProduk = new Set(produkList.map((p) => (p.sku || p.nama).toLowerCase()))
  const pilihanBahan = stok.filter((s) => !namaProduk.has((s.sku || s.nama).toLowerCase()))

  const ubahBlok = (bk: number, f: (b: Blok) => Blok) => setBlok((bs) => bs.map((b) => (b._k === bk ? f(b) : b)))

  function hitung(b: Blok) {
    const bahan = b.bahan.reduce((t, x) => t + Math.round((parseQty(x.qty) * (stokPeta.get(x.stokId)?.harga_pokok ?? 0)) / 1000), 0)
    const lain = b.biaya.reduce((t, x) => t + (x.nominal || 0), 0)
    const hasil = parseQty(b.hasil)
    return { bahan, lain, total: bahan + lain, perUnit: hasil > 0 ? Math.round(((bahan + lain) * 1000) / hasil) : 0 }
  }

  async function simpan() {
    setBusy(true)
    setError('')
    try {
      const res = await catatProduksi({
        tanggal, bayar_dari: bayarDari,
        produk: blok.map((b) => ({
          produk_id: b.produkId, hasil: parseQty(b.hasil),
          bahan: b.bahan.filter((x) => x.stokId && parseQty(x.qty) > 0).map((x) => ({ stok_id: x.stokId, qty: parseQty(x.qty) })),
          biaya: b.biaya.filter((x) => x.nominal > 0).map((x) => ({ keterangan: x.ket, nominal: x.nominal })),
        })),
      })
      onTersimpan(`Produksi ${tanggal} tersimpan: ${res.gerak.length} produk masuk stok${res.transaksi ? `, ${res.transaksi} biaya lain dicatat dari kas/bank` : ''}${res.stok_awal ? '. Kartu stok dimulai otomatis dari daftar barang.' : '.'}`)
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  const bisaSimpan = !busy && blok.length > 0 && blok.every((b) => b.produkId && parseQty(b.hasil) > 0)

  return (
    <div className="grid gap-4">
      <button type="button" className="inline-flex items-center gap-1.5 text-sm text-muted-foreground" style={{ justifySelf: 'start' }} onClick={onKembali}>
        <ArrowLeft size={15} aria-hidden="true" /> Kembali ke Produksi &amp; Supplier
      </button>

      {!stokMulai && (
        <div className="note-box" style={{ margin: 0 }}>
          <strong>Bisa langsung dicatat.</strong> Kartu stok belum dimulai, jadi saat produksi ini disimpan, jumlah &amp; harga pokok di daftar barang hari ini
          otomatis dipakai sebagai stok awal. Selisihnya bisa dikoreksi kapan saja lewat Stok opname.
        </div>
      )}

      <div className="card form-grid" style={{ padding: 18, gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))' }}>
        <label><span className="kpi-label">Tanggal produksi</span>
          <input className="input-field" type="date" required min={stokMulai || undefined} value={tanggal} onChange={(e) => setTanggal(e.target.value)} /></label>
        <label><span className="kpi-label">Biaya lain dibayar dari</span>
          <select className="input-field" value={bayarDari} onChange={(e) => setBayarDari(e.target.value)}>
            {rekening.map((a) => <option key={a.kode} value={a.kode}>{a.kode} {a.nama}</option>)}
          </select></label>
        <p className="text-xs text-muted-foreground" style={{ alignSelf: 'end', margin: 0 }}>Tanggal berlaku untuk semua produk di bawah.</p>
      </div>

      {blok.map((b) => {
        const p = produkPeta.get(b.produkId)
        const h = hitung(b)
        return (
          <div key={b._k} className="card grid gap-3" style={{ padding: 18 }}>
            <div className="flex flex-wrap items-end gap-3">
              <label className="grid gap-1" style={{ flex: '1 1 240px' }}><span className="kpi-label">Produk jadi</span>
                <select className="input-field" value={b.produkId} onChange={(e) => {
                  const np = produkPeta.get(Number(e.target.value))
                  ubahBlok(b._k, (x) => ({ ...x, produkId: Number(e.target.value), bahan: dariResep(np, parseQty(x.hasil)) }))
                }}>
                  {produkList.map((x) => <option key={x.id} value={x.id}>{x.nama}</option>)}
                </select></label>
              <label className="grid gap-1" style={{ width: 150 }}><span className="kpi-label">Jumlah jadi ({p?.satuan || 'unit'})</span>
                <input className="input-field tabular" inputMode="decimal" style={{ textAlign: 'right' }} value={b.hasil}
                  onChange={(e) => ubahBlok(b._k, (x) => ({ ...x, hasil: e.target.value.replace(/[^\d.,]/g, '') }))} /></label>
              <Button type="button" variant="outline" size="sm" onClick={() => ubahBlok(b._k, (x) => ({ ...x, bahan: dariResep(p, parseQty(x.hasil)) }))}>
                Isi dari resep
              </Button>
              {blok.length > 1 && <Button type="button" variant="ghost" size="sm" className="text-destructive" onClick={() => setBlok((bs) => bs.filter((x) => x._k !== b._k))}>Hapus produk</Button>}
            </div>

            <div className="flex items-center justify-between"><span className="kpi-label">Pemakaian bahan</span>
              <Button type="button" size="sm" onClick={() => ubahBlok(b._k, (x) => ({ ...x, bahan: [...x.bahan, { _k: k(), stokId: pilihanBahan[0]?.id ?? 0, qty: '' }] }))}><Plus aria-hidden="true" /> Bahan</Button></div>
            <div className="table-scroll">
              <table className="data-table tabular" style={{ minWidth: 680 }}>
                <thead><tr><th style={{ width: '34%' }}>Bahan</th><th style={{ textAlign: 'right' }}>Stok</th><th style={{ textAlign: 'right' }}>Harga / unit</th><th style={{ textAlign: 'right', width: 140 }}>Qty pakai</th><th style={{ textAlign: 'right' }}>Subtotal</th><th style={{ width: 44 }} /></tr></thead>
                <tbody>
                  {b.bahan.length === 0 && <tr><td colSpan={6} className="text-muted-foreground">Belum ada bahan. Klik "Isi dari resep" atau "+ Bahan".</td></tr>}
                  {b.bahan.map((x) => {
                    const s = stokPeta.get(x.stokId)
                    const pakai = parseQty(x.qty)
                    const kurang = s && pakai > s.jumlah_mili
                    return (
                      <tr key={x._k}>
                        <td><select className="input-field" aria-label="Bahan" value={x.stokId} onChange={(e) => ubahBlok(b._k, (y) => ({ ...y, bahan: y.bahan.map((z) => (z._k === x._k ? { ...z, stokId: Number(e.target.value) } : z)) }))}>
                          {pilihanBahan.map((sb) => <option key={sb.id} value={sb.id}>{sb.nama}</option>)}
                        </select></td>
                        <td style={{ textAlign: 'right', color: kurang ? RED : undefined }} title={kurang ? 'Pemakaian melebihi stok tercatat' : undefined}>{s ? `${formatQty(s.jumlah_mili)} ${s.satuan}` : '—'}</td>
                        <td style={{ textAlign: 'right' }}>{formatRp(s?.harga_pokok ?? 0)}</td>
                        <td><input className="input-field tabular" inputMode="decimal" style={{ textAlign: 'right' }} aria-label="Qty pakai" value={x.qty}
                          onChange={(e) => ubahBlok(b._k, (y) => ({ ...y, bahan: y.bahan.map((z) => (z._k === x._k ? { ...z, qty: e.target.value.replace(/[^\d.,]/g, '') } : z)) }))} /></td>
                        <td style={{ textAlign: 'right' }}>{formatRp(Math.round((pakai * (s?.harga_pokok ?? 0)) / 1000))}</td>
                        <td><Button type="button" variant="ghost" size="icon-sm" aria-label="Hapus bahan" onClick={() => ubahBlok(b._k, (y) => ({ ...y, bahan: y.bahan.filter((z) => z._k !== x._k) }))}><Trash2 /></Button></td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>

            <div className="flex items-center justify-between"><span className="kpi-label">Biaya lain <span style={{ textTransform: 'none', letterSpacing: 0, fontWeight: 400 }}>(tenaga harian, gas, ongkir; opsional)</span></span>
              <Button type="button" size="sm" onClick={() => ubahBlok(b._k, (x) => ({ ...x, biaya: [...x.biaya, { _k: k(), ket: '', nominal: 0 }] }))}><Plus aria-hidden="true" /> Biaya</Button></div>
            <div className="table-scroll">
              <table className="data-table tabular" style={{ minWidth: 480 }}>
                <thead><tr><th>Keterangan</th><th style={{ textAlign: 'right', width: 180 }}>Nominal</th><th style={{ width: 44 }} /></tr></thead>
                <tbody>
                  {b.biaya.map((x) => (
                    <tr key={x._k}>
                      <td><input className="input-field" aria-label="Keterangan biaya" placeholder="mis. Tenaga harian 2 orang" value={x.ket}
                        onChange={(e) => ubahBlok(b._k, (y) => ({ ...y, biaya: y.biaya.map((z) => (z._k === x._k ? { ...z, ket: e.target.value } : z)) }))} /></td>
                      <td><input className="input-field tabular" style={{ textAlign: 'right' }} aria-label="Nominal biaya"
                        {...angka(x.nominal, (n) => ubahBlok(b._k, (y) => ({ ...y, biaya: y.biaya.map((z) => (z._k === x._k ? { ...z, nominal: n } : z)) })))} /></td>
                      <td><Button type="button" variant="ghost" size="icon-sm" aria-label="Hapus biaya" onClick={() => ubahBlok(b._k, (y) => ({ ...y, biaya: y.biaya.filter((z) => z._k !== x._k) }))}><Trash2 /></Button></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            <div className="grid justify-end">
              <div className="grid gap-1.5 tabular" style={{ minWidth: 'min(320px, 100%)' }}>
                <div className="flex justify-between gap-6"><span>Total biaya bahan</span><strong>{formatRp(h.bahan)}</strong></div>
                <div className="flex justify-between gap-6"><span>Total biaya lain</span><strong>{formatRp(h.lain)}</strong></div>
                <div className="flex justify-between gap-6" style={{ borderTop: '1px solid var(--border-shell)', paddingTop: 6 }}><span>Sub total</span><strong>{formatRp(h.total)}</strong></div>
                <div className="flex justify-between gap-6 rounded-lg px-3 py-2" style={{ background: 'color-mix(in srgb, var(--accent-green) 12%, transparent)', border: '1px solid var(--accent-green)', color: 'var(--accent-green)', fontWeight: 700 }}>
                  <span>HPP / {p?.satuan || 'unit'} (perkiraan)</span><span>{h.perUnit ? formatRp(h.perUnit) : '—'}</span>
                </div>
              </div>
            </div>
          </div>
        )
      })}

      <button type="button" className="card text-sm font-semibold" style={{ padding: 14, borderStyle: 'dashed', cursor: 'pointer' }}
        onClick={() => setBlok((bs) => [...bs, blokBaru(produkList.find((p) => !bs.some((b) => b.produkId === p.id)) ?? produkList[0])])}>
        + Tambah produk
      </button>

      {error && <div className="pill-status pill-red" role="alert" style={{ whiteSpace: 'normal', alignSelf: 'flex-start' }}>{error}</div>}
      <div className="flex flex-wrap justify-end gap-2">
        <Button type="button" variant="outline" onClick={onKembali}>Batal</Button>
        <Button type="button" disabled={!bisaSimpan} onClick={() => void simpan()}>{busy ? 'Menyimpan…' : 'Simpan produksi'}</Button>
      </div>
    </div>
  )
}
