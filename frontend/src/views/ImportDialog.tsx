import { Download, FileSpreadsheet } from 'lucide-react'
import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { importTransactions, TEMPLATE_URL, type ImportResult } from '@/lib/api'
import { formatRp } from '@/lib/format'
import { formatTanggal } from '@/lib/transaksi'

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
  onImported: () => Promise<void>
}

export function ImportDialog({ open, onOpenChange, onImported }: Props) {
  const [file, setFile] = useState<File | null>(null)
  const [preview, setPreview] = useState<ImportResult | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  function reset(next: boolean) {
    if (!next) {
      setFile(null)
      setPreview(null)
      setError('')
    }
    onOpenChange(next)
  }

  async function choose(f: File | undefined) {
    if (!f) return
    setFile(f)
    setPreview(null)
    setError('')
    setBusy(true)
    try {
      setPreview(await importTransactions(f, false))
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setBusy(false)
    }
  }

  async function commit() {
    if (!file) return
    setBusy(true)
    setError('')
    try {
      await importTransactions(file, true)
      await onImported()
      reset(false)
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setBusy(false)
    }
  }

  const errors = preview?.errors ?? []

  return (
    <Dialog open={open} onOpenChange={reset}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Impor transaksi dari Excel</DialogTitle>
          <DialogDescription>
            Pakai template supaya kolomnya dikenali. Wajib: Tanggal, Jenis, Nominal. Opsional: Barang (SKU atau nama; isi "jasa" bila bukan barang) dan Jumlah Barang, satu barang per baris; di metode persediaan perpetual, Penjualan dan Beli Bahan / Stok wajib diisi salah satunya. Baris yang bermasalah dilewati dan ditampilkan di bawah.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4">
          <div className="flex flex-wrap items-center gap-3">
            <Button asChild variant="outline">
              <a href={TEMPLATE_URL} download>
                <Download /> Unduh template
              </a>
            </Button>
            <label className="btn-cta inline-flex cursor-pointer items-center gap-1.5">
              <FileSpreadsheet size={16} /> {file ? 'Ganti berkas' : 'Pilih berkas .xlsx'}
              <input
                type="file"
                accept=".xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
                className="sr-only"
                onChange={(e) => { void choose(e.target.files?.[0]); e.target.value = '' }}
              />
            </label>
            {file && <span className="text-sm text-muted-foreground">{file.name}</span>}
          </div>

          {busy && !preview && <p className="text-sm text-muted-foreground">Membaca berkas…</p>}
          {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
          {preview?.galat && <p className="text-sm text-destructive" role="alert">{preview.galat}. Perbaiki berkas ini sebelum mengimpor.</p>}

          {preview && (
            <>
              <div className="flex flex-wrap gap-2">
                <span className="pill-status pill-green">{preview.valid} baris siap diimpor</span>
                {errors.length > 0 && <span className="pill-status pill-red">{errors.length} baris dilewati</span>}
              </div>

              {(preview.sample ?? []).length > 0 && (
                <div>
                  <div className="kpi-label" style={{ marginBottom: 4 }}>Contoh baris yang akan masuk</div>
                  <table className="data-table" style={{ marginTop: 0 }}>
                    <thead>
                      <tr><th>Tanggal</th><th>Jenis</th><th>Keterangan</th><th style={{ textAlign: 'right' }}>Nominal</th><th>Status</th></tr>
                    </thead>
                    <tbody>
                      {preview.sample!.map((t, i) => (
                        <tr key={i}>
                          <td className="tabular">{formatTanggal(t.tanggal)}</td>
                          <td>{t.jenis}</td>
                          <td>{t.keterangan || '-'}</td>
                          <td className="tabular" style={{ textAlign: 'right' }}>{formatRp(t.nominal)}</td>
                          <td>{t.lunas ? 'Lunas' : 'Belum lunas'}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}

              {errors.length > 0 && (
                <div>
                  <div className="kpi-label" style={{ marginBottom: 4 }}>Baris yang dilewati</div>
                  <ul className="max-h-40 overflow-auto rounded-md border p-2 text-sm">
                    {errors.map((e, i) => (
                      <li key={i} className="py-0.5">
                        <span className="tabular text-muted-foreground">{e.baris ? `Baris ${e.baris}` : 'Header'}:</span> {e.pesan}
                      </li>
                    ))}
                  </ul>
                </div>
              )}
            </>
          )}
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => reset(false)}>Batal</Button>
          <Button type="button" disabled={busy || !preview || preview.valid === 0 || !!preview.galat} onClick={() => void commit()}>
            {busy && preview ? 'Mengimpor…' : `Impor ${preview?.valid ?? 0} transaksi`}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
