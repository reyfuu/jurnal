import { useState, type FormEvent } from 'react'
import { PageHeader } from '@/components/PageHeader'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { simulatePrice, type Dashboard, type PriceSim } from '@/lib/api'
import { formatRp } from '@/lib/format'

const fieldClass = 'h-[38px] bg-[#0d0f13] tabular'

export function Transaksi({ d, onAdd }: { d: Dashboard; onAdd: (tx: { title: string; type: 'INCOME' | 'EXPENSE'; gross: number }) => Promise<void> }) {
  const [title, setTitle] = useState('Penjualan Catering Rapat 25 Cup')
  const [type, setType] = useState<'INCOME' | 'EXPENSE'>('INCOME')
  const [gross, setGross] = useState('1000000')
  const [busy, setBusy] = useState(false)

  async function submit(e: FormEvent) {
    e.preventDefault()
    const amount = parseInt(gross || '0', 10)
    if (!title.trim() || amount <= 0) return
    setBusy(true)
    try {
      await onAdd({ title: title.trim(), type, gross: amount })
    } finally {
      setBusy(false)
    }
  }

  return (
    <>
      <PageHeader
        title="Transaksi Operasional"
        subtitle="Catat pemasukan & pengeluaran dalam 5 detik. Otomatis memperbarui Dashboard, Arus Kas & Pajak UMKM."
      />
      <form className="card" style={{ padding: '18px 20px' }} onSubmit={submit}>
        <div className="chart-title">Tambah Transaksi Cepat</div>
        <div className="form-grid">
          <Input className={fieldClass} aria-label="Keterangan" placeholder="Contoh: Penjualan Kopi Literan 10 Botol" value={title} onChange={(e) => setTitle(e.target.value)} />
          <Select value={type} onValueChange={(v) => setType(v as 'INCOME' | 'EXPENSE')}>
            <SelectTrigger className={fieldClass + ' w-full'} aria-label="Jenis transaksi">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="INCOME">Pemasukan (+ Omzet)</SelectItem>
              <SelectItem value="EXPENSE">Pengeluaran (- Biaya)</SelectItem>
            </SelectContent>
          </Select>
          <Input className={fieldClass} aria-label="Nominal" type="number" min={1} placeholder="Nominal (Rp)" value={gross} onChange={(e) => setGross(e.target.value)} />
          <button className="btn-cta" type="submit" disabled={busy}>{busy ? 'Menyimpan…' : '+ Simpan Transaksi'}</button>
        </div>
      </form>

      <div className="card" style={{ padding: 20 }}>
        <div className="chart-title">Riwayat Transaksi Terbaru</div>
        <table className="data-table tabular">
          <thead>
            <tr>
              <th>ID</th><th>Tanggal</th><th>Keterangan</th><th>Channel</th><th>Bruto</th><th>Potongan</th><th>Kas Bersih</th>
            </tr>
          </thead>
          <tbody>
            {d.recent_transactions.map((t, i) => (
              <tr key={t.id + i}>
                <td style={{ color: 'var(--text-muted)' }}>{t.id}</td>
                <td>{t.date}</td>
                <td style={{ fontWeight: 600 }}>{t.title}</td>
                <td>{t.channel}</td>
                <td>{formatRp(t.gross)}</td>
                <td style={{ color: '#fb7185' }}>{t.fee > 0 ? '-' + formatRp(t.fee) : 'Rp0'}</td>
                <td style={{ fontWeight: 700, color: t.net >= 0 ? '#10b981' : '#fb7185' }}>{formatRp(t.net)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  )
}

export function PiutangUtang({ d }: { d: Dashboard }) {
  return (
    <>
      <PageHeader title="Piutang & Utang" subtitle="Pantau tagihan pelanggan yang lewat 30 hari agar arus kas bisnis tidak macet." />
      <div className="card" style={{ padding: 20 }}>
        <table className="data-table tabular">
          <thead>
            <tr>
              <th>No. Dokumen</th><th>Pelanggan / Supplier</th><th>Jenis</th><th>Jatuh Tempo</th><th>Status Umur</th><th>Nominal</th>
            </tr>
          </thead>
          <tbody>
            {d.obligations.map((o) => (
              <tr key={o.invoice_no}>
                <td style={{ fontWeight: 600 }}>{o.invoice_no}</td>
                <td>{o.counterparty}</td>
                <td>{o.kind === 'RECEIVABLE' ? 'Piutang Masuk' : 'Utang Supplier'}</td>
                <td>{o.due_date}</td>
                <td>
                  <span className={'pill-status ' + (o.days_overdue > 30 ? 'pill-red' : 'pill-green')}>
                    {o.days_overdue > 30 ? `Lewat ${o.days_overdue} hari (>30 hari)` : 'Belum Jatuh Tempo'}
                  </span>
                </td>
                <td style={{ fontWeight: 700 }}>{formatRp(o.amount)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  )
}

export function StokModule({ d, title }: { d: Dashboard; title: string }) {
  return (
    <>
      <PageHeader title={title} subtitle="Data Master, Resep BOM, Stok Gudang & Panduan Akademi Pajak" />
      <div className="card" style={{ padding: 20 }}>
        <div className="chart-title">Stok Bahan Baku &amp; Produk Jadi ({d.tenant.name})</div>
        <table className="data-table tabular">
          <thead>
            <tr>
              <th>SKU</th><th>Nama Item</th><th>Kategori</th><th>Stok / Resep</th><th>HPP / Unit</th><th>Harga Jual</th><th>Status</th>
            </tr>
          </thead>
          <tbody>
            {d.inventory.map((item) => (
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
            ))}
          </tbody>
        </table>
      </div>
    </>
  )
}

export function SimulasiHarga() {
  const [cogs, setCogs] = useState('6800')
  const [profit, setProfit] = useState('7600')
  const [fee, setFee] = useState('20')
  const [res, setRes] = useState<PriceSim>({ recommended_price: 18_000, channel_fee: 3_600, net_take_home: 14_400 })
  const [feeShown, setFeeShown] = useState('20')
  const [busy, setBusy] = useState(false)

  async function run(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    try {
      setRes(await simulatePrice(parseInt(cogs || '0', 10), parseInt(profit || '0', 10), parseFloat(fee || '0')))
      setFeeShown(fee || '0')
    } finally {
      setBusy(false)
    }
  }

  const fields = [
    { id: 'sim-cogs', label: 'HPP Produk per Cup/Pcs (Rp)', value: cogs, set: setCogs },
    { id: 'sim-profit', label: 'Target Laba Bersih per Pcs (Rp)', value: profit, set: setProfit },
    { id: 'sim-fee', label: 'Komisi Aplikasi (%)', value: fee, set: setFee },
  ]
  const results = [
    { label: 'Harga Jual Rekomendasi di Aplikasi', value: res.recommended_price, color: '#facc15' },
    { label: `Potongan Komisi Aplikasi (${feeShown}%)`, value: res.channel_fee, color: '#fb7185' },
    { label: 'Uang Bersih Diterima (Take Home)', value: res.net_take_home, color: '#10b981' },
  ]

  return (
    <>
      <PageHeader
        title="Simulasi & Harga Channel"
        subtitle="Jangan sampai ramai orderan di GoFood/GrabFood tapi laba bersih tergerus potongan komisi aplikasi."
      />
      <form className="card" style={{ padding: 22 }} onSubmit={run}>
        <div className="chart-title">Kalkulator Anti-Boncos Komisi Aplikasi</div>
        <div className="form-grid">
          {fields.map((f) => (
            <div key={f.id}>
              <Label htmlFor={f.id} className="kpi-label">{f.label}</Label>
              <Input id={f.id} type="number" min={0} className={fieldClass + ' mt-1.5'} value={f.value} onChange={(e) => f.set(e.target.value)} />
            </div>
          ))}
          <div style={{ display: 'flex', alignItems: 'flex-end' }}>
            <button className="btn-cta" style={{ width: '100%' }} type="submit" disabled={busy}>Hitung Harga Jual Ideal</button>
          </div>
        </div>

        <div className="sim-results">
          {results.map((r) => (
            <div className="card" style={{ background: '#101318' }} key={r.label}>
              <div className="kpi-label">{r.label}</div>
              <div className="kpi-value tabular" style={{ color: r.color }}>{formatRp(r.value)}</div>
            </div>
          ))}
        </div>
      </form>
    </>
  )
}

export function PajakUMKM({ d }: { d: Dashboard }) {
  const tr = d.tax_report
  const kpis = [
    { label: 'Akumulasi Omzet 2026', value: tr.total_gross, meta: <span className="badge-up">Jan – Agu 2026</span> },
    { label: 'Kuota Bebas Pajak WP OP', value: tr.yearly_exemption, meta: <span>PP 55/2022 Pasal 60</span> },
    { label: 'Omzet Kena Pajak (Di atas Rp500 jt)', value: tr.total_taxable, color: '#facc15', meta: <span>Mulai terlewati di bulan Juli</span> },
    { label: 'Total PPh Final 0,5% Terutang', value: tr.total_tax_due, color: '#10b981', meta: <span>Siap dilampirkan ke SPT Tahunan</span> },
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
                <td style={{ fontWeight: 700, color: m.tax_due > 0 ? '#facc15' : '#10b981' }}>{formatRp(m.tax_due)}</td>
                <td><span className={'pill-status ' + (m.tax_due > 0 ? 'pill-amber' : 'pill-green')}>{m.status}</span></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  )
}
