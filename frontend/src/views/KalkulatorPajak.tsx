import { useEffect, useState } from 'react'
import { PageHeader } from '@/components/PageHeader'
import {
  cekPKP, getKalkulator, hitungPPh21, hitungPPhBadan, hitungPotong,
  type HasilPKP, type KalkulatorMaster, type Langkah, type PotongResult, type PPh21Response, type PPhBadanResult,
} from '@/lib/api'
import { formatRp } from '@/lib/format'

const AMBER = '#facc15'
const RED = '#fb7185'
const GREEN = '#10b981'

/** 1500 → "15%", 175 → "1,75%". */
const persen = (bps: number) =>
  (bps % 100 === 0 ? String(bps / 100) : (bps / 100).toFixed(2).replace(/0+$/, '').replace(/\.$/, '')).replace('.', ',') + '%'

type TabID = 'pph21' | 'pph-badan' | string

export function KalkulatorPajak() {
  const [m, setM] = useState<KalkulatorMaster | null>(null)
  const [tab, setTab] = useState<TabID>('pph21')
  const [error, setError] = useState('')

  useEffect(() => { getKalkulator().then(setM, (e) => setError((e as Error).message)) }, [])
  if (!m) return <p className="workspace-subtitle">{error || 'Memuat kalkulator…'}</p>

  const tabs: { id: TabID; label: string }[] = [
    { id: 'pph21', label: 'PPh 21' },
    ...m.jenis.map((j) => ({ id: j.id, label: j.nama.replace('PPh Pasal', 'PPh') })),
    { id: 'pph-badan', label: 'PPh Badan' },
    { id: 'pkp', label: 'Cek PKP' },
  ]

  return (
    <>
      <PageHeader
        title="Kalkulator Pajak"
        subtitle="Hitung PPh 21, 22, 23, 4 ayat (2), 15, PPh Badan, PPN, dan PPnBM. Setiap hasil menampilkan langkah perhitungannya."
      />

      <div className="card" style={{ padding: '12px 18px', marginBottom: 14, borderColor: 'rgba(250,204,21,0.3)' }}>
        <div style={{ fontSize: 12.5, color: 'var(--text-muted)', lineHeight: 1.6 }}>
          <span style={{ color: AMBER, fontWeight: 600 }}>Alat bantu, bukan pengganti konsultan.</span>{' '}
          Tarif mengikuti ketentuan yang berlaku umum dan setiap objek mencantumkan rujukan aturannya.
          Minta tim pajak memverifikasi baris yang relevan sebelum dipakai untuk SPT.
        </div>
      </div>

      <div className="tabbar" role="tablist" aria-label="Jenis pajak">
        {tabs.map((t) => (
          <button key={t.id} role="tab" className="tab-btn" aria-selected={t.id === tab} onClick={() => setTab(t.id)}>
            {t.label}
          </button>
        ))}
      </div>

      {tab === 'pph21' && <FormPPh21 m={m} />}
      {tab === 'pph-badan' && <FormPPhBadan m={m} />}
      {tab === 'pkp' && <FormCekPKP />}
      {m.jenis.filter((j) => j.id === tab).map((j) => <FormPotong key={j.id} jenis={j} />)}
    </>
  )
}

// ===== PPh 21 =====

function FormPPh21({ m }: { m: KalkulatorMaster }) {
  const [bruto, setBruto] = useState(10_000_000)
  const [ptkp, setPtkp] = useState('TK/0')
  const [pensiun, setPensiun] = useState(0)
  const [grossUp, setGrossUp] = useState(false)
  const [npwp, setNpwp] = useState(true)
  const [r, setR] = useState<PPh21Response | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  async function hitung(e: React.FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      setR(await hitungPPh21({ penghasilan_bruto: bruto, ptkp_status: ptkp, iuran_pensiun: pensiun, gross_up: grossUp, punya_npwp: npwp }))
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <Dua
      kiri={
        <form onSubmit={hitung}>
          <div className="chart-title">PPh Pasal 21</div>
          <div className="chart-subtitle">Pegawai tetap, metode tahunan Pasal 17</div>
          <Uang label="Penghasilan bruto sebulan" v={bruto} set={setBruto} hint="Gaji pokok + tunjangan tetap" />
          <label style={{ display: 'block', marginBottom: 12 }}>
            <span className="kpi-label" style={{ display: 'block', marginBottom: 4 }}>Status PTKP</span>
            <select className="input-field" value={ptkp} onChange={(e) => setPtkp(e.target.value)}>
              {m.ptkp_codes.map((c) => <option key={c} value={c}>{c} — {formatRp(m.ptkp[c])}</option>)}
            </select>
          </label>
          <Uang label="Iuran pensiun sebulan" v={pensiun} set={setPensiun} hint="Ditanggung karyawan; mengurangi penghasilan neto" />
          <label style={{ display: 'block', marginBottom: 10 }}>
            <span className="kpi-label" style={{ display: 'block', marginBottom: 4 }}>Skema penghitungan</span>
            <select className="input-field" value={grossUp ? 'up' : 'gross'} onChange={(e) => setGrossUp(e.target.value === 'up')}>
              <option value="gross">Gross — pajak dipotong dari gaji karyawan</option>
              <option value="up">Gross Up — pajak ditanggung perusahaan</option>
            </select>
          </label>
          <label style={{ display: 'flex', gap: 9, alignItems: 'center', fontSize: 13, marginBottom: 14 }}>
            <input type="checkbox" checked={npwp} onChange={(e) => setNpwp(e.target.checked)} />
            Karyawan punya NPWP <span style={{ color: 'var(--text-muted)' }}>(tanpa NPWP dipotong 20% lebih tinggi)</span>
          </label>
          <button className="btn-cta" style={{ padding: '9px 22px', fontSize: 13 }} disabled={busy}>
            {busy ? 'Menghitung…' : 'Hitung'}
          </button>
          {error && <p className="pill-status pill-red" role="alert" style={{ marginTop: 12 }}>{error}</p>}
        </form>
      }
      kanan={r && (
        <>
          <Hasil
            utama={[
              { label: 'PPh 21 sebulan', v: r.hasil.pph_sebulan, warna: AMBER },
              { label: 'PPh 21 setahun', v: r.hasil.pph_setahun },
              { label: 'Take home sebulan', v: r.hasil.take_home_sebulan, warna: GREEN },
            ]}
            langkah={r.hasil.langkah}
          />
          {r.hasil.tambahan_tanpa_npwp > 0 && (
            <p className="kpi-meta-row" style={{ color: RED, marginTop: 10 }}>
              Termasuk tambahan {formatRp(r.hasil.tambahan_tanpa_npwp)} karena tanpa NPWP.
            </p>
          )}
          <div style={{ marginTop: 16, paddingTop: 14, borderTop: '1px solid var(--line, #262c38)' }}>
            <div className="kpi-label" style={{ marginBottom: 6 }}>Pembanding: TER bulanan (PMK 168/2023)</div>
            {r.ter.tersedia ? (
              <div style={{ fontSize: 13 }}>
                Tarif efektif {persen(r.ter.tarif_bps!)} → <b>{formatRp(r.ter.pph_sebulan!)}</b> sebulan
              </div>
            ) : (
              <div style={{ fontSize: 12.5, color: 'var(--text-muted)', lineHeight: 1.6 }}>{r.ter.alasan}</div>
            )}
          </div>
        </>
      )}
    />
  )
}

// ===== PPh potong-pungut (23, 4(2), 22, 15, PPN, PPnBM) =====

export function FormPotong({ jenis }: { jenis: KalkulatorMaster['jenis'][number] }) {
  const [objek, setObjek] = useState(jenis.objek[0].kode)
  const [bruto, setBruto] = useState(10_000_000)
  const [npwp, setNpwp] = useState(true)
  const [r, setR] = useState<PotongResult | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  // Tidak perlu mereset saat ganti jenis: pemanggil memberi key={j.id}, jadi komponen ini
  // di-remount dan seluruh state mulai bersih dengan sendirinya.

  const dipilih = jenis.objek.find((o) => o.kode === objek)!

  async function hitung(e: React.FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      setR(await hitungPotong({ jenis: jenis.id, objek, nilai_bruto: bruto, punya_npwp: npwp }))
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <Dua
      kiri={
        <form onSubmit={hitung}>
          <div className="chart-title">{jenis.nama}</div>
          <div className="chart-subtitle">{jenis.keterangan}</div>
          <label style={{ display: 'block', marginBottom: 12 }}>
            <span className="kpi-label" style={{ display: 'block', marginBottom: 4 }}>Objek pajak</span>
            <select className="input-field" value={objek} onChange={(e) => setObjek(e.target.value)}>
              {jenis.objek.map((o) => (
                <option key={o.kode} value={o.kode}>{o.nama} — {persen(o.tarif_bps)}{o.final ? ' (final)' : ''}</option>
              ))}
            </select>
            <span style={{ display: 'block', fontSize: 11.5, color: 'var(--text-muted)', marginTop: 4 }}>
              Dasar: {dipilih.dasar}{dipilih.catatan ? ` · ${dipilih.catatan}` : ''}
            </span>
          </label>
          <Uang label="Nilai bruto" v={bruto} set={setBruto} />
          {jenis.tanpa_npwp_bps ? (
            <label style={{ display: 'flex', gap: 9, alignItems: 'center', fontSize: 13, marginBottom: 14 }}>
              <input type="checkbox" checked={npwp} onChange={(e) => setNpwp(e.target.checked)} />
              Lawan transaksi punya NPWP
              <span style={{ color: 'var(--text-muted)' }}>(tanpa NPWP {jenis.tanpa_npwp_bps / 10000}x lebih tinggi)</span>
            </label>
          ) : null}
          <button className="btn-cta" style={{ padding: '9px 22px', fontSize: 13 }} disabled={busy}>
            {busy ? 'Menghitung…' : 'Hitung'}
          </button>
          {error && <p className="pill-status pill-red" role="alert" style={{ marginTop: 12 }}>{error}</p>}
        </form>
      }
      kanan={r && (
        <>
          <Hasil
            utama={[
              { label: 'Dasar pengenaan pajak', v: r.dpp },
              { label: `Pajak (${persen(r.tarif_bps)})`, v: r.pajak, warna: AMBER },
              { label: 'Diterima setelah potong', v: r.diterima, warna: GREEN },
            ]}
            langkah={r.langkah}
          />
          <p className="kpi-meta-row" style={{ marginTop: 10 }}>
            {r.final ? 'Bersifat final — tidak diperhitungkan lagi di SPT Tahunan.' : 'Tidak final — menjadi kredit pajak di SPT Tahunan.'}
            {' '}Dasar: {r.dasar}.
          </p>
        </>
      )}
    />
  )
}

// ===== PPh Badan =====

export function FormPPhBadan({ m }: { m: KalkulatorMaster }) {
  const [omzet, setOmzet] = useState(4_000_000_000)
  const [pkp, setPkp] = useState(500_000_000)
  const [r, setR] = useState<PPhBadanResult | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  async function hitung(e: React.FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      setR(await hitungPPhBadan({ peredaran_bruto: omzet, penghasilan_kena_pajak: pkp }))
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <Dua
      kiri={
        <form onSubmit={hitung}>
          <div className="chart-title">PPh Badan</div>
          <div className="chart-subtitle">Tarif {persen(m.tarif_badan_bps)} dengan fasilitas Pasal 31E</div>
          <Uang label="Peredaran bruto setahun" v={omzet} set={setOmzet} hint="Menentukan besarnya fasilitas Pasal 31E" />
          <Uang label="Penghasilan kena pajak setahun" v={pkp} set={setPkp} hint="Laba fiskal setelah koreksi" />
          <button className="btn-cta" style={{ padding: '9px 22px', fontSize: 13 }} disabled={busy}>
            {busy ? 'Menghitung…' : 'Hitung'}
          </button>
          {error && <p className="pill-status pill-red" role="alert" style={{ marginTop: 12 }}>{error}</p>}
        </form>
      }
      kanan={r && (
        <>
          <Hasil
            utama={[
              { label: 'PPh badan terutang', v: r.pph_terutang, warna: AMBER },
              { label: 'Tarif efektif', v: r.pkp > 0 ? Math.round((r.pph_terutang / r.pkp) * 10000) / 100 : 0, persen: true },
            ]}
            langkah={r.langkah}
          />
          <p className="kpi-meta-row" style={{ marginTop: 10 }}>{r.catatan}</p>
        </>
      )}
    />
  )
}

// ===== Cek ambang PKP =====

export function FormCekPKP() {
  const [omzet, setOmzet] = useState(3_000_000_000)
  const [r, setR] = useState<HasilPKP | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  async function hitung(e: React.FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      setR(await cekPKP(omzet))
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <Dua
      kiri={
        <form onSubmit={hitung}>
          <div className="chart-title">Cek Ambang PKP</div>
          <div className="chart-subtitle">Batas pengusaha kecil PPN: omzet Rp4,8 miliar setahun</div>
          <Uang label="Omzet setahun (atau kumulatif tahun berjalan)" v={omzet} set={setOmzet} hint="Seluruh peredaran bruto" />
          <button className="btn-cta" style={{ padding: '9px 22px', fontSize: 13 }} disabled={busy}>
            {busy ? 'Menghitung…' : 'Cek'}
          </button>
          {error && <p className="pill-status pill-red" role="alert" style={{ marginTop: 12 }}>{error}</p>}
        </form>
      }
      kanan={r && (
        <>
          <div className="chart-title">Hasil</div>
          <p className={'pill-status ' + (r.wajib_pkp ? 'pill-red' : 'pill-green')} style={{ marginTop: 12 }}>
            {r.wajib_pkp ? 'Wajib dikukuhkan sebagai PKP' : 'Belum wajib PKP'}
          </p>
          <div className="progress-track" aria-hidden="true">
            <div className="progress-bar" style={{ width: r.pemakaian_bps / 100 + '%', background: r.wajib_pkp ? RED : undefined }} />
          </div>
          <div style={{ display: 'flex', justifyContent: 'space-between', gap: 16, padding: '10px 0', fontSize: 13 }}>
            <span style={{ color: 'var(--text-muted)' }}>{persen(r.pemakaian_bps)} dari batas {formatRp(r.batas)}</span>
            <span className="tabular" style={{ fontWeight: 700 }}>{r.wajib_pkp ? `Lewat ${formatRp(r.omzet - r.batas)}` : `Sisa ${formatRp(r.sisa)}`}</span>
          </div>
          <ul style={{ display: 'grid', gap: 8, paddingLeft: 18, listStyle: 'disc', fontSize: 13, lineHeight: 1.6 }}>
            {r.catatan.map((c) => <li key={c}>{c}</li>)}
          </ul>
        </>
      )}
    />
  )
}

// ===== Bagian bersama =====

function Dua({ kiri, kanan }: { kiri: React.ReactNode; kanan: React.ReactNode }) {
  return (
    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(330px, 1fr))', gap: 14 }}>
      <div className="card" style={{ padding: 20, justifyContent: 'flex-start' }}>{kiri}</div>
      <div className="card" style={{ padding: 20, justifyContent: 'flex-start' }}>
        {kanan ?? <p className="workspace-subtitle" style={{ margin: 0 }}>Isi formulir di sebelah kiri lalu tekan Hitung.</p>}
      </div>
    </div>
  )
}

function Uang({ label, v, set, hint }: { label: string; v: number; set: (n: number) => void; hint?: string }) {
  return (
    <label style={{ display: 'block', marginBottom: 12 }}>
      <span className="kpi-label" style={{ display: 'block', marginBottom: 4 }}>{label}</span>
      <input className="input-field tabular" type="number" min={0} value={v || ''} onChange={(e) => set(Number(e.target.value))} />
      <span style={{ display: 'block', fontSize: 11.5, color: 'var(--text-muted)', marginTop: 4 }}>
        {formatRp(v)}{hint ? ` · ${hint}` : ''}
      </span>
    </label>
  )
}

function Hasil({ utama, langkah }: { utama: { label: string; v: number; warna?: string; persen?: boolean }[]; langkah: Langkah[] }) {
  return (
    <>
      <div className="chart-title">Hasil</div>
      <div style={{ marginTop: 12, marginBottom: 16 }}>
        {utama.map((u) => (
          <div key={u.label} style={{ display: 'flex', justifyContent: 'space-between', gap: 16, padding: '8px 0' }}>
            <span style={{ color: 'var(--text-muted)', fontSize: 13 }}>{u.label}</span>
            <span className="tabular" style={{ fontWeight: 700, fontSize: 16, color: u.warna }}>
              {u.persen ? `${u.v}`.replace('.', ',') + '%' : formatRp(u.v)}
            </span>
          </div>
        ))}
      </div>
      <div className="chart-subtitle" style={{ marginBottom: 6 }}>Langkah perhitungan</div>
      <div className="list-rows tabular">
        {langkah.map((l, i) => (
          <div key={i} style={{ display: 'flex', justifyContent: 'space-between', gap: 16, padding: '6px 0', fontSize: 13 }}>
            <span>
              {l.label}
              {l.keterangan && <span style={{ display: 'block', fontSize: 11, color: 'var(--text-muted)' }}>{l.keterangan}</span>}
            </span>
            <span style={{ whiteSpace: 'nowrap' }}>{formatRp(l.nilai)}</span>
          </div>
        ))}
      </div>
    </>
  )
}
