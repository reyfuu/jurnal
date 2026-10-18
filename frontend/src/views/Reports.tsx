import { AmountRow, PageHeader } from '@/components/PageHeader'
import type { BalanceSheet, CashFlow, IncomeStatement } from '@/lib/api'
import { formatJt, formatPct, formatRp, formatSignedJt } from '@/lib/format'

const RED = '#fb7185'
const GREEN = '#10b981'
const BLUE = '#38bdf8'

export function LaporanKeuangan({ r, bs, periodLabel }: { r: IncomeStatement; bs: BalanceSheet; periodLabel: string }) {
  return (
    <>
      <PageHeader title="Laporan Keuangan" subtitle={`${periodLabel} · Laba Rugi & Neraca Sederhana`}>
        <span className={'pill-status ' + (bs.is_balanced ? 'pill-green' : 'pill-amber')}>
          {bs.is_balanced ? '✓ Aktiva = Pasiva (Seimbang)' : 'Aktiva ≠ Pasiva, periksa data'}
        </span>
      </PageHeader>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(320px, 1fr))', gap: 14 }}>
        <div className="card" style={{ padding: 20 }}>
          <div className="chart-title">Laporan Laba Rugi</div>
          <div className="chart-subtitle" style={{ marginBottom: 12 }}>Margin laba bersih {formatPct(r.margin_bps)}</div>
          <div className="list-rows tabular">
            <AmountRow label="Omzet (penjualan kotor)" value={formatRp(r.omzet)} color="#facc15" bold />
            <AmountRow label="Potongan channel / aplikasi" value={formatRp(-r.channel_fees)} color={RED} />
            <AmountRow label="Penjualan bersih" value={formatRp(r.net_sales)} bold />
            <AmountRow label="HPP (bahan baku)" value={formatRp(-r.cogs)} color={RED} />
            <AmountRow label="Laba kotor" value={formatRp(r.gross_profit)} bold />
            {(r.opex_lines ?? []).map((l, i) => (
              <AmountRow key={i} label={l.label} value={formatRp(-l.amount)} color={RED} />
            ))}
            <AmountRow label="Total beban operasional" value={formatRp(-r.opex)} color={RED} />
            {r.other_income > 0 && <AmountRow label="Pendapatan lain" value={formatRp(r.other_income)} color={GREEN} />}
            <AmountRow label="Laba bersih" value={formatRp(r.net_profit)} color={GREEN} bold />
          </div>
        </div>

        <div className="card" style={{ padding: 20 }}>
          <div className="chart-title">Neraca (Posisi Keuangan)</div>
          <div className="chart-subtitle" style={{ marginBottom: 12 }}>Posisi Kas, Piutang, Stok vs Utang &amp; Modal</div>
          <div className="list-rows tabular">
            <AmountRow label="ASET (AKTIVA)" value={formatRp(bs.assets.total)} color={BLUE} bold />
            <AmountRow label="· Saldo Kas & Bank" value={formatRp(bs.assets.cash)} color={BLUE} />
            <AmountRow label="· Piutang Belum Tertagih" value={formatRp(bs.assets.receivables)} color={RED} />
            <AmountRow label="· Nilai Persediaan Stok" value={formatRp(bs.assets.inventory)} />
            {bs.assets.fixed_assets !== 0 && <AmountRow label="· Alat & Aset Tetap" value={formatRp(bs.assets.fixed_assets)} />}
            <div style={{ height: 8 }} />
            <AmountRow label="KEWAJIBAN (UTANG)" value={formatRp(bs.liabilities.total)} color={RED} bold />
            <AmountRow label="· Utang Usaha / Operasional" value={formatRp(bs.liabilities.payables)} color={RED} />
            {bs.liabilities.loans !== 0 && <AmountRow label="· Pinjaman" value={formatRp(bs.liabilities.loans)} color={RED} />}
            <div style={{ height: 8 }} />
            <AmountRow label="MODAL (EKUITAS)" value={formatRp(bs.equity.total)} color={GREEN} bold />
            <AmountRow label="· Modal Awal Disetor" value={formatRp(bs.equity.opening_capital)} />
            {bs.equity.added_capital !== 0 && <AmountRow label="· Tambahan Modal − Ambil Pribadi" value={formatRp(bs.equity.added_capital)} />}
            <AmountRow label="· Laba Bersih Periode Berjalan" value={formatRp(bs.equity.current_profit)} color={GREEN} />
            <AmountRow
              label="TOTAL PASIVA (Utang + Modal)"
              value={formatRp(bs.liabilities.total + bs.equity.total)}
              color={GREEN}
              bold
              style={{ border: '1px solid rgba(16,185,129,0.3)', marginTop: 6 }}
            />
          </div>
        </div>
      </div>
    </>
  )
}

export function ArusKas({ c, netProfit }: { c: CashFlow; netProfit: number }) {
  const signed = (n: number) => (n >= 0 ? '+' : '-') + formatRp(Math.abs(n))
  return (
    <>
      <PageHeader
        title="Laporan Arus Kas & Rekonsiliasi"
        subtitle={`Kenapa Laba Bersih ${formatJt(netProfit)} tapi kenaikan kas hanya ${formatJt(c.net_cash)}? Ini jembatan penjelasannya.`}
      />
      <div className="kpi-grid" style={{ gridTemplateColumns: 'repeat(3, 1fr)' }}>
        <div className="card">
          <div className="kpi-label">Saldo Kas Awal Bulan</div>
          <div className="kpi-value tabular">{formatJt(c.opening_balance)}</div>
        </div>
        <div className="card">
          <div className="kpi-label">Arus Kas Bersih Bulan Ini</div>
          <div className="kpi-value tabular" style={{ color: GREEN }}>{formatSignedJt(c.net_cash)}</div>
          <div className="kpi-meta-row">
            <span>Masuk {formatRp(c.cash_in)} · Keluar {formatRp(c.cash_out)}</span>
          </div>
        </div>
        <div className="card">
          <div className="kpi-label">Saldo Kas Akhir Bulan</div>
          <div className="kpi-value tabular" style={{ color: BLUE }}>{formatJt(c.closing_balance)}</div>
        </div>
      </div>
      <div className="card" style={{ padding: 20 }}>
        <div className="chart-title">Rekonsiliasi Laba Bersih vs Kenaikan Kas</div>
        <div className="list-rows tabular">
          <AmountRow label="Laba Bersih (Laporan Laba Rugi)" value={signed(netProfit)} color={GREEN} />
          <AmountRow label="Dikurangi: penjualan tempo yang belum dibayar pelanggan (piutang baru)" value={signed(-c.new_receivables)} color={RED} />
          <AmountRow label="Ditambah: biaya yang belum dibayar (utang baru)" value={signed(c.new_payables)} color={BLUE} />
          {c.non_operating !== 0 && (
            <AmountRow label="Modal & pinjaman masuk, dikurangi beli aset, cicilan & ambil pribadi" value={signed(c.non_operating)} color={BLUE} />
          )}
          <AmountRow
            label="Kenaikan Kas Riil di Rekening Bank & Tunai"
            value={signed(c.net_cash)}
            color={BLUE}
            bold
            style={{ border: '1px solid rgba(56,189,248,0.3)' }}
          />
        </div>
      </div>
    </>
  )
}
