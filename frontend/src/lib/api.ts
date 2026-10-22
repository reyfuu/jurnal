import { pesanGalat } from './galat'

// Tipe mengikuti struct JSON di internal/finance dan internal/auth (Go).

// 'arus-kas' dihapus: Arus Kas jadi tab di dalam Laporan Keuangan, bukan menu sendiri.
export type View =
  | 'dashboard' | 'laporan-keuangan' | 'laporan-penjualan' | 'transaksi' | 'piutang-utang' | 'stok-gudang'
  | 'produksi-supplier' | 'invoice' | 'simulasi-harga' | 'pajak-umkm' | 'kalkulator-pajak' | 'perencanaan-pajak' | 'karyawan' | 'master-data'
  | 'panduan' | 'pengguna' | 'profil-usaha' | 'lapor-bug' | 'langganan' | 'umur-piutang' | 'tutup-buku' | 'saldo-awal' | 'buku-besar' | 'jurnal-umum' | 'laporan-transaksi'
  | 'transaksi-kelola' // izin, bukan menu: ubah & hapus transaksi
  | 'impor-data' // menu Impor Data Awal, sekaligus izin mengubah neraca awal
  | 'panel-master' // akun tim Master Pajak: seluruh aplikasi diganti Panel Master
  | 'usulan-koreksi' // pemilik usaha meninjau usulan koreksi tim Master Pajak
  | 'pustaka' // artikel pajak dari tim Master Pajak
  | 'primbon-pajak' // katalog peraturan resmi (khusus pelanggan)
  | 'laporan-ai' | 'alat-gratis' // menu mode terkunci (belum berlangganan), hanya di frontend
  | 'catatan' // catatan konsultan untuk klien, hanya di frontend

export type Role = 'owner' | 'admin' | 'kasir' | 'mp' | 'lihat'

export interface User {
  id: number
  name: string
  email: string
  role: Role
  active: boolean
  created_at: string
  wa?: string
  tenant_id?: number
  /** Terakhir memakai aplikasi; dicatat paling sering tiap 5 menit. */
  last_seen?: string
}

export interface Me {
  user: User
  role_label: string
  views: View[]
  langganan: LanggananStatus
  garansi: GaransiStatus
  /** Hanya akun tim Master Pajak: role tim dan izin efektifnya. */
  master?: MasterMe
  /** Akun tim yang sedang "lihat sebagai klien": views berisi menu klien versi baca. */
  lihat?: { klien_id: number; nama: string; alasan: string; sampai: string }
  /** Usulan koreksi yang menunggu keputusan pemilik usaha. */
  koreksi_menunggu?: number
  /** Catatan konsultan yang belum ditandai selesai, dan yang jatuh tempo diingatkan (pop-up). */
  catatan_terbuka?: number
  catatan_ingat?: number
  /** Tenggat laporan & pajak yang sedang dalam jendela pengingat (diatur Panel Master). */
  jadwal?: PengingatJadwal[]
}

/** Status langganan usaha; 'habis' = hanya bisa melihat data dan membayar. */
export interface LanggananStatus { status: 'trial' | 'aktif' | 'habis'; sampai?: string }
export interface Paket {
  id: string; nama: string; harga: number; hari: number; hemat?: string
  /** starter | pro | premium; kosong = paket lama satu tingkat. */
  tingkat?: string
  /** Harga sebelum promo, ditampilkan dicoret. */
  harga_normal?: number
}
export interface BillingOrder { id: string; paket: string; jumlah: number; status: 'pending' | 'lunas' | 'gagal'; waktu: string }
/** Garansi 7 hari uang kembali: pembayaran pertama, gugur setelah ekspor data. */
export interface GaransiStatus { berlaku: boolean; sampai?: string; alasan?: string; order_id?: string }
export const GARANSI_BELUM_BAYAR = 'belum ada pembayaran'

export interface Billing {
  langganan: LanggananStatus
  garansi: GaransiStatus
  paket: Paket[]
  orders: BillingOrder[]
  client_key: string
  snap_js: string
  aktif: boolean
}
export const getPlans = () => api<{ paket: Paket[] }>('/billing/plans')
export const getBilling = () => api<Billing>('/billing')
export const refreshBilling = () => api<Billing>('/billing/refresh', { method: 'POST' })
export const klaimGaransi = (alasan: string) => api<Me>('/billing/garansi', { method: 'POST', body: { alasan } })
export const checkout = (paket: string) => api<{ order_id: string; token: string; redirect_url: string }>('/billing/checkout', { method: 'POST', body: { paket } })

export interface RoleInfo {
  label: string
  description: string
  views: View[]
}

export interface UserInput {
  name: string
  email: string
  role: Role
  active: boolean
  password: string
}

interface TrendKPI {
  value: number
  display: string
  badge_text: string
  sub_text: string
  sparkline: number[]
}

export interface MonthPoint {
  month: string
  omzet: number
  total_biaya: number
  laba_bersih: number
  ending_cash: number
}

export interface DailyOmzet {
  day: number
  omzet: number
  above_target: boolean
}

export interface Transaction {
  id: string
  date: string
  type: 'INCOME' | 'EXPENSE'
  title: string
  channel: string
  gross: number
  fee: number
  net: number
}

export interface CashWarning {
  tanggal: string
  saldo: number
  penyebab: number[] | null
}

export interface Dashboard {
  cash_warnings: CashWarning[] | null
  tenant: { name: string; initials: string; category: string; total_transactions: number; is_demo: boolean }
  period?: string
  period_label: string
  kpis: {
    omzet: TrendKPI
    laba_bersih: TrendKPI
    margin_laba: TrendKPI
    saldo_kas: TrendKPI
    target_omzet: { display: string; surplus_display: string }
    piutang: {
      unpaid_display: string
      invoice_count: number
      overdue_30d_display: string
      payable_unpaid_display: string
    }
  }
  monthly_trend_11m: MonthPoint[]
  daily_target: number
  daily_omzet: DailyOmzet[]
  total_products: number
  top_products: { name: string; qty: number; omzet: number; pct: number }[]
  total_app_fees_label: string
  channels: { name: string; gross: number; fee: number; net: number }[]
  recent_transactions: Transaction[]
  obligations: {
    invoice_no: string
    counterparty: string
    kind: 'RECEIVABLE' | 'PAYABLE'
    due_date: string
    days_overdue: number
    amount: number
  }[]
  inventory: { sku: string; name: string; type: string; stock: string; cogs: number; sell_price: number; status: string }[]
  tax_report: {
    yearly_exemption: number
    total_gross: number
    total_taxable: number
    total_tax_due: number
    months: {
      month_name: string
      gross_omzet: number
      cumulative_omzet: number
      taxable_amount: number
      tax_due: number
      status: string
    }[]
  }
}

export interface IncomeStatement {
  omzet: number
  channel_fees: number
  net_sales: number
  cogs: number
  gross_profit: number
  /** Beban operasional yang mengeluarkan kas (tanpa penyusutan). */
  opex: number
  depreciation: number
  amortization: number
  total_opex: number
  other_income: number
  net_profit: number
  margin_bps: number
  /** Laba kotor − beban operasional tunai (sebelum penyusutan & amortisasi). */
  ebitda: number
  /** Laba usaha = EBITDA − penyusutan − amortisasi. */
  ebit: number
  opex_lines: { label: string; amount: number }[] | null
  /** Rincian HPP dari kartu stok (setelah titik nol); tidak ada = HPP = belanja bahan. */
  hpp_stok?: {
    aktif: boolean; mulai: string; persediaan_awal: number; persediaan_akhir: number
    kapitalisasi: number; hpp_jual: number; selisih_opname: number; koreksi: number
  }
}

export interface CashFlowLine {
  label: string
  /** Bertanda: positif menambah kas, negatif mengurangi. */
  amount: number
  detail?: { label: string; amount: number }[] | null
  /** Baris informasi yang sengaja Rp 0 karena tidak menggerakkan kas. */
  non_kas?: boolean
}

export interface CashFlowSection {
  lines: CashFlowLine[]
  net: number
}

export interface CashFlow {
  /** Nilai persediaan akhir − awal dari kartu stok (0 sebelum titik nol). */
  kenaikan_persediaan?: number
  opening_balance: number
  cash_in: number
  cash_out: number
  net_cash: number
  closing_balance: number
  new_receivables: number
  new_payables: number
  new_payables_other: number
  new_receivables_other: number
  depreciation: number
  amortization: number
  setor_modal: number
  prive: number
  operasi_langsung: CashFlowSection
  operasi_tidak_langsung: CashFlowSection
  investasi: CashFlowSection
  pendanaan: CashFlowSection
}

export interface BalanceSheet {
  assets: {
    cash: number
    receivables: number
    inventory: number
    fixed_assets_cost: number
    accum_depreciation: number
    /** Nilai buku aset tetap & tak berwujud. */
    fixed_assets: number
    /** Aset lancar lain dari saldo awal (uang muka, dibayar di muka). */
    other_assets?: number
    total: number
  }
  liabilities: { payables: number; loans: number; other?: number; total: number }
  equity: { opening_capital: number; added_capital: number; retained_earnings: number; current_profit: number; total: number }
  is_balanced: boolean
}

export interface JenisInfo {
  nama: string
  arah: 'masuk' | 'keluar'
  pos: string
}

/** Transaksi bertanggal (sumber semua laporan). */
export interface Tx {
  id: number
  tanggal: string
  jenis: string
  nominal: number
  keterangan: string
  item: string
  qty: number
  channel: string
  pihak: string
  lunas: boolean
  tanggal_lunas: string
  akun?: string // kode CoA koreksi; kosong = otomatis dari jenis
  /** Dibuat otomatis dari invoice ini (Penjualan DPP / PPN Keluaran); diubah lewat menu Invoice. */
  invoice_id?: number
  jatuh_tempo?: string
  /** Barang dari Stok & Gudang (Penjualan / Beli Bahan / Stok) dan jumlahnya per seribu satuan. */
  stok_id?: number
  qty_stok?: number
  /** Bagian dari catatan jurnal multi-akun (halaman Catat transaksi); diubah lewat halaman itu. */
  grup?: number
  no_bukti?: string
  /** Asal pencatatan: dari impor mutasi bank atau dibaca AI dari foto nota. */
  sumber?: 'mutasi' | 'foto'
}

export type TxInput = Omit<Tx, 'id'>

// --- Catat transaksi multi-akun (baris jurnal debit / kredit) ---
export interface BarisCatat { akun: string; keterangan?: string; debit: number; kredit: number; jatuh_tempo?: string }
export interface CatatanJurnal { tanggal: string; pihak?: string; no_bukti?: string; keterangan?: string; baris: BarisCatat[]; sumber?: 'foto' }
export interface AkunCatat { kode: string; nama: string; kelompok: string; lawan?: 'rekening' | 'piutang' | 'utang' }
export interface OpsiCatat { akun: AkunCatat[]; bawaan: Record<string, string> }
export const getOpsiCatat = () => api<OpsiCatat>('/catat/opsi')
export const getCatatanJurnal = (grup: number) => api<CatatanJurnal & { grup: number }>(`/catat/${grup}`)
export const simpanCatatanJurnal = (c: CatatanJurnal, grup?: number) =>
  api<CatatanJurnal & { grup: number; transaksi: number[] }>(grup ? `/catat/${grup}` : '/catat', { method: grup ? 'PUT' : 'POST', body: c })
export const hapusCatatanJurnal = (grup: number) => api<void>(`/catat/${grup}`, { method: 'DELETE' })

export interface BreakEven {
  price: number
  variable_cost: number
  contribution_margin: number
  fixed_cost: number
  bep_units: number
  bep_units_per_day: number
  bep_revenue: number
  monthly_volume: number
  net_profit_at_volume: number
  target_net_profit: number
  units_for_target: number
}

export interface PriceSim {
  recommended_price: number
  channel_fee: number
  net_take_home: number
  break_even?: BreakEven
  break_even_error?: string
}

export interface PriceSimInput {
  cogs: number
  targetProfit: number
  /** > 0: harga dihitung dari margin yang diinginkan (persen), targetProfit diabaikan. */
  marginPct?: number
  feePct: number
  fixedCost?: number // kosong = beban operasional periode berjalan
  monthlyVolume: number
  targetNetProfit: number
}

export interface AppData {
  dash: Dashboard
  report?: IncomeStatement
  cash?: CashFlow
  balanceSheet?: BalanceSheet
  assets?: AssetSummary
}

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

/** Galat dari balasan gagal; halaman HTML proxy (mis. Cloudflare 520) tidak ditampilkan mentah. */
export async function galatDari(res: Response): Promise<ApiError> {
  return new ApiError(res.status, pesanGalat(res.status, await res.text().catch(() => ''), res.statusText))
}

async function api<T>(path: string, init?: { method?: string; body?: unknown }): Promise<T> {
  const res = await fetch('/api/v1' + path, {
    method: init?.method ?? 'GET',
    headers: init?.body === undefined ? undefined : { 'Content-Type': 'application/json' },
    body: init?.body === undefined ? undefined : JSON.stringify(init.body),
  }).catch(() => {
    throw new ApiError(0, 'Tidak bisa terhubung ke server. Periksa koneksi internet lalu coba lagi.')
  })
  if (!res.ok) throw await galatDari(res)
  if (res.status === 204) return undefined as T
  return (await res.json().catch(() => {
    throw new ApiError(res.status, 'Balasan server tidak terbaca. Coba muat ulang halaman.')
  })) as T
}

export interface PeriodOption {
  id: string
  label: string
  /** Bulan yang dibuka pertama kali: bulan terakhir yang punya transaksi. */
  bawaan?: boolean
}

export const getPeriods = () => api<PeriodOption[]>('/periods')

export const getMe = () => api<Me>('/auth/me')
export const login = (email: string, password: string) => api<Me>('/auth/login', { method: 'POST', body: { email, password } })
export interface DaftarInput { name: string; email: string; password: string; wa: string; setuju_pdp: boolean }
export const register = (body: DaftarInput) => api<Me>('/auth/register', { method: 'POST', body })
export const logout = () => api<void>('/auth/logout', { method: 'POST' })

/** Ambil dashboard + laporan yang boleh diakses role ini (laporan lain 403 untuk Kasir). */
export async function loadAll(views: View[], period = '2026-08'): Promise<AppData> {
  const lk = views.includes('laporan-keuangan')
  const q = `?period=${encodeURIComponent(period)}`
  const [dash, report, cash, balanceSheet, assets] = await Promise.all([
    api<Dashboard>('/dashboard' + q),
    lk ? api<IncomeStatement>('/reports/financial' + q) : undefined,
    lk ? api<CashFlow>('/reports/cashflow' + q) : undefined,
    lk ? api<BalanceSheet>('/reports/balance-sheet' + q) : undefined,
    lk ? getAssets(period) : undefined,
  ])
  return { dash, report, cash, balanceSheet, assets }
}

// --- Register aset & penyusutan (PMK 72/2023) ---

export type MetodePenyusutan = 'garis_lurus' | 'saldo_menurun'
export type MetodeNilaiStok = 'fifo' | 'rata_rata'

export interface KelompokAset {
  id: string
  nama: string
  /** Masa manfaat dalam bulan. */
  masa_manfaat: number
  tarif_gl_bps: number
  /** 0 = saldo menurun tidak boleh (bangunan). */
  tarif_sm_bps: number
  bangunan: boolean
}

export interface JenisHarta {
  id: string
  nama: string
  kelompok: string
  tak_berwujud: boolean
  /** Rujukan tempat memeriksa baris ini di Lampiran PMK 72/2023. */
  lampiran: string
  /** false = pemetaan ke kelompok belum dicek tim pajak. */
  terverifikasi: boolean
}

export interface AssetInput {
  nama: string
  jenis_harta: string
  kelompok: string
  metode: MetodePenyusutan
  tanggal_perolehan: string
  harga_perolehan: number
  tak_berwujud: boolean
  /** Diisi saat melengkapi draft, supaya aset tertaut ke transaksi pembeliannya. */
  transaksi_id?: number
  /** Cara aset dibayar bila tidak berasal dari transaksi: kas, utang (kredit/cicilan), atau
   *  modal (barang setoran pemilik). Kosong = belum dipilih. */
  sumber_dana?: SumberDanaAset
  /** Akumulasi penyusutan yang sudah ada per tanggal saldo awal (impor template saldo awal);
   *  dibawa apa adanya saat aset diedit supaya nilai buku awal tidak berubah. */
  akumulasi_awal?: number
  akumulasi_per?: string
}

export type SumberDanaAset = '' | 'kas' | 'utang' | 'modal'

export interface AssetYear {
  tahun: number
  bulan: number
  dasar_awal: number
  penyusutan: number
  sisa_akhir: number
}

export interface Asset extends AssetInput {
  id: number
  kelompok_nama: string
  masa_manfaat: number
  tarif_bps: number
  beban_bulan_ini: number
  akumulasi: number
  nilai_buku: number
  nilai_buku_awal: number
  /** Dibuat otomatis dari transaksi "Beli Alat / Aset", kelompok belum dilengkapi. */
  draft: boolean
  jadwal?: AssetYear[] | null
}

export interface AssetSummary {
  assets: Asset[] | null
  kelompok: KelompokAset[]
  jenis_harta: JenisHarta[]
  total_harga_perolehan: number
  total_akumulasi: number
  total_nilai_buku: number
  nilai_buku_awal: number
  beban_penyusutan: number
  beban_amortisasi: number
  beban_total: number
  tambahan_tanpa_kas: number
  draft_count: number
  /** Aset tanpa transaksi yang sumber dananya belum dipilih (sementara dianggap modal). */
  sumber_belum_dipilih?: number
  /** Aset sebelum tanggal ini termasuk saldo awal, jadi tidak perlu sumber dana. */
  tanggal_saldo_awal?: string
  /** Jumlah baris pemetaan jenis harta yang belum diverifikasi ke Lampiran PMK. */
  pemetaan_belum_dicek: number
  /** Metode penyusutan usaha; '' = belum dipilih (usaha lama, metode masih per aset). */
  metode_usaha: '' | MetodePenyusutan
  /** Saat belum dipilih: metode yang sudah seragam dipakai aset, atau campur. */
  metode_saran?: '' | MetodePenyusutan
  metode_campur?: boolean
}

export const getAssets = (period: string) => api<AssetSummary>(`/assets?period=${encodeURIComponent(period)}`)
export const createAsset = (a: AssetInput) => api<Asset>('/assets', { method: 'POST', body: a })
export const updateAsset = (id: number, a: AssetInput) => api<Asset>(`/assets/${id}`, { method: 'PUT', body: a })
export const deleteAsset = (id: number) => api<void>(`/assets/${id}`, { method: 'DELETE' })
/** Pilih metode penyusutan usaha sekali; setelah itu terkunci untuk semua aset. */
export const pilihMetodePenyusutan = (metode: MetodePenyusutan) =>
  api<BusinessProfile>('/business/metode-penyusutan', { method: 'POST', body: { metode } })

/** Metode nilai stok usaha: bawaan FIFO; terkunci setelah dipilih atau kartu stok dimulai. */
export interface MetodeStok { metode: MetodeNilaiStok; terkunci: boolean; pilihan: Option[]; pencatatan: 'periodik' | 'perpetual' }
export const getMetodeStok = () => api<MetodeStok>('/stok/metode')
export const pilihMetodeStok = (metode: MetodeNilaiStok) => api<MetodeStok>('/stok/metode', { method: 'POST', body: { metode } })

export const listTransactions = () => api<{ transactions: Tx[]; jenis: JenisInfo[]; can_edit: boolean }>('/transactions')
export type SavedTx = Tx & { warnings?: CashWarning[] }
export const createTransaction = (t: TxInput) => api<SavedTx>('/transactions', { method: 'POST', body: t })
export const updateTransaction = (id: number, t: TxInput) => api<SavedTx>(`/transactions/${id}`, { method: 'PUT', body: t })
export const deleteTransaction = (id: number) => api<void>(`/transactions/${id}`, { method: 'DELETE' })

export const simulatePrice = (period: string, i: PriceSimInput) =>
  api<PriceSim>(`/pricing/simulate?period=${period}`, {
    method: 'POST',
    body: {
      cogs: i.cogs,
      target_profit: i.targetProfit,
      fee_bps: Math.round(i.feePct * 100),
      margin_bps: i.marginPct ? Math.round(i.marginPct * 100) : 0,
      fixed_cost: i.fixedCost,
      monthly_volume: i.monthlyVolume,
      target_net_profit: i.targetNetProfit,
    },
  })

export const listUsers = () => api<{ users: User[]; roles: Record<Role, RoleInfo> }>('/users')
export const createUser = (u: UserInput) => api<User>('/users', { method: 'POST', body: u })
export const updateUser = (id: number, u: UserInput) => api<User>(`/users/${id}`, { method: 'PUT', body: u })
export const deleteUser = (id: number) => api<void>(`/users/${id}`, { method: 'DELETE' })

export interface AIChatResponse {
  answer: string
  category: string
  action_suggestions: string[]
  /** Tombol ke halaman terkait (panduan menu), sudah disaring sesuai hak akses. */
  tautan?: { view: View; label: string; tab?: string }[]
}

export const askAIChat = (query: string, period = '2026-08') =>
  api<AIChatResponse>('/ai/chat', { method: 'POST', body: { query, period } })


export interface ImportResult {
  valid: number
  errors: { baris: number; pesan: string }[] | null
  sample: Tx[] | null
  imported: number
}

/** Unggah .xlsx; commit=false hanya pratinjau (tidak menyimpan). */
export async function importTransactions(file: File, commit: boolean): Promise<ImportResult> {
  const form = new FormData()
  form.append('file', file)
  const res = await fetch('/api/v1/transactions/import' + (commit ? '?commit=1' : ''), { method: 'POST', body: form })
  if (!res.ok) throw await galatDari(res)
  return res.json()
}

/** Hasil AI membaca foto nota/struk: pratinjau isian transaksi, belum tersimpan. */
export interface SaranNota {
  tanggal: string; jenis: string; nominal: number; pihak?: string; keterangan?: string; lunas: boolean
  item?: { nama: string; qty?: string; harga?: number }[]
  alasan?: string; yakin: 'tinggi' | 'sedang' | 'rendah'; catatan: string[]; model?: string
}

export async function bacaNota(foto: Blob): Promise<SaranNota> {
  const form = new FormData()
  form.append('foto', foto, 'nota.jpg')
  const res = await fetch('/api/v1/transactions/foto', { method: 'POST', body: form })
  if (!res.ok) throw await galatDari(res)
  return res.json()
}

export const TEMPLATE_URL = '/api/v1/transactions/template'

export type Granularity = 'day' | 'week' | 'month'

export interface SalesGroup { name: string; omzet: number; count: number; piutang: number }

export interface SalesReport {
  granularity: Granularity
  summary: { omzet: number; transaksi: number; rata_rata: number; qty: number; piutang: number }
  trend: { label: string; omzet: number; count: number }[]
  top_items: { item: string; qty: number; omzet: number }[]
  by_channel: SalesGroup[]
  by_customer: SalesGroup[]
}

export const getSalesReport = (period: string, granularity: Granularity) =>
  api<SalesReport>(`/reports/sales?period=${period}&granularity=${granularity}`)

export type JenisUsaha = 'perdagangan' | 'jasa' | 'manufaktur'
export type BentukUsaha = 'op' | 'pt_perorangan' | 'cv' | 'pt'

export interface BusinessProfile {
  nama_usaha: string
  jenis_usaha: JenisUsaha
  bentuk_usaha: BentukUsaha
  ptkp: string
  pkp: boolean
  metode_persediaan: '' | 'periodik' | 'perpetual'
  /** Dipilih sekali untuk semua aset (bangunan tetap garis lurus), lalu terkunci. */
  metode_penyusutan?: MetodePenyusutan
  /** fifo | rata_rata: cara menilai stok & HPP; dikunci setelah dipilih. '' = jasa / belum dipilih. */
  metode_nilai_stok?: MetodeNilaiStok | ''
  /** Kelompok KLU 3 digit; pembanding NPM & CTTOR nasional. */
  klu?: string
  /** Perkiraan setahun dari onboarding; laba sebelum pajak boleh negatif (rugi). */
  omzet_tahunan?: number
  laba_sebelum_pajak?: number
  /** Modal yang ditanam pemilik; dasar ROI tahun pertama (laba ÷ modal). */
  modal_awal?: number
  /** Lokasi usaha (onboarding); alamat juga dicetak di invoice. */
  alamat?: string
  kecamatan?: string
  kota?: string
  provinsi?: string
  /** Identitas penerbit & rekening: opsional, tapi dicetak di invoice begitu diisi. */
  npwp?: string
  telepon?: string
  email?: string
  bank_nama?: string
  bank_rekening?: string
  bank_atas_nama?: string
  /** Tarif PPN dalam basis poin (1100 = 11%); hanya dipakai bila pkp. */
  ppn_bps?: number
  penandatangan_nama?: string
  penandatangan_jabatan?: string
}

export interface Option { value: string; label: string; hint?: string }
/** contoh: kegiatan (uraian KBLI 5 digit) di bawah kategori ini, supaya orang awam tidak salah pilih. */
export interface KLUOption { kode: string; nama: string; jenis: JenisUsaha; contoh?: string[] }
export interface SaranKLU { kode: string; nama: string; jenis: JenisUsaha; alasan?: string; contoh: string[] }
/** Rekomendasi bidang usaha dari cerita pemilik; ai=false berarti hasil pencarian kata. */
export const saranKLU = (deskripsi: string, jenis?: string) =>
  api<{ ai: boolean; model?: string; saran: SaranKLU[] }>('/business/klu-saran', { method: 'POST', body: { deskripsi, jenis: jenis ?? '' } })

export interface BusinessInfo {
  profile: BusinessProfile | null
  options: { jenis_usaha: Option[]; bentuk_usaha: Option[]; ptkp: string[]; metode_persediaan: Option[]; metode_penyusutan: Option[]; metode_nilai_stok: Option[]; provinsi: string[]; klu: KLUOption[] }
}

export const getBusiness = () => api<BusinessInfo>('/business')

export type Kota = { nama: string; kecamatan: string[] }
const wilayahCache = new Map<string, Promise<Kota[]>>()
// getWilayah: kabupaten/kota + kecamatan satu provinsi (Kepmendagri 2025); di-cache per provinsi.
export function getWilayah(provinsi: string): Promise<Kota[]> {
  let p = wilayahCache.get(provinsi)
  if (!p) {
    p = api<Kota[]>('/wilayah?provinsi=' + encodeURIComponent(provinsi))
    p.catch(() => wilayahCache.delete(provinsi)) // gagal: boleh dicoba lagi
    wilayahCache.set(provinsi, p)
  }
  return p
}
export const saveBusiness = (p: BusinessProfile) => api<BusinessProfile>('/business', { method: 'PUT', body: p })

/** Hasil tax.CekAwal: NPM vs benchmark nasional dan risiko pajak. */
export interface CekAwal {
  /** Rata-rata nasional untuk KLU usaha; null bila KLU belum dipilih. */
  klu: { kode: string; nama: string; cttor_bps: number; npm_bps: number } | null
  npm_bps: number
  cttor_bps: number
  status: 'kurang_data' | 'di_bawah' | 'wajar'
  pesan: string
  cttor_pesan?: string
  sumber: string
  risiko: { level: 'tinggi' | 'sedang' | 'info'; judul: string; isi: string }[]
  /** Kesimpulan sheet "Bentuk Usaha": skema pajak paling hemat + alasan. null bila omzet kosong. */
  saran: SaranBentuk | null
  /** ROI tahun pertama (laba ÷ modal); hanya ada bila modal diisi. */
  modal_awal?: number
  roi_bps?: number
  roi_pesan?: string
}
export interface SaranBentuk {
  termurah: string; sekarang: string; judul: string; asumsi?: string; alasan: string[]; catatan: string[]
  hemat: number; skema: TaxSkema[]; laba: number; ptkp: number
  /** hemat ÷ pajak skema sekarang, basis poin (8523 = 85,23% lebih kecil). */
  hemat_bps: number; omzet: number
}
export const cekAwal = (p: BusinessProfile) => api<CekAwal>('/business/cek-awal', { method: 'POST', body: p })

export interface BugReport {
  id: number; judul: string; deskripsi: string; halaman: string; user_email: string; waktu: string
  screenshot?: string
  /** true = sudah sampai ke webhook pengembang; false dengan gagal_kirim = baru tersimpan di server. */
  diteruskan: boolean
  gagal_kirim?: string
  tenant_id?: number
  /** Status tiket di Panel Master; kosong = baru. */
  status?: TiketStatus
  /** Balasan tim Master Pajak, terbaca pelapor. */
  balasan?: { nama: string; isi: string; waktu: string }[]
}

export type TiketStatus = 'baru' | 'dikerjakan' | 'selesai'

/** Ke mana laporan dikirim. */
export interface BugInfo {
  penerusan_aktif: boolean
  /** Host tujuan penerusan, tanpa token di path-nya. */
  penerusan_ke: string
}

export const listBugs = () => api<BugReport[]>('/bugs')
export const getBugInfo = () => api<BugInfo>('/bugs/info')
export const bugScreenshotUrl = (id: number) => `/api/v1/bugs/${id}/screenshot`
/** Kirim laporan sebagai multipart supaya screenshot (opsional) ikut terunggah. */
export async function createBug(b: { judul: string; deskripsi: string; halaman: string; screenshot: File | null }): Promise<BugReport> {
  const form = new FormData()
  form.append('judul', b.judul)
  form.append('deskripsi', b.deskripsi)
  form.append('halaman', b.halaman)
  if (b.screenshot) form.append('screenshot', b.screenshot)
  const res = await fetch('/api/v1/bugs', { method: 'POST', body: form })
  if (!res.ok) throw await galatDari(res)
  return res.json()
}

// --- Panel Master (akun tim Master Pajak) ---

export type Izin = 'klien_semua' | 'klien_lihat' | 'tiket' | 'lihat_sebagai' | 'usul_koreksi' | 'langganan' | 'coa_template' | 'benchmark'
  | 'parameter_pajak' | 'klien_ubah' | 'pengguna_lihat' | 'pengguna_kelola' | 'pengguna_reset' | 'hapus_permanen' | 'catatan' | 'pengaturan' | 'pengetahuan' | 'artikel' | 'artikel_terbit' | 'pembayaran' | 'tim' | 'audit'
export interface IzinInfo { id: Izin; label: string; keterangan?: string }
export interface MasterRole { id: string; nama: string; kunci: boolean; izin: Izin[] }
/** klien -1 = semua klien. */
export interface MasterMe { role: MasterRole; izin: Izin[]; klien: number }
export interface Orang { id: number; nama: string }

export interface KlienRow {
  id: number; nama: string; jenis?: string; kota?: string; klu?: string; pkp: boolean; demo?: boolean
  owner_nama?: string; owner_email?: string; owner_wa?: string; user: number
  status: LanggananStatus['status']; sampai?: string; paket?: string
  terakhir_aktif?: string; transaksi_bulan_ini: number; onboarding: boolean; daftar?: string
  penangan: Orang[]; perlu_perhatian: boolean; alasan?: string
  /** Diarsipkan (soft delete): user usaha ini tidak bisa masuk. */
  arsip?: string; arsip_alasan?: string
}
export interface IdentitasUsaha { nama_usaha: string; alamat?: string; kecamatan?: string; kota?: string; provinsi?: string; npwp?: string; telepon?: string; email?: string }
export interface PenggunaKlien extends User {
  role_label: string; usaha: string; kota?: string; alamat?: string; telepon?: string; arsip: boolean; wa?: string; last_seen?: string
  klien_id: number; langganan: { status: 'trial' | 'aktif' | 'habis'; sampai?: string }; demo?: boolean
}
export const ubahKlien = (id: number, b: IdentitasUsaha) => api<BusinessProfile>(`/master/klien/${id}`, { method: 'PUT', body: b })
export const arsipKlien = (id: number, alasan: string) => api<void>(`/master/klien/${id}/arsip`, { method: 'POST', body: { alasan } })
export const pulihkanKlien = (id: number) => api<void>(`/master/klien/${id}/arsip`, { method: 'DELETE' })
export const hapusKlien = (id: number, konfirmasi: string) => api<void>(`/master/klien/${id}`, { method: 'DELETE', body: { konfirmasi } })
export const getPenggunaKlien = () => api<PenggunaKlien[]>('/master/pengguna')
export const ubahPenggunaKlien = (id: number, b: { name: string; email: string; wa: string; aktif: boolean }) => api<User>(`/master/pengguna/${id}`, { method: 'PUT', body: b })
export const resetPasswordKlien = (id: number) => api<{ password: string }>(`/master/pengguna/${id}/reset-password`, { method: 'POST' })
export const hapusPenggunaKlien = (id: number, konfirmasi: string) => api<void>(`/master/pengguna/${id}`, { method: 'DELETE', body: { konfirmasi } })
export const hapusAnggotaTim = (id: number, konfirmasi: string) => api<void>(`/master/tim/${id}`, { method: 'DELETE', body: { konfirmasi } })
export interface RingkasanMaster {
  klien: number; aktif: number; trial: number; habis: number; mrr?: number; tiket_terbuka: number
  perlu_perhatian: KlienRow[]; belum_ditugaskan?: KlienRow[]; terakhir_aktif: KlienRow[]
}
export interface TiketRow extends BugReport { tenant_id: number; status: TiketStatus; nomor: string; klien_nama: string }
export interface MasterOrder { id: string; paket: string; jumlah: number; status: 'pending' | 'lunas' | 'gagal' | 'manual'; waktu: string; dibayar_pada?: string }
export interface KlienDetail { klien: KlienRow; users: User[]; orders: MasterOrder[]; tiket: TiketRow[]; koreksi: KoreksiRow[]; identitas: IdentitasUsaha | null }
export interface PembayaranRow { klien_id: number; klien_nama: string; order: MasterOrder; garansi?: GaransiStatus }
export interface AnggotaRow { user: User; role: string; klien: number[] }
export interface TimData { anggota: AnggotaRow[]; roles: MasterRole[]; izin: IzinInfo[]; klien: Orang[] }
export interface AuditRow { id: number; waktu: string; user_id: number; nama: string; peran: string; aksi: string; klien_id?: number; keterangan?: string }
// --- Catatan konsultan ---

export type TingkatCatatan = 'info' | 'perbaiki' | 'penting'
export interface CatatanKonsultan {
  id: number; halaman: string; periode?: string; tingkat: TingkatCatatan; isi: string; transaksi_id?: number
  oleh_id: number; oleh: string; peran?: string; dibuat: string; status: 'terbuka' | 'selesai'
  selesai_oleh?: string; selesai_pada?: string; diingatkan_pada?: string; jumlah_pengingat: number
}
export interface Pengingat { aktif: boolean; interval_hari: number; tingkat_min: TingkatCatatan; kanal: string[] }
export interface CatatanMasterRow extends CatatanKonsultan { klien_id: number; klien_nama: string }
export const tulisCatatan = (klienId: number, b: { halaman: string; periode: string; tingkat: TingkatCatatan; isi: string }) =>
  api<CatatanKonsultan>(`/master/klien/${klienId}/catatan`, { method: 'POST', body: b })
export const getCatatanMaster = () => api<{ catatan: CatatanMasterRow[]; pengingat: Pengingat }>('/master/catatan')
export const aturPengingat = (p: Pengingat) => api<Pengingat>('/master/pengingat', { method: 'PUT', body: p })
// --- Jadwal pengingat laporan & pajak (Panel Master, berlaku untuk semua klien) ---

export type UntukJadwal = 'semua' | 'op' | 'badan' | 'pkp'
export interface JadwalPengingat {
  id: number; judul: string; keterangan?: string
  frekuensi: 'bulanan' | 'tahunan'; tanggal: number; bulan?: number
  hari_sebelum: number; untuk: UntukJadwal; aktif: boolean; perlu_verifikasi?: boolean
}
export interface PengingatJadwal { id: number; judul: string; keterangan?: string; jatuh_tempo: string; sisa_hari: number }
export const getJadwalMaster = () => api<{ item: JadwalPengingat[] }>('/master/jadwal')
export const aturJadwal = (item: JadwalPengingat[]) => api<{ item: JadwalPengingat[] }>('/master/jadwal', { method: 'PUT', body: { item } })
export const getCatatan = () => api<{ catatan: CatatanKonsultan[]; pengingat: number[]; aturan: Pengingat }>('/catatan')
export const selesaiCatatan = (id: number) => api<CatatanKonsultan>(`/catatan/${id}/selesai`, { method: 'POST' })
export const catatanDiingatkan = (ids: number[]) => api<void>('/catatan/diingatkan', { method: 'POST', body: { ids } })

/** Untuk pemilik usaha: semua catatan tim Master Pajak atas usahanya (lihat-sebagai, koreksi, perubahan data). */
export const getRiwayatAksesTim = () => api<AuditRow[]>('/akses-tim')

export const getRingkasanMaster = () => api<RingkasanMaster>('/master/ringkasan')
export const listKlien = () => api<KlienRow[]>('/master/klien')
export const getKlien = (id: number) => api<KlienDetail>(`/master/klien/${id}`)
export const aturLangganan = (id: number, body: { aksi: 'trial' | 'aktifkan'; hari?: number; paket?: string; alasan: string }) =>
  api<LanggananStatus>(`/master/klien/${id}/langganan`, { method: 'POST', body })
export const listTiket = () => api<TiketRow[]>('/master/tiket')
export const ubahTiket = (id: number, body: { status?: TiketStatus; balasan?: string }) => api<BugReport>(`/master/tiket/${id}`, { method: 'PUT', body })
export const tiketScreenshotUrl = (id: number) => `/api/v1/master/tiket/${id}/screenshot`
export const listPembayaran = () => api<PembayaranRow[]>('/master/pembayaran')
export const getTim = () => api<TimData>('/master/tim')
export const tambahAnggota = (body: { nama: string; email: string; password: string; role: string; klien: number[] }) =>
  api<AnggotaRow>('/master/tim', { method: 'POST', body })
export const ubahAnggota = (id: number, body: { role: string; klien: number[]; aktif: boolean }) =>
  api<AnggotaRow>(`/master/tim/${id}`, { method: 'PUT', body })
export const simpanRole = (r: { id?: string; nama: string; izin: Izin[] }) =>
  r.id ? api<MasterRole>(`/master/roles/${r.id}`, { method: 'PUT', body: r }) : api<MasterRole>('/master/roles', { method: 'POST', body: r })
export const hapusRole = (id: string) => api<void>(`/master/roles/${id}`, { method: 'DELETE' })
export const listAudit = () => api<AuditRow[]>('/master/audit')
export const mulaiLihat = (klien: number, alasan: string) => api<Me['lihat']>(`/master/klien/${klien}/lihat`, { method: 'POST', body: { alasan } })
export const selesaiLihat = () => api<void>('/master/lihat', { method: 'DELETE' })

// --- Usulan koreksi ---

export type KoreksiStatus = 'menunggu' | 'disetujui' | 'ditolak' | 'kedaluwarsa'
export interface KoreksiRow {
  id: number; nomor: string; transaksi_id: number; jenis: 'ubah' | 'hapus'
  sebelum: Tx; usulan?: Tx
  alasan: string; oleh: string; waktu: string; status: KoreksiStatus
  catatan?: string; diputuskan_oleh?: string; diputuskan_pada?: string
  klien_id: number; klien_nama: string
  /** Laba bersih per bulan sebelum/sesudah, dihitung server. */
  dampak: { periode: string; laba_sebelum: number; laba_sesudah: number }[]
}
export const usulKoreksi = (klien: number, body: { transaksi_id: number; jenis: 'ubah' | 'hapus'; usulan?: TxInput; alasan: string }) =>
  api<KoreksiRow>(`/master/klien/${klien}/koreksi`, { method: 'POST', body })
export const listKoreksiMaster = () => api<KoreksiRow[]>('/master/koreksi')
export const listKoreksi = () => api<KoreksiRow[]>('/koreksi')
export const putuskanKoreksi = (id: number, setuju: boolean, catatan: string) =>
  api<KoreksiRow>(`/koreksi/${id}/${setuju ? 'setujui' : 'tolak'}`, { method: 'POST', body: { catatan } })

export interface TaxSkema { id: string; nama: string; keterangan: string; pajak: number; lebih_mahal_bps: number }
export interface TaxEntitas { id: string; nama: string; pph: number; dividen: number; total: number; persen_bps: number }

export interface TaxPlanning {
  input: { omzet: number; net_margin_bps: number; ptkp: string }
  bentuk_usaha: { omzet: number; laba: number; ptkp: number; skema: TaxSkema[]; termurah: string }
  entitas: TaxEntitas[]
}

export const getTaxPlanning = (i: { omzet?: number; net_margin_bps?: number; ptkp?: string } = {}) =>
  api<TaxPlanning>('/tax/planning', { method: 'POST', body: i })

export interface Account {
  kode: string
  nama: string
  subtype: string
  normal: 'debit' | 'kredit'
  kelompok: string
  kondisi: 'selalu' | 'jika_ada' | 'jika_pt' | 'jika_non_pt' | 'jika_perpetual' | 'jika_periodik'
  catatan?: string
  opsional?: boolean
}

export interface CoAInfo {
  supported: boolean
  accounts: Account[] | null
  mapping: { jenis: string; kode: string; nama: string }[] | null
  kelompok: string[]
  /** Versi template COA yang dipakai usaha ini. */
  template?: { id: number; jenis: string; versi: number }
  /** Ada versi template lebih baru dari tim Master Pajak; berlaku setelah Owner menerapkannya. */
  pembaruan?: { id: number; versi: number; diff: DiffAkun } | null
}
export interface DiffAkun { tambah: Account[]; hapus: Account[]; ubah: Account[] }

export const getCoA = () => api<CoAInfo>('/coa')
export const perbaruiCoA = () => api<unknown>('/coa/perbarui', { method: 'POST' })

// --- Katalog Panel Master: template COA & benchmark NPM nasional ---

export type StatusKatalog = 'draft' | 'terbit' | 'aktif' | 'arsip'
export interface TemplateCOA { id: number; jenis: string; versi: number; status: StatusKatalog; akun: Account[]; catatan?: string; oleh?: string; waktu: string }
export interface KLUBenchmark { kode: string; nama: string; cttor_bps: number; npm_bps: number }
export interface BenchmarkVersi { id: number; nama: string; sumber: string; status: StatusKatalog; baris: KLUBenchmark[]; oleh?: string; waktu: string }
export interface PeranAkun { id: string; label: string; bawaan: string }
export interface KatalogCOA { jenis: string[]; template: TemplateCOA[]; peran_wajib: PeranAkun[]; klien_per_template: Record<number, number> }

async function unggah<T>(path: string, file: File, fields: Record<string, string>): Promise<T> {
  const form = new FormData()
  form.append('file', file)
  for (const [k, v] of Object.entries(fields)) form.append(k, v)
  const res = await fetch('/api/v1' + path, { method: 'POST', body: form })
  if (!res.ok) throw await galatDari(res)
  return res.json()
}

export const getKatalogCOA = () => api<KatalogCOA>('/master/coa')
export const simpanDraftCOA = (jenis: string, akun: Account[], catatan: string) =>
  api<TemplateCOA>('/master/coa/draft', { method: 'POST', body: { jenis, akun, catatan } })
export const imporDraftCOA = (file: File, jenis: string, catatan: string) => unggah<TemplateCOA>('/master/coa/draft', file, { jenis, catatan })
export const hapusDraftCOA = (id: number) => api<void>(`/master/coa/${id}`, { method: 'DELETE' })
export const terbitkanCOA = (id: number) => api<TemplateCOA>(`/master/coa/${id}/terbitkan`, { method: 'POST' })
export const unduhCOAUrl = (id: number) => `/api/v1/master/coa/${id}/xlsx`
export const getKatalogBenchmark = () => api<{ versi: BenchmarkVersi[]; klien_per_klu: Record<string, number> }>('/master/benchmark')
export const imporBenchmark = (file: File, nama: string, sumber: string) => unggah<BenchmarkVersi>('/master/benchmark', file, { nama, sumber })
export const aktifkanBenchmark = (id: number) => api<BenchmarkVersi>(`/master/benchmark/${id}/aktifkan`, { method: 'POST' })
export const unduhBenchmarkUrl = (id: number) => `/api/v1/master/benchmark/${id}/xlsx`

export interface ReportLine { kode: string; nama: string; jumlah: number }
export interface Balance extends Account { debit: number; kredit: number; saldo: number }

export interface CoAReport {
  supported: boolean
  trial_balance: Balance[] | null
  income_statement?: {
    pendapatan: ReportLine[] | null
    penjualan_bersih: number
    metode: 'periodik' | 'perpetual'
    persediaan_awal: number
    hpp_lines: ReportLine[] | null
    persediaan_akhir: number
    hpp: number
    laba_kotor: number
    beban_operasional: ReportLine[] | null
    total_beban_operasional: number
    laba_usaha: number
    ebitda: number
    pendapatan_lain: ReportLine[] | null
    beban_lain: ReportLine[] | null
    laba_bersih: number
    catatan: string[] | null
  }
  balance_sheet?: {
    aset: { kelompok: string; lines: ReportLine[] | null; total: number }[]
    liabilitas: { kelompok: string; lines: ReportLine[] | null; total: number }[]
    ekuitas: { kelompok: string; lines: ReportLine[] | null; total: number }
    total_aset: number
    total_liabilitas: number
    total_ekuitas: number
    seimbang: boolean
  }
}

export const getCoAReport = (period: string) => api<CoAReport>(`/reports/coa?period=${period}`)
export const coaXlsxUrl = (period: string) => `/api/v1/reports/coa.xlsx?period=${period}`

// --- Onboarding: neraca awal, impor data awal, karyawan ---

export interface OpeningBalance {
  tanggal: string
  kas: number
  piutang: number
  persediaan: number
  utang: number
  pinjaman: number
  /** Modal disetor pemilik; untuk saldo awal per akun wajib membuat Harta = Kewajiban + Modal. */
  modal_disetor: number
  laba_ditahan?: number
  aset_lain?: number
  kewajiban_lain?: number
  aset_tetap?: number
  /** Saldo awal per akun; bila ada, pos-pos di atas diturunkan dari sini oleh server. */
  rincian?: SaldoAkun[]
}

export interface SaldoAkun { kode?: string; nama: string; kategori: string; nominal: number; keterangan?: string }
export interface KategoriSaldo { id: string; label: string; sisi: 'harta' | 'kewajiban' | 'modal' }

export interface OpeningInfo {
  opening: OpeningBalance | null
  modal_awal: number
  /** Bukan 0 = isian neraca awal tidak seimbang, perlu diperiksa. */
  selisih_modal: number
  /** false = laporan masih memakai saldo awal data contoh. */
  aktif: boolean
  /** Tanggal transaksi paling awal; tanggal saldo awal tidak boleh sesudahnya. */
  transaksi_pertama?: string
  /** Nilai buku register aset per tanggal saldo awal (atau ?tanggal=). */
  nilai_buku_aset: number
  kategori: KategoriSaldo[]
  akun_template: SaldoAkun[]
  /** Baris aset tetap per akun CoA dari daftar aset (nilai buku); tidak diketik bebas. */
  aset_tetap: SaldoAkun[]
  jumlah_aset: number
  /** Nilai persediaan yang seharusnya di saldo awal (daftar barang / hitung fisik pertama). */
  persediaan: number
  label_persediaan: string
  /** Alasan saldo awal tidak bisa diubah lagi (ada bulan yang sudah tutup buku). */
  terkunci?: string
}

/** Saran kategori & kode untuk nama akun yang diketik di Saldo Awal. */
export interface SaranSaldo {
  kategori: string; kode: string; akun?: string; baru: boolean
  sumber: 'coa' | 'ai' | 'aturan' | 'pilihan' | ''; alasan?: string
}
export const saranSaldo = (body: { nama: string; kategori?: string; dipakai: string[] }) =>
  api<SaranSaldo>('/saldo-awal/saran', { method: 'POST', body })

export interface SheetError {
  sheet: string
  /** 0 = masalah header, bukan baris data. */
  baris: number
  pesan: string
}

export interface OnboardingResult {
  opening: OpeningBalance | null
  modal_awal: number
  selisih_modal: number
  assets: Asset[] | null
  karyawan: Karyawan[] | null
  transaksi_count: number
  transaksi_sample: Tx[] | null
  errors: SheetError[] | null
  terbaca: string[] | null
  imported: boolean
  replaced_assets: number
  replaced_karyawan: number
  /** Barang dari sheet Persediaan; mengganti daftar Stok & Gudang saat diimpor. */
  stok?: { nama: string }[] | null
  replaced_stok?: number
  /** Nilai buku aset tetap per tanggal mulai, pembanding baris Aset Tetap. */
  nilai_buku_aset?: number
  /** false: saldo awal belum seimbang / ada transaksi sebelum tanggal mulai. */
  bisa_disimpan?: boolean
}

export interface Karyawan {
  id: number
  nama: string
  jabatan: string
  tanggal_masuk: string
  gaji_pokok: number
  tunjangan: number
  ptkp: string
  npwp: string
  aktif: boolean
}

export type KaryawanInput = Omit<Karyawan, 'id'>

export interface KaryawanSummary {
  karyawan: Karyawan[] | null
  ptkp_codes: string[]
  total_aktif: number
  beban_gaji_bulanan: number
}

export const ONBOARDING_TEMPLATE_URL = '/api/v1/onboarding/template'

/** commit=false hanya pratinjau (tidak menyimpan apa pun). */
export async function importOnboarding(file: File, commit: boolean): Promise<OnboardingResult> {
  const form = new FormData()
  form.append('file', file)
  const res = await fetch('/api/v1/onboarding/import' + (commit ? '?commit=1' : ''), { method: 'POST', body: form })
  if (!res.ok) throw await galatDari(res)
  return res.json()
}

export interface BarisBukuBesar { tanggal: string; tx_id?: number; keterangan: string; debit: number; kredit: number; saldo: number }
export interface AkunBukuBesar {
  kode: string; nama: string; kelompok: string; normal: 'debit' | 'kredit'
  saldo_awal: number; debit: number; kredit: number; saldo_akhir: number; baris: BarisBukuBesar[]
}
export interface BukuBesar {
  supported: boolean; period: string; period_label: string; dari: string; sampai: string
  akun: AkunBukuBesar[]; total_debit: number; total_kredit: number; seimbang: boolean
  pilihan: { kode: string; nama: string }[]
}
/** Rentang tanggal (YYYY-MM-DD) dan opsional satu akun (riwayat). */
export interface FilterBukuBesar { dari: string; sampai: string; akun?: string }
const qBukuBesar = (f: FilterBukuBesar) => new URLSearchParams({ dari: f.dari, sampai: f.sampai, ...(f.akun ? { akun: f.akun } : {}) }).toString()
export const getBukuBesar = (f: FilterBukuBesar) => api<BukuBesar>(`/reports/buku-besar?${qBukuBesar(f)}`)
export const bukuBesarXlsxUrl = (f: FilterBukuBesar) => `/api/v1/reports/buku-besar.xlsx?${qBukuBesar(f)}`

export interface BarisJurnalUmum { kode: string; nama: string; debit: number; kredit: number }
/** bukti: SA saldo awal, TRX-id transaksi, BYR-id pelunasan, JU-yyyymm-n entri sistem (penyusutan, penyesuaian). */
export interface EntriJurnalUmum { bukti: string; tanggal: string; tx_id?: number; keterangan: string; baris: BarisJurnalUmum[]; debit: number; kredit: number }
export interface JurnalUmum {
  supported: boolean; period_label: string; dari: string; sampai: string
  entri: EntriJurnalUmum[]; total_debit: number; total_kredit: number; seimbang: boolean
  pilihan: { kode: string; nama: string }[]
}
export interface FilterJurnalUmum { dari: string; sampai: string; akun?: string; q?: string }
const qJurnal = (f: FilterJurnalUmum) => new URLSearchParams({ dari: f.dari, sampai: f.sampai, ...(f.akun ? { akun: f.akun } : {}), ...(f.q ? { q: f.q } : {}) }).toString()
export const getJurnalUmum = (f: FilterJurnalUmum) => api<JurnalUmum>(`/reports/jurnal?${qJurnal(f)}`)
export const jurnalUmumXlsxUrl = (f: FilterJurnalUmum) => `/api/v1/reports/jurnal.xlsx?${qJurnal(f)}`

export interface FilterTransaksi { dari: string; sampai: string; jenis?: string; arah?: string; status?: string; channel?: string; cari?: string }
export interface LaporanTransaksi {
  dari: string; sampai: string; transaksi: Tx[]; jumlah: number
  total_masuk: number; total_keluar: number; belum_lunas: number
  per_jenis: { jenis: string; arah: string; jumlah: number; total: number }[]
  channel: string[] | null; jenis_pilihan: { nama: string; arah: string }[]
}
const qTransaksi = (f: FilterTransaksi) => new URLSearchParams(Object.entries(f).filter(([, v]) => v) as [string, string][]).toString()
export const getLaporanTransaksi = (f: FilterTransaksi) => api<LaporanTransaksi>(`/reports/transaksi?${qTransaksi(f)}`)
export const laporanTransaksiXlsxUrl = (f: FilterTransaksi) => `/api/v1/reports/transaksi.xlsx?${qTransaksi(f)}`

/** tanggal: tanggal pembanding nilai buku aset (saat tanggal mulai di form diganti). */
export const getOpening = (tanggal?: string) => api<OpeningInfo>('/opening' + (tanggal ? `?tanggal=${encodeURIComponent(tanggal)}` : ''))
/** paksa: simpan walau belum seimbang; selisihnya masuk akun Laba Ditahan - Selisih Saldo Awal. */
export const saveOpening = (o: OpeningBalance, paksa = false) => api<OpeningInfo>('/opening' + (paksa ? '?paksa=1' : ''), { method: 'PUT', body: o })

/** Saldo akun yang berlawanan dengan saldo normalnya (PSAK) dan stok minus, per periode. */
export interface SaldoMinus { jenis: 'akun' | 'stok'; kode?: string; nama: string; normal?: 'debit' | 'kredit'; nilai: number; satuan?: string; pesan: string }
export const getSaldoMinus = (period: string) => api<SaldoMinus[]>(`/saldo-minus?period=${encodeURIComponent(period)}`)

export const getKaryawan = () => api<KaryawanSummary>('/karyawan')
export const createKaryawan = (k: KaryawanInput) => api<Karyawan>('/karyawan', { method: 'POST', body: k })
export const updateKaryawan = (id: number, k: KaryawanInput) => api<Karyawan>(`/karyawan/${id}`, { method: 'PUT', body: k })
export const deleteKaryawan = (id: number) => api<void>(`/karyawan/${id}`, { method: 'DELETE' })

// --- Piutang & Utang, Stok & Gudang: data usaha yang bisa diimpor lalu disunting ---

export type JenisTagihan = 'RECEIVABLE' | 'PAYABLE'

export interface Tagihan {
  id: number
  nomor: string
  pihak: string
  jenis: JenisTagihan
  /** YYYY-MM-DD */
  jatuh_tempo: string
  nominal: number
  lunas: boolean
  keterangan?: string
}

/** hari_terlambat dihitung server saat dibaca, bukan disimpan — supaya tidak basi. */
export interface TagihanRow extends Tagihan {
  hari_terlambat: number
}

export type TagihanInput = Omit<Tagihan, 'id'>

export interface TagihanSummary {
  items: TagihanRow[] | null
  total_piutang: number
  total_utang: number
  piutang_terlambat: number
  jumlah_terlambat: number
}

export interface StokItem {
  id: number
  sku: string
  nama: string
  kategori: string
  jumlah: number
  satuan: string
  stok_min: number
  harga_pokok: number
  harga_jual: number
}

export type StokInput = Omit<StokItem, 'id'>

/** Turunan (status, margin, nilai) datang dari server supaya rumusnya satu tempat. */
export interface StokRow extends StokItem {
  status: 'Habis' | 'Stok Menipis' | 'Aman' | 'Minus'
  margin_bps: number
  nilai: number
  /** Jumlah per seribu satuan (2500 = 2,5). dilacak: dari kartu stok, bukan diketik. */
  jumlah_mili: number
  dilacak: boolean
  minus: boolean
}

export interface StokSummary {
  items: StokRow[] | null
  total_item: number
  nilai_persediaan: number
  menipis: number
  habis: number
  /** Titik nol kartu stok (YYYY-MM-DD); kosong = belum dimulai. */
  mulai: string
  minus: number
}

// --- Kartu stok ---

export interface BarisGerak { stok_id: number; qty?: number; fisik?: number; harga_unit?: number }
export interface GerakStok {
  id: number; tanggal: string; jenis: 'awal' | 'opname' | 'produksi'; baris: BarisGerak[]
  produk_id?: number; batch?: number; keterangan?: string; oleh?: string; dibuat: string
}
export type JenisKartu = 'awal' | 'beli' | 'produksi_masuk' | 'produksi_pakai' | 'jual' | 'opname'
export interface BarisKartu {
  tanggal: string; stok_id: number; jenis: JenisKartu; transaksi_id?: number; gerak_id?: number
  channel?: string; keterangan?: string; qty: number; nilai: number; koreksi?: number
  saldo_qty: number; saldo_nilai: number; minus?: boolean
}
export interface KartuBarang { barang: StokItem; mulai: string; awal: { qty: number; nilai: number }; baris: BarisKartu[] }
export interface MutasiItem {
  stok_id: number; sku: string; nama: string; kategori: string; satuan: string
  awal_qty: number; awal_nilai: number; beli_qty: number; beli_nilai: number; produksi_qty: number; produksi_nilai: number
  jual_qty: number; jual_nilai: number; pakai_qty: number; pakai_nilai: number; opname_qty: number; opname_nilai: number
  koreksi_nilai: number; akhir_qty: number; akhir_nilai: number; minus: boolean
}
export interface LaporanMutasi { dari: string; sampai: string; mulai: string; items: MutasiItem[]; total: MutasiItem; hpp_jual: number; selisih_opname: number }
export interface PilihanBarang { id: number; sku: string; nama: string; kategori: string; satuan: string; jumlah_mili: number; harga_jual: number }

export const stokAwal = (b: { tanggal: string; keterangan?: string; baris: BarisGerak[] }) => api<GerakStok>('/stok/awal', { method: 'POST', body: b })
export const stokOpname = (b: { tanggal: string; keterangan?: string; baris: BarisGerak[] }) => api<GerakStok>('/stok/opname', { method: 'POST', body: b })
/** Catat produksi (halaman penuh): beberapa produk, bahan aktual, dan biaya lain dari Kas/Bank.
 *  Kartu stok yang belum dimulai otomatis dimulai dari daftar barang. Jumlah per seribu satuan. */
export interface ProduksiCatat {
  tanggal: string; bayar_dari?: string
  produk: { produk_id: number; hasil: number; bahan: { stok_id: number; qty: number }[]; biaya: { keterangan: string; nominal: number }[]; keterangan?: string }[]
}
export const catatProduksi = (b: ProduksiCatat) => api<{ gerak: GerakStok[]; stok_awal: boolean; transaksi: number }>('/stok/produksi/catat', { method: 'POST', body: b })
export const stokProduksi = (b: { tanggal: string; produk_id: number; batch: number; keterangan?: string; bahan?: { stok_id: number; qty: number }[]; hasil?: number }) => api<GerakStok>('/stok/produksi', { method: 'POST', body: b })
export const getGerakStok = () => api<GerakStok[]>('/stok/gerak')
export const hapusGerakStok = (id: number) => api<void>(`/stok/gerak/${id}`, { method: 'DELETE' })
export const getKartuBarang = (id: number, dari: string, sampai: string) => api<KartuBarang>(`/stok/kartu/${id}?dari=${dari}&sampai=${sampai}`)
export const getMutasiStok = (dari: string, sampai: string) => api<LaporanMutasi>(`/stok/mutasi?dari=${dari}&sampai=${sampai}`)
export const mutasiStokXlsxUrl = (dari: string, sampai: string) => `/api/v1/stok/mutasi.xlsx?dari=${dari}&sampai=${sampai}`
export const getPilihanBarang = () => api<{ mulai: string; barang: PilihanBarang[] }>('/stok/pilihan')

export const getTagihan = () => api<TagihanSummary>('/tagihan')
export const updateTagihan = (id: number, t: TagihanInput) => api<Tagihan>(`/tagihan/${id}`, { method: 'PUT', body: t })
export const deleteTagihan = (id: number) => api<void>(`/tagihan/${id}`, { method: 'DELETE' })

export const getStok = () => api<StokSummary>('/stok')
export const createStok = (s: StokInput) => api<StokItem>('/stok', { method: 'POST', body: s })
export const updateStok = (id: number, s: StokInput) => api<StokItem>(`/stok/${id}`, { method: 'PUT', body: s })
export const deleteStok = (id: number) => api<void>(`/stok/${id}`, { method: 'DELETE' })

export const STOK_TEMPLATE_URL = '/api/v1/stok/template'
export const INVOICE_TEMPLATE_URL = '/api/v1/invoice/template'

/** 'tambah' menambah ke daftar; 'ganti' mengosongkan daftar lama lebih dulu. */
export type ImportMode = 'tambah' | 'ganti'

export interface MenuImportResult {
  /** nama sheet yang terbaca; kosong bila tidak ada yang dikenali */
  sheet: string
  valid: number
  errors: SheetError[] | null
  imported: boolean
  /** baris lama yang ditimpa (mode 'ganti') */
  diganti: number
  tagihan?: Tagihan[] | null
  stok?: StokItem[] | null
  invoice?: Invoice[] | null
}

/** commit=false hanya pratinjau (tidak menyimpan apa pun). */
async function importMenu(path: string, file: File, commit: boolean, mode: ImportMode): Promise<MenuImportResult> {
  const form = new FormData()
  form.append('file', file)
  const q = commit ? `?commit=1&mode=${mode}` : ''
  const res = await fetch(`/api/v1${path}/import${q}`, { method: 'POST', body: form })
  if (!res.ok) throw await galatDari(res)
  return res.json()
}

export const importStok = (f: File, commit: boolean, mode: ImportMode = 'tambah') => importMenu('/stok', f, commit, mode)
/** Invoice selalu ditambahkan (tidak ada mode ganti: invoice lama sudah dibukukan). */
export const importInvoice = (f: File, commit: boolean) => importMenu('/invoice', f, commit, 'tambah')

// --- Impor mutasi bank: tebak jenis transaksi, periksa, baru simpan ---

/** Kesalahan per baris berkas impor. baris 0 = masalah berkas, bukan baris data. */
export interface ImportError {
  baris: number
  pesan: string
}

/** Tingkat keyakinan tebakan jenis. 'rendah' = hanya arah uangnya yang pasti. */
/** ai = baris yang tidak dikenali kata kunci, jenisnya disarankan AI (tetap periksa). */
export type Keyakinan = 'tinggi' | 'sedang' | 'rendah' | 'ai'

export interface MutasiRow {
  /** Nomor baris di berkas aslinya, supaya baris yang keliru bisa ditunjuk. */
  baris: number
  tanggal: string
  keterangan: string
  nominal: number
  arah: 'masuk' | 'keluar'
  jenis: string
  alasan: string
  keyakinan: Keyakinan
  /** Kemungkinan sudah tercatat (arah & nominal sama, tanggal berdekatan) — dugaan, bukan kepastian. */
  duplikat: boolean
  /** Transaksi yang cocok; belum_lunas = mutasi ini bisa dicatat sebagai pelunasannya. */
  mirip?: Mirip
  /** Diisi ID transaksi: baris dicatat sebagai pelunasan transaksi itu, bukan transaksi baru. */
  lunasi?: number
  pilih: boolean
}

/** Transaksi yang kemungkinan sama dengan uang masuk/keluar tertentu (cegah jurnal dobel). */
export interface Mirip {
  id: number; invoice_id?: number; tanggal: string; jenis: string; keterangan?: string; pihak?: string
  nominal: number; sumber?: 'mutasi' | 'foto'; belum_lunas: boolean
}
export const cariMirip = (tanggal: string, nominal: number, arah: 'masuk' | 'keluar', kecualiGrup?: number) =>
  api<Mirip[]>(`/transactions/mirip?tanggal=${tanggal}&nominal=${nominal}&arah=${arah}${kecualiGrup ? `&kecuali_grup=${kecualiGrup}` : ''}`)
export const getDobel = (period: string) => api<[Tx, Tx][]>(`/transactions/dobel?period=${encodeURIComponent(period)}`)

export interface MutasiGrup {
  jenis: string
  arah: 'masuk' | 'keluar'
  jumlah: number
  total: number
}

export interface MutasiHasil {
  rows: MutasiRow[] | null
  errors: ImportError[] | null
  ringkasan: MutasiGrup[] | null
  total_masuk: number
  total_keluar: number
  perlu_diperiksa: number
  duplikat: number
  format: string
  /** Jumlah baris per bulan (cek mutasi masuk ke bulan yang benar) dan hal yang perlu dicek. */
  per_bulan?: { bulan: string; label: string; jumlah: number }[] | null
  peringatan?: string[] | null
  /** Baris yang jenisnya disarankan AI, dan kenapa AI tidak dipakai (bila ada). */
  dibantu_ai?: number
  ai_pesan?: string
}

export interface MutasiCommitResult {
  disimpan: number
  errors: ImportError[] | null
  contoh: Tx[] | null
}

/** Langkah 1: baca berkas & tebak jenisnya. Tidak menyimpan apa pun. */
/** sandi: kata sandi PDF e-statement; hanya dipakai server membuka berkas ini, tidak disimpan. */
export async function previewMutasi(file: File, tahun?: number, sandi?: string): Promise<MutasiHasil> {
  const form = new FormData()
  form.append('file', file)
  if (tahun) form.append('tahun', String(tahun))
  if (sandi) form.append('sandi', sandi)
  const res = await fetch('/api/v1/transactions/mutasi', { method: 'POST', body: form })
  if (!res.ok) throw await galatDari(res)
  return res.json()
}

/** Langkah 2: simpan baris yang sudah diperiksa — termasuk koreksi jenisnya. */
export const commitMutasi = (rows: MutasiRow[]) =>
  api<MutasiCommitResult>('/transactions/mutasi/commit', { method: 'POST', body: { rows } })

// --- Produksi & Supplier: resep (BOM) dan HPP per produk ---

/**
 * Takaran bahan disimpan sebagai bilangan bulat per seribu satuan: 1500 = 1,5 satuan.
 * Pecahan dibutuhkan karena resep nyata memakai 0,18 kg biji atau 0,15 liter susu, sementara
 * nominal rupiah tetap bilangan bulat.
 */
export const TAKARAN_SKALA = 1000

export interface ResepBahan {
  /** 0 = bahan di luar daftar stok; nama & harga_manual diisi sendiri. */
  stok_id: number
  nama: string
  satuan: string
  takaran: number
  harga_manual: number
}

export interface Produk {
  id: number
  sku: string
  nama: string
  satuan: string
  kategori: string
  /** Berapa unit produk jadi dari satu kali resep. */
  hasil_per_batch: number
  bahan: ResepBahan[] | null
  biaya_tenaga_kerja: number
  biaya_overhead: number
  harga_jual: number
  catatan?: string
}

export type ProdukInput = Omit<Produk, 'id'>

export interface HPPBahan {
  nama: string
  satuan: string
  takaran: number
  harga_satuan: number
  subtotal: number
  sumber: 'stok' | 'manual'
  peringatan?: string
}

export interface HPP {
  bahan: HPPBahan[] | null
  total_bahan: number
  tenaga_kerja: number
  overhead: number
  total_batch: number
  hasil_per_batch: number
  per_unit: number
  harga_jual: number
  laba_per_unit: number
  margin_bps: number
  peringatan: string[] | null
}

export interface ProdukRow extends Produk {
  hpp: HPP
  /** Harga pokok yang tercatat di Stok & Gudang, dicocokkan lewat SKU. */
  hpp_stok: number
  ada_di_stok: boolean
  /** HPP resep − HPP di stok. Bukan 0 = angka di stok sudah basi. */
  selisih: number
}

export interface Supplier {
  id: number
  nama: string
  kontak: string
  telepon: string
  email: string
  alamat: string
  /** Termin pembayaran dalam hari; 0 = tunai. */
  tempo_hari: number
  catatan?: string
}

export type SupplierInput = Omit<Supplier, 'id'>

export interface SupplierRow extends Supplier {
  /** Perkiraan, dicocokkan dari nama pihak di Piutang & Utang. */
  utang_belum_dibayar: number
  jumlah_tagihan: number
}

export interface ProduksiSummary {
  produk: ProdukRow[] | null
  supplier: SupplierRow[] | null
  total_produk: number
  total_supplier: number
  produk_rugi: number
  produk_tanpa_hpp: number
  utang_supplier: number
}

export const getProduksi = () => api<ProduksiSummary>('/produksi')
export const createProduk = (p: ProdukInput) => api<Produk>('/produksi/produk', { method: 'POST', body: p })
export const updateProduk = (id: number, p: ProdukInput) => api<Produk>(`/produksi/produk/${id}`, { method: 'PUT', body: p })
export const deleteProduk = (id: number) => api<void>(`/produksi/produk/${id}`, { method: 'DELETE' })
export const terapkanHPP = (id: number) => api<{ sku: string; harga_pokok: number }>(`/produksi/produk/${id}/terapkan-hpp`, { method: 'POST' })
export const createSupplier = (s: SupplierInput) => api<Supplier>('/produksi/supplier', { method: 'POST', body: s })
export const updateSupplier = (id: number, s: SupplierInput) => api<Supplier>(`/produksi/supplier/${id}`, { method: 'PUT', body: s })
export const deleteSupplier = (id: number) => api<void>(`/produksi/supplier/${id}`, { method: 'DELETE' })

// --- Invoice: faktur penjualan + cetak PDF ---

export interface InvoiceItem {
  deskripsi: string
  qty: number
  satuan?: string
  harga_satuan: number
}

export interface InvoiceTotals {
  subtotal: number
  diskon: number
  dpp: number
  ppn: number
  total: number
}

export interface InvoiceInput {
  nomor?: string
  no_po?: string
  status: string
  tanggal: string
  jatuh_tempo: string
  pelanggan_nama: string
  pelanggan_alamat?: string
  pelanggan_npwp?: string
  items: InvoiceItem[]
  diskon_rp: number
  ppn_bps: number
  catatan?: string
  /** Tanggal pelanggan membayar (wajib untuk status lunas). */
  tanggal_lunas?: string
  /** true = tercatat di pembukuan (Penjualan + PPN Keluaran otomatis). */
  dibukukan?: boolean
}

export interface Invoice extends InvoiceInput {
  id: number
  nomor: string
  transaksi_id?: number
  transaksi_ppn_id?: number
}

export interface InvoiceRow extends Invoice {
  totals: InvoiceTotals
  hari_terlambat: number
}

export interface InvoiceSummary {
  invoices: InvoiceRow[] | null
  total_tagihan: number
  belum_dibayar: number
  terlambat: number
  jumlah_terlambat: number
}

export interface InvoiceInfo {
  summary: InvoiceSummary
  status: string[]
  nomor_berikutnya: string
  ppn_bps: number
  /** Data penerbit yang masih kosong di Profil Usaha; faktur tetap bisa dicetak tanpanya. */
  profil_lengkap: string[] | null
}

export const getInvoices = () => api<InvoiceInfo>('/invoice')
/** Bukukan semua invoice lama sekaligus (dibuat sebelum invoice tersambung ke pembukuan). */
export const bukukanInvoice = () => api<{ dibukukan: number; gagal: string[] }>('/invoice/bukukan', { method: 'POST' })
export const createInvoice = (inv: InvoiceInput) => api<Invoice>('/invoice', { method: 'POST', body: inv })
export const updateInvoice = (id: number, inv: InvoiceInput) => api<Invoice>(`/invoice/${id}`, { method: 'PUT', body: inv })
export const deleteInvoice = (id: number) => api<void>(`/invoice/${id}`, { method: 'DELETE' })

/** URL unduhan PDF; dipakai langsung sebagai href agar browser yang mengunduh. */
export const invoicePdfUrl = (id: number) => `/api/v1/invoice/${id}/pdf`

// --- Laporan AI: rincian seluruh laporan + analisis model ---

export interface FactsLangkah { label: string; nilai: number; keterangan?: string }

export interface BusinessFacts {
  periode: string
  periode_label: string
  nama_usaha: string
  laba_rugi: {
    omzet: number; hpp: number; laba_kotor: number; beban_operasional: number
    penyusutan: number; amortisasi: number; total_biaya: number
    laba_bersih: number; margin_bps: number
    omzet_sebelumnya?: number; laba_sebelumnya?: number
  }
  arus_kas: {
    saldo_awal: number; kas_masuk: number; kas_keluar: number; kas_bersih: number
    saldo_akhir: number; operasi_langsung: number; operasi_tidak_langsung: number
    investasi: number; pendanaan: number
  }
  neraca: {
    total_aktiva: number; total_pasiva: number; kas: number; piutang: number
    persediaan: number; utang: number; ekuitas: number; seimbang: boolean
  }
  aset: {
    jumlah: number; harga_perolehan: number; akumulasi: number; nilai_buku: number
    beban_bulan_ini: number; draft: number; pemetaan_belum_dicek: number
  }
  pajak: {
    bentuk_badan: string; omzet_kumulatif: number; batas_bebas_pajak: number
    dasar_kena_pajak: number; pajak_terutang: number; sisa_batas_bebas: number
  }
  tagihan: {
    piutang_total: number; utang_total: number; piutang_lewat_tempo: number
    jumlah_faktur_telat: number; invoice_belum_dibayar: number
  }
  operasi: {
    jumlah_transaksi: number; biaya_aplikasi: number
    channel?: { name: string; gross: number; fee: number; net: number; pct: number }[]
    produk_teratas?: { name: string; qty: number; omzet: number; pct: number }[]
    stok_menipis?: string[]
    jumlah_karyawan: number; beban_gaji_bulanan: number
  }
  tren: { bulan: string; omzet: number; laba_bersih: number; saldo_kas: number }[]
  peringatan?: string[]
}

export interface AIAnalysis {
  tersedia: boolean
  alasan?: string
  model?: string
  ringkasan?: string
  sorotan_positif?: string[]
  perlu_diperhatikan?: string[]
  rekomendasi?: { judul: string; dampak: string; detail: string }[]
}

export interface AIReport {
  /** transaksi = laporan keuangan periode; profil = belum ada transaksi, dari data pendaftaran. */
  sumber: 'transaksi' | 'profil'
  facts?: BusinessFacts
  profil_usaha?: BusinessProfile
  analysis: AIAnalysis
  llm: { aktif: boolean; model?: string; base_url?: string; petunjuk?: string }
}

export const getAIReport = (period: string) => api<AIReport>(`/reports/ai?period=${period}`)

// --- Kalkulator Pajak ---

export interface ObjekPajak {
  kode: string; nama: string; tarif_bps: number; dasar_bps: number
  final: boolean; dasar: string; catatan?: string
}

export interface JenisPajak {
  id: string; nama: string; keterangan: string
  objek: ObjekPajak[]; tanpa_npwp_bps?: number
}

export interface KalkulatorMaster {
  jenis: JenisPajak[]
  ptkp_codes: string[]
  ptkp: Record<string, number>
  ter: {
    kategori: Record<string, string>
    lapisan: Record<string, { batas_bawah: number; batas_atas: number; tarif_bps: number }[]>
    belum_lengkap: string[] | null
  }
  tarif_badan_bps: number
  parameter: { ppn_efektif_bps: number; dasar_umkm: string; batas_nppn: number; batas_pkp: number }
}

export interface Langkah { label: string; nilai: number; keterangan?: string }

export interface PPh21Result {
  bruto_setahun: number; biaya_jabatan: number; iuran_pensiun_setahun: number
  neto_setahun: number; ptkp: number; pkp: number
  pph_setahun: number; pph_sebulan: number; tambahan_tanpa_npwp: number
  tunjangan_pajak: number; take_home_sebulan: number; langkah: Langkah[]
}

export interface PPh21Response {
  hasil: PPh21Result
  ter: { tersedia: boolean; tarif_bps?: number; pph_sebulan?: number; alasan?: string }
}

export interface PotongResult {
  jenis_nama: string; objek_nama: string; nilai_bruto: number; dpp: number
  tarif_bps: number; pajak: number; diterima: number; final: boolean
  dasar: string; catatan?: string; tambahan_tanpa_npwp: number; langkah: Langkah[]
}

export interface PPhBadanResult {
  peredaran_bruto: number; pkp: number; tarif_bps: number
  dapat_fasilitas: boolean; pph_terutang: number; langkah: Langkah[]; catatan: string
}

export const getKalkulator = () => api<KalkulatorMaster>('/kalkulator')
export const hitungPPh21 = (b: { penghasilan_bruto: number; ptkp_status: string; iuran_pensiun: number; gross_up: boolean; punya_npwp: boolean }) =>
  api<PPh21Response>('/kalkulator/pph21', { method: 'POST', body: b })
export const hitungPotong = (b: { jenis: string; objek: string; nilai_bruto: number; punya_npwp: boolean }) =>
  api<PotongResult>('/kalkulator/potong', { method: 'POST', body: b })
export const hitungPPhBadan = (b: { peredaran_bruto: number; penghasilan_kena_pajak: number }) =>
  api<PPhBadanResult>('/kalkulator/pph-badan', { method: 'POST', body: b })

/** Posisi omzet setahun terhadap batas wajib PKP Rp4,8 M. */
export interface HasilPKP { omzet: number; batas: number; wajib_pkp: boolean; sisa: number; pemakaian_bps: number; catatan: string[] }
export const cekPKP = (omzet: number) => api<HasilPKP>('/kalkulator/pkp', { method: 'POST', body: { omzet } })

// --- Kalkulator perencanaan (Tax Planning Calculator V2.1) ---

export type Wilayah = 'ibukota10' | 'ibukota_lain' | 'daerah_lain'
/** Norma per KLU 5 digit; -1 = tidak tersedia untuk wilayah itu. */
export interface Norma { klu: string; uraian: string; ibukota10_bps: number; ibukota_lain_bps: number; daerah_lain_bps: number }
export interface NPPNResult {
  norma: Norma | null; norma_bps: number; neto: number; ptkp: number; pkp: number; pph: number
  pph_final: number; boleh: boolean; catatan: string[]; langkah: Langkah[]
}
export interface GrossUpResult {
  nilai_kontrak: number; dpp: number; pajak: number; diterima: number; tarif_efektif_bps: number; final: boolean; langkah: Langkah[]
}
export interface SkemaPPN {
  id: 'non_pkp' | 'pkp_dibebankan' | 'pkp_ditanggung'; nama: string; keterangan: string
  harga_pembeli: number; dpp: number; ppn_keluaran: number; ppn_masukan: number; ppn_setor: number; hpp: number; opex: number; laba: number
}
export interface PPNHargaResult { tarif_efektif_bps: number; skema: SkemaPPN[]; kenaikan_harga_bps: number; laba_tertinggi: string; catatan: string[] }
export interface TahunAset { tahun: number; beban: number; laba: number; pph: number }
export interface OpsiAset { tahun: TahunAset[]; total_pph: number; total_kas: number }
export interface BeliSewaResult {
  metode: string; beli: OpsiAset; sewa: OpsiAset; potong_sewa: number; rekomendasi: 'beli' | 'sewa'; selisih: number; catatan: string[]
}
export interface SaranPajak { judul: string; isi: string; langganan: boolean }
export interface MarketplaceResult {
  omzet_total: number; laba: number; final: boolean; dipungut: number; terutang: number; terutang_umum: number
  lebih_bayar: number; kurang_bayar: number; npm_impas_bps: number
  status: 'bebas' | 'lunas' | 'kurang_bayar' | 'lebih_bayar'
  rekomendasi: SaranPajak[]; langkah: Langkah[]; catatan: string[]
}
export const hitungMarketplace = (b: { omzet_marketplace: number; omzet_lain: number; npm_bps: number; laba?: number; bentuk: 'op' | 'badan'; skema: 'final' | 'umum'; ptkp: string }) =>
  api<MarketplaceResult>('/kalkulator/pph22-marketplace', { method: 'POST', body: b })
export const cariNorma = (q: string) => api<Norma[]>('/kalkulator/norma?q=' + encodeURIComponent(q))
export const hitungNPPN = (b: { omzet: number; klu: string; wilayah: string; norma_bps: number; ptkp: string }) =>
  api<NPPNResult>('/kalkulator/nppn', { method: 'POST', body: b })
export const hitungGrossUp = (b: { jenis: string; objek: string; diterima: number; punya_npwp: boolean }) =>
  api<GrossUpResult>('/kalkulator/gross-up', { method: 'POST', body: b })
export const hitungPPNHarga = (b: { pembelian: number; termasuk_ppn: boolean; porsi_dari_pkp_bps: number; markup_bps: number; opex_bps: number; opex_nominal: number }) =>
  api<PPNHargaResult>('/kalkulator/ppn-harga', { method: 'POST', body: b })
export const hitungBeliSewa = (b: {
  bentuk: 'badan' | 'op'; ptkp: string; omzet: number; biaya_lain: number; harga_aset: number
  umur_tahun: number; metode: string; bangunan: boolean; sewa_per_tahun: number
}) => api<BeliSewaResult>('/kalkulator/beli-sewa', { method: 'POST', body: b })

// --- Parameter pajak (Panel Master) ---

export interface Lapis { sampai: number; tarif_bps: number }
export interface TERBaris { batas_bawah: number; batas_atas: number; tarif_bps: number }
/** Isi satu versi parameter; norma tidak ikut (dikelola lewat Excel). */
export interface ParameterIsi {
  ptkp: Record<string, number>
  pasal17: Lapis[]
  biaya_jabatan_bps: number; biaya_jabatan_maks: number; tanpa_npwp_pph21_bps: number; dasar_bukan_pegawai_bps: number
  ter: { kategori: Record<string, string>; lapisan: Record<string, TERBaris[]> }
  umkm_tarif_bps: number; umkm_bebas_op: number; batas_omzet_umkm: number; dasar_umkm: string
  badan_tarif_bps: number; fasilitas_31e_bps: number; batas_31e_penuh: number; batas_31e: number; dividen_bps: number
  batas_pkp: number; ppn_tarif_bps: number; ppn_dpp_pembilang: number; ppn_dpp_penyebut: number
  marketplace_tarif_bps: number; marketplace_bebas_op: number; dasar_marketplace: string
  batas_nppn: number; norma_perbandingan_bps: number
  objek: JenisPajak[]
  perlu_verifikasi: string[] | null
}
export interface ParameterVersi {
  id: number; nama: string; dasar: string; status: StatusKatalog; isi: ParameterIsi
  oleh?: string; waktu: string; jumlah_norma: number
}
export const getParameter = () => api<{ versi: ParameterVersi[]; ptkp_codes: string[]; objek_terikat: string[] }>('/master/parameter')
export const salinParameter = (id: number) => api<ParameterVersi>(`/master/parameter/${id}/salin`, { method: 'POST' })
export const ubahParameter = (id: number, b: { nama: string; dasar: string; isi: ParameterIsi }) =>
  api<ParameterVersi>(`/master/parameter/${id}`, { method: 'PUT', body: b })
export const hapusParameter = (id: number) => api<void>(`/master/parameter/${id}`, { method: 'DELETE' })
export const aktifkanParameter = (id: number) => api<ParameterVersi>(`/master/parameter/${id}/aktifkan`, { method: 'POST' })
export const imporNorma = (id: number, file: File) => unggah<ParameterVersi>(`/master/parameter/${id}/norma`, file, {})
export const unduhNormaUrl = (id: number) => `/api/v1/master/parameter/${id}/norma.xlsx`

// --- Artikel pajak, basis pengetahuan AI, analisa AI per klien ---

export const KATEGORI_ARTIKEL = ['PPh Final UMKM', 'PPh 21', 'PPh 23', 'PPh Badan', 'PPN', 'Pembukuan', 'Administrasi'] as const
export interface Artikel {
  id: number; judul: string; kategori: string; dasar_hukum?: string; berlaku_sejak?: string; tag: string[]; isi: string
  status: 'draft' | 'terbit'; tampil_klien: boolean; oleh: string; diverifikasi?: string; waktu: string
}
export type ArtikelInput = Pick<Artikel, 'judul' | 'kategori' | 'dasar_hukum' | 'berlaku_sejak' | 'tag' | 'isi' | 'tampil_klien'>
export interface SumberPengetahuan {
  id: number; judul: string; jenis: 'manual' | 'excel' | 'pdf'; kategori?: KategoriSumber; topik: string[]; status: 'menunggu' | 'terverifikasi'
  dipakai_ai: boolean; oleh: string; diverifikasi?: string; waktu: string; potongan: number; cuplikan: string
}
export interface Potongan { ref: string; judul: string; teks: string; skor?: number }
export interface FaktaKlien {
  nama: string; jenis_usaha: string; bentuk_usaha: string; pkp: boolean; klu?: string; klu_nama?: string
  omzet_setahun: number; npm_bps: number; benchmark_npm_bps?: number; benchmark_cttor_bps?: number; sumber_benchmark?: string
  status_npm: 'kurang_data' | 'di_bawah' | 'wajar'; risiko?: string[]
}
export interface AnalisaKlien {
  fakta: FaktaKlien; sumber: Potongan[]
  analisa: { tersedia: boolean; alasan?: string; model?: string; penilaian?: string; poin?: { judul: string; isi: string; sumber: string[] }[] }
}

export const listArtikel = () => api<Artikel[]>('/master/artikel')
export const simpanArtikel = (id: number | null, a: ArtikelInput, terbit: boolean) =>
  api<Artikel>(id ? `/master/artikel/${id}` : '/master/artikel', { method: id ? 'PUT' : 'POST', body: { ...a, terbit } })
export const listPustaka = () => api<Artikel[]>('/pustaka')

// --- Primbon Pajak: katalog peraturan resmi (internal/primbon, cmd/server/primbon.go) ---

export interface CatatanPrimbon { isi: string; diubah: string }

export interface Peraturan {
  uuid: string
  kode: string
  judul: string
  jenis: string // UU | PP | PERPRES | PMK | KMK | PERDIRJEN | SE_DJP | SEMA | LAIN
  jenis_label: string
  kategori: string[]
  kategori_label: string[]
  status?: string
  tahun?: number
  ditetapkan?: string
  diundangkan?: string
  berlaku?: string
  sumber_ln?: string
  halaman?: number
  /** Naskah di server; kosong = belum tersedia. */
  format?: 'pdf' | 'htm'
  lampiran?: boolean
  abstrak?: boolean
  sumber_url?: string
  bookmark: boolean
  ada_catatan: boolean
  waktu?: string
  catatan?: CatatanPrimbon
}

export interface OpsiPrimbon { kode: string; label: string; jumlah: number }

export interface HasilPrimbon {
  items: Peraturan[]
  total: number
  halaman: number
  total_halaman: number
  facet: { kategori: OpsiPrimbon[]; jenis: OpsiPrimbon[]; status: OpsiPrimbon[]; tahun_min: number; tahun_max: number }
}

export interface FilterPrimbon {
  q: string
  kategori: string[]
  jenis: string[]
  status: string[]
  dari: string
  sampai: string
  urut: 'terbaru' | 'terlama' | 'abjad'
  hal: number
}

export const FILTER_PRIMBON_AWAL: FilterPrimbon = { q: '', kategori: [], jenis: [], status: [], dari: '', sampai: '', urut: 'terbaru', hal: 1 }

export function cariPrimbon(f: FilterPrimbon) {
  const p = new URLSearchParams()
  if (f.q.trim()) p.set('q', f.q.trim())
  for (const k of ['kategori', 'jenis', 'status'] as const) f[k].forEach((v) => p.append(k, v))
  if (f.dari) p.set('dari', f.dari)
  if (f.sampai) p.set('sampai', f.sampai)
  if (f.urut !== 'terbaru') p.set('urut', f.urut)
  if (f.hal > 1) p.set('hal', String(f.hal))
  return api<HasilPrimbon>('/primbon/peraturan?' + p)
}
const uuidPath = (uuid: string) => encodeURIComponent(uuid)
export const getPeraturan = (uuid: string) => api<Peraturan>(`/primbon/peraturan/${uuidPath(uuid)}`)
export const berkasPrimbonUrl = (uuid: string, opsi: { unduh?: boolean; lampiran?: boolean } = {}) =>
  `/api/v1/primbon/peraturan/${uuidPath(uuid)}/berkas` + (opsi.unduh || opsi.lampiran
    ? '?' + new URLSearchParams([...(opsi.lampiran ? [['lampiran', '1']] : []), ...(opsi.unduh ? [['unduh', '1']] : [])])
    : '')
export const setBookmarkPrimbon = (uuid: string, aktif: boolean) =>
  api<void>(`/primbon/bookmark/${uuidPath(uuid)}`, { method: aktif ? 'PUT' : 'DELETE' })
export const listBookmarkPrimbon = () => api<Peraturan[]>('/primbon/bookmark')
export const listCatatanPrimbon = () => api<Peraturan[]>('/primbon/catatan')
export const simpanCatatanPrimbon = (uuid: string, isi: string) =>
  api<CatatanPrimbon | null>(`/primbon/catatan/${uuidPath(uuid)}`, { method: 'PUT', body: { isi } })
export const listPengetahuan = () => api<{ sumber: SumberPengetahuan[]; artikel_terbit: number; potongan_dipakai: number }>('/master/pengetahuan')
/** Kategori isi pengetahuan AI: bukan hanya aturan — pengalaman lapangan & pengetahuan umum juga. */
export type KategoriSumber = 'peraturan' | 'artikel' | 'praktik' | 'umum' | 'lainnya'
export const KATEGORI_SUMBER: [KategoriSumber, string, string][] = [
  ['praktik', 'Praktik lapangan', 'Pengalaman tim menangani klien, kebiasaan di KPP, tips yang tidak ada di teori'],
  ['umum', 'Pengetahuan umum', 'Bisnis, keuangan, operasional UMKM, istilah sehari-hari'],
  ['peraturan', 'Peraturan & UU', 'UU, PP, PMK, PER, SE'],
  ['artikel', 'Artikel / teori pajak', 'Penjelasan konsep dan teori'],
  ['lainnya', 'Lainnya', ''],
]
export const tulisPengetahuan = (judul: string, isi: string, topik: string[], kategori: KategoriSumber) =>
  api<SumberPengetahuan>('/master/pengetahuan', { method: 'POST', body: { judul, isi, topik, kategori } })
export const unggahPengetahuan = (file: File, judul: string, topik: string, kategori: KategoriSumber) => unggah<SumberPengetahuan>('/master/pengetahuan', file, { judul, topik, kategori })
export const verifikasiPengetahuan = (id: number) => api<unknown>(`/master/pengetahuan/${id}/verifikasi`, { method: 'POST' })
export const pakaiPengetahuan = (id: number, dipakai_ai: boolean) => api<unknown>(`/master/pengetahuan/${id}`, { method: 'PUT', body: { dipakai_ai } })
export const hapusPengetahuan = (id: number) => api<void>(`/master/pengetahuan/${id}`, { method: 'DELETE' })
export const cariPengetahuan = (q: string) => api<Potongan[]>(`/master/pengetahuan/cari?q=${encodeURIComponent(q)}`)
export const analisaKlien = (id: number) => api<AnalisaKlien>(`/master/klien/${id}/analisa`)
// --- Simulasi usaha (Simulasi & Harga) ---

export interface DasarSimulasi {
  periode: string[]; omzet: number; modal_bahan: number; biaya_variabel: number; biaya_tetap: number
  penyusutan: number; pendapatan_lain: number; laba: number; hari_buka: number
}
export interface HasilSimulasi {
  omzet: number; modal_bahan: number; biaya_variabel: number; biaya_tetap: number; penyusutan: number
  pendapatan_lain: number; laba: number; margin_bps: number
}
export interface TitikImpasUsaha { ada: boolean; per_bulan: number; per_hari: number; biaya_tetap: number; sisa_per_100rb: number; di_atas_bps: number }
export interface PorsiProduk { nama: string; harga: number; modal: number; ada_modal: boolean; variabel: number; sisa: number; porsi_per_hari: number; ada: boolean }
/** Perubahan dalam basis poin: 500 = +5%. */
export interface Skenario { harga_bps: number; volume_bps: number; modal_bps: number; tambah_tetap: number }
export interface SimulasiUsaha { dasar: DasarSimulasi; hasil: HasilSimulasi; titik_impas: TitikImpasUsaha; produk: PorsiProduk[] }
export const simulasiUsaha = (period: string, s: Skenario) => api<SimulasiUsaha>(`/simulasi?period=${period}`, { method: 'POST', body: s })

// --- Ekspor PDF & perbandingan multi-bulan Laporan Keuangan ---

export type BagianPDF = 'semua' | 'laba-rugi' | 'neraca' | 'arus-kas' | 'aset'
/** Semua daftar (piutang, utang, invoice, barang, pelanggan, pemasok) untuk satu bulan. */
export const daftarUrl = (period: string, format: 'xlsx' | 'pdf') => `/api/v1/reports/daftar.${format}?period=${period}`
/** rentang: laba rugi & arus kas dari..sampai (neraca tetap posisi akhir periode). */
export const laporanPdfUrl = (period: string, bagian: BagianPDF, metode?: 'langsung' | 'tidak-langsung', rentang?: { dari: string; sampai: string }) =>
  `/api/v1/reports/pdf?period=${period}&bagian=${bagian}${metode ? `&metode=${metode}` : ''}${rentang ? `&dari=${rentang.dari}&sampai=${rentang.sampai}` : ''}`

/** Laba rugi & arus kas untuk rentang tanggal bebas. */
export interface LaporanRentang { dari: string; sampai: string; label: string; income: IncomeStatement; cashflow: CashFlow }
export const getLaporanRentang = (dari: string, sampai: string) => api<LaporanRentang>(`/reports/rentang?dari=${dari}&sampai=${sampai}`)
export interface BarisBanding {
  kunci: string; label: string; gaya: 'judul' | 'baris' | 'sub' | 'total'
  /** Isi kolom Total: jumlah (laba rugi, arus kas), awal (saldo awal), akhir (neraca, saldo akhir). */
  agregat?: 'jumlah' | 'awal' | 'akhir'
  nilai: number[]; total: number
  /** Laba rugi opsi B: huruf baris, rumus baris jumlah, sifat angka (lib/angkaLaporan). */
  huruf?: string; rumus?: string; sifat?: '' | 'pengurang' | 'kontra'
}
export interface BagianBanding { kode: 'laba-rugi' | 'neraca' | 'arus-kas'; judul: string; baris: BarisBanding[] }
export interface Banding { kolom: { periode: string; label: string }[]; bagian: BagianBanding[] }
export const MAKS_BANDING = 12
export const getBanding = (periode: string[]) => api<Banding>(`/reports/banding?periode=${periode.join(',')}`)
export const bandingPdfUrl = (periode: string[]) => `/api/v1/reports/banding.pdf?periode=${periode.join(',')}`

// --- Laporan produksi ---
export interface BarisProduksi {
  id: number; tanggal: string; produk_id: number; produk: string; batch: number; hasil_qty: number; hasil_standar: number; satuan: string
  nilai: number; per_unit: number; bahan: { nama: string; satuan: string; qty: number; standar: number; nilai: number }[]; keterangan?: string; oleh?: string
}
export interface LaporanProduksi { dari: string; sampai: string; mulai: string; baris: BarisProduksi[]; total_nilai: number }
export const getLaporanProduksi = (dari: string, sampai: string) => api<LaporanProduksi>(`/produksi/laporan?dari=${dari}&sampai=${sampai}`)
export const laporanProduksiXlsxUrl = (dari: string, sampai: string) => `/api/v1/produksi/laporan.xlsx?dari=${dari}&sampai=${sampai}`

// --- Umur piutang & utang (dari transaksi belum lunas + saldo awal; sama dengan Neraca) ---
export interface UmurSaldo {
  per: string; piutang: boolean
  kelompok: { label: string; nilai: number; jumlah: number }[]
  pihak: { nama: string; nilai: number[]; total: number }[]
  baris: { transaksi_id?: number; tanggal: string; pihak: string; keterangan: string; jenis: string; nilai: number; hari: number; kelompok: number }[]
  total: number
}
export const getUmurSaldo = (jenis: 'piutang' | 'utang', per: string) => api<UmurSaldo>(`/reports/umur?jenis=${jenis}&per=${per}`)
export const lunasiTransaksi = (id: number, tanggal: string) => api<SavedTx>(`/transactions/${id}/lunas`, { method: 'POST', body: { tanggal } })

// --- Alur kerja: tutup buku, langkah memulai, pelunasan saldo awal ---
export interface CekTutup { kode: string; judul: string; status: 'ok' | 'peringatan' | 'info'; detail: string; view?: View }
export interface TutupBuku {
  periode: string; label: string; tutup_sampai: string; terkunci: boolean; bisa_tutup: boolean; bisa_buka: boolean
  cek: CekTutup[]; laba: number; ebitda: number; penyusutan: number; kas_akhir: number
}
export const getTutupBuku = (period: string) => api<TutupBuku>(`/tutup-buku?period=${period}`)
export const tutupBulan = (periode: string) => api<TutupBuku>('/tutup-buku', { method: 'POST', body: { periode } })
export const bukaBulan = () => api<TutupBuku>('/tutup-buku', { method: 'DELETE' })
export interface LangkahMulai { kode: string; judul: string; penjelasan: string; selesai: boolean; view: View; aksi?: string; peringatan?: string }
export const getLangkahMulai = () => api<LangkahMulai[]>('/mulai')

/** Kecocokan saldo awal dengan daftar aset, daftar barang & kartu stok, dan Harta = Kewajiban + Modal. */
export interface CekKecocokan {
  kode: string; judul: string; status: 'ok' | 'selisih' | 'info'
  saldo_awal: number; pembanding: number; label_banding: string; selisih: number
  pesan: string; view: View; langkah: string
}
export const getCekSaldoAwal = () => api<CekKecocokan[]>('/saldo-awal/cek')
export const pelunasanSaldoAwal = (b: { jenis: 'piutang' | 'utang'; tanggal: string; nominal: number }) => api<unknown>('/opening/pelunasan', { method: 'POST', body: b })
