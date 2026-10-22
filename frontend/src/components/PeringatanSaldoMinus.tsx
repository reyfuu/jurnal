import { useEffect, useState } from 'react'
import { AlertTriangle } from 'lucide-react'
import { getSaldoMinus, type SaldoMinus, type View } from '@/lib/api'

/**
 * Peringatan saldo minus (PSAK): akun yang saldonya berlawanan dengan saldo normal (kas di
 * kredit, utang di debit, …) dan stok minus. Muncul di halaman pencatatan & laporan; dimuat ulang
 * setiap data berubah (versi), mis. sesudah transaksi disimpan.
 */
export function PeringatanSaldoMinus({ period, versi, navigate }: { period: string; versi: unknown; navigate?: (v: View, tab?: string) => void }) {
  const [daftar, setDaftar] = useState<SaldoMinus[]>([])
  useEffect(() => {
    let hidup = true
    getSaldoMinus(period).then((d) => { if (hidup) setDaftar(d) }, () => { if (hidup) setDaftar([]) })
    return () => { hidup = false }
  }, [period, versi])
  if (daftar.length === 0) return null
  return (
    <div className="note-box warn" role="alert" style={{ marginTop: 0, marginBottom: 14 }}>
      <div className="flex items-center gap-2" style={{ fontWeight: 700 }}>
        <AlertTriangle size={16} aria-hidden="true" /> Ada {daftar.length} saldo minus
      </div>
      <ul style={{ margin: '6px 0 0 18px', listStyle: 'disc' }}>
        {daftar.map((m) => (
          <li key={(m.kode ?? '') + m.nama} className="text-sm">
            {m.pesan}
            {navigate && m.jenis === 'akun' && m.kode && (
              <>{' '}<button type="button" className="underline underline-offset-4" onClick={() => navigate('buku-besar', m.kode)}>Lihat Buku Besar {m.kode}</button></>
            )}
          </li>
        ))}
      </ul>
      <p className="text-xs" style={{ marginTop: 6, opacity: 0.85 }}>
        Minus artinya saldo berlawanan dengan saldo normal akunnya. Penjualan di sisi kredit atau biaya di sisi debit itu normal, jadi tidak ikut diperingatkan.
      </p>
    </div>
  )
}
