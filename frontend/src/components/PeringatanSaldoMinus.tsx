import { useEffect, useState } from 'react'
import { AlertTriangle, ChevronDown, ChevronUp } from 'lucide-react'
import { getSaldoMinus, type SaldoMinus } from '@/lib/api'
import { useHP } from '@/lib/hp'

/**
 * Peringatan saldo minus (PSAK): akun yang saldonya berlawanan dengan saldo normal (kas di
 * kredit, utang di debit, …) dan stok minus. Muncul di halaman pencatatan & laporan; dimuat ulang
 * setiap data berubah (versi), mis. sesudah transaksi disimpan.
 */
export function PeringatanSaldoMinus({ period, versi }: { period: string; versi: unknown }) {
  const [daftar, setDaftar] = useState<SaldoMinus[]>([])
  const hp = useHP()
  const [bukaHP, setBukaHP] = useState(false) // di HP terlipat supaya tidak menutupi halaman
  useEffect(() => {
    let hidup = true
    getSaldoMinus(period).then((d) => { if (hidup) setDaftar(d) }, () => { if (hidup) setDaftar([]) })
    return () => { hidup = false }
  }, [period, versi])
  if (daftar.length === 0) return null
  const judul = <><AlertTriangle size={16} aria-hidden="true" /> Ada {daftar.length} saldo minus</>
  if (hp && !bukaHP) {
    return (
      <button type="button" className="note-box warn flex w-full items-center gap-2" role="alert" aria-expanded={false}
        style={{ marginTop: 0, marginBottom: 12, fontWeight: 700, textAlign: 'left' }} onClick={() => setBukaHP(true)}>
        {judul}<ChevronDown size={16} aria-hidden="true" style={{ marginLeft: 'auto' }} />
      </button>
    )
  }
  return (
    <div className="note-box warn" role="alert" style={{ marginTop: 0, marginBottom: 14 }}>
      {hp
        ? <button type="button" className="flex w-full items-center gap-2" style={{ fontWeight: 700 }} aria-expanded onClick={() => setBukaHP(false)}>
            {judul}<ChevronUp size={16} aria-hidden="true" style={{ marginLeft: 'auto' }} />
          </button>
        : <div className="flex items-center gap-2" style={{ fontWeight: 700 }}>{judul}</div>}
      <ul style={{ margin: '6px 0 0 18px', listStyle: 'disc' }}>
        {daftar.map((m) => <li key={(m.kode ?? '') + m.nama} className="text-sm">{m.pesan}</li>)}
      </ul>
      <p className="text-xs" style={{ marginTop: 6, opacity: 0.85 }}>
        Minus artinya saldo berlawanan dengan saldo normal akunnya. Penjualan di sisi kredit atau biaya di sisi debit itu normal, jadi tidak ikut diperingatkan.
      </p>
    </div>
  )
}
