import { useEffect, useState, type FormEvent } from 'react'
import { PageHeader } from '@/components/PageHeader'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { simulatePrice, type Dashboard, type PriceSim } from '@/lib/api'
import { formatRp } from '@/lib/format'
import { angka } from '@/lib/angka'

const fieldClass = 'h-[38px] bg-[var(--bg-input)] tabular'

export function StokModule({ d, title }: { d: Dashboard; title: string }) {
  const [cari, setCari] = useState('')
  const [typeFilter, setTypeFilter] = useState('ALL')
  const [statusFilter, setStatusFilter] = useState('ALL')

  const filtered = (d.inventory || []).filter((item) => {
    if (typeFilter !== 'ALL' && item.type !== typeFilter) return false
    if (statusFilter === 'MENIPIS' && !item.status.includes('Menipis')) return false
    if (statusFilter === 'AMAN' && item.status.includes('Menipis')) return false
    if (cari.trim()) {
      const q = cari.toLowerCase()
      if (!item.sku.toLowerCase().includes(q) && !item.name.toLowerCase().includes(q)) return false
    }
    return true
  })

  return (
    <>
      <PageHeader title={title} subtitle="Data Master, Resep BOM, Stok Gudang & Panduan Master Pajak" />

      <div className="card" style={{ padding: '16px 20px', marginBottom: 16 }}>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 12, alignItems: 'center' }}>
          <Input
            style={{ flex: '1 1 240px', minWidth: 200, height: 38, background: 'var(--bg-input)' }}
            placeholder="Cari SKU atau nama item / bahan..."
            value={cari}
            onChange={(e) => setCari(e.target.value)}
          />
          <select
            className="period-select"
            style={{ height: 38, background: 'var(--bg-input)', border: '1px solid var(--border-strong)', color: 'var(--text-primary)', borderRadius: 8, padding: '0 12px', fontSize: 13 }}
            value={typeFilter}
            onChange={(e) => setTypeFilter(e.target.value)}
          >
            <option value="ALL">Semua Kategori</option>
            <option value="Bahan Baku">Bahan Baku</option>
            <option value="Produk Jadi">Produk Jadi</option>
          </select>
          <select
            className="period-select"
            style={{ height: 38, background: 'var(--bg-input)', border: '1px solid var(--border-strong)', color: 'var(--text-primary)', borderRadius: 8, padding: '0 12px', fontSize: 13 }}
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
          >
            <option value="ALL">Semua Status Stok</option>
            <option value="MENIPIS">Stok Menipis (Perlu Restock)</option>
            <option value="AMAN">Stok Aman / Resep BOM Aktif</option>
          </select>
        </div>
      </div>

      <div className="card" style={{ padding: 20 }}>
        <div className="chart-title" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 14 }}>
          <span>Stok Bahan Baku &amp; Produk Jadi ({d.tenant.name})</span>
          <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>Menampilkan {filtered.length} dari {d.inventory.length} item</span>
        </div>
        <table className="data-table tabular">
          <thead>
            <tr>
              <th>SKU</th><th>Nama Item</th><th>Kategori</th><th>Stok / Resep</th><th>HPP / Unit</th><th>Harga Jual</th><th>Status</th>
            </tr>
          </thead>
          <tbody>
            {filtered.length === 0 ? (
              <tr>
                <td colSpan={7} style={{ textAlign: 'center', padding: '28px 0', color: 'var(--text-muted)' }}>
                  Tidak ada item stok yang cocok dengan filter.
                </td>
              </tr>
            ) : (
              filtered.map((item) => (
                <tr key={item.sku}>
                  <td style={{ color: 'var(--text-muted)' }}>{item.sku}</td>
                  <td style={{ fontWeight: 600 }}>{item.name}</td>
                  <td>{item.type}</td>
                  <td>{item.stock}</td>
                  <td>{formatRp(item.cogs)}</td>
                  <td>{item.sell_price > 0 ? formatRp(item.sell_price) : '-'}</td>
                  <td>
                    <span className={'pill-status ' + (item.status.includes('Menipis') ? 'pill-red' : 'pill-green')}>{item.status}</span>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </>
  )
}

export function SimulasiHarga({ period }: { period: string }) {
  const [cogs, setCogs] = useState('6800')
  const [profit, setProfit] = useState('7600')
  const [fee, setFee] = useState('20')
  const [fixedCost, setFixedCost] = useState('')
  const [volume, setVolume] = useState('1500')
  const [targetNet, setTargetNet] = useState('10000000')
  const [res, setRes] = useState<PriceSim | null>(null)
  const [feeShown, setFeeShown] = useState('20')
  const [busy, setBusy] = useState(false)

  const num = (v: string) => parseFloat(v || '0')

  async function run(e?: FormEvent) {
    e?.preventDefault()
    setBusy(true)
    try {
      setRes(await simulatePrice(period, {
        cogs: num(cogs),
        targetProfit: num(profit),
        feePct: num(fee),
        fixedCost: fixedCost.trim() === '' ? undefined : num(fixedCost),
        monthlyVolume: num(volume),
        targetNetProfit: num(targetNet),
      }))
      setFeeShown(fee || '0')
    } finally {
      setBusy(false)
    }
  }

  // hitung ulang otomatis saat periode (sumber biaya tetap) berganti
  useEffect(() => { void run() }, [period])

  const fields = [
    { id: 'sim-cogs', label: 'HPP Produk per Cup/Pcs (Rp)', value: cogs, set: setCogs },
    { id: 'sim-profit', label: 'Target Laba Kotor per Pcs (Rp)', value: profit, set: setProfit },
    { id: 'sim-fee', label: 'Komisi Aplikasi (%)', value: fee, set: setFee, persen: true },
    { id: 'sim-fixed', label: 'Biaya Tetap per Bulan (Rp)', value: fixedCost, set: setFixedCost, placeholder: 'Otomatis dari data' },
    { id: 'sim-volume', label: 'Perkiraan Terjual per Bulan (Pcs)', value: volume, set: setVolume },
    { id: 'sim-target', label: 'Target Laba Bersih per Bulan (Rp)', value: targetNet, set: setTargetNet },
  ]
  const be = res?.break_even
  const results = res ? [
    { label: 'Harga Jual Rekomendasi di Aplikasi', value: formatRp(res.recommended_price), color: 'var(--text-warn)' },
    { label: `Potongan Komisi Aplikasi (${feeShown}%)`, value: formatRp(res.channel_fee), color: 'var(--text-danger)' },
    { label: 'Uang Bersih Diterima (Take Home)', value: formatRp(res.net_take_home), color: 'var(--accent-green)' },
  ] : []
  const beCards = be ? [
    { label: 'Biaya Tetap per Bulan', value: formatRp(be.fixed_cost), meta: fixedCost.trim() === '' ? 'Dari total beban operasional periode ini' : 'Input manual' },
    { label: 'Titik Impas (BEP)', value: `${be.bep_units.toLocaleString('id-ID')} pcs`, meta: `± ${be.bep_units_per_day.toLocaleString('id-ID')} pcs/hari · omzet ${formatRp(be.bep_revenue)}` },
    {
      label: `Laba Bersih pada ${be.monthly_volume.toLocaleString('id-ID')} pcs`,
      value: formatRp(be.net_profit_at_volume),
      color: be.net_profit_at_volume < 0 ? 'var(--text-danger)' : 'var(--accent-green)',
      meta: be.net_profit_at_volume < 0 ? 'Masih rugi, volume di bawah BEP' : 'Sudah di atas BEP',
    },
    { label: 'Harus Terjual untuk Target Laba', value: `${be.units_for_target.toLocaleString('id-ID')} pcs`, meta: `Target ${formatRp(be.target_net_profit)} per bulan` },
  ] : []

  return (
    <>
      <PageHeader
        title="Simulasi & Harga Channel"
        subtitle="Jangan sampai ramai orderan di GoFood/GrabFood tapi laba bersih tergerus potongan komisi aplikasi dan biaya tetap."
      />
      <form className="card" style={{ padding: 22 }} onSubmit={run}>
        <div className="chart-title">Kalkulator Harga, Titik Impas & Laba Bersih</div>
        <div className="form-grid">
          {fields.map((f) => (
            <div key={f.id}>
              <Label htmlFor={f.id} className="kpi-label">{f.label}</Label>
              {f.persen
                ? <Input id={f.id} type="number" min={0} step="0.1" placeholder={f.placeholder} className={fieldClass + ' mt-1.5'} value={f.value} onChange={(e) => f.set(e.target.value)} />
                : <Input id={f.id} placeholder={f.placeholder} className={fieldClass + ' mt-1.5'} {...angka(f.value, (n) => f.set(n ? String(n) : ''))} />}
            </div>
          ))}
          <div style={{ display: 'flex', alignItems: 'flex-end' }}>
            <button className="btn-cta" style={{ width: '100%' }} type="submit" disabled={busy}>Hitung</button>
          </div>
        </div>

        <div className="sim-results">
          {results.map((r) => (
            <div className="card" style={{ background: 'var(--bg-raised)' }} key={r.label}>
              <div className="kpi-label">{r.label}</div>
              <div className="kpi-value tabular" style={{ color: r.color }}>{r.value}</div>
            </div>
          ))}
        </div>

        {res?.break_even_error && <p className="text-sm text-destructive" role="alert" style={{ marginTop: 16 }}>{res.break_even_error}</p>}
        {be && (
          <div className="sim-results sim-results-4">
            {beCards.map((c) => (
              <div className="card" style={{ background: 'var(--bg-raised)' }} key={c.label}>
                <div className="kpi-label">{c.label}</div>
                <div className="kpi-value tabular" style={{ color: c.color }}>{c.value}</div>
                <div className="kpi-meta-row">{c.meta}</div>
              </div>
            ))}
          </div>
        )}
      </form>
    </>
  )
}

export function PajakUMKM({ d }: { d: Dashboard }) {
  const tr = d.tax_report
  const kpis = [
    { label: 'Akumulasi Omzet 2026', value: tr.total_gross, meta: <span className="badge-up">Jan – Agu 2026</span> },
    { label: 'Kuota Bebas Pajak WP OP', value: tr.yearly_exemption, meta: <span>PP 55/2022 Pasal 60</span> },
    { label: 'Omzet Kena Pajak (Di atas Rp500 jt)', value: tr.total_taxable, color: 'var(--text-warn)', meta: <span>Mulai terlewati di bulan Juli</span> },
    { label: 'Total PPh Final 0,5% Terutang', value: tr.total_tax_due, color: 'var(--accent-green)', meta: <span>Siap dilampirkan ke SPT Tahunan</span> },
  ]
  return (
    <>
      <PageHeader
        title="Pajak UMKM (PP 55 Tahun 2022)"
        subtitle="Kalkulasi PPh Final 0,5% & Pemantauan Kuota Bebas Pajak Rp500 Juta/Tahun (WP Orang Pribadi)"
      >
        <span className="pill-status pill-amber">Status WP: Orang Pribadi (Bebas Pajak s/d Rp500 jt)</span>
      </PageHeader>

      <div className="kpi-grid" style={{ gridTemplateColumns: 'repeat(4, 1fr)' }}>
        {kpis.map((k) => (
          <div className="card" key={k.label}>
            <div className="kpi-label">{k.label}</div>
            <div className="kpi-value tabular" style={k.color ? { color: k.color } : undefined}>{formatRp(k.value)}</div>
            <div className="kpi-meta-row">{k.meta}</div>
          </div>
        ))}
      </div>

      <div className="card" style={{ padding: 20 }}>
        <div className="chart-title">Kertas Kerja Rekapitulasi Peredaran Bruto Bulanan</div>
        <table className="data-table tabular">
          <thead>
            <tr>
              <th>Bulan</th><th>Omzet Bruto</th><th>Akumulasi Tahunan</th><th>Omzet Kena Pajak</th><th>PPh Final (0,5%)</th><th>Status Pajak</th>
            </tr>
          </thead>
          <tbody>
            {tr.months.map((m) => (
              <tr key={m.month_name}>
                <td style={{ fontWeight: 600 }}>{m.month_name}</td>
                <td>{formatRp(m.gross_omzet)}</td>
                <td>{formatRp(m.cumulative_omzet)}</td>
                <td>{formatRp(m.taxable_amount)}</td>
                <td style={{ fontWeight: 700, color: m.tax_due > 0 ? 'var(--text-warn)' : 'var(--accent-green)' }}>{formatRp(m.tax_due)}</td>
                <td><span className={'pill-status ' + (m.tax_due > 0 ? 'pill-amber' : 'pill-green')}>{m.status}</span></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  )
}
