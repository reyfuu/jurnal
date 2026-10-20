// Tema tampilan. Default terang (Brand Guideline v1.0); kelas `dark` di <html> = gelap, dipasang
// script di index.html sebelum React jalan supaya tidak berkedip. Pilihan disimpan di localStorage.
export type Tema = 'gelap' | 'terang'

const KUNCI = 'tema'

export const temaDari = (simpanan: string | null): Tema => (simpanan === 'gelap' ? 'gelap' : 'terang')

export const temaSekarang = (): Tema => (document.documentElement.classList.contains('dark') ? 'gelap' : 'terang')

export function pasangTema(tema: Tema) {
  document.documentElement.classList.toggle('dark', tema === 'gelap')
  try {
    localStorage.setItem(KUNCI, tema)
  } catch {
    // mode privat / penyimpanan diblokir: tema tetap berlaku sampai halaman dimuat ulang
  }
}
