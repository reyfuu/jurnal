import { useState } from 'react'
import { PageHeader } from '@/components/PageHeader'
import { Input } from '@/components/ui/input'
import { type Dashboard } from '@/lib/api'
import { formatRp } from '@/lib/format'
import { FiturTerkunci } from '@/components/FiturTerkunci'
import { RekonsiliasiPajak } from '@/views/Rekonsiliasi'

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
      <PageHeader title={title} subtitle="Data Master, Resep BOM, Stok Gudang & Panduan MasterPajak AI" />

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

export function PajakUMKM({ d, period, pro, bisaEkspor, bisaCatat, onLangganan }: {
  d: Dashboard; period: string; pro: boolean; bisaEkspor: boolean; bisaCatat: boolean; onLangganan?: () => void
}) {
  const tr = d.tax_report
  const tahun = period.slice(0, 4)
  const op = tr.entity_type === 'OP'
  const akhir = tr.months.at(-1)?.month_name.slice(0, 3)
  const mulai = tr.months.find((m) => m.taxable_amount > 0)?.month_name
  const kpis = [
    { label: `Akumulasi Omzet ${tahun}`, value: tr.total_gross, meta: <span className="badge-up">{akhir ? `Jan – ${akhir} ${tahun}` : tahun}</span> },
    op
      ? { label: 'Kuota Bebas Pajak WP OP', value: tr.yearly_exemption, meta: <span>PP 55/2022 Pasal 60</span> }
      : { label: 'Kuota Bebas Pajak', value: 0, meta: <span>Hanya untuk orang pribadi</span> },
    { label: op ? 'Omzet Kena Pajak (Di atas Rp500.000.000)' : 'Omzet Kena Pajak', value: tr.total_taxable, color: 'var(--text-warn)',
      meta: <span>{!op ? 'Sejak rupiah pertama' : mulai ? `Mulai terlewati di bulan ${mulai}` : 'Kuota belum terlewati'}</span> },
    { label: 'Total PPh Final 0,5% Terutang', value: tr.total_tax_due, color: 'var(--accent-green)', meta: <span>Rekap untuk SPT Tahunan</span> },
  ]
  return (
    <>
      <PageHeader
        title="Pajak UMKM (PP 55 Tahun 2022)"
        subtitle={op ? 'Kalkulasi PPh Final 0,5% & Pemantauan Kuota Bebas Pajak Rp500 Juta/Tahun (WP Orang Pribadi)' : 'Kalkulasi PPh Final 0,5% dari omzet bruto (WP Badan)'}
      >
        <span className="pill-status pill-brand">{op ? 'Status WP: Orang Pribadi (Bebas Pajak s/d Rp500.000.000)' : 'Status WP: Badan'}</span>
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

      {pro
        ? <RekonsiliasiPajak tahunAwal={Number(tahun)} bisaEkspor={bisaEkspor} bisaCatat={bisaCatat} />
        : <FiturTerkunci pro nama="Rekonsiliasi bulanan vs tahunan" onLangganan={onLangganan} />}

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
