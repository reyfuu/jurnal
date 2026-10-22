import { useEffect } from 'react'

// Draft form: isian disimpan di browser selama diketik, supaya error, muat ulang, atau tab yang
// tertutup tidak memaksa mengetik ulang. Dihapus setelah tersimpan atau saat pengguna menekan
// Batal. localStorage bisa tidak tersedia (mode privat): semua pemanggilan aman bila gagal.

const AWALAN = 'draft:'

export function bacaDraft<T>(kunci: string): T | null {
  try {
    const s = localStorage.getItem(AWALAN + kunci)
    return s ? (JSON.parse(s) as T) : null
  } catch {
    return null
  }
}

export function simpanDraft(kunci: string, nilai: unknown) {
  try { localStorage.setItem(AWALAN + kunci, JSON.stringify(nilai)) } catch { /* abaikan */ }
}

export function hapusDraft(kunci: string) {
  try { localStorage.removeItem(AWALAN + kunci) } catch { /* abaikan */ }
}

/** Simpan nilai sebagai draft setiap berubah; kunci null = form tidak sedang diisi. */
export function useSimpanDraft(kunci: string | null, nilai: unknown) {
  useEffect(() => {
    if (kunci && nilai != null) simpanDraft(kunci, nilai)
  }, [kunci, nilai])
}
