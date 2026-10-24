import { useEffect, useRef } from 'react'
import { ChevronLeft, ChevronRight } from 'lucide-react'

// Tim MasterPajak AI (foto dari mockup final, frontend/public/tim). Rel diputar tanpa ujung:
// kartu digandakan, posisi dibungkus modulo lebar satu putaran.
const TIM: [string, string][] = [
  ['Bayu', 'Founder & CEO'], ['Eko', 'Co-Founder'], ['Fandi', 'Konsultan Pajak Senior'], ['Fadil', 'Konsultan Pajak Senior'],
  ['Helmi', 'Konsultan Pajak'], ['Farhan', 'Konsultan Pajak'], ['Alif', 'Tax Analyst'], ['Andri', 'Tax Analyst'],
  ['Tindar', 'AI Engineer'], ['Randi', 'Customer Success'],
]

const KECEPATAN = 22 // px per detik
const CELAH = 20

export function TimCarousel() {
  const track = useRef<HTMLDivElement>(null)
  const rail = useRef<HTMLDivElement>(null)
  const bar = useRef<HTMLElement>(null)
  const geser = useRef<(arah: number) => void>(() => {})

  useEffect(() => {
    const tr = track.current, rl = rail.current, br = bar.current
    if (!tr || !rl || !br) return
    const reduce = matchMedia('(prefers-reduced-motion: reduce)').matches
    let off = 0, last = 0, hold = false, resumeAt = 0, raf = 0
    let tw: { from: number; to: number; t0: number } | null = null
    let drag: { x: number; o: number } | null = null
    const kartu = () => rl.children as HTMLCollectionOf<HTMLElement>
    const putaran = () => kartu()[TIM.length].offsetLeft - kartu()[0].offsetLeft
    const langkah = () => kartu()[0].getBoundingClientRect().width + CELAH
    const norm = (v: number) => { const w = putaran(); return w > 0 ? ((v % w) + w) % w : 0 }
    const ease = (x: number) => (x < 0.5 ? 4 * x * x * x : 1 - Math.pow(-2 * x + 2, 3) / 2)
    geser.current = (arah) => { tw = { from: off, to: off + arah * langkah(), t0: performance.now() }; resumeAt = performance.now() + 4500 }

    function frame(ts: number) {
      const dt = last ? Math.min(ts - last, 60) : 0
      last = ts
      if (tw) {
        const k = Math.min(1, (ts - tw.t0) / 700)
        off = tw.from + (tw.to - tw.from) * ease(k)
        if (k >= 1) tw = null
      } else if (!drag && !hold && !reduce && ts > resumeAt && !document.hidden) {
        const r = tr!.getBoundingClientRect()
        if (r.bottom > 0 && r.top < innerHeight) off += (KECEPATAN * dt) / 1000
      }
      if (!tw) off = norm(off)
      rl!.style.transform = `translate3d(${(-off).toFixed(2)}px,0,0)`
      br!.style.transform = `translateX(${(off / (putaran() || 1)) * 233.33}%)`
      raf = requestAnimationFrame(frame)
    }
    const masuk = () => { hold = true }
    const keluar = () => { hold = false; resumeAt = performance.now() + 800 }
    const tekan = (e: PointerEvent) => { drag = { x: e.clientX, o: off }; tw = null; tr.setPointerCapture(e.pointerId); tr.dataset.drag = '' }
    const gerak = (e: PointerEvent) => { if (drag) off = drag.o - (e.clientX - drag.x) }
    const lepas = () => { if (drag) { drag = null; delete tr.dataset.drag; resumeAt = performance.now() + 2500 } }
    tr.addEventListener('mouseenter', masuk)
    tr.addEventListener('mouseleave', keluar)
    tr.addEventListener('pointerdown', tekan)
    tr.addEventListener('pointermove', gerak)
    tr.addEventListener('pointerup', lepas)
    tr.addEventListener('pointercancel', lepas)
    raf = requestAnimationFrame(frame)
    return () => {
      cancelAnimationFrame(raf)
      tr.removeEventListener('mouseenter', masuk)
      tr.removeEventListener('mouseleave', keluar)
      tr.removeEventListener('pointerdown', tekan)
      tr.removeEventListener('pointermove', gerak)
      tr.removeEventListener('pointerup', lepas)
      tr.removeEventListener('pointercancel', lepas)
    }
  }, [])

  const kartu = (salinan: boolean) => TIM.map(([nama, peran]) => (
    <figure key={(salinan ? 'b-' : '') + nama} className="lp-tm" aria-hidden={salinan || undefined}>
      <div className="lp-tm-img"><img src={`/tim/${nama.toLowerCase()}.jpg`} alt={salinan ? '' : `${nama}, ${peran}`} width={480} height={600} loading="lazy" draggable={false} /></div>
      <figcaption><b>{nama}</b><span>{peran}</span></figcaption>
    </figure>
  ))

  return (
    <div className="lp-tm-wrap">
      <div ref={track} className="lp-team" tabIndex={0} aria-label="Tim MasterPajak AI, geser untuk melihat semua"
        onKeyDown={(e) => {
          if (e.key === 'ArrowRight') { e.preventDefault(); geser.current(1) }
          if (e.key === 'ArrowLeft') { e.preventDefault(); geser.current(-1) }
        }}>
        <div ref={rail} className="lp-tm-rail">{kartu(false)}{kartu(true)}</div>
      </div>
      <div className="lp-tm-nav">
        <button type="button" className="lp-tm-btn" aria-label="Sebelumnya" onClick={() => geser.current(-1)}><ChevronLeft aria-hidden="true" /></button>
        <div className="lp-tm-bar"><i ref={bar} /></div>
        <button type="button" className="lp-tm-btn" aria-label="Berikutnya" onClick={() => geser.current(1)}><ChevronRight aria-hidden="true" /></button>
      </div>
    </div>
  )
}
