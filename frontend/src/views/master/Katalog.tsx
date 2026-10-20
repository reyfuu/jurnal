import { useCallback, useEffect, useMemo, useState, type FormEvent } from 'react'
import { Download, Plus, Trash2, Upload } from 'lucide-react'
import { PageHeader } from '@/components/PageHeader'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import {
  aktifkanBenchmark, getKatalogBenchmark, getKatalogCOA, hapusDraftCOA, imporBenchmark, imporDraftCOA, simpanDraftCOA, terbitkanCOA,
  unduhBenchmarkUrl, unduhCOAUrl, type Account, type BenchmarkVersi, type KatalogCOA, type StatusKatalog,
} from '@/lib/api'
import { formatTanggal } from '@/lib/format'
import { bandingkanAkun } from '@/lib/katalog'

const STATUS: Record<StatusKatalog, [string, string]> = {
  draft: ['pill-amber', 'Draft'], terbit: ['pill-green', 'Terbit'], aktif: ['pill-green', 'Aktif'], arsip: ['pill-status-muted', 'Arsip'],
}
export const Status = ({ s }: { s: StatusKatalog }) => <span className={'pill-status ' + STATUS[s][0]}>{STATUS[s][1]}</span>
const JENIS_LABEL: Record<string, string> = { perdagangan: 'Perdagangan', jasa: 'Jasa', manufaktur: 'Manufaktur / Produksi' }
const KELOMPOK = ['Aset Lancar', 'Aset Tidak Lancar', 'Liabilitas Jangka Pendek', 'Liabilitas Jangka Panjang', 'Ekuitas',
  'Pendapatan Usaha', 'Harga Pokok Penjualan', 'Beban Operasional', 'Pendapatan Lain-lain', 'Beban Lain-lain']
const KONDISI: [Account['kondisi'], string][] = [['selalu', 'Selalu'], ['jika_ada', 'Jika dimiliki'], ['jika_pt', 'Khusus PT'],
  ['jika_non_pt', 'Khusus OP / CV'], ['jika_perpetual', 'Persediaan perpetual'], ['jika_periodik', 'Persediaan periodik']]
const pct = (bps: number) => (bps < 0 ? '−' : '') + (Math.abs(bps) / 100).toFixed(2).replace('.', ',') + '%'

export function Pesan({ error, info }: { error: string; info: string }) {
  if (error) return <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start', whiteSpace: 'normal' }}>{error}</div>
  if (info) return <div className="pill-status pill-green" role="status" style={{ alignSelf: 'flex-start' }}>{info}</div>
  return null
}

export function ImporDialog({ open, onClose, judul, keterangan, fields, onKirim }: {
  open: boolean; onClose: () => void; judul: string; keterangan: string
  fields: { id: string; label: string; placeholder?: string; wajib?: boolean }[]
  onKirim: (file: File, nilai: Record<string, string>) => Promise<void>
}) {
  const [file, setFile] = useState<File | null>(null)
  const [nilai, setNilai] = useState<Record<string, string>>({})
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  async function kirim(e: FormEvent) {
    e.preventDefault()
    if (!file) return setError('Pilih berkas .xlsx')
    setBusy(true)
    setError('')
    try {
      await onKirim(file, nilai)
      setFile(null)
      setNilai({})
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent>
        <form onSubmit={kirim} className="grid gap-4">
          <DialogHeader><DialogTitle>{judul}</DialogTitle><DialogDescription>{keterangan}</DialogDescription></DialogHeader>
          <div className="grid gap-1.5">
            <Label htmlFor="imp-file">Berkas .xlsx</Label>
            <Input id="imp-file" type="file" accept=".xlsx" onChange={(e) => setFile(e.target.files?.[0] ?? null)} />
          </div>
          {fields.map((f) => (
            <div key={f.id} className="grid gap-1.5">
              <Label htmlFor={'imp-' + f.id}>{f.label}</Label>
              <Input id={'imp-' + f.id} required={f.wajib} placeholder={f.placeholder} value={nilai[f.id] ?? ''} onChange={(e) => setNilai((n) => ({ ...n, [f.id]: e.target.value }))} />
            </div>
          ))}
          {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>Batal</Button>
            <Button type="submit" disabled={busy}>{busy ? 'Mengimpor…' : 'Impor sebagai draft'}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

const AKUN_KOSONG: Account = { kode: '', nama: '', subtype: 'opex', normal: 'debit', kelompok: 'Beban Operasional', kondisi: 'selalu', catatan: '' }

export function TemplateCOAView() {
  const [k, setK] = useState<KatalogCOA | null>(null)
  const [jenis, setJenis] = useState('perdagangan')
  const [pilih, setPilih] = useState<number | null>(null)
  const [impor, setImpor] = useState(false)
  const [akunBaru, setAkunBaru] = useState<Account | null>(null)
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [busy, setBusy] = useState(false)
  const muat = useCallback(() => getKatalogCOA().then(setK, (e: Error) => setError(e.message)), [])
  useEffect(() => { void muat() }, [muat])

  const versi = useMemo(() => (k?.template ?? []).filter((t) => t.jenis === jenis).sort((a, b) => (b.versi || 1e9) - (a.versi || 1e9)), [k, jenis])
  const aktif = versi.find((t) => t.status === 'terbit')
  const draft = versi.find((t) => t.status === 'draft')
  const tampil = versi.find((t) => t.id === pilih) ?? draft ?? aktif
  const diff = tampil?.status === 'draft' && aktif ? bandingkanAkun(aktif.akun, tampil.akun) : null

  async function jalankan(f: () => Promise<unknown>, pesan: string) {
    setBusy(true)
    setError('')
    setInfo('')
    try {
      await f()
      setInfo(pesan)
      await muat()
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setBusy(false)
    }
  }
  const simpanDraft = (akun: Account[], pesan: string) => jalankan(() => simpanDraftCOA(jenis, akun, draft?.catatan ?? ''), pesan)

  return (
    <>
      <PageHeader title="Template COA" subtitle="Bagan akun bawaan per jenis usaha. Klien baru memakai versi terbit terbaru; klien lama tetap di versinya sampai Owner memperbarui.">
        <div className="flex flex-wrap gap-2">
          <Button variant="outline" className="h-10 rounded-full" onClick={() => setImpor(true)}><Upload /> Impor Excel</Button>
          {aktif && !draft && <button className="btn-cta" disabled={busy} onClick={() => void simpanDraft(aktif.akun, 'Draft dibuat dari versi aktif.')}>Buat draft dari versi aktif</button>}
          {!aktif && !draft && jenis !== 'perdagangan' && (
            <button className="btn-cta" disabled={busy} onClick={() => {
              const dagang = k?.template.find((t) => t.jenis === 'perdagangan' && t.status === 'terbit')
              if (dagang) void simpanDraft(dagang.akun, 'Draft dibuat dari template Perdagangan.')
            }}>Mulai dari template Perdagangan</button>
          )}
        </div>
      </PageHeader>
      <div className="tabbar" role="tablist" aria-label="Jenis usaha">
        {(k?.jenis ?? []).map((j) => (
          <button key={j} role="tab" className="tab-btn" aria-selected={jenis === j} onClick={() => { setJenis(j); setPilih(null) }}>{JENIS_LABEL[j] ?? j}</button>
        ))}
      </div>
      <Pesan error={error} info={info} />
      {k && versi.length === 0 && (
        <div className="card text-sm text-muted-foreground" style={{ padding: 20 }}>
          Belum ada template untuk {JENIS_LABEL[jenis]}. Klien jenis ini memakai laporan format ringkas sampai versi pertama terbit.
        </div>
      )}
      {versi.length > 0 && (
        <div className="card" style={{ padding: 20 }}>
          <div className="chart-title">Versi</div>
          <div className="table-scroll">
            <table className="data-table" style={{ minWidth: 720 }}>
              <thead><tr><th>Versi</th><th>Status</th><th style={{ textAlign: 'right' }}>Akun</th><th style={{ textAlign: 'right' }}>Klien</th><th>Catatan</th><th>Oleh</th><th style={{ textAlign: 'right' }}>Aksi</th></tr></thead>
              <tbody>
                {versi.map((t) => (
                  <tr key={t.id} onClick={() => setPilih(t.id)} style={{ cursor: 'pointer', background: t.id === tampil?.id ? 'var(--bg-card-hover)' : undefined }}>
                    <td className="font-semibold">{t.versi ? `v${t.versi}` : 'Draft'}</td>
                    <td><Status s={t.status} /></td>
                    <td className="tabular" style={{ textAlign: 'right' }}>{t.akun.length}</td>
                    <td className="tabular" style={{ textAlign: 'right' }}>{k?.klien_per_template[t.id] ?? 0}</td>
                    <td className="text-sm text-muted-foreground">{t.catatan || '—'}</td>
                    <td className="text-sm">{t.oleh || 'bawaan'}{t.oleh && ` · ${formatTanggal(t.waktu)}`}</td>
                    <td style={{ textAlign: 'right', whiteSpace: 'nowrap' }} onClick={(e) => e.stopPropagation()}>
                      <a className="inline-flex" href={unduhCOAUrl(t.id)} aria-label={`Unduh ${t.versi ? 'v' + t.versi : 'draft'}`}><Button variant="ghost" size="icon-sm" tabIndex={-1}><Download /></Button></a>
                      {t.status === 'draft' && (
                        <>
                          <Button size="sm" disabled={busy} onClick={() => void jalankan(() => terbitkanCOA(t.id), 'Versi baru terbit. Klien lama tetap di versinya sampai Owner memperbarui.')}>Terbitkan</Button>
                          <Button variant="ghost" size="icon-sm" aria-label="Hapus draft" disabled={busy} onClick={() => void jalankan(() => hapusDraftCOA(t.id), 'Draft dihapus.')}><Trash2 /></Button>
                        </>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
      {tampil && (
        <div className="card" style={{ padding: 20 }}>
          <div className="flex flex-wrap items-center gap-3">
            <div className="chart-title">{tampil.versi ? `Akun v${tampil.versi}` : 'Akun di draft'}</div>
            {diff && (
              <span className="text-sm text-muted-foreground">
                dibanding v{aktif!.versi}: <strong style={{ color: 'var(--accent-green)' }}>+{diff.tambah.length}</strong> · <strong style={{ color: 'var(--accent-red)' }}>−{diff.hapus.length}</strong> · {diff.ubah.length} berubah
              </span>
            )}
            {tampil.status === 'draft' && <Button size="sm" variant="outline" style={{ marginLeft: 'auto' }} onClick={() => setAkunBaru({ ...AKUN_KOSONG })}><Plus /> Tambah akun</Button>}
          </div>
          {tampil.status === 'draft' && k && (
            <p className="text-xs text-muted-foreground" style={{ margin: '6px 0' }}>Akun yang dipakai jurnal otomatis wajib ada: {k.kode_wajib.join(', ')}. Diperiksa saat terbit.</p>
          )}
          <div className="table-scroll">
            <table className="data-table tabular" style={{ minWidth: 760 }}>
              <thead><tr><th>Kode</th><th>Nama akun</th><th>Kelompok</th><th>Saldo normal</th><th>Muncul bila</th>{tampil.status === 'draft' && <th />}</tr></thead>
              <tbody>
                {tampil.akun.map((a) => {
                  const baru = diff?.tambah.some((x) => x.kode === a.kode)
                  const ubah = diff?.ubah.some((x) => x.kode === a.kode)
                  return (
                    <tr key={a.kode} style={baru ? { background: 'color-mix(in srgb, var(--accent-green) 10%, transparent)' } : ubah ? { background: 'color-mix(in srgb, var(--accent-amber) 10%, transparent)' } : undefined}>
                      <td>{a.kode}</td><td>{a.nama}</td><td className="text-muted-foreground">{a.kelompok}</td><td>{a.normal}</td>
                      <td>{a.kondisi === 'selalu' ? <span className="text-muted-foreground">Selalu</span> : <span className="pill-status pill-status-muted">{KONDISI.find((x) => x[0] === a.kondisi)?.[1]}</span>}</td>
                      {tampil.status === 'draft' && (
                        <td style={{ textAlign: 'right' }}>
                          <Button variant="ghost" size="icon-sm" aria-label={`Hapus akun ${a.kode}`} disabled={busy}
                            onClick={() => void simpanDraft(tampil.akun.filter((x) => x.kode !== a.kode), `Akun ${a.kode} dihapus dari draft.`)}><Trash2 /></Button>
                        </td>
                      )}
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}
      <ImporDialog open={impor} onClose={() => setImpor(false)} judul={`Impor template COA ${JENIS_LABEL[jenis]}`}
        keterangan="Kolom: Kode, Nama, Subtype, Saldo normal (debit/kredit), Kelompok, Kondisi, Catatan. Unduh versi yang ada sebagai contoh format."
        fields={[{ id: 'catatan', label: 'Catatan perubahan', placeholder: 'mis. Revisi akuntan Okt 2026' }]}
        onKirim={async (file, n) => { await imporDraftCOA(file, jenis, n.catatan ?? ''); setImpor(false); setInfo('Draft dari Excel tersimpan.'); await muat() }} />
      <Dialog open={akunBaru !== null} onOpenChange={(o) => !o && setAkunBaru(null)}>
        <DialogContent>
          {akunBaru && tampil && (
            <form className="grid gap-3" onSubmit={(e) => { e.preventDefault(); void simpanDraft([...tampil.akun, akunBaru].sort((x, y) => x.kode.localeCompare(y.kode)), `Akun ${akunBaru.kode} ditambahkan ke draft.`).then(() => setAkunBaru(null)) }}>
              <DialogHeader><DialogTitle>Tambah akun ke draft</DialogTitle><DialogDescription>Berlaku untuk klien setelah draft diterbitkan.</DialogDescription></DialogHeader>
              <div className="grid gap-3 sm:grid-cols-2">
                <div className="grid gap-1.5"><Label htmlFor="ak-kode">Kode</Label><Input id="ak-kode" required pattern="[0-9]{4}" placeholder="6119" value={akunBaru.kode} onChange={(e) => setAkunBaru({ ...akunBaru, kode: e.target.value })} /></div>
                <div className="grid gap-1.5"><Label htmlFor="ak-nama">Nama</Label><Input id="ak-nama" required value={akunBaru.nama} onChange={(e) => setAkunBaru({ ...akunBaru, nama: e.target.value })} /></div>
                <div className="grid gap-1.5">
                  <Label htmlFor="ak-kel">Kelompok</Label>
                  <Select value={akunBaru.kelompok} onValueChange={(v) => setAkunBaru({ ...akunBaru, kelompok: v })}>
                    <SelectTrigger id="ak-kel" className="w-full"><SelectValue /></SelectTrigger>
                    <SelectContent>{KELOMPOK.map((x) => <SelectItem key={x} value={x}>{x}</SelectItem>)}</SelectContent>
                  </Select>
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="ak-normal">Saldo normal</Label>
                  <Select value={akunBaru.normal} onValueChange={(v) => setAkunBaru({ ...akunBaru, normal: v as Account['normal'] })}>
                    <SelectTrigger id="ak-normal" className="w-full"><SelectValue /></SelectTrigger>
                    <SelectContent><SelectItem value="debit">Debit</SelectItem><SelectItem value="kredit">Kredit</SelectItem></SelectContent>
                  </Select>
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="ak-kon">Muncul bila</Label>
                  <Select value={akunBaru.kondisi} onValueChange={(v) => setAkunBaru({ ...akunBaru, kondisi: v as Account['kondisi'] })}>
                    <SelectTrigger id="ak-kon" className="w-full"><SelectValue /></SelectTrigger>
                    <SelectContent>{KONDISI.map(([v, l]) => <SelectItem key={v} value={v}>{l}</SelectItem>)}</SelectContent>
                  </Select>
                </div>
                <div className="grid gap-1.5"><Label htmlFor="ak-sub">Subtype</Label><Input id="ak-sub" required value={akunBaru.subtype} onChange={(e) => setAkunBaru({ ...akunBaru, subtype: e.target.value })} /></div>
              </div>
              <DialogFooter>
                <Button type="button" variant="outline" onClick={() => setAkunBaru(null)}>Batal</Button>
                <Button type="submit" disabled={busy}>Tambah</Button>
              </DialogFooter>
            </form>
          )}
        </DialogContent>
      </Dialog>
    </>
  )
}

export function BenchmarkView() {
  const [versi, setVersi] = useState<BenchmarkVersi[]>([])
  const [klien, setKlien] = useState<Record<string, number>>({})
  const [pilih, setPilih] = useState<number | null>(null)
  const [q, setQ] = useState('')
  const [impor, setImpor] = useState(false)
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [busy, setBusy] = useState(false)
  const muat = useCallback(() => getKatalogBenchmark().then((r) => { setVersi(r.versi); setKlien(r.klien_per_klu) }, (e: Error) => setError(e.message)), [])
  useEffect(() => { void muat() }, [muat])
  const aktif = versi.find((v) => v.status === 'aktif')
  const tampil = versi.find((v) => v.id === pilih) ?? aktif
  const baris = (tampil?.baris ?? []).filter((b) => (b.kode + ' ' + b.nama).toLowerCase().includes(q.trim().toLowerCase()))

  async function aktifkan(v: BenchmarkVersi) {
    setBusy(true)
    setError('')
    setInfo('')
    try {
      await aktifkanBenchmark(v.id)
      setInfo(`${v.nama} sekarang dipakai untuk semua perbandingan NPM & CTTOR klien.`)
      await muat()
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <>
      <PageHeader title="Benchmark NPM Nasional" subtitle="Rata-rata nasional margin laba (NPM) dan beban pajak (CTTOR) per KLU. Versi aktif dipakai untuk menilai wajar tidaknya laba klien.">
        <Button variant="outline" className="h-10 rounded-full" onClick={() => setImpor(true)}><Upload /> Impor versi baru</Button>
      </PageHeader>
      <Pesan error={error} info={info} />
      <div className="card" style={{ padding: 20 }}>
        <div className="chart-title">Versi data</div>
        <div className="table-scroll">
          <table className="data-table" style={{ minWidth: 720 }}>
            <thead><tr><th>Nama</th><th>Status</th><th style={{ textAlign: 'right' }}>KLU</th><th>Sumber</th><th>Oleh</th><th style={{ textAlign: 'right' }}>Aksi</th></tr></thead>
            <tbody>
              {[...versi].reverse().map((v) => (
                <tr key={v.id} onClick={() => setPilih(v.id)} style={{ cursor: 'pointer', background: v.id === tampil?.id ? 'var(--bg-card-hover)' : undefined }}>
                  <td className="font-semibold">{v.nama}</td>
                  <td><Status s={v.status} /></td>
                  <td className="tabular" style={{ textAlign: 'right' }}>{v.baris.length}</td>
                  <td className="text-sm text-muted-foreground">{v.sumber}</td>
                  <td className="text-sm">{v.oleh || 'bawaan'}{v.oleh && ` · ${formatTanggal(v.waktu)}`}</td>
                  <td style={{ textAlign: 'right', whiteSpace: 'nowrap' }} onClick={(e) => e.stopPropagation()}>
                    <a className="inline-flex" href={unduhBenchmarkUrl(v.id)} aria-label={`Unduh ${v.nama}`}><Button variant="ghost" size="icon-sm" tabIndex={-1}><Download /></Button></a>
                    {v.status === 'draft' && <Button size="sm" disabled={busy} onClick={() => void aktifkan(v)}>Aktifkan</Button>}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
      {tampil && (
        <div className="card" style={{ padding: 20 }}>
          <div className="flex flex-wrap items-center gap-3" style={{ marginBottom: 8 }}>
            <div className="chart-title">{tampil.nama}</div>
            <Input aria-label="Cari KLU" placeholder="Cari KLU atau nama" value={q} onChange={(e) => setQ(e.target.value)} style={{ maxWidth: 260, marginLeft: 'auto' }} />
          </div>
          <div className="table-scroll" style={{ maxHeight: 520, overflowY: 'auto' }}>
            <table className="data-table tabular" style={{ minWidth: 640 }}>
              <thead><tr><th>KLU</th><th>Kelompok usaha</th><th style={{ textAlign: 'right' }}>NPM</th><th style={{ textAlign: 'right' }}>CTTOR</th><th style={{ textAlign: 'right' }}>Klien</th></tr></thead>
              <tbody>
                {baris.map((b) => (
                  <tr key={b.kode}>
                    <td>{b.kode}</td><td style={{ whiteSpace: 'normal' }}>{b.nama}</td>
                    <td style={{ textAlign: 'right', color: b.npm_bps < 0 ? 'var(--accent-red)' : undefined }}>{pct(b.npm_bps)}</td>
                    <td style={{ textAlign: 'right' }}>{pct(b.cttor_bps)}</td>
                    <td style={{ textAlign: 'right' }}>{klien[b.kode] ?? <span className="text-muted-foreground">0</span>}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
      <ImporDialog open={impor} onClose={() => setImpor(false)} judul="Impor benchmark NPM nasional"
        keterangan="Kolom: KLU, Nama, NPM (%), CTTOR (%). Versi baru disimpan sebagai draft; aktifkan setelah diperiksa. KLU yang dipakai klien wajib tetap ada."
        fields={[{ id: 'nama', label: 'Nama versi', placeholder: 'mis. Buku sektoral DJP 2024 · SPT 2023', wajib: true }, { id: 'sumber', label: 'Sumber (tampil di laporan klien)', placeholder: 'Opsional' }]}
        onKirim={async (file, n) => { await imporBenchmark(file, n.nama ?? '', n.sumber ?? ''); setImpor(false); setInfo('Versi baru tersimpan sebagai draft.'); await muat() }} />
    </>
  )
}
