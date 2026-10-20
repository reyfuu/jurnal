import { useEffect, useState } from 'react'
import { ArrowRight, Calculator, ChevronDown, FileSpreadsheet, ReceiptText, ShieldCheck, Sparkles } from 'lucide-react'
import { KebijakanPrivasi } from '@/components/KebijakanPrivasi'
import { Logo } from '@/components/Logo'
import { Button } from '@/components/ui/button'
import { getPlans, type Paket } from '@/lib/api'
import { formatRp } from '@/lib/format'

// Landing untuk pengunjung yang belum login, mengikuti mockup Brand Guideline v1.0 hlm. 25:
// latar Awan, kartu putih radius 16, tombol primer navy, satu titik kuning per layar.
// Mengikuti tema terang/gelap. Visual memakai screenshot asli aplikasi di public/landing.

const CTA_DAFTAR = 'Daftar sekarang'

const FAQ = [
  ['Apakah saya perlu paham akuntansi?', 'Tidak. Anda cukup menjawab beberapa pertanyaan tentang usaha Anda. Daftar akun, laporan laba rugi, dan neraca disiapkan otomatis dari transaksi yang Anda catat.'],
  ['Bagaimana cara berlangganan?', 'Setelah daftar dan mengisi profil usaha, pilih paket bulanan atau tahunan lalu bayar lewat Midtrans. Semua fitur langsung aktif begitu pembayaran berhasil.'],
  ['Ada garansi uang kembali?', 'Ada. Garansi 7 hari uang kembali untuk pembayaran pertama, selama Anda belum mengunduh atau mengekspor data (Excel/PDF). Ajukan dari menu Langganan; akses berhenti dan dana dikembalikan lewat Midtrans dalam 3–7 hari kerja.'],
  ['Pembayarannya lewat apa?', 'Pembayaran diproses oleh Midtrans. Anda bisa memilih metode yang tersedia di halaman pembayaran, seperti virtual account bank, QRIS, e-wallet, atau kartu.'],
  ['Dari mana angka benchmark nasionalnya?', 'Pembandingnya adalah rata-rata NPM dan CTTOR nasional per bidang usaha (KLU) dari buku sektoral DJP, data SPT tahun pajak 2020. Hasilnya acuan untuk bersiap, bukan nasihat pajak; untuk keputusan pajak tetap konsultasikan dengan konsultan pajak.'],
  ['Siapa saja yang bisa memakai akun saya?', 'Owner bisa menambah Admin Keuangan dan Kasir. Tiap peran hanya melihat menu yang sesuai, dan data tiap usaha terpisah dari usaha lain.'],
] as const

function Pricing({ onDaftar }: { onDaftar: () => void }) {
  const [paket, setPaket] = useState<Paket[] | null>(null)
  const [error, setError] = useState(false)
  useEffect(() => {
    getPlans().then((r) => setPaket(r.paket), () => setError(true))
  }, [])

  if (error) return <p className="text-sm text-[var(--text-secondary)]">Harga belum bisa dimuat. Silakan muat ulang halaman.</p>
  return (
    <div className="grid gap-5 md:grid-cols-2">
      {(paket ?? [null, null]).map((p, i) =>
        p ? (
          <div key={p.id} className={`flex flex-col gap-5 rounded-[16px] bg-[var(--bg-card)] p-7 shadow-[var(--shadow-1)] ${p.hemat ? 'border-2 border-[var(--ink)]' : 'border border-[var(--border-subtle)]'}`}>
            <div className="flex items-center justify-between gap-3">
              <h3 className="text-lg font-bold">{p.nama}</h3>
              {p.hemat && <span className="rounded-full bg-[var(--yellow-500)] px-3 py-1 text-xs font-bold text-[var(--navy-900)]">{p.hemat}</span>}
            </div>
            <div>
              <span className="text-4xl font-bold tracking-tight tabular">{formatRp(p.harga)}</span>
              <span className="ml-1 text-sm text-[var(--text-secondary)]">/ {p.hari >= 365 ? 'tahun' : 'bulan'}</span>
            </div>
            <p className="text-sm leading-relaxed text-[var(--text-secondary)]">Semua fitur, pengguna tanpa batas, dan pembaruan aturan pajak termasuk.</p>
            <Button size="lg" variant={p.hemat ? 'default' : 'outline'} className="mt-auto" onClick={onDaftar}>{CTA_DAFTAR}</Button>
          </div>
        ) : (
          <div key={i} className="h-[268px] animate-pulse rounded-[16px] border border-[var(--border-subtle)] bg-[var(--bg-card)]" aria-hidden="true" />
        ),
      )}
    </div>
  )
}

const KARTU = 'rounded-[16px] border border-[var(--border-subtle)] bg-[var(--bg-card)] shadow-[var(--shadow-1)]'
const TAUTAN = 'rounded-sm outline-none hover:text-[var(--ink)] focus-visible:ring-2 focus-visible:ring-ring'

/** Ikon garis di kotak membulat (elemen grafis hlm. 18). */
function Ikon({ icon: Icon, className = 'bg-[var(--bg-card-hover)]' }: { icon: typeof Calculator; className?: string }) {
  return (
    <span className={`grid size-12 place-items-center rounded-[10px] ${className}`}>
      <Icon className="size-6 text-[var(--ink)]" strokeWidth={2} aria-hidden="true" />
    </span>
  )
}

export function Landing({ onMasuk, onDaftar }: { onMasuk: () => void; onDaftar: () => void }) {
  return (
    <div className="min-h-[100dvh] bg-[var(--bg-shell)] text-[var(--text-primary)]">
      <header className="sticky top-0 z-20 border-b border-[var(--border-subtle)] bg-[var(--bg-card)]/90 backdrop-blur">
        <nav className="mx-auto flex h-16 max-w-6xl items-center justify-between gap-4 px-4 md:px-6" aria-label="Utama">
          <a href="#" className="rounded-md outline-none focus-visible:ring-2 focus-visible:ring-ring" aria-label="Master Pajak, ke atas">
            <Logo tinggi={26} />
          </a>
          <div className="hidden items-center gap-8 text-sm font-medium text-[var(--text-secondary)] md:flex">
            <a href="#fitur" className={TAUTAN}>Fitur</a>
            <a href="#harga" className={TAUTAN}>Harga</a>
            <a href="#faq" className={TAUTAN}>FAQ</a>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="outline" onClick={onMasuk}>Masuk</Button>
            <Button className="hidden sm:inline-flex" onClick={onDaftar}>{CTA_DAFTAR}</Button>
          </div>
        </nav>
      </header>

      <main>
        {/* Hero: teks kiri, screenshot dasbor asli kanan; setengah lingkaran seperti mockup hlm. 25 */}
        <section className="relative overflow-hidden">
          <span aria-hidden="true" className="pointer-events-none absolute -right-48 -top-48 size-[30rem] rounded-full bg-[var(--bg-card-hover)]" />
          <div className="relative mx-auto grid max-w-6xl items-center gap-12 px-4 pb-20 pt-14 md:px-6 lg:grid-cols-[1.1fr_1fr] lg:pt-20">
            <div className="motion-safe:animate-in motion-safe:fade-in motion-safe:slide-in-from-bottom-3 motion-safe:duration-700">
              <span className="inline-flex rounded-full bg-[var(--bg-card-hover)] px-3 py-1 text-xs font-bold text-[var(--ink)]">Untuk UMKM dan seller online</span>
              <h1 className="mt-5 text-[40px] font-bold leading-[48px] tracking-tight md:text-5xl md:leading-[56px]">
                Jualan lancar,<br />pajak beres.
              </h1>
              <p className="mt-5 max-w-[46ch] text-lg leading-relaxed text-[var(--text-secondary)]">
                Catat transaksi harian. Laba rugi, neraca, arus kas, dan hitungan pajak UMKM tersusun otomatis.
              </p>
              <div className="mt-8 flex flex-wrap gap-3">
                <Button size="lg" onClick={onDaftar}>
                  {CTA_DAFTAR} <ArrowRight aria-hidden="true" />
                </Button>
                <Button size="lg" variant="outline" asChild>
                  <a href="#harga">Lihat harga</a>
                </Button>
              </div>
            </div>
            <figure className="relative motion-safe:animate-in motion-safe:fade-in motion-safe:slide-in-from-right-4 motion-safe:duration-1000 lg:-mr-24">
              <img
                src="/landing/dashboard.webp" width={1440} height={900} fetchPriority="high"
                alt="Dasbor Master Pajak: omzet, laba bersih, margin, saldo kas, dan grafik bulanan"
                className="w-full rounded-[16px] border border-[var(--border-subtle)] shadow-[var(--shadow-3)]"
              />
              <span aria-hidden="true" className="absolute -left-2.5 -top-2.5 size-5 rounded-full bg-[var(--yellow-500)]" />
            </figure>
          </div>
        </section>

        {/* Sebelum vs sesudah: dua kolom tanpa kartu */}
        <section className="border-y border-[var(--border-subtle)] bg-[var(--bg-card)]">
          <div className="mx-auto grid max-w-6xl gap-10 px-4 py-16 md:grid-cols-2 md:px-6">
            <div>
              <h2 className="text-2xl font-bold tracking-tight">Akhir tahun tidak perlu panik lagi.</h2>
              <p className="mt-3 max-w-[50ch] leading-relaxed text-[var(--text-secondary)]">
                Kebanyakan UMKM baru merapikan catatan saat SPT harus dilaporkan. Master Pajak membuat catatan Anda siap setiap hari.
              </p>
            </div>
            <dl className="grid gap-6 sm:grid-cols-2">
              {[
                ['Nota dan transfer tercecer', 'Satu daftar transaksi, bisa diimpor dari Excel.'],
                ['Tidak tahu untung atau rugi', 'Laba rugi dan neraca per bulan, langsung terlihat.'],
                ['Bingung skema pajak', 'Enam skema dibandingkan, ketahuan mana paling hemat.'],
                ['Invoice dibuat manual', 'Invoice PDF lengkap NPWP dan rekening sekali klik.'],
              ].map(([before, after]) => (
                <div key={before}>
                  <dt className="text-sm text-[var(--text-muted)] line-through decoration-[var(--text-muted)]">{before}</dt>
                  <dd className="mt-1.5 font-semibold leading-snug">{after}</dd>
                </div>
              ))}
            </dl>
          </div>
        </section>

        {/* Fitur: bento 6 sel, satu sel besar berisi screenshot asli */}
        <section id="fitur" className="mx-auto max-w-6xl scroll-mt-20 px-4 py-20 md:px-6">
          <h2 className="max-w-[24ch] text-3xl font-bold tracking-tight md:text-4xl">Semua yang dibutuhkan pembukuan UMKM, dalam satu aplikasi.</h2>
          <div className="mt-10 grid gap-4 md:grid-cols-3 md:grid-rows-[auto_auto_auto]">
            <article className={`overflow-hidden md:col-span-2 md:row-span-2 ${KARTU}`}>
              <div className="p-7">
                <Ikon icon={ShieldCheck} />
                <h3 className="mt-4 text-lg font-bold">Cek benchmark nasional & perencanaan pajak</h3>
                <p className="mt-2 max-w-[56ch] text-sm leading-relaxed text-[var(--text-secondary)]">NPM dan CTTOR usaha dibandingkan rata-rata nasional untuk bidang usaha Anda, risiko pemeriksaan ditandai, dan enam skema PPh dibandingkan.</p>
              </div>
              <img src="/landing/pajak.webp" width={1440} height={900} loading="lazy" alt="Halaman Perencanaan Pajak dengan cek benchmark nasional dan perbandingan skema pajak" className="ml-7 w-[calc(100%-1.75rem)] rounded-tl-[10px] border-l border-t border-[var(--border-subtle)]" />
            </article>
            {/* kartu sorotan: navy (pulau gelap), seperti kartu Klinik Master hlm. 24 */}
            <article className="dark rounded-[16px] bg-[var(--navy-700)] p-7 shadow-[var(--shadow-1)]">
              <Ikon icon={FileSpreadsheet} className="bg-[var(--navy-900)]" />
              <h3 className="mt-4 font-bold">Laporan keuangan otomatis</h3>
              <p className="mt-2 text-sm leading-relaxed text-[var(--text-secondary)]">Laba rugi, neraca, dan arus kas sesuai daftar akun usaha Anda, bisa diunduh ke Excel.</p>
            </article>
            <article className={`p-7 ${KARTU}`}>
              <Ikon icon={ReceiptText} />
              <h3 className="mt-4 font-bold">Invoice PDF</h3>
              <p className="mt-2 text-sm leading-relaxed text-[var(--text-secondary)]">Faktur dengan PPN, NPWP, rekening, dan tanda tangan, siap dikirim ke pelanggan.</p>
            </article>
            <article className={`p-7 ${KARTU}`}>
              <Ikon icon={Calculator} />
              <h3 className="mt-4 font-bold">Kalkulator pajak</h3>
              <p className="mt-2 text-sm leading-relaxed text-[var(--text-secondary)]">PPh 21, PPh 23, PPh badan, dan PPh final UMKM dihitung dari data Anda.</p>
            </article>
            <article className={`p-7 md:col-span-2 ${KARTU}`}>
              <Ikon icon={Sparkles} />
              <h3 className="mt-4 font-bold">Laporan AI dan Tanya AI Pajak</h3>
              <p className="mt-2 max-w-[60ch] text-sm leading-relaxed text-[var(--text-secondary)]">Ringkasan kondisi usaha dalam bahasa sehari-hari, plus asisten untuk pertanyaan pajak sehari-hari.</p>
            </article>
          </div>
        </section>

        {/* Cara mulai: daftar berurutan vertikal */}
        <section className="border-y border-[var(--border-subtle)] bg-[var(--bg-card)]">
          <div className="mx-auto grid max-w-6xl gap-10 px-4 py-20 md:grid-cols-[1fr_1.4fr] md:px-6">
            <h2 className="text-3xl font-bold tracking-tight md:text-4xl">Mulai dalam 10 menit.</h2>
            <ol className="grid gap-8">
              {[
                ['Daftar akun', 'Cukup nama, email, nomor WhatsApp, dan password.'],
                ['Kenalkan usaha Anda', 'Jawab pertanyaan sederhana: jenis dan bidang usaha, lokasi, omzet, dan bentuk usaha. Kami siapkan daftar akun dan cek awal pajaknya.'],
                ['Pilih paket & bayar', 'Bulanan atau tahunan, bayar lewat Midtrans: virtual account, QRIS, e-wallet, atau kartu.'],
                ['Catat atau impor transaksi', 'Ketik langsung atau unggah Excel. Laporan dan dasbor langsung terisi.'],
              ].map(([judul, isi], i) => (
                <li key={judul} className="grid grid-cols-[3rem_1fr] gap-4">
                  <span className="font-heading flex size-12 items-center justify-center rounded-full bg-[var(--bg-card-hover)] text-lg font-bold text-[var(--ink)] tabular" aria-hidden="true">{i + 1}</span>
                  <div>
                    <h3 className="text-lg font-bold">{judul}</h3>
                    <p className="mt-1 leading-relaxed text-[var(--text-secondary)]">{isi}</p>
                  </div>
                </li>
              ))}
            </ol>
          </div>
        </section>

        <section id="harga" className="mx-auto max-w-4xl scroll-mt-20 px-4 py-20 md:px-6">
          <h2 className="text-3xl font-bold tracking-tight md:text-4xl">Harga sederhana.</h2>
          <p className="mt-3 text-[var(--text-secondary)]">Semua fitur dalam satu paket. Pilih bulanan atau tahunan. Garansi 7 hari uang kembali.*</p>
          <div className="mt-10"><Pricing onDaftar={onDaftar} /></div>
          <p className="mt-4 text-xs text-[var(--text-secondary)]">*Untuk pembayaran pertama, selama data belum diunduh/diekspor. Lihat FAQ.</p>
        </section>

        <section id="faq" className="mx-auto max-w-3xl scroll-mt-20 px-4 pb-20 md:px-6">
          <h2 className="text-3xl font-bold tracking-tight">Pertanyaan yang sering ditanyakan</h2>
          <div className="mt-8 grid gap-3">
            {FAQ.map(([q, a]) => (
              <details key={q} className={`group px-6 py-4 open:pb-5 ${KARTU}`}>
                <summary className="flex cursor-pointer list-none items-center justify-between gap-4 rounded-sm font-semibold outline-none focus-visible:ring-2 focus-visible:ring-ring [&::-webkit-details-marker]:hidden">
                  {q}
                  <ChevronDown className="size-5 shrink-0 text-[var(--text-secondary)] transition-transform group-open:rotate-180" aria-hidden="true" />
                </summary>
                <p className="mt-3 leading-relaxed text-[var(--text-secondary)]">{a}</p>
              </details>
            ))}
          </div>
        </section>

        <section className="mx-auto max-w-6xl px-4 pb-20 md:px-6">
          <div className="dark flex flex-col items-start justify-between gap-6 rounded-[16px] bg-[var(--navy-700)] px-8 py-10 md:flex-row md:items-center">
            <div>
              <h2 className="text-2xl font-bold tracking-tight md:text-3xl">Rapikan pembukuan usaha mulai hari ini.</h2>
              <p className="mt-2 text-[var(--text-secondary)]">Isi profil usaha, pilih paket, langsung pakai.</p>
            </div>
            <Button size="lg" variant="accent" onClick={onDaftar}>
              {CTA_DAFTAR} <ArrowRight aria-hidden="true" />
            </Button>
          </div>
        </section>
      </main>

      <footer className="border-t border-[var(--border-subtle)] bg-[var(--bg-card)]">
        <div className="mx-auto grid max-w-6xl gap-5 px-4 py-8 text-sm text-[var(--text-secondary)] md:px-6">
          <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
            <Logo tinggi={24} />
            <div className="flex items-center gap-4">
              <KebijakanPrivasi>
                <button type="button" className="rounded-sm underline-offset-4 outline-none hover:underline focus-visible:ring-2 focus-visible:ring-ring">Kebijakan Privasi</button>
              </KebijakanPrivasi>
              <span>© {new Date().getFullYear()} PT Master Pajak Indonesia</span>
            </div>
          </div>
          <p className="max-w-[90ch] text-xs leading-relaxed text-[var(--text-muted)]">
            Master Pajak adalah aplikasi independen milik PT Master Pajak Indonesia dan tidak berafiliasi dengan Direktorat Jenderal Pajak. Pelaporan pajak tetap melalui Coretax DJP.
          </p>
        </div>
      </footer>
    </div>
  )
}
