import { useEffect, useState } from 'react'
import { ArrowRight, Calculator, ChevronDown, FileSpreadsheet, ReceiptText, ShieldCheck, Sparkles } from 'lucide-react'
import { KebijakanPrivasi } from '@/components/KebijakanPrivasi'
import { BrandMark } from '@/components/Sidebar'
import { Button } from '@/components/ui/button'
import { getPlans, type Paket } from '@/lib/api'
import { formatRp } from '@/lib/format'

// Landing untuk pengunjung yang belum login. Satu tema (navy aplikasi), satu aksen (amber),
// radius kartu 16px dan tombol pill. Visual memakai screenshot asli aplikasi di public/landing.

const CTA_DAFTAR = 'Daftar sekarang'

const FAQ = [
  ['Apakah saya perlu paham akuntansi?', 'Tidak. Anda cukup menjawab beberapa pertanyaan tentang usaha Anda. Daftar akun, laporan laba rugi, dan neraca disiapkan otomatis dari transaksi yang Anda catat.'],
  ['Bagaimana cara berlangganan?', 'Setelah daftar dan mengisi profil usaha, pilih paket bulanan atau tahunan lalu bayar lewat Midtrans. Semua fitur langsung aktif begitu pembayaran berhasil.'],
  ['Pembayarannya lewat apa?', 'Pembayaran diproses oleh Midtrans. Anda bisa memilih metode yang tersedia di halaman pembayaran, seperti virtual account bank, QRIS, e-wallet, atau kartu.'],
  ['Dari mana angka benchmark DJP-nya?', 'Pembandingnya adalah rata-rata NPM dan CTTOR nasional per bidang usaha (KLU) dari buku sektoral DJP, data SPT tahun pajak 2020. Hasilnya acuan untuk bersiap, bukan nasihat pajak; untuk keputusan pajak tetap konsultasikan dengan konsultan pajak.'],
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
          <div key={p.id} className={`flex flex-col gap-5 rounded-2xl border p-7 ${p.hemat ? 'border-[var(--accent-amber)] bg-[rgba(234,179,8,0.06)]' : 'border-[var(--border-shell)] bg-[var(--bg-card)]'}`}>
            <div className="flex items-center justify-between gap-3">
              <h3 className="text-lg font-bold">{p.nama}</h3>
              {p.hemat && <span className="rounded-full bg-[var(--accent-amber)] px-3 py-1 text-xs font-bold text-[#1a1a1a]">{p.hemat}</span>}
            </div>
            <div>
              <span className="text-4xl font-bold tracking-tight tabular">{formatRp(p.harga)}</span>
              <span className="ml-1 text-sm text-[var(--text-secondary)]">/ {p.hari >= 365 ? 'tahun' : 'bulan'}</span>
            </div>
            <p className="text-sm leading-relaxed text-[var(--text-secondary)]">Semua fitur, pengguna tanpa batas, dan pembaruan aturan pajak termasuk.</p>
            <Button size="lg" variant={p.hemat ? 'default' : 'outline'} className="mt-auto rounded-full font-bold active:scale-[0.98]" onClick={onDaftar}>{CTA_DAFTAR}</Button>
          </div>
        ) : (
          <div key={i} className="h-[268px] animate-pulse rounded-2xl border border-[var(--border-shell)] bg-[var(--bg-card)]" aria-hidden="true" />
        ),
      )}
    </div>
  )
}

export function Landing({ onMasuk, onDaftar }: { onMasuk: () => void; onDaftar: () => void }) {
  return (
    <div className="min-h-[100dvh] bg-[var(--brand-navy)] text-[var(--text-primary)]">
      <header className="sticky top-0 z-20 border-b border-white/5 bg-[var(--brand-navy)]/90 backdrop-blur">
        <nav className="mx-auto flex h-16 max-w-6xl items-center justify-between gap-4 px-4 md:px-6" aria-label="Utama">
          <a href="#" className="flex items-center gap-2.5" aria-label="Laporan Bisnis, ke atas">
            <BrandMark size={26} />
            <span className="leading-tight">
              <span className="block text-[15px] font-bold tracking-tight">Laporan Bisnis</span>
              <span className="brand-sub block">by Master Pajak</span>
            </span>
          </a>
          <div className="hidden items-center gap-7 text-sm text-[var(--text-secondary)] md:flex">
            <a href="#fitur" className="hover:text-[var(--text-primary)]">Fitur</a>
            <a href="#harga" className="hover:text-[var(--text-primary)]">Harga</a>
            <a href="#faq" className="hover:text-[var(--text-primary)]">FAQ</a>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="ghost" className="rounded-full" onClick={onMasuk}>Masuk</Button>
            <Button className="hidden rounded-full font-bold sm:inline-flex" onClick={onDaftar}>{CTA_DAFTAR}</Button>
          </div>
        </nav>
      </header>

      <main>
        {/* Hero: teks kiri, screenshot dashboard asli kanan */}
        <section className="mx-auto grid max-w-6xl items-center gap-12 px-4 pb-20 pt-14 md:px-6 lg:grid-cols-[1.1fr_1fr] lg:pt-20">
          <div className="motion-safe:animate-in motion-safe:fade-in motion-safe:slide-in-from-bottom-3 motion-safe:duration-700">
            <h1 className="text-4xl font-bold leading-[1.1] tracking-tight md:text-5xl">
              Pembukuan dan pajak UMKM, <span className="text-[var(--accent-amber-bright)]">rapi tanpa akuntan.</span>
            </h1>
            <p className="mt-5 max-w-[46ch] text-lg leading-relaxed text-[var(--text-secondary)]">
              Catat transaksi harian. Laba rugi, neraca, arus kas, dan hitungan pajak UMKM tersusun otomatis.
            </p>
            <div className="mt-8 flex flex-wrap gap-3">
              <Button size="lg" className="h-12 rounded-full px-6 text-base font-bold active:scale-[0.98]" onClick={onDaftar}>
                {CTA_DAFTAR} <ArrowRight aria-hidden="true" />
              </Button>
              <Button size="lg" variant="outline" className="h-12 rounded-full px-6 text-base" asChild>
                <a href="#fitur">Lihat fitur</a>
              </Button>
            </div>
          </div>
          <figure className="motion-safe:animate-in motion-safe:fade-in motion-safe:slide-in-from-right-4 motion-safe:duration-1000 lg:-mr-24">
            <img
              src="/landing/dashboard.webp" width={1440} height={900} fetchPriority="high"
              alt="Dashboard Laporan Bisnis: omzet, laba bersih, margin, saldo kas, dan grafik bulanan"
              className="w-full rounded-2xl border border-white/10 shadow-[0_30px_80px_-20px_rgba(8,10,30,0.8)]"
            />
          </figure>
        </section>

        {/* Sebelum vs sesudah: dua kolom tanpa kartu */}
        <section className="border-y border-white/5 bg-[#1d2247]">
          <div className="mx-auto grid max-w-6xl gap-10 px-4 py-16 md:grid-cols-2 md:px-6">
            <div>
              <h2 className="text-2xl font-bold tracking-tight">Akhir tahun tidak perlu panik lagi.</h2>
              <p className="mt-3 max-w-[50ch] leading-relaxed text-[var(--text-secondary)]">
                Kebanyakan UMKM baru merapikan catatan saat SPT harus dilaporkan. Laporan Bisnis membuat catatan Anda siap setiap hari.
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
                  <dt className="text-sm text-[var(--text-secondary)] line-through decoration-white/30">{before}</dt>
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
            <article className="overflow-hidden rounded-2xl border border-white/10 bg-[var(--bg-card)] md:col-span-2 md:row-span-2">
              <div className="p-7">
                <h3 className="flex items-center gap-2 text-lg font-bold"><ShieldCheck className="size-5 text-[var(--accent-amber)]" aria-hidden="true" />Cek benchmark DJP & perencanaan pajak</h3>
                <p className="mt-2 max-w-[56ch] text-sm leading-relaxed text-[var(--text-secondary)]">NPM dan CTTOR usaha dibandingkan rata-rata nasional DJP untuk bidang usaha Anda, risiko pemeriksaan ditandai, dan enam skema PPh dibandingkan.</p>
              </div>
              <img src="/landing/pajak.webp" width={1440} height={900} loading="lazy" alt="Halaman Perencanaan Pajak dengan cek benchmark DJP dan perbandingan skema pajak" className="ml-7 w-[calc(100%-1.75rem)] rounded-tl-xl border-l border-t border-white/10" />
            </article>
            <article className="rounded-2xl border border-[var(--accent-amber)]/40 bg-[rgba(234,179,8,0.08)] p-7">
              <FileSpreadsheet className="size-6 text-[var(--accent-amber)]" aria-hidden="true" />
              <h3 className="mt-4 font-bold">Laporan keuangan otomatis</h3>
              <p className="mt-2 text-sm leading-relaxed text-[var(--text-secondary)]">Laba rugi, neraca, dan arus kas sesuai daftar akun usaha Anda, bisa diunduh ke Excel.</p>
            </article>
            <article className="rounded-2xl border border-white/10 bg-[var(--bg-card)] p-7">
              <ReceiptText className="size-6 text-[var(--accent-amber)]" aria-hidden="true" />
              <h3 className="mt-4 font-bold">Invoice PDF</h3>
              <p className="mt-2 text-sm leading-relaxed text-[var(--text-secondary)]">Faktur dengan PPN, NPWP, rekening, dan tanda tangan, siap dikirim ke pelanggan.</p>
            </article>
            <article className="rounded-2xl border border-white/10 bg-[var(--bg-card)] p-7">
              <Calculator className="size-6 text-[var(--accent-amber)]" aria-hidden="true" />
              <h3 className="mt-4 font-bold">Kalkulator pajak</h3>
              <p className="mt-2 text-sm leading-relaxed text-[var(--text-secondary)]">PPh 21, PPh 23, PPh badan, dan PPh final UMKM dihitung dari data Anda.</p>
            </article>
            <article className="rounded-2xl border border-white/10 bg-[linear-gradient(135deg,#2b3266,#1d2247)] p-7 md:col-span-2">
              <Sparkles className="size-6 text-[var(--accent-amber)]" aria-hidden="true" />
              <h3 className="mt-4 font-bold">Laporan AI dan Tanya AI Pajak</h3>
              <p className="mt-2 max-w-[60ch] text-sm leading-relaxed text-[var(--text-secondary)]">Ringkasan kondisi usaha dalam bahasa sehari-hari, plus asisten untuk pertanyaan pajak sehari-hari.</p>
            </article>
          </div>
        </section>

        {/* Cara mulai: daftar berurutan vertikal */}
        <section className="border-y border-white/5 bg-[#1d2247]">
          <div className="mx-auto grid max-w-6xl gap-10 px-4 py-20 md:grid-cols-[1fr_1.4fr] md:px-6">
            <h2 className="text-3xl font-bold tracking-tight md:text-4xl">Mulai dalam 10 menit.</h2>
            <ol className="grid gap-8">
              {[
                ['Daftar akun', 'Cukup nama, email, nomor WhatsApp, dan password.'],
                ['Kenalkan usaha Anda', 'Jawab pertanyaan sederhana: jenis dan bidang usaha, lokasi, omzet, dan bentuk usaha. Kami siapkan daftar akun dan cek awal pajaknya.'],
                ['Pilih paket & bayar', 'Bulanan atau tahunan, bayar lewat Midtrans: virtual account, QRIS, e-wallet, atau kartu.'],
                ['Catat atau impor transaksi', 'Ketik langsung atau unggah Excel. Laporan dan dashboard langsung terisi.'],
              ].map(([judul, isi], i) => (
                <li key={judul} className="grid grid-cols-[3rem_1fr] gap-4">
                  <span className="flex size-12 items-center justify-center rounded-full border border-[var(--accent-amber)]/50 text-lg font-bold text-[var(--accent-amber-bright)] tabular" aria-hidden="true">{i + 1}</span>
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
          <p className="mt-3 text-[var(--text-secondary)]">Semua fitur dalam satu paket. Pilih bulanan atau tahunan.</p>
          <div className="mt-10"><Pricing onDaftar={onDaftar} /></div>
        </section>

        <section id="faq" className="mx-auto max-w-3xl scroll-mt-20 px-4 pb-20 md:px-6">
          <h2 className="text-3xl font-bold tracking-tight">Pertanyaan yang sering ditanyakan</h2>
          <div className="mt-8 grid gap-3">
            {FAQ.map(([q, a]) => (
              <details key={q} className="group rounded-2xl border border-white/10 bg-[var(--bg-card)] px-6 py-4 open:pb-5">
                <summary className="flex cursor-pointer list-none items-center justify-between gap-4 font-semibold [&::-webkit-details-marker]:hidden">
                  {q}
                  <ChevronDown className="size-5 shrink-0 text-[var(--text-secondary)] transition-transform group-open:rotate-180" aria-hidden="true" />
                </summary>
                <p className="mt-3 leading-relaxed text-[var(--text-secondary)]">{a}</p>
              </details>
            ))}
          </div>
        </section>

        <section className="mx-auto max-w-6xl px-4 pb-20 md:px-6">
          <div className="flex flex-col items-start justify-between gap-6 rounded-2xl bg-[var(--accent-amber)] px-8 py-10 text-[#1a1a1a] md:flex-row md:items-center">
            <div>
              <h2 className="text-2xl font-bold tracking-tight md:text-3xl">Rapikan pembukuan usaha mulai hari ini.</h2>
              <p className="mt-2 text-[#3a3320]">Isi profil usaha, pilih paket, langsung pakai.</p>
            </div>
            <Button size="lg" className="h-12 rounded-full bg-[#1a1d3a] px-6 text-base font-bold text-white hover:bg-[#262b55] active:scale-[0.98]" onClick={onDaftar}>
              {CTA_DAFTAR} <ArrowRight aria-hidden="true" />
            </Button>
          </div>
        </section>
      </main>

      <footer className="border-t border-white/5">
        <div className="mx-auto flex max-w-6xl flex-col gap-3 px-4 py-8 text-sm text-[var(--text-secondary)] md:flex-row md:items-center md:justify-between md:px-6">
          <div className="flex items-center gap-2"><BrandMark size={18} /> Laporan Bisnis by Master Pajak</div>
          <div className="flex items-center gap-4">
            <KebijakanPrivasi>
              <button type="button" className="underline-offset-4 hover:underline">Kebijakan Privasi</button>
            </KebijakanPrivasi>
            <span>© {new Date().getFullYear()} Master Pajak</span>
          </div>
        </div>
      </footer>
    </div>
  )
}
