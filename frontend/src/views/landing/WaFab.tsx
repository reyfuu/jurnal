import { useEffect, useState } from 'react'

// Tombol WhatsApp mengambang. Isi WA_NUMBER (format 62xxxxxxxxxx) untuk menampilkannya.
export const WA_NUMBER = ''
const PESAN = 'Halo MasterPajak AI, saya ingin bertanya tentang paket langganan.'

export function WaFab() {
  const [intip, setIntip] = useState(false)
  useEffect(() => {
    if (!WA_NUMBER) return
    let t2 = 0
    const t1 = setTimeout(() => { setIntip(true); t2 = window.setTimeout(() => setIntip(false), 4000) }, 3000)
    return () => { clearTimeout(t1); clearTimeout(t2) }
  }, [])
  if (!WA_NUMBER) return null
  return (
    <a className="lp-wa" data-intip={intip || undefined} href={`https://wa.me/${WA_NUMBER}?text=${encodeURIComponent(PESAN)}`}
      target="_blank" rel="noopener noreferrer" aria-label="Hubungi kami via WhatsApp">
      <span className="lp-wa-tip">Butuh bantuan? <b>Chat kami</b></span>
      <span className="lp-wa-ic">
        <svg viewBox="0 0 32 32" aria-hidden="true"><path fill="currentColor" d="M16.04 3C8.86 3 3.03 8.82 3.03 16c0 2.29.6 4.53 1.74 6.5L3 29l6.68-1.75A12.97 12.97 0 0 0 16.04 29C23.2 29 29 23.18 29 16S23.2 3 16.04 3zm0 23.7c-2.02 0-4-.54-5.72-1.57l-.41-.24-3.96 1.04 1.06-3.86-.27-.4A10.68 10.68 0 0 1 5.33 16c0-5.9 4.8-10.7 10.71-10.7 5.9 0 10.66 4.8 10.66 10.7 0 5.9-4.77 10.7-10.66 10.7zm5.87-8.02c-.32-.16-1.9-.94-2.2-1.04-.3-.11-.51-.16-.73.16-.21.32-.83 1.04-1.02 1.26-.19.21-.38.24-.7.08-.32-.16-1.36-.5-2.59-1.6-.96-.85-1.6-1.9-1.79-2.22-.19-.32-.02-.5.14-.66.14-.14.32-.38.48-.56.16-.19.21-.32.32-.54.1-.21.05-.4-.03-.56-.08-.16-.73-1.75-1-2.4-.26-.63-.53-.54-.73-.55h-.62c-.21 0-.56.08-.86.4-.29.32-1.12 1.1-1.12 2.68 0 1.58 1.15 3.1 1.31 3.32.16.21 2.26 3.45 5.48 4.84.77.33 1.37.53 1.83.68.77.24 1.47.21 2.03.13.62-.09 1.9-.78 2.17-1.53.27-.75.27-1.4.19-1.53-.08-.13-.29-.21-.61-.37z" /></svg>
      </span>
    </a>
  )
}
