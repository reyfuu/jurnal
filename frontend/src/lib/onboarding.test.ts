import { describe, expect, it } from 'vitest'
import { DEFAULT_ANSWERS, answered, steps, summary, toProfile, type Answers } from './onboarding'

const a = (x: Answers): Answers => ({ ...DEFAULT_ANSWERS, ...x })

describe('toProfile', () => {
  it('warung pribadi, menikah, 2 tanggungan', () => {
    expect(toProfile(a({ nama: '  Warung Bu Siti ', jenis: 'perdagangan', bentuk: 'op', menikah: 'sudah', tanggungan: '2' }))).toEqual({
      nama_usaha: 'Warung Bu Siti', alamat: '', kecamatan: '', kota: '', provinsi: '', jenis_usaha: 'perdagangan', bentuk_usaha: 'op', ptkp: 'K/2', pkp: false, metode_persediaan: 'periodik', omzet_tahunan: 0, laba_sebelum_pajak: 0,
    })
  })

  it('PTKP: belum menikah tanpa tanggungan = TK/0, penghasilan digabung = K/I/n', () => {
    expect(toProfile(a({ jenis: 'jasa', bentuk: 'op', menikah: 'belum', tanggungan: '0' })).ptkp).toBe('TK/0')
    expect(toProfile(a({ jenis: 'jasa', bentuk: 'op', menikah: 'sudah', tanggungan: '3', gabung: 'ya' })).ptkp).toBe('K/I/3')
  })

  it('belum menikah mengabaikan jawaban gabung', () => {
    expect(toProfile(a({ jenis: 'jasa', bentuk: 'op', menikah: 'belum', tanggungan: '1', gabung: 'ya' })).ptkp).toBe('TK/1')
  })

  it('"tidak tahu" dianggap usaha pribadi', () => {
    expect(toProfile(a({ jenis: 'jasa', bentuk: 'tidak_tahu', menikah: 'belum', tanggungan: '0' })).bentuk_usaha).toBe('op')
  })

  it('jasa tidak punya metode stok; badan usaha tidak punya PTKP', () => {
    const p = toProfile(a({ jenis: 'jasa', bentuk: 'cv', stok: 'perpetual', menikah: 'sudah', tanggungan: '2', pkp: 'ya' }))
    expect(p.metode_persediaan).toBe('')
    expect(p.ptkp).toBe('')
    expect(p.pkp).toBe(true)
  })
})

describe('lokasi', () => {
  it('dipetakan ke profil dan tampil di ringkasan', () => {
    const p = toProfile(a({ nama: 'X', alamat: ' Jl. Merdeka 1 ', kecamatan: 'Sukajadi', kota: 'Kota Bandung', provinsi: 'Jawa Barat', jenis: 'jasa', bentuk: 'pt' }))
    expect([p.alamat, p.kecamatan, p.kota, p.provinsi]).toEqual(['Jl. Merdeka 1', 'Sukajadi', 'Kota Bandung', 'Jawa Barat'])
    expect(summary(p)[0]).toEqual({ judul: 'Lokasi usaha', isi: 'Jl. Merdeka 1, Kec. Sukajadi, Kota Bandung, Jawa Barat.' })
  })
})

describe('omzet & laba', () => {
  it('parse rupiah berformat, minus = rugi', () => {
    const p = toProfile(a({ jenis: 'jasa', bentuk: 'pt', omzet: 'Rp1.250.000.000', laba: '-25.000.000' }))
    expect(p.omzet_tahunan).toBe(1_250_000_000)
    expect(p.laba_sebelum_pajak).toBe(-25_000_000)
  })
})

describe('steps', () => {
  it('pribadi yang jual barang dan sudah menikah melihat semua langkah', () => {
    expect(steps(a({ jenis: 'perdagangan', bentuk: 'op', menikah: 'sudah' }))).toEqual(
      ['nama', 'lokasi', 'jenis', 'stok', 'omzet', 'laba', 'bentuk', 'menikah', 'tanggungan', 'gabung', 'pkp', 'ringkasan'],
    )
  })

  it('jasa berbentuk PT melewati stok dan pertanyaan keluarga', () => {
    expect(steps(a({ jenis: 'jasa', bentuk: 'pt' }))).toEqual(['nama', 'lokasi', 'jenis', 'omzet', 'laba', 'bentuk', 'pkp', 'ringkasan'])
  })

  it('belum menikah melewati pertanyaan penghasilan pasangan', () => {
    expect(steps(a({ jenis: 'manufaktur', bentuk: 'tidak_tahu', menikah: 'belum' }))).not.toContain('gabung')
  })
})

describe('answered', () => {
  it('nama wajib diisi, pilihan wajib dipilih', () => {
    expect(answered('nama', a({ nama: '   ' }))).toBe(false)
    expect(answered('nama', a({ nama: 'Toko A' }))).toBe(true)
    expect(answered('jenis', a({}))).toBe(false)
    expect(answered('stok', a({}))).toBe(true) // default periodik
    expect(answered('lokasi', a({ alamat: 'Jl. A', kecamatan: 'B', kota: 'C' }))).toBe(false) // provinsi belum
    expect(answered('lokasi', a({ alamat: 'Jl. A', kecamatan: 'B', kota: 'C', provinsi: 'Bali' }))).toBe(true)
    expect(answered('omzet', a({ omzet: '0' }))).toBe(false)
    expect(answered('omzet', a({ omzet: 'Rp750.000.000' }))).toBe(true)
    expect(answered('laba', a({ laba: '0' }))).toBe(true) // impas boleh
    expect(answered('laba', a({}))).toBe(false)
  })
})

describe('summary', () => {
  it('menjelaskan akun, stok, pajak, dan PPN', () => {
    const s = summary(toProfile(a({ nama: 'X', jenis: 'perdagangan', bentuk: 'op', menikah: 'sudah', tanggungan: '2' })))
    expect(s.map((x) => x.judul)).toEqual(['Daftar akun', 'Pencatatan stok', 'Pajak penghasilan', 'PPN'])
    expect(s[2].isi).toContain('0,5%')
    expect(s[2].isi).toContain('K/2')
  })

  it('jasa tidak menampilkan baris stok', () => {
    const s = summary(toProfile(a({ nama: 'X', jenis: 'jasa', bentuk: 'pt' })))
    expect(s.map((x) => x.judul)).not.toContain('Pencatatan stok')
  })
})
