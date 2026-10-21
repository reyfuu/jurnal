import { useState } from 'react'
import { Dua, Hasil, Persen, Pilih, TombolHitung, Uang } from '@/components/KalkulatorUI'
import { hitungMarketplace, type BusinessProfile, type KalkulatorMaster, type MarketplaceResult } from '@/lib/api'
import { formatRp } from '@/lib/format'
import { GREEN, RED, SOROT, keBps, persen } from '@/lib/kalkulator'

// PPh 22 marketplace (PMK 37/2025): berapa yang dipungut marketplace, apakah jadi kredit pajak
// atau pelunasan PPh final, dan saran (SKB, surat pernyataan) dari server. Dipakai di Alat Gratis
// (dengan ajakan berlangganan) dan Kalkulator Pajak.

export function FormMarketplace({ m, profile, onLangganan }: { m: KalkulatorMaster; profile: BusinessProfile | null; onLangganan?: () => void }) {
  const omzetProfil = profile?.omzet_tahunan || 0
  const npmProfil = omzetProfil > 0 && profile?.laba_sebelum_pajak != null ? (profile.laba_sebelum_pajak / omzetProfil) * 100 : 5
  const [omzet, setOmzet] = useState(omzetProfil || 1_000_000_000)
  const [lain, setLain] = useState(0)
  const [npm, setNpm] = useState(String(Math.round(npmProfil * 10) / 10).replace('.', ','))
  // Margin bisa diisi rupiah atau persen; yang terakhir diketik jadi acuan, yang lain dihitung
  // dari total omzet (sama dengan server: laba = total omzet x margin).
  const [acuan, setAcuan] = useState<'persen' | 'rupiah'>('persen')
  const [labaRp, setLabaRp] = useState(0)
  const total = omzet + lain
  const laba = acuan === 'rupiah' ? labaRp : Math.round((total * keBps(npm)) / 10_000)
  const npmTampil = acuan === 'persen' ? npm
    : total > 0 ? String(Math.round((labaRp / total) * 10_000) / 100).replace('.', ',') : '0'
  const [bentuk, setBentuk] = useState<'op' | 'badan'>(!profile || profile.bentuk_usaha === 'op' ? 'op' : 'badan')
  const [skema, setSkema] = useState<'final' | 'umum'>(
    profile && (profile.bentuk_usaha === 'cv' || profile.bentuk_usaha === 'pt') ? 'umum' : 'final')
  const [ptkp, setPtkp] = useState(profile?.bentuk_usaha === 'op' && profile.ptkp ? profile.ptkp : 'TK/0')
  const [r, setR] = useState<MarketplaceResult | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  async function hitung(e: React.FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      setR(await hitungMarketplace({ omzet_marketplace: omzet, omzet_lain: lain, npm_bps: keBps(npmTampil), laba: acuan === 'rupiah' ? labaRp : undefined, bentuk, skema, ptkp }))
    } catch (err) {
      setR(null)
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <>
      <Dua
        kiri={
          <form onSubmit={hitung}>
            <div className="chart-title">PPh 22 Marketplace</div>
            <div className="chart-subtitle">Marketplace memungut PPh 22 dari omzet penjual. Cek apakah pungutan itu melebihi pajak Anda.</div>
            {omzetProfil > 0 && <p className="kpi-meta-row" style={{ marginBottom: 8 }}>Omzet dan margin laba awal dari profil usaha Anda; ubah untuk mencoba angka lain.</p>}
            <Uang label="Omzet lewat marketplace setahun" v={omzet} set={setOmzet} hint="Shopee, Tokopedia, TikTok Shop, dll." />
            <Uang label="Omzet di luar marketplace setahun" v={lain} set={setLain} hint="Toko, reseller, transfer langsung" />
            <div className="grid gap-x-3 sm:grid-cols-2">
              <Uang label="Laba bersih setahun (Rp)" v={laba} set={(n) => { setAcuan('rupiah'); setLabaRp(n) }}
                hint={acuan === 'persen' ? 'dihitung dari margin %' : 'diisi manual'} />
              <Persen label="atau margin laba bersih (%)" v={npmTampil} set={(v) => { setAcuan('persen'); setNpm(v) }}
                hint={acuan === 'rupiah' ? 'dihitung dari laba Rp' : 'Laba bersih ÷ total omzet'} />
            </div>
            <p style={{ fontSize: 11.5, color: 'var(--text-muted)', margin: '-4px 0 12px' }}>
              Isi salah satu, yang lain terisi otomatis dari total omzet {formatRp(total)}. Margin kecil = pungutan bisa melebihi pajak.
            </p>
            <Pilih label="Usaha terdaftar sebagai" v={bentuk} set={(v) => setBentuk(v as 'op' | 'badan')} opsi={[['op', 'Orang pribadi'], ['badan', 'Badan usaha (PT / CV)']]} />
            <Pilih label="Cara bayar pajak penghasilan" v={skema} set={(v) => setSkema(v as 'final' | 'umum')}
              opsi={[['final', 'PPh final 0,5% dari omzet (UMKM)'], ['umum', 'Pembukuan, pajak dari laba (tarif umum)']]} />
            {bentuk === 'op' && <Pilih label="Status PTKP" v={ptkp} set={setPtkp} opsi={m.ptkp_codes.map((c) => [c, `${c} — ${formatRp(m.ptkp[c])}`])} />}
            <TombolHitung busy={busy} error={error} />
          </form>
        }
        kanan={r && (
          <>
            <Hasil
              utama={[
                { label: 'PPh 22 dipungut marketplace', v: r.dipungut, warna: SOROT },
                { label: r.final ? 'PPh final terutang' : 'PPh terutang (pembukuan)', v: r.terutang },
                r.lebih_bayar > 0
                  ? { label: r.final ? 'Pungutan melebihi PPh final' : 'Lebih bayar (kredit pajak berlebih)', v: r.lebih_bayar, warna: RED }
                  : { label: 'Sisa disetor sendiri', v: r.kurang_bayar, warna: GREEN },
              ]}
              langkah={r.langkah}
            />
            {r.npm_impas_bps >= 0 && (
              <p className="kpi-meta-row" style={{ marginTop: 10 }}>
                Dengan pembukuan, pungutan marketplace melebihi pajak Anda bila margin laba di bawah <b>{persen(r.npm_impas_bps)}</b>.
              </p>
            )}
          </>
        )}
      />
      {r && r.rekomendasi.length > 0 && (
        <div className="card" style={{ padding: 20, borderColor: 'color-mix(in srgb, var(--text-brand) 35%, transparent)' }}>
          <div className="chart-title">Rekomendasi untuk usaha Anda</div>
          <div className="grid gap-4" style={{ marginTop: 12 }}>
            {r.rekomendasi.map((s) => (
              <div key={s.judul} className="border-l-2 pl-3" style={{ borderColor: s.langganan ? 'var(--text-brand)' : 'var(--border-strong)' }}>
                <div className="text-sm font-bold">{s.judul}</div>
                <p className="mt-1 text-sm leading-relaxed text-muted-foreground">{s.isi}</p>
                {s.langganan && onLangganan && (
                  <button type="button" className="btn-cta accent" style={{ marginTop: 10, padding: '8px 18px', fontSize: 13 }} onClick={onLangganan}>
                    Isi data &amp; berlangganan
                  </button>
                )}
              </div>
            ))}
          </div>
          <ul className="grid gap-1" style={{ marginTop: 14 }}>
            {r.catatan.map((c) => <li key={c} className="text-xs text-muted-foreground">{c}</li>)}
          </ul>
        </div>
      )}
    </>
  )
}
