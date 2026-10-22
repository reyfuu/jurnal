import { useCallback, useEffect, useRef, useState } from 'react'
import type { AIForm } from './api'

/** Sinyal App → halaman form: isi form dengan isian AI, atau urungkan isian ber-id itu. */
export type SinyalAI = { id: number; form: AIForm; isian?: object; urungkan?: boolean }

// Tingkat modul, bukan per komponen: halaman di-remount tiap navigasi (key={bukaTab.n}), padahal
// sinyal yang sama tidak boleh mengisi ulang form, dan keadaan sebelum diisi harus tetap ada
// untuk Urungkan.
const diproses = new Set<string>()
const sebelum = new Map<number, unknown>()

/** Pasang isian AI ke form halaman ini. ambil/pasang = potret & pulihkan state form (untuk
 *  Urungkan); isi = buka form baru dengan isian. siap=false menunda (data form belum dimuat).
 *  aktif = id isian yang sedang tampil di form (untuk banner), hilang saat form ditutup. */
export function useIsiAI<S, I>(sinyal: SinyalAI | undefined, form: AIForm,
  o: { ambil: () => S; pasang: (s: S) => void; isi: (isian: I) => void; terbuka: boolean; siap?: boolean }) {
  const [aktif, setAktif] = useState<number | null>(null)
  // Fungsi terbaru dari render terakhir; disinkronkan sebelum effect sinyal di bawah berjalan.
  const cb = useRef(o)
  useEffect(() => { cb.current = o })

  const pulihkan = useCallback((id: number | null) => {
    if (id === null || !sebelum.has(id)) return
    cb.current.pasang(sebelum.get(id) as S)
    sebelum.delete(id)
    setAktif(null)
  }, [])

  const siap = o.siap ?? true
  useEffect(() => {
    if (!sinyal || sinyal.form !== form || !siap) return
    const kunci = (sinyal.urungkan ? 'u' : 'i') + sinyal.id
    if (diproses.has(kunci)) return
    diproses.add(kunci)
    if (sinyal.urungkan) {
      pulihkan(sinyal.id)
      return
    }
    sebelum.set(sinyal.id, cb.current.ambil())
    cb.current.isi(sinyal.isian as I)
    setAktif(sinyal.id) // sinyal datang dari luar komponen (Tanya AI)
  }, [sinyal, form, siap, pulihkan])

  // Form ditutup (Simpan / Batal): banner hilang; isian sudah bukan "isian AI" lagi.
  const tadi = useRef(o.terbuka)
  useEffect(() => {
    if (tadi.current && !o.terbuka) setAktif(null)
    tadi.current = o.terbuka
  }, [o.terbuka])

  return { aktif, urungkan: () => pulihkan(aktif) }
}
