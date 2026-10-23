import { FileSpreadsheet, FileText, Plus } from 'lucide-react'
import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { createTransaction, getRekonsiliasi, rekonsiliasiUrl, type RekonBulan, type Rekonsiliasi, type StatusRekon } from '@/lib/api'
import { angka } from '@/lib/angka'
import { formatRp, formatTanggal } from '@/lib/format'
import { ekspor } from '@/lib/garansi'

// Rekonsiliasi bulanan vs tahunan PPh Final (paket Pro): setoran per masa vs PPh terutang
// menurut pembukuan; totalnya kurang / lebih bayar di SPT Tahunan.

const STATUS: Record<StatusRekon, [string, string]> = {
  cocok: ['Cocok', 'pill-green'], kurang: ['Kurang bayar', 'pill-red'], lebih: ['Lebih bayar', 'pill-amber'],
  belum_jatuh_tempo: ['Belum jatuh tempo', 'pill-status-muted'], nihil: ['Nihil', 'pill-status-muted'], berjalan: ['Masa berjalan', 'pill-status-muted'],
}
const hariIni = () => new Date().toLocaleDateString('sv-SE', { timeZone: 'Asia/Jakarta' })

export function RekonsiliasiPajak({ tahunAwal, bisaEkspor, bisaCatat }: { tahunAwal: number; bisaEkspor: boolean; bisaCatat: boolean }) {
  const [tahun, setTahun] = useState(tahunAwal)
  const [r, setR] = useState<Rekonsiliasi | null>(null)
  const [error, setError] = useState('')
  const [setor, setSetor] = useState<RekonBulan | null>(null)
  const muat = useCallback(() => getRekonsiliasi(tahun).then((x) => { setR(x); setError('') }, (e: Error) => { setR(null); setError(e.message) }), [tahun])
  useEffect(() => { void muat() }, [muat])
  const tahunIni = Number(hariIni().slice(0, 4))

  return (
    <div className="card" style={{ padding: 20 }}>
      <div className="mb-3 flex flex-wrap items-center justify-between gap-3">
        <div>
          <div className="chart-title">Rekonsiliasi bulanan vs tahunan</div>
          <p className="text-sm text-muted-foreground">Setoran PPh Final tiap masa dibandingkan dengan pajak terutang menurut pembukuan sekarang. Selisihnya menjadi kurang / lebih bayar di SPT Tahunan.</p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <select className="input-field" aria-label="Tahun pajak" value={tahun} onChange={(e) => setTahun(Number(e.target.value))}>
            {[...new Set([tahunIni, tahunIni - 1, tahunIni - 2, tahunAwal])].sort((a, b) => b - a).map((t) => <option key={t} value={t}>{t}</option>)}
          </select>
          {bisaEkspor && <>
            <Button size="sm" variant="outline" onClick={() => void ekspor(rekonsiliasiUrl(tahun, 'xlsx'))}><FileSpreadsheet /> Excel</Button>
            <Button size="sm" variant="outline" onClick={() => void ekspor(rekonsiliasiUrl(tahun, 'pdf'))}><FileText /> PDF</Button>
          </>}
        </div>
      </div>
      {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
      {r && <>
        {r.peringatan?.map((p) => <p key={p} className="mb-2 text-sm" style={{ color: 'var(--text-warn)' }} role="note">{p}</p>)}
        <div className="kpi-grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))' }}>
          <Ringkas label={`PPh terutang ${r.tahun}`} nilai={r.total_terutang} />
          <Ringkas label="Sudah disetor" nilai={r.total_disetor} />
          <Ringkas label={(r.kurang_bayar > 0 ? 'Kurang bayar' : r.kurang_bayar < 0 ? 'Lebih bayar' : 'Selisih') + (r.selesai ? ' (SPT Tahunan)' : ' s.d. jatuh tempo')} nilai={Math.abs(r.kurang_bayar)}
            warna={r.kurang_bayar !== 0 ? 'var(--text-warn)' : 'var(--accent-green)'} />
          <Ringkas label="Masa perlu dicek" teks={String(r.perlu_dicek)} warna={r.perlu_dicek ? 'var(--text-warn)' : undefined} />
        </div>
        <div className="overflow-x-auto">
          <table className="data-table tabular">
            <thead>
              <tr>
                <th>Masa</th><th>Omzet</th><th>Dasar kena pajak</th><th>PPh terutang</th><th>Disetor</th><th>Selisih</th><th>Status</th>
                {r.pkp && <><th>PPN Keluaran</th><th>PPN disetor</th></>}
                {bisaCatat && <th><span className="sr-only">Aksi</span></th>}
              </tr>
            </thead>
            <tbody>
              {r.bulan.map((b) => {
                const [label, kelas] = STATUS[b.status]
                return (
                  <tr key={b.masa} className={b.status === 'kurang' || b.status === 'lebih' ? 'flag' : undefined}>
                    <td style={{ fontWeight: 600 }}>{b.bulan}</td>
                    <td>{formatRp(b.omzet)}</td>
                    <td>{formatRp(b.dasar_pajak)}</td>
                    <td>{formatRp(b.pph_terutang)}</td>
                    <td>{formatRp(b.pph_disetor)}</td>
                    <td>{b.selisih ? formatRp(b.selisih) : '–'}</td>
                    <td>
                      <span className={'pill-status ' + kelas} title={`Jatuh tempo ${formatTanggal(b.jatuh_tempo)}`}>{label}</span>
                      {b.status === 'kurang' && !!b.pajak_lain && (
                        <p className="mt-1 text-xs text-muted-foreground">Ada "Pajak & Izin" {formatRp(b.pajak_lain)} di bulan berikutnya. Kalau itu setoran PPh Final, ubah jenisnya ke Setor PPh Final.</p>
                      )}
                    </td>
                    {r.pkp && <><td>{formatRp(b.ppn_keluaran ?? 0)}</td><td>{formatRp(b.ppn_disetor ?? 0)}</td></>}
                    {bisaCatat && (
                      <td>{(b.status === 'kurang' || b.status === 'belum_jatuh_tempo') && (
                        <Button size="sm" variant="ghost" onClick={() => setSetor(b)} aria-label={`Catat setoran PPh Final ${b.bulan}`}><Plus /> Setoran</Button>
                      )}</td>
                    )}
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
        {r.pkp && <p className="mt-2 text-xs text-muted-foreground">Selisih PPN Keluaran dan PPN disetor wajar bila ada PPN Masukan yang dikreditkan.</p>}
      </>}
      <CatatSetoran key={setor?.masa ?? ''} masa={setor} onTutup={() => setSetor(null)} onTersimpan={() => { setSetor(null); void muat() }} />
    </div>
  )
}

function Ringkas({ label, nilai, teks, warna }: { label: string; nilai?: number; teks?: string; warna?: string }) {
  return (
    <div className="card">
      <div className="kpi-label">{label}</div>
      <div className="kpi-value tabular" style={warna ? { color: warna } : undefined}>{teks ?? formatRp(nilai ?? 0)}</div>
    </div>
  )
}

/** Catat setoran PPh Final untuk satu masa (jenis Setor PPh Final, dari Bank/Kas bawaan).
 *  Di-key per masa supaya isian awal mengikuti masa yang dipilih. */
function CatatSetoran({ masa, onTutup, onTersimpan }: { masa: RekonBulan | null; onTutup: () => void; onTersimpan: () => void }) {
  const [tanggal, setTanggal] = useState(hariIni)
  const [nominal, setNominal] = useState(() => (masa ? -masa.selisih : 0))
  const [error, setError] = useState('')
  const [simpan, setSimpan] = useState(false)
  async function kirim(e: FormEvent) {
    e.preventDefault()
    if (!masa) return
    setSimpan(true)
    try {
      await createTransaction({ tanggal, jenis: 'Setor PPh Final', nominal, keterangan: `PPh Final masa ${masa.bulan} ${masa.masa.slice(0, 4)}`,
        item: '', qty: 0, channel: '', pihak: '', lunas: true, tanggal_lunas: '', masa_pajak: masa.masa })
      onTersimpan()
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setSimpan(false)
    }
  }
  return (
    <Dialog open={masa !== null} onOpenChange={(o) => !o && onTutup()}>
      <DialogContent className="sm:max-w-md">
        <form onSubmit={kirim} className="grid gap-4">
          <DialogHeader>
            <DialogTitle>Catat setoran PPh Final {masa?.bulan ?? ''}</DialogTitle>
            <DialogDescription>Setelah membayar dengan kode billing 411128-420. Tercatat sebagai Setor PPh Final dari rekening bawaan.</DialogDescription>
          </DialogHeader>
          <div className="grid gap-2">
            <Label htmlFor="setor-tgl">Tanggal setor</Label>
            <Input id="setor-tgl" type="date" required min={masa ? masa.masa + '-01' : undefined} value={tanggal} onChange={(e) => setTanggal(e.target.value)} />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="setor-n">Nominal (Rp)</Label>
            <Input id="setor-n" required className="tabular" {...angka(nominal, setNominal)} />
          </div>
          {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onTutup}>Batal</Button>
            <Button type="submit" disabled={simpan || nominal <= 0}>Simpan setoran</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
