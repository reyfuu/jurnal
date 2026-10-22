import { useEffect, useState } from 'react'
import { tanggalAkhirPeriode } from '@/lib/rentang'
import { angkaSifat } from '@/lib/angkaLaporan'
import { AlertTriangle, CheckCircle2, Lightbulb, Sparkles, TrendingUp } from 'lucide-react'
import { CekAwalCard } from '@/components/CekAwalCard'
import { getAIReport, type AIReport, type BusinessFacts } from '@/lib/api'
import { formatRp } from '@/lib/format'

const AMBER = 'var(--text-warn)'
const SOROT = 'var(--text-brand)'
const RED = 'var(--text-danger)'
// beban & HPP hitam tanpa minus, kontra dalam kurung (lib/angkaLaporan), dengan awalan Rp
const rpSifat = (v: number, sifat: 'pengurang' | 'kontra') => angkaSifat(v, sifat).teks.replace(/^(\(?)/, '$1Rp')
const GREEN = 'var(--accent-green)'
const BLUE = 'var(--accent-blue)'

const persenBps = (bps: number) => (bps / 100).toFixed(2).replace('.', ',') + '%'

const DAMPAK: Record<string, { label: string; warna: string }> = {
  tinggi: { label: 'dampak tinggi', warna: RED },
  sedang: { label: 'dampak sedang', warna: AMBER },
  rendah: { label: 'dampak rendah', warna: 'var(--text-muted)' },
}

/**
 * Tab Laporan AI: seluruh angka laporan periode ini, dibaca dan dianalisis model.
 *
 * Rincian di bawah selalu tampil karena dihitung server. Bagian analisis hanya muncul bila
 * LLM sudah dikonfigurasi — tanpa itu halaman tetap berguna sebagai ringkasan menyeluruh.
 */
export function LaporanAI({ period }: { period: string }) {
  const [r, setR] = useState<AIReport | null>(null)
  const [error, setError] = useState('')

  // Pemanggil memberi key={period}, jadi ganti periode me-remount komponen ini dan state
  // mulai bersih tanpa perlu direset manual di dalam effect.
  useEffect(() => {
    getAIReport(period).then(setR, (e) => setError((e as Error).message))
  }, [period])

  if (error) return <div className="pill-status pill-red" role="alert">{error}</div>
  if (!r) return <p className="workspace-subtitle">Membaca laporan…</p>

  const a = r.analysis
  if (r.sumber === 'profil' && r.profil_usaha) {
    return (
      <>
        <div className="card" style={{ padding: '14px 18px', marginBottom: 16, borderColor: 'color-mix(in srgb, var(--accent-blue) 30%, transparent)' }}>
          <div className="chart-title">Analisis awal dari data pendaftaran</div>
          <p style={{ margin: '6px 0 0', fontSize: 13, color: 'var(--text-muted)', lineHeight: 1.7 }}>
            Belum ada transaksi tercatat, jadi analisis ini hanya memakai perkiraan omzet dan laba setahun yang Anda isi saat mendaftar.
            Laba rugi, arus kas, dan neraca dianalisis setelah transaksi dicatat.
          </p>
        </div>
        <AnalisisAI a={a} llm={r.llm} profil />
        <CekAwalCard profile={r.profil_usaha} />
      </>
    )
  }
  const f = r.facts!

  return (
    <>
      {/* Analisis di atas: itu yang dicari orang saat membuka tab ini. */}
      <AnalisisAI a={a} llm={r.llm} f={f} />

      {/* Peringatan: fakta objektif, bukan pendapat model — selalu tampil. */}
      {f.peringatan && f.peringatan.length > 0 && (
        <div className="card" style={{ padding: 18, marginBottom: 16, borderColor: 'color-mix(in srgb, var(--text-danger) 30%, transparent)' }}>
          <div className="chart-title" style={{ color: RED }}>Perlu diperiksa</div>
          <ul style={{ margin: '10px 0 0', paddingLeft: 20, fontSize: 13, lineHeight: 1.8 }}>
            {f.peringatan.map((p, i) => <li key={i}>{p}</li>)}
          </ul>
        </div>
      )}

      <RincianLaporan f={f} period={period} />
    </>
  )
}

function RincianLaporan({ f, period }: { f: BusinessFacts; period: string }) {
  const lr = f.laba_rugi
  return (
    <>
      <div className="kpi-grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(190px, 1fr))', marginBottom: 14 }}>
        <KPI label="Omzet" v={lr.omzet} banding={lr.omzet_sebelumnya} />
        <KPI label="Laba (Rugi) Bersih" v={lr.laba_bersih} banding={lr.laba_sebelumnya} warna={lr.laba_bersih >= 0 ? GREEN : RED} />
        <KPI label="Margin Laba" teks={persenBps(lr.margin_bps)} warna={SOROT} />
        <KPI label="Saldo Kas Akhir" v={f.arus_kas.saldo_akhir} warna={BLUE} />
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(330px, 1fr))', gap: 14 }}>
        <Kartu judul="Laba Rugi">
          <Baris l="Omzet" v={lr.omzet} />
          <Baris l="Harga pokok penjualan" teks={rpSifat(-lr.hpp, 'pengurang')} />
          <Baris l="Laba (rugi) kotor" v={lr.laba_kotor} tebal />
          <Baris l="Beban operasional" teks={rpSifat(-lr.beban_operasional, 'pengurang')} />
          <Baris l="Beban penyusutan" teks={rpSifat(-lr.penyusutan, 'pengurang')} />
          <Baris l="Beban amortisasi" teks={rpSifat(-lr.amortisasi, 'pengurang')} />
          <Baris l="Laba (rugi) bersih tahun berjalan" v={lr.laba_bersih} tebal warna={lr.laba_bersih >= 0 ? GREEN : RED} />
        </Kartu>

        <Kartu judul="Arus Kas">
          <Baris l="Saldo kas awal" v={f.arus_kas.saldo_awal} />
          <Baris l="Kas masuk" v={f.arus_kas.kas_masuk} />
          <Baris l="Kas keluar" v={-f.arus_kas.kas_keluar} />
          <Baris l="Operasi (langsung)" v={f.arus_kas.operasi_langsung} />
          <Baris l="Operasi (tidak langsung)" v={f.arus_kas.operasi_tidak_langsung} />
          <Baris l="Investasi" v={f.arus_kas.investasi} />
          <Baris l="Pendanaan" v={f.arus_kas.pendanaan} />
          <Baris l="Saldo kas akhir" v={f.arus_kas.saldo_akhir} tebal warna={BLUE} />
        </Kartu>

        <Kartu judul={`Neraca per ${tanggalAkhirPeriode(period)}`} sub={f.neraca.seimbang ? 'Aktiva = Pasiva (seimbang)' : 'Aktiva ≠ Pasiva, periksa data'}>
          <Baris l="Kas" v={f.neraca.kas} />
          <Baris l="Piutang" v={f.neraca.piutang} />
          <Baris l="Persediaan" v={f.neraca.persediaan} />
          <Baris l="Total aktiva" v={f.neraca.total_aktiva} tebal warna={BLUE} />
          <Baris l="Utang" v={f.neraca.utang} />
          <Baris l="Ekuitas" v={f.neraca.ekuitas} />
          <Baris l="Total pasiva" v={f.neraca.total_pasiva} tebal warna={GREEN} />
        </Kartu>

        <Kartu judul="Aset & Penyusutan" sub={`${f.aset.jumlah} aset terdaftar`}>
          <Baris l="Harga perolehan" v={f.aset.harga_perolehan} />
          <Baris l="Akumulasi penyusutan" v={-f.aset.akumulasi} />
          <Baris l="Nilai buku" v={f.aset.nilai_buku} tebal warna={BLUE} />
          <Baris l="Beban periode ini" v={f.aset.beban_bulan_ini} warna={SOROT} />
        </Kartu>

        <Kartu judul="Pajak" sub={f.pajak.bentuk_badan}>
          <Baris l="Omzet kumulatif tahun ini" v={f.pajak.omzet_kumulatif} />
          <Baris l="Batas bebas pajak" v={f.pajak.batas_bebas_pajak} />
          <Baris l="Sisa batas bebas" v={f.pajak.sisa_batas_bebas} />
          <Baris l="Dasar kena pajak" v={f.pajak.dasar_kena_pajak} />
          <Baris l="PPh final terutang" v={f.pajak.pajak_terutang} tebal warna={SOROT} />
        </Kartu>

        <Kartu judul="Tagihan">
          <Baris l="Piutang" v={f.tagihan.piutang_total} />
          <Baris l="Utang" v={f.tagihan.utang_total} />
          <Baris l="Invoice belum dibayar" v={f.tagihan.invoice_belum_dibayar} />
          <Baris l={`Lewat jatuh tempo (${f.tagihan.jumlah_faktur_telat} faktur)`} v={f.tagihan.piutang_lewat_tempo}
            tebal warna={f.tagihan.jumlah_faktur_telat ? RED : undefined} />
        </Kartu>

        <Kartu judul="Operasional">
          <Baris l="Jumlah transaksi" teks={String(f.operasi.jumlah_transaksi)} />
          <Baris l="Biaya aplikasi / channel" v={f.operasi.biaya_aplikasi} />
          <Baris l="Karyawan aktif" teks={String(f.operasi.jumlah_karyawan)} />
          <Baris l="Beban gaji bulanan" v={f.operasi.beban_gaji_bulanan} />
          {f.operasi.stok_menipis && f.operasi.stok_menipis.length > 0 && (
            <div style={{ marginTop: 8, fontSize: 12.5, color: RED, lineHeight: 1.6 }}>
              Stok menipis: {f.operasi.stok_menipis.join(', ')}
            </div>
          )}
        </Kartu>

        {f.operasi.channel && f.operasi.channel.length > 0 && (
          <Kartu judul="Penjualan per Channel">
            {f.operasi.channel.map((c) => (
              <Baris key={c.name} l={`${c.name} (${c.pct}%)`} v={c.net} />
            ))}
          </Kartu>
        )}

        {f.operasi.produk_teratas && f.operasi.produk_teratas.length > 0 && (
          <Kartu judul="Produk Teratas">
            {f.operasi.produk_teratas.map((p) => (
              <Baris key={p.name} l={`${p.name} · ${p.qty} unit`} v={p.omzet} />
            ))}
          </Kartu>
        )}
      </div>

      {f.tren.length > 0 && (
        <div className="card" style={{ padding: 20, marginTop: 14 }}>
          <div className="chart-title">Tren Bulanan</div>
          <div className="table-scroll">
            <table className="data-table tabular" style={{ minWidth: 520 }}>
              <thead>
                <tr>
                  <th>Bulan</th>
                  <th style={{ textAlign: 'right' }}>Omzet</th>
                  <th style={{ textAlign: 'right' }}>Laba (Rugi) Bersih</th>
                  <th style={{ textAlign: 'right' }}>Saldo Kas</th>
                </tr>
              </thead>
              <tbody>
                {f.tren.map((t) => (
                  <tr key={t.bulan}>
                    <td>{t.bulan}</td>
                    <td style={{ textAlign: 'right' }}>{formatRp(t.omzet)}</td>
                    <td style={{ textAlign: 'right', color: t.laba_bersih >= 0 ? GREEN : RED }}>{formatRp(t.laba_bersih)}</td>
                    <td style={{ textAlign: 'right' }}>{formatRp(t.saldo_kas)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </>
  )
}

function KPI({ label, v, teks, warna, banding }: { label: string; v?: number; teks?: string; warna?: string; banding?: number }) {
  const delta = v !== undefined && banding !== undefined && banding !== 0
    ? Math.round(((v - banding) / Math.abs(banding)) * 1000) / 10
    : null
  return (
    <div className="card" style={{ padding: '14px 18px' }}>
      <div className="kpi-label">{label}</div>
      <div className="kpi-value tabular" style={{ fontSize: 20, color: warna }}>{teks ?? formatRp(v ?? 0)}</div>
      {delta !== null && (
        <div className="kpi-meta-row" style={{ color: delta >= 0 ? GREEN : RED }}>
          {delta >= 0 ? '▲' : '▼'} {Math.abs(delta).toString().replace('.', ',')}% vs bulan lalu
        </div>
      )}
    </div>
  )
}

function Kartu({ judul, sub, children }: { judul: string; sub?: string; children: React.ReactNode }) {
  return (
    <div className="card" style={{ padding: 20, justifyContent: 'flex-start' }}>
      <div className="chart-title">{judul}</div>
      {sub && <div className="chart-subtitle">{sub}</div>}
      <div className="list-rows tabular" style={{ marginTop: 10 }}>{children}</div>
    </div>
  )
}

function Baris({ l, v, teks, tebal, warna }: { l: string; v?: number; teks?: string; tebal?: boolean; warna?: string }) {
  return (
    <div style={{
      display: 'flex', justifyContent: 'space-between', gap: 16, padding: '7px 0',
      fontWeight: tebal ? 700 : 400, fontSize: 13,
      borderTop: tebal ? '1px solid var(--border-strong)' : undefined,
      marginTop: tebal ? 4 : undefined,
    }}>
      <span style={{ color: tebal ? undefined : 'var(--text-muted)' }}>{l}</span>
      <span style={{ color: warna, whiteSpace: 'nowrap' }}>{teks ?? formatRp(v ?? 0)}</span>
    </div>
  )
}

/** Bagian analisis model: dipakai laporan dari transaksi dan dari data pendaftaran. Dibuat
 *  menonjol (kartu navy, angka kunci, poin berikon) karena inilah yang dicari di tab ini. */
function AnalisisAI({ a, llm, profil, f }: { a: AIReport['analysis']; llm: AIReport['llm']; profil?: boolean; f?: BusinessFacts }) {
  if (!a.tersedia) {
    return (
      <div className="ai-hero ai-hero-mati" style={{ marginBottom: 16 }}>
        <div className="ai-badge"><Sparkles size={13} aria-hidden="true" /> Analisis AI</div>
        <div className="ai-judul">Analisis AI belum aktif</div>
        <p className="ai-ringkas" style={{ fontSize: 14 }}>{a.alasan ?? llm.petunjuk}</p>
        {llm.petunjuk && a.alasan && <p style={{ margin: 0, fontSize: 12.5, opacity: 0.85 }}>{llm.petunjuk}</p>}
        <p style={{ margin: 0, fontSize: 12.5, opacity: 0.85 }}>
          {profil ? 'Cek awal di bawah tetap lengkap karena dihitung sendiri oleh aplikasi.' : 'Rincian laporan di bawah tetap lengkap karena dihitung sendiri oleh aplikasi.'}
        </p>
      </div>
    )
  }
  const lr = f?.laba_rugi
  return (
    <div className="grid gap-3" style={{ marginBottom: 16 }}>
      <div className="ai-hero">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <span className="ai-badge"><Sparkles size={13} aria-hidden="true" /> Analisis AI</span>
          {a.model && <span style={{ fontSize: 11.5, opacity: 0.75 }}>dibaca oleh {a.model}</span>}
        </div>
        <div className="ai-judul">Analisis &amp; Rekomendasi AI</div>
        {a.ringkasan && <p className="ai-ringkas">{a.ringkasan}</p>}
        {lr && (
          <div className="ai-angka tabular">
            <div><span>Omzet</span><strong>{formatRp(lr.omzet)}</strong></div>
            <div><span>Laba (rugi) bersih</span><strong style={{ color: lr.laba_bersih >= 0 ? 'var(--ai-hijau)' : 'var(--ai-merah)' }}>{formatRp(lr.laba_bersih)}</strong></div>
            <div><span>Margin</span><strong>{persenBps(lr.margin_bps)}</strong></div>
            <div><span>Kas akhir</span><strong>{formatRp(f!.arus_kas.saldo_akhir)}</strong></div>
          </div>
        )}
      </div>

      {(!!a.sorotan_positif?.length || !!a.perlu_diperhatikan?.length) && (
        <div className="grid gap-3" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(290px, 1fr))' }}>
          {!!a.sorotan_positif?.length && (
            <div className="ai-poin ai-poin-baik">
              <div className="ai-poin-judul"><TrendingUp size={16} aria-hidden="true" /> Sorotan positif</div>
              {a.sorotan_positif.map((x, i) => <div key={i} className="ai-poin-isi"><CheckCircle2 size={15} aria-hidden="true" />{x}</div>)}
            </div>
          )}
          {!!a.perlu_diperhatikan?.length && (
            <div className="ai-poin ai-poin-awas">
              <div className="ai-poin-judul"><AlertTriangle size={16} aria-hidden="true" /> Perlu diperhatikan</div>
              {a.perlu_diperhatikan.map((x, i) => <div key={i} className="ai-poin-isi"><AlertTriangle size={15} aria-hidden="true" />{x}</div>)}
            </div>
          )}
        </div>
      )}

      {a.rekomendasi && a.rekomendasi.length > 0 && (
        <div className="card" style={{ padding: 20 }}>
          <div className="flex items-center gap-2 chart-title"><Lightbulb size={17} aria-hidden="true" style={{ color: 'var(--accent-orange)' }} /> Rekomendasi aksi</div>
          <div className="grid gap-2.5" style={{ marginTop: 12 }}>
            {a.rekomendasi.map((k, i) => {
              const d = DAMPAK[k.dampak] ?? DAMPAK.sedang
              return (
                <div key={i} className="ai-rek" style={{ borderLeftColor: d.warna }}>
                  <span className="ai-rek-no" style={{ background: d.warna }}>{i + 1}</span>
                  <div style={{ minWidth: 0 }}>
                    <div className="flex flex-wrap items-center gap-2">
                      <strong style={{ fontSize: 14 }}>{k.judul}</strong>
                      <span className="tab-badge" style={{ color: d.warna }}>{d.label}</span>
                    </div>
                    <p style={{ margin: '4px 0 0', fontSize: 13, color: 'var(--text-muted)', lineHeight: 1.6 }}>{k.detail}</p>
                  </div>
                </div>
              )
            })}
          </div>
        </div>
      )}
    </div>
  )
}
