import { useEffect, useRef, useState } from 'react'
import { PageHeader } from '@/components/PageHeader'
import {
  getOpening, importOnboarding, saveOpening, ONBOARDING_TEMPLATE_URL,
  type OnboardingResult, type OpeningBalance, type OpeningInfo,
} from '@/lib/api'
import { formatJt, formatRp } from '@/lib/format'
import { angka } from '@/lib/angka'

const RED = 'var(--text-danger)'
const BLUE = 'var(--accent-blue)'

const KOSONG: OpeningBalance = {
  tanggal: '', kas: 0, piutang: 0, persediaan: 0, utang: 0, pinjaman: 0, modal_disetor: 0,
}

const FIELDS: { key: keyof OpeningBalance; label: string; hint: string }[] = [
  { key: 'kas', label: 'Kas & Bank', hint: 'saldo tunai + seluruh rekening usaha' },
  { key: 'piutang', label: 'Piutang Usaha', hint: 'tagihan pelanggan yang belum dibayar' },
  { key: 'persediaan', label: 'Persediaan', hint: 'nilai stok bahan baku & barang dagang' },
  { key: 'utang', label: 'Utang Usaha', hint: 'tagihan supplier & biaya belum dibayar' },
  { key: 'pinjaman', label: 'Pinjaman', hint: 'sisa pokok pinjaman bank / koperasi' },
  { key: 'modal_disetor', label: 'Modal Disetor', hint: 'opsional, hanya untuk dicocokkan' },
]

export function ImporDataAwal({ onChanged }: { onChanged: () => void }) {
  const [info, setInfo] = useState<OpeningInfo | null>(null)
  const [form, setForm] = useState<OpeningBalance | null>(null)
  const [preview, setPreview] = useState<OnboardingResult | null>(null)
  const [file, setFile] = useState<File | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const fileRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    getOpening().then(setInfo, (e: Error) => setError(e.message))
  }, [])

  async function pilihBerkas(f: File | null) {
    setFile(f)
    setPreview(null)
    setError('')
    if (!f) return
    setBusy(true)
    try {
      setPreview(await importOnboarding(f, false))
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setBusy(false)
    }
  }

  async function commit() {
    if (!file) return
    setBusy(true)
    try {
      const res = await importOnboarding(file, true)
      setPreview(res)
      setFile(null)
      if (fileRef.current) fileRef.current.value = ''
      setInfo(await getOpening())
      onChanged()
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setBusy(false)
    }
  }

  async function simpanNeracaAwal(e: React.FormEvent) {
    e.preventDefault()
    if (!form) return
    setBusy(true)
    try {
      setInfo(await saveOpening(form))
      setForm(null)
      onChanged()
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  const errors = preview?.errors ?? []
  const bisaCommit = preview !== null && !preview.imported && (preview.opening !== null ||
    (preview.assets?.length ?? 0) > 0 || (preview.karyawan?.length ?? 0) > 0 || preview.transaksi_count > 0)

  return (
    <>
      <PageHeader title="Impor Data Awal" subtitle="Satu berkas Excel untuk neraca awal, aset, karyawan, dan transaksi">
        <a className="btn-cta" style={{ padding: '9px 18px', fontSize: 13, textDecoration: 'none' }} href={ONBOARDING_TEMPLATE_URL}>
          Unduh Template
        </a>
      </PageHeader>

      {error && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start', marginBottom: 12 }}>{error}</div>}

      <div className="note-box" style={{ marginTop: 0, marginBottom: 14 }}>
        <strong>Yang diimpor dari laporan keuangan adalah posisi awalnya, bukan laporannya.</strong> Neraca dan
        arus kas di aplikasi ini dihitung dari transaksi + register aset, jadi kalau angka jadi ikut diimpor,
        dia akan langsung bertabrakan dengan hasil hitung begitu ada satu transaksi baru. Isi sheet{' '}
        <strong>Saldo Awal</strong> dengan posisi di hari pertama pakai aplikasi — sisanya terbentuk sendiri.
      </div>

      {/* --- Langkah 1: unggah --- */}
      <div className="card" style={{ padding: 20, marginBottom: 14 }}>
        <div className="chart-title">1. Unggah Berkas</div>
        <div className="chart-subtitle">
          Format .xlsx, maksimal 5 MB. Nama sheet bebas selama memuat kata Saldo Awal / Aset / Karyawan / Transaksi.
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap', marginTop: 14 }}>
          <input
            ref={fileRef}
            className="input-field"
            style={{ maxWidth: 360 }}
            type="file"
            accept=".xlsx"
            onChange={(e) => void pilihBerkas(e.target.files?.[0] ?? null)}
          />
          {busy && <span className="chart-subtitle">Memproses…</span>}
        </div>
      </div>

      {/* --- Langkah 2: pratinjau --- */}
      {preview && (
        <div className="card" style={{ padding: 20, marginBottom: 14 }}>
          <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 14, flexWrap: 'wrap' }}>
            <div>
              <div className="chart-title">2. Periksa Dulu</div>
              <div className="chart-subtitle">
                {preview.imported
                  ? 'Sudah diimpor. Angka laporan sudah memakai data ini.'
                  : 'Belum ada yang disimpan. Periksa ringkasannya, lalu klik Impor Sekarang.'}
              </div>
            </div>
            {preview.imported
              ? <span className="pill-status pill-green">Berhasil diimpor</span>
              : <span className="pill-status pill-amber">Pratinjau — belum disimpan</span>}
          </div>

          <div className="list-rows tabular">
            <div className="section-head">Sheet yang terbaca</div>
            {(preview.terbaca ?? []).length === 0 && (
              <div className="list-row"><span>Tidak ada sheet yang dikenali</span><span style={{ color: RED }}>0</span></div>
            )}
            {preview.opening && (
              <div className="list-row">
                <span>Saldo Awal — mulai {preview.opening.tanggal}</span>
                <span className="tabular">Kas {formatJt(preview.opening.kas)}</span>
              </div>
            )}
            <div className="list-row">
              <span>Aset</span>
              <span className="tabular">{preview.assets?.length ?? 0} baris</span>
            </div>
            <div className="list-row">
              <span>Karyawan</span>
              <span className="tabular">{preview.karyawan?.length ?? 0} baris</span>
            </div>
            <div className="list-row">
              <span>Transaksi</span>
              <span className="tabular">{preview.transaksi_count} baris</span>
            </div>
          </div>

          {preview.opening && (
            <div className="list-rows tabular">
              <div className="section-head">Modal awal hasil hitung</div>
              <div className="list-row">
                <span>Aktiva awal − kewajiban awal</span>
                <strong style={{ color: BLUE }}>{formatRp(preview.modal_awal)}</strong>
              </div>
              {preview.selisih_modal !== 0 && (
                <div className="list-row" style={{ border: '1px solid color-mix(in srgb, var(--text-danger) 40%, transparent)' }}>
                  <span>Selisih dengan Modal Disetor yang kamu isi</span>
                  <strong style={{ color: RED }}>{formatRp(preview.selisih_modal)}</strong>
                </div>
              )}
            </div>
          )}

          {preview.selisih_modal !== 0 && (
            <div className="note-box warn">
              Modal disetor yang kamu isi tidak sama dengan hasil hitung. Laporan tetap memakai angka hasil
              hitung supaya neraca seimbang — periksa lagi isian kas, piutang, persediaan, utang, dan pinjaman.
            </div>
          )}

          {errors.length > 0 && (
            <>
              <div className="section-head" style={{ marginTop: 16 }}>Baris yang dilewati ({errors.length})</div>
              <div className="table-scroll">
                <table className="data-table">
                  <thead>
                    <tr><th>Sheet</th><th>Baris Excel</th><th>Masalah</th></tr>
                  </thead>
                  <tbody>
                    {errors.map((e, i) => (
                      <tr key={i}>
                        <td>{e.sheet || '—'}</td>
                        <td className="tabular">{e.baris === 0 ? 'header' : e.baris}</td>
                        <td style={{ color: RED }}>{e.pesan}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              <div className="note-box warn">
                Baris di atas tidak ikut diimpor. Baris lain tetap bisa dilanjutkan — perbaiki yang bermasalah
                lalu unggah ulang berkasnya kalau perlu.
              </div>
            </>
          )}

          {preview.imported && (preview.replaced_assets > 0 || preview.replaced_karyawan > 0) && (
            <div className="note-box">
              Mengganti {preview.replaced_assets} aset dan {preview.replaced_karyawan} karyawan yang sebelumnya
              tercatat. Transaksi tidak diganti, hanya ditambahkan.
            </div>
          )}

          {bisaCommit && (
            <div style={{ display: 'flex', gap: 10, marginTop: 16, alignItems: 'center', flexWrap: 'wrap' }}>
              <button className="btn-cta" style={{ padding: '9px 20px', fontSize: 13 }} onClick={() => void commit()} disabled={busy}>
                {busy ? 'Mengimpor…' : 'Impor Sekarang'}
              </button>
              <span className="chart-subtitle">
                Aset &amp; karyawan hasil impor <strong>mengganti</strong> yang sekarang; transaksi ditambahkan.
              </span>
            </div>
          )}
        </div>
      )}

      {/* --- Neraca awal: status & isi manual --- */}
      <div className="card" style={{ padding: 20 }}>
        <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 14, flexWrap: 'wrap' }}>
          <div>
            <div className="chart-title">Neraca Awal</div>
            <div className="chart-subtitle">
              {info?.aktif
                ? `Berlaku sejak ${info.opening?.tanggal}. Laporan memakai angka ini sebagai posisi awal.`
                : 'Belum diisi — laporan masih memakai saldo awal data contoh.'}
            </div>
          </div>
          <div style={{ display: 'flex', gap: 10, alignItems: 'center' }}>
            <span className={'pill-status ' + (info?.aktif ? 'pill-green' : 'pill-amber')}>
              {info?.aktif ? 'Aktif' : 'Belum diisi'}
            </span>
            <button
              className="nav-item"
              style={{ width: 'auto', padding: '8px 14px' }}
              onClick={() => setForm(info?.opening ? { ...info.opening } : { ...KOSONG })}
            >
              {info?.aktif ? 'Ubah' : 'Isi Manual'}
            </button>
          </div>
        </div>

        {info?.aktif && info.opening && !form && (
          <div className="list-rows tabular">
            {FIELDS.filter((f) => f.key !== 'modal_disetor').map((f) => (
              <div className="list-row" key={f.key}>
                <span>{f.label}</span>
                <span className="tabular">{formatRp(info.opening![f.key] as number)}</span>
              </div>
            ))}
            <div className="list-row" style={{ border: '1px solid color-mix(in srgb, var(--accent-blue) 30%, transparent)' }}>
              <span><strong>Modal Awal (hasil hitung)</strong></span>
              <strong style={{ color: BLUE }}>{formatRp(info.modal_awal)}</strong>
            </div>
            {info.selisih_modal !== 0 && (
              <div className="list-row">
                <span>Selisih dengan Modal Disetor</span>
                <strong style={{ color: RED }}>{formatRp(info.selisih_modal)}</strong>
              </div>
            )}
          </div>
        )}

        {form && (
          <form onSubmit={simpanNeracaAwal}>
            <div className="form-grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(190px, 1fr))' }}>
              <label>
                <span className="kpi-label">Tanggal Mulai Pembukuan</span>
                <input
                  className="input-field" type="date" required value={form.tanggal}
                  onChange={(e) => setForm({ ...form, tanggal: e.target.value })}
                />
              </label>
              {FIELDS.map((f) => (
                <label key={f.key}>
                  <span className="kpi-label">{f.label}</span>
                  <input
                    className="input-field tabular" {...angka(form[f.key] as number, (n) => setForm({ ...form, [f.key]: n }))}
                  />
                  <span style={{ fontSize: 10.5, color: 'var(--text-muted)' }}>{f.hint}</span>
                </label>
              ))}
            </div>
            <div style={{ display: 'flex', gap: 10, marginTop: 16 }}>
              <button className="btn-cta" style={{ padding: '9px 20px', fontSize: 13 }} type="submit" disabled={busy}>
                {busy ? 'Menyimpan…' : 'Simpan Neraca Awal'}
              </button>
              <button type="button" className="nav-item" style={{ width: 'auto', padding: '9px 16px' }} onClick={() => setForm(null)}>
                Batal
              </button>
            </div>
          </form>
        )}

        <div className="note-box warn">
          Piutang, utang, dan pinjaman awal hanya dipakai di periode onboarding; pelunasannya lintas periode
          belum ditelusuri. Saldo kas tetap berjalan ke periode berikutnya.
          <br />
          Menyimpan neraca awal mengubah angka seluruh laporan — lakukan sekali di awal, bukan tiap bulan.
        </div>
      </div>

      <div className="note-box" style={{ marginBottom: 0 }}>
        Setelah impor selesai, cek <strong>Laporan Keuangan → Neraca</strong>: badge{' '}
        <span className="pill-status pill-green" style={{ fontSize: 10 }}>Aktiva = Pasiva</span> harus hijau.
        Kalau merah, ada isian neraca awal yang belum pas.
      </div>
    </>
  )
}
