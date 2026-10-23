import { useEffect, useRef, useState } from 'react'
import { KUERI_HP, useHP } from '@/lib/hp'

/**
 * Menu laci (drawer) di HP: sidebar yang sama dengan laptop, meluncur dari kiri. Membuka laci
 * menambah satu langkah riwayat, jadi tombol Back HP menutup laci, bukan meninggalkan halaman.
 */
export function useLaci() {
  const hp = useHP()
  const [buka, setBukaState] = useState(false)
  const lewatRiwayat = useRef(false) // laci ini membuat entri riwayat yang belum dikonsumsi

  const setBuka = (b: boolean) => {
    if (b === buka) return
    if (b) {
      history.pushState({ ...(history.state ?? {}), laci: true }, '')
      lewatRiwayat.current = true
      setBukaState(true)
    } else if (lewatRiwayat.current) {
      // Entri riwayat laci dikonsumsi sekarang juga: ketuk latar dua kali tidak mundur dua langkah.
      lewatRiwayat.current = false
      setBukaState(false)
      history.back()
    } else {
      setBukaState(false)
    }
  }

  useEffect(() => {
    // Mendarat di entri laci yang sudah tidak terbuka (reload saat laci terbuka, atau Forward):
    // lewati entri itu supaya Back berikutnya tidak tampak diam.
    const lewatiEntriLaci = () => { if (!lewatRiwayat.current && (history.state as { laci?: boolean } | null)?.laci) history.back() }
    lewatiEntriLaci()
    const kembali = () => {
      if (!lewatRiwayat.current) return lewatiEntriLaci()
      lewatRiwayat.current = false
      setBukaState(false)
    }
    window.addEventListener('popstate', kembali)
    return () => window.removeEventListener('popstate', kembali)
  }, [])

  // Layar melebar (HP diputar / jendela diperbesar): sidebar desktop biasa, laci ditutup.
  useEffect(() => {
    if (!buka || !window.matchMedia) return
    const mq = window.matchMedia(KUERI_HP)
    const ubah = () => { if (!mq.matches && lewatRiwayat.current) history.back(); else if (!mq.matches) setBukaState(false) }
    mq.addEventListener('change', ubah)
    return () => mq.removeEventListener('change', ubah)
  }, [buka])

  // Halaman di belakang laci tidak ikut tergulir.
  useEffect(() => {
    if (!buka || !hp) return
    const lama = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => { document.body.style.overflow = lama }
  }, [buka, hp])

  /**
   * Menutup laci karena pindah halaman. true = entri riwayat laci masih ada dan pemanggil sebaiknya
   * MENGGANTI entri itu dengan rute baru (replaceState), supaya Back tidak berhenti di entri laci.
   */
  const tutupUntukPindah = () => {
    const ganti = lewatRiwayat.current
    lewatRiwayat.current = false
    setBukaState(false)
    return ganti
  }

  return { buka, setBuka, tutupUntukPindah }
}
