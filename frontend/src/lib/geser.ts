import { useRef, useState, type PointerEvent } from 'react'

/** Jarak tombol mengambang dari tepi kanan & bawah layar (px). */
export interface Posisi { right: number; bottom: number }

const BAWAAN: Posisi = { right: 24, bottom: 24 }

/** jepit: posisi tetap di dalam layar (sisa minimal 8px, tombol selebar lebar × tinggi). */
export function jepit(p: Posisi, layar: { w: number; h: number }, tombol = { w: 180, h: 48 }): Posisi {
  return {
    right: Math.min(Math.max(8, p.right), Math.max(8, layar.w - tombol.w - 8)),
    bottom: Math.min(Math.max(8, p.bottom), Math.max(8, layar.h - tombol.h - 8)),
  }
}

function baca(kunci: string): Posisi {
  try {
    const p = JSON.parse(localStorage.getItem(kunci) ?? '') as Posisi
    if (typeof p.right === 'number' && typeof p.bottom === 'number') return jepit(p, { w: window.innerWidth, h: window.innerHeight })
  } catch { /* belum pernah digeser / storage diblokir */ }
  return BAWAAN
}

/**
 * useGeser: tombol mengambang yang bisa digeser dengan mouse / jari. Posisi diingat per
 * perangkat. `diklik()` false bila pointer baru saja menggeser, supaya geser tidak membuka panel.
 */
export function useGeser(kunci: string) {
  const [pos, setPos] = useState<Posisi>(() => baca(kunci))
  const awal = useRef<{ x: number; y: number; p: Posisi; geser: boolean } | null>(null)
  const barusGeser = useRef(false)

  function onPointerDown(e: PointerEvent<HTMLElement>) {
    if (e.button !== 0) return
    awal.current = { x: e.clientX, y: e.clientY, p: pos, geser: false }
    barusGeser.current = false
    e.currentTarget.setPointerCapture(e.pointerId)
  }
  function onPointerMove(e: PointerEvent<HTMLElement>) {
    const a = awal.current
    if (!a) return
    const dx = e.clientX - a.x
    const dy = e.clientY - a.y
    if (!a.geser && Math.hypot(dx, dy) < 6) return
    a.geser = true
    const r = e.currentTarget.getBoundingClientRect()
    setPos(jepit({ right: a.p.right - dx, bottom: a.p.bottom - dy }, { w: window.innerWidth, h: window.innerHeight }, { w: r.width, h: r.height }))
  }
  function onPointerUp() {
    const a = awal.current
    awal.current = null
    if (!a?.geser) return
    barusGeser.current = true
    setPos((p) => {
      try { localStorage.setItem(kunci, JSON.stringify(p)) } catch { /* abaikan */ }
      return p
    })
  }
  const diklik = () => {
    const g = barusGeser.current
    barusGeser.current = false
    return !g
  }
  return { pos, diklik, handlers: { onPointerDown, onPointerMove, onPointerUp, onPointerCancel: onPointerUp } }
}
