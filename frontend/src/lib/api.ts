// Tipe mengikuti struct JSON di internal/finance dan internal/auth (Go).

// 'arus-kas' dihapus: Arus Kas jadi tab di dalam Laporan Keuangan, bukan menu sendiri.
export type View =
  | 'dashboard' | 'laporan-keuangan' | 'laporan-penjualan' | 'transaksi' | 'piutang-utang' | 'stok-gudang'
  | 'produksi-supplier' | 'invoice' | 'simulasi-harga' | 'pajak-umkm' | 'perencanaan-pajak' | 'karyawan' | 'master-data'
  | 'panduan' | 'pengguna' | 'profil-usaha'
  | 'transaksi-kelola' // izin, bukan menu: ubah & hapus transaksi
  | 'impor-data' // menu Impor Data Awal, sekaligus izin mengubah neraca awal

export type Role = 'owner' | 'admin' | 'kasir'

export interface User {
  id: number
  name: string
  email: string
  role: Role
  active: boolean
  created_at: string
}

export interface Me {
  user: User
  role_label: string
  views: View[]
}

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
  opex_lines: { label: string; amount: number }[] | null
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
    total: number
  }
  liabilities: { payables: number; loans: number; total: number }
  equity: { opening_capital: number; added_capital: number; current_profit: number; total: number }
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
}

export type TxInput = Omit<Tx, 'id'>

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

async function api<T>(path: string, init?: { method?: string; body?: unknown }): Promise<T> {
  const res = await fetch('/api/v1' + path, {
    method: init?.method ?? 'GET',
    headers: init?.body === undefined ? undefined : { 'Content-Type': 'application/json' },
    body: init?.body === undefined ? undefined : JSON.stringify(init.body),
  })
  if (!res.ok) throw new ApiError(res.status, (await res.text()).trim() || res.statusText)
  return (res.status === 204 ? undefined : await res.json()) as T
}

export interface PeriodOption {
  id: string
  label: string
}

export const getPeriods = () => api<PeriodOption[]>('/periods')

export const getMe = () => api<Me>('/auth/me')
export const login = (email: string, password: string) => api<Me>('/auth/login', { method: 'POST', body: { email, password } })
export const register = (name: string, email: string, password: string, kode: string) =>
  api<Me>('/auth/register', { method: 'POST', body: { name, email, password, kode } })
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
}

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
  /** Jumlah baris pemetaan jenis harta yang belum diverifikasi ke Lampiran PMK. */
  pemetaan_belum_dicek: number
}

export const getAssets = (period: string) => api<AssetSummary>(`/assets?period=${encodeURIComponent(period)}`)
export const createAsset = (a: AssetInput) => api<Asset>('/assets', { method: 'POST', body: a })
export const updateAsset = (id: number, a: AssetInput) => api<Asset>(`/assets/${id}`, { method: 'PUT', body: a })
export const deleteAsset = (id: number) => api<void>(`/assets/${id}`, { method: 'DELETE' })

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
  if (!res.ok) throw new ApiError(res.status, (await res.text()).trim() || res.statusText)
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
}

export interface Option { value: string; label: string; hint?: string }

export interface BusinessInfo {
  profile: BusinessProfile | null
  options: { jenis_usaha: Option[]; bentuk_usaha: Option[]; ptkp: string[]; metode_persediaan: Option[] }
}

export const getBusiness = () => api<BusinessInfo>('/business')
export const saveBusiness = (p: BusinessProfile) => api<BusinessProfile>('/business', { method: 'PUT', body: p })

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
}

export const getCoA = () => api<CoAInfo>('/coa')

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
  /** Opsional, hanya untuk dicocokkan dengan modal hasil hitung. */
  modal_disetor: number
}

export interface OpeningInfo {
  opening: OpeningBalance | null
  modal_awal: number
  /** Bukan 0 = isian neraca awal tidak seimbang, perlu diperiksa. */
  selisih_modal: number
  /** false = laporan masih memakai saldo awal data contoh. */
  aktif: boolean
}

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
  if (!res.ok) throw new ApiError(res.status, (await res.text()).trim() || res.statusText)
  return res.json()
}

export const getOpening = () => api<OpeningInfo>('/opening')
export const saveOpening = (o: OpeningBalance) => api<OpeningInfo>('/opening', { method: 'PUT', body: o })

export const getKaryawan = () => api<KaryawanSummary>('/karyawan')
export const createKaryawan = (k: KaryawanInput) => api<Karyawan>('/karyawan', { method: 'POST', body: k })
export const updateKaryawan = (id: number, k: KaryawanInput) => api<Karyawan>(`/karyawan/${id}`, { method: 'PUT', body: k })
export const deleteKaryawan = (id: number) => api<void>(`/karyawan/${id}`, { method: 'DELETE' })
