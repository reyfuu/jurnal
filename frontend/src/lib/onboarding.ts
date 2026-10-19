import type { BentukUsaha, BusinessProfile, JenisUsaha, Option } from './api'
import { formatRp } from './format'

// Onboarding bahasa awam: tiap jawaban dipetakan ke field BusinessProfile.
// Validasi akhir tetap di backend (finance.BusinessProfile.Validate).

export type Field = 'nama' | 'alamat' | 'kecamatan' | 'kota' | 'provinsi' | 'jenis' | 'stok' | 'omzet' | 'laba' | 'bentuk' | 'menikah' | 'tanggungan' | 'gabung' | 'pkp'
// lokasi: satu layar berisi alamat, kecamatan, kota, provinsi
export const LOKASI_FIELDS = ['alamat', 'kecamatan', 'kota', 'provinsi'] as const
export type StepId = Exclude<Field, (typeof LOKASI_FIELDS)[number]> | 'lokasi' | 'ringkasan'
export type Answers = Partial<Record<Field, string>>

// jawaban aman bila pengguna ragu
export const DEFAULT_ANSWERS: Answers = { stok: 'periodik', gabung: 'tidak', pkp: 'tidak' }

export interface Question { judul: string; alasan: string; options?: Option[] }

// parseRp: "Rp1.250.000" → 1250000; tanda minus di depan = rugi. Kosong = NaN.
export const parseRp = (s = '') => {
  const digits = s.replace(/\D/g, '')
  return digits ? Number(digits) * (s.trim().startsWith('-') ? -1 : 1) : NaN
}

export const QUESTIONS: Record<Exclude<StepId, 'ringkasan'>, Question> = {
  nama: {
    judul: 'Apa nama usaha Anda?',
    alasan: 'Nama ini muncul di laporan keuangan dan file Excel yang Anda unduh.',
  },
  lokasi: {
    judul: 'Di mana lokasi usaha Anda?',
    alasan: 'Lokasi dicetak di invoice dan laporan, dan menentukan kantor pajak (KPP) tempat usaha Anda terdaftar.',
  },
  jenis: {
    judul: 'Usaha Anda paling mirip yang mana?',
    alasan: 'Supaya daftar akun dan laporan keuangan yang kami siapkan sesuai dengan cara usaha Anda menghasilkan uang.',
    options: [
      { value: 'perdagangan', label: 'Saya menjual barang yang saya beli dari orang lain', hint: 'Warung, toko, reseller, distributor, minimarket' },
      { value: 'jasa', label: 'Saya menjual jasa atau keahlian, bukan barang', hint: 'Salon, laundry, servis, konsultan, agensi' },
      { value: 'manufaktur', label: 'Saya membuat atau mengolah barang sendiri lalu menjualnya', hint: 'Kue, catering, konveksi, kerajinan, kopi sangrai' },
    ],
  },
  stok: {
    judul: 'Bagaimana Anda tahu sisa stok barang?',
    alasan: 'Supaya laporan laba menghitung modal barang dengan cara yang sama seperti Anda mencatatnya.',
    options: [
      { value: 'periodik', label: 'Saya hitung stok sesekali, misalnya tiap akhir bulan', hint: 'Paling umum. Pilih ini kalau ragu.' },
      { value: 'perpetual', label: 'Setiap barang masuk dan keluar langsung saya catat', hint: 'Sisa stok selalu ketahuan tanpa menghitung ulang' },
    ],
  },
  omzet: {
    judul: 'Kira-kira berapa omzet usaha Anda per tahun?',
    alasan: 'Omzet adalah total penjualan sebelum dikurangi biaya. Angka ini menentukan batas bebas pajak (Rp500 juta untuk usaha pribadi) dan batas PPh Final UMKM (Rp4,8 miliar). Perkiraan kasar sudah cukup.',
  },
  laba: {
    judul: 'Kira-kira berapa laba bersih usaha per tahun, sebelum dipotong pajak?',
    alasan: 'Omzet dikurangi semua biaya usaha. Kami bandingkan dengan rasio benchmark DJP untuk usaha sejenis. Isi 0 bila impas, atau beri tanda minus bila rugi.',
  },
  bentuk: {
    judul: 'Usaha Anda terdaftar atas nama siapa?',
    alasan: 'Ini menentukan jenis pajak penghasilan yang berlaku untuk usaha Anda.',
    options: [
      { value: 'op', label: 'Atas nama saya pribadi', hint: 'Belum punya badan usaha, pakai NPWP/NIK sendiri' },
      { value: 'pt_perorangan', label: 'PT Perorangan', hint: 'Badan usaha yang saya dirikan sendiri' },
      { value: 'cv', label: 'CV', hint: 'Didirikan bersama rekan, ada akta' },
      { value: 'pt', label: 'PT', hint: 'Ada akta notaris dan pemegang saham' },
      { value: 'tidak_tahu', label: 'Saya tidak tahu', hint: 'Kami anggap atas nama pribadi; bisa diubah nanti' },
    ],
  },
  menikah: {
    judul: 'Apakah Anda sudah menikah?',
    alasan: 'Untuk usaha pribadi, status keluarga menentukan bagian penghasilan yang tidak kena pajak.',
    options: [
      { value: 'belum', label: 'Belum menikah' },
      { value: 'sudah', label: 'Sudah menikah' },
    ],
  },
  tanggungan: {
    judul: 'Berapa orang yang Anda tanggung biaya hidupnya?',
    alasan: 'Anak, orang tua, atau saudara yang tinggal bersama dan Anda biayai. Tiap orang (maksimal 3) menambah bagian yang tidak kena pajak.',
    options: [
      { value: '0', label: 'Tidak ada' },
      { value: '1', label: '1 orang' },
      { value: '2', label: '2 orang' },
      { value: '3', label: '3 orang atau lebih' },
    ],
  },
  gabung: {
    judul: 'Apakah penghasilan pasangan Anda dilaporkan bersama dalam satu NPWP?',
    alasan: 'Biasanya tidak. Ini hanya berlaku bila suami dan istri sama-sama berpenghasilan dan pajaknya digabung.',
    options: [
      { value: 'tidak', label: 'Tidak / tidak tahu' },
      { value: 'ya', label: 'Ya, digabung' },
    ],
  },
  pkp: {
    judul: 'Apakah Anda menarik PPN dari pembeli dan membuat faktur pajak?',
    alasan: 'Hanya usaha yang sudah dikukuhkan sebagai PKP yang wajib memungut PPN. Kebanyakan UMKM belum.',
    options: [
      { value: 'tidak', label: 'Tidak / belum' },
      { value: 'ya', label: 'Ya, usaha saya sudah PKP' },
    ],
  },
}

const pribadi = (a: Answers) => a.bentuk === 'op' || a.bentuk === 'tidak_tahu'
const jualBarang = (a: Answers) => a.jenis === 'perdagangan' || a.jenis === 'manufaktur'

export function steps(a: Answers): StepId[] {
  return [
    'nama',
    'lokasi',
    'jenis',
    ...(jualBarang(a) ? ['stok' as const] : []),
    'omzet',
    'laba',
    'bentuk',
    ...(pribadi(a) ? ['menikah' as const, 'tanggungan' as const] : []),
    ...(pribadi(a) && a.menikah === 'sudah' ? ['gabung' as const] : []),
    'pkp',
    'ringkasan',
  ]
}

export const answered = (s: StepId, a: Answers) =>
  s === 'ringkasan' || (s === 'lokasi' ? LOKASI_FIELDS.every((f) => Boolean(a[f]?.trim())) : s === 'omzet' ? parseRp(a.omzet) > 0 : s === 'laba' ? !Number.isNaN(parseRp(a.laba)) : Boolean(a[s]?.trim()))

export function toProfile(a: Answers): BusinessProfile {
  const jenis = a.jenis as JenisUsaha
  const bentuk = (pribadi(a) ? 'op' : a.bentuk) as BentukUsaha
  const n = Math.min(Number(a.tanggungan) || 0, 3)
  const status = a.menikah !== 'sudah' ? 'TK' : a.gabung === 'ya' ? 'K/I' : 'K'
  return {
    nama_usaha: (a.nama ?? '').trim(),
    alamat: (a.alamat ?? '').trim(),
    kecamatan: (a.kecamatan ?? '').trim(),
    kota: (a.kota ?? '').trim(),
    provinsi: a.provinsi ?? '',
    jenis_usaha: jenis,
    bentuk_usaha: bentuk,
    ptkp: bentuk === 'op' ? `${status}/${n}` : '',
    pkp: a.pkp === 'ya',
    metode_persediaan: jenis === 'jasa' ? '' : ((a.stok as BusinessProfile['metode_persediaan']) || 'periodik'),
    omzet_tahunan: parseRp(a.omzet) || 0,
    laba_sebelum_pajak: parseRp(a.laba) || 0,
  }
}

const PAJAK: Record<BentukUsaha, string> = {
  op: 'PPh Final 0,5% dari omzet. Omzet sampai Rp500 juta per tahun bebas pajak.',
  pt_perorangan: 'PPh Final 0,5% dari seluruh omzet (berlaku untuk badan usaha, tanpa batas bebas Rp500 juta).',
  cv: 'PPh Badan dengan potongan tarif 50% (Pasal 31E) untuk omzet sampai Rp4,8 miliar per tahun.',
  pt: 'PPh Badan dengan potongan tarif 50% (Pasal 31E) untuk omzet sampai Rp4,8 miliar per tahun, ditambah pajak dividen saat laba dibagikan.',
}

export function summary(p: BusinessProfile): { judul: string; isi: string }[] {
  return [
    ...(p.kota
      ? [{ judul: 'Lokasi usaha', isi: [p.alamat, p.kecamatan && `Kec. ${p.kecamatan}`, p.kota, p.provinsi].filter(Boolean).join(', ') + '.' }]
      : []),
    {
      judul: 'Daftar akun',
      isi: p.jenis_usaha === 'perdagangan'
        ? 'Daftar akun usaha dagang kami siapkan otomatis. Setiap transaksi langsung masuk ke akun yang tepat.'
        : 'Laporan laba rugi dan neraca dasar siap dipakai. Daftar akun khusus untuk jenis usaha ini sedang kami siapkan.',
    },
    ...(p.omzet_tahunan
      ? [{ judul: 'Omzet & laba per tahun', isi: `Omzet ${formatRp(p.omzet_tahunan)}, laba sebelum pajak ${formatRp(p.laba_sebelum_pajak ?? 0)}.` }]
      : []),
    ...(p.metode_persediaan
      ? [{
          judul: 'Pencatatan stok',
          isi: p.metode_persediaan === 'perpetual'
            ? 'Perpetual: modal barang tercatat setiap kali barang terjual.'
            : 'Periodik: modal barang dihitung dari pembelian dan hitung stok di akhir periode.',
        }]
      : []),
    { judul: 'Pajak penghasilan', isi: PAJAK[p.bentuk_usaha] + (p.ptkp ? ` Status PTKP Anda: ${p.ptkp}.` : '') + ' Perbandingan skema lengkap ada di menu Perencanaan Pajak.' },
    { judul: 'PPN', isi: p.pkp ? 'Usaha Anda memungut PPN; akun PPN masukan dan keluaran disiapkan.' : 'Usaha Anda belum memungut PPN.' },
  ]
}
