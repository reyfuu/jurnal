import { useEffect, useRef, useState } from 'react'
import { PageHeader } from '@/components/PageHeader'
import {
  getOpening, importOnboarding, ONBOARDING_TEMPLATE_URL,
  type OnboardingResult, type OpeningInfo,
} from '@/lib/api'
import { formatRp } from '@/lib/format'

const RED = 'var(--text-danger)'

export function ImporDataAwal({ onChanged }: { onChanged: () => void }) {
  const [info, setInfo] = useState<OpeningInfo | null>(null)
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

  const errors = preview?.errors ?? []
  const bisaCommit = preview !== null && !preview.imported && preview.bisa_disimpan !== false && (preview.opening !== null ||
    (preview.assets?.length ?? 0) > 0 || (preview.karyawan?.length ?? 0) > 0 || preview.transaksi_count > 0)

  return (
    <>
      <PageHeader title="Impor Data Awal" subtitle="Satu berkas Excel untuk saldo awal rinci, aset tetap, stok, karyawan, dan transaksi">
        <a className="btn-cta" style={{ padding: '9px 18px', fontSize: 13, textDecoration: 'none' }} href={ONBOARDING_TEMPLATE_URL}>
          Unduh Template
        </a>
      </PageHeader>

      {error && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start', marginBottom: 12 }}>{error}</div>}

      <div className="note-box" style={{ marginTop: 0, marginBottom: 14 }}>
        <strong>Yang diimpor dari laporan keuangan adalah posisi awalnya, bukan laporannya.</strong> Neraca dan
        arus kas di aplikasi ini dihitung dari transaksi + register aset, jadi kalau angka jadi ikut diimpor,
        dia akan langsung bertabrakan dengan hasil hitung begitu ada satu transaksi baru. Isi Tanggal Saldo Awal
        di sheet <strong>Petunjuk</strong>, lalu sheet rincian <strong>Kas dan Bank</strong> s.d. <strong>Modal</strong>{' '}
        hingga sheet <strong>Neraca Saldo Awal</strong> berstatus SEIMBANG — sisanya terbentuk sendiri.
      </div>

      {/* --- Langkah 1: unggah --- */}
      <div className="card" style={{ padding: 20, marginBottom: 14 }}>
        <div className="chart-title">1. Unggah Berkas</div>
        <div className="chart-subtitle">
          Format .xlsx, maksimal 5 MB. Pakai template: Petunjuk, Neraca Saldo Awal, Kas dan Bank, Piutang, Persediaan,
          Perlengkapan, Aset Lancar Lain, Aset Tetap, Utang, Modal, plus Karyawan &amp; Transaksi (opsional). Berkas format
          lama (sheet Saldo Awal / Aset) tetap bisa dibaca.
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
                <span className="tabular">Kas {formatRp(preview.opening.kas)}</span>
              </div>
            )}
            <div className="list-row">
              <span>Aset Tetap</span>
              <span className="tabular">{preview.assets?.length ?? 0} baris</span>
            </div>
            <div className="list-row">
              <span>Persediaan → Stok &amp; Gudang</span>
              <span className="tabular">{preview.stok?.length ?? 0} barang</span>
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

          {preview.opening && (() => {
            const o = preview.opening!
            const nbv = preview.nilai_buku_aset ?? 0
            const harta = o.kas + o.piutang + o.persediaan + (o.aset_lain ?? 0) + nbv
            const kewajiban = o.utang + o.pinjaman + (o.kewajiban_lain ?? 0)
            const modal = o.modal_disetor + (o.laba_ditahan ?? 0)
            const perAkun = (o.rincian?.length ?? 0) > 0
            const seimbang = !perAkun || harta - kewajiban - modal === 0
            return (
              <div className="list-rows tabular">
                <div className="section-head">Pembanding saldo awal: Harta = Kewajiban + Modal</div>
                <div className="list-row"><span>Total Harta (termasuk aset tetap {formatRp(nbv)})</span><span>{formatRp(harta)}</span></div>
                <div className="list-row"><span>Total Kewajiban</span><span>{formatRp(kewajiban)}</span></div>
                <div className="list-row"><span>Total Modal</span><span>{perAkun ? formatRp(modal) : `${formatRp(preview.modal_awal)} (dihitung, format lama)`}</span></div>
                <div className="list-row" style={{ border: `1px solid ${seimbang ? 'var(--accent-green)' : RED}` }}>
                  <strong>Selisih</strong>
                  <strong style={{ color: seimbang ? 'var(--accent-green)' : RED }}>{seimbang ? 'Seimbang' : formatRp(harta - kewajiban - modal)}</strong>
                </div>
              </div>
            )
          })()}

          {preview.bisa_disimpan === false && (
            <div className="note-box warn">
              <strong>Belum bisa disimpan.</strong> Perbaiki masalah bertanda <em>Saldo Awal</em> di tabel di bawah (saldo awal harus seimbang dan
              aset tetapnya sama dengan nilai buku sheet Aset Tetap), lalu unggah ulang berkasnya.
            </div>
          )}

          {errors.length > 0 && (
            <>
              <div className="section-head" style={{ marginTop: 16 }}>Masalah &amp; baris yang dilewati ({errors.length})</div>
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

          {preview.imported && (preview.replaced_assets > 0 || preview.replaced_karyawan > 0 || (preview.replaced_stok ?? 0) > 0) && (
            <div className="note-box">
              Mengganti {preview.replaced_assets} aset, {preview.replaced_stok ?? 0} barang stok, dan {preview.replaced_karyawan} karyawan
              yang sebelumnya tercatat. Transaksi tidak diganti, hanya ditambahkan.
            </div>
          )}

          {bisaCommit && (
            <div style={{ display: 'flex', gap: 10, marginTop: 16, alignItems: 'center', flexWrap: 'wrap' }}>
              <button className="btn-cta" style={{ padding: '9px 20px', fontSize: 13 }} onClick={() => void commit()} disabled={busy}>
                {busy ? 'Mengimpor…' : 'Impor Sekarang'}
              </button>
              <span className="chart-subtitle">
                Aset, stok &amp; karyawan hasil impor <strong>mengganti</strong> yang sekarang; transaksi ditambahkan.
              </span>
            </div>
          )}
        </div>
      )}

      <div className="note-box" style={{ marginBottom: 0 }}>
        Mau mengisi saldo awal tanpa Excel? Pakai menu <strong>Pengaturan → Saldo Awal</strong>.
        {info?.aktif && info.opening && <> Saldo awal sekarang mulai {info.opening.tanggal}.</>}
      </div>

      <div className="note-box" style={{ marginBottom: 0 }}>
        Setelah impor selesai, cek <strong>Laporan Keuangan → Neraca</strong>: badge{' '}
        <span className="pill-status pill-green" style={{ fontSize: 10 }}>Aktiva = Pasiva</span> harus hijau.
        Kalau merah, ada isian neraca awal yang belum pas.
      </div>
    </>
  )
}
