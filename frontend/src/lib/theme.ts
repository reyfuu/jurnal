// Tema tampilan. Kelas `dark` di <html> = gelap (default); script di index.html memasangnya
// sebelum React jalan supaya tidak berkedip. Pilihan disimpan per peramban di localStorage.
export type Tema = 'gelap' | 'terang'

const KUNCI = 'tema'

export const temaDari = (simpanan: string | null): Tema => (simpanan === 'terang' ? 'terang' : 'gelap')

export const temaSekarang = (): Tema => (document.documentElement.classList.contains('dark') ? 'gelap' : 'terang')

export function pasangTema(tema: Tema) {
  document.documentElement.classList.toggle('dark', tema === 'gelap')
  try {
    localStorage.setItem(KUNCI, tema)
  } catch {
    // mode privat / penyimpanan diblokir: tema tetap berlaku sampai halaman dimuat ulang
  }
}
