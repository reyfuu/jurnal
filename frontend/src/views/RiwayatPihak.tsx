import { useEffect, useState } from 'react'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { getRiwayatPihak, type RiwayatPihak } from '@/lib/api'
import { formatRp } from '@/lib/format'
import { formatTanggal } from '@/lib/transaksi'

// Riwayat transaksi per pelanggan / supplier (checklist tester No 25). Bawaan 12 bulan terakhir.

const hariIni = () => new Date().toLocaleDateString('sv-SE', { timeZone: 'Asia/Jakarta' })
const setahunLalu = () => { const d = new Date(); d.setFullYear(d.getFullYear() - 1); return d.toLocaleDateString('sv-SE', { timeZone: 'Asia/Jakarta' }) }

export function RiwayatPihakDialog({ nama, onTutup }: { nama: string | null; onTutup: () => void }) {
  const [rentang, setRentang] = useState(() => ({ dari: setahunLalu(), sampai: hariIni() }))
  const [hasil, setHasil] = useState<{ kunci: string; d?: RiwayatPihak; error?: string } | null>(null)
  const kunci = `${nama}|${rentang.dari}|${rentang.sampai}`
  useEffect(() => {
    if (!nama || rentang.dari > rentang.sampai) return
    let hidup = true
    getRiwayatPihak(nama, rentang.dari, rentang.sampai).then((d) => hidup && setHasil({ kunci, d }), (e: Error) => hidup && setHasil({ kunci, error: e.message }))
    return () => { hidup = false }
  }, [nama, rentang.dari, rentang.sampai, kunci])
  const r = hasil?.kunci === kunci ? hasil : null

  return (
    <Dialog open={!!nama} onOpenChange={(o) => !o && onTutup()}>
      <DialogContent className="sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>Riwayat {nama}</DialogTitle>
          <DialogDescription>Transaksi berpihak nama ini, termasuk invoice yang sudah dibukukan.</DialogDescription>
        </DialogHeader>
        <div className="flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
          <label className="flex items-center gap-1">dari <input className="input-field" type="date" style={{ width: 'auto' }} value={rentang.dari} onChange={(e) => setRentang({ ...rentang, dari: e.target.value })} /></label>
          <label className="flex items-center gap-1">sampai <input className="input-field" type="date" style={{ width: 'auto' }} value={rentang.sampai} onChange={(e) => setRentang({ ...rentang, sampai: e.target.value })} /></label>
        </div>
        {rentang.dari > rentang.sampai ? <p className="text-sm text-destructive" role="alert">Tanggal mulai harus sebelum tanggal akhir.</p>
          : !r ? <p className="text-sm text-muted-foreground">Memuat…</p>
          : r.error ? <p className="text-sm text-destructive" role="alert">{r.error}</p>
          : r.d && (
            <>
              <div className="list-rows tabular">
                <div className="list-row"><span>Total masuk (penjualan, pelunasan)</span><span>{formatRp(r.d.total_masuk)}</span></div>
                <div className="list-row"><span>Total keluar (pembelian, pembayaran)</span><span>{formatRp(r.d.total_keluar)}</span></div>
                {r.d.piutang_terbuka > 0 && <div className="list-row"><span>Piutang belum lunas (semua tanggal)</span><span>{formatRp(r.d.piutang_terbuka)}</span></div>}
                {r.d.utang_terbuka > 0 && <div className="list-row"><span>Utang belum lunas (semua tanggal)</span><span>{formatRp(r.d.utang_terbuka)}</span></div>}
              </div>
              <div className="table-scroll" style={{ maxHeight: 360 }}>
                <table className="data-table tabular">
                  <thead><tr><th>Tanggal</th><th>No. jurnal</th><th>Jenis</th><th>Keterangan</th><th style={{ textAlign: 'right' }}>Nominal</th><th>Status</th></tr></thead>
                  <tbody>
                    {r.d.transaksi.length === 0 ? <tr><td colSpan={6} className="text-muted-foreground">Tidak ada transaksi di rentang ini.</td></tr>
                      : r.d.transaksi.map((t) => (
                        <tr key={t.id}>
                          <td style={{ whiteSpace: 'nowrap' }}>{formatTanggal(t.tanggal)}</td>
                          <td>{t.no_jurnal}</td><td>{t.jenis}</td><td>{t.keterangan}</td>
                          <td style={{ textAlign: 'right' }}>{formatRp(t.nominal)}</td>
                          <td>{t.lunas ? 'Lunas' : 'Belum lunas'}</td>
                        </tr>
                      ))}
                  </tbody>
                </table>
              </div>
            </>
          )}
      </DialogContent>
    </Dialog>
  )
}
