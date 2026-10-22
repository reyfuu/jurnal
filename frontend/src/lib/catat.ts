/** Jenis Catat cepat: menentukan baris awal (akun pokok & sisinya). */
export const JENIS_CATAT = [
  { id: 'Penjualan', label: 'Penjualan', masuk: true },
  { id: 'Biaya Lain', label: 'Biaya / pengeluaran', masuk: false },
  { id: 'Beli Bahan / Stok', label: 'Beli bahan / barang', masuk: false },
] as const
