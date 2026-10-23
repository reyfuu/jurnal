import { useEffect, useRef, useState, type ReactNode } from 'react'
import { Menu } from 'lucide-react'
import { Logo } from '@/components/Logo'
import { useHP } from '@/lib/hp'

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
      history.back() // popstate di bawah yang menutup
    } else {
      setBukaState(false)
    }
  }

  useEffect(() => {
    const kembali = () => {
      if (!lewatRiwayat.current) return
      lewatRiwayat.current = false
      setBukaState(false)
    }
    window.addEventListener('popstate', kembali)
    return () => window.removeEventListener('popstate', kembali)
  }, [])

  // Layar melebar (HP diputar / jendela diperbesar): sidebar desktop biasa, laci ditutup.
  useEffect(() => {
    if (!hp && buka) setBuka(false)
  })

  // Halaman di belakang laci tidak ikut tergulir.
  useEffect(() => {
    if (!buka) return
    const lama = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => { document.body.style.overflow = lama }
  }, [buka])

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

/** Bar atas tipis khusus HP: ☰, logo, dan isi kanan (mis. pil periode). */
export function BarAtasHP({ onMenu, judul, children }: { onMenu: () => void; judul?: ReactNode; children?: ReactNode }) {
  if (!useHP()) return null
  return (
    <header className="bar-atas-hp dark">
      <button type="button" className="bar-atas-menu" aria-label="Buka menu" onClick={onMenu}>
        <Menu size={20} aria-hidden="true" />
      </button>
      {judul ?? <Logo tinggi={20} />}
      <div className="bar-atas-kanan">{children}</div>
    </header>
  )
}

/** Pembungkus sidebar: di HP jadi laci dengan latar gelap; di laptop tidak mengubah apa pun. */
export function Laci({ buka, onTutup, children }: { buka: boolean; onTutup: () => void; children: ReactNode }) {
  const mulaiX = useRef<number | null>(null)
  useEffect(() => {
    if (!buka) return
    const esc = (e: KeyboardEvent) => { if (e.key === 'Escape') onTutup() }
    window.addEventListener('keydown', esc)
    return () => window.removeEventListener('keydown', esc)
  }, [buka, onTutup])
  return (
    <div className={'laci' + (buka ? ' laci-terbuka' : '')}
      onTouchStart={(e) => { mulaiX.current = e.touches[0].clientX }}
      onTouchEnd={(e) => {
        // geser ke kiri ≥ 60 px menutup laci
        if (buka && mulaiX.current !== null && mulaiX.current - e.changedTouches[0].clientX >= 60) onTutup()
        mulaiX.current = null
      }}>
      {children}
      <div className="laci-latar" aria-hidden="true" onClick={onTutup} />
    </div>
  )
}
