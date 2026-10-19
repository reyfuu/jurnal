import { Bug, Send } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { PageHeader } from '@/components/PageHeader'
import { getBugInfo, kirimBug, tanggapiBug, type BugInfo, type BugInput, type LaporanBug, type StatusBug, type View } from '@/lib/api'
import { VIEW_LABEL } from '@/lib/nav'

const STATUS_CLS: Record<string, string> = {
  baru: 'pill-amber',
  dibaca: 'pill-status-muted',
  selesai: 'pill-green',
  'bukan bug': 'pill-status-muted',
}

const KEPARAHAN_LABEL: Record<string, string> = {
  ringan: 'Ringan — masih bisa dipakai',
  mengganggu: 'Mengganggu — angkanya salah atau harus diakali',
  parah: 'Parah — tidak bisa dipakai / data hilang',
  pertanyaan: 'Bukan bug — cuma bertanya',
}

const KOSONG: BugInput = { judul: '', deskripsi: '', langkah: '', keparahan: 'mengganggu', halaman: '' }

const waktu = (iso: string) => {
  const d = new Date(iso)
  return Number.isNaN(d.getTime())
    ? iso
    : d.toLocaleString('id-ID', { day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}

/**
 * Menu Lapor Bug.
 *
 * `view` adalah menu yang terakhir dibuka pengguna, dipakai mengisi kolom Halaman otomatis —
 * halaman tempat masalah terjadi adalah hal pertama yang ditanyakan saat menelusuri bug, dan
 * paling sering tidak diisi kalau harus ditulis sendiri.
 */
export function LaporBug({ view }: { view: View }) {
  const [info, setInfo] = useState<BugInfo | null>(null)
  const [form, setForm] = useState<BugInput>({ ...KOSONG, halaman: VIEW_LABEL[view] ?? '' })
  const [error, setError] = useState('')
  const [hasil, setHasil] = useState<LaporanBug | null>(null)
  const [busy, setBusy] = useState(false)

  const muat = useCallback(() => { getBugInfo().then(setInfo, (e: Error) => setError(e.message)) }, [])
  useEffect(muat, [muat])

  async function kirim(e: React.FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      const l = await kirimBug(form)
      setHasil(l)
      setForm({ ...KOSONG, halaman: VIEW_LABEL[view] ?? '' })
      muat()
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  async function tanggapi(l: LaporanBug, status: StatusBug) {
    try {
      await tanggapiBug(l.id, status, l.balasan ?? '')
      muat()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  if (!info) {
    return error
      ? <p className="workspace-subtitle" role="alert">{error}</p>
      : <p className="workspace-subtitle">Memuat data…</p>
  }

  const list = info.laporan ?? []
  const k = info.konteks

  return (
    <>
      <PageHeader
        title="Lapor Bug"
        subtitle={info.master
          ? 'Anda masuk sebagai ID Master: daftar di bawah berisi laporan dari semua usaha dan bisa ditanggapi.'
          : 'Ada angka yang keliru atau tombol yang tidak jalan? Ceritakan di sini supaya bisa diperbaiki.'}
      >
        <span className={'pill-status ' + (info.penerusan_aktif ? 'pill-green' : 'pill-amber')}>
          {info.penerusan_aktif ? `Diteruskan ke ${info.penerusan_ke}` : 'Tersimpan di server'}
        </span>
      </PageHeader>

      {error && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start', marginBottom: 12 }}>{error}</div>}

      {hasil && (
        <div className={'note-box ' + (hasil.diteruskan ? '' : 'warn')} style={{ marginBottom: 14 }}>
          <strong>Laporan #{hasil.id} terkirim.</strong>{' '}
          {hasil.diteruskan
            ? 'Sudah diteruskan ke pengembang.'
            : info.penerusan_aktif
              ? `Tersimpan di server, tapi penerusan ke pengembang gagal: ${hasil.gagal_kirim}. Laporannya tidak hilang — pengembang masih bisa membacanya di aplikasi.`
              : 'Tersimpan di server dan bisa dibaca pengembang di aplikasi. Penerusan otomatis belum diaktifkan.'}
        </div>
      )}

      <form className="card" style={{ padding: 22, marginBottom: 16 }} onSubmit={kirim}>
        <div className="chart-title">Ceritakan Masalahnya</div>
        <div className="chart-subtitle">Semakin jelas langkahnya, semakin cepat bisa ditelusuri dan diperbaiki.</div>

        <div className="form-grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', marginTop: 14 }}>
          <label style={{ gridColumn: '1 / -1' }}>
            <span className="kpi-label">Judul singkat</span>
            <input
              className="input-field" required maxLength={150} placeholder="mis. Laba bersih di Laporan Keuangan beda dengan Transaksi"
              value={form.judul} onChange={(e) => setForm({ ...form, judul: e.target.value })}
            />
          </label>
          <label>
            <span className="kpi-label">Seberapa mengganggu?</span>
            <select className="input-field" value={form.keparahan} onChange={(e) => setForm({ ...form, keparahan: e.target.value })}>
              {info.keparahan.map((x) => <option key={x} value={x}>{KEPARAHAN_LABEL[x] ?? x}</option>)}
            </select>
          </label>
          <label>
            <span className="kpi-label">Terjadi di menu</span>
            <input
              className="input-field" value={form.halaman} placeholder="mis. Laporan Keuangan"
              onChange={(e) => setForm({ ...form, halaman: e.target.value })}
            />
            <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>terisi otomatis dari menu terakhir yang dibuka</span>
          </label>
          <label style={{ gridColumn: '1 / -1' }}>
            <span className="kpi-label">Apa yang terjadi?</span>
            <textarea
              className="input-field" required rows={4} maxLength={4000} style={{ resize: 'vertical', minHeight: 90, paddingTop: 10 }}
              placeholder="Apa yang Anda harapkan, dan apa yang muncul? Sebutkan angka atau pesan errornya kalau ada."
              value={form.deskripsi} onChange={(e) => setForm({ ...form, deskripsi: e.target.value })}
            />
            <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>{form.deskripsi.length}/4000</span>
          </label>
          <label style={{ gridColumn: '1 / -1' }}>
            <span className="kpi-label">Langkah untuk memunculkannya (opsional, tapi sangat membantu)</span>
            <textarea
              className="input-field" rows={3} maxLength={2000} style={{ resize: 'vertical', minHeight: 70, paddingTop: 10 }}
              placeholder={'1. Buka menu Transaksi\n2. Pilih bulan September\n3. Angka totalnya ...'}
              value={form.langkah} onChange={(e) => setForm({ ...form, langkah: e.target.value })}
            />
          </label>
        </div>

        <div className="note-box" style={{ marginTop: 14 }}>
          <strong>Yang ikut terkirim bersama laporan ini:</strong>{' '}
          nama ({k.pelapor || '—'}), email ({k.email}), peran ({k.peran}), nomor usaha ({k.usaha}),
          versi aplikasi ({k.versi}), dan jenis browser yang Anda pakai. Tidak ada data transaksi
          atau laporan keuangan yang ikut dikirim.
          {!info.penerusan_aktif && ' Laporan hanya disimpan di server aplikasi ini, tidak dikirim ke layanan luar.'}
        </div>

        <div style={{ display: 'flex', gap: 10, marginTop: 16 }}>
          <button className="btn-cta inline-flex items-center gap-1.5" style={{ padding: '9px 20px', fontSize: 13 }} type="submit" disabled={busy}>
            <Send size={15} aria-hidden /> {busy ? 'Mengirim…' : 'Kirim Laporan'}
          </button>
        </div>
      </form>

      <div className="card" style={{ padding: 20 }}>
        <div className="chart-title" style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <Bug size={16} aria-hidden /> {info.master ? 'Semua Laporan Masuk' : 'Laporan dari Usaha Anda'}
        </div>
        <div className="chart-subtitle">
          {info.rekap.total} laporan
          {Object.entries(info.rekap.per_status).map(([st, n]) => ` · ${n} ${st}`).join('')}
        </div>

        {list.length === 0 ? (
          <p className="workspace-subtitle" style={{ padding: '20px 0' }}>Belum ada laporan.</p>
        ) : (
          <div style={{ display: 'grid', gap: 12, marginTop: 12 }}>
            {list.map((l) => (
              <div key={l.id} style={{ background: '#101318', borderRadius: 10, padding: 16 }}>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8, alignItems: 'center', marginBottom: 6 }}>
                  <span className={'pill-status ' + (STATUS_CLS[l.status] ?? 'pill-status-muted')} style={{ fontSize: 10 }}>{l.status}</span>
                  <strong>#{l.id} {l.judul}</strong>
                  <span className="pill-status pill-status-muted" style={{ fontSize: 10 }}>{l.keparahan}</span>
                  {l.halaman && <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>di {l.halaman}</span>}
                  {!l.diteruskan && l.gagal_kirim && (
                    <span className="pill-status pill-red" style={{ fontSize: 10 }} title={l.gagal_kirim}>gagal diteruskan</span>
                  )}
                </div>
                <p style={{ whiteSpace: 'pre-wrap', margin: '0 0 8px' }}>{l.deskripsi}</p>
                {l.langkah && (
                  <>
                    <div className="kpi-label">Langkah</div>
                    <p style={{ whiteSpace: 'pre-wrap', margin: '0 0 8px', color: 'var(--text-secondary)' }}>{l.langkah}</p>
                  </>
                )}
                <p className="kpi-meta-row" style={{ margin: 0 }}>
                  {waktu(l.dibuat)} · {l.pelapor || l.email} ({l.peran})
                  {info.master && ` · usaha ${l.tenant_id}`} · versi {l.versi}
                </p>
                {info.master && (
                  <p className="kpi-meta-row" style={{ margin: '4px 0 0', fontSize: 11 }}>{l.user_agent}</p>
                )}

                {l.balasan && (
                  <div className="note-box" style={{ marginTop: 10 }}>
                    <strong>Tanggapan pengembang:</strong> {l.balasan}
                  </div>
                )}

                {info.master && (
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8, marginTop: 10 }}>
                    {info.status_list.filter((st) => st !== l.status).map((st) => (
                      <button key={st} className="nav-item" style={{ width: 'auto', padding: '4px 10px', fontSize: 12 }} onClick={() => void tanggapi(l, st)}>
                        Tandai {st}
                      </button>
                    ))}
                  </div>
                )}
              </div>
            ))}
          </div>
        )}

        {!info.penerusan_aktif && info.master && (
          <div className="note-box warn" style={{ marginTop: 14 }}>
            Penerusan otomatis belum aktif. Isi <code>BUG_WEBHOOK_URL</code> di server (webhook Slack,
            Discord, atau endpoint sendiri) supaya setiap laporan baru langsung masuk ke sana; tanpa itu
            laporan hanya bisa dibaca di halaman ini.
          </div>
        )}
      </div>
    </>
  )
}
