import { useEffect, useRef, useState } from 'react'

// Master, maskot asisten pajak AI (mockup final): kepala, mata, dan lengan mengikuti kursor,
// bergoyang pelan saat diam; gelembung kata berganti tiap 3,8 detik.
const PESAN = ['Halo! Saya Master, asisten pajak AI Anda.', 'Nota hari ini sudah saya bukukan.', 'PPN bulan ini siap dilaporkan.', 'Ada pertanyaan pajak? Tanya saja, 24/7.']

export function RobotMaster() {
  const svg = useRef<SVGSVGElement>(null)
  const head = useRef<SVGGElement>(null)
  const eyes = useRef<SVGGElement>(null)
  const smile = useRef<SVGPathElement>(null)
  const armL = useRef<SVGGElement>(null)
  const armR = useRef<SVGGElement>(null)
  const [k, setK] = useState(0)
  const [keluar, setKeluar] = useState(false)

  useEffect(() => {
    if (matchMedia('(prefers-reduced-motion: reduce)').matches) return
    let tx = 0, ty = 0, cx = 0, cy = 0, aktif = false, idle = 0, raf = 0
    function arah(x: number, y: number) {
      const r = svg.current!.getBoundingClientRect()
      tx = Math.max(-1, Math.min(1, (x - (r.left + r.width * 0.5)) / (innerWidth * 0.5)))
      ty = Math.max(-1, Math.min(1, (y - (r.top + r.height * 0.3)) / (innerHeight * 0.5)))
      aktif = true
      idle = performance.now()
    }
    const mouse = (e: MouseEvent) => arah(e.clientX, e.clientY)
    const touch = (e: TouchEvent) => { if (e.touches[0]) arah(e.touches[0].clientX, e.touches[0].clientY) }
    const pergi = () => { tx = 0; ty = 0 }
    function loop(ts: number) {
      if (!aktif || ts - idle > 4000) { tx = Math.sin(ts / 2600) * 0.35; ty = Math.sin(ts / 3700) * 0.15 }
      cx += (tx - cx) * 0.08
      cy += (ty - cy) * 0.08
      head.current?.setAttribute('transform', `rotate(${(cx * 7).toFixed(2)} 200 210) translate(${(cx * 6).toFixed(2)} ${(cy * 4).toFixed(2)})`)
      eyes.current?.setAttribute('transform', `translate(${(cx * 14).toFixed(2)} ${(cy * 9).toFixed(2)})`)
      smile.current?.setAttribute('transform', `translate(${(cx * 9).toFixed(2)} ${(cy * 5).toFixed(2)})`)
      armL.current?.setAttribute('transform', `rotate(${(4 - cx * 3).toFixed(2)} 91 244)`)
      armR.current?.setAttribute('transform', `rotate(${(-4 - cx * 3).toFixed(2)} 309 244)`)
      raf = requestAnimationFrame(loop)
    }
    addEventListener('mousemove', mouse, { passive: true })
    addEventListener('touchmove', touch, { passive: true })
    document.addEventListener('mouseleave', pergi)
    raf = requestAnimationFrame(loop)
    let t = 0
    const iv = setInterval(() => {
      setKeluar(true)
      t = window.setTimeout(() => { setK((x) => (x + 1) % PESAN.length); setKeluar(false) }, 380)
    }, 3800)
    return () => {
      cancelAnimationFrame(raf)
      clearInterval(iv)
      clearTimeout(t)
      removeEventListener('mousemove', mouse)
      removeEventListener('touchmove', touch)
      document.removeEventListener('mouseleave', pergi)
    }
  }, [])

  const logo = (warna: string) => <>
    <polyline points="20,78 20,24 50,56" fill="none" stroke={warna} strokeWidth="13" strokeLinecap="round" strokeLinejoin="round" />
    <polyline points="50,56 80,24 80,78" fill="none" stroke="#FED111" strokeWidth="13" strokeLinecap="round" strokeLinejoin="round" />
  </>

  return (
    <div className="lp-meet-vis">
      <div className="lp-meet-orb" aria-hidden="true" />
      <div className="lp-meet-bub" data-out={keluar || undefined} aria-live="polite">{PESAN[k]}</div>
      <svg ref={svg} className="lp-bot" viewBox="0 0 400 460" role="img" aria-label="Master, asisten pajak AI MasterPajak AI">
        <defs>
          <linearGradient id="bot-body" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stopColor="#3A4889" /><stop offset=".55" stopColor="#212C5F" /><stop offset="1" stopColor="#131B40" /></linearGradient>
          <linearGradient id="bot-head" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stopColor="#FFFFFF" /><stop offset=".7" stopColor="#EEF0F7" /><stop offset="1" stopColor="#C9CFE3" /></linearGradient>
          <linearGradient id="bot-visor" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stopColor="#1A234B" /><stop offset="1" stopColor="#0B1030" /></linearGradient>
          <linearGradient id="bot-gloss" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stopColor="#fff" stopOpacity=".22" /><stop offset="1" stopColor="#fff" stopOpacity="0" /></linearGradient>
          <radialGradient id="bot-shadow" cx=".5" cy=".5" r=".5"><stop offset="0" stopColor="#131B40" stopOpacity=".28" /><stop offset="1" stopColor="#131B40" stopOpacity="0" /></radialGradient>
          <filter id="bot-glow" x="-50%" y="-50%" width="200%" height="200%"><feGaussianBlur stdDeviation="4" result="b" /><feMerge><feMergeNode in="b" /><feMergeNode in="SourceGraphic" /></feMerge></filter>
        </defs>
        <ellipse className="lp-bot-floor" cx="200" cy="432" rx="110" ry="16" fill="url(#bot-shadow)" />
        <g className="lp-bot-all">
          <g ref={armL}><rect x="74" y="236" width="34" height="112" rx="17" fill="url(#bot-body)" /><circle cx="91" cy="352" r="15" fill="#DDE1EE" /></g>
          <g ref={armR}><rect x="292" y="236" width="34" height="112" rx="17" fill="url(#bot-body)" /><circle cx="309" cy="352" r="15" fill="#DDE1EE" /></g>
          <rect x="112" y="222" width="176" height="168" rx="46" fill="url(#bot-body)" />
          <rect x="112" y="222" width="176" height="70" rx="40" fill="url(#bot-gloss)" />
          <rect x="160" y="252" width="80" height="76" rx="18" fill="#131B40" opacity=".55" />
          <g transform="translate(176,262) scale(.48)">{logo('#fff')}</g>
          <circle cx="200" cy="356" r="5" fill="#FED111" className="lp-bot-led" />
          <rect x="182" y="206" width="36" height="22" rx="8" fill="#131B40" />
          <g ref={head}>
            <line x1="200" y1="62" x2="200" y2="38" stroke="#C9CFE3" strokeWidth="5" strokeLinecap="round" />
            <circle cx="200" cy="32" r="9" fill="#FED111" filter="url(#bot-glow)" className="lp-bot-led" />
            <rect x="84" y="112" width="20" height="56" rx="10" fill="#212C5F" />
            <rect x="296" y="112" width="20" height="56" rx="10" fill="#212C5F" />
            <rect x="96" y="58" width="208" height="160" rx="58" fill="url(#bot-head)" />
            <rect x="118" y="88" width="164" height="96" rx="40" fill="url(#bot-visor)" />
            <rect x="128" y="94" width="144" height="30" rx="15" fill="url(#bot-gloss)" />
            <g ref={eyes} filter="url(#bot-glow)">
              <rect className="lp-bot-eye" x="152" y="122" width="26" height="34" rx="13" fill="#FED111" />
              <rect className="lp-bot-eye" x="222" y="122" width="26" height="34" rx="13" fill="#FED111" />
            </g>
            <path ref={smile} d="M186 168 Q200 176 214 168" fill="none" stroke="#FED111" strokeWidth="4" strokeLinecap="round" opacity=".85" />
          </g>
        </g>
      </svg>
    </div>
  )
}
