import { FileSpreadsheet, Landmark } from 'lucide-react'
import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { commitMutasi, previewMutasi, type JenisInfo, type Keyakinan, type MutasiHasil, type MutasiRow } from '@/lib/api'
import { formatRp } from '@/lib/format'

const WARNA: Record<Keyakinan, string> = { tinggi: 'pill-green', sedang: 'pill-amber', rendah: 'pill-red', ai: 'pill-brand' }
const LABEL: Record<Keyakinan, string> = { tinggi: 'Yakin', sedang: 'Perkiraan', rendah: 'Periksa', ai: 'Saran AI' }

/** Ringkasan per jenis dihitung ulang di sini supaya ikut berubah saat jenisnya dikoreksi. */
function rekap(rows: MutasiRow[]) {
  const per = new Map<string, { jenis: string; arah: string; jumlah: number; total: number }>()
  for (const r of rows) {
    if (!r.pilih) continue
    const g = per.get(r.jenis) ?? { jenis: r.jenis, arah: r.arah, jumlah: 0, total: 0 }
    g.jumlah++
    g.total += r.nominal
    per.set(r.jenis, g)
  }
  return [...per.values()].sort((a, b) => b.total - a.total)
}

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
  onImported: () => void
  jenis: JenisInfo[]
}

/**
 * Impor mutasi rekening bank: unggah → jenis tiap baris ditebak → DIPERIKSA → baru dibukukan.
 *
 * Tebakan dibuat dari kata kunci di keterangan bank, jadi ia bisa salah. Karena itu yang
 * dikirim saat menyimpan adalah baris hasil koreksi di layar ini, bukan hasil baca ulang
 * berkasnya — kalau tidak, koreksi pengguna hilang dan tebakan mesin yang terbukukan.
 */
export function MutasiDialog({ open, onOpenChange, onImported, jenis }: Props) {
  const [hasil, setHasil] = useState<MutasiHasil | null>(null)
  const [rows, setRows] = useState<MutasiRow[]>([])
  const [nama, setNama] = useState('')
  // Lazy: tahun berjalan dihitung sekali saat dialog dibuat, bukan tiap render.
  const [tahun, setTahun] = useState(() => String(new Date().getFullYear()))
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [berkas, setBerkas] = useState<File | null>(null)
  // Kata sandi PDF e-statement: hanya di memori dialog ini, dikosongkan saat dialog ditutup.
  const [sandi, setSandi] = useState('')
  const pdf = berkas?.name.toLowerCase().endsWith('.pdf') ?? false
  const perluSandi = pdf && /kata sandi/i.test(error)
  // PDF terkunci & sandi belum diisi: ajakan mengisi sandi, bukan galat.
  const mintaSandi = pdf && /dikunci kata sandi/i.test(error)

  function reset(next: boolean) {
    if (!next) {
      setHasil(null)
      setRows([])
      setNama('')
      setBerkas(null)
      setSandi('')
      setError('')
    }
    onOpenChange(next)
  }

  async function pilihBerkas(f: File | undefined | null, th = tahun) {
    if (!f) return
    setBerkas(f)
    setNama(f.name)
    setHasil(null)
    setError('')
    setBusy(true)
    try {
      const h = await previewMutasi(f, Number(th) || undefined, sandi)
      setHasil(h)
      setRows(h.rows ?? [])
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setBusy(false)
    }
  }

  async function simpan() {
    setBusy(true)
    setError('')
    try {
      await commitMutasi(rows)
      onImported()
      reset(false)
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setBusy(false)
    }
  }

  const ubah = (i: number, patch: Partial<MutasiRow>) => setRows(rows.map((r, j) => (j === i ? { ...r, ...patch } : r)))
  const terpilih = rows.filter((r) => r.pilih)
  const perluDiperiksa = terpilih.filter((r) => r.keyakinan === 'rendah').length
  const grup = rekap(rows)
  const errors = hasil?.errors ?? []

  return (
    <Dialog open={open} onOpenChange={reset}>
      <DialogContent className="sm:max-w-5xl">
        <DialogHeader>
          <DialogTitle className="inline-flex items-center gap-2"><Landmark size={18} aria-hidden /> Impor dari mutasi bank</DialogTitle>
          <DialogDescription>
            Unggah mutasi rekening: <strong>PDF e-statement / rekening koran</strong>, atau .xlsx / .csv dari m-banking. Jenis transaksi ditebak dari keterangan bank —
            periksa dan betulkan dulu di bawah, karena salah kelompok ikut mengubah omzet dan pajaknya.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4">
          <div className="flex flex-wrap items-end gap-3">
            <label className="btn-cta inline-flex cursor-pointer items-center gap-1.5">
              <FileSpreadsheet size={16} aria-hidden /> {nama ? 'Ganti berkas' : 'Pilih berkas mutasi'}
              <input
                type="file" className="sr-only"
                accept=".pdf,application/pdf,.xlsx,.csv,text/csv,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
                onChange={(e) => { void pilihBerkas(e.target.files?.[0]); e.target.value = '' }}
              />
            </label>
            {/* Lebar dikunci ke lebar input: tanpa itu keterangan di bawahnya melebar
                melewati kolomnya sendiri dan menabrak nama berkas di sebelahnya. */}
            <label className="grid gap-1" style={{ width: 200 }}>
              <span className="kpi-label">Tahun mutasi</span>
              <input
                className="input-field tabular" type="number" min={2000} max={2100}
                value={tahun} onChange={(e) => setTahun(e.target.value)}
                // Ganti tahun setelah berkas dipilih: baca ulang supaya tanggalnya ikut berubah.
                onBlur={() => { if (berkas && tahun.length === 4) void pilihBerkas(berkas) }}
              />
              <span style={{ fontSize: 11, lineHeight: 1.3, color: 'var(--text-muted)' }}>
                hanya untuk tanggal tanpa tahun (mis. 15/01). Mutasi Des–Jan: isi tahun Januari-nya.
              </span>
            </label>
            {pdf && (
              <form className="grid gap-1" style={{ width: 220 }} onSubmit={(e) => { e.preventDefault(); if (berkas) void pilihBerkas(berkas) }}>
                <label className="kpi-label" htmlFor="mutasi-sandi">Kata sandi PDF (bila dikunci)</label>
                <div className="flex gap-1.5">
                  <input id="mutasi-sandi" className="input-field" type="password" autoComplete="off" value={sandi}
                    autoFocus={perluSandi} onChange={(e) => setSandi(e.target.value)} />
                  <Button type="submit" size="sm" variant="outline" disabled={!berkas || busy || !sandi}>Buka</Button>
                </div>
                <span style={{ fontSize: 11, lineHeight: 1.3, color: 'var(--text-muted)' }}>
                  hanya untuk membuka berkas ini; tidak disimpan.
                </span>
              </form>
            )}
            {nama && <span className="text-sm text-muted-foreground" style={{ paddingBottom: 2 }}>{nama}</span>}
          </div>

          {busy && !hasil && <p className="text-sm text-muted-foreground">Membaca mutasi…</p>}
          {error && (mintaSandi
            ? <div className="note-box" role="status">{error}</div>
            : <p className="text-sm text-destructive" role="alert">{error}</p>)}

          {hasil && (
            <>
              <div className="flex flex-wrap items-center gap-2">
                <span className="pill-status pill-green">Masuk {formatRp(hasil.total_masuk)}</span>
                <span className="pill-status pill-red">Keluar {formatRp(hasil.total_keluar)}</span>
                {hasil.duplikat > 0 && <span className="pill-status pill-amber">{hasil.duplikat} diduga sudah pernah dicatat</span>}
                {!!hasil.dibantu_ai && <span className="pill-status pill-brand">{hasil.dibantu_ai} baris dikelompokkan AI</span>}
                {hasil.format && <span className="text-sm text-muted-foreground">kolom terbaca: {hasil.format}</span>}
              </div>

              {!!hasil.per_bulan?.length && (
                <div className="flex flex-wrap items-center gap-2 text-sm">
                  <span className="text-muted-foreground">Masuk ke bulan:</span>
                  {hasil.per_bulan.map((b) => <span key={b.bulan} className="pill-status pill-brand">{b.label}: {b.jumlah} baris</span>)}
                </div>
              )}
              {(hasil.peringatan ?? []).map((p) => <div key={p} className="note-box warn">{p}</div>)}

              {!!hasil.dibantu_ai && (
                <div className="note-box">
                  <strong>{hasil.dibantu_ai} baris tidak dikenali kata kunci, jadi jenisnya disarankan AI</strong> (tanda <em>Saran AI</em>).
                  Baca alasannya di setiap baris dan ganti bila kurang tepat. Nomor rekening disamarkan sebelum dikirim ke AI.
                </div>
              )}
              {hasil.ai_pesan && <p className="text-sm text-muted-foreground">{hasil.ai_pesan}</p>}
              {perluDiperiksa > 0 && (
                <div className="note-box warn">
                  <strong>{perluDiperiksa} baris masih perlu diperiksa.</strong> Keterangannya tidak memuat penanda
                  yang dikenali, jadi jenisnya cuma diisi default menurut arah uangnya. Uang masuk yang sebenarnya
                  setoran modal atau pinjaman akan terhitung sebagai omzet — dan omzet itu dasar PPh final.
                </div>
              )}

              {grup.length > 0 && (
                <div>
                  <div className="kpi-label" style={{ marginBottom: 4 }}>Pengelompokan yang akan disimpan</div>
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
                    {grup.map((g) => (
                      <span key={g.jenis} className="pill-status pill-status-muted" style={{ fontSize: 11 }}>
                        {g.jenis}: {g.jumlah} baris · {formatRp(g.total)}
                      </span>
                    ))}
                  </div>
                </div>
              )}

              <div className="table-scroll" style={{ maxHeight: 'min(360px, 45dvh)', overflowY: 'auto' }}>
                <table className="data-table tabular" style={{ minWidth: 940, marginTop: 0 }}>
                  <thead>
                    <tr>
                      <th style={{ width: 36 }}>
                        <input
                          type="checkbox" aria-label="Pilih semua baris"
                          checked={rows.length > 0 && rows.every((r) => r.pilih)}
                          onChange={(e) => setRows(rows.map((r) => ({ ...r, pilih: e.target.checked })))}
                        />
                      </th>
                      <th>Tanggal</th><th>Keterangan dari bank</th>
                      <th style={{ textAlign: 'right' }}>Nominal</th><th>Arah</th>
                      <th style={{ minWidth: 190 }}>Jenis transaksi</th><th>Tebakan</th>
                    </tr>
                  </thead>
                  <tbody>
                    {rows.length === 0 && (
                      <tr><td colSpan={7} style={{ color: 'var(--text-muted)' }}>Tidak ada baris mutasi yang terbaca.</td></tr>
                    )}
                    {rows.map((r, i) => {
                      const pilihan = jenis.filter((j) => j.arah === r.arah)
                      return (
                        <tr key={`${r.baris}-${i}`} style={{ opacity: r.pilih ? 1 : 0.45 }}>
                          <td>
                            <input
                              type="checkbox" checked={r.pilih} aria-label={`Bukukan baris ${r.baris}`}
                              onChange={(e) => ubah(i, { pilih: e.target.checked })}
                            />
                          </td>
                          <td style={{ whiteSpace: 'nowrap' }}>{r.tanggal}</td>
                          <td style={{ maxWidth: 300 }}>
                            {r.keterangan || <span style={{ color: 'var(--text-muted)' }}>(tanpa keterangan)</span>}
                            {r.duplikat && (
                              <span className="pill-status pill-amber" style={{ fontSize: 10, marginLeft: 6 }}>sudah pernah dicatat?</span>
                            )}
                          </td>
                          <td style={{ textAlign: 'right', fontWeight: 600 }}>{formatRp(r.nominal)}</td>
                          <td style={{ color: r.arah === 'masuk' ? 'var(--text-profit)' : 'var(--text-danger)' }}>{r.arah}</td>
                          <td>
                            <select
                              className="input-field" value={r.jenis} aria-label={`Jenis untuk baris ${r.baris}`}
                              onChange={(e) => ubah(i, { jenis: e.target.value, keyakinan: 'tinggi', alasan: 'dipilih sendiri' })}
                            >
                              {pilihan.map((j) => <option key={j.nama} value={j.nama}>{j.nama}</option>)}
                            </select>
                          </td>
                          <td>
                            <span className={'pill-status ' + WARNA[r.keyakinan]} style={{ fontSize: 10 }} title={r.alasan}>
                              {LABEL[r.keyakinan]}
                            </span>
                            <div style={{ fontSize: 10.5, color: 'var(--text-muted)', maxWidth: 220 }}>{r.alasan}</div>
                          </td>
                        </tr>
                      )
                    })}
                  </tbody>
                </table>
              </div>

              {errors.length > 0 && (
                <div>
                  <div className="kpi-label" style={{ marginBottom: 4 }}>Baris yang dilewati</div>
                  <ul className="max-h-32 overflow-auto rounded-md border p-2 text-sm">
                    {errors.map((e, i) => (
                      <li key={i} className="py-0.5">
                        <span className="tabular text-muted-foreground">{e.baris ? `Baris ${e.baris}` : 'Berkas'}:</span> {e.pesan}
                      </li>
                    ))}
                  </ul>
                </div>
              )}
            </>
          )}
        </div>

        <DialogFooter className="sticky -bottom-5 z-10 bg-popover">
          <Button type="button" variant="outline" onClick={() => reset(false)}>Batal</Button>
          <Button type="button" disabled={busy || terpilih.length === 0} onClick={() => void simpan()}>
            {busy && hasil ? 'Menyimpan…' : `Bukukan ${terpilih.length} transaksi`}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
