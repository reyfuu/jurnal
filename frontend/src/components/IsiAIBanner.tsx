import { Button } from '@/components/ui/button'

/** Penanda di dalam form yang baru diisi AI, dengan tombol Urungkan (kembalikan isi form
 *  seperti sebelum diisi). Dialog modal memblokir bubble AI, jadi tombolnya ada di sini. */
export function IsiAIBanner({ ai }: { ai: { aktif: number | null; urungkan: () => void } }) {
  if (ai.aktif === null) return null
  return (
    <div className="note-box" role="status" style={{ margin: 0, display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 12 }}>
      <span><span className="pill-status pill-brand">Diisi AI</span> Periksa semua kolom sebelum menyimpan.</span>
      <Button type="button" variant="outline" size="sm" onClick={ai.urungkan}>Urungkan</Button>
    </div>
  )
}
