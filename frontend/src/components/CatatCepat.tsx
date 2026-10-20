import { useEffect, useRef, useState } from 'react'
import { ArrowDownLeft, ArrowUpRight, ClipboardCheck, Factory, HandCoins, Plus, ShoppingCart } from 'lucide-react'
import type { View } from '@/lib/api'

// Tombol "+ Catat" di atas setiap halaman: pekerjaan harian tanpa harus mencari menu.
// Tiap pilihan membuka halaman tujuan langsung dengan formulirnya (aksi).

const PILIHAN: { label: string; ket: string; view: View; aksi: string; ikon: typeof Plus }[] = [
  { label: 'Penjualan', ket: 'Uang masuk dari pelanggan (tunai atau tempo)', view: 'transaksi', aksi: 'baru:Penjualan', ikon: ArrowDownLeft },
  { label: 'Biaya / pengeluaran', ket: 'Gaji, sewa, listrik, iklan, dll.', view: 'transaksi', aksi: 'baru:Biaya Lain', ikon: ArrowUpRight },
  { label: 'Beli bahan / barang', ket: 'Menambah stok & harga pokok', view: 'stok-gudang', aksi: 'beli', ikon: ShoppingCart },
  { label: 'Produksi', ket: 'Bahan dipakai, produk jadi bertambah', view: 'produksi-supplier', aksi: 'catat', ikon: Factory },
  { label: 'Pelunasan piutang / utang', ket: 'Pelanggan bayar atau Anda bayar supplier', view: 'piutang-utang', aksi: '', ikon: HandCoins },
  { label: 'Stok opname', ket: 'Hitung fisik barang di gudang', view: 'stok-gudang', aksi: 'opname', ikon: ClipboardCheck },
]

export function CatatCepat({ boleh, onPilih }: { boleh: (v: View) => boolean; onPilih: (v: View, aksi: string) => void }) {
  const [buka, setBuka] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  useEffect(() => {
    if (!buka) return
    const tutup = (e: MouseEvent) => { if (!ref.current?.contains(e.target as Node)) setBuka(false) }
    const esc = (e: KeyboardEvent) => { if (e.key === 'Escape') setBuka(false) }
    document.addEventListener('mousedown', tutup)
    document.addEventListener('keydown', esc)
    return () => { document.removeEventListener('mousedown', tutup); document.removeEventListener('keydown', esc) }
  }, [buka])
  const daftar = PILIHAN.filter((p) => boleh(p.view))
  if (daftar.length === 0) return null
  return (
    <div ref={ref} style={{ position: 'relative' }}>
      <button className="btn-cta" aria-haspopup="menu" aria-expanded={buka} onClick={() => setBuka((b) => !b)}
        style={{ padding: '7px 16px', fontSize: 13, display: 'inline-flex', alignItems: 'center', gap: 6 }}>
        <Plus size={16} aria-hidden /> Catat
      </button>
      {buka && (
        <div role="menu" className="card" style={{ position: 'absolute', top: 'calc(100% + 6px)', left: 0, zIndex: 40, width: 320, padding: 6, boxShadow: 'var(--shadow-3)' }}>
          {daftar.map((p) => (
            <button key={p.label} role="menuitem" onClick={() => { setBuka(false); onPilih(p.view, p.aksi) }}
              className="flex w-full items-start gap-3 rounded-lg px-3 py-2.5 text-left hover:bg-[var(--bg-card-hover)]">
              <p.ikon size={18} style={{ color: 'var(--text-brand)', marginTop: 2, flexShrink: 0 }} aria-hidden />
              <span><span className="block text-sm font-semibold">{p.label}</span><span className="block text-xs text-muted-foreground">{p.ket}</span></span>
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
