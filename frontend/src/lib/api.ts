// Tipe mengikuti struct JSON di internal/finance dan internal/auth (Go).

export type View =
  | 'dashboard' | 'laporan-keuangan' | 'arus-kas' | 'transaksi' | 'piutang-utang' | 'stok-gudang'
  | 'produksi-supplier' | 'invoice' | 'simulasi-harga' | 'pajak-umkm' | 'master-data' | 'panduan' | 'pengguna'
  | 'transaksi-kelola' // izin, bukan menu: ubah & hapus transaksi

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

export interface Dashboard {
  tenant: { name: string; initials: string; category: string; total_transactions: number }
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
  opex: number
  other_income: number
  net_profit: number
  margin_bps: number
  opex_lines: { label: string; amount: number }[] | null
}

export interface CashFlow {
  opening_balance: number
  cash_in: number
  cash_out: number
  net_cash: number
  closing_balance: number
  new_receivables: number
  new_payables: number
  non_operating: number
}

export interface BalanceSheet {
  assets: { cash: number; receivables: number; inventory: number; fixed_assets: number; total: number }
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
}

export type TxInput = Omit<Tx, 'id'>

export interface PriceSim {
  recommended_price: number
  channel_fee: number
  net_take_home: number
}

export interface AppData {
  dash: Dashboard
  report?: IncomeStatement
  cash?: CashFlow
  balanceSheet?: BalanceSheet
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
export const logout = () => api<void>('/auth/logout', { method: 'POST' })

/** Ambil dashboard + laporan yang boleh diakses role ini (laporan lain 403 untuk Kasir). */
export async function loadAll(views: View[], period = '2026-08'): Promise<AppData> {
  const can = (v: View) => views.includes(v)
  const q = `?period=${encodeURIComponent(period)}`
  const [dash, report, cash, balanceSheet] = await Promise.all([
    api<Dashboard>('/dashboard' + q),
    can('laporan-keuangan') || can('arus-kas') ? api<IncomeStatement>('/reports/financial' + q) : undefined,
    can('arus-kas') ? api<CashFlow>('/reports/cashflow' + q) : undefined,
    can('laporan-keuangan') ? api<BalanceSheet>('/reports/balance-sheet' + q) : undefined,
  ])
  return { dash, report, cash, balanceSheet }
}

export const listTransactions = () => api<{ transactions: Tx[]; jenis: JenisInfo[]; can_edit: boolean }>('/transactions')
export const createTransaction = (t: TxInput) => api<Tx>('/transactions', { method: 'POST', body: t })
export const updateTransaction = (id: number, t: TxInput) => api<Tx>(`/transactions/${id}`, { method: 'PUT', body: t })
export const deleteTransaction = (id: number) => api<void>(`/transactions/${id}`, { method: 'DELETE' })

export const simulatePrice = (cogs: number, targetProfit: number, feePct: number) =>
  api<PriceSim>('/pricing/simulate', {
    method: 'POST',
    body: { cogs, target_profit: targetProfit, fee_bps: Math.round(feePct * 100), fixed_fee: 0 },
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

