import {
  ArrowLeftRight, BookOpen, CalendarCheck, Hourglass, Gift, MessageSquareText, Sparkles, Bug, FileCheck2, Library, CreditCard, Building2, ChartColumn, Database, Factory, FileText, Files, LayoutGrid, Package,
  Landmark, ListFilter, NotebookText, Receipt, Shield, SlidersVertical, SquarePercent, Target, Upload, UserCog, Users, type LucideIcon,
} from 'lucide-react'
import type { View } from './api'

// Mode terkunci (belum berlangganan): menu yang tetap terbuka, di atas menu lain yang bergembok.
/** Halaman di luar sidebar (dibuka dari banner), supaya punya label untuk catatan & lapor bug. */
export const NAV_LAIN = { group: 'Lain', items: [{ view: 'catatan' as View, label: 'Catatan Konsultan', icon: MessageSquareText }, { view: 'umur-piutang' as View, label: 'Piutang & Utang', icon: Files }] }

export const NAV_COBA = { group: 'Coba gratis', items: [
  { view: 'laporan-ai' as View, label: 'Laporan AI', icon: Sparkles },
  { view: 'alat-gratis' as View, label: 'Alat Gratis', icon: Gift },
] }
export const BEBAS_TERKUNCI: View[] = ['laporan-ai', 'alat-gratis', 'langganan', 'profil-usaha', 'lapor-bug', 'catatan']

export const NAV: { group: string; items: { view: View; label: string; icon: LucideIcon }[] }[] = [
  { group: 'Ringkasan', items: [{ view: 'dashboard', label: 'Dashboard', icon: LayoutGrid }] },
  {
    // Pekerjaan sehari-hari: semua yang dicatat. Tombol "+ Catat" di atas membuka formulirnya langsung.
    group: 'Catat Harian',
    items: [
      { view: 'transaksi', label: 'Transaksi', icon: ArrowLeftRight },
      { view: 'stok-gudang', label: 'Stok & Gudang', icon: Package },
      { view: 'produksi-supplier', label: 'Produksi & Supplier', icon: Factory },
    ],
  },
  {
    group: 'Tagihan',
    items: [
      { view: 'piutang-utang', label: 'Piutang & Utang', icon: Hourglass },
      { view: 'invoice', label: 'Invoice', icon: Receipt },
    ],
  },
  {
    group: 'Laporan',
    items: [
      { view: 'laporan-keuangan', label: 'Laporan Keuangan', icon: FileText },
      { view: 'buku-besar', label: 'Buku Besar', icon: NotebookText },
      { view: 'laporan-transaksi', label: 'Laporan Transaksi', icon: ListFilter },
      { view: 'laporan-penjualan', label: 'Laporan Penjualan', icon: ChartColumn },
      { view: 'tutup-buku', label: 'Tutup Buku Bulanan', icon: CalendarCheck },
    ],
  },
  {
    group: 'Pajak',
    items: [
      { view: 'pajak-umkm', label: 'Pajak UMKM', icon: Shield },
      { view: 'kalkulator-pajak', label: 'Kalkulator Pajak', icon: SquarePercent },
      { view: 'perencanaan-pajak', label: 'Perencanaan Pajak', icon: Target },
      { view: 'pustaka', label: 'Pustaka Pajak', icon: Library },
      { view: 'primbon-pajak', label: 'Primbon Pajak', icon: LibraryBig },
    ],
  },
  {
    group: 'Alat',
    items: [{ view: 'simulasi-harga', label: 'Simulasi & Harga', icon: SlidersVertical }],
  },
  {
    group: 'Pengaturan',
    items: [
      { view: 'profil-usaha', label: 'Profil Usaha', icon: Building2 },
      { view: 'saldo-awal', label: 'Saldo Awal', icon: Landmark },
      { view: 'impor-data', label: 'Impor Data Awal', icon: Upload },
      { view: 'karyawan', label: 'Karyawan', icon: Users },
      { view: 'pengguna', label: 'Pengguna & Akses', icon: UserCog },
      { view: 'langganan', label: 'Langganan', icon: CreditCard },
      { view: 'master-data', label: 'Daftar Akun (CoA)', icon: Database },
      { view: 'panduan', label: 'Panduan', icon: BookOpen },
      { view: 'usulan-koreksi', label: 'Usulan Koreksi', icon: FileCheck2 },
      { view: 'lapor-bug', label: 'Lapor Bug', icon: Bug },
    ],
  },
]

export const VIEW_LABEL = Object.fromEntries([...NAV, NAV_COBA, NAV_LAIN].flatMap((g) => g.items.map((i) => [i.view, i.label]))) as Record<View, string>
