import { Lock } from 'lucide-react'

/** Layar pengganti fitur berbayar untuk usaha yang belum berlangganan. */
export function FiturTerkunci({ nama, onLangganan }: { nama: string; onLangganan?: () => void }) {
  return (
    <div className="card" style={{ padding: 28, maxWidth: 620, alignItems: 'flex-start', gap: 12 }}>
      <Lock size={28} aria-hidden="true" style={{ color: 'var(--text-brand)' }} />
      <h1 className="text-xl font-bold tracking-tight">{nama} terbuka setelah berlangganan</h1>
      <p className="text-sm leading-relaxed text-muted-foreground">
        Dengan berlangganan, transaksi Anda tercatat dan laporan keuangan, pajak bulanan, invoice, serta analisis AI lengkap
        tersusun otomatis. Selama belum berlangganan, Anda tetap bisa memakai Laporan AI dari data pendaftaran dan Alat Gratis.
      </p>
      {onLangganan
        ? <button type="button" className="btn-cta accent" style={{ padding: '9px 20px', fontSize: 13 }} onClick={onLangganan}>Pilih paket</button>
        : <p className="text-sm">Hubungi Owner usaha Anda untuk memilih paket.</p>}
    </div>
  )
}
