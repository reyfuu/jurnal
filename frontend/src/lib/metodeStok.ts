import { useEffect, useState } from 'react'
import { getMetodeStok, type MetodeNilaiStok, type MetodeStok } from '@/lib/api'

export const NAMA_METODE: Record<MetodeNilaiStok, string> = { fifo: 'FIFO (yang dibeli duluan keluar duluan)', rata_rata: 'rata-rata bergerak' }

/** Cara mencatat stok (profil usaha), untuk keterangan perhitungan stok & HPP. */
export const NAMA_PENCATATAN: Record<'periodik' | 'perpetual', string> = {
  periodik: 'Periodik: stok dihitung (opname) di akhir periode; HPP = persediaan awal + pembelian - persediaan akhir',
  perpetual: 'Perpetual: setiap barang masuk & keluar langsung dicatat; HPP tercatat setiap kali barang terjual',
}

/** Metode nilai stok usaha (null selama dimuat); dipakai teks penjelasan & saldo awal stok. */
export function useMetodeStok() {
  const [m, setM] = useState<MetodeStok | null>(null)
  useEffect(() => { getMetodeStok().then(setM, () => setM(null)) }, [])
  return m
}
