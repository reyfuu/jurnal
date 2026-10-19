import {
  ArrowLeftRight, BookOpen, Bug, Building2, Calculator, ChartColumn, Database, Factory, FileText, Files, LayoutGrid,
  Package, Receipt, Shield, SlidersVertical, Upload, UserCog, Users, type LucideIcon,
} from 'lucide-react'
import type { View } from './api'

export const NAV: { group: string; items: { view: View; label: string; icon: LucideIcon }[] }[] = [
  { group: 'Ringkasan', items: [{ view: 'dashboard', label: 'Dashboard', icon: LayoutGrid }] },
  {
    group: 'Laporan',
    items: [
      { view: 'laporan-keuangan', label: 'Laporan Keuangan', icon: FileText },
      { view: 'laporan-penjualan', label: 'Laporan Penjualan', icon: ChartColumn },
    ],
  },
  {
    group: 'Operasional',
    items: [
      { view: 'transaksi', label: 'Transaksi', icon: ArrowLeftRight },
      { view: 'piutang-utang', label: 'Piutang & Utang', icon: Files },
      { view: 'stok-gudang', label: 'Stok & Gudang', icon: Package },
      { view: 'produksi-supplier', label: 'Produksi & Supplier', icon: Factory },
    ],
  },
  {
    group: 'Alat',
    items: [
      { view: 'invoice', label: 'Invoice', icon: Receipt },
      { view: 'simulasi-harga', label: 'Simulasi & Harga', icon: SlidersVertical },
      { view: 'pajak-umkm', label: 'Pajak UMKM', icon: Shield },
      { view: 'kalkulator-pajak', label: 'Kalkulator Pajak', icon: Calculator },
      { view: 'perencanaan-pajak', label: 'Perencanaan Pajak', icon: Calculator },
    ],
  },
  {
    group: 'Pengaturan',
    items: [
      { view: 'profil-usaha', label: 'Profil Usaha', icon: Building2 },
      { view: 'impor-data', label: 'Impor Data Awal', icon: Upload },
      { view: 'karyawan', label: 'Karyawan', icon: Users },
      { view: 'pengguna', label: 'Pengguna & Akses', icon: UserCog },
      { view: 'master-data', label: 'Daftar Akun (CoA)', icon: Database },
      { view: 'panduan', label: 'Panduan', icon: BookOpen },
      { view: 'lapor-bug', label: 'Lapor Bug', icon: Bug },
    ],
  },
]

export const VIEW_LABEL = Object.fromEntries(NAV.flatMap((g) => g.items.map((i) => [i.view, i.label]))) as Record<View, string>
