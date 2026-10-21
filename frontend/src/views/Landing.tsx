import { useEffect, useState, type FormEvent, type ReactNode } from 'react'
import { ArrowRight, ChartColumn, Check, Clock, CreditCard } from 'lucide-react'
import { KebijakanPrivasi } from '@/components/KebijakanPrivasi'
import { Logo, Monogram } from '@/components/Logo'
import { PaketHarga } from '@/components/PaketHarga'
import { SegeraHadir } from '@/components/SegeraHadir'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { getPlans, type Paket } from '@/lib/api'
import { formatRpRingkas } from '@/lib/format'
import type { AwalDaftar } from '@/views/Login'
import './landing.css'

// Landing untuk pengunjung yang belum login, mengikuti mockup "Master Pajak V2 Revisi".
// Fitur yang dijanjikan tetapi belum bisa dipakai diberi penanda <SegeraHadir />. Slot yang
// butuh data asli (jumlah UMKM, foto tim, testimoni) belum ditampilkan.

const SEGERA = true

const PERSONA: [typeof Clock, string, string][] = [
  [ChartColumn, 'UMKM dari berbagai skala', 'Usaha terus berjalan, tapi pencatatan masih cara tradisional dan tanpa tim pajak internal.'],
  [Clock, 'Pernah mencoba software akuntansi, lalu menyerah', 'Harus paham jurnal, debit, kredit, dan tutup buku dulu sebelum bisa memakainya.'],
  [CreditCard, 'Menyewa konsultan pajak terasa mahal', 'Biaya konsultan tradisional bisa Rp5 sampai 10 juta per bulan.'],
]

const RANTAI = [
  ['Pencatatan bulanan tidak rapi', 'Nota tercecer, penjualan dari kasir, marketplace, dan transfer bank dicatat di tempat berbeda.'],
  ['Laporan bulanan tidak sama dengan tahunan', 'Angka yang dilaporkan setiap bulan berbeda dengan angka di SPT Tahunan.'],
  ['Perlakuan PKP dan Non-PKP tertukar', 'Pajak yang seharusnya bisa dikreditkan malah dibiayakan, atau sebaliknya.'],
  ['SP2DK datang', 'Anda harus menjelaskan selisih data yang sebenarnya bisa dicegah sejak awal.'],
]

const LANGKAH: [string, string, string, string, boolean?][] = [
  ['1', 'foto', 'Foto nota, kirim via WhatsApp', 'AI membaca nota Anda (OCR) dan langsung menjadikannya jurnal pembukuan. Sementara itu, catat transaksi di aplikasi atau impor mutasi bank dari PDF dan Excel.', SEGERA],
  ['2', 'status', 'PKP dan Non-PKP dibedakan otomatis', 'Untuk PKP, pajak masukan dicatat sebagai kredit pajak. Untuk Non-PKP, dicatat sebagai kapitalisasi.', SEGERA],
  ['4', 'laporan', 'Laporan keuangan tersusun sendiri', 'Laba Rugi, Neraca, Arus Kas, serta Penyusutan dan Amortisasi.'],
  ['XML', '', 'Siap unggah ke Coretax', 'Hasil akhir diekspor ke format XML. Anda tinggal memeriksa dan melaporkan.', SEGERA],
]

const AI_POIN = [
  [<>24<i className="not-italic text-[var(--yellow-600)]">/</i>7</>, 'Tanpa jam kantor. Tetap aktif di akhir pekan dan hari libur.'],
  ['Di mana pun', 'Dari HP, tablet, atau laptop, selama ada internet.'],
  ['Diawasi ahli', 'Sistem diawasi ahli pajak dan mantan petugas DJP berpengalaman 20+ tahun.'],
] as const

// [baris, software konvensional, konsultan tradisional, Master Pajak]; kelas y/x/m = ya/tidak/netral
const BANDING: [string, [string, string], [string, string], ReactNode][] = [
  ['Perlu paham debit, kredit, tutup buku', ['x', 'Ya'], ['y', 'Tidak'], 'Tidak'],
  ['Input dari foto nota via WhatsApp', ['x', 'Tidak'], ['m', 'Anda kirim berkas'], <>Ya, dibaca AI <SegeraHadir /></>],
  ['PKP dan Non-PKP dibedakan otomatis', ['m', 'Diatur manual'], ['y', 'Ya'], <>Otomatis <SegeraHadir /></>],
  ['Rekonsiliasi bulanan vs tahunan', ['m', 'Manual'], ['m', 'Tergantung jasa'], <>Otomatis <SegeraHadir /></>],
  ['File XML siap Coretax', ['m', 'Tergantung'], ['y', 'Ya'], <>Ya <SegeraHadir /></>],
  ['Diawasi ahli pajak', ['x', 'Tidak'], ['y', 'Ya'], 'Ya, termasuk eks petugas DJP'],
]

const DAPAT: [string, boolean?][] = [
  ['Foto nota via WhatsApp jadi jurnal otomatis', SEGERA],
  ['Data marketplace, kasir, dan bank dalam satu pembukuan'],
  ['Perlakuan PKP dan Non-PKP otomatis', SEGERA],
  ['Laba Rugi, Neraca, Arus Kas, Penyusutan'],
  ['Rekonsiliasi bulanan vs tahunan', SEGERA],
  ['Ekspor XML siap unggah ke Coretax', SEGERA],
  ['Konsultan Pajak AI di dalam aplikasi'],
  ['Zoom mingguan bersama tim ahli'],
]

const FAQ: [string, ReactNode][] = [
  ['Saya tidak paham akuntansi. Apakah tetap bisa memakai Master Pajak?', 'Bisa. Master Pajak dibuat supaya Anda tidak perlu memahami debit, kredit, rekonsiliasi, atau tutup buku. Anda cukup memasukkan data, misalnya mencatat transaksi atau mengimpor mutasi bank, dan AI yang menyusun pembukuannya. Foto nota via WhatsApp segera hadir.'],
  ['Apakah Master Pajak menggantikan software akuntansi saya?', 'Ya, Master Pajak dirancang sebagai pengganti software akuntansi konvensional. Pembukuan, laporan keuangan, dan pajak ada di satu platform.'],
  ['Bedanya dengan Coretax DJP?', 'Coretax adalah tempat melapor. Master Pajak menyiapkan datanya: membukukan dan menghitung pajak. Rekonsiliasi otomatis dan ekspor file XML siap unggah ke Coretax segera hadir.'],
  ['Siapa yang mengawasi hasil hitungannya?', 'Ahli pajak dan mantan petugas DJP berpengalaman lebih dari 20 tahun dalam tim Master Pajak. Untuk keputusan penting, Anda tetap disarankan mengecek dasar hukumnya.'],
  ['Bagaimana kalau ada aturan pajak baru?', 'Tim ahli kami mengadakan Zoom mingguan untuk membahas pembaruan aturan pajak dan menjawab pertanyaan komunitas.'],
  ['Pembayarannya lewat apa?', 'Pembayaran diproses oleh Midtrans: virtual account bank, QRIS, e-wallet, atau kartu.'],
  ['Bagaimana kalau ternyata tidak cocok?', 'Anda bisa mengajukan pengembalian dana dalam 7 hari sejak pembayaran pertama. Garansi gugur jika Anda sudah mengekspor data dalam bentuk PDF atau Excel.'],
]

function Ceklis({ className = 'size-[22px]' }: { className?: string }) {
  return <Check className={`shrink-0 text-[var(--text-brand)] ${className}`} strokeWidth={2.2} aria-hidden="true" />
}

/** Ilustrasi sesi Zoom Klinik Master (dari mockup), warna lewat token brand. */
function IlustrasiKlinik() {
  const f = (c: string) => ({ style: { fill: `var(--${c})` } })
  return (
    <svg viewBox="0 0 560 400" role="img" aria-label="Ilustrasi sesi Zoom mingguan Klinik Master" className="block h-auto w-full">
      <rect width="560" height="400" rx="24" {...f('navy-50')} />
      <rect x="36" y="36" width="488" height="300" rx="16" {...f('navy-900')} />
      <rect x="52" y="52" width="224" height="134" rx="10" {...f('navy-600')} /><circle cx="164" cy="104" r="26" {...f('navy-300')} /><path d="M118 172 c6-30 86-30 92 0" {...f('navy-300')} />
      <rect x="62" y="160" width="96" height="18" rx="9" {...f('yellow-500')} />
      <text x="72" y="173" fontFamily="Public Sans, Arial" fontSize="10.5" fontWeight="700" {...f('neutral-900')}>Tim ahli pajak</text>
      <rect x="284" y="52" width="224" height="134" rx="10" {...f('navy-700')} /><circle cx="396" cy="104" r="26" {...f('navy-200')} /><path d="M350 172 c6-30 86-30 92 0" {...f('navy-200')} />
      <rect x="52" y="194" width="224" height="126" rx="10" {...f('navy-700')} /><circle cx="164" cy="242" r="24" {...f('navy-200')} /><path d="M120 306 c6-28 82-28 88 0" {...f('navy-200')} />
      <rect x="284" y="194" width="224" height="126" rx="10" {...f('navy-600')} />
      <rect x="304" y="214" width="184" height="86" rx="8" {...f('neutral-0')} />
      <text x="318" y="238" fontFamily="Rubik, Arial" fontSize="12" fontWeight="700" {...f('navy-700')}>Aturan terbaru</text>
      <rect x="318" y="252" width="40" height="34" rx="4" {...f('navy-200')} /><rect x="366" y="240" width="40" height="46" rx="4" {...f('navy-600')} /><rect x="414" y="232" width="40" height="54" rx="4" {...f('yellow-500')} />
      <circle cx="230" cy="364" r="16" {...f('neutral-0')} /><circle cx="280" cy="364" r="16" {...f('neutral-0')} /><circle cx="330" cy="364" r="16" {...f('error')} />
      <rect x="392" y="300" width="150" height="60" rx="14" {...f('neutral-0')} />
      <text x="408" y="326" fontFamily="Public Sans, Arial" fontSize="12" fontWeight="700" {...f('neutral-900')}>Klinik Master</text>
      <text x="408" y="344" fontFamily="Public Sans, Arial" fontSize="11" {...f('neutral-600')}>Zoom mingguan</text>
    </svg>
  )
}

/** Form "Kalkulator gratis": isiannya dibawa ke form Daftar, lalu pengunjung masuk ke Alat Gratis. */
function FormGratis({ onDaftar }: { onDaftar: (a: AwalDaftar) => void }) {
  const [a, setA] = useState<AwalDaftar>({ name: '', email: '', wa: '', setuju_pdp: false })
  const [error, setError] = useState('')
  function kirim(e: FormEvent) {
    e.preventDefault()
    if (!a.name?.trim() || !a.email?.trim() || !a.wa?.trim()) return setError('Isi nama, email, dan nomor WhatsApp.')
    if (!a.setuju_pdp) return setError('Centang persetujuan pengolahan data pribadi untuk melanjutkan.')
    onDaftar({ ...a, alat: true })
  }
  const field = 'h-12 bg-[var(--bg-card)] md:text-base'
  return (
    <form onSubmit={kirim} noValidate className="mt-[30px] grid gap-4 rounded-[18px] border border-[var(--border-subtle)] bg-[var(--bg-raised)] p-7">
      <label className="grid gap-1.5 text-sm font-semibold">Nama
        <Input className={field} autoComplete="name" placeholder="Nama lengkap" value={a.name} onChange={(e) => setA({ ...a, name: e.target.value })} />
      </label>
      <label className="grid gap-1.5 text-sm font-semibold">Email
        <Input className={field} type="email" autoComplete="email" placeholder="nama@usaha.com" value={a.email} onChange={(e) => setA({ ...a, email: e.target.value })} />
      </label>
      <label className="grid gap-1.5 text-sm font-semibold">WhatsApp
        <Input className={field} type="tel" autoComplete="tel" placeholder="08xx xxxx xxxx" value={a.wa} onChange={(e) => setA({ ...a, wa: e.target.value })} />
      </label>
      <label className="flex items-start gap-2.5 text-sm text-[var(--text-secondary)]">
        <input type="checkbox" className="mt-[3px] size-[18px] shrink-0 accent-[var(--navy-700)]" checked={a.setuju_pdp} onChange={(e) => setA({ ...a, setuju_pdp: e.target.checked })} />
        <span>
          Saya menyetujui pengolahan data pribadi saya sesuai Undang-Undang Perlindungan Data Pribadi (UU PDP).{' '}
          <KebijakanPrivasi><button type="button" className="font-semibold text-[var(--text-brand)] underline underline-offset-2">Kebijakan Privasi</button></KebijakanPrivasi>
        </span>
      </label>
      {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
      <Button type="submit" size="lg" className="justify-self-start">Hitung gratis <ArrowRight aria-hidden="true" /></Button>
      <p className="text-[13px] text-[var(--text-secondary)]">Langkah berikutnya: buat password dan kenalkan usaha Anda, lalu kalkulator & perencanaan pajak langsung terbuka.</p>
    </form>
  )
}

export function Landing({ onMasuk, onDaftar }: { onMasuk: () => void; onDaftar: (awal?: AwalDaftar) => void }) {
  const [paket, setPaket] = useState<Paket[] | null>(null)
  const [paketGagal, setPaketGagal] = useState(false)
  useEffect(() => {
    getPlans().then((r) => setPaket(r.paket), () => setPaketGagal(true))
  }, [])
  const bulanan = paket?.filter((p) => p.hari < 365) ?? []
  const termurah = bulanan.length ? Math.min(...bulanan.map((p) => p.harga)) : 199_000
  function pilihPaket(p: Paket) {
    try { sessionStorage.setItem('paket-pilihan', p.id) } catch { /* mode privat */ }
    onDaftar()
  }
  const tautan = 'rounded-sm px-2.5 py-2 text-[15px] font-medium text-[var(--text-secondary)] hover:text-[var(--text-primary)]'

  return (
    <div className="lp min-h-[100dvh]">
      <header className="sticky top-0 z-20 border-b border-[var(--border-subtle)] bg-[var(--bg-card)]/95 backdrop-blur">
        <nav className="lp-wide flex h-16 items-center justify-between gap-4" aria-label="Utama">
          <a href="#atas" className="shrink-0 rounded-md" aria-label="Master Pajak, ke atas">
            <span className="sm:hidden"><Logo tinggi={22} /></span><span className="hidden sm:inline"><Logo tinggi={26} /></span>
          </a>
          <div className="flex items-center gap-1 sm:gap-2">
            <div className="hidden items-center lg:flex">
              <a href="#kalkulator" className={tautan}>Kalkulator gratis</a>
              <a href="#klinik" className={tautan}>Klinik Master</a>
              <a href="#harga" className={tautan}>Harga</a>
              <a href="#faq" className={tautan}>FAQ</a>
            </div>
            <Button variant="ghost" size="sm" className="px-2 sm:px-3" onClick={onMasuk}>Masuk</Button>
            <Button size="sm" asChild><a href="#harga">Beli sekarang</a></Button>
          </div>
        </nav>
      </header>

      <main id="atas">
        <section className="lp-hero">
          <div className="lp-col">
            <span className="lp-pill"><i />Platform pajak berbasis AI yang siap melayani Anda 24/7, kapan pun dan di mana pun</span>
            <h1>Pembukuan rapi, <mark>pajak beres.</mark></h1>
            <p className="lp-sub">
              Tanpa perlu paham debit, kredit, rekonsiliasi, atau tutup buku. Catat transaksi atau impor mutasi bank, AI Master Pajak yang <b>membukukan, menghitung pajak, dan menyusun laporan keuangan Anda.</b>
            </p>
            <p className="mx-auto mt-3 max-w-[34em] text-base text-[var(--text-secondary)]">Cukup foto nota via WhatsApp, lalu terima file XML siap unggah ke Coretax <SegeraHadir /></p>
            <div className="mt-9 flex flex-wrap justify-center gap-3">
              <Button size="lg" asChild><a href="#harga">Beli sekarang <ArrowRight aria-hidden="true" /></a></Button>
              <Button size="lg" variant="outline" asChild><a href="#kalkulator">Coba kalkulator pajak gratis</a></Button>
            </div>
            <div className="lp-trust">
              <span><Check aria-hidden="true" />Diawasi ahli pajak dan eks petugas DJP</span>
              <span><Check aria-hidden="true" />Laporan keuangan otomatis</span>
              <span><Check aria-hidden="true" />Garansi 7 hari</span>
            </div>
          </div>
        </section>

        <section className="lp-preview" aria-label="Contoh tampilan rekonsiliasi">
          <div className="lp-wide">
            <div className="lp-frame">
              <div className="lp-chrome" aria-hidden="true"><i /><i /><i /><span>Rekonsiliasi bulanan vs tahunan · contoh tampilan</span></div>
              <div className="overflow-x-auto p-[22px]">
                <div className="mb-3.5 flex flex-wrap items-center justify-between gap-3">
                  <b className="font-heading text-[17px] font-semibold">Laporan bulanan vs SPT Tahunan <SegeraHadir /></b>
                  <span className="lp-tag warn">1 selisih perlu dicek</span>
                </div>
                <table className="lp-table">
                  <thead><tr><th>Masa</th><th>Sumber</th><th className="n">Laporan bulanan</th><th className="n">Pembukuan</th><th>Status</th></tr></thead>
                  <tbody>
                    {[
                      ['Juli', 'Nota WhatsApp + bank', '412.500.000', '412.500.000', ''],
                      ['Agustus', 'Marketplace + bank', '438.200.000', '438.200.000', ''],
                      ['September', 'Marketplace + kasir', '451.000.000', '466.750.000', 'Selisih Rp15,7 jt'],
                      ['Oktober', 'Nota WhatsApp + bank', '429.900.000', '429.900.000', ''],
                    ].map(([masa, sumber, bulanan, buku, selisih]) => (
                      <tr key={masa} className={selisih ? 'flag' : undefined}>
                        <td>{masa}</td><td>{sumber}</td><td className="n">Rp {bulanan}</td><td className="n">Rp {buku}</td>
                        <td>{selisih ? <span className="lp-tag warn">{selisih}</span> : <span className="lp-tag ok">Cocok</span>}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
            <p className="mt-4 text-center text-sm text-[var(--text-muted)]">Selisih antara laporan bulanan dan tahunan ditandai sejak awal, sebelum menjadi alasan datangnya SP2DK. Angka di atas ilustrasi.</p>
          </div>
        </section>

        <section>
          <div className="lp-col">
            <span className="lp-kicker">Kenal dengan ini?</span>
            <h2 className="lp-h2">Usaha terus jalan, tapi pembukuan masih di buku tulis, Excel, dan tumpukan nota?</h2>
            <div className="mt-9 grid gap-3">
              {PERSONA.map(([Ikon, judul, isi]) => (
                <div key={judul} className="grid grid-cols-[44px_1fr] items-start gap-4 rounded-[14px] border border-[var(--border-subtle)] p-5">
                  <span className="grid size-11 place-items-center rounded-xl bg-[var(--bg-card-hover)] text-[var(--ink)]"><Ikon className="size-[22px]" aria-hidden="true" /></span>
                  <div><h3 className="text-lg font-semibold">{judul}</h3><p className="mt-1 text-[15.5px] lp-muted">{isi}</p></div>
                </div>
              ))}
            </div>
            <p className="mt-7 rounded-r-xl border-l-4 border-[var(--yellow-500)] bg-[var(--bg-raised)] px-6 py-[22px] text-lg">
              Ini bukan karena Anda kurang teliti. Akuntansi memang dibuat untuk akuntan, bukan untuk pemilik usaha. <mark>Master Pajak dibuat untuk Anda.</mark>
            </p>
          </div>
        </section>

        <section>
          <div className="lp-col">
            <span className="lp-kicker">Kenapa ini penting</span>
            <h2 className="lp-h2">Di pajak, selisih kecil bisa jadi masalah besar.</h2>
            <ol className="lp-chain">
              {RANTAI.map(([judul, isi]) => <li key={judul}><h3>{judul}</h3><p>{isi}</p></li>)}
            </ol>
          </div>
        </section>

        <section>
          <div className="lp-col">
            <span className="lp-kicker">Siapa di balik Master Pajak</span>
            <h2 className="lp-h2">Teknologi AI, diawasi ahli pajak.</h2>
            <p className="lp-lead">Pembukuan otomatis dari sistem diawasi ahli pajak dan mantan petugas DJP yang berpengalaman lebih dari 20 tahun. Setiap minggu, tim ahli juga membagikan pembaruan aturan pajak lewat Zoom.</p>
            <div className="mt-[22px] flex flex-wrap gap-2.5">
              {['Muamalah Pajak', 'Mahir Pajak', 'AI Preneur'].map((x) => (
                <span key={x} className="rounded-full border border-[var(--border-subtle)] px-3.5 py-1.5 text-sm font-semibold text-[var(--ink)]">{x}</span>
              ))}
            </div>
          </div>
        </section>

        <section>
          <div className="lp-col">
            <span className="lp-kicker">Cara kerjanya</span>
            <h2 className="lp-h2">Anda cukup kirim datanya. Sisanya dikerjakan AI.</h2>
            <div className="mt-10 border-t border-[var(--border-subtle)]">
              {LANGKAH.map(([angka, satuan, judul, isi, segera]) => (
                <div key={judul} className="lp-stat">
                  <div className="big">{angka}{satuan && <small>{satuan}</small>}</div>
                  <div><h3 className="text-xl font-semibold">{judul}{segera && <SegeraHadir />}</h3><p className="mt-1.5 text-base lp-muted">{isi}</p></div>
                </div>
              ))}
            </div>
          </div>
        </section>

        <section id="konsultan-ai">
          <div className="lp-col">
            <span className="lp-kicker">Konsultan Pajak AI</span>
            <h2 className="lp-h2">Tanya pajak jam 2 pagi? Tetap dijawab.</h2>
            <p className="lp-lead">Konsultan Pajak AI siap melayani Anda 24/7, kapan pun dan di mana pun, cukup dari HP atau laptop. Tanya soal kondisi pajak, cara pembayaran, format pelaporan, sampai analisis bisnis seperti ROI, ROE, dan apakah menambah modal menguntungkan untuk usaha Anda.</p>
            <div className="mt-[26px] grid gap-3 sm:grid-cols-3">
              {AI_POIN.map(([judul, isi], i) => (
                <div key={i} className="rounded-2xl border border-[var(--border-subtle)] bg-[var(--bg-card)] p-4">
                  <b className="font-heading block text-[22px] leading-none font-bold text-[var(--text-brand)]">{judul}</b>
                  <span className="mt-2 block text-sm leading-normal lp-muted">{isi}</span>
                </div>
              ))}
            </div>
            <div className="lp-chat" aria-label="Contoh percakapan dengan Konsultan Pajak AI">
              <div className="mb-4 flex items-center justify-between border-b border-white/15 pb-3.5 text-[13px]">
                <b className="flex items-center gap-2 text-sm text-white"><span className="dark inline-flex"><Monogram size={20} /></span>Konsultan Pajak AI</b>
                <span className="lp-online flex items-center gap-1.5 font-semibold"><i />Online 24/7</span>
              </div>
              <div className="lp-msg me">Omzet saya bulan ini Rp620 juta. Pajak apa saja yang harus saya bayar?<small>02.14</small></div>
              <div className="lp-msg ai">Berdasarkan pembukuan Anda, usaha Anda tercatat sebagai PKP. Bulan ini ada PPN yang perlu disetor dan PPh yang perlu dihitung. Rinciannya sudah saya siapkan di dashboard.<small>02.14</small></div>
              <div className="lp-msg me">Kalau saya tambah modal Rp500 juta untuk buka cabang, apakah ROE usaha saya naik atau turun?<small>02.16</small></div>
              <span className="lp-typing" aria-hidden="true"><i /><i /><i /></span>
              <div className="mt-3.5 flex items-center gap-2.5 border-t border-white/15 pt-3.5 text-xs text-[var(--navy-200)]">
                <Check className="size-[18px] shrink-0 text-[var(--yellow-500)]" aria-hidden="true" />
                Diawasi ahli pajak dan mantan petugas DJP. Kasus kompleks dibahas di Zoom mingguan bersama tim ahli.
              </div>
            </div>
          </div>
        </section>

        <section id="klinik">
          <div className="lp-wide grid items-center gap-8 lg:grid-cols-[1fr_1.05fr] lg:gap-14">
            <IlustrasiKlinik />
            <div>
              <span className="lp-kicker">Klinik Master</span>
              <h2 className="lp-h2">Software yang menghitung, ahli yang mengawasi.</h2>
              <p className="lp-lead" style={{ fontSize: 17 }}>Setiap pelanggan mendapat Zoom mingguan bersama tim ahli: update peraturan pajak terbaru dan panduan komunitas. Tim diisi ahli pajak dan mantan petugas DJP berpengalaman lebih dari 20 tahun.</p>
              <dl className="mt-[26px] border-t border-[var(--border-subtle)]">
                {[['Format', 'Zoom mingguan'], ['Isi sesi', 'Update aturan pajak dan tanya jawab'], ['Kasus kompleks', 'Paket Premium Master']].map(([k, v]) => (
                  <div key={k} className="flex justify-between gap-4 border-b border-[var(--border-subtle)] py-3 text-[15px]">
                    <dt className="lp-muted">{k}</dt><dd className="text-right font-semibold">{v}</dd>
                  </div>
                ))}
              </dl>
              <div className="mt-7 flex flex-wrap gap-3">
                <Button size="lg" asChild><a href="#harga">Beli sekarang <ArrowRight aria-hidden="true" /></a></Button>
                <Button size="lg" variant="outline" asChild><a href="#kalkulator">Hitung pajak gratis</a></Button>
              </div>
            </div>
          </div>
        </section>

        <section>
          <div className="lp-wide">
            <div className="max-w-[720px]"><span className="lp-kicker">Perbandingan</span><h2 className="lp-h2">Cara lama vs Master Pajak</h2></div>
            <div className="lp-cmp">
              <table>
                <thead><tr><th><span className="sr-only">Hal</span></th><th>Software akuntansi konvensional</th><th>Konsultan pajak tradisional</th><th className="me">Master Pajak</th></tr></thead>
                <tbody>
                  {BANDING.map(([hal, [ks, kv], [ts, tv], mp]) => (
                    <tr key={hal}><td>{hal}</td><td className={ks}>{kv}</td><td className={ts}>{tv}</td><td className="me">{mp}</td></tr>
                  ))}
                  <tr>
                    <td>Biaya</td><td className="m">Lisensi + waktu belajar</td><td className="x">Rp5 sampai 10 jt / bulan</td>
                    <td className="me">Mulai {formatRpRingkas(termurah)} / bulan</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </section>

        <section id="kalkulator">
          <div className="lp-col">
            <span className="lp-kicker">Gratis</span>
            <h2 className="lp-h2">Kalkulator & perencanaan pajak gratis</h2>
            <p className="lp-lead">Belum siap berlangganan? Hitung dulu estimasi pajak dan rencana pajak usaha Anda, gratis. Isi data di bawah untuk mendapatkan akses.</p>
            <FormGratis onDaftar={onDaftar} />
          </div>
        </section>

        <section>
          <div className="lp-col">
            <span className="lp-kicker">Yang Anda dapatkan</span>
            <h2 className="lp-h2">Satu platform untuk akuntansi dan pajak usaha Anda.</h2>
            <ul className="mt-8 grid gap-x-7 gap-y-3.5 sm:grid-cols-2">
              {DAPAT.map(([x, segera]) => <li key={x} className="flex gap-3 text-base"><Ceklis className="mt-0.5 size-[22px]" /><span>{x}{segera && <SegeraHadir />}</span></li>)}
            </ul>
          </div>
        </section>

        <section id="harga" className="text-center">
          <div className="lp-wide">
            <span className="lp-kicker">Harga</span>
            <h2 className="lp-h2">Ratusan ribu per bulan, bukan jutaan.</h2>
            <p className="lp-lead mx-auto max-w-[36em]">Bandingkan dengan konsultan pajak tradisional yang bisa mencapai Rp5 sampai 10 juta per bulan.</p>
            <div className="mt-7">
              {paket ? <PaketHarga paket={paket} label={() => 'Beli sekarang'} onPilih={pilihPaket} />
                : paketGagal ? <p className="text-sm lp-muted">Harga belum bisa dimuat. Silakan muat ulang halaman.</p>
                  : <div className="mx-auto h-[420px] max-w-[1000px] animate-pulse rounded-[18px] bg-[var(--bg-raised)]" aria-hidden="true" />}
            </div>
            <div className="mt-6 flex items-center justify-center gap-3.5 text-left text-[15px] lp-muted">
              <span className="font-heading grid size-[46px] shrink-0 place-items-center rounded-full border-[1.5px] border-[var(--yellow-500)] bg-[var(--yellow-50)] text-center text-xs leading-none font-bold text-[var(--navy-700)]">7<br />hari</span>
              <p><b className="text-[var(--text-primary)]">Garansi 7 hari, FULL REFUND.</b> Untuk pembayaran pertama, selama Anda belum mengekspor data dalam bentuk PDF atau Excel.</p>
            </div>
          </div>
        </section>

        <section id="faq">
          <div className="lp-col">
            <span className="lp-kicker">Pertanyaan yang sering muncul</span>
            <h2 className="lp-h2 mb-7">Sebelum Anda memutuskan</h2>
            {FAQ.map(([q, a], i) => (
              <details key={q} open={i === 0}><summary>{q}</summary><p>{a}</p></details>
            ))}
          </div>
        </section>

        <section>
          <div className="lp-col">
            <span className="lp-kicker">Keputusannya di tangan Anda</span>
            <h2 className="lp-h2">Dua pilihan, satu keputusan.</h2>
            <div className="mt-9 grid gap-4 sm:grid-cols-2">
              <div className="rounded-[18px] border border-[var(--border-subtle)] p-[26px]">
                <span className="text-xs font-bold tracking-[0.12em] text-[var(--text-muted)] uppercase">Pilihan 1</span>
                <h3 className="mt-2.5 text-xl font-semibold">Tetap seperti sekarang</h3>
                <p className="mt-2 text-[15.5px] lp-muted">Nota menumpuk, laporan bulanan dan tahunan tidak sinkron, dan berharap tidak ada SP2DK yang datang.</p>
              </div>
              <div className="rounded-[18px] bg-[var(--navy-700)] p-[26px] text-white">
                <span className="text-xs font-bold tracking-[0.12em] text-[var(--yellow-500)] uppercase">Pilihan 2</span>
                <h3 className="mt-2.5 text-xl font-semibold text-white">Biarkan AI yang membukukan</h3>
                <p className="mt-2 text-[15.5px] text-[var(--navy-100)]">Catat atau impor transaksi, dan pembukuan serta pajak Anda tersusun rapi, diawasi ahli pajak.</p>
                <Button size="lg" variant="accent" className="mt-5" asChild><a href="#harga">Beli sekarang <ArrowRight aria-hidden="true" /></a></Button>
              </div>
            </div>
            <div className="mt-12 rounded-[18px] border border-[var(--border-subtle)] bg-[var(--bg-raised)] px-7 py-[26px] text-[16.5px]">
              <p><b className="font-heading">P.S.</b> Belum yakin? Mulai dari kalkulator dan perencanaan pajak gratis. <a href="#kalkulator" className="font-semibold text-[var(--text-brand)] underline underline-offset-2">Coba sekarang</a>, lalu putuskan setelah melihat angkanya sendiri.</p>
              <p className="mt-3.5 text-sm lp-muted">Tim Master Pajak</p>
            </div>
          </div>
        </section>
      </main>

      <footer className="border-t border-[var(--border-subtle)] py-10 text-sm lp-muted">
        <div className="lp-wide flex flex-wrap items-start justify-between gap-x-10 gap-y-4">
          <div>
            <Logo tinggi={24} />
            <p className="mt-2.5 text-[11px] font-bold tracking-[0.14em] text-[var(--text-muted)] uppercase">Ekosistem Muamalah Pajak · Mahir Pajak · Master Pajak</p>
          </div>
          <div className="grid max-w-[520px] gap-2">
            <p>© {new Date().getFullYear()} PT Master Pajak Indonesia. Tidak berafiliasi dengan Direktorat Jenderal Pajak; pelaporan pajak tetap melalui Coretax DJP.</p>
            <KebijakanPrivasi>
              <button type="button" className="justify-self-start underline underline-offset-4 hover:text-[var(--text-primary)]">Kebijakan Privasi</button>
            </KebijakanPrivasi>
          </div>
        </div>
      </footer>
    </div>
  )
}
