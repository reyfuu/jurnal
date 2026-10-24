import { useState } from 'react'
import { Plus, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { createStok, type PilihanAkunStok, type StokInput } from '@/lib/api'
import { angka } from '@/lib/angka'
import { buatSKU } from '@/lib/sku'

// Tambah banyak barang sekaligus dalam satu tabel (tidak satu per satu). SKU terisi otomatis dari
// nama barang; begitu SKU diketik sendiri, tidak ditimpa lagi.

type Baris = StokInput & { _k: number; skuManual?: boolean }
let urut = 0
const kosong = (akun?: string): Baris => ({ _k: ++urut, sku: '', nama: '', kategori: 'Bahan Baku', akun, jumlah: 0, satuan: 'pcs', stok_min: 0, harga_pokok: 0, harga_jual: 0 })

/** Akun persediaan awal barang baru: Bahan Baku bila ada, kalau tidak yang pertama. */
export const akunAwalBarang = (pilihan: PilihanAkunStok[]) => (pilihan.find((p) => /bahan baku/i.test(p.nama)) ?? pilihan[0])?.kode


export function TambahBarangBanyak({ pilihanAkun = [], skuTerpakai, namaTerpakai = [], onBatal, onSelesai }: { pilihanAkun?: PilihanAkunStok[]; skuTerpakai: string[]; namaTerpakai?: string[]; onBatal: () => void; onSelesai: (jumlah: number) => void }) {
  const [baris, setBaris] = useState<Baris[]>(() => [kosong(akunAwalBarang(pilihanAkun)), kosong(akunAwalBarang(pilihanAkun)), kosong(akunAwalBarang(pilihanAkun))])
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const terisi = baris.filter((b) => b.nama.trim())
  // SKU & nama tidak boleh sama dengan barang lain (server juga menolak)
  const kunci = (s: string) => s.trim().replace(/\s+/g, ' ').toLowerCase()
  const kembar = (() => {
    const sku = new Set(skuTerpakai.map(kunci)), nama = new Set(namaTerpakai.map(kunci))
    for (const b of terisi) {
      if (b.sku && sku.has(kunci(b.sku))) return `SKU ${b.sku} sudah dipakai barang lain.`
      if (nama.has(kunci(b.nama))) return `Barang bernama "${b.nama.trim()}" sudah ada.`
      if (b.sku) sku.add(kunci(b.sku))
      nama.add(kunci(b.nama))
    }
    return ''
  })()

  // SKU otomatis unik terhadap barang yang sudah ada dan baris lain di tabel ini.
  function skuOtomatis(bs: Baris[], k: number, nama: string) {
    const lain = new Set([...skuTerpakai, ...bs.filter((b) => b._k !== k && b.sku).map((b) => b.sku)].map((s) => s.toUpperCase()))
    return buatSKU(nama, lain)
  }
  const ubah = (k: number, p: Partial<Baris>) => setBaris((bs) => bs.map((b) => {
    if (b._k !== k) return b
    const x = { ...b, ...p }
    if ('nama' in p && !x.skuManual) x.sku = skuOtomatis(bs, k, p.nama ?? '')
    return x
  }))

  async function simpan() {
    setBusy(true)
    setError('')
    let ok = 0
    try {
      for (const b of terisi) {
        const { _k, skuManual, ...input } = b
        void _k; void skuManual
        await createStok(input)
        ok++
      }
      onSelesai(ok)
    } catch (err) {
      // Yang sudah tersimpan dibuang dari tabel supaya tidak tersimpan dua kali saat diulang.
      setBaris((bs) => bs.filter((b) => !terisi.slice(0, ok).some((t) => t._k === b._k)))
      setError(`${ok ? `${ok} barang tersimpan. ` : ''}Baris "${terisi[ok]?.nama}": ${(err as Error).message}`)
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="grid gap-3" style={{ padding: 14, border: '2px solid var(--accent-orange)', borderRadius: 12, background: 'var(--accent-orange-soft)' }}>
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <div className="chart-title">Tambah barang</div>
          <div className="chart-subtitle">Isi beberapa barang sekaligus. SKU terisi otomatis dari nama, boleh diganti sendiri.</div>
        </div>
        <Button type="button" size="sm" onClick={() => setBaris((bs) => [...bs, kosong(akunAwalBarang(pilihanAkun))])}><Plus aria-hidden="true" /> Baris</Button>
      </div>
      <div className="table-scroll">
        <table className="data-table tabular" style={{ minWidth: 980 }}>
          <thead>
            <tr><th style={{ width: 120 }}>SKU</th><th>Nama barang</th><th style={{ width: 140 }}>Kategori (akun persediaan)</th><th style={{ width: 90 }}>Satuan</th>
              <th style={{ textAlign: 'right', width: 100 }}>Jumlah</th><th style={{ textAlign: 'right', width: 100 }}>Stok min.</th>
              <th style={{ textAlign: 'right', width: 130 }}>Harga pokok</th><th style={{ textAlign: 'right', width: 130 }}>Harga jual</th><th style={{ width: 40 }} /></tr>
          </thead>
          <tbody>
            {baris.map((b, i) => (
              <tr key={b._k}>
                <td><input className="input-field" aria-label={`SKU baris ${i + 1}`} placeholder="otomatis" value={b.sku}
                  onChange={(e) => ubah(b._k, { sku: e.target.value.toUpperCase(), skuManual: e.target.value.trim() !== '' })} /></td>
                <td><input className="input-field" aria-label={`Nama barang baris ${i + 1}`} placeholder="mis. Singkong kupas" value={b.nama} onChange={(e) => ubah(b._k, { nama: e.target.value })} /></td>
                <td>
                  <select className="input-field" aria-label={`Kategori baris ${i + 1}`} value={b.akun ?? ''}
                    onChange={(e) => ubah(b._k, { akun: e.target.value, kategori: pilihanAkun.find((p) => p.kode === e.target.value)?.nama ?? b.kategori })}>
                    {pilihanAkun.length === 0 && <option value="">{b.kategori}</option>}
                    {pilihanAkun.map((p) => <option key={p.kode} value={p.kode}>{p.kode} · {p.nama}</option>)}
                  </select>
                </td>
                <td><input className="input-field" aria-label={`Satuan baris ${i + 1}`} value={b.satuan} onChange={(e) => ubah(b._k, { satuan: e.target.value })} /></td>
                <td><input className="input-field tabular" style={{ textAlign: 'right' }} aria-label={`Jumlah baris ${i + 1}`} {...angka(b.jumlah, (n) => ubah(b._k, { jumlah: n }))} /></td>
                <td><input className="input-field tabular" style={{ textAlign: 'right' }} aria-label={`Stok minimum baris ${i + 1}`} {...angka(b.stok_min, (n) => ubah(b._k, { stok_min: n }))} /></td>
                <td><input className="input-field tabular" style={{ textAlign: 'right' }} aria-label={`Harga pokok baris ${i + 1}`} {...angka(b.harga_pokok, (n) => ubah(b._k, { harga_pokok: n }))} /></td>
                <td><input className="input-field tabular" style={{ textAlign: 'right' }} aria-label={`Harga jual baris ${i + 1}`} {...angka(b.harga_jual, (n) => ubah(b._k, { harga_jual: n }))} /></td>
                <td><Button type="button" variant="ghost" size="icon-sm" aria-label={`Hapus baris ${i + 1}`} onClick={() => setBaris((bs) => bs.filter((x) => x._k !== b._k))}><Trash2 /></Button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {error && <div className="pill-status pill-red" role="alert" style={{ whiteSpace: 'normal', alignSelf: 'flex-start' }}>{error}</div>}
      {kembar && <div className="pill-status pill-red" role="alert" style={{ whiteSpace: 'normal', alignSelf: 'flex-start' }}>{kembar} SKU dan nama barang tidak boleh sama.</div>}
      <div className="flex flex-wrap items-center gap-2">
        <Button type="button" disabled={busy || terisi.length === 0 || !!kembar} onClick={() => void simpan()}>{busy ? 'Menyimpanâ€¦' : `Simpan ${terisi.length} barang`}</Button>
        <Button type="button" variant="outline" onClick={onBatal}>Batal</Button>
        <span className="text-xs text-muted-foreground">Baris tanpa nama dilewati. Banyak barang dari Excel? Pakai tombol Impor Excel.</span>
      </div>
    </div>
  )
}
