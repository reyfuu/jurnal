import { useEffect, useRef, type ReactNode } from 'react'
import { Menu } from 'lucide-react'
import { Logo } from '@/components/Logo'
import { useHP } from '@/lib/hp'

/** Bar atas tipis khusus HP: ☰, logo, dan isi kanan (mis. pil periode). */
export function BarAtasHP({ onMenu, terbuka = false, judul, children }: { onMenu: () => void; terbuka?: boolean; judul?: ReactNode; children?: ReactNode }) {
  if (!useHP()) return null
  return (
    <header className="bar-atas-hp dark">
      <button type="button" className="bar-atas-menu" aria-label="Buka menu" aria-expanded={terbuka} onClick={onMenu}>
        <Menu size={20} aria-hidden="true" />
      </button>
      {judul ?? <Logo tinggi={20} />}
      <div className="bar-atas-kanan">{children}</div>
    </header>
  )
}

/**
 * Pembungkus sidebar: di HP jadi laci dengan latar gelap; di laptop tidak mengubah apa pun.
 * Laci tertutup di HP dibuat inert (tidak bisa difokus / dibaca pembaca layar). Saat dibuka,
 * fokus pindah ke menu; saat ditutup, kembali ke tombol ☰.
 */
export function Laci({ buka, onTutup, children }: { buka: boolean; onTutup: () => void; children: ReactNode }) {
  const hp = useHP()
  const akar = useRef<HTMLDivElement>(null)
  const mulaiX = useRef<number | null>(null)
  const tutup = useRef(onTutup)
  useEffect(() => { tutup.current = onTutup })
  useEffect(() => {
    if (!buka || !hp) return
    akar.current?.querySelector<HTMLElement>('.nav-item')?.focus()
    const esc = (e: KeyboardEvent) => { if (e.key === 'Escape') tutup.current() }
    window.addEventListener('keydown', esc)
    return () => {
      window.removeEventListener('keydown', esc)
      document.querySelector<HTMLElement>('.bar-atas-menu')?.focus()
    }
  }, [buka, hp])
  return (
    <div ref={akar} className={'laci' + (buka ? ' laci-terbuka' : '')} inert={hp && !buka}
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
