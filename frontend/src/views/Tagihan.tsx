import { useState } from 'react'
import { PageHeader } from '@/components/PageHeader'
import { DaftarPelanggan } from '@/views/Pelanggan'
import { RiwayatPihakDialog } from '@/views/RiwayatPihak'

// Menu Tagihan di sidebar berisi Invoice, Piutang & Utang, dan Pelanggan. Halaman Pelanggan tidak
// punya izin sendiri: izinnya ikut Invoice (lihat IZIN_VIEW di lib/nav.ts).
export function HalamanPelanggan() {
  const [riwayat, setRiwayat] = useState<string | null>(null)
  return (
    <>
      <PageHeader title="Pelanggan" subtitle="Data pelanggan untuk invoice: nama, alamat, dan NPWP terisi otomatis saat membuat invoice." />
      <DaftarPelanggan onRiwayat={(p) => setRiwayat(p.nama)} />
      <RiwayatPihakDialog nama={riwayat} onTutup={() => setRiwayat(null)} />
    </>
  )
}
