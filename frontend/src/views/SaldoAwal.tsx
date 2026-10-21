import { useEffect, useState } from 'react'
import { PageHeader } from '@/components/PageHeader'
import { getOpening, saveOpening, type OpeningBalance, type OpeningInfo, type View } from '@/lib/api'
import { formatRp, formatTanggal } from '@/lib/format'
import { angka } from '@/lib/angka'

const RED = 'var(--text-danger)'
const BLUE = 'var(--accent-blue)'

const KOSONG: OpeningBalance = {
  tanggal: '', kas: 0, piutang: 0, persediaan: 0, utang: 0, pinjaman: 0, modal_disetor: 0,
}

// Isian saldo awal dalam bahasa sehari-hari: apa yang dimiliki dan apa yang masih harus dibayar.
const FIELDS: { key: keyof OpeningBalance; label: string; hint: string; sisi: 'punya' | 'utang' | 'cek' }[] = [
  { key: 'kas', label: 'Uang tunai & saldo rekening usaha', hint: 'Uang di laci/kasir + saldo semua rekening bank & e-wallet usaha pada pagi hari tanggal mulai.', sisi: 'punya' },
  { key: 'piutang', label: 'Tagihan ke pelanggan (piutang)', hint: 'Penjualan yang sudah terjadi tapi belum dibayar pelanggan.', sisi: 'punya' },
  { key: 'persediaan', label: 'Nilai stok barang (persediaan)', hint: 'Bahan baku & barang dagangan yang ada, dihitung dengan harga beli.', sisi: 'punya' },
  { key: 'utang', label: 'Tagihan dari supplier (utang usaha)', hint: 'Pembelian atau biaya yang sudah diterima tapi belum dibayar.', sisi: 'utang' },
  { key: 'pinjaman', label: 'Sisa pinjaman', hint: 'Sisa pokok pinjaman bank, koperasi, leasing, atau pribadi untuk usaha.', sisi: 'utang' },
  { key: 'modal_disetor', label: 'Modal yang Anda setor (opsional)', hint: 'Hanya untuk dicocokkan. Modal awal dihitung otomatis dari isian di atas.', sisi: 'cek' },
]

/** Menu Saldo Awal: posisi usaha di hari pertama memakai aplikasi, diisi sekali. */
export function SaldoAwal({ onChanged, navigate, bisaUbah }: { onChanged: () => void; navigate: (v: View, tab?: string) => void; bisaUbah: boolean }) {
  const [info, setInfo] = useState<OpeningInfo | null>(null)
  const [form, setForm] = useState<OpeningBalance | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    getOpening().then(setInfo, (e: Error) => setError(e.message))
  }, [])

  async function simpan(e: React.FormEvent) {
    e.preventDefault()
    if (!form) return
    setBusy(true)
    setError('')
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

  const punya = form ? form.kas + form.piutang + form.persediaan : 0
  const kewajiban = form ? form.utang + form.pinjaman : 0
  const pertama = info?.transaksi_pertama

  return (
    <>
      <PageHeader title="Saldo Awal" subtitle="Posisi usaha di hari pertama memakai aplikasi. Cukup diisi sekali; bulan-bulan berikutnya terbentuk sendiri dari transaksi." />

      <div className="note-box" style={{ marginTop: 0, marginBottom: 14 }}>
        <strong>Cara mengisi:</strong> pilih tanggal mulai pembukuan (mis. 1 Januari), lalu isi posisi usaha di <strong>pagi hari tanggal itu</strong>:
        uang yang ada, tagihan ke pelanggan, stok, dan yang masih harus dibayar. <strong>Modal awal dihitung otomatis</strong> (yang dimiliki dikurangi utang).
        Mesin, kendaraan, dan peralatan yang sudah dimiliki didaftarkan terpisah di{' '}
        <button type="button" className="underline underline-offset-4" onClick={() => navigate('laporan-keuangan', 'aset')}>Laporan Keuangan → Aset &amp; Penyusutan</button>{' '}
        dengan tanggal pembelian aslinya.
      </div>

      {error && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start', marginBottom: 12 }}>{error}</div>}

      <div className="card" style={{ padding: 20 }}>
        <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 14, flexWrap: 'wrap' }}>
          <div>
            <div className="chart-title">Saldo awal usaha</div>
            <div className="chart-subtitle">
              {info?.aktif && info.opening
                ? `Mulai ${formatTanggal(info.opening.tanggal)}. Semua laporan memakai angka ini sebagai posisi awal.`
                : 'Belum diisi. Laporan dimulai dari nol di bulan transaksi pertama.'}
            </div>
          </div>
          <div style={{ display: 'flex', gap: 10, alignItems: 'center' }}>
            <span className={'pill-status ' + (info?.aktif ? 'pill-green' : 'pill-amber')}>{info?.aktif ? 'Sudah diisi' : 'Belum diisi'}</span>
            {bisaUbah && !form && (
              <button className="btn-cta" style={{ padding: '8px 16px', fontSize: 13 }}
                onClick={() => setForm(info?.opening ? { ...info.opening } : { ...KOSONG, tanggal: pertama ?? '' })}>
                {info?.aktif ? 'Ubah saldo awal' : 'Isi saldo awal'}
              </button>
            )}
          </div>
        </div>

        {info?.aktif && info.opening && !form && (
          <div className="list-rows tabular">
            {FIELDS.filter((f) => f.sisi !== 'cek').map((f) => (
              <div className="list-row" key={f.key}>
                <span>{f.label}</span>
                <span className="tabular" style={{ color: f.sisi === 'utang' ? RED : undefined }}>{formatRp(info.opening![f.key] as number)}</span>
              </div>
            ))}
            <div className="list-row" style={{ border: '1px solid color-mix(in srgb, var(--accent-blue) 30%, transparent)' }}>
              <span><strong>Modal awal (otomatis)</strong> <span className="text-xs text-muted-foreground">termasuk nilai buku aset yang sudah dimiliki</span></span>
              <strong style={{ color: BLUE }}>{formatRp(info.modal_awal)}</strong>
            </div>
            {info.selisih_modal !== 0 && (
              <div className="list-row">
                <span>Selisih dengan modal yang Anda setor</span>
                <strong style={{ color: RED }}>{formatRp(info.selisih_modal)}</strong>
              </div>
            )}
          </div>
        )}

        {form && (
          <form onSubmit={simpan} style={{ marginTop: 14 }}>
            <div className="form-grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(230px, 1fr))' }}>
              <label>
                <span className="kpi-label">Tanggal mulai pembukuan</span>
                <input className="input-field" type="date" required value={form.tanggal} max={pertama || undefined}
                  onChange={(e) => setForm({ ...form, tanggal: e.target.value })} />
                <span style={{ fontSize: 11, lineHeight: 1.35, color: 'var(--text-muted)' }}>
                  {pertama
                    ? `Paling lambat ${formatTanggal(pertama)} (transaksi pertama yang sudah dicatat), supaya tidak terhitung dua kali.`
                    : 'Hari pertama yang mau dicatat di aplikasi ini.'}
                </span>
              </label>
              {FIELDS.map((f) => (
                <label key={f.key}>
                  <span className="kpi-label">{f.label}</span>
                  <input className="input-field tabular" {...angka(form[f.key] as number, (n) => setForm({ ...form, [f.key]: n }))} />
                  <span style={{ fontSize: 11, lineHeight: 1.35, color: 'var(--text-muted)' }}>{f.hint}</span>
                </label>
              ))}
            </div>
            <div className="list-rows tabular" style={{ marginTop: 14 }}>
              <div className="list-row"><span>Yang dimiliki (uang + tagihan + stok)</span><span>{formatRp(punya)}</span></div>
              <div className="list-row"><span>Yang masih harus dibayar (utang + pinjaman)</span><span style={{ color: RED }}>{formatRp(kewajiban)}</span></div>
              <div className="list-row"><strong>Modal awal, belum termasuk aset</strong><strong style={{ color: BLUE }}>{formatRp(punya - kewajiban)}</strong></div>
            </div>
            <div style={{ display: 'flex', gap: 10, marginTop: 16 }}>
              <button className="btn-cta" style={{ padding: '9px 20px', fontSize: 13 }} type="submit" disabled={busy}>
                {busy ? 'Menyimpan…' : 'Simpan saldo awal'}
              </button>
              <button type="button" className="nav-item" style={{ width: 'auto', padding: '9px 16px' }} onClick={() => setForm(null)}>Batal</button>
            </div>
          </form>
        )}

        <div className="note-box warn">
          Isi sekali di awal, bukan tiap bulan. Saldo bulan berikutnya terbentuk sendiri dari transaksi.
          Piutang dan utang awal yang dibayar kemudian dicatat di <strong>Piutang &amp; Utang</strong>.
        </div>
      </div>
    </>
  )
}
