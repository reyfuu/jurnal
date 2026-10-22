import { useEffect, useMemo, useRef, useState } from 'react'
import { bacaDraft, hapusDraft, useSimpanDraft } from '@/lib/draft'
import { AlertTriangle, CheckCircle2, Info, Plus, Sparkles, Trash2 } from 'lucide-react'
import { PageHeader } from '@/components/PageHeader'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Aset } from '@/views/Aset'
import {
  getAssets, getCekSaldoAwal, getOpening, saranSaldo, saveOpening,
  type AssetSummary, type CekKecocokan, type KategoriSaldo, type OpeningBalance, type OpeningInfo, type SaldoAkun, type SaranSaldo, type View,
} from '@/lib/api'
import { formatRp, formatTanggal } from '@/lib/format'
import { angka } from '@/lib/angka'

const RED = 'var(--text-danger)'
const GREEN = 'var(--accent-green)'
const AMBER = 'var(--accent-amber)'
// Sama dengan ketSelisihSaldoAwal di server.
const DRAFT_SALDO = 'saldo-awal'
const KET_PAKSA = 'selisih saldo awal yang disimpan walau belum seimbang'

// Baris layar: _k kunci baris; _baru = ditambah pengguna (dapat saran kategori & kode);
// _katManual / _kodeAuto menjaga pilihan pengguna tidak tertimpa saran.
type Baris = SaldoAkun & { _k: number; _baru?: boolean; _katManual?: boolean; _kodeAuto?: boolean; _saran?: SaranSaldo }
let urut = 0
const jadiBaris = (b: SaldoAkun, extra: Partial<Baris> = {}): Baris => ({ ...b, ...extra, _k: ++urut })
const bersih = (b: SaldoAkun): SaldoAkun => ({ kode: b.kode, nama: b.nama, kategori: b.kategori, nominal: b.nominal, keterangan: b.keterangan })

// Saldo awal lama (tanpa rincian): pos-posnya dijadikan baris supaya bisa diedit per akun.
// Aset tetap tidak termasuk: barisnya selalu datang dari daftar aset. Kode ikut CoA usaha ini
// (akun pertama berkategori sama di template), mis. pinjaman manufaktur = 2131, bukan 2102.
function dariPos(o: OpeningBalance, nbv: number, template: SaldoAkun[]): SaldoAkun[] {
  const kode = (kat: string, cadangan: string) => template.find((a) => a.kategori === kat)?.kode || cadangan
  const baris: SaldoAkun[] = [
    { kode: kode('kas', '1101'), nama: 'Kas & Bank', kategori: 'kas', nominal: o.kas },
    { kode: kode('piutang', '1201'), nama: 'Piutang Usaha', kategori: 'piutang', nominal: o.piutang },
    { kode: kode('persediaan', '1301'), nama: 'Persediaan', kategori: 'persediaan', nominal: o.persediaan },
    { kode: kode('utang', '2101'), nama: 'Utang Usaha', kategori: 'utang', nominal: o.utang },
    { kode: kode('pinjaman', '2102'), nama: 'Utang Bank', kategori: 'pinjaman', nominal: o.pinjaman },
  ]
  const modal = o.kas + o.piutang + o.persediaan + nbv - o.utang - o.pinjaman
  return [...baris, { kode: kode('modal', '3101'), nama: 'Modal Pemilik', kategori: 'modal', nominal: modal }]
}

const bulanIni = () => new Date().toISOString().slice(0, 7)

/**
 * Menu Saldo Awal, berurutan: 1. aset tetap (daftar aset), 2. barang & persediaan, 3. neraca
 * pembukaan per akun — hanya bisa disimpan bila Harta = Kewajiban + Modal.
 */
export function SaldoAwal({ onChanged, navigate, bisaUbah }: { onChanged: () => void; navigate: (v: View, tab?: string) => void; bisaUbah: boolean }) {
  const [info, setInfo] = useState<OpeningInfo | null>(null)
  const [tanggal, setTanggal] = useState('')
  const [baris, setBaris] = useState<Baris[] | null>(null)
  const [asetEdit, setAsetEdit] = useState<SaldoAkun[]>([])
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [paksa, setPaksa] = useState(false)
  const [dipulihkan, setDipulihkan] = useState(false)
  const [cek, setCek] = useState<CekKecocokan[]>([])
  const [bukaAset, setBukaAset] = useState(false)
  const [asetData, setAsetData] = useState<AssetSummary | null>(null)
  const [versiAset, setVersiAset] = useState(0)
  const kartuAset = useRef<HTMLDivElement>(null)
  const timer = useRef<Record<number, number>>({})

  useEffect(() => {
    getOpening().then(setInfo, (e: Error) => setError(e.message))
  }, [versiAset])
  // Diperiksa ulang tiap saldo awal / daftar aset berubah, supaya selisih baru langsung terlihat.
  useEffect(() => { getCekSaldoAwal().then(setCek, () => setCek([])) }, [info])

  const editing = baris !== null
  // Nilai buku aset mengikuti tanggal mulai yang dipilih (aset dibeli sebelum tanggal itu).
  useEffect(() => {
    if (!editing || tanggal.length !== 10) return
    let hidup = true
    getOpening(tanggal).then((i) => { if (hidup) setAsetEdit(i.aset_tetap ?? []) }, () => {})
    return () => { hidup = false }
  }, [tanggal, editing, versiAset])

  const o = info?.opening
  const periodAset = (tanggal || o?.tanggal || '').slice(0, 7) || bulanIni()
  useEffect(() => {
    if (!bukaAset) return
    let hidup = true
    getAssets(periodAset).then((d) => { if (hidup) setAsetData(d) }, (e: Error) => setError(e.message))
    return () => { hidup = false }
  }, [bukaAset, periodAset, versiAset])

  const kategori: KategoriSaldo[] = info?.kategori ?? []
  const sisi = (id: string) => kategori.find((k) => k.id === id)?.sisi
  const asetRows: SaldoAkun[] = ((editing ? asetEdit : info?.aset_tetap) ?? [])
  const nbv = asetRows.reduce((t, r) => t + r.nominal, 0)
  // Baris aset tetap disisipkan sesudah harta lain, sebelum kewajiban.
  const biasa = baris ?? []
  const pos = biasa.findIndex((b) => b.kategori && sisi(b.kategori) !== 'harta')
  const tampil: (Baris | (SaldoAkun & { _k: -1 }))[] = [
    ...biasa.slice(0, pos < 0 ? biasa.length : pos),
    ...asetRows.map((r) => ({ ...r, _k: -1 as const })),
    ...(pos < 0 ? [] : biasa.slice(pos)),
  ]
  const jumlah = (s: string) => tampil.filter((b) => sisi(b.kategori) === s).reduce((t, b) => t + (b.nominal || 0), 0)
  const harta = jumlah('harta'), kewajiban = jumlah('kewajiban'), modal = jumlah('modal')
  const selisih = harta - kewajiban - modal
  const tanpaKategori = biasa.filter((b) => (b.nama.trim() || b.nominal) && !b.kategori)
  const totalPersediaan = biasa.filter((b) => b.kategori === 'persediaan').reduce((t, b) => t + (b.nominal || 0), 0)
  const nilaiBarang = info?.persediaan ?? 0

  function mulaiUbah() {
    if (!info) return
    setTanggal(o?.tanggal ?? info.transaksi_pertama ?? '')
    setAsetEdit(info.aset_tetap ?? [])
    const awal = o?.rincian?.length ? o.rincian : o ? dariPos(o, info.nilai_buku_aset, info.akun_template) : info.akun_template.map((a) => ({ ...a, nominal: 0 }))
    // Baris penyeimbang hasil simpan paksa tidak diedit: dihitung ulang server setiap simpan.
    const rows = awal.filter((r) => r.kategori !== 'aset_tetap' && r.keterangan !== KET_PAKSA).map((r) => jadiBaris({ ...r }))
    // Saldo awal baru: persediaan langsung diisi dari daftar barang, bukan diketik ulang.
    const p = rows.find((r) => r.kategori === 'persediaan')
    if (!o && p && nilaiBarang > 0) {
      p.nominal = nilaiBarang
      p.keterangan = info.label_persediaan
    }
    // Isian yang belum tersimpan (error, muat ulang) dipulihkan dari draft.
    const draft = bacaDraft<{ tanggal: string; baris: SaldoAkun[] }>(DRAFT_SALDO)
    if (draft?.baris?.length) {
      setTanggal(draft.tanggal)
      setBaris(draft.baris.map((r) => jadiBaris(r)))
      setDipulihkan(true)
    } else {
      setBaris(rows)
      setDipulihkan(false)
    }
    setError('')
  }

  useSimpanDraft(baris ? DRAFT_SALDO : null, useMemo(() => (baris ? { tanggal, baris: baris.map(bersih) } : null), [tanggal, baris]))

  const ubah = (k: number, patch: Partial<Baris>) => setBaris((bs) => bs && bs.map((b) => (b._k === k ? { ...b, ...patch } : b)))

  // Saran kategori & kode dari server (akun CoA yang cocok → AI → kata kunci). Ketikan dijeda
  // sebentar supaya tidak meminta saran tiap huruf.
  function mintaSaran(b: Baris, nama: string, pilihKategori?: string) {
    window.clearTimeout(timer.current[b._k])
    const dipakai = biasa.filter((x) => x._k !== b._k && x.kode).map((x) => x.kode!)
    timer.current[b._k] = window.setTimeout(() => {
      saranSaldo({ nama, kategori: pilihKategori, dipakai }).then((s) => {
        setBaris((bs) => bs && bs.map((x) => {
          if (x._k !== b._k || x.nama !== nama) return x // ketikan sudah berubah
          const p: Partial<Baris> = { _saran: s }
          if (s.kategori && s.kategori !== 'aset_tetap' && !x._katManual) p.kategori = s.kategori
          if (s.kode && (!x.kode || x._kodeAuto)) { p.kode = s.kode; p._kodeAuto = true }
          return { ...x, ...p }
        }))
      }, () => { /* saran gagal: kategori dipilih sendiri */ })
    }, pilihKategori ? 0 : 600)
  }

  // Dari dalam jendela saldo awal: tutup dulu (isian tetap tersimpan sebagai draf), lalu buka daftar aset.
  function lihatAset() {
    setBaris(null)
    setBukaAset(true)
    setTimeout(() => kartuAset.current?.scrollIntoView({ behavior: 'smooth', block: 'start' }))
  }

  function asetBerubah() {
    setVersiAset((v) => v + 1)
    onChanged()
  }

  async function simpan(e: React.FormEvent) {
    e.preventDefault()
    if (!baris || (selisih !== 0 && !paksa) || tanpaKategori.length) return
    setBusy(true)
    setError('')
    try {
      const rincian = tampil.filter((b) => (b._k === -1 ? b.nominal !== 0 : b.nama.trim() || b.nominal)).map(bersih)
      const res = await saveOpening({ tanggal, kas: 0, piutang: 0, persediaan: 0, utang: 0, pinjaman: 0, modal_disetor: 0, rincian }, selisih !== 0 && paksa)
      setPaksa(false)
      hapusDraft(DRAFT_SALDO)
      setInfo(res)
      setBaris(null)
      onChanged()
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  const pertama = info?.transaksi_pertama
  const tampilSimpan = o ? (o.rincian?.length ? o.rincian : [...dariPos(o, info!.nilai_buku_aset, info!.akun_template), ...(info?.aset_tetap ?? [])]) : []

  return (
    <>
      <PageHeader title="Saldo Awal" subtitle="Neraca pembukaan: posisi usaha di hari pertama memakai aplikasi. Diisi sekali; bulan berikutnya terbentuk sendiri dari transaksi." />

      <div className="note-box" style={{ marginTop: 0, marginBottom: 14 }}>
        Urutannya: <strong>1. daftarkan aset tetap</strong>, <strong>2. daftarkan barang</strong>, lalu <strong>3. isi neraca pembukaan</strong>.
        Nilai buku aset dan nilai persediaan barang otomatis masuk ke saldo awal, jadi tidak diketik dua kali.
        Saldo awal baru bisa disimpan bila <strong>Harta = Kewajiban + Modal</strong>. Banyak akun sekaligus bisa diisi lewat template Excel di <strong>Impor Data Awal</strong>.
      </div>

      {error && !baris && (
        <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start', marginBottom: 12, whiteSpace: 'normal', display: 'flex', gap: 10, alignItems: 'center' }}>
          {error}
          {!info && <button type="button" className="underline underline-offset-4" onClick={() => { setError(''); setVersiAset((v) => v + 1) }}>Muat ulang</button>}
        </div>
      )}

      <div ref={kartuAset} className="card" style={{ padding: 20, marginBottom: 14, scrollMarginTop: 16 }}>
        <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 14, flexWrap: 'wrap' }}>
          <div style={{ flex: 1, minWidth: 240 }}>
            <div className="chart-title">1. Aset tetap</div>
            <div className="chart-subtitle">
              Kendaraan, mesin, peralatan, bangunan yang sudah dimiliki. Nilai bukunya menjadi baris aset tetap di saldo awal, per akun CoA. Tidak punya aset tetap? Lewati langkah ini.
            </div>
          </div>
          <div style={{ display: 'flex', gap: 10, alignItems: 'center', flexWrap: 'wrap' }}>
            <span className={'pill-status ' + (info?.jumlah_aset ? 'pill-green' : 'pill-amber')}>
              {info?.jumlah_aset ? `${info.jumlah_aset} aset · nilai buku ${formatRp(nbv)}` : 'Belum ada aset'}
            </span>
            {bisaUbah && (
              <button type="button" className={bukaAset ? 'nav-item' : 'btn-cta'} style={{ width: 'auto', padding: '8px 16px', fontSize: 13 }} onClick={() => setBukaAset((b) => !b)}>
                {bukaAset ? 'Tutup daftar aset' : info?.jumlah_aset ? 'Kelola daftar aset' : 'Tambah aset tetap'}
              </button>
            )}
          </div>
        </div>
        {bukaAset && (
          <div style={{ marginTop: 14 }}>
            {asetData ? <Aset data={asetData} onChanged={asetBerubah} tanggalSaldoAwal={tanggal || o?.tanggal} /> : <p className="text-sm text-muted-foreground">Memuat daftar aset…</p>}
          </div>
        )}
      </div>

      <div className="card" style={{ padding: 20, marginBottom: 14 }}>
        <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 14, flexWrap: 'wrap' }}>
          <div style={{ flex: 1, minWidth: 240 }}>
            <div className="chart-title">2. Barang &amp; persediaan</div>
            <div className="chart-subtitle">
              {info?.label_persediaan ?? 'Nilai daftar barang'}: <strong>{formatRp(nilaiBarang)}</strong>. Angka ini yang dipakai sebagai persediaan di saldo awal.
            </div>
          </div>
          <button type="button" className="nav-item" style={{ width: 'auto', padding: '8px 16px', fontSize: 13 }} onClick={() => navigate('stok-gudang')}>Buka Stok &amp; Gudang</button>
        </div>
      </div>

      <div className="card" style={{ padding: 20 }}>
        <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 14, flexWrap: 'wrap' }}>
          <div>
            <div className="chart-title">3. Neraca pembukaan</div>
            <div className="chart-subtitle">
              {!info ? (error ? 'Saldo awal belum bisa dimuat.' : 'Memuat…')
                : info.aktif && o ? `Mulai ${formatTanggal(o.tanggal)}.` : 'Belum diisi. Laporan dimulai dari nol di bulan transaksi pertama.'}
            </div>
          </div>
          {info && (
            <div style={{ display: 'flex', gap: 10, alignItems: 'center' }}>
              <span className={'pill-status ' + (info.aktif ? 'pill-green' : 'pill-amber')}>{info.aktif ? 'Sudah diisi' : 'Belum diisi'}</span>
              {info.terkunci && <span className="pill-status pill-amber">Terkunci</span>}
              {bisaUbah && !baris && !info.terkunci && (
                <button className="btn-cta" style={{ padding: '8px 16px', fontSize: 13 }} onClick={mulaiUbah}>
                  {info.aktif ? 'Ubah saldo awal' : 'Isi saldo awal'}
                </button>
              )}
            </div>
          )}
        </div>
        {info?.terkunci && (
          <div className="note-box warn" style={{ marginTop: 12, display: 'flex', gap: 10, alignItems: 'center', flexWrap: 'wrap' }}>
            <span style={{ flex: 1, minWidth: 220 }}>{info.terkunci}</span>
            <button type="button" className="nav-item" style={{ width: 'auto', padding: '7px 14px', fontSize: 13 }} onClick={() => navigate('tutup-buku')}>Buka Tutup Buku</button>
          </div>
        )}

        {!baris && info?.aktif && o && (
          <div className="table-scroll" style={{ marginTop: 14 }}>
            <table className="data-table tabular">
              <thead><tr><th>Akun</th><th>Kategori</th><th style={{ textAlign: 'right' }}>Nominal</th></tr></thead>
              <tbody>
                {tampilSimpan.filter((r) => r.nominal).map((r, i) => (
                  <tr key={i}>
                    <td>{r.kode && <span className="text-muted-foreground">{r.kode} </span>}{r.nama}</td>
                    <td>{kategori.find((k) => k.id === r.kategori)?.label ?? r.kategori}</td>
                    <td style={{ textAlign: 'right' }}>{formatRp(r.nominal)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
            {info.selisih_modal !== 0 && (
              <p className="text-sm" style={{ color: RED, marginTop: 8 }}>
                Saldo awal ini belum seimbang (selisih {formatRp(info.selisih_modal)}). Klik <strong>Ubah saldo awal</strong> lalu samakan Harta dengan Kewajiban + Modal.
              </p>
            )}
          </div>
        )}

        <Dialog open={baris !== null}>
          <DialogContent className="sm:max-w-5xl" showCloseButton={false}
            onInteractOutside={(e) => e.preventDefault()} onEscapeKeyDown={(e) => e.preventDefault()}>
            <DialogHeader>
              <DialogTitle>{info?.aktif ? 'Ubah saldo awal' : 'Isi saldo awal'}</DialogTitle>
              <DialogDescription>
                Neraca pembukaan per akun. Tekan <strong>Simpan saldo awal</strong> untuk menyimpan, atau <strong>Batal</strong> untuk menutup tanpa menyimpan.
              </DialogDescription>
            </DialogHeader>
            {error && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start', whiteSpace: 'normal' }}>{error}</div>}
          {baris && <form id="form-saldo-awal" onSubmit={simpan}>
            {dipulihkan && (
              <p className="text-xs" role="status" style={{ color: 'var(--text-warn)', marginBottom: 10 }}>
                Isian sebelumnya yang belum tersimpan dipulihkan.{' '}
                <button type="button" className="underline underline-offset-4" onClick={() => { hapusDraft(DRAFT_SALDO); setBaris(null); setTimeout(mulaiUbah) }}>Mulai dari awal</button>
              </p>
            )}
            <label style={{ display: 'grid', gap: 4, maxWidth: 320, marginBottom: 12 }}>
              <span className="kpi-label">Tanggal mulai pembukuan</span>
              <input className="input-field" type="date" required value={tanggal} max={pertama || undefined} onChange={(e) => setTanggal(e.target.value)} />
              <span style={{ fontSize: 11, lineHeight: 1.35, color: 'var(--text-muted)' }}>
                {pertama ? `Paling lambat ${formatTanggal(pertama)} (transaksi pertama yang sudah dicatat).` : 'Posisi saldo di pagi hari tanggal ini.'}
              </span>
            </label>

            <div className="table-scroll">
              <table className="data-table tabular" style={{ minWidth: 820 }}>
                <thead>
                  <tr><th style={{ width: 90 }}>Kode</th><th>Nama akun</th><th style={{ width: 230 }}>Kategori akun</th><th style={{ width: 170, textAlign: 'right' }}>Nominal (Rp)</th><th>Keterangan</th><th style={{ width: 40 }} /></tr>
                </thead>
                <tbody>
                  {tampil.map((b, i) => {
                    if (b._k === -1) {
                      return (
                        <tr key={`aset-${i}`}>
                          <td><div className="input-field tabular" style={{ borderStyle: 'dashed' }}>{b.kode}</div></td>
                          <td><div className="input-field" style={{ borderStyle: 'dashed' }}>{b.nama}</div></td>
                          <td><div className="input-field" style={{ borderStyle: 'dashed' }}>{kategori.find((k) => k.id === 'aset_tetap')?.label}</div></td>
                          <td><div className="input-field tabular" style={{ textAlign: 'right', borderStyle: 'dashed' }} title="Dari daftar aset">{formatRp(b.nominal)}</div></td>
                          <td>
                            <button type="button" className="text-xs underline underline-offset-4" onClick={lihatAset}>
                              {b.nominal ? `${b.keterangan ?? 'dari daftar aset'} · ubah di langkah 1` : 'Tambah aset di langkah 1'}
                            </button>
                          </td>
                          <td />
                        </tr>
                      )
                    }
                    const r = b as Baris
                    const s = r._saran
                    return (
                      <tr key={r._k}>
                        <td><input className="input-field" aria-label="Kode akun" value={r.kode ?? ''} placeholder="otomatis" onChange={(e) => ubah(r._k, { kode: e.target.value, _kodeAuto: false })} /></td>
                        <td>
                          <input className="input-field" aria-label="Nama akun" value={r.nama} placeholder="mis. Bank BCA, Pinjaman koperasi"
                            onChange={(e) => {
                              const nama = e.target.value
                              ubah(r._k, { nama })
                              if (r._baru && nama.trim().length >= 3) mintaSaran(r, nama)
                            }} />
                          {r._baru && s && (
                            s.kategori === 'aset_tetap' ? (
                              <div className="text-xs" style={{ color: AMBER, marginTop: 4 }}>
                                Ini aset tetap: daftarkan di <button type="button" className="underline underline-offset-4" onClick={lihatAset}>langkah 1 (daftar aset)</button> supaya nilai buku &amp; penyusutannya dihitung, lalu hapus baris ini.
                              </div>
                            ) : s.kategori && (
                              <div className="text-xs text-muted-foreground" style={{ marginTop: 4, display: 'flex', gap: 4, alignItems: 'flex-start' }}>
                                {s.sumber === 'ai' && <Sparkles size={12} style={{ flexShrink: 0, marginTop: 2, color: 'var(--accent-blue)' }} aria-hidden />}
                                <span>
                                  {s.sumber === 'ai' ? `Saran AI${s.alasan ? `: ${s.alasan}` : ''}` : s.sumber === 'coa' ? `Sesuai akun CoA "${s.akun}"` : s.sumber === 'aturan' ? 'Ditebak dari nama akun' : 'Kategori dipilih sendiri'}
                                  {s.baru && s.kode && ` · kode baru ${s.kode}`}
                                </span>
                              </div>
                            )
                          )}
                        </td>
                        <td>
                          <select className="input-field" aria-label="Kategori akun" value={r.kategori}
                            onChange={(e) => {
                              const kat = e.target.value
                              ubah(r._k, { kategori: kat, _katManual: true })
                              if (kat && (!r.kode || r._kodeAuto)) mintaSaran(r, r.nama, kat)
                            }}>
                            {!r.kategori && <option value="">Pilih kategori…</option>}
                            {kategori.filter((k) => k.id !== 'aset_tetap').map((k) => <option key={k.id} value={k.id}>{k.label}</option>)}
                          </select>
                        </td>
                        <td><input className="input-field tabular" style={{ textAlign: 'right' }} aria-label="Nominal" {...angka(r.nominal, (n) => ubah(r._k, { nominal: n }))} /></td>
                        <td><input className="input-field" aria-label="Keterangan" value={r.keterangan ?? ''} onChange={(e) => ubah(r._k, { keterangan: e.target.value })} /></td>
                        <td>
                          <button type="button" className="nav-item" style={{ width: 'auto', padding: 6 }} aria-label={`Hapus baris ${r.nama}`} onClick={() => setBaris((bs) => bs && bs.filter((x) => x._k !== r._k))}>
                            <Trash2 size={14} aria-hidden="true" />
                          </button>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
            <div style={{ display: 'flex', gap: 10, marginTop: 8, flexWrap: 'wrap' }}>
              <button type="button" className="nav-item" style={{ width: 'auto', padding: '7px 12px' }} onClick={() => setBaris((bs) => [...(bs ?? []), jadiBaris({ kode: '', nama: '', kategori: '', nominal: 0 }, { _baru: true })])}>
                <Plus size={14} aria-hidden="true" /> Tambah akun
              </button>
              <span className="text-xs text-muted-foreground" style={{ alignSelf: 'center' }}>Ketik nama akunnya; kategori &amp; kode terisi otomatis. Rekening kas/bank, aset lancar lain, dan kewajiban lain baru menjadi akun sendiri di Daftar Akun &amp; Buku Besar.</span>
            </div>

            <div className="list-rows tabular" style={{ marginTop: 16 }} aria-live="polite">
              <div className="list-row"><span>Total Harta</span><span>{formatRp(harta)}</span></div>
              <div className="list-row"><span>Total Utang (Kewajiban)</span><span>{formatRp(kewajiban)}</span></div>
              <div className="list-row"><span>Total Modal</span><span>{formatRp(modal)}</span></div>
              <div className="list-row" style={{ border: `1px solid ${selisih === 0 ? GREEN : RED}` }}>
                <strong>Harta = Utang + Modal</strong>
                <strong style={{ color: selisih === 0 ? GREEN : RED }}>{selisih === 0 ? 'Seimbang' : `Belum seimbang · selisih ${formatRp(selisih)}`}</strong>
              </div>
              {selisih !== 0 && (
                <p className="text-sm" style={{ color: RED }}>
                  {selisih > 0 ? 'Harta lebih besar' : 'Kewajiban + Modal lebih besar'} {formatRp(Math.abs(selisih))}. Periksa kembali angka per akun, atau sesuaikan Modal Pemilik / Laba Ditahan.
                </p>
              )}
              {totalPersediaan !== nilaiBarang && (
                <p className="text-sm" style={{ color: AMBER }}>
                  Persediaan di sini {formatRp(totalPersediaan)}, sedangkan {(info?.label_persediaan ?? 'nilai daftar barang').toLowerCase()} {formatRp(nilaiBarang)}.{' '}
                  {biasa.some((b) => b.kategori === 'persediaan') && nilaiBarang - totalPersediaan + (biasa.find((b) => b.kategori === 'persediaan')!.nominal || 0) >= 0 && (
                    <button type="button" className="underline underline-offset-4" onClick={() => {
                      const p = biasa.find((b) => b.kategori === 'persediaan')!
                      ubah(p._k, { nominal: (p.nominal || 0) + nilaiBarang - totalPersediaan })
                    }}>Samakan dengan daftar barang</button>
                  )}
                </p>
              )}
              {selisih !== 0 && (
                <label className="note-box warn" style={{ display: 'flex', gap: 8, alignItems: 'flex-start', margin: 0 }}>
                  <input type="checkbox" checked={paksa} onChange={(e) => setPaksa(e.target.checked)} />
                  <span>
                    <strong>Saya sudah memeriksa dan tetap ingin menyimpan.</strong> Selisih {formatRp(selisih)} akan dicatat di akun
                    <strong> Laba Ditahan – Selisih Saldo Awal</strong> supaya neraca tetap seimbang, dan terus diperingatkan sampai angka per akunnya dibereskan.
                  </span>
                </label>
              )}
              {biasa.some((b) => b.nominal < 0) && (
                <p className="text-sm" style={{ color: AMBER }}>
                  Ada saldo minus: {biasa.filter((b) => b.nominal < 0).map((b) => `${b.nama} ${formatRp(b.nominal)}`).join(', ')}. Saldo minus berarti berlawanan dengan saldo normal akunnya
                  (mis. laba ditahan minus = rugi tahun lalu); pastikan memang begitu.
                </p>
              )}
              {tanpaKategori.length > 0 && (
                <p className="text-sm" style={{ color: RED }}>Pilih kategori untuk: {tanpaKategori.map((b) => b.nama || 'baris tanpa nama').join(', ')}.</p>
              )}
            </div>

          </form>}
            <DialogFooter className="sticky -bottom-5 z-10 bg-popover" style={{ paddingTop: 10, paddingBottom: 10 }}>
              <button type="button" className="nav-item" style={{ width: 'auto', padding: '9px 16px' }} onClick={() => { hapusDraft(DRAFT_SALDO); setBaris(null); setError('') }}>Batal</button>
              <button className="btn-cta" style={{ padding: '9px 20px', fontSize: 13 }} type="submit" form="form-saldo-awal" disabled={busy || (selisih !== 0 && !paksa) || tanpaKategori.length > 0}>
                {busy ? 'Menyimpan…' : selisih !== 0 && paksa ? 'Simpan dengan selisih' : 'Simpan saldo awal'}
              </button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        <div className="note-box warn">
          Isi sekali di awal, bukan tiap bulan. Piutang dan utang awal yang dibayar kemudian dicatat di <strong>Piutang &amp; Utang</strong>.
        </div>
      </div>
      {info?.aktif && cek.length > 0 && <PemeriksaanKecocokan cek={cek} navigate={navigate} />}
    </>
  )
}

/** Panel kecocokan: saldo awal dibandingkan dengan daftar aset, daftar barang/kartu stok, dan neraca. */
function PemeriksaanKecocokan({ cek, navigate }: { cek: CekKecocokan[]; navigate: (v: View, tab?: string) => void }) {
  const selisih = cek.filter((c) => c.status === 'selisih').length
  return (
    <div className="card" style={{ padding: 20, marginTop: 14 }}>
      <div className="chart-title">Pemeriksaan kecocokan</div>
      <div className="chart-subtitle">
        {selisih ? `${selisih} selisih perlu dibereskan supaya laporan dan pajak tidak keliru.` : 'Saldo awal cocok dengan daftar aset dan daftar barang.'}
      </div>
      <div className="grid gap-2" style={{ marginTop: 12 }}>
        {cek.map((c) => {
          const warna = c.status === 'selisih' ? 'var(--accent-amber)' : c.status === 'ok' ? 'var(--accent-green)' : 'var(--text-muted)'
          const Ikon = c.status === 'selisih' ? AlertTriangle : c.status === 'ok' ? CheckCircle2 : Info
          return (
            <div key={c.kode} className="flex items-start gap-3 rounded-lg border px-3 py-2.5" style={{ borderColor: c.status === 'selisih' ? 'var(--accent-amber)' : 'var(--border-shell)' }}>
              <Ikon size={18} style={{ color: warna, flexShrink: 0, marginTop: 2 }} aria-label={c.status} />
              <div style={{ flex: 1, minWidth: 0 }}>
                <div className="text-sm font-semibold">{c.judul}</div>
                {c.status !== 'info' && (
                  <div className="text-xs" style={{ marginTop: 2 }}>
                    Saldo awal <strong>{formatRp(c.saldo_awal)}</strong> · {c.label_banding} <strong>{formatRp(c.pembanding)}</strong>
                    {c.selisih !== 0 && <> · selisih <strong style={{ color: warna }}>{formatRp(c.selisih)}</strong></>}
                  </div>
                )}
                {c.pesan && <div className="text-xs text-muted-foreground" style={{ marginTop: 2 }}>{c.pesan}</div>}
              </div>
              {c.status === 'selisih' && c.view !== 'saldo-awal' && (
                <button type="button" className="text-xs underline underline-offset-4" style={{ flexShrink: 0 }}
                  onClick={() => navigate(c.view, c.kode === 'aset' ? 'aset' : undefined)}>Buka</button>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}
