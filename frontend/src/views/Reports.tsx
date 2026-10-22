import { useEffect, useState } from 'react'
import { FileDown, FileSpreadsheet } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { AmountRow, PageHeader } from '@/components/PageHeader'
import { Aset } from '@/views/Aset'
import { LaporanAI } from '@/views/LaporanAI'
import { BandingLaporan } from '@/views/BandingLaporan'
import { ekspor } from '@/lib/garansi'
import { daftarUrl, getLaporanRentang, laporanPdfUrl, type AssetSummary, type LaporanRentang, type BalanceSheet, type BagianPDF, type CashFlow, type CashFlowSection, type IncomeStatement, type PeriodOption } from '@/lib/api'
import { formatRp, formatPct, formatSignedRp } from '@/lib/format'
import { pilihanCepat, rentangBulan } from '@/lib/rentang'

const RED = 'var(--text-danger)'
const GREEN = 'var(--accent-green)'
const BLUE = 'var(--accent-blue)'

type Tab = 'laba-rugi' | 'neraca' | 'arus-kas' | 'banding' | 'aset' | 'ai'
type Metode = 'langsung' | 'tidak-langsung'

const TABS: { id: Tab; label: string; badge?: string }[] = [
  { id: 'laba-rugi', label: 'Laba Rugi' },
  { id: 'neraca', label: 'Neraca' },
  { id: 'arus-kas', label: 'Arus Kas', badge: '2 METODE' },
  { id: 'banding', label: 'Banding', badge: 'MULTI-BULAN' },
  { id: 'aset', label: 'Aset & Penyusutan' },
  { id: 'ai', label: 'Laporan AI', badge: 'ANALISIS' },
]

const SUBTITLE: Record<Tab, string> = {
  'laba-rugi': 'Laba Rugi, Neraca, Arus Kas & Aset dalam satu tempat',
  neraca: 'Posisi keuangan per akhir periode',
  'arus-kas': 'Dua metode penyajian: Langsung & Tidak Langsung',
  banding: 'Bandingkan beberapa bulan sekaligus, lengkap dengan totalnya',
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
  periods: PeriodOption[]
  /** Tab yang dibuka pertama, mis. 'banding' dari tautan Tanya AI. */
  tabAwal?: string
  /** false di mode Lihat sebagai klien: server menolak ekspor. */
  bisaEkspor: boolean
  onAssetsChanged: () => void
}

/** Satu menu untuk seluruh laporan keuangan; tiap laporan jadi tab. */
export function LaporanKeuangan({ r, bs, cf, assets, period, periodLabel, periods, tabAwal, bisaEkspor, onAssetsChanged }: Props) {
  const [tab, setTab] = useState<Tab>(() => TABS.find((t) => t.id === tabAwal)?.id ?? 'laba-rugi')
  const [metode, setMetode] = useState<Metode>('langsung')
  const bagianTab: BagianPDF | null = tab === 'laba-rugi' || tab === 'neraca' || tab === 'arus-kas' || tab === 'aset' ? tab : null
  // Rentang tanggal laba rugi & arus kas; bawaan = sebulan penuh periode yang dipilih.
  const [rentang, setRentang] = useState(() => rentangBulan(period))
  const [hasil, setHasil] = useState<{ kunci: string; d?: LaporanRentang; error?: string } | null>(null)
  useEffect(() => { setRentang(rentangBulan(period)) }, [period])
  const bulanan = rentangBulan(period)
  const custom = rentang.dari !== bulanan.dari || rentang.sampai !== bulanan.sampai
  const kunci = `${rentang.dari}|${rentang.sampai}`
  const salah = rentang.dari && rentang.sampai && rentang.dari > rentang.sampai ? 'Tanggal sampai tidak boleh sebelum tanggal dari.' : ''
  useEffect(() => {
    if (!custom || salah || !rentang.dari || !rentang.sampai) return
    let hidup = true
    getLaporanRentang(rentang.dari, rentang.sampai).then((d) => hidup && setHasil({ kunci, d }), (e: Error) => hidup && setHasil({ kunci, error: e.message }))
    return () => { hidup = false }
  }, [custom, salah, kunci, rentang.dari, rentang.sampai])
  const dataRentang = custom && hasil?.kunci === kunci ? hasil.d : undefined
  const rr = custom ? dataRentang?.income : r
  const cff = custom ? dataRentang?.cashflow : cf
  const labelRentang = custom ? (dataRentang?.label ?? 'memuat…') : periodLabel
  const pakaiRentang = custom && (tab === 'laba-rugi' || tab === 'arus-kas')
  const pdfRentang = custom && !salah ? rentang : undefined

  return (
    <>
      <PageHeader title="Laporan Keuangan" subtitle={`${pakaiRentang ? labelRentang : periodLabel} · ${SUBTITLE[tab]}`}>
        {(tab === 'laba-rugi' || tab === 'neraca') && (
          <span className={'pill-status ' + (bs.is_balanced ? 'pill-green' : 'pill-amber')}>
            {bs.is_balanced ? 'Aktiva = Pasiva (Seimbang)' : 'Aktiva ≠ Pasiva, periksa data'}
          </span>
        )}
        {bisaEkspor && bagianTab && (
          <Button variant="outline" onClick={() => void ekspor(laporanPdfUrl(period, bagianTab, bagianTab === 'arus-kas' ? metode : undefined, bagianTab === 'laba-rugi' || bagianTab === 'arus-kas' ? pdfRentang : undefined))}>
            <FileDown /> Unduh PDF {TABS.find((t) => t.id === tab)?.label}
          </Button>
        )}
        {bisaEkspor && tab !== 'banding' && (
          <Button onClick={() => void ekspor(laporanPdfUrl(period, 'semua', metode, pdfRentang))} title="Laba Rugi, Neraca, dan Arus Kas dalam satu PDF">
            <FileDown /> Unduh semua laporan (PDF)
          </Button>
        )}
      </PageHeader>

      {bisaEkspor && tab !== 'banding' && (
        <div className="flex flex-wrap items-center justify-end gap-2" style={{ marginBottom: 10 }}>
          <span className="text-sm text-muted-foreground">Daftar piutang, utang, invoice, barang, pelanggan &amp; pemasok {periodLabel}:</span>
          <Button size="sm" variant="outline" onClick={() => void ekspor(daftarUrl(period, 'xlsx'))}><FileSpreadsheet /> Unduh Excel</Button>
          <Button size="sm" variant="outline" onClick={() => void ekspor(daftarUrl(period, 'pdf'))}><FileDown /> Unduh PDF</Button>
        </div>
      )}

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

      {(tab === 'laba-rugi' || tab === 'arus-kas') && (
        <div className="card" style={{ padding: 16, marginBottom: 14 }}>
          <div className="flex flex-wrap items-end gap-3">
            <label className="grid gap-1"><span className="kpi-label">Dari tanggal</span>
              <input className="input-field" type="date" value={rentang.dari} onChange={(e) => setRentang({ ...rentang, dari: e.target.value })} /></label>
            <label className="grid gap-1"><span className="kpi-label">Sampai tanggal</span>
              <input className="input-field" type="date" value={rentang.sampai} onChange={(e) => setRentang({ ...rentang, sampai: e.target.value })} /></label>
            <div className="flex flex-wrap gap-1.5">
              {pilihanCepat(period).map((p) => (
                <button key={p.label} type="button" className="pill-status" style={{ cursor: 'pointer', outline: rentang.dari === p.dari && rentang.sampai === p.sampai ? '1px solid var(--text-brand)' : undefined }}
                  onClick={() => setRentang({ dari: p.dari, sampai: p.sampai })}>{p.label}</button>
              ))}
            </div>
          </div>
          {salah && <p className="pill-status pill-red" role="alert" style={{ marginTop: 10 }}>{salah}</p>}
          {custom && hasil?.kunci === kunci && hasil.error && <p className="pill-status pill-red" role="alert" style={{ marginTop: 10 }}>{hasil.error}</p>}
          <p className="text-xs text-muted-foreground" style={{ marginTop: 8 }}>
            {custom ? <><strong>{labelRentang}</strong>. </> : null}
            Penyusutan aset dihitung sebulan penuh untuk tiap bulan yang akhir bulannya masuk rentang. Neraca tetap posisi akhir periode {periodLabel}.
          </p>
        </div>
      )}
      {tab === 'laba-rugi' && (rr && cff ? <LabaRugi r={rr} cf={cff} /> : <p className="workspace-subtitle">Menghitung laporan…</p>)}
      {tab === 'neraca' && <Neraca bs={bs} kartuAktif={!!r.hpp_stok} />}
      {tab === 'arus-kas' && (rr && cff
        ? <ArusKas cf={cff} netProfit={rr.net_profit} metode={metode} setMetode={setMetode} kartuAktif={!!rr.hpp_stok} />
        : <p className="workspace-subtitle">Menghitung laporan…</p>)}
      {tab === 'banding' && <BandingLaporan periods={periods} period={period} bisaEkspor={bisaEkspor} />}
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
          <AmountRow label="Omzet (penjualan kotor)" value={formatRp(r.omzet)} color="var(--text-brand)" bold />
          <AmountRow label="Potongan channel / aplikasi" value={formatRp(-r.channel_fees)} color={RED} />
          <AmountRow label="Penjualan bersih" value={formatRp(r.net_sales)} bold />
          <AmountRow label={r.hpp_stok ? 'HPP (modal barang terjual)' : 'HPP (bahan baku)'} value={formatRp(-r.cogs)} color={RED} />
          {r.hpp_stok && (() => {
            const h = r.hpp_stok
            const takDilacak = r.cogs - h.hpp_jual + h.selisih_opname + h.koreksi
            return <>
              <div className="list-row sub"><span>Modal barang terjual (kartu stok)</span><span className="tabular">{formatRp(-h.hpp_jual)}</span></div>
              {h.selisih_opname !== 0 && <div className="list-row sub"><span>Selisih stok opname {h.selisih_opname < 0 ? '(hilang / rusak)' : '(lebih)'}</span><span className="tabular">{formatRp(h.selisih_opname)}</span></div>}
              {h.koreksi !== 0 && <div className="list-row sub"><span>Koreksi nilai stok minus</span><span className="tabular">{formatRp(h.koreksi)}</span></div>}
              {takDilacak !== 0 && <div className="list-row sub"><span>Belanja bahan tanpa barang di stok</span><span className="tabular">{formatRp(-takDilacak)}</span></div>}
            </>
          })()}
          <AmountRow label="Laba kotor" value={formatRp(r.gross_profit)} bold />
          <div className="section-head">Beban operasional</div>
          {(r.opex_lines ?? []).map((l, i) => (
            <AmountRow key={i} label={l.label} value={formatRp(-l.amount)} color={RED} />
          ))}
          <AmountRow label="Total beban operasional (tunai)" value={formatRp(-r.opex)} color={RED} bold />
          <AmountRow label="EBITDA" value={formatRp(r.ebitda)} color={r.ebitda < 0 ? RED : 'var(--text-brand)'} bold
            style={{ border: '1px solid color-mix(in srgb, var(--accent-blue) 30%, transparent)' }} />
          <div className="list-row sub"><span>Laba sebelum penyusutan & amortisasi — kemampuan usaha menghasilkan kas dari operasi</span><span /></div>
          <div className="section-head">Penutupan bulan: penyusutan & amortisasi</div>
          <AmountRow label="Beban penyusutan aset tetap" value={formatRp(-r.depreciation)} color={RED} />
          <AmountRow label="Beban amortisasi harta tak berwujud" value={formatRp(-r.amortization)} color={RED} />
          <AmountRow label="Laba usaha (EBIT)" value={formatRp(r.ebit)} bold />
          {r.other_income !== 0 && <AmountRow label="Pendapatan lain" value={formatRp(r.other_income)} color={GREEN} />}
          <AmountRow label="Laba bersih" value={formatRp(r.net_profit)} color={r.net_profit < 0 ? RED : GREEN} bold />
        </div>
      </div>

      <div className="card" style={{ padding: 20 }}>
        <div className="chart-title">Jembatan ke Tab Lain</div>
        <div className="chart-subtitle">Angka kunci yang dipakai Neraca, Arus Kas &amp; Aset</div>
        <div className="list-rows tabular">
          <AmountRow label="Laba bersih → Modal di Neraca" value={formatRp(r.net_profit)} color={GREEN} bold />
          <AmountRow label="Kenaikan kas riil → tab Arus Kas" value={formatSignedRp(cf.net_cash)} color={BLUE} bold />
          <AmountRow label="Selisih laba vs kas" value={formatSignedRp(-selisih)} color={selisih === 0 ? GREEN : RED} />
        </div>
        {selisih !== 0 && (
          <div className="note-box">
            <strong>Kenapa laba {formatRp(r.net_profit)} tapi kas bergerak {formatSignedRp(cf.net_cash)}?</strong>{' '}
            {cf.new_receivables > 0 && <>{formatRp(cf.new_receivables)} masih tertahan di piutang. </>}
            {cf.new_payables > 0 && <>{formatRp(cf.new_payables)} biaya sudah dicatat tapi belum dibayar. </>}
            {nonKas > 0 && <>{formatRp(nonKas)} penyusutan &amp; amortisasi tidak mengeluarkan uang sama sekali. </>}
            {!!cf.kenaikan_persediaan && cf.kenaikan_persediaan > 0 && <>{formatRp(cf.kenaikan_persediaan)} jadi stok yang belum terjual. </>}
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

function Neraca({ bs, kartuAktif }: { bs: BalanceSheet; kartuAktif: boolean }) {
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
          {!!a.other_assets && <AmountRow label="Aset Lancar Lain (uang muka, dibayar di muka)" value={formatRp(a.other_assets)} />}
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
          {!!bs.liabilities.other && <AmountRow label="Kewajiban Lain (utang pajak, beban terutang)" value={formatRp(bs.liabilities.other)} color={RED} />}

          <div className="section-head">Modal (Ekuitas)</div>
          <AmountRow label="TOTAL MODAL" value={formatRp(bs.equity.total)} color={GREEN} bold />
          <AmountRow label="Modal Awal (saldo awal pembukuan)" value={formatRp(bs.equity.opening_capital)} />
          {bs.equity.added_capital !== 0 && (
            <AmountRow label="Tambahan Modal (setoran − ambil pribadi + aset & stok tanpa kas)" value={formatRp(bs.equity.added_capital)} />
          )}
          <AmountRow label="Laba Ditahan (saldo awal + laba bulan-bulan sebelumnya)" value={formatRp(bs.equity.retained_earnings)} color={bs.equity.retained_earnings < 0 ? RED : GREEN} />
          <AmountRow label="Laba Bersih Periode Berjalan" value={formatRp(bs.equity.current_profit)} color={bs.equity.current_profit < 0 ? RED : GREEN} />

          <AmountRow
            label="TOTAL PASIVA (Utang + Modal)"
            value={formatRp(bs.liabilities.total + bs.equity.total)}
            color={BLUE}
            bold
            style={{ border: '1px solid color-mix(in srgb, var(--accent-blue) 30%, transparent)', marginTop: 6 }}
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
          · Kas {formatRp(a.cash)} = saldo akhir di tab Arus Kas.<br />
          · Piutang &amp; utang = penyesuaian modal kerja di Arus Kas metode tidak langsung.<br />
          · Nilai buku aset {formatRp(a.fixed_assets)} = total tab Aset &amp; Penyusutan.<br />
          · Laba {formatRp(bs.equity.current_profit)} = baris Laba Bersih di tab Laba Rugi.
        </div>
        {!kartuAktif && (
          <div className="note-box warn">
            Nilai persediaan masih angka saldo awal karena kartu stok belum dimulai. Mulai di menu Stok &amp; Gudang supaya persediaan & HPP dihitung dari pergerakan barang.
          </div>
        )}
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
            value={l.non_kas ? 'Rp0' : formatSignedRp(l.amount)}
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
      <AmountRow label={total} value={formatSignedRp(section.net)} color={section.net < 0 ? RED : GREEN} bold />
    </>
  )
}

function ArusKas({ cf, netProfit, metode, setMetode, kartuAktif }: { cf: CashFlow; netProfit: number; metode: Metode; setMetode: (m: Metode) => void; kartuAktif: boolean }) {
  const operasi = metode === 'langsung' ? cf.operasi_langsung : cf.operasi_tidak_langsung
  const cocok = cf.operasi_langsung.net === cf.operasi_tidak_langsung.net

  return (
    <>
      <div className="kpi-grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(210px, 1fr))' }}>
        <div className="card">
          <div className="kpi-label">Saldo Kas Awal Periode</div>
          <div className="kpi-value tabular">{formatRp(cf.opening_balance)}</div>
        </div>
        <div className="card">
          <div className="kpi-label">Arus Kas Bersih Periode Ini</div>
          <div className="kpi-value tabular" style={{ color: cf.net_cash < 0 ? RED : GREEN }}>{formatSignedRp(cf.net_cash)}</div>
          <div className="kpi-meta-row">
            <span>Masuk {formatRp(cf.cash_in)} · Keluar {formatRp(cf.cash_out)}</span>
          </div>
        </div>
        <div className="card">
          <div className="kpi-label">Saldo Kas Akhir Periode</div>
          <div className="kpi-value tabular" style={{ color: BLUE }}>{formatRp(cf.closing_balance)}</div>
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
            <AmountRow label="Kenaikan Kas Bersih (A + B + C)" value={formatSignedRp(cf.net_cash)} color={cf.net_cash < 0 ? RED : GREEN} bold />
            <AmountRow label="Saldo Kas Awal Periode" value={formatRp(cf.opening_balance)} />
            <AmountRow
              label="SALDO KAS AKHIR PERIODE"
              value={formatRp(cf.closing_balance)}
              color={BLUE}
              bold
              style={{ border: '1px solid color-mix(in srgb, var(--accent-blue) 30%, transparent)' }}
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
              Metode ini menjawab <strong>kenapa laba {formatRp(netProfit)} tapi kas bergerak {formatSignedRp(cf.net_cash)}</strong>.
              Beban non-kas dibalik lagi, lalu piutang dan utang disesuaikan. Ini format standar yang diminta bank dan PSAK.
            </div>
          )}
          <div className="list-rows tabular" style={{ marginTop: 16 }}>
            <div className="section-head">Jembatan laba → kas</div>
            <AmountRow label="Laba bersih" value={formatRp(netProfit)} color={GREEN} />
            <AmountRow label="Beban non-kas (penyusutan + amortisasi)" value={formatSignedRp(cf.depreciation + cf.amortization)} color={GREEN} />
            <AmountRow label="Tertahan di piutang" value={formatSignedRp(-cf.new_receivables)} color={RED} />
            <AmountRow label="Tertunda di utang" value={formatSignedRp(cf.new_payables)} color={GREEN} />
            {kartuAktif && !!cf.kenaikan_persediaan && (
              <AmountRow label={cf.kenaikan_persediaan > 0 ? 'Tertahan di persediaan stok' : 'Persediaan stok berkurang'}
                value={formatSignedRp(-cf.kenaikan_persediaan)} color={cf.kenaikan_persediaan > 0 ? RED : GREEN} />
            )}
            <AmountRow label="Kas bersih dari operasi" value={formatRp(cf.operasi_tidak_langsung.net)} color={BLUE} bold />
          </div>
          <div className="note-box warn">
            {!kartuAktif
              ? 'Perubahan persediaan belum ikut dihitung (kartu stok belum dimulai), dan piutang'
              : 'Kenaikan persediaan dihitung dari kartu stok. Piutang'}{' '}
            masih diukur sebagai piutang yang terbit periode ini — bukan selisih saldo awal dan akhir.
          </div>
        </div>
      </div>
    </>
  )
}
