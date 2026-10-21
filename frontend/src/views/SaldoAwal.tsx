import { useEffect, useState } from 'react'
import { AlertTriangle, CheckCircle2, Info, Plus, Trash2 } from 'lucide-react'
import { PageHeader } from '@/components/PageHeader'
import { getCekSaldoAwal, getOpening, saveOpening, type CekKecocokan, type KategoriSaldo, type OpeningBalance, type OpeningInfo, type SaldoAkun, type View } from '@/lib/api'
import { formatRp, formatTanggal } from '@/lib/format'
import { angka } from '@/lib/angka'

const RED = 'var(--text-danger)'
const GREEN = 'var(--accent-green)'

// Saldo awal lama (tanpa rincian): pos-posnya dijadikan baris supaya bisa diedit per akun.
function dariPos(o: OpeningBalance, nbv: number): SaldoAkun[] {
  const baris: SaldoAkun[] = [
    { kode: '1101', nama: 'Kas & Bank', kategori: 'kas', nominal: o.kas },
    { kode: '1201', nama: 'Piutang Usaha', kategori: 'piutang', nominal: o.piutang },
    { kode: '1301', nama: 'Persediaan', kategori: 'persediaan', nominal: o.persediaan },
    { kode: '', nama: 'Aset tetap (nilai buku, dari daftar aset)', kategori: 'aset_tetap', nominal: nbv },
    { kode: '2101', nama: 'Hutang Usaha', kategori: 'utang', nominal: o.utang },
    { kode: '2102', nama: 'Hutang Bank', kategori: 'pinjaman', nominal: o.pinjaman },
  ]
  const modal = o.kas + o.piutang + o.persediaan + nbv - o.utang - o.pinjaman
  return [...baris, { kode: '3101', nama: 'Modal Pemilik', kategori: 'modal', nominal: modal }]
}

/** Menu Saldo Awal: neraca pembukaan per akun; hanya bisa disimpan bila Harta = Kewajiban + Modal. */
export function SaldoAwal({ onChanged, navigate, bisaUbah }: { onChanged: () => void; navigate: (v: View, tab?: string) => void; bisaUbah: boolean }) {
  const [info, setInfo] = useState<OpeningInfo | null>(null)
  const [tanggal, setTanggal] = useState('')
  const [baris, setBaris] = useState<SaldoAkun[] | null>(null)
  const [nbv, setNbv] = useState(0)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const [cek, setCek] = useState<CekKecocokan[]>([])

  useEffect(() => {
    getOpening().then(setInfo, (e: Error) => setError(e.message))
  }, [])
  // Diperiksa ulang tiap saldo awal tersimpan, supaya selisih yang baru terlihat langsung.
  useEffect(() => { getCekSaldoAwal().then(setCek, () => setCek([])) }, [info])

  // Nilai buku aset mengikuti tanggal mulai yang dipilih (aset dibeli sebelum tanggal itu).
  useEffect(() => {
    if (!baris || tanggal.length !== 10) return
    let hidup = true
    getOpening(tanggal).then((i) => { if (hidup) setNbv(i.nilai_buku_aset) }, () => {})
    return () => { hidup = false }
  }, [tanggal, baris === null]) // eslint-disable-line react-hooks/exhaustive-deps

  const kategori: KategoriSaldo[] = info?.kategori ?? []
  const sisi = (id: string) => kategori.find((k) => k.id === id)?.sisi
  // Baris aset tetap selalu = nilai buku daftar aset; tidak diketik bebas.
  const isi = (baris ?? []).map((b) => (b.kategori === 'aset_tetap' ? { ...b, nominal: nbv } : b))
  const jumlah = (s: string) => isi.filter((b) => sisi(b.kategori) === s).reduce((t, b) => t + (b.nominal || 0), 0)
  const harta = jumlah('harta'), kewajiban = jumlah('kewajiban'), modal = jumlah('modal')
  const selisih = harta - kewajiban - modal
  const adaAsetTetap = isi.some((b) => b.kategori === 'aset_tetap')

  function mulaiUbah() {
    if (!info) return
    const o = info.opening
    setTanggal(o?.tanggal ?? info.transaksi_pertama ?? '')
    setNbv(info.nilai_buku_aset)
    setBaris(o?.rincian?.length ? o.rincian.map((r) => ({ ...r }))
      : o ? dariPos(o, info.nilai_buku_aset)
      : info.akun_template.map((a) => ({ ...a, nominal: 0 })))
    setError('')
  }

  const ubah = (i: number, patch: Partial<SaldoAkun>) => setBaris((bs) => bs!.map((b, j) => (j === i ? { ...b, ...patch } : b)))

  async function simpan(e: React.FormEvent) {
    e.preventDefault()
    if (!baris || selisih !== 0) return
    setBusy(true)
    setError('')
    try {
      const rincian = isi.filter((b) => b.nama.trim() || b.nominal)
      const res = await saveOpening({ tanggal, kas: 0, piutang: 0, persediaan: 0, utang: 0, pinjaman: 0, modal_disetor: 0, rincian })
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
  const o = info?.opening

  return (
    <>
      <PageHeader title="Saldo Awal" subtitle="Neraca pembukaan: posisi usaha di hari pertama memakai aplikasi. Diisi sekali; bulan berikutnya terbentuk sendiri dari transaksi." />

      <div className="note-box" style={{ marginTop: 0, marginBottom: 14 }}>
        Isi per akun: <strong>apa yang dimiliki (Harta)</strong>, <strong>yang masih harus dibayar (Kewajiban)</strong>, dan <strong>modal pemilik</strong>.
        Saldo awal baru bisa disimpan bila <strong>Harta = Kewajiban + Modal</strong>, supaya selisihnya tidak terbawa ke semua laporan.
        Akun yang belum ada bisa ditambah sebagai baris baru. Aset tetap diambil dari{' '}
        <button type="button" className="underline underline-offset-4" onClick={() => navigate('laporan-keuangan', 'aset')}>daftar aset</button> (nilai buku per tanggal mulai).
        Banyak akun sekaligus bisa diisi lewat template Excel di <strong>Impor Data Awal</strong>.
      </div>

      {error && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start', marginBottom: 12, whiteSpace: 'normal' }}>{error}</div>}

      <div className="card" style={{ padding: 20 }}>
        <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 14, flexWrap: 'wrap' }}>
          <div>
            <div className="chart-title">Neraca pembukaan</div>
            <div className="chart-subtitle">
              {info?.aktif && o ? `Mulai ${formatTanggal(o.tanggal)}.` : 'Belum diisi. Laporan dimulai dari nol di bulan transaksi pertama.'}
            </div>
          </div>
          <div style={{ display: 'flex', gap: 10, alignItems: 'center' }}>
            <span className={'pill-status ' + (info?.aktif ? 'pill-green' : 'pill-amber')}>{info?.aktif ? 'Sudah diisi' : 'Belum diisi'}</span>
            {bisaUbah && !baris && info && (
              <button className="btn-cta" style={{ padding: '8px 16px', fontSize: 13 }} onClick={mulaiUbah}>
                {info.aktif ? 'Ubah saldo awal' : 'Isi saldo awal'}
              </button>
            )}
          </div>
        </div>

        {!baris && info?.aktif && o && (
          <div className="table-scroll" style={{ marginTop: 14 }}>
            <table className="data-table tabular">
              <thead><tr><th>Akun</th><th>Kategori</th><th style={{ textAlign: 'right' }}>Nominal</th></tr></thead>
              <tbody>
                {(o.rincian?.length ? o.rincian : dariPos(o, info.nilai_buku_aset)).filter((r) => r.nominal).map((r, i) => (
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

        {baris && (
          <form onSubmit={simpan} style={{ marginTop: 14 }}>
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
                  {isi.map((b, i) => {
                    const tetap = b.kategori === 'aset_tetap'
                    return (
                      <tr key={i}>
                        <td><input className="input-field" aria-label="Kode akun" value={b.kode ?? ''} onChange={(e) => ubah(i, { kode: e.target.value })} /></td>
                        <td><input className="input-field" aria-label="Nama akun" value={b.nama} onChange={(e) => ubah(i, { nama: e.target.value })} /></td>
                        <td>
                          <select className="input-field" aria-label="Kategori akun" value={b.kategori} onChange={(e) => ubah(i, { kategori: e.target.value })}>
                            {kategori.map((k) => <option key={k.id} value={k.id}>{k.label}</option>)}
                          </select>
                        </td>
                        <td>
                          {tetap
                            ? <div className="input-field tabular" style={{ textAlign: 'right', borderStyle: 'dashed' }} title="Dari daftar aset">{formatRp(nbv)}</div>
                            : <input className="input-field tabular" style={{ textAlign: 'right' }} aria-label="Nominal" {...angka(b.nominal, (n) => ubah(i, { nominal: n }))} />}
                        </td>
                        <td><input className="input-field" aria-label="Keterangan" value={b.keterangan ?? ''} onChange={(e) => ubah(i, { keterangan: e.target.value })} /></td>
                        <td>
                          <button type="button" className="nav-item" style={{ width: 'auto', padding: 6 }} aria-label={`Hapus baris ${b.nama}`} onClick={() => setBaris((bs) => bs!.filter((_, j) => j !== i))}>
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
              <button type="button" className="nav-item" style={{ width: 'auto', padding: '7px 12px' }} onClick={() => setBaris((bs) => [...bs!, { kode: '', nama: '', kategori: 'kas', nominal: 0 }])}>
                <Plus size={14} aria-hidden="true" /> Tambah akun
              </button>
              {!adaAsetTetap && nbv > 0 && (
                <button type="button" className="nav-item" style={{ width: 'auto', padding: '7px 12px' }} onClick={() => setBaris((bs) => [...bs!, { kode: '', nama: 'Aset tetap (nilai buku, dari daftar aset)', kategori: 'aset_tetap', nominal: nbv }])}>
                  <Plus size={14} aria-hidden="true" /> Tambah baris aset tetap ({formatRp(nbv)})
                </button>
              )}
            </div>

            <div className="list-rows tabular" style={{ marginTop: 16 }} aria-live="polite">
              <div className="list-row"><span>Total Harta</span><span>{formatRp(harta)}</span></div>
              <div className="list-row"><span>Total Kewajiban</span><span>{formatRp(kewajiban)}</span></div>
              <div className="list-row"><span>Total Modal</span><span>{formatRp(modal)}</span></div>
              <div className="list-row" style={{ border: `1px solid ${selisih === 0 ? GREEN : RED}` }}>
                <strong>Harta − (Kewajiban + Modal)</strong>
                <strong style={{ color: selisih === 0 ? GREEN : RED }}>{selisih === 0 ? 'Seimbang' : formatRp(selisih)}</strong>
              </div>
              {selisih !== 0 && (
                <p className="text-sm" style={{ color: RED }}>
                  {selisih > 0 ? 'Harta lebih besar' : 'Kewajiban + Modal lebih besar'} {formatRp(Math.abs(selisih))}. Periksa kembali angka per akun, atau sesuaikan Modal Pemilik / Laba Ditahan.
                </p>
              )}
              {!adaAsetTetap && nbv > 0 && (
                <p className="text-sm" style={{ color: RED }}>Daftar aset punya nilai buku {formatRp(nbv)} per tanggal ini; tambahkan baris aset tetap supaya ikut dihitung.</p>
              )}
            </div>

            <div style={{ display: 'flex', gap: 10, marginTop: 16 }}>
              <button className="btn-cta" style={{ padding: '9px 20px', fontSize: 13 }} type="submit" disabled={busy || selisih !== 0 || (!adaAsetTetap && nbv > 0)}>
                {busy ? 'Menyimpan…' : 'Simpan saldo awal'}
              </button>
              <button type="button" className="nav-item" style={{ width: 'auto', padding: '9px 16px' }} onClick={() => setBaris(null)}>Batal</button>
            </div>
          </form>
        )}

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
