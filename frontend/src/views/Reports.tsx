import { useState } from 'react'
import { AmountRow, PageHeader } from '@/components/PageHeader'
import { Aset } from '@/views/Aset'
import { LaporanAI } from '@/views/LaporanAI'
import type { AssetSummary, BalanceSheet, CashFlow, CashFlowSection, IncomeStatement } from '@/lib/api'
import { formatJt, formatPct, formatRp, formatSignedJt } from '@/lib/format'

const RED = '#fb7185'
const GREEN = '#10b981'
const BLUE = '#38bdf8'

type Tab = 'laba-rugi' | 'neraca' | 'arus-kas' | 'aset' | 'ai'

const TABS: { id: Tab; label: string; badge?: string }[] = [
  { id: 'laba-rugi', label: 'Laba Rugi' },
  { id: 'neraca', label: 'Neraca' },
  { id: 'arus-kas', label: 'Arus Kas', badge: '2 METODE' },
  { id: 'aset', label: 'Aset & Penyusutan' },
  { id: 'ai', label: 'Laporan AI', badge: 'ANALISIS' },
]

const SUBTITLE: Record<Tab, string> = {
  'laba-rugi': 'Laba Rugi, Neraca, Arus Kas & Aset dalam satu tempat',
  neraca: 'Posisi keuangan per akhir periode',
  'arus-kas': 'Dua metode penyajian: Langsung & Tidak Langsung',
  aset: 'Register aset, penyusutan & amortisasi',
  ai: 'Seluruh laporan dalam satu halaman, dibaca & dianalisis AI',
}

interface Props {
  r: IncomeStatement
  bs: BalanceSheet
  cf: CashFlow
  assets: AssetSummary
  period: string
  periodLabel: string
  onAssetsChanged: () => void
}

/** Satu menu untuk seluruh laporan keuangan; tiap laporan jadi tab. */
export function LaporanKeuangan({ r, bs, cf, assets, period, periodLabel, onAssetsChanged }: Props) {
  const [tab, setTab] = useState<Tab>('laba-rugi')

  return (
    <>
      <PageHeader title="Laporan Keuangan" subtitle={`${periodLabel} · ${SUBTITLE[tab]}`}>
        {(tab === 'laba-rugi' || tab === 'neraca') && (
          <span className={'pill-status ' + (bs.is_balanced ? 'pill-green' : 'pill-amber')}>
            {bs.is_balanced ? 'Aktiva = Pasiva (Seimbang)' : 'Aktiva ≠ Pasiva, periksa data'}
          </span>
        )}
      </PageHeader>

      <div className="tabbar" role="tablist" aria-label="Bagian laporan keuangan">
        {TABS.map((t) => (
          <button
            key={t.id}
            role="tab"
            className="tab-btn"
            aria-selected={t.id === tab}
            onClick={() => setTab(t.id)}
          >
            {t.label}
            {t.badge && <span className="tab-badge">{t.badge}</span>}
          </button>
        ))}
      </div>

      {tab === 'laba-rugi' && <LabaRugi r={r} cf={cf} />}
      {tab === 'neraca' && <Neraca bs={bs} />}
      {tab === 'arus-kas' && <ArusKas cf={cf} netProfit={r.net_profit} />}
      {tab === 'aset' && <Aset data={assets} onChanged={onAssetsChanged} />}
      {tab === 'ai' && <LaporanAI key={period} period={period} />}
    </>
  )
}

function LabaRugi({ r, cf }: { r: IncomeStatement; cf: CashFlow }) {
  const nonKas = r.depreciation + r.amortization
  const selisih = r.net_profit - cf.net_cash
  return (
    <div className="report-grid">
      <div className="card" style={{ padding: 20 }}>
        <div className="chart-title">Laporan Laba Rugi</div>
        <div className="chart-subtitle" style={{ marginBottom: 12 }}>
          Margin laba bersih {formatPct(r.margin_bps)} · basis akrual
        </div>
        <div className="list-rows tabular">
          <AmountRow label="Omzet (penjualan kotor)" value={formatRp(r.omzet)} color="#facc15" bold />
          <AmountRow label="Potongan channel / aplikasi" value={formatRp(-r.channel_fees)} color={RED} />
          <AmountRow label="Penjualan bersih" value={formatRp(r.net_sales)} bold />
          <AmountRow label="HPP (bahan baku)" value={formatRp(-r.cogs)} color={RED} />
          <AmountRow label="Laba kotor" value={formatRp(r.gross_profit)} bold />
          {(r.opex_lines ?? []).map((l, i) => (
            <AmountRow key={i} label={l.label} value={formatRp(-l.amount)} color={RED} />
          ))}
          {r.depreciation !== 0 && (
            <AmountRow label="Beban penyusutan aset tetap" value={formatRp(-r.depreciation)} color={RED} />
          )}
          {r.amortization !== 0 && (
            <AmountRow label="Beban amortisasi harta tak berwujud" value={formatRp(-r.amortization)} color={RED} />
          )}
          <AmountRow label="Total beban operasional" value={formatRp(-r.total_opex)} color={RED} bold />
          {r.other_income > 0 && <AmountRow label="Pendapatan lain" value={formatRp(r.other_income)} color={GREEN} />}
          <AmountRow label="Laba bersih" value={formatRp(r.net_profit)} color={GREEN} bold />
        </div>
      </div>

      <div className="card" style={{ padding: 20 }}>
        <div className="chart-title">Jembatan ke Tab Lain</div>
        <div className="chart-subtitle">Angka kunci yang dipakai Neraca, Arus Kas &amp; Aset</div>
        <div className="list-rows tabular">
          <AmountRow label="Laba bersih → Modal di Neraca" value={formatJt(r.net_profit)} color={GREEN} bold />
          <AmountRow label="Kenaikan kas riil → tab Arus Kas" value={formatSignedJt(cf.net_cash)} color={BLUE} bold />
          <AmountRow label="Selisih laba vs kas" value={formatSignedJt(-selisih)} color={selisih === 0 ? GREEN : RED} />
        </div>
        {selisih !== 0 && (
          <div className="note-box">
            <strong>Kenapa laba {formatJt(r.net_profit)} tapi kas bergerak {formatSignedJt(cf.net_cash)}?</strong>{' '}
            {cf.new_receivables > 0 && <>{formatJt(cf.new_receivables)} masih tertahan di piutang. </>}
            {cf.new_payables > 0 && <>{formatJt(cf.new_payables)} biaya sudah dicatat tapi belum dibayar. </>}
            {nonKas > 0 && <>{formatJt(nonKas)} penyusutan &amp; amortisasi tidak mengeluarkan uang sama sekali. </>}
            Rinciannya di tab <strong>Arus Kas</strong>.
          </div>
        )}
        <div className="note-box warn">
          Belum ada baris <em>Pajak Penghasilan</em> di sini — angkanya dihitung di menu Pajak UMKM &amp;
          Perencanaan Pajak, tapi belum ditarik masuk ke laba rugi.
        </div>
      </div>
    </div>
  )
}

function Neraca({ bs }: { bs: BalanceSheet }) {
  const a = bs.assets
  return (
    <div className="report-grid">
      <div className="card" style={{ padding: 20 }}>
        <div className="chart-title">Neraca (Posisi Keuangan)</div>
        <div className="chart-subtitle" style={{ marginBottom: 12 }}>Posisi kas, piutang, stok &amp; aset vs utang &amp; modal</div>
        <div className="list-rows tabular">
          <div className="section-head">Aset (Aktiva)</div>
          <AmountRow label="TOTAL ASET" value={formatRp(a.total)} color={BLUE} bold />
          <AmountRow label="Saldo Kas & Bank" value={formatRp(a.cash)} color={BLUE} />
          <AmountRow label="Piutang Belum Tertagih" value={formatRp(a.receivables)} color={RED} />
          <AmountRow label="Nilai Persediaan Stok" value={formatRp(a.inventory)} />
          <AmountRow label="Aset Tetap & Tak Berwujud (nilai buku)" value={formatRp(a.fixed_assets)} />
          <div className="list-row sub">
            <span>Harga perolehan</span>
            <span className="tabular">{formatRp(a.fixed_assets_cost)}</span>
          </div>
          <div className="list-row sub">
            <span>Akumulasi penyusutan &amp; amortisasi</span>
            <span className="tabular" style={{ color: RED }}>{formatRp(-a.accum_depreciation)}</span>
          </div>

          <div className="section-head">Kewajiban (Utang)</div>
          <AmountRow label="TOTAL KEWAJIBAN" value={formatRp(bs.liabilities.total)} color={RED} bold />
          <AmountRow label="Utang Usaha / Operasional" value={formatRp(bs.liabilities.payables)} color={RED} />
          {bs.liabilities.loans !== 0 && <AmountRow label="Pinjaman" value={formatRp(bs.liabilities.loans)} color={RED} />}

          <div className="section-head">Modal (Ekuitas)</div>
          <AmountRow label="TOTAL MODAL" value={formatRp(bs.equity.total)} color={GREEN} bold />
          <AmountRow label="Modal Awal (kas + stok + nilai buku aset awal)" value={formatRp(bs.equity.opening_capital)} />
          {bs.equity.added_capital !== 0 && (
            <AmountRow label="Tambahan Modal − Ambil Pribadi" value={formatRp(bs.equity.added_capital)} />
          )}
          <AmountRow label="Laba Bersih Periode Berjalan" value={formatRp(bs.equity.current_profit)} color={GREEN} />

          <AmountRow
            label="TOTAL PASIVA (Utang + Modal)"
            value={formatRp(bs.liabilities.total + bs.equity.total)}
            color={BLUE}
            bold
            style={{ border: '1px solid rgba(56,189,248,0.3)', marginTop: 6 }}
          />
        </div>
      </div>

      <div className="card" style={{ padding: 20 }}>
        <div className="chart-title">Uji Keseimbangan</div>
        <div className="chart-subtitle">Aktiva harus sama dengan Pasiva</div>
        <div className="list-rows tabular">
          <AmountRow label="Total Aktiva" value={formatRp(a.total)} color={BLUE} bold />
          <AmountRow label="Total Pasiva" value={formatRp(bs.liabilities.total + bs.equity.total)} color={BLUE} bold />
          <AmountRow
            label="Selisih"
            value={formatRp(a.total - bs.liabilities.total - bs.equity.total)}
            color={bs.is_balanced ? GREEN : RED}
            bold
          />
        </div>
        <div className="note-box">
          <strong>Hubungan antar tab:</strong><br />
          · Kas {formatJt(a.cash)} = saldo akhir di tab Arus Kas.<br />
          · Piutang &amp; utang = penyesuaian modal kerja di Arus Kas metode tidak langsung.<br />
          · Nilai buku aset {formatJt(a.fixed_assets)} = total tab Aset &amp; Penyusutan.<br />
          · Laba {formatJt(bs.equity.current_profit)} = baris Laba Bersih di tab Laba Rugi.
        </div>
        <div className="note-box warn">
          Nilai persediaan masih angka tetap karena modul stok belum menghitung persediaan dari transaksi.
        </div>
      </div>
    </div>
  )
}

/** Satu bagian arus kas (Operasi / Investasi / Pendanaan) beserta rinciannya. */
function Section({ title, section, total }: { title: string; section: CashFlowSection; total: string }) {
  return (
    <>
      <div className="section-head">{title}</div>
      {section.lines.map((l, i) => (
        <div key={i}>
          <AmountRow
            label={l.label}
            value={l.non_kas ? 'Rp0' : formatSignedJt(l.amount)}
            color={l.non_kas ? 'var(--text-muted)' : l.amount < 0 ? RED : l.amount > 0 ? GREEN : 'var(--text-muted)'}
          />
          {(l.detail ?? []).map((d, j) => (
            <div className="list-row sub" key={j}>
              <span>{d.label}</span>
              <span className="tabular">{formatRp(d.amount)}</span>
            </div>
          ))}
        </div>
      ))}
      <AmountRow label={total} value={formatSignedJt(section.net)} color={section.net < 0 ? RED : GREEN} bold />
    </>
  )
}

function ArusKas({ cf, netProfit }: { cf: CashFlow; netProfit: number }) {
  const [metode, setMetode] = useState<'langsung' | 'tidak-langsung'>('langsung')
  const operasi = metode === 'langsung' ? cf.operasi_langsung : cf.operasi_tidak_langsung
  const cocok = cf.operasi_langsung.net === cf.operasi_tidak_langsung.net

  return (
    <>
      <div className="kpi-grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(210px, 1fr))' }}>
        <div className="card">
          <div className="kpi-label">Saldo Kas Awal Periode</div>
          <div className="kpi-value tabular">{formatJt(cf.opening_balance)}</div>
        </div>
        <div className="card">
          <div className="kpi-label">Arus Kas Bersih Periode Ini</div>
          <div className="kpi-value tabular" style={{ color: cf.net_cash < 0 ? RED : GREEN }}>{formatSignedJt(cf.net_cash)}</div>
          <div className="kpi-meta-row">
            <span>Masuk {formatRp(cf.cash_in)} · Keluar {formatRp(cf.cash_out)}</span>
          </div>
        </div>
        <div className="card">
          <div className="kpi-label">Saldo Kas Akhir Periode</div>
          <div className="kpi-value tabular" style={{ color: BLUE }}>{formatJt(cf.closing_balance)}</div>
        </div>
      </div>

      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 14, flexWrap: 'wrap', margin: '14px 0' }}>
        <div className="segmented" role="tablist" aria-label="Metode penyusunan arus kas">
          <button role="tab" className="seg-btn" aria-selected={metode === 'langsung'} onClick={() => setMetode('langsung')}>
            Metode Langsung
          </button>
          <button role="tab" className="seg-btn" aria-selected={metode === 'tidak-langsung'} onClick={() => setMetode('tidak-langsung')}>
            Metode Tidak Langsung
          </button>
        </div>
        <span className={'pill-status ' + (cocok ? 'pill-green' : 'pill-red')}>
          {cocok
            ? `Kas operasi kedua metode cocok: ${formatRp(cf.operasi_langsung.net)}`
            : 'Kas operasi dua metode berbeda — periksa data'}
        </span>
      </div>

      <div className="report-grid">
        <div className="card" style={{ padding: 20 }}>
          <div className="chart-title">
            Arus Kas — {metode === 'langsung' ? 'Metode Langsung' : 'Metode Tidak Langsung'}
          </div>
          <div className="chart-subtitle">
            {metode === 'langsung'
              ? 'Menelusuri uang yang benar-benar masuk & keluar kas'
              : 'Mulai dari laba bersih, lalu disesuaikan sampai ketemu kas'}
          </div>
          <div className="list-rows tabular">
            <Section title="A. Aktivitas Operasi" section={operasi} total="Kas Bersih dari Aktivitas Operasi" />
            <Section title="B. Aktivitas Investasi" section={cf.investasi} total="Kas Bersih dari Aktivitas Investasi" />
            <Section title="C. Aktivitas Pendanaan" section={cf.pendanaan} total="Kas Bersih dari Aktivitas Pendanaan" />
            <div style={{ height: 8 }} />
            <AmountRow label="Kenaikan Kas Bersih (A + B + C)" value={formatSignedJt(cf.net_cash)} color={cf.net_cash < 0 ? RED : GREEN} bold />
            <AmountRow label="Saldo Kas Awal Periode" value={formatRp(cf.opening_balance)} />
            <AmountRow
              label="SALDO KAS AKHIR PERIODE"
              value={formatRp(cf.closing_balance)}
              color={BLUE}
              bold
              style={{ border: '1px solid rgba(56,189,248,0.3)' }}
            />
          </div>
        </div>

        <div className="card" style={{ padding: 20 }}>
          <div className="chart-title">Cara Membacanya</div>
          <div className="chart-subtitle">
            {metode === 'langsung' ? 'Untuk pemilik usaha: "uangnya ke mana?"' : 'Untuk bank, investor & pelaporan pajak'}
          </div>
          {metode === 'langsung' ? (
            <div className="note-box">
              Metode ini mendaftar <strong>penerimaan dan pembayaran kas apa adanya</strong>, seperti mutasi
              rekening. Tiap baris bisa dicocokkan ke transaksi yang sudah berstatus lunas. Penjualan tempo,
              biaya yang belum dibayar, dan penyusutan tidak muncul di sini karena tidak ada uang yang bergerak.
            </div>
          ) : (
            <div className="note-box">
              Metode ini menjawab <strong>kenapa laba {formatJt(netProfit)} tapi kas bergerak {formatSignedJt(cf.net_cash)}</strong>.
              Beban non-kas dibalik lagi, lalu piutang dan utang disesuaikan. Ini format standar yang diminta bank dan PSAK.
            </div>
          )}
          <div className="list-rows tabular" style={{ marginTop: 16 }}>
            <div className="section-head">Jembatan laba → kas</div>
            <AmountRow label="Laba bersih" value={formatRp(netProfit)} color={GREEN} />
            <AmountRow label="Beban non-kas (penyusutan + amortisasi)" value={formatSignedJt(cf.depreciation + cf.amortization)} color={GREEN} />
            <AmountRow label="Tertahan di piutang" value={formatSignedJt(-cf.new_receivables)} color={RED} />
            <AmountRow label="Tertunda di utang" value={formatSignedJt(cf.new_payables)} color={GREEN} />
            <AmountRow label="Kas bersih dari operasi" value={formatRp(cf.operasi_tidak_langsung.net)} color={BLUE} bold />
          </div>
          <div className="note-box warn">
            Perubahan persediaan belum ikut dihitung, dan piutang masih diukur sebagai piutang yang terbit
            periode ini — bukan selisih saldo awal dan akhir. Perlu dibenahi sebelum dipakai antar-periode.
          </div>
        </div>
      </div>
    </>
  )
}
