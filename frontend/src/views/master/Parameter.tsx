import { useCallback, useEffect, useState } from 'react'
import { Copy, Download, Plus, Trash2, Upload } from 'lucide-react'
import { PageHeader } from '@/components/PageHeader'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  aktifkanParameter, getParameter, hapusParameter, imporNorma, salinParameter, ubahParameter, unduhNormaUrl,
  type ParameterIsi, type ParameterVersi,
} from '@/lib/api'
import { angka } from '@/lib/angka'
import { formatRp, formatTanggal } from '@/lib/format'
import { persen } from '@/lib/kalkulator'
import { ImporDialog, Pesan, Status } from './Katalog'

// Parameter pajak: angka aturan yang dipakai semua kalkulator klien. Versi aktif tidak bisa
// diubah langsung — buat draft dari versi mana pun, ubah, lalu aktifkan (versi lama jadi arsip
// dan bisa diaktifkan lagi).

const grid2 = { display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(min(320px, 100%), 1fr))', gap: 14 } as const

function Bagian({ judul, ket, children }: { judul: string; ket?: string; children: React.ReactNode }) {
  return (
    <div className="card" style={{ padding: 20, justifyContent: 'flex-start', minWidth: 0 }}>
      <div className="chart-title">{judul}</div>
      {ket && <div className="chart-subtitle" style={{ marginBottom: 10 }}>{ket}</div>}
      {children}
    </div>
  )
}

function Baris({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label style={{ display: 'grid', gridTemplateColumns: 'minmax(0, 1fr) minmax(0, 170px)', gap: 10, alignItems: 'center', marginBottom: 8 }}>
      <span style={{ fontSize: 13 }}>{label}</span>
      {children}
    </label>
  )
}

function Rp({ v, set, off, label }: { v: number; set: (n: number) => void; off: boolean; label?: string }) {
  return <Input className="tabular text-right" aria-label={label} disabled={off} {...angka(v, set)} />
}

/** Basis poin sebagai persen; teks diketik disimpan lokal supaya "12," tidak langsung hilang. */
const teksPct = (bps: number) => String(bps / 100).replace('.', ',')
const bacaPct = (t: string) => Math.round(parseFloat(t.replace(',', '.')) * 100)

function Pct({ bps, set, off, label }: { bps: number; set: (n: number) => void; off: boolean; label?: string }) {
  const [teks, setTeks] = useState(teksPct(bps))
  const [lalu, setLalu] = useState(bps)
  if (bps !== lalu) { // berubah dari luar (baris dihapus, versi dimuat ulang)
    setLalu(bps)
    if (bacaPct(teks) !== bps) setTeks(teksPct(bps))
  }
  return (
    <div className="flex items-center gap-1.5">
      <Input className="tabular text-right" inputMode="decimal" aria-label={label} disabled={off} value={teks}
        onChange={(e) => {
          const t = e.target.value.replace(/[^\d.,]/g, '')
          setTeks(t)
          const n = bacaPct(t)
          if (!Number.isNaN(n)) set(n)
        }} />
      <span className="text-sm text-muted-foreground">%</span>
    </div>
  )
}

export function ParameterView() {
  const [versi, setVersi] = useState<ParameterVersi[]>([])
  const [ptkpCodes, setPtkpCodes] = useState<string[]>([])
  const [terikat, setTerikat] = useState<string[]>([])
  const [pilih, setPilih] = useState<number | null>(null)
  const [isi, setIsi] = useState<ParameterIsi | null>(null)
  const [nama, setNama] = useState('')
  const [dasar, setDasar] = useState('')
  const [impor, setImpor] = useState(false)
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [busy, setBusy] = useState(false)

  const muat = useCallback((fokus?: number) => getParameter().then((r) => {
    setVersi(r.versi)
    setPtkpCodes(r.ptkp_codes)
    setTerikat(r.objek_terikat)
    const v = r.versi.find((x) => x.id === fokus) ?? r.versi.find((x) => x.status === 'draft') ?? r.versi.find((x) => x.status === 'aktif')
    if (v) { setPilih(v.id); setIsi(v.isi); setNama(v.nama); setDasar(v.dasar) }
  }, (e: Error) => setError(e.message)), [])
  useEffect(() => { void muat() }, [muat])

  const v = versi.find((x) => x.id === pilih)
  const off = v?.status !== 'draft'
  const ubah = (f: (x: ParameterIsi) => void) => setIsi((lama) => { if (!lama) return lama; const c = structuredClone(lama); f(c); return c })

  function tampilkan(x: ParameterVersi) {
    setPilih(x.id); setIsi(x.isi); setNama(x.nama); setDasar(x.dasar); setError(''); setInfo('')
  }

  async function aksi(f: () => Promise<unknown>, pesan: string, fokus?: number) {
    setBusy(true); setError(''); setInfo('')
    try {
      const r = await f()
      setInfo(pesan)
      await muat(fokus ?? (r as ParameterVersi | undefined)?.id)
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <>
      <PageHeader title="Parameter Pajak" subtitle="Tarif, PTKP, batas omzet, TER, tarif potong-pungut, dan norma NPPN yang dipakai semua kalkulator & perencanaan pajak klien." />
      <Pesan error={error} info={info} />

      <div className="card" style={{ padding: 20 }}>
        <div className="chart-title">Versi</div>
        <div className="table-scroll">
          <table className="data-table" style={{ minWidth: 760 }}>
            <thead><tr><th>Nama</th><th>Status</th><th>Dasar aturan</th><th style={{ textAlign: 'right' }}>Norma</th><th>Oleh</th><th style={{ textAlign: 'right' }}>Aksi</th></tr></thead>
            <tbody>
              {[...versi].reverse().map((x) => (
                <tr key={x.id} onClick={() => tampilkan(x)} style={{ cursor: 'pointer', background: x.id === pilih ? 'var(--bg-card-hover)' : undefined }}>
                  <td className="font-semibold">{x.nama}</td>
                  <td><Status s={x.status} /></td>
                  <td className="text-sm text-muted-foreground" style={{ whiteSpace: 'normal' }}>{x.dasar || '—'}</td>
                  <td className="tabular" style={{ textAlign: 'right' }}>{x.jumlah_norma.toLocaleString('id-ID')} KLU</td>
                  <td className="text-sm">{x.oleh || 'bawaan'}{x.oleh && ` · ${formatTanggal(x.waktu)}`}</td>
                  <td style={{ textAlign: 'right', whiteSpace: 'nowrap' }} onClick={(e) => e.stopPropagation()}>
                    <a className="inline-flex" href={unduhNormaUrl(x.id)} aria-label={`Unduh norma ${x.nama}`}><Button variant="ghost" size="icon-sm" tabIndex={-1}><Download /></Button></a>
                    <Button variant="ghost" size="icon-sm" aria-label={`Buat draft dari ${x.nama}`} title="Buat draft dari versi ini" disabled={busy}
                      onClick={() => void aksi(() => salinParameter(x.id), 'Draft baru dibuat. Ubah angkanya lalu simpan.')}><Copy /></Button>
                    {x.status === 'draft' && (
                      <Button variant="ghost" size="icon-sm" aria-label={`Hapus ${x.nama}`} disabled={busy}
                        onClick={() => void aksi(() => hapusParameter(x.id), 'Draft dihapus.', -1)}><Trash2 /></Button>
                    )}
                    {x.status !== 'aktif' && (
                      <Button size="sm" disabled={busy} onClick={() => void aksi(() => aktifkanParameter(x.id), `${x.nama} sekarang dipakai semua kalkulator klien.`)}>Aktifkan</Button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {v && isi && (
        <form key={v.id} style={{ display: 'flex', flexDirection: 'column', gap: 14 }}
          onSubmit={(e) => { e.preventDefault(); void aksi(() => ubahParameter(v.id, { nama, dasar, isi }), 'Draft tersimpan.', v.id) }}>
          <div className="card" style={{ padding: 20 }}>
            <div className="flex flex-wrap items-center gap-3">
              <div className="chart-title">{v.nama}</div>
              <Status s={v.status} />
              {off && <span className="text-sm text-muted-foreground">Hanya baca. Klik ikon salin untuk membuat draft dari versi ini.</span>}
              {!off && <Button type="submit" className="ml-auto" disabled={busy}>Simpan draft</Button>}
            </div>
            {!off && (
              <div style={{ ...grid2, marginTop: 12 }}>
                <label className="grid gap-1.5"><span className="kpi-label">Nama versi</span><Input value={nama} onChange={(e) => setNama(e.target.value)} required /></label>
                <label className="grid gap-1.5"><span className="kpi-label">Dasar aturan</span><Input value={dasar} placeholder="mis. PMK 81/2024" onChange={(e) => setDasar(e.target.value)} /></label>
              </div>
            )}
            {(isi.perlu_verifikasi ?? []).length > 0 && (
              <div style={{ marginTop: 12 }}>
                <div className="kpi-label" style={{ color: 'var(--text-warn)' }}>Perlu verifikasi tim pajak</div>
                <textarea className="input-field" rows={3} disabled={off} style={{ marginTop: 6, width: '100%', fontSize: 12.5 }}
                  value={(isi.perlu_verifikasi ?? []).join('\n')}
                  onChange={(e) => ubah((x) => { x.perlu_verifikasi = e.target.value.split('\n') })} />
              </div>
            )}
          </div>

          <div style={grid2}>
            <Bagian judul="PTKP setahun" ket="Penghasilan tidak kena pajak per status">
              {ptkpCodes.map((c) => (
                <Baris key={c} label={c}><Rp label={`PTKP ${c}`} v={isi.ptkp[c] ?? 0} off={off} set={(n) => ubah((x) => { x.ptkp[c] = n })} /></Baris>
              ))}
            </Bagian>

            <Bagian judul="Tarif Pasal 17 (orang pribadi)" ket="Lapisan progresif atas penghasilan kena pajak">
              {isi.pasal17.map((l, i) => {
                const akhir = i === isi.pasal17.length - 1
                return (
                  <div key={i} style={{ display: 'grid', gridTemplateColumns: 'minmax(0, 1fr) minmax(0, 110px) auto', gap: 8, alignItems: 'center', marginBottom: 8 }}>
                    {akhir ? <span className="text-sm text-muted-foreground">di atas {formatRp(isi.pasal17[i - 1]?.sampai ?? 0)}</span>
                      : <Rp label={`Batas lapisan ${i + 1}`} v={l.sampai} off={off} set={(n) => ubah((x) => { x.pasal17[i].sampai = n })} />}
                    <Pct label={`Tarif lapisan ${i + 1}`} bps={l.tarif_bps} off={off} set={(n) => ubah((x) => { x.pasal17[i].tarif_bps = n })} />
                    {!off && !akhir ? <Button type="button" variant="ghost" size="icon-sm" aria-label={`Hapus lapisan ${i + 1}`} onClick={() => ubah((x) => { x.pasal17.splice(i, 1) })}><Trash2 /></Button> : <span style={{ width: 28 }} />}
                  </div>
                )
              })}
              {!off && <Button type="button" variant="outline" size="sm" onClick={() => ubah((x) => { const n = x.pasal17.length; x.pasal17.splice(n - 1, 0, { sampai: (x.pasal17[n - 2]?.sampai ?? 0) + 1_000_000, tarif_bps: x.pasal17[n - 1].tarif_bps }) })}><Plus /> Tambah lapisan</Button>}
            </Bagian>

            <Bagian judul="PPh 21">
              <Baris label="Biaya jabatan"><Pct bps={isi.biaya_jabatan_bps} off={off} set={(n) => ubah((x) => { x.biaya_jabatan_bps = n })} /></Baris>
              <Baris label="Biaya jabatan maksimal setahun"><Rp v={isi.biaya_jabatan_maks} off={off} set={(n) => ubah((x) => { x.biaya_jabatan_maks = n })} /></Baris>
              <Baris label="Tanpa NPWP (pengali tarif)"><Pct bps={isi.tanpa_npwp_pph21_bps} off={off} set={(n) => ubah((x) => { x.tanpa_npwp_pph21_bps = n })} /></Baris>
              <Baris label="DPP bukan pegawai (dari bruto)"><Pct bps={isi.dasar_bukan_pegawai_bps} off={off} set={(n) => ubah((x) => { x.dasar_bukan_pegawai_bps = n })} /></Baris>
            </Bagian>

            <Bagian judul="PPh final UMKM">
              <Baris label="Tarif"><Pct bps={isi.umkm_tarif_bps} off={off} set={(n) => ubah((x) => { x.umkm_tarif_bps = n })} /></Baris>
              <Baris label="Omzet bebas pajak WP OP setahun"><Rp v={isi.umkm_bebas_op} off={off} set={(n) => ubah((x) => { x.umkm_bebas_op = n })} /></Baris>
              <Baris label="Batas omzet setahun"><Rp v={isi.batas_omzet_umkm} off={off} set={(n) => ubah((x) => { x.batas_omzet_umkm = n })} /></Baris>
              <Baris label="Rujukan aturan (tampil di klien)"><Input disabled={off} value={isi.dasar_umkm} onChange={(e) => ubah((x) => { x.dasar_umkm = e.target.value })} /></Baris>
            </Bagian>

            <Bagian judul="PPh 22 marketplace">
              <Baris label="Tarif atas omzet marketplace"><Pct bps={isi.marketplace_tarif_bps} off={off} set={(n) => ubah((x) => { x.marketplace_tarif_bps = n })} /></Baris>
              <Baris label="OP tidak dipungut bila omzet setahun s.d."><Rp v={isi.marketplace_bebas_op} off={off} set={(n) => ubah((x) => { x.marketplace_bebas_op = n })} /></Baris>
              <Baris label="Rujukan aturan"><Input disabled={off} value={isi.dasar_marketplace} onChange={(e) => ubah((x) => { x.dasar_marketplace = e.target.value })} /></Baris>
            </Bagian>

            <Bagian judul="PPh badan & Pasal 31E">
              <Baris label="Tarif badan"><Pct bps={isi.badan_tarif_bps} off={off} set={(n) => ubah((x) => { x.badan_tarif_bps = n })} /></Baris>
              <Baris label="Pengurangan tarif Pasal 31E"><Pct bps={isi.fasilitas_31e_bps} off={off} set={(n) => ubah((x) => { x.fasilitas_31e_bps = n })} /></Baris>
              <Baris label="Omzet yang mendapat fasilitas"><Rp v={isi.batas_31e_penuh} off={off} set={(n) => ubah((x) => { x.batas_31e_penuh = n })} /></Baris>
              <Baris label="Omzet maksimal Pasal 31E"><Rp v={isi.batas_31e} off={off} set={(n) => ubah((x) => { x.batas_31e = n })} /></Baris>
              <Baris label="PPh dividen"><Pct bps={isi.dividen_bps} off={off} set={(n) => ubah((x) => { x.dividen_bps = n })} /></Baris>
            </Bagian>

            <Bagian judul="PPN" ket={`Efektif ${persen(Math.floor(isi.ppn_tarif_bps * isi.ppn_dpp_pembilang / (isi.ppn_dpp_penyebut || 1)))} dari harga jual`}>
              <Baris label="Tarif"><Pct bps={isi.ppn_tarif_bps} off={off} set={(n) => ubah((x) => { x.ppn_tarif_bps = n })} /></Baris>
              <Baris label="DPP nilai lain (pembilang / penyebut)">
                <div className="flex items-center gap-1.5">
                  <Input className="tabular text-right" aria-label="Pembilang DPP" disabled={off} {...angka(isi.ppn_dpp_pembilang, (n) => ubah((x) => { x.ppn_dpp_pembilang = n }))} />
                  <span>/</span>
                  <Input className="tabular text-right" aria-label="Penyebut DPP" disabled={off} {...angka(isi.ppn_dpp_penyebut, (n) => ubah((x) => { x.ppn_dpp_penyebut = n }))} />
                </div>
              </Baris>
              <Baris label="Batas omzet wajib PKP"><Rp v={isi.batas_pkp} off={off} set={(n) => ubah((x) => { x.batas_pkp = n })} /></Baris>
            </Bagian>

            <Bagian judul="Norma (NPPN)" ket={`${v.jumlah_norma.toLocaleString('id-ID')} KLU 5 digit; diubah lewat Excel`}>
              <Baris label="Omzet di bawah ini boleh memakai norma"><Rp v={isi.batas_nppn} off={off} set={(n) => ubah((x) => { x.batas_nppn = n })} /></Baris>
              <Baris label="Norma untuk perbandingan bentuk usaha"><Pct bps={isi.norma_perbandingan_bps} off={off} set={(n) => ubah((x) => { x.norma_perbandingan_bps = n })} /></Baris>
              <div className="flex flex-wrap gap-2" style={{ marginTop: 6 }}>
                <a href={unduhNormaUrl(v.id)}><Button type="button" variant="outline" size="sm" tabIndex={-1}><Download /> Unduh tabel norma</Button></a>
                {!off && <Button type="button" variant="outline" size="sm" onClick={() => setImpor(true)}><Upload /> Ganti dari Excel</Button>}
              </div>
            </Bagian>
          </div>

          <Bagian judul="TER bulanan PPh 21 (PMK 168/2023)" ket="Kategori yang belum sampai lapisan tanpa batas dianggap belum lengkap; kalkulator memakai metode tahunan dan menyebut TER belum tersedia.">
            <div style={grid2}>
              {Object.keys(isi.ter.lapisan).sort().map((k) => {
                const rows = isi.ter.lapisan[k]
                return (
                  <div key={k} style={{ minWidth: 0 }}>
                    <div className="kpi-label" style={{ marginBottom: 6 }}>
                      Kategori {k} · {Object.entries(isi.ter.kategori).filter(([, kat]) => kat === k).map(([s]) => s).join(', ')}
                    </div>
                    {rows.map((b, i) => (
                      <div key={i} style={{ display: 'grid', gridTemplateColumns: 'minmax(0, 1fr) minmax(0, 1fr) minmax(0, 90px)', gap: 6, marginBottom: 6, alignItems: 'center' }}>
                        <Rp label={`TER ${k} batas bawah ${i + 1}`} v={b.batas_bawah} off={off} set={(n) => ubah((x) => { x.ter.lapisan[k][i].batas_bawah = n })} />
                        {b.batas_atas < 0 ? <span className="text-sm text-muted-foreground">tanpa batas</span>
                          : <Rp label={`TER ${k} batas atas ${i + 1}`} v={b.batas_atas} off={off} set={(n) => ubah((x) => { x.ter.lapisan[k][i].batas_atas = n })} />}
                        <Pct label={`TER ${k} tarif ${i + 1}`} bps={b.tarif_bps} off={off} set={(n) => ubah((x) => { x.ter.lapisan[k][i].tarif_bps = n })} />
                      </div>
                    ))}
                    {!off && (
                      <div className="flex flex-wrap gap-2">
                        <Button type="button" variant="outline" size="sm" disabled={rows.at(-1)?.batas_atas === -1}
                          onClick={() => ubah((x) => { const r = x.ter.lapisan[k]; const a = r.at(-1)?.batas_atas ?? -1; r.push({ batas_bawah: a + 1, batas_atas: a + 1, tarif_bps: r.at(-1)?.tarif_bps ?? 0 }) })}><Plus /> Lapisan</Button>
                        {rows.length > 0 && (
                          <Button type="button" variant="ghost" size="sm" onClick={() => ubah((x) => { const r = x.ter.lapisan[k]; r[r.length - 1].batas_atas = r[r.length - 1].batas_atas < 0 ? r[r.length - 1].batas_bawah : -1 })}>
                            {rows.at(-1)!.batas_atas < 0 ? 'Beri batas atas' : 'Jadikan tanpa batas'}
                          </Button>
                        )}
                        {rows.length > 1 && <Button type="button" variant="ghost" size="sm" onClick={() => ubah((x) => { x.ter.lapisan[k].pop() })}><Trash2 /> Terakhir</Button>}
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          </Bagian>

          <Bagian judul="Tarif potong-pungut per objek" ket="PPh 23, 4 ayat (2), 22, 15, PPN, PPnBM — dipakai kalkulator potong dan gross up.">
            {isi.objek.map((j, ji) => (
              <div key={j.id} style={{ marginBottom: 14 }}>
                <div className="flex flex-wrap items-center gap-3" style={{ marginBottom: 6 }}>
                  <span className="font-semibold">{j.nama}</span>
                  {j.tanpa_npwp_bps ? (
                    <span className="flex items-center gap-1.5 text-sm text-muted-foreground">tanpa NPWP
                      <span style={{ width: 110 }}><Pct label={`${j.nama} tanpa NPWP`} bps={j.tanpa_npwp_bps} off={off} set={(n) => ubah((x) => { x.objek[ji].tanpa_npwp_bps = n })} /></span>
                    </span>
                  ) : null}
                </div>
                <div className="table-scroll">
                  <table className="data-table" style={{ minWidth: 620 }}>
                    <thead><tr><th>Objek</th><th>Dasar aturan</th><th style={{ width: 130 }}>Tarif</th><th style={{ width: 130 }}>DPP dari bruto</th></tr></thead>
                    <tbody>
                      {j.objek.map((o, oi) => (
                        <tr key={o.kode}>
                          <td style={{ whiteSpace: 'normal' }}>
                            {o.nama}{o.final && <span className="pill-status pill-status-muted" style={{ marginLeft: 6 }}>final</span>}
                            {terikat.includes(o.kode) && <span className="block text-xs text-muted-foreground">Ikut tarif {o.kode.startsWith('ppn') ? 'PPN' : 'PPh final UMKM'} di atas</span>}
                          </td>
                          <td style={{ whiteSpace: 'normal' }}><Input disabled={off} aria-label={`Dasar ${o.nama}`} value={o.dasar} onChange={(e) => ubah((x) => { x.objek[ji].objek[oi].dasar = e.target.value })} /></td>
                          <td><Pct label={`Tarif ${o.nama}`} bps={o.tarif_bps} off={off || terikat.includes(o.kode)} set={(n) => ubah((x) => { x.objek[ji].objek[oi].tarif_bps = n })} /></td>
                          <td><Pct label={`DPP ${o.nama}`} bps={o.dasar_bps} off={off || terikat.includes(o.kode)} set={(n) => ubah((x) => { x.objek[ji].objek[oi].dasar_bps = n })} /></td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            ))}
          </Bagian>
          {!off && <Button type="submit" style={{ alignSelf: 'flex-start' }} disabled={busy}>Simpan draft</Button>}
        </form>
      )}

      {v && (
        <ImporDialog open={impor} onClose={() => setImpor(false)} judul="Ganti tabel norma NPPN"
          keterangan="Kolom: KLU (5 angka), Uraian, 10 ibukota provinsi (%), Ibukota provinsi lainnya (%), Daerah lainnya (%). Isi '-' atau kosongkan bila tidak tersedia. Seluruh tabel norma draft ini diganti."
          fields={[]}
          onKirim={async (file) => { await imporNorma(v.id, file); setImpor(false); setInfo('Tabel norma draft diganti.'); await muat(v.id) }} />
      )}
    </>
  )
}
