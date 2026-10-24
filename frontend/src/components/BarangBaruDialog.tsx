import { useState, type FormEvent } from 'react'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { akunAwalBarang, createStok, type PilihanAkunStok, type PilihanBarang } from '@/lib/api'
import { buatSKU } from '@/lib/sku'
import { SKALA } from '@/lib/qty'

// Barang baru langsung dari formulir transaksi: cukup nama, satuan, dan (bila usaha punya lebih
// dari satu) akun persediaannya. SKU dibuat otomatis; detail lain bisa dilengkapi di Stok & Gudang.

export function BarangBaruDialog({ pilihanAkun, skuTerpakai = [], onSelesai, onBatal }: {
  pilihanAkun: PilihanAkunStok[]; skuTerpakai?: string[]; onSelesai: (barang: PilihanBarang) => void; onBatal: () => void
}) {
  const [nama, setNama] = useState('')
  const [satuan, setSatuan] = useState('pcs')
  const [akun, setAkun] = useState(() => akunAwalBarang(pilihanAkun) ?? '')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const banyakAkun = pilihanAkun.length > 1 // satu akun: tak perlu dipilih, tetap terkirim

  async function simpan(e: FormEvent) {
    e.preventDefault()
    e.stopPropagation() // dialog ini di dalam formulir transaksi: submit jangan ikut menyimpan transaksi
    setBusy(true)
    setError('')
    try {
      const sku = buatSKU(nama, new Set(skuTerpakai.map((s) => s.toUpperCase())))
      const kategori = pilihanAkun.find((p) => p.kode === akun)?.nama ?? 'Bahan Baku'
      const b = await createStok({ sku, nama, kategori, akun: akun || undefined, jumlah: 0, satuan, stok_min: 0, harga_pokok: 0, harga_jual: 0 })
      onSelesai({ id: b.id, sku: b.sku, nama: b.nama, kategori: b.kategori, satuan: b.satuan, jumlah_mili: b.jumlah * SKALA, harga_jual: b.harga_jual })
    } catch (err) {
      setError((err as Error).message)
      setBusy(false)
    }
  }

  return (
    <Dialog open onOpenChange={(o) => !o && onBatal()}>
      <DialogContent>
        <form onSubmit={simpan} className="grid gap-3">
          <DialogHeader>
            <DialogTitle>Barang baru</DialogTitle>
            <DialogDescription>Ditambahkan ke Stok & Gudang lalu langsung dipilih di transaksi ini.</DialogDescription>
          </DialogHeader>
          <div className="grid gap-1.5"><Label htmlFor="bbaru-nama">Nama barang</Label>
            <Input id="bbaru-nama" required autoFocus placeholder="mis. Kain katun" value={nama} onChange={(e) => setNama(e.target.value)} /></div>
          <div className="grid gap-3 sm:grid-cols-2">
            <div className="grid gap-1.5"><Label htmlFor="bbaru-satuan">Satuan</Label>
              <Input id="bbaru-satuan" required placeholder="pcs, kg, m" value={satuan} onChange={(e) => setSatuan(e.target.value)} /></div>
            {banyakAkun && (
              <div className="grid gap-1.5"><Label htmlFor="bbaru-akun">Akun persediaan</Label>
                <select id="bbaru-akun" className="input-field" value={akun} onChange={(e) => setAkun(e.target.value)}>
                  {pilihanAkun.map((p) => <option key={p.kode} value={p.kode}>{p.kode} · {p.nama}</option>)}
                </select></div>
            )}
          </div>
          {nama.trim() && <p className="text-xs text-muted-foreground">SKU otomatis: {buatSKU(nama, new Set(skuTerpakai.map((s) => s.toUpperCase()))) || '—'}</p>}
          {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onBatal}>Batal</Button>
            <Button type="submit" disabled={busy || !nama.trim() || !satuan.trim()}>{busy ? 'Menyimpan…' : 'Simpan barang'}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
