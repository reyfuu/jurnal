import { Lock } from 'lucide-react'

/** Layar pengganti fitur berbayar untuk usaha yang belum berlangganan, atau (pro) fitur khusus
 *  paket Pro untuk pelanggan Starter. */
export function FiturTerkunci({ nama, onLangganan, pro }: { nama: string; onLangganan?: () => void; pro?: boolean }) {
  const Judul = pro ? 'h2' : 'h1' // kartu Pro tampil di dalam halaman yang sudah punya judul
  return (
    <div className="card" style={{ padding: 28, maxWidth: 620, alignItems: 'flex-start', gap: 12 }}>
      <Lock size={28} aria-hidden="true" style={{ color: 'var(--text-brand)' }} />
      <Judul className="text-xl font-bold tracking-tight">{nama} {pro ? 'khusus paket Pro' : 'terbuka setelah berlangganan'}</Judul>
      <p className="text-sm leading-relaxed text-muted-foreground">
        {pro
          ? 'Paket Pro mencocokkan setoran pajak tiap bulan dengan pembukuan, menandai selisihnya sebelum SPT Tahunan, dan menyiapkan rekapnya dalam Excel dan PDF.'
          : <>Dengan berlangganan, transaksi Anda tercatat dan laporan keuangan, pajak bulanan, invoice, serta analisis AI lengkap
            tersusun otomatis. Selama belum berlangganan, Anda tetap bisa memakai Laporan AI dari data pendaftaran dan Alat Gratis.</>}
      </p>
      {onLangganan
        ? <button type="button" className="btn-cta accent" style={{ padding: '9px 20px', fontSize: 13 }} onClick={onLangganan}>{pro ? 'Upgrade ke Pro' : 'Pilih paket'}</button>
        : <p className="text-sm">Hubungi Owner usaha Anda untuk memilih paket.</p>}
    </div>
  )
}
