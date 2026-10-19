import { Download, FileSpreadsheet } from 'lucide-react'
import { useState, type ReactNode } from 'react'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import type { ImportMode, MenuImportResult } from '@/lib/api'

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
  onImported: () => void
  title: string
  description: string
  templateUrl: string
  templateName: string
  /** jumlah baris yang sudah ada; dipakai memperingatkan mode "ganti" */
  existing: number
  satuan: string
  doImport: (file: File, commit: boolean, mode: ImportMode) => Promise<MenuImportResult>
  renderPreview: (r: MenuImportResult) => ReactNode
}

/**
 * Dialog impor Excel per menu: unduh template → pilih berkas → PRATINJAU → baru impor.
 *
 * Pratinjau bukan hiasan. Pengelompokan dan arah tagihan ditebak dari isi berkas, dan satu
 * kolom yang salah tafsir bisa membuat seluruh piutang tercatat sebagai utang tanpa satu pun
 * baris gagal. Jadi tidak ada jalur yang menyimpan tanpa ditampilkan dulu.
 */
export function MenuImportDialog({
  open, onOpenChange, onImported, title, description, templateUrl, templateName,
  existing, satuan, doImport, renderPreview,
}: Props) {
  const [file, setFile] = useState<File | null>(null)
  const [preview, setPreview] = useState<MenuImportResult | null>(null)
  const [mode, setMode] = useState<ImportMode>('tambah')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  function reset(next: boolean) {
    if (!next) {
      setFile(null)
      setPreview(null)
      setError('')
      setMode('tambah')
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
      setPreview(await doImport(f, false, mode))
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
      await doImport(file, true, mode)
      onImported()
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
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>

        <div className="grid gap-4">
          <div className="flex flex-wrap items-center gap-3">
            <Button asChild variant="outline">
              <a href={templateUrl} download={templateName}>
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

          {preview && (
            <>
              <div className="flex flex-wrap items-center gap-2">
                <span className="pill-status pill-green">{preview.valid} baris siap diimpor</span>
                {errors.length > 0 && <span className="pill-status pill-red">{errors.length} baris dilewati</span>}
                {preview.sheet && <span className="text-sm text-muted-foreground">dari sheet "{preview.sheet}"</span>}
              </div>

              {existing > 0 && preview.valid > 0 && (
                <div>
                  <div className="kpi-label" style={{ marginBottom: 6 }}>Sudah ada {existing} {satuan} tercatat</div>
                  <div className="flex flex-wrap gap-4">
                    {([
                      ['tambah', `Tambahkan, jadi ${existing + preview.valid} ${satuan}`],
                      ['ganti', `Ganti seluruh daftar — ${existing} ${satuan} lama dihapus`],
                    ] as const).map(([v, label]) => (
                      <label key={v} className="flex cursor-pointer items-center gap-2 text-sm">
                        <input type="radio" name="impor-mode" checked={mode === v} onChange={() => setMode(v)} />
                        {label}
                      </label>
                    ))}
                  </div>
                </div>
              )}

              {preview.valid > 0 && (
                <div>
                  <div className="kpi-label" style={{ marginBottom: 4 }}>Periksa dulu sebelum diimpor</div>
                  {renderPreview(preview)}
                </div>
              )}

              {errors.length > 0 && (
                <div>
                  <div className="kpi-label" style={{ marginBottom: 4 }}>Baris yang dilewati</div>
                  <ul className="max-h-40 overflow-auto rounded-md border p-2 text-sm">
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

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => reset(false)}>Batal</Button>
          <Button type="button" disabled={busy || !preview || preview.valid === 0} onClick={() => void commit()}>
            {busy && preview ? 'Mengimpor…' : `Impor ${preview?.valid ?? 0} ${satuan}`}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
