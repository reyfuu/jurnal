import { useState } from 'react'
import { Plus, Trash2 } from 'lucide-react'
import { BarangBaruDialog } from '@/components/BarangBaruDialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { getStok, type PilihanAkunStok, type PilihanBarang, type RincianBarang } from '@/lib/api'
import { angka } from '@/lib/angka'
import { formatRp } from '@/lib/format'
import { formatQty, parseQty, qtyKeTeks } from '@/lib/qty'
import { JENIS_BELI, nilaiRincian, sisaNominal, type NilaiRincian } from '@/lib/rincian'

// Pemilih barang transaksi (Penjualan / Beli Bahan / Stok): beberapa baris barang + jumlah, nilai
// per barang untuk pembelian, "+ Barang baru" di tempat, dan tanda Bukan barang (jasa).
// Di perpetual (wajib) salah satunya harus diisi; aturannya di lib/rincian.ts dan server.

const BARU = -1 // nilai opsi "+ Barang baru"
const barisKosong = (): RincianBarang => ({ stok_id: 0, qty_stok: 0 })

/** onBarangBaru: barang yang baru dibuat lewat "+ Barang baru", supaya induk ikut memperbarui daftarnya.
 *  bolehJasa=false menyembunyikan centang Bukan barang (jasa), mis. di dialog Beli bahan.
 *  bolehBarangBaru: opsi "+ Barang baru" hanya untuk yang boleh menambah barang (bukan lihat-sebagai). */
export function PilihRincianBarang({ id = 'rb', jenis, nominal, value, onChange, barang, wajib, onBarangBaru, bolehJasa = true, bolehBarangBaru = false }: {
  id?: string; jenis: string; nominal: number; value: NilaiRincian; onChange: (v: NilaiRincian) => void; barang: PilihanBarang[]; wajib: boolean
  onBarangBaru?: (b: PilihanBarang) => void; bolehJasa?: boolean; bolehBarangBaru?: boolean
}) {
  const [tambahan, setTambahan] = useState<PilihanBarang[]>([]) // barang baru dari dialog
  const [teks, setTeks] = useState<string[]>(() => value.rincian.map((r) => qtyKeTeks(r.qty_stok)))
  const [baru, setBaru] = useState<{ baris: number; akun: PilihanAkunStok[] | null } | null>(null)
  const semua = [...barang, ...tambahan.filter((t) => !barang.some((b) => b.id === t.id))]
  const beli = jenis === JENIS_BELI
  const baris = value.rincian.length ? value.rincian : [barisKosong()]

  const ubahBaris = (i: number, patch: Partial<RincianBarang>) =>
    onChange({ rincian: baris.map((r, j) => (j === i ? { ...r, ...patch } : r)), bukan_barang: false })
  const hapus = (i: number) => {
    setTeks((ts) => ts.filter((_, j) => j !== i))
    onChange({ rincian: baris.filter((_, j) => j !== i), bukan_barang: false })
  }
  function bukaBaru(i: number) {
    setBaru({ baris: i, akun: null })
    // pilihan akun persediaan dari Stok & Gudang; gagal (mis. tanpa akses) → server menebak akun
    getStok().then((s) => setBaru((b) => b && { ...b, akun: s.pilihan_akun ?? [] }), () => setBaru((b) => b && { ...b, akun: [] }))
  }

  return (
    <div className="grid gap-2 sm:col-span-2">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <span className="text-sm font-medium">Barang dari Stok & Gudang{wajib ? '' : ' (opsional)'}</span>
        {bolehJasa && (
          <label className="flex items-center gap-2 text-sm">
            <input type="checkbox" checked={value.bukan_barang}
              onChange={(e) => { setTeks([]); onChange({ rincian: [], bukan_barang: e.target.checked }) }} /> Bukan barang (jasa)
          </label>
        )}
      </div>
      {!value.bukan_barang && <>
        {baris.map((r, i) => {
          const b = semua.find((x) => x.id === r.stok_id)
          const idSel = i === 0 ? id : `${id}-${i}`
          return (
            <div key={i} className="flex items-start gap-2">
              <div className={`grid min-w-0 flex-1 gap-2 ${beli ? 'sm:grid-cols-[minmax(0,1fr)_7rem_9rem]' : 'sm:grid-cols-[minmax(0,1fr)_7rem]'} items-start`}>
              <div className="grid gap-1">
                <select id={idSel} aria-label={`Barang baris ${i + 1}`} className="input-field" value={r.stok_id || 0}
                  onChange={(e) => {
                    const v = Number(e.target.value)
                    if (v === BARU) bukaBaru(i)
                    else ubahBaris(i, { stok_id: v })
                  }}>
                  <option value={0}>— pilih barang —</option>
                  {semua.map((x) => <option key={x.id} value={x.id}>{x.nama}{x.sku ? ` (${x.sku})` : ''}</option>)}
                  {bolehBarangBaru && <option value={BARU}>+ Barang baru</option>}
                </select>
                {b && <span className="text-xs text-muted-foreground">Stok sekarang {formatQty(b.jumlah_mili)} {b.satuan}</span>}
              </div>
              <Input aria-label={`Jumlah baris ${i + 1}${b?.satuan ? ` (${b.satuan})` : ''}`} inputMode="decimal" className="tabular"
                placeholder={b?.satuan ? `jumlah (${b.satuan})` : 'jumlah'} value={teks[i] ?? qtyKeTeks(r.qty_stok)}
                onChange={(e) => {
                  const v = e.target.value.replace(/[^\d.,]/g, '')
                  setTeks((ts) => { const n = [...ts]; n[i] = v; return n })
                  ubahBaris(i, { qty_stok: parseQty(v) })
                }} />
              {beli && <Input aria-label={`Nilai baris ${i + 1} (Rp)`} className="tabular" placeholder={baris.length > 1 ? 'nilai (Rp)' : 'nilai (Rp), opsional'}
                {...angka(r.nilai ?? 0, (n) => ubahBaris(i, { nilai: n || undefined }))} />}
              </div>
              <Button type="button" variant="ghost" size="icon-sm" aria-label={`Hapus baris ${i + 1}`} disabled={baris.length === 1 && !r.stok_id}
                onClick={() => hapus(i)}><Trash2 /></Button>
            </div>
          )
        })}
        <div className="flex flex-wrap items-center gap-3">
          <Button type="button" variant="outline" size="sm" onClick={() => onChange({ rincian: [...baris, barisKosong()], bukan_barang: false })}>
            <Plus aria-hidden="true" /> Tambah barang
          </Button>
          {beli && nilaiRincian(baris) > 0 && (
            <span className="text-xs text-muted-foreground">Sisa {formatRp(sisaNominal(nominal, baris))} dari nominal dicatat ke akun bawaan (mis. ongkir).</span>
          )}
          {beli && baris.length === 1 && !baris[0].nilai && <span className="text-xs text-muted-foreground">Nilai kosong = seluruh nominal.</span>}
        </div>
      </>}
      {!bolehBarangBaru && !value.bukan_barang && (
        <p className="text-xs text-muted-foreground">Barang belum ada di daftar? Minta pemilik menambahkannya di Stok & Gudang{bolehJasa ? ', atau centang Bukan barang (jasa)' : ''}.</p>
      )}
      {wajib && bolehJasa && <p className="text-xs text-muted-foreground">Pencatatan perpetual: pilih barang & jumlahnya, atau tandai Bukan barang (jasa).</p>}
      {baru && baru.akun && (
        <BarangBaruDialog pilihanAkun={baru.akun} skuTerpakai={semua.map((x) => x.sku).filter(Boolean)} onBatal={() => setBaru(null)}
          onSelesai={(x) => {
            setTambahan((ts) => [...ts, x])
            onBarangBaru?.(x)
            ubahBaris(baru.baris, { stok_id: x.id })
            setBaru(null)
          }} />
      )}
    </div>
  )
}
