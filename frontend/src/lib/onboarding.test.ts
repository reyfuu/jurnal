import { describe, expect, it } from 'vitest'
import { DEFAULT_ANSWERS, answered, steps, summary, toProfile, type Answers } from './onboarding'

const a = (x: Answers): Answers => ({ ...DEFAULT_ANSWERS, ...x })

describe('toProfile', () => {
  it('warung pribadi, menikah, 2 tanggungan', () => {
    expect(toProfile(a({ nama: '  Warung Bu Siti ', jenis: 'perdagangan', bentuk: 'op', menikah: 'sudah', tanggungan: '2' }))).toEqual({
      nama_usaha: 'Warung Bu Siti', jenis_usaha: 'perdagangan', bentuk_usaha: 'op', ptkp: 'K/2', pkp: false, metode_persediaan: 'periodik',
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

describe('steps', () => {
  it('pribadi yang jual barang dan sudah menikah melihat semua langkah', () => {
    expect(steps(a({ jenis: 'perdagangan', bentuk: 'op', menikah: 'sudah' }))).toEqual(
      ['nama', 'jenis', 'stok', 'bentuk', 'menikah', 'tanggungan', 'gabung', 'pkp', 'ringkasan'],
    )
  })

  it('jasa berbentuk PT melewati stok dan pertanyaan keluarga', () => {
    expect(steps(a({ jenis: 'jasa', bentuk: 'pt' }))).toEqual(['nama', 'jenis', 'bentuk', 'pkp', 'ringkasan'])
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
