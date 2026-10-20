import { useState } from 'react'
import { Moon, Sun } from 'lucide-react'
import { pasangTema, temaSekarang } from '@/lib/theme'

/** Tombol ganti tema terang/gelap. `className` mengikuti gaya tombol di sekitarnya. */
export function ThemeToggle({ className, style }: { className?: string; style?: React.CSSProperties }) {
  const [tema, setTema] = useState(temaSekarang)
  const ke = tema === 'gelap' ? 'terang' : 'gelap'
  return (
    <button
      type="button"
      className={className}
      style={style}
      title={`Ganti ke tema ${ke}`}
      aria-label={`Ganti ke tema ${ke}`}
      onClick={() => { pasangTema(ke); setTema(ke) }}
    >
      {tema === 'gelap' ? <Sun className="nav-icon" /> : <Moon className="nav-icon" />}
    </button>
  )
}
