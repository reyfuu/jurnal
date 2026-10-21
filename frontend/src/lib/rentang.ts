// Rentang tanggal untuk laporan: awal & akhir bulan dari periode "YYYY-MM", dan pilihan cepat.

const pad = (n: number) => String(n).padStart(2, '0')

export function rentangBulan(period: string): { dari: string; sampai: string } {
  const [y, m] = period.split('-').map(Number)
  const akhir = new Date(y, m, 0).getDate()
  return { dari: `${y}-${pad(m)}-01`, sampai: `${y}-${pad(m)}-${pad(akhir)}` }
}

/** Pilihan cepat relatif terhadap periode laporan yang sedang dipilih. */
export function pilihanCepat(period: string): { label: string; dari: string; sampai: string }[] {
  const [y, m] = period.split('-').map(Number)
  const bulan = (yy: number, mm: number) => {
    const d = new Date(yy, mm - 1, 1)
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}`
  }
  const ini = rentangBulan(period)
  const lalu = rentangBulan(bulan(y, m - 1))
  return [
    { label: 'Bulan ini', ...ini },
    { label: 'Bulan lalu', ...lalu },
    { label: '3 bulan terakhir', dari: rentangBulan(bulan(y, m - 2)).dari, sampai: ini.sampai },
    { label: 'Tahun ini', dari: `${y}-01-01`, sampai: ini.sampai },
  ]
}
