import { useEffect, useState } from 'react'
import { Centang, Dua, Hasil, Persen, Pilih, TombolHitung, Uang } from '@/components/KalkulatorUI'
import {
  cariNorma, hitungBeliSewa, hitungGrossUp, hitungNPPN, hitungPPNHarga,
  type BeliSewaResult, type BusinessProfile, type GrossUpResult, type KalkulatorMaster, type Norma, type NPPNResult, type PPNHargaResult, type Wilayah,
} from '@/lib/api'
import { formatRp } from '@/lib/format'
import { AMBER, GREEN, RED, keBps, persen, wilayahDariKota } from '@/lib/kalkulator'

// Kalkulator perencanaan (Tax Planning Calculator V2.1): dihitung di server dengan parameter pajak
// versi aktif dari Panel Master. Isian awal diambil dari Profil Usaha bila ada.

type Props = { m: KalkulatorMaster; profile: BusinessProfile | null }

/** Jalankan f, simpan hasil atau pesan galatnya. */
function useHitung<T>() {
  const [r, setR] = useState<T | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  async function jalan(f: () => Promise<T>) {
    setBusy(true)
    setError('')
    try {
      setR(await f())
    } catch (e) {
      setR(null)
      setError((e as Error).message)
    } finally {
      setBusy(false)
    }
  }
  return { r, error, busy, jalan }
}

const Catatan = ({ isi }: { isi: string[] }) => (
  <ul style={{ margin: '12px 0 0', paddingLeft: 18, fontSize: 12.5, color: 'var(--text-muted)', lineHeight: 1.6 }}>
    {isi.map((c) => <li key={c}>{c}</li>)}
  </ul>
)

const ptkpOpsi = (m: KalkulatorMaster): [string, string][] => m.ptkp_codes.map((c) => [c, `${c} — ${formatRp(m.ptkp[c])}`])

// ===== NPPN =====

const WILAYAH: [Wilayah, string][] = [
  ['ibukota10', '10 ibukota provinsi (Medan, Palembang, Jakarta, Bandung, Semarang, Surabaya, Denpasar, Manado, Makassar, Pontianak)'],
  ['ibukota_lain', 'Ibukota provinsi lainnya'],
  ['daerah_lain', 'Daerah lainnya'],
]
const normaWilayah = (n: Norma, w: Wilayah) => ({ ibukota10: n.ibukota10_bps, ibukota_lain: n.ibukota_lain_bps, daerah_lain: n.daerah_lain_bps })[w]

export function FormNPPN({ m, profile }: Props) {
  const [omzet, setOmzet] = useState(profile?.omzet_tahunan || 1_200_000_000)
  const [ptkp, setPtkp] = useState(profile?.bentuk_usaha === 'op' && profile.ptkp ? profile.ptkp : 'TK/0')
  const [q, setQ] = useState(profile?.klu ?? '')
  const [daftar, setDaftar] = useState<Norma[]>([])
  const [klu, setKlu] = useState('')
  const [wilayah, setWilayah] = useState<Wilayah>((wilayahDariKota(profile?.kota) || 'ibukota_lain') as Wilayah)
  const [manual, setManual] = useState('')
  const h = useHitung<NPPNResult>()

  useEffect(() => {
    const t = setTimeout(() => {
      if (!q.trim()) return setDaftar([])
      cariNorma(q).then((d) => { setDaftar(d); setKlu((k) => (d.some((n) => n.klu === k) ? k : d[0]?.klu ?? '')) }, () => setDaftar([]))
    }, 250)
    return () => clearTimeout(t)
  }, [q])
  const dipilih = daftar.find((n) => n.klu === klu)
  const pakaiManual = klu === '' || manual !== ''

  return (
    <Dua
      kiri={
        <form onSubmit={(e) => { e.preventDefault(); void h.jalan(() => hitungNPPN({ omzet, ptkp, wilayah, klu: pakaiManual ? '' : klu, norma_bps: pakaiManual ? keBps(manual) : 0 })) }}>
          <div className="chart-title">Norma Penghitungan (NPPN)</div>
          <div className="chart-subtitle">WP orang pribadi beromzet di bawah {formatRp(m.parameter.batas_nppn)} setahun, tanpa pembukuan</div>
          <Uang label="Omzet setahun" v={omzet} set={setOmzet} />
          <label style={{ display: 'block', marginBottom: 12 }}>
            <span className="kpi-label" style={{ display: 'block', marginBottom: 4 }}>Bidang usaha (KLU 5 digit)</span>
            <input className="input-field" placeholder="Cari kode atau nama, mis. 47111 atau kelontong" value={q} onChange={(e) => setQ(e.target.value)} />
            {daftar.length > 0 && (
              <select className="input-field" style={{ marginTop: 6 }} value={klu} onChange={(e) => setKlu(e.target.value)} aria-label="Pilih KLU">
                {daftar.map((n) => <option key={n.klu} value={n.klu}>{n.klu} — {n.uraian}</option>)}
              </select>
            )}
          </label>
          <Pilih label="Wilayah usaha" v={wilayah} set={(v) => setWilayah(v as Wilayah)} opsi={WILAYAH}
            hint={dipilih ? (normaWilayah(dipilih, wilayah) < 0 ? 'Norma tidak tersedia untuk wilayah ini; isi manual di bawah.' : `Norma ${persen(normaWilayah(dipilih, wilayah))}`) : undefined} />
          <Persen label="Norma manual (%)" v={manual} set={setManual} hint="Opsional; dipakai bila diisi atau KLU tidak ditemukan" />
          <Pilih label="Status PTKP" v={ptkp} set={setPtkp} opsi={ptkpOpsi(m)} />
          <TombolHitung busy={h.busy} error={h.error} />
        </form>
      }
      kanan={h.r && (
        <>
          <Hasil
            utama={[
              { label: 'PPh setahun dengan norma', v: h.r.pph, warna: AMBER },
              { label: `Pembanding PPh final ${m.parameter.dasar_umkm}`, v: h.r.pph_final },
              { label: `Penghasilan neto (${persen(h.r.norma_bps)})`, v: h.r.neto, warna: GREEN },
            ]}
            langkah={h.r.langkah}
          />
          {!h.r.boleh && <p className="kpi-meta-row" style={{ color: RED, marginTop: 10 }}>Omzet ini wajib pembukuan; norma tidak bisa dipakai.</p>}
          {h.r.norma && <p className="kpi-meta-row" style={{ marginTop: 10 }}>KLU {h.r.norma.klu} · {h.r.norma.uraian}</p>}
          <Catatan isi={h.r.catatan} />
        </>
      )}
    />
  )
}

// ===== Gross up =====

export function FormGrossUp({ m }: Props) {
  const opsi: [string, string][] = [
    ['pph21-bukan-pegawai|', 'PPh 21 — orang pribadi bukan pegawai (DPP 50%)'],
    ...m.jenis.filter((j) => j.id === 'pph23' || j.id === 'pph4-2').flatMap((j) =>
      j.objek.map((o): [string, string] => [`${j.id}|${o.kode}`, `${j.nama.replace('PPh Pasal', 'PPh')} — ${o.nama} (${persen(o.tarif_bps)})`])),
  ]
  const [pilih, setPilih] = useState('pph23|23-jasa')
  const [diterima, setDiterima] = useState(200_000_000)
  const [npwp, setNpwp] = useState(true)
  const h = useHitung<GrossUpResult>()
  const [jenis, objek] = pilih.split('|')
  const adaNPWP = jenis === 'pph21-bukan-pegawai' || !!m.jenis.find((j) => j.id === jenis)?.tanpa_npwp_bps

  return (
    <Dua
      kiri={
        <form onSubmit={(e) => { e.preventDefault(); void h.jalan(() => hitungGrossUp({ jenis, objek, diterima, punya_npwp: npwp })) }}>
          <div className="chart-title">Gross Up Nilai Kontrak</div>
          <div className="chart-subtitle">Berapa nilai kontrak / invoice supaya vendor tetap menerima jumlah yang diminta setelah dipotong pajak</div>
          <Pilih label="Jenis pemotongan" v={pilih} set={setPilih} opsi={opsi} />
          <Uang label="Yang harus diterima bersih oleh vendor" v={diterima} set={setDiterima} />
          {adaNPWP && <Centang v={npwp} set={setNpwp}>Vendor punya NPWP <span style={{ color: 'var(--text-muted)' }}>(tanpa NPWP dipotong lebih tinggi)</span></Centang>}
          <TombolHitung busy={h.busy} error={h.error} />
        </form>
      }
      kanan={h.r && (
        <>
          <Hasil
            utama={[
              { label: 'Tulis di kontrak / invoice', v: h.r.nilai_kontrak, warna: AMBER },
              { label: `Pajak dipotong (efektif ${persen(h.r.tarif_efektif_bps)})`, v: h.r.pajak },
              { label: 'Diterima vendor', v: h.r.diterima, warna: GREEN },
            ]}
            langkah={h.r.langkah}
          />
          <p className="kpi-meta-row" style={{ marginTop: 10 }}>
            Selisih nilai kontrak dengan yang diterima vendor ({formatRp(h.r.pajak)}) ditanggung pemberi kerja dan disetor sebagai pajak.
          </p>
        </>
      )}
    />
  )
}

// ===== PPN & harga jual =====

const BARIS_PPN: [string, (s: PPNHargaResult['skema'][number]) => number][] = [
  ['Dibayar pembeli', (s) => s.harga_pembeli], ['Penjualan (DPP)', (s) => s.dpp], ['HPP', (s) => s.hpp],
  ['PPN keluaran', (s) => s.ppn_keluaran], ['PPN masukan dikreditkan', (s) => s.ppn_masukan], ['PPN disetor', (s) => s.ppn_setor],
  ['Biaya operasional', (s) => s.opex], ['Laba bersih', (s) => s.laba],
]

export function FormPPNHarga({ m }: Props) {
  const [pembelian, setPembelian] = useState(1_000_000_000)
  const [termasuk, setTermasuk] = useState(false)
  const [porsi, setPorsi] = useState('100')
  const [markup, setMarkup] = useState('10')
  const [opexMode, setOpexMode] = useState('persen')
  const [opexPersen, setOpexPersen] = useState('2')
  const [opexNominal, setOpexNominal] = useState(0)
  const h = useHitung<PPNHargaResult>()

  return (
    <>
      <Dua
        kiri={
          <form onSubmit={(e) => {
            e.preventDefault()
            void h.jalan(() => hitungPPNHarga({ pembelian, termasuk_ppn: termasuk, porsi_dari_pkp_bps: keBps(porsi), markup_bps: keBps(markup),
              opex_bps: opexMode === 'persen' ? keBps(opexPersen) : 0, opex_nominal: opexMode === 'nominal' ? opexNominal : 0 }))
          }}>
            <div className="chart-title">PPN &amp; Harga Jual</div>
            <div className="chart-subtitle">Belum PKP, PKP dengan PPN dibebankan ke pembeli, atau PKP yang menanggung PPN di dalam harga (efektif {persen(m.parameter.ppn_efektif_bps)})</div>
            <Uang label="Pembelian / HPP" v={pembelian} set={setPembelian} />
            <Centang v={termasuk} set={setTermasuk}>Nilai pembelian sudah termasuk PPN</Centang>
            <Persen label="Porsi pembelian dari PKP (%)" v={porsi} set={setPorsi} hint="Hanya pembelian dari PKP yang punya PPN masukan" />
            <Persen label="Markup (%)" v={markup} set={setMarkup} />
            <Pilih label="Biaya operasional" v={opexMode} set={setOpexMode} opsi={[['persen', '% dari penjualan'], ['nominal', 'Nominal (Rp)']]} />
            {opexMode === 'persen' ? <Persen label="Biaya operasional (% penjualan)" v={opexPersen} set={setOpexPersen} /> : <Uang label="Biaya operasional" v={opexNominal} set={setOpexNominal} />}
            <TombolHitung busy={h.busy} error={h.error} />
          </form>
        }
        kanan={h.r && (
          <>
            <div className="chart-title">Ringkasan</div>
            {(() => {
              const top = h.r.skema.find((s) => s.id === h.r!.laba_tertinggi)!
              return (
                <div style={{ margin: '12px 0', fontSize: 13.5, lineHeight: 1.7 }}>
                  Laba tertinggi: <b style={{ color: GREEN }}>{top.nama}</b> ({formatRp(top.laba)}).<br />
                  {h.r.kenaikan_harga_bps > 0
                    ? <>Jadi PKP dan membebankan PPN membuat harga ke pembeli naik <b style={{ color: AMBER }}>{persen(h.r.kenaikan_harga_bps)}</b> dibanding belum PKP.</>
                    : <>Harga ke pembeli tidak naik bila jadi PKP dan membebankan PPN: PPN masukan yang tadinya ikut di-markup sebagai HPP kini dikreditkan.</>}
                </div>
              )
            })()}
            <Catatan isi={h.r.catatan} />
          </>
        )}
      />
      {h.r && (
        <div className="card" style={{ padding: 20 }}>
          <div className="chart-title">Perbandingan skema</div>
          <div className="table-scroll">
            <table className="data-table tabular" style={{ minWidth: 640 }}>
              <thead>
                <tr><th />{h.r.skema.map((s) => <th key={s.id} style={{ textAlign: 'right', whiteSpace: 'normal' }}>{s.nama}</th>)}</tr>
              </thead>
              <tbody>
                {BARIS_PPN.map(([label, f]) => (
                  <tr key={label} style={label === 'Laba bersih' ? { fontWeight: 700 } : undefined}>
                    <td>{label}</td>
                    {h.r!.skema.map((s) => (
                      <td key={s.id} style={{ textAlign: 'right', color: label === 'Laba bersih' ? (f(s) < 0 ? RED : s.id === h.r!.laba_tertinggi ? GREEN : undefined) : undefined }}>
                        {formatRp(f(s))}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <Catatan isi={h.r.skema.map((s) => `${s.nama}: ${s.keterangan}`)} />
        </div>
      )}
    </>
  )
}

// ===== Beli vs sewa aset =====

// Kelompok harta PMK 72/2023: umur ekonomis; bangunan hanya garis lurus.
const KELOMPOK: [string, string, number, boolean][] = [
  ['k1', 'Kelompok 1 — 4 tahun (komputer, perabot)', 4, false], ['k2', 'Kelompok 2 — 8 tahun (kendaraan, mesin ringan)', 8, false],
  ['k3', 'Kelompok 3 — 16 tahun (mesin berat)', 16, false], ['k4', 'Kelompok 4 — 20 tahun', 20, false],
  ['bp', 'Bangunan permanen — 20 tahun', 20, true], ['btp', 'Bangunan tidak permanen — 10 tahun', 10, true],
]

export function FormBeliSewa({ m, profile }: Props) {
  const op = profile?.bentuk_usaha === 'op'
  const [bentuk, setBentuk] = useState<'badan' | 'op'>(op ? 'op' : 'badan')
  const [ptkp, setPtkp] = useState(op && profile?.ptkp ? profile.ptkp : 'TK/0')
  const [omzet, setOmzet] = useState(profile?.omzet_tahunan || 1_000_000_000)
  const [biaya, setBiaya] = useState(profile?.omzet_tahunan && profile.laba_sebelum_pajak != null ? Math.max(0, profile.omzet_tahunan - profile.laba_sebelum_pajak) : 400_000_000)
  const [harga, setHarga] = useState(100_000_000)
  const [kelompok, setKelompok] = useState('k2')
  const [metode, setMetode] = useState('garis_lurus')
  const [sewa, setSewa] = useState(15_000_000)
  const h = useHitung<BeliSewaResult>()
  const [, , umur, bangunan] = KELOMPOK.find((k) => k[0] === kelompok)!

  return (
    <>
      <Dua
        kiri={
          <form onSubmit={(e) => {
            e.preventDefault()
            void h.jalan(() => hitungBeliSewa({ bentuk, ptkp, omzet, biaya_lain: biaya, harga_aset: harga, umur_tahun: umur, metode, bangunan, sewa_per_tahun: sewa }))
          }}>
            <div className="chart-title">Beli Aset vs Sewa Aset</div>
            <div className="chart-subtitle">Pajak dan total uang keluar selama umur ekonomis aset</div>
            <Pilih label="Bentuk usaha" v={bentuk} set={(v) => setBentuk(v as 'badan' | 'op')} opsi={[['badan', 'Badan (PT / CV) — Pasal 31E'], ['op', 'Orang pribadi dengan pembukuan — Pasal 17']]} />
            {bentuk === 'op' && <Pilih label="Status PTKP" v={ptkp} set={setPtkp} opsi={ptkpOpsi(m)} />}
            <Uang label="Omzet setahun" v={omzet} set={setOmzet} />
            <Uang label="HPP + biaya lain setahun" v={biaya} set={setBiaya} hint="Di luar penyusutan / sewa aset ini" />
            <Uang label="Harga aset (jika beli)" v={harga} set={setHarga} />
            <Pilih label="Kelompok aset" v={kelompok} set={setKelompok} opsi={KELOMPOK.map(([v, l]) => [v, l])} />
            {!bangunan && <Pilih label="Metode penyusutan" v={metode} set={setMetode} opsi={[['garis_lurus', 'Garis lurus'], ['saldo_menurun', 'Saldo menurun']]} />}
            <Uang label="Sewa setahun (jika sewa)" v={sewa} set={setSewa} />
            <TombolHitung busy={h.busy} error={h.error} />
          </form>
        }
        kanan={h.r && (
          <>
            <Hasil
              utama={[
                { label: 'Total kas keluar — beli', v: h.r.beli.total_kas },
                { label: 'Total kas keluar — sewa', v: h.r.sewa.total_kas },
                { label: 'PPh tahun pertama — beli', v: h.r.beli.tahun[0].pph, warna: AMBER },
                { label: 'PPh tahun pertama — sewa', v: h.r.sewa.tahun[0].pph, warna: AMBER },
              ]}
              langkah={[]}
            />
            <div className="card" style={{ padding: '12px 14px', borderColor: 'var(--accent-green)' }}>
              <div className="kpi-label">Lebih hemat selama {umur} tahun</div>
              <div className="kpi-value" style={{ color: GREEN, fontSize: 20 }}>{h.r.rekomendasi === 'beli' ? 'Beli aset' : 'Sewa aset'}</div>
              <div className="kpi-meta-row">selisih {formatRp(h.r.selisih)}</div>
            </div>
            <Catatan isi={h.r.catatan} />
          </>
        )}
      />
      {h.r && (
        <div className="card" style={{ padding: 20 }}>
          <div className="chart-title">Per tahun ({h.r.metode === 'garis_lurus' ? 'garis lurus' : 'saldo menurun'})</div>
          <div className="table-scroll">
            <table className="data-table tabular" style={{ minWidth: 560 }}>
              <thead><tr><th>Tahun</th><th style={{ textAlign: 'right' }}>Penyusutan</th><th style={{ textAlign: 'right' }}>PPh (beli)</th><th style={{ textAlign: 'right' }}>Sewa</th><th style={{ textAlign: 'right' }}>PPh (sewa)</th></tr></thead>
              <tbody>
                {h.r.beli.tahun.map((b, i) => (
                  <tr key={b.tahun}>
                    <td>{b.tahun}</td>
                    <td style={{ textAlign: 'right' }}>{formatRp(b.beban)}</td>
                    <td style={{ textAlign: 'right' }}>{formatRp(b.pph)}</td>
                    <td style={{ textAlign: 'right' }}>{formatRp(h.r!.sewa.tahun[i].beban)}</td>
                    <td style={{ textAlign: 'right' }}>{formatRp(h.r!.sewa.tahun[i].pph)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </>
  )
}
