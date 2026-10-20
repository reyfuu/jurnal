// Tipe mengikuti struct JSON di internal/finance dan internal/auth (Go).

// 'arus-kas' dihapus: Arus Kas jadi tab di dalam Laporan Keuangan, bukan menu sendiri.
export type View =
  | 'dashboard' | 'laporan-keuangan' | 'laporan-penjualan' | 'transaksi' | 'piutang-utang' | 'stok-gudang'
  | 'produksi-supplier' | 'invoice' | 'simulasi-harga' | 'pajak-umkm' | 'kalkulator-pajak' | 'perencanaan-pajak' | 'karyawan' | 'master-data'
  | 'panduan' | 'pengguna' | 'profil-usaha' | 'lapor-bug' | 'langganan'
  | 'transaksi-kelola' // izin, bukan menu: ubah & hapus transaksi
  | 'impor-data' // menu Impor Data Awal, sekaligus izin mengubah neraca awal
  | 'panel-master' // akun tim MasterPajak: seluruh aplikasi diganti Panel Master
  | 'usulan-koreksi' // pemilik usaha meninjau usulan koreksi tim MasterPajak

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
  /** Hanya akun tim MasterPajak: role tim dan izin efektifnya. */
  master?: MasterMe
  /** Akun tim yang sedang "lihat sebagai klien": views berisi menu klien versi baca. */
  lihat?: { klien_id: number; nama: string; alasan: string; sampai: string }
  /** Usulan koreksi yang menunggu keputusan pemilik usaha. */
  koreksi_menunggu?: number
}

/** Status langganan usaha; 'habis' = hanya bisa melihat data dan membayar. */
export interface LanggananStatus { status: 'trial' | 'aktif' | 'habis'; sampai?: string }
export interface Paket { id: string; nama: string; harga: number; hari: number; hemat?: string }
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
  /** Kelompok KLU 3 digit; pembanding NPM & CTTOR DJP. */
  klu?: string
  /** Perkiraan setahun dari onboarding; laba sebelum pajak boleh negatif (rugi). */
  omzet_tahunan?: number
  laba_sebelum_pajak?: number
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
export interface KLUOption { kode: string; nama: string; jenis: JenisUsaha }

export interface BusinessInfo {
  profile: BusinessProfile | null
  options: { jenis_usaha: Option[]; bentuk_usaha: Option[]; ptkp: string[]; metode_persediaan: Option[]; provinsi: string[]; klu: KLUOption[] }
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

/** Hasil tax.CekAwal: NPM vs benchmark DJP dan risiko pajak. */
export interface CekAwal {
  /** Rata-rata nasional DJP untuk KLU usaha; null bila KLU belum dipilih. */
  klu: { kode: string; nama: string; cttor_bps: number; npm_bps: number } | null
  npm_bps: number
  cttor_bps: number
  status: 'kurang_data' | 'di_bawah' | 'wajar'
  pesan: string
  cttor_pesan?: string
  sumber: string
  risiko: { level: 'tinggi' | 'sedang' | 'info'; judul: string; isi: string }[]
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
  /** Balasan tim MasterPajak, terbaca pelapor. */
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
  if (!res.ok) throw new ApiError(res.status, (await res.text()).trim() || res.statusText)
  return res.json()
}

// --- Panel Master (akun tim MasterPajak) ---

export type Izin = 'klien_semua' | 'klien_lihat' | 'tiket' | 'lihat_sebagai' | 'usul_koreksi' | 'langganan' | 'coa_template' | 'benchmark'
  | 'pembayaran' | 'tim' | 'audit'
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
}
export interface RingkasanMaster {
  klien: number; aktif: number; trial: number; habis: number; mrr?: number; tiket_terbuka: number
  perlu_perhatian: KlienRow[]; belum_ditugaskan?: KlienRow[]; terakhir_aktif: KlienRow[]
}
export interface TiketRow extends BugReport { tenant_id: number; status: TiketStatus; nomor: string; klien_nama: string }
export interface MasterOrder { id: string; paket: string; jumlah: number; status: 'pending' | 'lunas' | 'gagal' | 'manual'; waktu: string; dibayar_pada?: string }
export interface KlienDetail { klien: KlienRow; users: User[]; orders: MasterOrder[]; tiket: TiketRow[]; koreksi: KoreksiRow[] }
export interface PembayaranRow { klien_id: number; klien_nama: string; order: MasterOrder; garansi?: GaransiStatus }
export interface AnggotaRow { user: User; role: string; klien: number[] }
export interface TimData { anggota: AnggotaRow[]; roles: MasterRole[]; izin: IzinInfo[]; klien: Orang[] }
export interface AuditRow { id: number; waktu: string; user_id: number; nama: string; peran: string; aksi: string; klien_id?: number; keterangan?: string }

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
  /** Ada versi template lebih baru dari tim MasterPajak; berlaku setelah Owner menerapkannya. */
  pembaruan?: { id: number; versi: number; diff: DiffAkun } | null
}
export interface DiffAkun { tambah: Account[]; hapus: Account[]; ubah: Account[] }

export const getCoA = () => api<CoAInfo>('/coa')
export const perbaruiCoA = () => api<unknown>('/coa/perbarui', { method: 'POST' })

// --- Katalog Panel Master: template COA & benchmark DJP ---

export type StatusKatalog = 'draft' | 'terbit' | 'aktif' | 'arsip'
export interface TemplateCOA { id: number; jenis: string; versi: number; status: StatusKatalog; akun: Account[]; catatan?: string; oleh?: string; waktu: string }
export interface KLUBenchmark { kode: string; nama: string; cttor_bps: number; npm_bps: number }
export interface BenchmarkVersi { id: number; nama: string; sumber: string; status: StatusKatalog; baris: KLUBenchmark[]; oleh?: string; waktu: string }
export interface KatalogCOA { jenis: string[]; template: TemplateCOA[]; kode_wajib: string[]; klien_per_template: Record<number, number> }

async function unggah<T>(path: string, file: File, fields: Record<string, string>): Promise<T> {
  const form = new FormData()
  form.append('file', file)
  for (const [k, v] of Object.entries(fields)) form.append(k, v)
  const res = await fetch('/api/v1' + path, { method: 'POST', body: form })
  if (!res.ok) throw new ApiError(res.status, (await res.text()).trim() || res.statusText)
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
  status: 'Habis' | 'Stok Menipis' | 'Aman'
  margin_bps: number
  nilai: number
}

export interface StokSummary {
  items: StokRow[] | null
  total_item: number
  nilai_persediaan: number
  menipis: number
  habis: number
}

export const getTagihan = () => api<TagihanSummary>('/tagihan')
export const createTagihan = (t: TagihanInput) => api<Tagihan>('/tagihan', { method: 'POST', body: t })
export const updateTagihan = (id: number, t: TagihanInput) => api<Tagihan>(`/tagihan/${id}`, { method: 'PUT', body: t })
export const deleteTagihan = (id: number) => api<void>(`/tagihan/${id}`, { method: 'DELETE' })

export const getStok = () => api<StokSummary>('/stok')
export const createStok = (s: StokInput) => api<StokItem>('/stok', { method: 'POST', body: s })
export const updateStok = (id: number, s: StokInput) => api<StokItem>(`/stok/${id}`, { method: 'PUT', body: s })
export const deleteStok = (id: number) => api<void>(`/stok/${id}`, { method: 'DELETE' })

export const TAGIHAN_TEMPLATE_URL = '/api/v1/tagihan/template'
export const STOK_TEMPLATE_URL = '/api/v1/stok/template'

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
}

/** commit=false hanya pratinjau (tidak menyimpan apa pun). */
async function importMenu(path: string, file: File, commit: boolean, mode: ImportMode): Promise<MenuImportResult> {
  const form = new FormData()
  form.append('file', file)
  const q = commit ? `?commit=1&mode=${mode}` : ''
  const res = await fetch(`/api/v1${path}/import${q}`, { method: 'POST', body: form })
  if (!res.ok) throw new ApiError(res.status, (await res.text()).trim() || res.statusText)
  return res.json()
}

export const importTagihan = (f: File, commit: boolean, mode: ImportMode = 'tambah') => importMenu('/tagihan', f, commit, mode)
export const importStok = (f: File, commit: boolean, mode: ImportMode = 'tambah') => importMenu('/stok', f, commit, mode)

// --- Impor mutasi bank: tebak jenis transaksi, periksa, baru simpan ---

/** Kesalahan per baris berkas impor. baris 0 = masalah berkas, bukan baris data. */
export interface ImportError {
  baris: number
  pesan: string
}

/** Tingkat keyakinan tebakan jenis. 'rendah' = hanya arah uangnya yang pasti. */
export type Keyakinan = 'tinggi' | 'sedang' | 'rendah'

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
  /** Sudah ada transaksi dengan tanggal & nominal sama — dugaan, bukan kepastian. */
  duplikat: boolean
  pilih: boolean
}

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
}

export interface MutasiCommitResult {
  disimpan: number
  errors: ImportError[] | null
  contoh: Tx[] | null
}

/** Langkah 1: baca berkas & tebak jenisnya. Tidak menyimpan apa pun. */
export async function previewMutasi(file: File, tahun?: number): Promise<MutasiHasil> {
  const form = new FormData()
  form.append('file', file)
  if (tahun) form.append('tahun', String(tahun))
  const res = await fetch('/api/v1/transactions/mutasi', { method: 'POST', body: form })
  if (!res.ok) throw new ApiError(res.status, (await res.text()).trim() || res.statusText)
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
}

export interface Invoice extends InvoiceInput {
  id: number
  nomor: string
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
  facts: BusinessFacts
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
