// Pratinjau & langkah rencana untuk isian form yang disiapkan AI (lihat internal/ai/isiform.go).
import type { AIForm, AIIsiForm } from './api'
import { formatRp, formatTanggal } from './format'

export const JUDUL_FORM: Record<AIForm, string> = {
  transaksi: 'Catat transaksi', invoice: 'Buat invoice', stok: 'Barang baru', pemasok: 'Pemasok baru',
}

const BUKA: Record<AIForm, string> = {
  transaksi: 'Buka Transaksi, form catat transaksi baru',
  invoice: 'Buka Invoice, form invoice baru',
  stok: 'Buka Stok & Gudang, form barang baru',
  pemasok: 'Buka Produksi & Supplier, tab Supplier, form pemasok baru',
}

export type BarisPratinjau = { label: string; nilai: string }

/** Kolom yang akan diisi AI, dalam bahasa & format yang dilihat di form. Kolom kosong dilewati. */
export function pratinjau(a: AIIsiForm): BarisPratinjau[] {
  const rows: [string, string | false | undefined | 0][] = []
  switch (a.form) {
    case 'transaksi': {
      const t = a.isian
      rows.push(['Tanggal', t.tanggal && formatTanggal(t.tanggal)], ['Jenis', t.jenis], ['Nominal', !!t.nominal && formatRp(t.nominal)],
        ['Keterangan', t.keterangan], ['Pihak', t.pihak], ['Barang', t.item && (t.qty ? `${t.item} × ${t.qty}` : t.item)],
        ['Status bayar', t.lunas === undefined ? undefined : t.lunas ? 'Lunas' : 'Belum lunas'])
      break
    }
    case 'invoice': {
      const v = a.isian
      rows.push(['Tanggal', v.tanggal && formatTanggal(v.tanggal)], ['Jatuh tempo', v.jatuh_tempo && formatTanggal(v.jatuh_tempo)],
        ['Pelanggan', v.pelanggan_nama], ['Alamat', v.pelanggan_alamat])
      v.items?.forEach((it, i) => rows.push([`Item ${i + 1}`, `${it.deskripsi} — ${it.qty}${it.satuan ? ' ' + it.satuan : ''} × ${formatRp(it.harga_satuan)}`]))
      rows.push(['Diskon', !!v.diskon_rp && formatRp(v.diskon_rp)], ['Catatan', v.catatan])
      break
    }
    case 'stok': {
      const s = a.isian
      rows.push(['Nama barang', s.nama], ['Kategori', s.kategori], ['Satuan', s.satuan],
        ['Stok awal', !!s.jumlah && `${s.jumlah.toLocaleString('id-ID')}${s.satuan ? ' ' + s.satuan : ''}`],
        ['Stok minimum', !!s.stok_min && s.stok_min.toLocaleString('id-ID')], ['Harga pokok', !!s.harga_pokok && formatRp(s.harga_pokok)],
        ['Harga jual', !!s.harga_jual && formatRp(s.harga_jual)])
      break
    }
    case 'pemasok': {
      const p = a.isian
      rows.push(['Nama', p.nama], ['Kontak', p.kontak], ['Telepon', p.telepon], ['Email', p.email], ['Alamat', p.alamat],
        ['Tempo bayar', !!p.tempo_hari && `${p.tempo_hari} hari`])
      break
    }
  }
  return rows.filter((r): r is [string, string] => !!r[1]).map(([label, nilai]) => ({ label, nilai }))
}

/** Langkah yang ditampilkan di mode rencana sebelum pengguna menekan Jalankan. */
export function rencana(a: AIIsiForm): string[] {
  const kolom = [...new Set(pratinjau(a).map((r) => r.label.replace(/ \d+$/, '')))]
  return [BUKA[a.form], `Isi ${kolom.length} kolom: ${kolom.join(', ')}`, 'Anda periksa, lalu klik Simpan. Belum ada yang tersimpan sebelum itu.']
}
