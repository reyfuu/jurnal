import { useEffect, useRef, type ReactNode } from 'react'
import { Menu } from 'lucide-react'
import { Logo } from '@/components/Logo'
import { useHP } from '@/lib/hp'

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
