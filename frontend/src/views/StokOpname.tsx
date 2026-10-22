import { useMemo, useState, type FormEvent } from 'react'
import { CheckCircle2 } from 'lucide-react'
import { pilihMetodeStok, stokAwal, stokOpname, type MetodeNilaiStok, type StokRow } from '@/lib/api'
import { angka } from '@/lib/angka'
import { bacaDraft, hapusDraft, useSimpanDraft } from '@/lib/draft'
import { formatRp, formatTanggal } from '@/lib/format'
import { formatQty, parseQty, qtyKeTeks } from '@/lib/qty'
import { NAMA_METODE, useMetodeStok } from '@/lib/metodeStok'

// Stok opname di satu layar (bukan dialog): semua barang dalam satu tabel — stok sistem, hitungan
// fisik yang sudah terisi sama dengan sistem, dan selisihnya langsung terlihat. Opname pertama
// sekaligus saldo awal stok (titik nol kartu stok), lengkap dengan harga pokok & metode nilai stok.

const hariIni = () => new Date().toLocaleDateString('sv-SE', { timeZone: 'Asia/Jakarta' })
const RED = 'var(--text-danger)'
const GREEN = 'var(--accent-green)'
const DRAFT = 'stok-opname'

type Isian = { tanggal: string; fisik: Record<number, string>; harga: Record<number, number> }

export function StokOpnamePanel({ items, mulai, bisaUbah, onDone }: { items: StokRow[]; mulai: string; bisaUbah: boolean; onDone: (pesan: string) => void }) {
  const pertama = !mulai
  const [isi, setIsi] = useState<Isian>(() => {
    const d = bacaDraft<Isian>(DRAFT)
    return {
      tanggal: d?.tanggal ?? hariIni(),
      fisik: { ...Object.fromEntries(items.map((it) => [it.id, qtyKeTeks(it.jumlah_mili)])), ...d?.fisik },
      harga: { ...Object.fromEntries(items.map((it) => [it.id, it.harga_pokok])), ...d?.harga },
    }
  })
  useSimpanDraft(bisaUbah ? DRAFT : null, isi)
  const [cari, setCari] = useState('')
  const [hanyaSelisih, setHanyaSelisih] = useState(false)
  const [ket, setKet] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const metode = useMetodeStok()
  const [pilih, setPilih] = useState<MetodeNilaiStok | ''>('')

  const baris = useMemo(() => items.map((it) => {
    const teks = isi.fisik[it.id] ?? ''
    const fisik = teks.trim() === '' ? it.jumlah_mili : parseQty(teks)
    const harga = isi.harga[it.id] ?? it.harga_pokok
    return { it, teks, fisik, harga, selisih: fisik - it.jumlah_mili }
  }), [items, isi])
  const berubah = baris.filter((b) => b.selisih !== 0)
  const nilaiSelisih = berubah.reduce((n, b) => n + Math.round((b.selisih * b.harga) / 1000), 0)
  const totalAwal = baris.reduce((n, b) => n + Math.round((b.fisik * b.harga) / 1000), 0)
  const q = cari.trim().toLowerCase()
  const tampil = baris.filter((b) => (!hanyaSelisih || b.selisih !== 0) && (!q || (b.it.nama + ' ' + b.it.sku).toLowerCase().includes(q)))

  async function simpan(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      if (pertama) {
        if (metode && !metode.terkunci && pilih && pilih !== metode.metode) {
          await pilihMetodeStok(pilih).catch((err: Error) => {
            throw new Error(/akses|forbidden|izin/i.test(err.message) ? 'Hanya Owner yang bisa memilih metode rata-rata; biarkan FIFO atau minta Owner memilihnya di Profil Usaha.' : err.message)
          })
        }
        await stokAwal({ tanggal: isi.tanggal, keterangan: ket, baris: baris.map((b) => ({ stok_id: b.it.id, fisik: b.fisik, harga_unit: b.harga })) })
        hapusDraft(DRAFT)
        onDone(`Saldo awal stok per ${formatTanggal(isi.tanggal)} tersimpan. Mulai sekarang stok bergerak otomatis dari beli, jual, dan produksi.`)
      } else {
        await stokOpname({ tanggal: isi.tanggal, keterangan: ket, baris: berubah.map((b) => ({ stok_id: b.it.id, fisik: b.fisik })) })
        hapusDraft(DRAFT)
        onDone(`Stok opname ${formatTanggal(isi.tanggal)} tersimpan: ${berubah.length} barang disesuaikan.`)
      }
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  if (items.length === 0) {
    return <div className="card" style={{ padding: 20 }}>Belum ada barang. Tambah barang dulu di tab <strong>Daftar barang</strong>, lalu hitung fisiknya di sini.</div>
  }

  return (
    <form onSubmit={simpan} className="grid gap-3">
      <div className="card" style={{ padding: 20 }}>
        <div className="chart-title">{pertama ? 'Stok opname pertama / saldo awal stok' : 'Stok opname'}</div>
        <div className="chart-subtitle">
          {pertama
            ? 'Hitung fisik semua barang di gudang lalu isi jumlah & harga pokoknya. Hasilnya jadi saldo awal stok; sesudah itu stok bergerak otomatis dari beli, jual, dan produksi.'
            : 'Hitungan fisik sudah terisi sama dengan stok sistem. Ubah hanya barang yang jumlahnya berbeda; selisihnya dicatat sebagai penyesuaian (hilang, rusak, atau salah catat).'}
        </div>
        <div className="flex flex-wrap items-end gap-3" style={{ marginTop: 12 }}>
          <label className="grid gap-1">
            <span className="kpi-label">Tanggal hitung</span>
            <input className="input-field" type="date" required min={mulai || undefined} value={isi.tanggal} onChange={(e) => setIsi({ ...isi, tanggal: e.target.value })} disabled={!bisaUbah} />
          </label>
          <label className="grid gap-1" style={{ flex: '1 1 200px' }}>
            <span className="kpi-label">Keterangan</span>
            <input className="input-field" placeholder={pertama ? 'mis. Hitung fisik awal' : 'mis. Opname akhir bulan'} value={ket} onChange={(e) => setKet(e.target.value)} disabled={!bisaUbah} />
          </label>
          <label className="grid gap-1" style={{ flex: '1 1 200px' }}>
            <span className="kpi-label">Cari barang</span>
            <input className="input-field" placeholder="Nama atau SKU" value={cari} onChange={(e) => setCari(e.target.value)} />
          </label>
          {!pertama && (
            <label className="flex items-center gap-2 text-sm" style={{ paddingBottom: 8 }}>
              <input type="checkbox" checked={hanyaSelisih} onChange={(e) => setHanyaSelisih(e.target.checked)} /> Hanya yang selisih
            </label>
          )}
        </div>
        {pertama && metode && (metode.terkunci ? (
          <p className="text-sm" style={{ marginTop: 10 }}>Nilai stok dihitung dengan <strong>{NAMA_METODE[metode.metode]}</strong> (sudah dikunci).</p>
        ) : (
          <label className="grid gap-1" style={{ marginTop: 10, maxWidth: 460 }}>
            <span className="kpi-label">Cara menghitung nilai stok &amp; modal barang terjual</span>
            <select className="input-field" value={pilih || metode.metode} onChange={(e) => setPilih(e.target.value as MetodeNilaiStok)} disabled={!bisaUbah}>
              {metode.pilihan.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
            </select>
            <span className="text-xs text-muted-foreground">Dikunci begitu saldo awal stok disimpan; perubahan metode memerlukan persetujuan DJP.</span>
          </label>
        ))}
      </div>

      <div className="card" style={{ padding: 0 }}>
        <div className="table-scroll" style={{ maxHeight: 560 }}>
          <table className="data-table tabular" style={{ minWidth: pertama ? 760 : 680 }}>
            <thead>
              <tr>
                <th>Produk</th>
                <th style={{ textAlign: 'right' }}>Stok sistem</th>
                <th style={{ textAlign: 'right', width: 160 }}>Hitungan fisik</th>
                {pertama ? <><th style={{ textAlign: 'right', width: 160 }}>Harga pokok / satuan</th><th style={{ textAlign: 'right' }}>Nilai</th></>
                  : <><th style={{ textAlign: 'right' }}>Selisih</th><th style={{ textAlign: 'right' }}>Nilai selisih</th></>}
              </tr>
            </thead>
            <tbody>
              {tampil.length === 0 && <tr><td colSpan={5} className="text-muted-foreground">{hanyaSelisih ? 'Tidak ada barang yang selisih.' : 'Tidak ada barang yang cocok.'}</td></tr>}
              {tampil.map(({ it, teks, fisik, harga, selisih }) => (
                <tr key={it.id} style={selisih !== 0 && !pertama ? { background: 'var(--accent-orange-soft)' } : undefined}>
                  <td><div className="font-semibold">{it.nama}</div><div className="text-xs text-muted-foreground">{it.sku || it.kategori}</div></td>
                  <td style={{ textAlign: 'right', color: it.minus ? RED : undefined }}>{formatQty(it.jumlah_mili)} {it.satuan}</td>
                  <td style={{ textAlign: 'right' }}>
                    <span className="inline-flex items-center gap-1.5">
                      <input className="input-field tabular" aria-label={`Hitungan fisik ${it.nama}`} inputMode="decimal" style={{ width: 110, textAlign: 'right' }} disabled={!bisaUbah}
                        value={teks} onChange={(e) => setIsi({ ...isi, fisik: { ...isi.fisik, [it.id]: e.target.value.replace(/[^\d.,]/g, '') } })} />
                      <span className="text-xs text-muted-foreground" style={{ minWidth: 28 }}>{it.satuan}</span>
                    </span>
                  </td>
                  {pertama ? <>
                    <td style={{ textAlign: 'right' }}>
                      <input className="input-field tabular" aria-label={`Harga pokok ${it.nama}`} style={{ width: 140, textAlign: 'right', marginLeft: 'auto' }} disabled={!bisaUbah}
                        {...angka(harga, (n) => setIsi({ ...isi, harga: { ...isi.harga, [it.id]: n } }))} />
                    </td>
                    <td style={{ textAlign: 'right' }}>{formatRp(Math.round((fisik * harga) / 1000))}</td>
                  </> : <>
                    <td style={{ textAlign: 'right', fontWeight: selisih ? 700 : undefined, color: selisih === 0 ? 'var(--text-muted)' : selisih < 0 ? RED : GREEN }}>
                      {selisih === 0 ? '0' : `${selisih > 0 ? '+' : ''}${formatQty(selisih)}`}
                    </td>
                    <td style={{ textAlign: 'right', color: selisih === 0 ? 'var(--text-muted)' : selisih < 0 ? RED : GREEN }}>
                      {selisih === 0 ? '—' : formatRp(Math.round((selisih * harga) / 1000))}
                    </td>
                  </>}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {error && <div className="pill-status pill-red" role="alert" style={{ whiteSpace: 'normal', alignSelf: 'flex-start' }}>{error}</div>}
      {bisaUbah && (
        <div className="flex flex-wrap items-center gap-3">
          <button className="btn-cta accent" style={{ padding: '10px 22px', fontSize: 14 }} type="submit" disabled={busy || (!pertama && berubah.length === 0)}>
            {busy ? 'Menyimpan…' : pertama ? 'Simpan saldo awal stok' : 'Simpan opname'}
          </button>
          <span className="text-sm">
            {pertama
              ? <>Nilai persediaan awal: <strong>{formatRp(totalAwal)}</strong></>
              : berubah.length === 0
                ? <span className="inline-flex items-center gap-1.5" style={{ color: GREEN }}><CheckCircle2 size={15} aria-hidden /> Semua hitungan fisik sama dengan stok sistem.</span>
                : <><strong>{berubah.length}</strong> barang selisih · nilai selisih <strong style={{ color: nilaiSelisih < 0 ? RED : GREEN }}>{formatRp(nilaiSelisih)}</strong></>}
          </span>
        </div>
      )}
    </form>
  )
}
