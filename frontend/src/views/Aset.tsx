import { useState } from 'react'
import { PageHeader } from '@/components/PageHeader'
import { createAsset, deleteAsset, updateAsset, type Asset, type AssetInput, type AssetSummary, type MetodePenyusutan } from '@/lib/api'
import { formatJt, formatRp } from '@/lib/format'

const AMBER = '#facc15'
const RED = '#fb7185'
const BLUE = '#38bdf8'

/** Tarif tahunan dari basis poin: 625 → "6,25%", 2500 → "25%". */
const formatTarif = (bps: number) => (bps / 100).toString().replace('.', ',') + '%'

/** Masa manfaat dalam bulan → "48 bln (4 th)". */
const formatMasaManfaat = (bulan: number) => `${bulan} bln (${bulan / 12} th)`

const KOSONG: AssetInput = {
  nama: '', jenis_harta: '', kelompok: 'k1', metode: 'garis_lurus',
  tanggal_perolehan: '', harga_perolehan: 0, tak_berwujud: false,
}

function toInput(a: Asset): AssetInput {
  return {
    nama: a.nama, jenis_harta: a.jenis_harta, kelompok: a.kelompok || 'k1',
    metode: a.metode || 'garis_lurus', tanggal_perolehan: a.tanggal_perolehan,
    harga_perolehan: a.harga_perolehan, tak_berwujud: a.tak_berwujud, transaksi_id: a.transaksi_id,
  }
}

export function Aset({ data, periodLabel, onChanged }: { data: AssetSummary; periodLabel: string; onChanged: () => void }) {
  const [form, setForm] = useState<AssetInput | null>(null)
  const [editId, setEditId] = useState<number | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const assets = data.assets ?? []
  // Jenis harta menentukan kelompoknya; "Lainnya" membiarkan user memilih kelompok sendiri.
  const jenisTerpilih = data.jenis_harta.find((j) => j.id === form?.jenis_harta)
  const kelompokAktif = data.kelompok.find((k) => k.id === (jenisTerpilih?.kelompok ?? form?.kelompok))
  const bolehSaldoMenurun = (kelompokAktif?.tarif_sm_bps ?? 0) > 0

  function buka(a?: Asset) {
    setError('')
    setEditId(a && !a.draft ? a.id : null)
    setForm(a ? toInput(a) : { ...KOSONG })
  }

  async function simpan(e: React.FormEvent) {
    e.preventDefault()
    if (!form) return
    setBusy(true)
    try {
      // Kelompok bangunan tidak boleh saldo menurun; samakan sebelum dikirim.
      const body = { ...form, metode: bolehSaldoMenurun ? form.metode : ('garis_lurus' as MetodePenyusutan) }
      await (editId === null ? createAsset(body) : updateAsset(editId, body))
      setForm(null)
      setEditId(null)
      onChanged()
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  async function hapus(a: Asset) {
    if (!confirm(`Hapus aset "${a.nama}" dari register? Penyusutannya akan hilang dari laporan.`)) return
    try {
      await deleteAsset(a.id)
      onChanged()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  const terpilih = assets.find((a) => a.id === editId)

  return (
    <>
      <PageHeader
        title="Aset & Penyusutan"
        subtitle={`${periodLabel} · penyusutan fiskal PMK 72/2023, tanpa nilai residu`}
      >
        <button className="btn-cta" style={{ padding: '8px 18px', fontSize: 13 }} onClick={() => buka()}>
          + Tambah Aset
        </button>
      </PageHeader>

      {error && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start', marginBottom: 12 }}>{error}</div>}

      <div className="note-box law" style={{ marginTop: 0, marginBottom: 14 }}>
        Kelompok dan tarif mengikuti <strong>Pasal 2 ayat (3)</strong>, amortisasi harta tak berwujud{' '}
        <strong>Pasal 9 ayat (3)</strong>. Penyusutan dimulai <strong>bulan dilakukannya pengeluaran</strong>{' '}
        (Pasal 5 ayat 1); pada saldo menurun, di akhir masa manfaat sisa nilai buku disusutkan sekaligus
        (Pasal 2 ayat 2). Bangunan hanya boleh garis lurus.
      </div>

      {data.pemetaan_belum_dicek > 0 && (
        <div className="note-box warn" style={{ marginTop: 0, marginBottom: 14 }}>
          <strong>{data.pemetaan_belum_dicek} jenis harta belum diverifikasi</strong> ke Lampiran PMK 72/2023.
          Tarif dan masa manfaat per kelompok sudah sesuai pasal, yang belum dicek adalah pemetaan nama harta
          ke kelompoknya. Periksa dulu bersama tim pajak sebelum angkanya dipakai untuk SPT; kalau tidak yakin,
          pilih <em>Lainnya</em> dan tentukan kelompoknya sendiri.
        </div>
      )}

      <div className="kpi-grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(185px, 1fr))' }}>
        <div className="card">
          <div className="kpi-label">Harga Perolehan</div>
          <div className="kpi-value tabular">{formatJt(data.total_harga_perolehan)}</div>
          <div className="kpi-meta-row"><span>{assets.length} aset tercatat</span></div>
        </div>
        <div className="card">
          <div className="kpi-label">Akumulasi Penyusutan</div>
          <div className="kpi-value tabular" style={{ color: RED }}>{formatJt(data.total_akumulasi)}</div>
          <div className="kpi-meta-row"><span>Sejak perolehan s/d akhir periode</span></div>
        </div>
        <div className="card">
          <div className="kpi-label">Nilai Buku</div>
          <div className="kpi-value tabular" style={{ color: BLUE }}>{formatJt(data.total_nilai_buku)}</div>
          <div className="kpi-meta-row"><span>Masuk Neraca sebagai aset</span></div>
        </div>
        <div className="card">
          <div className="kpi-label">Beban Bulan Ini</div>
          <div className="kpi-value tabular" style={{ color: AMBER }}>{formatJt(data.beban_total)}</div>
          <div className="kpi-meta-row">
            <span>Penyusutan {formatJt(data.beban_penyusutan)} + amortisasi {formatJt(data.beban_amortisasi)}</span>
          </div>
        </div>
      </div>

      {data.draft_count > 0 && (
        <div className="note-box warn" style={{ marginTop: 0, marginBottom: 14 }}>
          <strong>{data.draft_count} aset perlu dilengkapi.</strong> Transaksi "Beli Alat / Aset" sudah tercatat
          dan nilainya masuk Neraca, tapi belum menghasilkan penyusutan sampai kelompok &amp; metodenya diisi.
          Klik <em>Lengkapi</em> di barisnya.
        </div>
      )}

      {form && (
        <form className="card" style={{ padding: 20, marginBottom: 14 }} onSubmit={simpan}>
          <div className="chart-title">{editId === null ? 'Tambah Aset' : 'Ubah Aset'}</div>
          <div className="chart-subtitle">Pilih jenis harta, kelompok &amp; tarif terisi otomatis dari PMK</div>

          <div className="form-grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(190px, 1fr))' }}>
            <label style={{ gridColumn: '1 / -1' }}>
              <span className="kpi-label">Nama Aset</span>
              <input className="input-field" required value={form.nama} onChange={(e) => setForm({ ...form, nama: e.target.value })} />
            </label>

            <label style={{ gridColumn: '1 / -1' }}>
              <span className="kpi-label">Jenis Harta</span>
              <select
                className="input-field"
                value={form.jenis_harta}
                onChange={(e) => {
                  const j = data.jenis_harta.find((x) => x.id === e.target.value)
                  setForm({ ...form, jenis_harta: e.target.value, kelompok: j?.kelompok ?? form.kelompok, tak_berwujud: j?.tak_berwujud ?? false })
                }}
              >
                <option value="">Lainnya — pilih kelompok sendiri</option>
                {data.jenis_harta.map((j) => (
                  <option key={j.id} value={j.id}>
                    {j.nama}{j.terverifikasi ? '' : ' — pemetaan perlu dicek'}
                  </option>
                ))}
              </select>
            </label>

            {form.jenis_harta === '' && (
              <label>
                <span className="kpi-label">Kelompok PMK</span>
                <select className="input-field" value={form.kelompok} onChange={(e) => setForm({ ...form, kelompok: e.target.value })}>
                  {data.kelompok.map((k) => (
                    <option key={k.id} value={k.id}>{k.nama}</option>
                  ))}
                </select>
              </label>
            )}

            <label>
              <span className="kpi-label">Tanggal Perolehan</span>
              <input
                className="input-field" type="date" required value={form.tanggal_perolehan}
                onChange={(e) => setForm({ ...form, tanggal_perolehan: e.target.value })}
              />
            </label>

            <label>
              <span className="kpi-label">Harga Perolehan</span>
              <input
                className="input-field tabular" type="number" min={1} required value={form.harga_perolehan || ''}
                onChange={(e) => setForm({ ...form, harga_perolehan: Number(e.target.value) })}
              />
            </label>

            <label>
              <span className="kpi-label">Metode Penyusutan</span>
              <select
                className="input-field" value={form.metode} disabled={!bolehSaldoMenurun}
                onChange={(e) => setForm({ ...form, metode: e.target.value as MetodePenyusutan })}
              >
                <option value="garis_lurus">
                  Garis Lurus{kelompokAktif && ` — ${formatTarif(kelompokAktif.tarif_gl_bps)}`}
                </option>
                {bolehSaldoMenurun && (
                  <option value="saldo_menurun">Saldo Menurun — {formatTarif(kelompokAktif!.tarif_sm_bps)}</option>
                )}
              </select>
            </label>

            <div>
              <span className="kpi-label">Kelompok &amp; masa manfaat (otomatis)</span>
              <div className="input-field tabular" style={{ color: BLUE, borderStyle: 'dashed' }}>
                {kelompokAktif ? `${kelompokAktif.nama} · ${formatMasaManfaat(kelompokAktif.masa_manfaat)}` : '—'}
              </div>
            </div>
          </div>

          {!bolehSaldoMenurun && kelompokAktif && (
            <div className="note-box warn">Bangunan hanya boleh garis lurus, jadi metode dikunci.</div>
          )}
          {form.tak_berwujud && (
            <div className="note-box">Harta tak berwujud: bebannya dicatat sebagai <strong>amortisasi</strong>, bukan penyusutan.</div>
          )}

          <div style={{ display: 'flex', gap: 10, marginTop: 16 }}>
            <button className="btn-cta" style={{ padding: '9px 20px', fontSize: 13 }} type="submit" disabled={busy}>
              {busy ? 'Menyimpan…' : 'Simpan'}
            </button>
            <button type="button" className="nav-item" style={{ width: 'auto', padding: '9px 16px' }} onClick={() => setForm(null)}>
              Batal
            </button>
          </div>
        </form>
      )}

      <div className="card" style={{ padding: 20 }}>
        <div className="chart-title">Register Aset</div>
        <div className="chart-subtitle">Metode dipilih per aset: garis lurus atau saldo menurun</div>
        <div className="table-scroll">
          <table className="data-table tabular" style={{ minWidth: 980 }}>
            <thead>
              <tr>
                <th>Aset</th>
                <th>Kelompok PMK</th>
                <th>Perolehan</th>
                <th style={{ textAlign: 'right' }}>Harga Perolehan</th>
                <th>Metode</th>
                <th style={{ textAlign: 'right' }}>Tarif</th>
                <th style={{ textAlign: 'right' }}>Beban / Bln</th>
                <th style={{ textAlign: 'right' }}>Akumulasi</th>
                <th style={{ textAlign: 'right' }}>Nilai Buku</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {assets.length === 0 && (
                <tr><td colSpan={10} style={{ color: 'var(--text-muted)' }}>Belum ada aset pada periode ini.</td></tr>
              )}
              {assets.map((a) => (
                <tr key={a.id || `draft-${a.transaksi_id}`}>
                  <td>
                    <div style={{ fontWeight: 600 }}>{a.nama}</div>
                    {a.draft && <span className="pill-status pill-amber" style={{ fontSize: 10 }}>Perlu dilengkapi</span>}
                  </td>
                  <td>{a.draft ? '—' : `${a.kelompok_nama} · ${formatMasaManfaat(a.masa_manfaat)}`}</td>
                  <td>{a.tanggal_perolehan}</td>
                  <td style={{ textAlign: 'right' }}>{formatRp(a.harga_perolehan)}</td>
                  <td>
                    {a.draft ? '—' : a.metode === 'saldo_menurun' ? 'Saldo Menurun' : 'Garis Lurus'}
                    {a.tak_berwujud && <div style={{ fontSize: 10, color: 'var(--text-muted)' }}>amortisasi</div>}
                  </td>
                  <td style={{ textAlign: 'right' }}>{a.draft ? '—' : formatTarif(a.tarif_bps)}</td>
                  <td style={{ textAlign: 'right', color: AMBER }}>{formatRp(a.beban_bulan_ini)}</td>
                  <td style={{ textAlign: 'right', color: RED }}>{formatRp(a.akumulasi)}</td>
                  <td style={{ textAlign: 'right', color: BLUE }}>{formatRp(a.nilai_buku)}</td>
                  <td style={{ whiteSpace: 'nowrap' }}>
                    <button className="nav-item" style={{ width: 'auto', padding: '4px 8px', fontSize: 12 }} onClick={() => buka(a)}>
                      {a.draft ? 'Lengkapi' : 'Ubah'}
                    </button>
                    {!a.draft && (
                      <button className="nav-item" style={{ width: 'auto', padding: '4px 8px', fontSize: 12, color: RED }} onClick={() => hapus(a)}>
                        Hapus
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
            <tfoot>
              <tr style={{ fontWeight: 700 }}>
                <td colSpan={3}>TOTAL — {assets.length} aset</td>
                <td style={{ textAlign: 'right' }}>{formatRp(data.total_harga_perolehan)}</td>
                <td colSpan={2} />
                <td style={{ textAlign: 'right', color: AMBER }}>{formatRp(data.beban_total)}</td>
                <td style={{ textAlign: 'right', color: RED }}>{formatRp(data.total_akumulasi)}</td>
                <td style={{ textAlign: 'right', color: BLUE }}>{formatRp(data.total_nilai_buku)}</td>
                <td />
              </tr>
            </tfoot>
          </table>
        </div>
      </div>

      {terpilih?.jadwal && terpilih.jadwal.length > 0 && (
        <div className="card" style={{ padding: 20, marginTop: 14 }}>
          <div className="chart-title">Jadwal Penyusutan — {terpilih.nama}</div>
          <div className="chart-subtitle">
            {formatRp(terpilih.harga_perolehan)} · {terpilih.kelompok_nama} · tarif {formatTarif(terpilih.tarif_bps)}
          </div>
          <div className="table-scroll">
            <table className="data-table tabular">
              <thead>
                <tr>
                  <th>Tahun</th>
                  <th style={{ textAlign: 'right' }}>Dasar (Sisa Buku Awal)</th>
                  <th style={{ textAlign: 'right' }}>Penyusutan</th>
                  <th style={{ textAlign: 'right' }}>Sisa Buku Akhir</th>
                </tr>
              </thead>
              <tbody>
                {terpilih.jadwal.map((y) => (
                  <tr key={y.tahun}>
                    <td>{y.tahun}{y.bulan < 12 && <span style={{ color: 'var(--text-muted)' }}> ({y.bulan} bln)</span>}</td>
                    <td style={{ textAlign: 'right' }}>{formatRp(y.dasar_awal)}</td>
                    <td style={{ textAlign: 'right', color: AMBER }}>{formatRp(y.penyusutan)}</td>
                    <td style={{ textAlign: 'right', color: y.sisa_akhir === 0 ? '#34d399' : BLUE }}>{formatRp(y.sisa_akhir)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {terpilih.metode === 'saldo_menurun' && (
            <div className="note-box">
              Baris terakhir menyusutkan seluruh sisa nilai buku sekaligus, sesuai Pasal 2 ayat (2) — tanpa itu
              nilai buku tidak akan pernah mencapai nol.
            </div>
          )}
        </div>
      )}
    </>
  )
}
