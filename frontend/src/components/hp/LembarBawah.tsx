import { useEffect, useRef, type ReactNode } from 'react'
import { createPortal } from 'react-dom'

/** Panel yang muncul dari bawah layar (HP): pilihan aksi atau filter. Ketuk latar / Escape menutup. */
export function LembarBawah({ buka, judul, onTutup, children }: { buka: boolean; judul: ReactNode; onTutup: () => void; children: ReactNode }) {
  const panel = useRef<HTMLDivElement>(null)
  // onTutup lewat ref: pemanggil mengirim fungsi baru tiap render, dan effect di bawah tidak boleh
  // berjalan ulang (fokus akan melompat dari isian ke panel setiap kali induk dirender ulang).
  const tutup = useRef(onTutup)
  useEffect(() => { tutup.current = onTutup })
  useEffect(() => {
    if (!buka) return
    const kembaliKe = document.activeElement as HTMLElement | null
    const esc = (e: KeyboardEvent) => { if (e.key === 'Escape') tutup.current() }
    window.addEventListener('keydown', esc)
    const lama = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    document.body.classList.add('lembar-terbuka')
    panel.current?.focus()
    return () => {
      window.removeEventListener('keydown', esc)
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
