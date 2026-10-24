import { useEffect, type RefObject } from 'react'

// Animasi muncul saat di-scroll (mockup final). Penanda lewat atribut data-rv / data-rv-in, bukan
// className, supaya tidak tertimpa React. Tanpa IntersectionObserver atau dengan reduced motion
// semuanya langsung tampil.
const GRUP: [string, string][] = [
  ['.lp-hero-txt > *', ''], ['.lp-vid', 'r'], ['.lp-sl-k', ''], ['.lp-sl-t', 's'],
  ['main > section:not(.lp-hero):not(.lp-slogan) .lp-kicker', ''], ['main > section:not(.lp-hero):not(.lp-slogan) .lp-h2', ''],
  ['main > section:not(.lp-hero):not(.lp-slogan) .lp-lead', ''],
  ['.lp-frame', 's'], ['.lp-caption', ''], ['.lp-persona', 'l'], ['.lp-reassure', ''], ['.lp-chain li', 'l'], ['.lp-tm-wrap', ''],
  ['.lp-stat', ''], ['.lp-ai-pt', ''], ['.lp-chat', 's'], ['.lp-klinik-img', 'l'], ['.lp-klinik-txt', 'r'],
  ['.lp-cmp', 's'], ['.lp-form', ''], ['.lp-incl li', ''], ['.lp-refund', ''], ['.lp-choice', ''], ['.lp-ps', ''], ['.lp details', ''],
]

export function useReveal(root: RefObject<HTMLElement | null>) {
  useEffect(() => {
    const el = root.current
    if (!el) return
    const hero = el.querySelector('.lp-hero mark')
    const tampilSemua = () => {
      hero?.setAttribute('data-on', '')
      el.querySelector('.lp-sl-t')?.setAttribute('data-rv-in', '')
    }
    if (matchMedia('(prefers-reduced-motion: reduce)').matches || !('IntersectionObserver' in window)) {
      tampilSemua()
      return
    }
    el.setAttribute('data-anim', '')
    const els: Element[] = []
    for (const [sel, arah] of GRUP) {
      el.querySelectorAll(sel).forEach((x) => {
        if (x.hasAttribute('data-rv')) return
        x.setAttribute('data-rv', arah)
        const i = x.parentElement ? Array.prototype.indexOf.call(x.parentElement.children, x) : 0
        ;(x as HTMLElement).style.setProperty('--d', `${Math.min(i, 6) * 0.08}s`)
        els.push(x)
      })
    }
    const io = new IntersectionObserver((es) => es.forEach((e) => {
      if (e.isIntersecting) { e.target.setAttribute('data-rv-in', ''); io.unobserve(e.target) }
    }), { threshold: 0.12, rootMargin: '0px 0px -8% 0px' })
    els.forEach((x) => io.observe(x))
    const t1 = setTimeout(() => hero?.setAttribute('data-on', ''), 150)
    // pengaman: konten yang sudah di layar jangan sampai tetap tersembunyi
    const t2 = setTimeout(() => els.forEach((x) => { if (x.getBoundingClientRect().top < innerHeight) x.setAttribute('data-rv-in', '') }), 2500)
    return () => { io.disconnect(); clearTimeout(t1); clearTimeout(t2); el.removeAttribute('data-anim') }
  }, [root])
}
