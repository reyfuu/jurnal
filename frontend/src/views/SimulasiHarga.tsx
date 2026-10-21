import { useEffect, useMemo, useState } from 'react'
import { Info } from 'lucide-react'
import { PageHeader } from '@/components/PageHeader'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { getProduksi, simulasiUsaha, simulatePrice, type PriceSim, type ProdukRow, type Skenario, type SimulasiUsaha } from '@/lib/api'
import { formatRp } from '@/lib/format'
import { kalimatPerubahan, labelPeriode } from '@/lib/simulasi'
import { angka } from '@/lib/angka'

const fieldClass = 'h-[38px] bg-[var(--bg-input)] tabular'
const NOL: Skenario = { harga_bps: 0, volume_bps: 0, modal_bps: 0, tambah_tetap: 0 }
const SKENARIO: [string, Skenario][] = [
  ['Naik harga 5%', { ...NOL, harga_bps: 500 }],
  ['Naik harga 10%, pembeli turun 5%', { ...NOL, harga_bps: 1000, volume_bps: -500 }],
  ['Hemat bahan 10%', { ...NOL, modal_bps: -1000 }],
  ['Tambah 1 karyawan (Rp3.000.000)', { ...NOL, tambah_tetap: 3_000_000 }],
  ['Jualan naik 20%', { ...NOL, volume_bps: 2000 }],
]
// Komisi contoh; tetap bisa diubah di kolom Komisi.
const SALURAN: [string, string, number][] = [['toko', 'Toko / Offline', 0], ['gofood', 'GoFood', 20], ['grabfood', 'GrabFood', 20], ['shopeefood', 'ShopeeFood', 20], ['lain', 'Lainnya (isi sendiri)', 0]]
const pct = (bps: number) => `${(bps / 100).toFixed(0)}%`

/** Tunda f sampai nilai berhenti berubah ms milidetik (slider memicu banyak perubahan). */
function useTunda<T>(nilai: T, ms: number) {
  const [v, setV] = useState(nilai)
  useEffect(() => {
    const t = setTimeout(() => setV(nilai), ms)
    return () => clearTimeout(t)
  }, [nilai, ms])
  return v
}

function Slider({ id, label, nilai, min, max, step, tampil, onChange }: {
  id: string; label: string; nilai: number; min: number; max: number; step: number; tampil: string; onChange: (v: number) => void
}) {
  return (
    <div className="grid gap-1.5">
      <div className="flex items-center justify-between text-sm"><Label htmlFor={id}>{label}</Label><span className="tabular font-semibold">{tampil}</span></div>
      <input id={id} type="range" min={min} max={max} step={step} value={nilai} onChange={(e) => onChange(Number(e.target.value))}
        style={{ width: '100%', accentColor: 'var(--accent-amber)' }} />
    </div>
  )
}

function Baris({ label, nilai, warna }: { label: string; nilai: string; warna?: string }) {
  return <div className="list-row"><span className="text-muted-foreground">{label}</span><strong className="tabular" style={warna ? { color: warna } : undefined}>{nilai}</strong></div>
}

export function SimulasiHarga({ period }: { period: string }) {
  const [sk, setSk] = useState<Skenario>(NOL)
  const [sim, setSim] = useState<SimulasiUsaha | null>(null)
  const [error, setError] = useState('')
  const skTunda = useTunda(sk, 150)
  useEffect(() => {
    simulasiUsaha(period, skTunda).then((r) => { setSim(r); setError('') }, (e: Error) => setError(e.message))
  }, [period, skTunda])
  const set = (p: Partial<Skenario>) => setSk((s) => ({ ...s, ...p }))
  const d = sim?.dasar, h = sim?.hasil, bep = sim?.titik_impas

  return (
    <>
      <PageHeader title="Simulasi & Harga" subtitle="Kalau harga naik, biaya turun, karyawan nambah — laba jadi berapa, dan berapa harga jual per produk." />
      {error && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start' }}>{error}</div>}
      {d && (
        <div className="card text-sm" style={{ padding: '12px 18px', display: 'flex', flexDirection: 'row', alignItems: 'center', justifyContent: 'flex-start', textAlign: 'left', gap: 12 }}>
          <Info size={16} style={{ color: 'var(--accent-amber)', flexShrink: 0 }} />
          {d.periode.length
            ? <span>Dasar hitung: rata-rata <strong>{labelPeriode(d.periode)}</strong>. Omzet {formatRp(d.omzet)}, laba {formatRp(d.laba)} per bulan.</span>
            : <span>Belum ada penjualan tercatat. Catat transaksi dulu supaya simulasi memakai angka usaha Anda.</span>}
        </div>
      )}

      <div className="grid gap-4" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(100%, 460px), 1fr))', alignItems: 'stretch' }}>
        <div className="card" style={{ padding: 22, display: 'grid', gridTemplateColumns: 'minmax(0, 1fr)', gap: 16, alignContent: 'start' }}>
          <div>
            <div className="chart-title">Kalau gini, gimana?</div>
            <div className="chart-subtitle">Geser dan lihat dampaknya ke laba bulanan.</div>
          </div>
          <div className="flex flex-wrap gap-2">
            {SKENARIO.map(([l, s]) => (
              <button key={l} type="button" className="pill-status pill-status-muted" style={{ padding: '6px 12px', cursor: 'pointer' }} onClick={() => setSk(s)}>{l}</button>
            ))}
            <button type="button" className="pill-status pill-status-muted" style={{ padding: '6px 12px', cursor: 'pointer' }} onClick={() => setSk(NOL)}>Reset</button>
          </div>
          <Slider id="sl-harga" label="Harga jual" nilai={sk.harga_bps} min={-5000} max={5000} step={100} tampil={`${sk.harga_bps > 0 ? '+' : ''}${pct(sk.harga_bps)}`} onChange={(v) => set({ harga_bps: v })} />
          <Slider id="sl-volume" label="Jumlah pembeli / porsi terjual" nilai={sk.volume_bps} min={-5000} max={10000} step={100} tampil={`${sk.volume_bps > 0 ? '+' : ''}${pct(sk.volume_bps)}`} onChange={(v) => set({ volume_bps: v })} />
          <Slider id="sl-modal" label="Modal bahan per porsi" nilai={sk.modal_bps} min={-5000} max={5000} step={100} tampil={`${sk.modal_bps > 0 ? '+' : ''}${pct(sk.modal_bps)}`} onChange={(v) => set({ modal_bps: v })} />
          <Slider id="sl-tetap" label="Tambah biaya tetap per bulan" nilai={sk.tambah_tetap} min={0} max={50_000_000} step={500_000} tampil={formatRp(sk.tambah_tetap)} onChange={(v) => set({ tambah_tetap: v })} />
        </div>

        <div className="card" style={{ padding: 22, display: 'grid', gridTemplateColumns: 'minmax(0, 1fr)', gap: 10, alignContent: 'start', background: 'linear-gradient(160deg, color-mix(in srgb, var(--accent-amber) 9%, var(--bg-card)), var(--bg-card) 55%)' }}>
          <div className="kpi-label" style={{ color: 'var(--accent-amber)', letterSpacing: '0.08em' }}>LABA PER BULAN JADI</div>
          {h && d && (
            <>
              <div className="tabular" style={{ fontSize: 'clamp(30px, 5vw, 44px)', fontWeight: 800, lineHeight: 1.1, color: h.laba < 0 ? 'var(--accent-red)' : undefined }}>{formatRp(h.laba)}</div>
              <p className="text-sm">{kalimatPerubahan(d.laba, h.laba)}</p>
              <div className="list-rows">
                <Baris label="Omzet" nilai={formatRp(h.omzet)} />
                <Baris label="Modal bahan" nilai={formatRp(h.modal_bahan)} />
                <Baris label="Biaya variabel (komisi aplikasi)" nilai={formatRp(h.biaya_variabel)} />
                <Baris label="Biaya tetap" nilai={formatRp(h.biaya_tetap)} />
                {h.penyusutan > 0 && <Baris label="Penyusutan aset" nilai={formatRp(h.penyusutan)} />}
                <Baris label="Margin laba" nilai={`${(h.margin_bps / 100).toFixed(0)}%`} warna={h.margin_bps < 0 ? 'var(--accent-red)' : undefined} />
                <Baris label="Titik impas baru" nilai={bep?.ada ? `${formatRp(bep.per_bulan)} / bln` : 'Tidak tercapai'} warna={bep?.ada ? undefined : 'var(--accent-red)'} />
              </div>
            </>
          )}
        </div>

        <div className="card" style={{ padding: 22, display: 'grid', gridTemplateColumns: 'minmax(0, 1fr)', gap: 10, alignContent: 'start' }}>
          <div>
            <div className="chart-title">Titik impas (BEP)</div>
            <div className="chart-subtitle">Omzet minimal supaya tidak rugi, dari {sk === NOL ? 'rata-rata biaya Anda' : 'skenario di atas'}.</div>
          </div>
          {bep && (bep.ada ? (
            <>
              <div className="grid gap-3" style={{ gridTemplateColumns: '1fr 1fr' }}>
                <div><div className="text-xs text-muted-foreground">Per bulan</div><div className="tabular" style={{ fontSize: 26, fontWeight: 800 }}>{formatRp(bep.per_bulan)}</div></div>
                <div><div className="text-xs text-muted-foreground">Per hari</div><div className="tabular" style={{ fontSize: 26, fontWeight: 800 }}>{formatRp(bep.per_hari)}</div></div>
              </div>
              <div className="list-rows">
                <Baris label="Biaya tetap + penyusutan per bulan" nilai={formatRp(bep.biaya_tetap)} />
                <Baris label="Sisa tiap Rp100rb jualan setelah bahan & komisi" nilai={formatRp(bep.sisa_per_100rb)} />
                <Baris label={bep.di_atas_bps >= 0 ? 'Omzet di atas titik impas' : 'Omzet masih di bawah titik impas'} nilai={`${Math.abs(Math.round(bep.di_atas_bps / 100))}%`}
                  warna={bep.di_atas_bps >= 0 ? 'var(--accent-green)' : 'var(--accent-red)'} />
              </div>
            </>
          ) : <p className="text-sm" style={{ color: 'var(--accent-red)' }}>Dengan skenario ini modal bahan + komisi sudah sebesar omzet, jadi setiap penjualan menambah rugi.</p>)}
        </div>

        <KalkulatorProduk period={period} />
      </div>

      {sim && (
        <div className="card" style={{ padding: 22 }}>
          <div className="chart-title">Berapa porsi per hari supaya balik modal</div>
          <div className="chart-subtitle">Kalau biaya tetap hanya ditanggung satu produk. Berguna untuk menilai produk andalan. Modal diambil dari HPP resep di Produksi &amp; Supplier.</div>
          {sim.produk.length === 0 ? <p className="text-sm text-muted-foreground" style={{ paddingTop: 8 }}>Belum ada produk. Isi resep di Produksi &amp; Supplier atau catat nama produk di transaksi penjualan.</p> : (
            <div className="table-scroll">
              <table className="data-table tabular" style={{ minWidth: 620 }}>
                <thead><tr><th>Produk</th><th style={{ textAlign: 'right' }}>Harga</th><th style={{ textAlign: 'right' }}>Modal</th><th style={{ textAlign: 'right' }}>Sisa per porsi</th><th style={{ textAlign: 'right' }}>Porsi / hari</th></tr></thead>
                <tbody>
                  {sim.produk.map((p) => (
                    <tr key={p.nama}>
                      <td>{p.nama}</td>
                      <td style={{ textAlign: 'right' }}>{formatRp(p.harga)}</td>
                      <td style={{ textAlign: 'right' }}>{p.ada_modal ? formatRp(p.modal) : <span className="text-muted-foreground" title="Belum ada resep di Produksi & Supplier">–</span>}</td>
                      <td style={{ textAlign: 'right', color: p.ada_modal && p.sisa <= 0 ? 'var(--accent-red)' : undefined }}>{p.ada_modal ? formatRp(p.sisa) : '–'}</td>
                      <td style={{ textAlign: 'right', fontWeight: 700 }}>{p.ada ? p.porsi_per_hari.toLocaleString('id-ID') : '–'}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}
    </>
  )
}

/** Kalkulator harga & titik impas per produk (kalkulator lama, dibuat otomatis & terhubung HPP resep). */
function KalkulatorProduk({ period }: { period: string }) {
  const [produk, setProduk] = useState<ProdukRow[]>([])
  const [pilih, setPilih] = useState('manual')
  const [cogs, setCogs] = useState(8000)
  const [saluran, setSaluran] = useState('toko')
  const [fee, setFee] = useState('0')
  const [mode, setMode] = useState<'margin' | 'rupiah'>('margin')
  const [margin, setMargin] = useState('60')
  const [profit, setProfit] = useState(7600)
  const [volume, setVolume] = useState(1500)
  const [targetNet, setTargetNet] = useState(10_000_000)
  const [fixed, setFixed] = useState<number | ''>('')
  const [res, setRes] = useState<PriceSim | null>(null)
  const [error, setError] = useState('')
  useEffect(() => { getProduksi().then((r) => setProduk(r.produk ?? []), () => {}) }, [])
  const input = useMemo(() => ({
    cogs, feePct: Number(fee || 0), targetProfit: mode === 'rupiah' ? profit : 0, marginPct: mode === 'margin' ? Number(margin) : 0,
    fixedCost: fixed === '' ? undefined : fixed, monthlyVolume: volume, targetNetProfit: targetNet,
  }), [cogs, fee, mode, profit, margin, fixed, volume, targetNet])
  const inputTunda = useTunda(input, 250)
  useEffect(() => {
    simulatePrice(period, inputTunda).then((r) => { setRes(r); setError('') }, (e: Error) => { setRes(null); setError(e.message) })
  }, [period, inputTunda])

  function pilihProduk(v: string) {
    setPilih(v)
    const p = produk.find((x) => String(x.id) === v)
    if (p) setCogs(p.hpp.per_unit)
  }
  function pilihSaluran(v: string) {
    setSaluran(v)
    const s = SALURAN.find((x) => x[0] === v)
    if (s && v !== 'lain') setFee(String(s[2]))
  }
  const be = res?.break_even

  return (
    <div className="card" style={{ padding: 22, display: 'grid', gridTemplateColumns: 'minmax(0, 1fr)', gap: 14, alignContent: 'start' }}>
      <div>
        <div className="chart-title">Kalkulator harga jual per produk</div>
        <div className="chart-subtitle">Harga minimal supaya dapat margin yang Anda mau, sudah termasuk potongan aplikasi.</div>
      </div>
      <div className="grid gap-3 sm:grid-cols-2">
        <div className="grid gap-1.5">
          <Label htmlFor="kp-produk">Produk</Label>
          <Select value={pilih} onValueChange={pilihProduk}>
            <SelectTrigger id="kp-produk" className="w-full"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="manual">Isi modal sendiri</SelectItem>
              {produk.map((p) => <SelectItem key={p.id} value={String(p.id)}>{p.nama} · HPP {formatRp(p.hpp.per_unit)}</SelectItem>)}
            </SelectContent>
          </Select>
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor="kp-modal">Modal per porsi (Rp)</Label>
          <Input id="kp-modal" className={fieldClass} {...angka(cogs, (n) => { setCogs(n); setPilih('manual') })} />
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor="kp-saluran">Dijual lewat</Label>
          <Select value={saluran} onValueChange={pilihSaluran}>
            <SelectTrigger id="kp-saluran" className="w-full"><SelectValue /></SelectTrigger>
            <SelectContent>{SALURAN.map(([v, l, f]) => <SelectItem key={v} value={v}>{l}{v !== 'lain' && f > 0 ? ` · komisi ${f}%` : ''}</SelectItem>)}</SelectContent>
          </Select>
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor="kp-fee">Komisi aplikasi (%)</Label>
          <Input id="kp-fee" type="number" min={0} max={90} step="0.1" className={fieldClass} value={fee} onChange={(e) => { setFee(e.target.value); setSaluran('lain') }} />
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor="kp-mode">Target untung</Label>
          <Select value={mode} onValueChange={(v) => setMode(v as 'margin' | 'rupiah')}>
            <SelectTrigger id="kp-mode" className="w-full"><SelectValue /></SelectTrigger>
            <SelectContent><SelectItem value="margin">Margin (%)</SelectItem><SelectItem value="rupiah">Rupiah per porsi</SelectItem></SelectContent>
          </Select>
        </div>
        {mode === 'margin' ? (
          <div className="grid gap-1.5">
            <Label htmlFor="kp-margin">Margin yang diinginkan</Label>
            <Select value={margin} onValueChange={setMargin}>
              <SelectTrigger id="kp-margin" className="w-full"><SelectValue /></SelectTrigger>
              <SelectContent>{['20', '30', '40', '50', '60', '70', '80'].map((m) => <SelectItem key={m} value={m}>{m}%</SelectItem>)}</SelectContent>
            </Select>
          </div>
        ) : (
          <div className="grid gap-1.5">
            <Label htmlFor="kp-profit">Untung per porsi (Rp)</Label>
            <Input id="kp-profit" className={fieldClass} {...angka(profit, setProfit)} />
          </div>
        )}
      </div>
      {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
      {res && (
        <div>
          <div className="text-xs text-muted-foreground">Harga jual minimal</div>
          <div className="tabular" style={{ fontSize: 34, fontWeight: 800, lineHeight: 1.15 }}>{formatRp(res.recommended_price)}</div>
          <p className="text-sm text-muted-foreground">
            Dari {formatRp(res.recommended_price)}: potongan aplikasi {formatRp(res.channel_fee)}, modal {formatRp(cogs)}, sisa untuk Anda {formatRp(res.net_take_home - cogs)}.
          </p>
        </div>
      )}
      <details className="text-sm">
        <summary className="cursor-pointer text-muted-foreground">Titik impas & target laba produk ini</summary>
        <div className="mt-3 grid gap-3 sm:grid-cols-3">
          <div className="grid gap-1.5"><Label htmlFor="kp-vol">Terjual per bulan (porsi)</Label><Input id="kp-vol" className={fieldClass} {...angka(volume, setVolume)} /></div>
          <div className="grid gap-1.5"><Label htmlFor="kp-target">Target laba bersih / bulan</Label><Input id="kp-target" className={fieldClass} {...angka(targetNet, setTargetNet)} /></div>
          <div className="grid gap-1.5"><Label htmlFor="kp-fixed">Biaya tetap / bulan</Label><Input id="kp-fixed" placeholder="Otomatis dari data" className={fieldClass} {...angka(fixed === '' ? 0 : fixed, (n) => setFixed(n || ''))} /></div>
        </div>
        {res?.break_even_error && <p className="mt-2 text-destructive">{res.break_even_error}</p>}
        {be && (
          <div className="list-rows" style={{ marginTop: 10 }}>
            <Baris label={`Titik impas${fixed === '' ? ' (biaya tetap dari data)' : ''}`} nilai={`${be.bep_units.toLocaleString('id-ID')} porsi · ±${be.bep_units_per_day.toLocaleString('id-ID')}/hari`} />
            <Baris label={`Laba bersih pada ${be.monthly_volume.toLocaleString('id-ID')} porsi`} nilai={formatRp(be.net_profit_at_volume)} warna={be.net_profit_at_volume < 0 ? 'var(--accent-red)' : 'var(--accent-green)'} />
            <Baris label={`Harus terjual untuk laba ${formatRp(be.target_net_profit)}`} nilai={`${be.units_for_target.toLocaleString('id-ID')} porsi`} />
          </div>
        )}
      </details>
    </div>
  )
}
