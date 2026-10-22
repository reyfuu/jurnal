import { useCallback, useEffect, useMemo, useState, type FormEvent } from 'react'
import { Plus, Trash2, Upload } from 'lucide-react'
import { PageHeader } from '@/components/PageHeader'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import {
  KATEGORI_ARTIKEL, KATEGORI_SUMBER, type KategoriSumber, cariPengetahuan, hapusPengetahuan, listArtikel, listPengetahuan, pakaiPengetahuan, simpanArtikel, tulisPengetahuan,
  unggahPengetahuan, verifikasiPengetahuan, type Artikel, type ArtikelInput, type Potongan, type SumberPengetahuan,
} from '@/lib/api'
import { formatTanggal } from '@/lib/format'

const textareaClass = 'w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm outline-none placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 dark:bg-input/30'
const refArtikel = (id: number) => `A-${String(id).padStart(3, '0')}`

function Pesan({ error, info }: { error: string; info: string }) {
  if (error) return <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start', whiteSpace: 'normal' }}>{error}</div>
  if (info) return <div className="pill-status pill-green" role="status" style={{ alignSelf: 'flex-start' }}>{info}</div>
  return null
}

const KOSONG: ArtikelInput = { judul: '', kategori: 'PPh Final UMKM', dasar_hukum: '', berlaku_sejak: '', tag: [], isi: '', tampil_klien: true }

export function ArtikelView({ bolehTerbit }: { bolehTerbit: boolean }) {
  const [rows, setRows] = useState<Artikel[]>([])
  const [kat, setKat] = useState('Semua')
  const [q, setQ] = useState('')
  const [edit, setEdit] = useState<{ id: number | null; a: ArtikelInput; tag: string; terbit: boolean } | null>(null)
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [busy, setBusy] = useState(false)
  const muat = useCallback(() => listArtikel().then(setRows, (e: Error) => setError(e.message)), [])
  useEffect(() => { void muat() }, [muat])
  const tampil = rows.filter((a) => (kat === 'Semua' || a.kategori === kat)
    && [refArtikel(a.id), a.judul, a.isi, a.dasar_hukum, ...a.tag].join(' ').toLowerCase().includes(q.trim().toLowerCase()))
    .sort((a, b) => b.id - a.id)

  async function simpan(terbit: boolean) {
    if (!edit) return
    setBusy(true)
    setError('')
    try {
      const a = await simpanArtikel(edit.id, { ...edit.a, tag: edit.tag.split(',').map((t) => t.trim()).filter(Boolean) }, terbit)
      setInfo(`${refArtikel(a.id)} ${terbit ? 'terbit dan diindeks untuk AI' : 'tersimpan sebagai draft'}.`)
      setEdit(null)
      await muat()
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setBusy(false)
    }
  }
  const set = (p: Partial<ArtikelInput>) => setEdit((e) => e && { ...e, a: { ...e.a, ...p } })

  return (
    <>
      <PageHeader title="Artikel Pajak" subtitle="Diindeks per kategori, dasar hukum, dan tag. Artikel terbit dipakai AI; yang ditandai juga tampil di menu Artikel Pajak klien.">
        <button className="btn-cta inline-flex items-center gap-1.5" onClick={() => { setError(''); setEdit({ id: null, a: KOSONG, tag: '', terbit: false }) }}><Plus size={16} /> Tulis artikel</button>
      </PageHeader>
      <Pesan error={!edit ? error : ''} info={info} />
      <div className="flex flex-wrap items-center gap-3">
        <div className="tabbar" role="tablist" aria-label="Kategori">
          {['Semua', ...KATEGORI_ARTIKEL].map((k) => <button key={k} role="tab" className="tab-btn" aria-selected={kat === k} onClick={() => setKat(k)}>{k}</button>)}
        </div>
        <Input aria-label="Cari artikel" placeholder="Cari judul, PP/PMK, tag, isi" value={q} onChange={(e) => setQ(e.target.value)} style={{ maxWidth: 260, marginLeft: 'auto' }} />
      </div>
      <div className="card" style={{ padding: 8 }}>
        {tampil.length === 0 && <p className="text-sm text-muted-foreground" style={{ padding: 16 }}>Belum ada artikel{q || kat !== 'Semua' ? ' yang cocok' : ''}.</p>}
        <ul>
          {tampil.map((a) => (
            <li key={a.id} className="list-row" style={{ cursor: 'pointer', padding: '12px', gap: 12, alignItems: 'flex-start' }}
              onClick={() => { setError(''); setEdit({ id: a.id, a, tag: a.tag.join(', '), terbit: a.status === 'terbit' }) }}>
              <div style={{ minWidth: 0, flex: 1 }}>
                <span className="tabular text-xs text-muted-foreground">{refArtikel(a.id)}</span> <span className="font-semibold">{a.judul}</span>
                <div className="mt-1 flex flex-wrap gap-1.5 text-xs">
                  {a.dasar_hukum && <span className="pill-status pill-amber">{a.dasar_hukum}</span>}
                  <span className="pill-status pill-status-muted">{a.kategori}</span>
                  {a.tag.map((t) => <span key={t} className="text-muted-foreground">#{t}</span>)}
                </div>
              </div>
              <div className="flex flex-col items-end gap-1 text-xs">
                <span className={'pill-status ' + (a.status === 'terbit' ? 'pill-green' : 'pill-amber')}>{a.status === 'terbit' ? 'Terbit' : 'Draft'}</span>
                <span className="text-muted-foreground">{a.tampil_klien ? 'tampil ke klien' : 'internal'} · {formatTanggal(a.waktu)}</span>
              </div>
            </li>
          ))}
        </ul>
      </div>
      <Dialog open={edit !== null} onOpenChange={(o) => !o && setEdit(null)}>
        <DialogContent className="sm:max-w-2xl">
          {edit && (
            <form className="grid gap-3" onSubmit={(e: FormEvent) => { e.preventDefault(); void simpan(false) }}>
              <DialogHeader>
                <DialogTitle>{edit.id ? `${refArtikel(edit.id)} · Ubah artikel` : 'Tulis artikel'}</DialogTitle>
                <DialogDescription>{edit.terbit && !bolehTerbit
                  ? 'Artikel ini sudah terbit; hanya yang punya izin verifikasi yang bisa mengubahnya.'
                  : 'Tulis dengan bahasa pemilik usaha. Artikel terbit otomatis diindeks untuk AI.'}</DialogDescription>
              </DialogHeader>
              <div className="grid gap-1.5"><Label htmlFor="ar-j">Judul</Label><Input id="ar-j" required value={edit.a.judul} onChange={(e) => set({ judul: e.target.value })} /></div>
              <div className="grid gap-3 sm:grid-cols-3">
                <div className="grid gap-1.5">
                  <Label htmlFor="ar-k">Kategori</Label>
                  <Select value={edit.a.kategori} onValueChange={(v) => set({ kategori: v })}>
                    <SelectTrigger id="ar-k" className="w-full"><SelectValue /></SelectTrigger>
                    <SelectContent>{KATEGORI_ARTIKEL.map((k) => <SelectItem key={k} value={k}>{k}</SelectItem>)}</SelectContent>
                  </Select>
                </div>
                <div className="grid gap-1.5"><Label htmlFor="ar-h">Dasar hukum</Label><Input id="ar-h" placeholder="PP 55/2022" value={edit.a.dasar_hukum ?? ''} onChange={(e) => set({ dasar_hukum: e.target.value })} /></div>
                <div className="grid gap-1.5"><Label htmlFor="ar-b">Berlaku sejak</Label><Input id="ar-b" placeholder="1 Jan 2022" value={edit.a.berlaku_sejak ?? ''} onChange={(e) => set({ berlaku_sejak: e.target.value })} /></div>
              </div>
              <div className="grid gap-1.5"><Label htmlFor="ar-t">Tag (pisah koma)</Label><Input id="ar-t" placeholder="omzet 4,8 M, OP, batas waktu" value={edit.tag} onChange={(e) => setEdit({ ...edit, tag: e.target.value })} /></div>
              <div className="grid gap-1.5"><Label htmlFor="ar-i">Isi</Label><textarea id="ar-i" required rows={10} className={textareaClass} value={edit.a.isi} onChange={(e) => set({ isi: e.target.value })} /></div>
              <div className="flex items-center justify-between"><Label htmlFor="ar-tk">Tampilkan ke klien di Artikel Pajak</Label><Switch id="ar-tk" checked={edit.a.tampil_klien} onCheckedChange={(v) => set({ tampil_klien: v })} /></div>
              {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
              <DialogFooter>
                <Button type="button" variant="outline" onClick={() => setEdit(null)}>Batal</Button>
                {!(edit.terbit && !bolehTerbit) && <Button type="submit" variant="outline" disabled={busy}>{edit.terbit ? 'Tarik jadi draft' : 'Simpan draft'}</Button>}
                {bolehTerbit && <Button type="button" disabled={busy} onClick={() => void simpan(true)}>Verifikasi &amp; terbitkan</Button>}
              </DialogFooter>
            </form>
          )}
        </DialogContent>
      </Dialog>
    </>
  )
}

function KategoriPilih({ v, set }: { v: KategoriSumber; set: (k: KategoriSumber) => void }) {
  return (
    <fieldset className="grid gap-1.5">
      <legend className="text-sm font-medium">Kategori</legend>
      <div className="grid gap-2" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(100%, 200px), 1fr))' }}>
        {KATEGORI_SUMBER.map(([id, label, ket]) => (
          <label key={id} className="flex cursor-pointer items-start gap-2 rounded-md border p-2 text-sm" style={{ borderColor: v === id ? 'var(--text-brand)' : 'var(--border-subtle)' }}>
            <input type="radio" name="kb-kategori" className="mt-1" checked={v === id} onChange={() => set(id)} />
            <span><span className="font-semibold">{label}</span>{ket && <span className="block text-xs text-muted-foreground">{ket}</span>}</span>
          </label>
        ))}
      </div>
    </fieldset>
  )
}

export function BasisPengetahuan() {
  const [data, setData] = useState<{ sumber: SumberPengetahuan[]; artikel_terbit: number; potongan_dipakai: number } | null>(null)
  const [tulis, setTulis] = useState<{ judul: string; isi: string; topik: string; kategori: KategoriSumber } | null>(null)
  const [unggah, setUnggah] = useState<{ file: File | null; judul: string; topik: string; kategori: KategoriSumber } | null>(null)
  const [q, setQ] = useState('')
  const [hasil, setHasil] = useState<Potongan[] | null>(null)
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [busy, setBusy] = useState(false)
  const muat = useCallback(() => listPengetahuan().then(setData, (e: Error) => setError(e.message)), [])
  useEffect(() => { void muat() }, [muat])
  const menunggu = useMemo(() => data?.sumber.filter((s) => s.status === 'menunggu').length ?? 0, [data])

  async function jalankan(f: () => Promise<unknown>, pesan: string, tutup?: () => void) {
    setBusy(true)
    setError('')
    setInfo('')
    try {
      await f()
      setInfo(pesan)
      tutup?.()
      await muat()
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <>
      <PageHeader title="Basis Pengetahuan AI" subtitle="Bahan yang dibaca AI saat menilai klien. Hanya artikel terbit dan sumber yang sudah diverifikasi yang dipakai.">
        <div className="flex flex-wrap gap-2">
          <Button variant="outline" className="h-10 rounded-full" onClick={() => setTulis({ judul: '', isi: '', topik: '', kategori: 'praktik' })}><Plus /> Tulis manual</Button>
          <button className="btn-cta inline-flex items-center gap-1.5" onClick={() => setUnggah({ file: null, judul: '', topik: '', kategori: 'peraturan' })}><Upload size={16} /> Impor Excel / PDF</button>
        </div>
      </PageHeader>
      <Pesan error={!tulis && !unggah ? error : ''} info={info} />
      {data && (
        <div className="grid gap-3" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(170px, 1fr))' }}>
          {[
            ['Sumber unggahan', data.sumber.length, `${data.artikel_terbit} artikel terbit ikut dipakai`],
            ['Potongan dipakai AI', data.potongan_dipakai, 'dicari setiap AI menilai klien'],
            ['Menunggu verifikasi', menunggu, 'belum dipakai AI'],
          ].map(([l, v, s]) => (
            <div key={l} className="card" style={{ padding: 18 }}>
              <div className="kpi-label">{l}</div>
              <div className="kpi-value tabular" style={l === 'Menunggu verifikasi' && menunggu ? { color: 'var(--accent-amber)' } : undefined}>{v}</div>
              <div className="text-xs text-muted-foreground" style={{ marginTop: 4 }}>{s}</div>
            </div>
          ))}
        </div>
      )}
      <form className="card" style={{ padding: 20, display: 'grid', gridTemplateColumns: 'minmax(0, 1fr)', gap: 10 }} onSubmit={(e) => { e.preventDefault(); void cariPengetahuan(q).then(setHasil, (er: Error) => setError(er.message)) }}>
        <div className="chart-title">Uji pertanyaan</div>
        <div className="chart-subtitle">Lihat sumber mana yang ditemukan sebelum AI menjawab.</div>
        <div className="flex flex-wrap gap-2">
          <Input aria-label="Pertanyaan uji" placeholder="mis. warung kopi omzet 3 miliar masih boleh pakai PPh final?" value={q} onChange={(e) => setQ(e.target.value)} style={{ flex: 1, minWidth: 220 }} />
          <Button type="submit" disabled={!q.trim()}>Cari sumber</Button>
        </div>
        {hasil && (hasil.length === 0
          ? <p className="text-sm text-muted-foreground">Tidak ada sumber yang cocok. AI akan menilai tanpa rujukan untuk topik ini.</p>
          : <ul className="grid gap-2">{hasil.map((p, i) => (
            <li key={i} className="rounded-lg border border-border" style={{ padding: '8px 12px' }}>
              <div className="text-sm"><span className="pill-status pill-status-muted tabular">{p.ref}</span> <strong>{p.judul}</strong> <span className="tabular text-xs text-muted-foreground">skor {p.skor}</span></div>
              <p className="mt-1 text-xs text-muted-foreground" style={{ whiteSpace: 'pre-wrap' }}>{p.teks.slice(0, 280)}{p.teks.length > 280 ? '…' : ''}</p>
            </li>))}</ul>)}
      </form>
      <div className="card" style={{ padding: 20 }}>
        <div className="chart-title">Sumber</div>
        <div className="table-scroll">
          <table className="data-table" style={{ minWidth: 780 }}>
            <thead><tr><th>Sumber</th><th>Kategori</th><th>Topik</th><th style={{ textAlign: 'right' }}>Potongan</th><th>Status</th><th>Dipakai AI</th><th /></tr></thead>
            <tbody>
              {data?.sumber.map((s) => (
                <tr key={s.id}>
                  <td>
                    <span className="tabular text-xs text-muted-foreground">KB-{String(s.id).padStart(3, '0')}</span> <span className="font-semibold">{s.judul}</span>
                    <div className="text-xs text-muted-foreground" title={s.cuplikan}>{s.cuplikan.slice(0, 90)}{s.cuplikan.length > 90 ? '…' : ''}</div>
                  </td>
                  <td>
                    <span className="pill-status pill-status-muted">{KATEGORI_SUMBER.find((k) => k[0] === (s.kategori ?? 'umum'))?.[1]}</span>
                    <div className="text-xs text-muted-foreground" style={{ marginTop: 2 }}>{s.jenis.toUpperCase()}</div>
                  </td>
                  <td className="text-xs">{s.topik.join(', ') || '—'}</td>
                  <td className="tabular" style={{ textAlign: 'right' }}>{s.potongan}</td>
                  <td>{s.status === 'menunggu'
                    ? <Button size="sm" disabled={busy} onClick={() => void jalankan(() => verifikasiPengetahuan(s.id), `KB-${String(s.id).padStart(3, '0')} terverifikasi dan dipakai AI.`)}>Verifikasi</Button>
                    : <span className="pill-status pill-green" title={`oleh ${s.diverifikasi}`}>Terverifikasi</span>}</td>
                  <td><Switch aria-label={`Dipakai AI: ${s.judul}`} checked={s.dipakai_ai} disabled={s.status !== 'terverifikasi' || busy}
                    onCheckedChange={(v) => void jalankan(() => pakaiPengetahuan(s.id, v), v ? 'Sumber dipakai AI.' : 'Sumber dilepas dari AI.')} /></td>
                  <td style={{ textAlign: 'right' }}><Button variant="ghost" size="icon-sm" aria-label={`Hapus ${s.judul}`} disabled={busy} onClick={() => void jalankan(() => hapusPengetahuan(s.id), 'Sumber dihapus.')}><Trash2 /></Button></td>
                </tr>
              ))}
              {data?.sumber.length === 0 && <tr><td colSpan={7} className="text-center text-muted-foreground" style={{ padding: 24 }}>Belum ada sumber. Tulis manual, atau impor Excel/PDF.</td></tr>}
            </tbody>
          </table>
        </div>
      </div>
      <Dialog open={tulis !== null} onOpenChange={(o) => !o && setTulis(null)}>
        <DialogContent className="sm:max-w-2xl">
          {tulis && (
            <form className="grid gap-3" onSubmit={(e) => { e.preventDefault(); void jalankan(() => tulisPengetahuan(tulis.judul, tulis.isi, tulis.topik.split(','), tulis.kategori), 'Tersimpan. Menunggu verifikasi sebelum dipakai AI.', () => setTulis(null)) }}>
              <DialogHeader><DialogTitle>Tulis pengetahuan</DialogTitle><DialogDescription>Tidak harus peraturan atau artikel: pengalaman lapangan, kebiasaan di KPP, tips menangani klien, dan pengetahuan bisnis umum juga boleh. Tulis seperti menjelaskan ke rekan konsultan. Dipakai AI setelah diverifikasi.</DialogDescription></DialogHeader>
              <KategoriPilih v={tulis.kategori} set={(kategori) => setTulis({ ...tulis, kategori })} />
              <div className="grid gap-1.5"><Label htmlFor="kb-j">Judul</Label><Input id="kb-j" required value={tulis.judul} onChange={(e) => setTulis({ ...tulis, judul: e.target.value })} /></div>
              <div className="grid gap-1.5"><Label htmlFor="kb-i">Isi</Label><textarea id="kb-i" required rows={8} className={textareaClass} value={tulis.isi} onChange={(e) => setTulis({ ...tulis, isi: e.target.value })} /></div>
              <div className="grid gap-1.5"><Label htmlFor="kb-t">Topik (pisah koma)</Label><Input id="kb-t" placeholder="PPh Final, Pembukuan" value={tulis.topik} onChange={(e) => setTulis({ ...tulis, topik: e.target.value })} /></div>
              {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
              <DialogFooter><Button type="button" variant="outline" onClick={() => setTulis(null)}>Batal</Button><Button type="submit" disabled={busy}>Simpan</Button></DialogFooter>
            </form>
          )}
        </DialogContent>
      </Dialog>
      <Dialog open={unggah !== null} onOpenChange={(o) => !o && setUnggah(null)}>
        <DialogContent>
          {unggah && (
            <form className="grid gap-3" onSubmit={(e) => { e.preventDefault(); if (unggah.file) void jalankan(() => unggahPengetahuan(unggah.file!, unggah.judul, unggah.topik, unggah.kategori), 'Diunggah dan diindeks. Menunggu verifikasi sebelum dipakai AI.', () => setUnggah(null)) }}>
              <DialogHeader>
                <DialogTitle>Impor Excel / PDF</DialogTitle>
                <DialogDescription>Excel: kolom Judul, Isi, Topik, Sumber (satu baris satu potong). PDF: teks diambil per halaman; PDF hasil scan (gambar) belum bisa dibaca.</DialogDescription>
              </DialogHeader>
              <KategoriPilih v={unggah.kategori} set={(kategori) => setUnggah({ ...unggah, kategori })} />
              <div className="grid gap-1.5"><Label htmlFor="kb-f">Berkas .xlsx / .pdf</Label><Input id="kb-f" type="file" accept=".xlsx,.pdf" required onChange={(e) => setUnggah({ ...unggah, file: e.target.files?.[0] ?? null })} /></div>
              <div className="grid gap-1.5"><Label htmlFor="kb-uj">Judul</Label><Input id="kb-uj" required placeholder="mis. PMK 81/2024" value={unggah.judul} onChange={(e) => setUnggah({ ...unggah, judul: e.target.value })} /></div>
              <div className="grid gap-1.5"><Label htmlFor="kb-ut">Topik (pisah koma)</Label><Input id="kb-ut" placeholder="Administrasi, Coretax" value={unggah.topik} onChange={(e) => setUnggah({ ...unggah, topik: e.target.value })} /></div>
              {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
              <DialogFooter><Button type="button" variant="outline" onClick={() => setUnggah(null)}>Batal</Button><Button type="submit" disabled={busy}>{busy ? 'Mengunggah…' : 'Unggah'}</Button></DialogFooter>
            </form>
          )}
        </DialogContent>
      </Dialog>
    </>
  )
}
