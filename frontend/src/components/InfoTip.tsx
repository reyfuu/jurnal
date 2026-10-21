import { CircleHelp } from 'lucide-react'
import { useId, useState, type ReactNode } from 'react'

/** Tebal: teks bertanda **…** ditampilkan tebal, supaya poin penting di penjelasan mudah ditangkap. */
export function Tebal({ teks }: { teks: string }) {
  return <>{teks.split(/\*\*(.+?)\*\*/g).map((b, i) => (i % 2 ? <strong key={i}>{b}</strong> : b))}</>
}

/** TebalAngka: nominal rupiah dan persen di kalimat ditebalkan (pesan dari server berupa teks biasa). */
export function TebalAngka({ teks }: { teks: string }) {
  return <>{teks.split(/(-?Rp ?[\d.]+(?:,\d+)?|-?\d+(?:[.,]\d+)*%)/g).map((b, i) => (i % 2 ? <strong key={i}>{b}</strong> : b))}</>
}

/**
 * InfoTip: tanda tanya kecil; penjelasan muncul saat kursor di atasnya, saat difokus dengan
 * keyboard, atau saat diketuk di layar sentuh. Esc menutupnya.
 */
export function InfoTip({ label, children }: { label: string; children: ReactNode }) {
  const [buka, setBuka] = useState(false)
  const id = useId()
  return (
    <span className="relative inline-flex align-middle" onMouseEnter={() => setBuka(true)} onMouseLeave={() => setBuka(false)}>
      <button
        type="button"
        aria-label={`Apa itu ${label}?`}
        aria-describedby={buka ? id : undefined}
        aria-expanded={buka}
        className="inline-flex size-6 items-center justify-center rounded-full text-muted-foreground hover:text-foreground focus-visible:outline-2 focus-visible:outline-offset-1"
        onFocus={() => setBuka(true)}
        onBlur={() => setBuka(false)}
        onClick={() => setBuka((b) => !b)}
        onKeyDown={(e) => { if (e.key === 'Escape') setBuka(false) }}
      >
        <CircleHelp size={15} aria-hidden="true" />
      </button>
      {buka && (
        <span role="tooltip" id={id} className="absolute right-0 top-full z-50 mt-1 grid w-80 max-w-[80vw] gap-1.5 rounded-md border border-border bg-popover p-3 text-left text-xs font-normal leading-relaxed text-popover-foreground shadow-lg">
          {children}
        </span>
      )}
    </span>
  )
}
