import { useEffect, useRef, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { KUERI_HP } from '@/lib/hp'

const FOKUS = 'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'

/**
 * Panel yang muncul dari bawah layar (HP): pilihan aksi atau filter. Ketuk latar / Escape menutup;
 * Tab berputar di dalam panel; layar yang melebar (HP diputar) juga menutup panel.
 */
export function LembarBawah({ buka, judul, onTutup, children }: { buka: boolean; judul: ReactNode; onTutup: () => void; children: ReactNode }) {
  const panel = useRef<HTMLDivElement>(null)
  // onTutup lewat ref: pemanggil mengirim fungsi baru tiap render, dan effect di bawah tidak boleh
  // berjalan ulang (fokus akan melompat dari isian ke panel setiap kali induk dirender ulang).
  const tutup = useRef(onTutup)
  useEffect(() => { tutup.current = onTutup })
  useEffect(() => {
    if (!buka) return
    const kembaliKe = document.activeElement as HTMLElement | null
    const tombol = (e: KeyboardEvent) => {
      if (e.key === 'Escape') return tutup.current()
      if (e.key !== 'Tab' || !panel.current) return
      const bisa = [...panel.current.querySelectorAll<HTMLElement>(FOKUS)]
      if (bisa.length === 0) { e.preventDefault(); return }
      const [awal, akhir] = [bisa[0], bisa[bisa.length - 1]]
      const di = document.activeElement
      if (e.shiftKey && (di === awal || di === panel.current)) { e.preventDefault(); akhir.focus() }
      else if (!e.shiftKey && (di === akhir || !panel.current.contains(di))) { e.preventDefault(); awal.focus() }
    }
    window.addEventListener('keydown', tombol)
    const mq = window.matchMedia?.(KUERI_HP)
    const lebar = () => { if (mq && !mq.matches) tutup.current() }
    mq?.addEventListener('change', lebar)
    const lama = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    document.body.classList.add('lembar-terbuka')
    panel.current?.focus()
    return () => {
      window.removeEventListener('keydown', tombol)
      mq?.removeEventListener('change', lebar)
      document.body.style.overflow = lama
      document.body.classList.remove('lembar-terbuka')
      if (kembaliKe?.isConnected) kembaliKe.focus() // fokus kembali ke tombol pembuka (⋯ / filter)
    }
  }, [buka])
  if (!buka) return null
  return createPortal(
    <div className="lembar-bawah-akar">
      <div className="lembar-bawah-latar" aria-hidden="true" onClick={onTutup} />
      <div className="lembar-bawah" role="dialog" aria-modal="true" aria-label={typeof judul === 'string' ? judul : undefined} tabIndex={-1} ref={panel}>
        <div className="lembar-bawah-pegangan" aria-hidden="true" />
        <div className="lembar-bawah-judul">{judul}</div>
        <div className="lembar-bawah-isi">{children}</div>
      </div>
    </div>,
    document.body,
  )
}

/** Satu baris aksi di LembarBawah; bahaya = merah (hapus, dll.). */
export function AksiLembar({ ikon, bahaya, onClick, children, disabled }: { ikon?: ReactNode; bahaya?: boolean; onClick: () => void; children: ReactNode; disabled?: boolean }) {
  return (
    <button type="button" className={'lembar-aksi' + (bahaya ? ' bahaya' : '')} onClick={onClick} disabled={disabled}>
      {ikon}<span>{children}</span>
    </button>
  )
}
