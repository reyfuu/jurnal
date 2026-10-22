import { useEffect, useState } from 'react'
import { Camera, RotateCcw, Sparkles } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { bacaNota, type SaranNota } from '@/lib/api'
import { formatRp, formatTanggal } from '@/lib/format'

// Catat transaksi dari foto: foto nota/struk dikecilkan di browser, dibaca AI, lalu hasilnya
// ditampilkan sebagai pratinjau beserta rekomendasi AI. "Pakai" mengisi form transaksi biasa;
// pengguna memeriksa lalu menyimpan sendiri.

const SISI_MAKS = 1600

/** Perkecil foto (maks 1600 px, JPEG) supaya cepat diunggah dan hemat kuota AI. */
async function kecilkan(file: File): Promise<Blob> {
  const url = URL.createObjectURL(file)
  try {
    const img = await new Promise<HTMLImageElement>((ok, gagal) => {
      const i = new Image()
      i.onload = () => ok(i)
      i.onerror = () => gagal(new Error('Foto tidak bisa dibuka. Pakai JPG atau PNG.'))
      i.src = url
    })
    const skala = Math.min(1, SISI_MAKS / Math.max(img.width, img.height))
    const c = document.createElement('canvas')
    c.width = Math.round(img.width * skala)
    c.height = Math.round(img.height * skala)
    c.getContext('2d')!.drawImage(img, 0, 0, c.width, c.height)
    return await new Promise<Blob>((ok, gagal) => c.toBlob((b) => (b ? ok(b) : gagal(new Error('Foto gagal diproses'))), 'image/jpeg', 0.85))
  } finally {
    URL.revokeObjectURL(url)
  }
}

const WARNA_YAKIN = { tinggi: 'pill-green', sedang: 'pill-amber', rendah: 'pill-red' } as const

export function BacaFotoNota({ onPakai }: { onPakai: (s: SaranNota) => void }) {
  const [pratinjau, setPratinjau] = useState('')
  const [membaca, setMembaca] = useState(false)
  const [hasil, setHasil] = useState<SaranNota | null>(null)
  const [error, setError] = useState('')
  useEffect(() => () => { if (pratinjau) URL.revokeObjectURL(pratinjau) }, [pratinjau])

  async function pilih(file: File | undefined) {
    if (!file) return
    setError('')
    setHasil(null)
    setMembaca(true)
    try {
      const kecil = await kecilkan(file)
      setPratinjau(URL.createObjectURL(kecil))
      setHasil(await bacaNota(kecil))
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setMembaca(false)
    }
  }

  return (
    <div className="grid gap-3">
      {!pratinjau && !membaca && (
        <label className="flex cursor-pointer flex-col items-center gap-2 rounded-lg border-2 border-dashed px-4 py-8 text-center" style={{ borderColor: 'var(--accent-orange)' }}>
          <Camera size={28} aria-hidden="true" style={{ color: 'var(--accent-orange)' }} />
          <span className="text-sm font-semibold">Foto atau pilih gambar nota / struk / bukti transfer</span>
          <span className="text-xs text-muted-foreground">AI membaca tanggal, total, toko, dan jenis transaksinya. Anda periksa dulu sebelum disimpan.</span>
          <input type="file" accept="image/*" capture="environment" className="sr-only" onChange={(e) => void pilih(e.target.files?.[0])} />
        </label>
      )}
      {(pratinjau || membaca) && (
        <div className="grid gap-3 sm:grid-cols-[160px_1fr]">
          {pratinjau ? <img src={pratinjau} alt="Foto nota" className="w-full rounded-md border object-contain" style={{ maxHeight: 220 }} />
            : <div className="rounded-md border" style={{ height: 160 }} />}
          <div className="grid content-start gap-2">
            {membaca && <p className="text-sm" role="status">AI sedang membaca nota…</p>}
            {hasil && (
              <>
                <div className="flex flex-wrap items-center gap-2">
                  <span className="inline-flex items-center gap-1 text-sm font-semibold"><Sparkles size={14} aria-hidden="true" style={{ color: 'var(--accent-blue)' }} /> Pratinjau hasil bacaan AI</span>
                  <span className={'pill-status ' + WARNA_YAKIN[hasil.yakin]} style={{ fontSize: 10 }}>keyakinan {hasil.yakin}</span>
                </div>
                <table className="data-table text-sm">
                  <tbody>
                    <tr><td className="text-muted-foreground">Tanggal</td><td>{formatTanggal(hasil.tanggal)}</td></tr>
                    <tr><td className="text-muted-foreground">Jenis</td><td>{hasil.jenis || <em>belum ditebak, pilih sendiri</em>}</td></tr>
                    <tr><td className="text-muted-foreground">Total</td><td className="tabular"><strong>{formatRp(hasil.nominal)}</strong></td></tr>
                    {hasil.pihak && <tr><td className="text-muted-foreground">Toko / pihak</td><td>{hasil.pihak}</td></tr>}
                    {hasil.keterangan && <tr><td className="text-muted-foreground">Keterangan</td><td>{hasil.keterangan}</td></tr>}
                    <tr><td className="text-muted-foreground">Status</td><td>{hasil.lunas ? 'Lunas' : 'Belum lunas (tempo)'}</td></tr>
                  </tbody>
                </table>
                {!!hasil.item?.length && (
                  <details className="text-xs">
                    <summary className="cursor-pointer">{hasil.item.length} barang terbaca</summary>
                    <ul style={{ margin: '4px 0 0 16px', listStyle: 'disc' }}>
                      {hasil.item.map((it, i) => <li key={i}>{it.nama}{it.qty ? ` × ${it.qty}` : ''}{it.harga ? ` — ${formatRp(it.harga)}` : ''}</li>)}
                    </ul>
                  </details>
                )}
                {hasil.alasan && <p className="text-xs"><strong>Rekomendasi AI:</strong> {hasil.alasan}</p>}
                {hasil.catatan.length > 0 && (
                  <ul className="text-xs" style={{ color: 'var(--text-warn)', margin: '0 0 0 16px', listStyle: 'disc' }}>
                    {hasil.catatan.map((c, i) => <li key={i}>{c}</li>)}
                  </ul>
                )}
                <div className="flex flex-wrap gap-2">
                  <Button type="button" onClick={() => onPakai(hasil)}>Pakai &amp; periksa</Button>
                  <Button type="button" variant="outline" onClick={() => { setPratinjau(''); setHasil(null) }}><RotateCcw aria-hidden="true" /> Foto ulang</Button>
                </div>
              </>
            )}
          </div>
        </div>
      )}
      {error && (
        <div className="grid gap-2">
          <p className="text-sm text-destructive" role="alert">{error}</p>
          <Button type="button" variant="outline" size="sm" style={{ justifySelf: 'start' }} onClick={() => { setError(''); setPratinjau('') }}>Coba foto lain</Button>
        </div>
      )}
    </div>
  )
}
