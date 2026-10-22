import { useEffect, type RefObject } from 'react'

/** Saat form dibuka: geser halaman ke form dan taruh kursor di isian pertama, supaya form tidak
 *  terlewat (mis. layar di-zoom). */
export function useGulirKeForm(ref: RefObject<HTMLElement | null>, terbuka: boolean) {
  useEffect(() => {
    if (!terbuka || !ref.current) return
    ref.current.scrollIntoView({ behavior: 'smooth', block: 'start' })
    ref.current.querySelector<HTMLElement>('input:not([type=hidden]):not([disabled]), select, textarea')?.focus({ preventScroll: true })
  }, [terbuka, ref])
}
