import { useEffect, useMemo, useState } from 'react'
import { ArrowLeft, ArrowLeftRight, Camera, Plus, Sparkles, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  bacaNota, buktiCatatUrl, cariMirip, getCatatanJurnal, getOpsiCatat, getPilihanBarang, hapusCatatanJurnal, simpanCatatanJurnal, type Mirip,
  type AkunCatat, type BarisCatat, type FotoNota, type OpsiCatat, type PilihanBarang, type SaranNota,
} from '@/lib/api'
import { PilihRincianBarang } from '@/components/PilihRincianBarang'
import { barisBerbarang, PESAN_PECAH_CATAT, rincianDariNota } from '@/lib/catatBarang'
import { berbarang, bodyRincian, rincianDariTx, rincianSah, wajibBarang, type NilaiRincian } from '@/lib/rincian'
import { angka } from '@/lib/angka'
import { bacaDraft, hapusDraft, useSimpanDraft } from '@/lib/draft'
import { formatRp, formatTanggal } from '@/lib/format'
import { kecilkan } from '@/lib/foto'
import { JENIS_CATAT } from '@/lib/catat'

// Halaman Catat transaksi (Catat cepat): satu catatan = beberapa baris jurnal (akun, debit /
// kredit), bisa diisi dari foto nota di halaman yang sama. Tombol ⇄ menukar debit dan kredit per
// baris atau semuanya, mis. bila AI menaruh angka di sisi yang terbalik. Baris penjualan /
// pembelian barang membawa rincian barang (wajib di perpetual), juga yang terbaca dari nota.

type Baris = BarisCatat & { _k: number; ai?: boolean }
type Kepala = { tanggal: string; pihak: string; no_bukti: string; keterangan: string }
let urut = 0
const baru = (b: Partial<BarisCatat>, ai = false): Baris => ({ akun: '', debit: 0, kredit: 0, ...b, _k: ++urut, ai })
const hariIni = () => new Date().toLocaleDateString('sv-SE', { timeZone: 'Asia/Jakarta' })
const KEPALA_KOSONG = (): Kepala => ({ tanggal: hariIni(), pihak: '', no_bukti: '', keterangan: '' })
const RINCIAN_KOSONG = (): NilaiRincian => ({ rincian: [], bukan_barang: false })

/** Rekening bawaan untuk uang masuk / keluar: Bank (Kas bila usaha tanpa akun bank). */
export const rekeningBawaan = (o: OpsiCatat) => o.bawaan.bank || o.bawaan.kas

/** Baris awal: akun pokok jenis itu + rekening bawaan (Bank), di sisi yang benar. */
function barisAwal(jenis: string, o: OpsiCatat): Baris[] {
  const masuk = JENIS_CATAT.find((j) => j.id === jenis)?.masuk ?? false
  const pokok = o.bawaan[jenis] ?? ''
  // pindah dana: bawaan tarik tunai (Kas di debit, Bank di kredit); keduanya bisa diganti
  if (jenis === 'Pindah Dana') return [baru({ akun: o.bawaan.kas }), baru({ akun: o.bawaan.bank })]
  return masuk
    ? [baru({ akun: rekeningBawaan(o) }), baru({ akun: pokok })]
    : [baru({ akun: pokok }), baru({ akun: rekeningBawaan(o) })]
}

/** Hasil baca foto → baris jurnal: barang per baris bila jumlahnya pas dengan total.
 *  satuBaris: satu baris pokok saja (barangnya dicatat sebagai rincian satu transaksi). */
export function barisDariNota(n: SaranNota, o: OpsiCatat, satuBaris = false): Baris[] {
  const jenis = n.jenis || 'Biaya Lain'
  const masuk = ['Penjualan', 'Pendapatan Lain', 'Setor Modal', 'Terima Pinjaman', 'PPN Keluaran'].includes(jenis)
  const pokok = o.bawaan[jenis] ?? ''
  // Nota / struk yang sudah dibayar: uang bergerak di Bank (bukan Kas); boleh diganti.
  const lawan = n.lunas ? rekeningBawaan(o) : masuk ? o.bawaan.piutang : o.bawaan.utang
  const item = (n.item ?? []).filter((i) => (i.harga ?? 0) > 0)
  const rinci = !satuBaris && item.length > 1 && item.reduce((t, i) => t + (i.harga ?? 0), 0) === n.nominal
    ? item.map((i) => i.harga!)
    : [n.nominal]
  const sisiPokok = rinci.map((x) => baru(masuk ? { akun: pokok, kredit: x } : { akun: pokok, debit: x }, true))
  const sisiLawan = baru(masuk ? { akun: lawan, debit: n.nominal } : { akun: lawan, kredit: n.nominal }, true)
  return masuk ? [sisiLawan, ...sisiPokok] : [...sisiPokok, sisiLawan]
}

/** perpetual: barang wajib di baris penjualan / pembelian barang (sama dengan form Transaksi).
 *  bolehBarangBaru: pengguna boleh menambah barang lewat "+ Barang baru". */
export function CatatJurnal({ jenisAwal, grup, onKembali, onTersimpan, perpetual = false, bolehBarangBaru = false }: {
  jenisAwal: string; grup?: number; onKembali: () => void; onTersimpan: (pesan: string, tanggal?: string) => Promise<void> | void
  perpetual?: boolean; bolehBarangBaru?: boolean
}) {
  const draftKey = grup ? null : `catat:${jenisAwal}`
  const [opsi, setOpsi] = useState<OpsiCatat | null>(null)
  const [jenis, setJenis] = useState(jenisAwal)
  const [kepala, setKepala] = useState<Kepala>(KEPALA_KOSONG)
  const [baris, setBaris] = useState<Baris[]>([])
  const [nota, setNota] = useState<SaranNota | null>(null)
  const [fotoNota, setFotoNota] = useState<FotoNota | null>(null) // bukti catatan yang sedang diubah
  const [membaca, setMembaca] = useState(false)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  // Cegah dobel: transaksi yang sudah ada (mis. dari impor mutasi) dengan arah & nominal kas/bank
  // sama di tanggal berdekatan. Simpan baru jalan setelah pengguna menyatakan berbeda.
  const [mirip, setMirip] = useState<{ baris: string; m: Mirip }[] | null>(null)
  const [yakinBeda, setYakinBeda] = useState(false)
  const [dipulihkan, setDipulihkan] = useState(false)
  // Barang: rincian untuk satu-satunya baris penjualan / pembelian barang catatan ini.
  const [barang, setBarang] = useState<{ mulai: string; barang: PilihanBarang[] }>({ mulai: '', barang: [] })
  const [rincian, setRincianAsli] = useState<NilaiRincian>(RINCIAN_KOSONG)
  const [versiRincian, setVersiRincian] = useState(0) // naik saat diisi dari luar → pemilih barang dimuat ulang
  const [tidakCocok, setTidakCocok] = useState<string[]>([]) // barang di nota yang tidak ada di daftar
  const isiRincian = (v: NilaiRincian) => { setRincianAsli(v); setVersiRincian((n) => n + 1) }
  const muatBarang = () => getPilihanBarang().then((b) => { setBarang(b); return b }, () => barang)

  useEffect(() => {
    let hidup = true
    getOpsiCatat().then(async (o) => {
      if (!hidup) return
      setOpsi(o)
      if (grup) {
        const c = await getCatatanJurnal(grup)
        if (!hidup) return
        setKepala({ tanggal: c.tanggal, pihak: c.pihak ?? '', no_bukti: c.no_bukti ?? '', keterangan: c.keterangan ?? '' })
        setBaris(c.baris.map((b) => baru(b)))
        isiRincian(rincianDariTx(c))
        setFotoNota(c.foto_nota ?? null)
        return
      }
      const d = draftKey ? bacaDraft<{ kepala: Kepala; baris: BarisCatat[]; rincian?: NilaiRincian }>(draftKey) : null
      if (d?.baris?.length) {
        setKepala(d.kepala)
        setBaris(d.baris.map((b) => baru(b)))
        if (d.rincian) isiRincian(d.rincian)
        setDipulihkan(true)
      } else setBaris(barisAwal(jenisAwal, o))
    }, (e: Error) => setError(e.message))
    return () => { hidup = false }
  }, [grup, jenisAwal, draftKey])
  useEffect(() => { getPilihanBarang().then(setBarang, () => {}) }, [])

  useSimpanDraft(draftKey && baris.length ? draftKey : null, useMemo(() => ({ kepala, rincian, baris: baris.map(({ _k, ai, ...b }) => { void _k; void ai; return b }) }), [kepala, rincian, baris]))

  const akunPeta = useMemo(() => new Map((opsi?.akun ?? []).map((a) => [a.kode, a])), [opsi])
  const kelompok = useMemo(() => [...new Set((opsi?.akun ?? []).map((a) => a.kelompok))], [opsi])
  const totalD = baris.reduce((t, b) => t + (b.debit || 0), 0)
  const totalK = baris.reduce((t, b) => t + (b.kredit || 0), 0)
  const selisih = totalD - totalK
  // Baris penjualan / pembelian barang; pemilih barang tampil seperti di form Transaksi.
  const berbarangBaris = barisBerbarang(baris, akunPeta)
  const bolehBarang = berbarangBaris.length > 0 && (barang.barang.length > 0 || perpetual)
  const satuBarang = berbarangBaris.length === 1 ? berbarangBaris[0] : null
  const wajib = bolehBarang && wajibBarang({ perpetual, jenis: berbarangBaris[0].jenis, tanggal: kepala.tanggal, mulai: barang.mulai })
  const bisaSimpan = !busy && totalD > 0 && selisih === 0 && baris.every((b) => !(b.debit || b.kredit) || b.akun)

  const ubah = (k: number, p: Partial<Baris>) => setBaris((bs) => bs.map((b) => (b._k === k ? { ...b, ...p, ai: false } : b)))
  const balik = (b: Baris) => ({ ...b, debit: b.kredit, kredit: b.debit })

  function gantiJenis(j: string) {
    setJenis(j)
    // Baris yang masih kosong / bawaan diganti baris awal jenis baru; isian pengguna dipertahankan.
    if (opsi && baris.every((b) => !b.debit && !b.kredit)) setBaris(barisAwal(j, opsi))
  }

  async function bacaFoto(file: File | undefined) {
    if (!file || !opsi) return
    setError('')
    setMembaca(true)
    try {
      const [n, daftar] = await Promise.all([bacaNota(await kecilkan(file)), muatBarang()])
      setNota(n)
      if (n.jenis && JENIS_CATAT.some((j) => j.id === n.jenis)) setJenis(n.jenis)
      // satu keterangan per catatan: tanpa keterangan dari AI, pakai daftar barang di nota
      const isiBarang = (n.item ?? []).map((i) => i.nama + (i.qty ? ` × ${i.qty}` : '')).join(', ')
      setKepala((k) => ({ ...k, tanggal: n.tanggal || k.tanggal, pihak: n.pihak ?? k.pihak, keterangan: n.keterangan || isiBarang || k.keterangan }))
      // nota berbarang: satu baris pokok, barangnya jadi rincian (dicocokkan ke nama di daftar barang)
      const pakaiBarang = berbarang(n.jenis) && (daftar.barang.length > 0 || perpetual)
      setBaris(barisDariNota(n, opsi, pakaiBarang))
      const r = pakaiBarang ? rincianDariNota(n.item ?? [], daftar.barang, n.jenis) : { rincian: [], tidakCocok: [] }
      isiRincian({ rincian: r.rincian, bukan_barang: false })
      setTidakCocok(r.tidakCocok)
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setMembaca(false)
    }
  }

  async function cekMirip(): Promise<{ baris: string; m: Mirip }[]> {
    const hasil: { baris: string; m: Mirip }[] = []
    for (const b of baris) {
      const a = akunPeta.get(b.akun)
      const n = b.debit || b.kredit
      if (a?.lawan !== 'rekening' || !n) continue
      const m = await cariMirip(kepala.tanggal, n, b.debit ? 'masuk' : 'keluar', grup).catch(() => [])
      for (const x of m) hasil.push({ baris: `${a.kode} ${a.nama} ${formatRp(n)}`, m: x })
    }
    return hasil
  }

  async function simpan(lagi: boolean) {
    // barang: sama dengan server (wajib di perpetual; rincian hanya untuk satu transaksi berbarang)
    if (bolehBarang) {
      // beberapa baris berbarang: hanya Bukan barang (jasa) yang bisa dipasang ke semuanya
      const salah = satuBarang ? rincianSah(rincian, wajib, satuBarang) : wajib && !rincian.bukan_barang ? PESAN_PECAH_CATAT : ''
      if (salah) return setError(salah)
    }
    setBusy(true)
    setError('')
    try {
      if (!yakinBeda) {
        const m = await cekMirip()
        if (m.length) {
          setMirip(m)
          return
        }
      }
      // Satu keterangan per catatan (kepala); server memakainya untuk setiap baris.
      const isi = baris.filter((b) => b.debit || b.kredit).map(({ akun, debit, kredit, jatuh_tempo }) => ({ akun, keterangan: '', debit, kredit, jatuh_tempo }))
      // dari foto: tandai sumbernya (cek dobel); catatan baru juga menyimpan fotonya sebagai bukti
      const b = satuBarang ? bodyRincian(rincian, satuBarang.jenis) : { rincian: undefined, bukan_barang: bolehBarang && rincian.bukan_barang }
      await simpanCatatanJurnal({ ...kepala, baris: isi, rincian: b.rincian, bukan_barang: b.bukan_barang || undefined, ...(nota ? { sumber: 'foto' as const } : {}), ...(!grup && nota ? { bukti: nota.bukti } : {}) }, grup)
      setMirip(null)
      setYakinBeda(false)
      if (draftKey) hapusDraft(draftKey)
      await onTersimpan(grup ? 'Catatan diperbarui.' : 'Catatan tersimpan.', kepala.tanggal)
      if (lagi && opsi) {
        setKepala({ ...KEPALA_KOSONG(), tanggal: kepala.tanggal })
        setBaris(barisAwal(jenis, opsi))
        setNota(null)
        isiRincian(RINCIAN_KOSONG())
        setTidakCocok([])
        void muatBarang() // stok & titik nol kartu bisa berubah (mulai otomatis)
      } else onKembali()
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  async function hapus() {
    if (!grup) return
    setBusy(true)
    try {
      await hapusCatatanJurnal(grup)
      await onTersimpan('Catatan dihapus.')
      onKembali()
    } catch (err) {
      setError((err as Error).message)
      setBusy(false)
    }
  }

  const pilihAkun = (b: Baris) => (
    <select className="input-field" aria-label="Akun CoA" value={b.akun} onChange={(e) => ubah(b._k, { akun: e.target.value })}>
      {!b.akun && <option value="">Pilih akun…</option>}
      {b.akun && !akunPeta.has(b.akun) && <option value={b.akun}>{b.akun}</option>}
      {kelompok.map((g) => (
        <optgroup key={g} label={g}>
          {(opsi?.akun ?? []).filter((a) => a.kelompok === g).map((a: AkunCatat) => <option key={a.kode} value={a.kode}>{a.kode} {a.nama}</option>)}
        </optgroup>
      ))}
    </select>
  )

  return (
    <div className="grid gap-4">
      <button type="button" className="inline-flex items-center gap-1.5 text-sm text-muted-foreground" style={{ justifySelf: 'start' }} onClick={onKembali}>
        <ArrowLeft size={15} aria-hidden="true" /> Kembali ke daftar transaksi
      </button>

      <div className="catat-grid">
        <div className="grid gap-4" style={{ minWidth: 0 }}>
          <div className="card grid gap-3" style={{ padding: 18 }}>
            <div className="flex flex-wrap items-center justify-between gap-2">
              <h2 className="chart-title" style={{ fontSize: 17 }}>{grup ? 'Ubah catatan' : JENIS_CATAT.find((j) => j.id === jenis)?.label ?? 'Catat transaksi'}</h2>
              {dipulihkan && (
                <span className="text-xs" role="status" style={{ color: 'var(--text-warn)' }}>
                  Isian sebelumnya dipulihkan.{' '}
                  <button type="button" className="underline underline-offset-4" onClick={() => { if (draftKey) hapusDraft(draftKey); setDipulihkan(false); setKepala(KEPALA_KOSONG()); isiRincian(RINCIAN_KOSONG()); if (opsi) setBaris(barisAwal(jenis, opsi)) }}>Mulai kosong</button>
                </span>
              )}
            </div>
            {!grup && (
              <div className="flex flex-wrap gap-2" role="group" aria-label="Jenis catatan">
                {JENIS_CATAT.map((j) => (
                  <button key={j.id} type="button" className={'catat-jenis' + (jenis === j.id ? ' aktif' : '')} aria-pressed={jenis === j.id} onClick={() => gantiJenis(j.id)}>{j.label}</button>
                ))}
              </div>
            )}

            <div className="catat-foto">
              <Camera size={22} aria-hidden="true" style={{ color: 'var(--accent-orange)', flexShrink: 0 }} />
              <div style={{ flex: '1 1 240px', minWidth: 0 }}>
                {membaca ? <span className="text-sm" role="status">AI sedang membaca nota…</span>
                  : nota ? (
                    <span className="text-sm">
                      <strong>Diisi dari foto nota</strong> (keyakinan {nota.yakin}). Baris bertanda <span className="catat-ai">AI</span> perlu Anda periksa.
                      {nota.alasan && <> Rekomendasi AI: {nota.alasan}.</>}
                      {nota.catatan.length > 0 && <span style={{ color: 'var(--text-warn)' }}> Cek: {nota.catatan.join('; ')}.</span>}
                    </span>
                  ) : grup && fotoNota ? (
                    <span className="text-sm">
                      <strong>Foto nota tersimpan sebagai bukti.</strong> Dibaca AI, disimpan oleh {fotoNota.disimpan_oleh} pada {formatTanggal(fotoNota.disimpan_pada)}.{' '}
                      <a className="underline underline-offset-4" href={buktiCatatUrl(grup)} target="_blank" rel="noopener">Lihat foto nota</a>
                    </span>
                  ) : <span className="text-sm"><strong>Isi dari foto nota</strong>: foto nota, struk, atau bukti transfer, AI mengisi baris di bawah.</span>}
              </div>
              <label className="btn-cta accent" style={{ padding: '7px 14px', fontSize: 13, cursor: 'pointer' }}>
                {nota ? 'Ganti foto' : 'Pilih / foto nota'}
                <input type="file" accept="image/*" capture="environment" className="sr-only" disabled={membaca} onChange={(e) => { void bacaFoto(e.target.files?.[0]); e.target.value = '' }} />
              </label>
            </div>
            {nota?.duplikat?.length ? (
              <div className="note-box warn" role="status" style={{ marginTop: 0 }}>
                <strong>Mungkin sudah pernah dicatat.</strong> Periksa dulu supaya tidak tercatat dua kali; tetap bisa disimpan bila memang berbeda.
                <ul style={{ margin: '4px 0 0 16px', listStyle: 'disc' }}>
                  {nota.duplikat.map((d, i) => (
                    <li key={i}>{formatTanggal(d.tanggal)} · {formatRp(d.nominal)}{d.pihak ? ` · ${d.pihak}` : ''}: {d.alasan}</li>
                  ))}
                </ul>
              </div>
            ) : null}

            <div className="form-grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(170px, 1fr))' }}>
              <label><span className="kpi-label">Tanggal</span>
                <input className="input-field" type="date" required value={kepala.tanggal} onChange={(e) => setKepala({ ...kepala, tanggal: e.target.value })} /></label>
              <label><span className="kpi-label">Pelanggan / supplier</span>
                <input className="input-field" value={kepala.pihak} placeholder="Opsional" onChange={(e) => setKepala({ ...kepala, pihak: e.target.value })} /></label>
              <label><span className="kpi-label">No. nota / bukti</span>
                <input className="input-field" value={kepala.no_bukti} placeholder="Opsional" onChange={(e) => setKepala({ ...kepala, no_bukti: e.target.value })} /></label>
              <label><span className="kpi-label">Keterangan</span>
                <input className="input-field" value={kepala.keterangan} placeholder="mis. Belanja bahan minggu ini" onChange={(e) => setKepala({ ...kepala, keterangan: e.target.value })} /></label>
            </div>
          </div>

          <div className="card grid gap-3" style={{ padding: 18 }}>
            <div className="flex flex-wrap items-center justify-between gap-2">
              <h3 className="kpi-label" style={{ margin: 0 }}>Baris jurnal</h3>
              <div className="flex flex-wrap gap-2">
                <Button type="button" variant="outline" size="sm" onClick={() => setBaris((bs) => bs.map(balik))} title="Tukar debit dan kredit di semua baris">
                  <ArrowLeftRight aria-hidden="true" /> Balik semua debit/kredit
                </Button>
                <Button type="button" size="sm" onClick={() => setBaris((bs) => [...bs, baru({})])}><Plus aria-hidden="true" /> Baris</Button>
              </div>
            </div>
            <div className="table-scroll">
              <table className="data-table tabular" style={{ minWidth: 560 }}>
                <thead>
                  <tr><th>Akun CoA</th><th style={{ textAlign: 'right', width: 140 }}>Debit</th><th style={{ textAlign: 'right', width: 140 }}>Kredit</th><th style={{ width: 44 }} /><th style={{ width: 70 }} /></tr>
                </thead>
                <tbody>
                  {baris.map((b) => {
                    const lawan = akunPeta.get(b.akun)?.lawan
                    return (
                      <tr key={b._k} className={b.ai ? 'catat-baris-ai' : undefined}>
                        <td>
                          {pilihAkun(b)}
                          {(lawan === 'utang' || lawan === 'piutang') && (
                            <label className="flex items-center gap-2 text-xs text-muted-foreground" style={{ marginTop: 4 }}>
                              Jatuh tempo
                              <input className="input-field" type="date" style={{ maxWidth: 160 }} value={b.jatuh_tempo ?? ''} onChange={(e) => ubah(b._k, { jatuh_tempo: e.target.value })} />
                            </label>
                          )}
                        </td>
                        <td><input className="input-field tabular" style={{ textAlign: 'right' }} aria-label="Debit" {...angka(b.debit, (n) => ubah(b._k, { debit: n, ...(n ? { kredit: 0 } : {}) }))} /></td>
                        <td><input className="input-field tabular" style={{ textAlign: 'right' }} aria-label="Kredit" {...angka(b.kredit, (n) => ubah(b._k, { kredit: n, ...(n ? { debit: 0 } : {}) }))} /></td>
                        <td><Button type="button" variant="outline" size="icon-sm" title="Tukar debit/kredit baris ini" aria-label="Balik debit kredit" onClick={() => setBaris((bs) => bs.map((x) => (x._k === b._k ? balik(x) : x)))}><ArrowLeftRight /></Button></td>
                        <td style={{ whiteSpace: 'nowrap' }}>
                          {b.ai && <span className="catat-ai">AI</span>}{' '}
                          <Button type="button" variant="ghost" size="icon-sm" aria-label="Hapus baris" onClick={() => setBaris((bs) => bs.filter((x) => x._k !== b._k))}><Trash2 /></Button>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
                <tfoot>
                  <tr><td colSpan={2}><strong>Total</strong></td><td style={{ textAlign: 'right' }}><strong>{formatRp(totalD)}</strong></td><td style={{ textAlign: 'right' }}><strong>{formatRp(totalK)}</strong></td><td colSpan={2} /></tr>
                </tfoot>
              </table>
            </div>
            <p className="text-xs text-muted-foreground">
              Satu baris = satu akun, isi Debit <em>atau</em> Kredit. Uang keluar: akun biaya / barang di Debit, Kas / Bank / Utang di Kredit. Uang masuk: sebaliknya.
              Tombol ⇄ menukar angka satu baris, misalnya bila AI menaruhnya di sisi yang terbalik.
            </p>
          </div>

          {bolehBarang && (
            <div className="card grid gap-3" style={{ padding: 18 }}>
              <h3 className="kpi-label" style={{ margin: 0 }}>Barang {satuBarang?.jenis === 'Penjualan' ? 'yang dijual' : 'yang dibeli'}</h3>
              {satuBarang ? <>
                <PilihRincianBarang key={versiRincian} id="catat-barang" jenis={satuBarang.jenis} nominal={satuBarang.nominal}
                  value={rincian} onChange={setRincianAsli} barang={barang.barang} wajib={wajib} bolehBarangBaru={bolehBarangBaru}
                  onBarangBaru={(b) => setBarang((x) => ({ ...x, barang: [...x.barang, b] }))} />
                {tidakCocok.length > 0 && !rincian.bukan_barang && (
                  <p className="text-xs" role="status" style={{ color: 'var(--text-warn)', margin: 0 }}>
                    Belum ada di daftar barang: {tidakCocok.join(', ')}.{bolehBarangBaru ? ' Tambahkan lewat pilihan "+ Barang baru" di atas, lalu isi jumlahnya.' : ''}
                  </p>
                )}
              </> : (
                <>
                  {!rincian.bukan_barang && <p className="text-sm" style={{ margin: 0 }}>
                    Ada beberapa baris penjualan / pembelian barang. Barang hanya bisa dipilih untuk satu baris: pecah jadi beberapa catatan{wajib ? ' (pencatatan perpetual mewajibkan barang)' : ''}.
                  </p>}
                  <label className="flex items-center gap-2 text-sm">
                    <input type="checkbox" checked={rincian.bukan_barang} onChange={(e) => isiRincian({ rincian: [], bukan_barang: e.target.checked })} /> Bukan barang (jasa)
                  </label>
                </>
              )}
            </div>
          )}
        </div>

        <aside className="grid content-start gap-3">
          <div className="card grid gap-2 tabular" style={{ padding: 18 }}>
            <div className="kpi-label">Ringkasan</div>
            <div className="flex justify-between"><span>Total debit</span><strong>{formatRp(totalD)}</strong></div>
            <div className="flex justify-between"><span>Total kredit</span><strong>{formatRp(totalK)}</strong></div>
            <div className="flex justify-between">
              <span>Debit = Kredit</span>
              <strong style={{ color: selisih === 0 && totalD > 0 ? 'var(--accent-green)' : 'var(--text-danger)' }}>{selisih === 0 && totalD > 0 ? 'Seimbang ✓' : `Selisih ${formatRp(selisih)}`}</strong>
            </div>
            <p className="text-xs text-muted-foreground" style={{ margin: 0 }}>
              {selisih === 0 && totalD > 0 ? 'Langsung masuk Buku Besar, Jurnal Umum, dan laporan keuangan.' : 'Simpan aktif setelah seimbang. Cek baris yang terbalik (⇄) atau angka yang kurang.'}
            </p>
          </div>
          {error && <div className="pill-status pill-red" role="alert" style={{ whiteSpace: 'normal' }}>{error}</div>}
          {mirip && mirip.length > 0 && (
            <div className="note-box warn" role="alert" style={{ margin: 0 }}>
              <strong>Mungkin sudah tercatat.</strong> Ada transaksi dengan arah &amp; nominal kas/bank yang sama di tanggal berdekatan:
              <ul style={{ margin: '6px 0 6px 16px', listStyle: 'disc' }}>
                {mirip.map((x, i) => (
                  <li key={i} className="text-sm">
                    {x.baris} ↔ {x.m.tanggal} · {x.m.invoice_id ? `invoice #${x.m.invoice_id}` : x.m.jenis}{x.m.pihak ? ` · ${x.m.pihak}` : x.m.keterangan ? ` · ${x.m.keterangan}` : ''}
                    {x.m.sumber === 'mutasi' ? ' (dari impor mutasi)' : x.m.sumber === 'foto' ? ' (dari foto nota)' : ''}
                    {x.m.belum_lunas && ' — masih belum lunas: kalau ini pembayarannya, catat lewat Piutang & Utang → Catat pelunasan'}
                  </li>
                ))}
              </ul>
              <label className="flex items-center gap-2 text-sm">
                <input type="checkbox" checked={yakinBeda} onChange={(e) => setYakinBeda(e.target.checked)} />
                Saya sudah cek, ini transaksi yang berbeda — tetap simpan
              </label>
            </div>
          )}
          <div className="flex flex-wrap justify-end gap-2">
            {grup && <Button type="button" variant="ghost" className="text-destructive" disabled={busy} onClick={() => void hapus()}>Hapus</Button>}
            <Button type="button" variant="outline" onClick={() => { if (draftKey) hapusDraft(draftKey); onKembali() }}>Batal</Button>
            {!grup && <Button type="button" variant="outline" disabled={!bisaSimpan} onClick={() => void simpan(true)}>Simpan &amp; catat lagi</Button>}
            <Button type="button" disabled={!bisaSimpan} onClick={() => void simpan(false)}>{busy ? 'Menyimpan…' : 'Simpan'}</Button>
          </div>
          {nota?.item?.length ? (
            <div className="card text-xs" style={{ padding: 14 }}>
              <div className="flex items-center gap-1.5 font-semibold"><Sparkles size={13} aria-hidden="true" /> Barang terbaca dari nota</div>
              <ul style={{ margin: '4px 0 0 16px', listStyle: 'disc' }}>
                {nota.item.map((it, i) => <li key={i}>{it.nama}{it.qty ? ` × ${it.qty}` : ''}{it.harga ? ` — ${formatRp(it.harga)}` : ''}</li>)}
              </ul>
            </div>
          ) : null}
        </aside>
      </div>
    </div>
  )
}
