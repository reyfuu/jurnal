import type { ReactNode } from 'react'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'

// Versi kebijakan; samakan dengan versiPDP di cmd/server/main.go dan naikkan bila isinya berubah.
export const VERSI_PDP = '2026-10-22'
// TODO(owner): isi alamat email resmi untuk permintaan hak subjek data sebelum rilis.
const KONTAK_PRIVASI = '[email kontak privasi]'

const BAGIAN: [string, ReactNode][] = [
  ['Data yang kami kumpulkan', <>Nama, email, nomor WhatsApp, dan password (disimpan dalam bentuk hash) saat Anda mendaftar; profil usaha; serta data keuangan yang Anda catat atau impor (transaksi, aset, invoice, stok, karyawan), termasuk foto nota yang Anda simpan sebagai bukti transaksi.</>],
  ['Tujuan pemrosesan', <>Menyediakan akun dan laporan keuangan, menghitung perkiraan pajak, memproses pembayaran langganan, mengirim pemberitahuan terkait akun dan kewajiban pajak lewat aplikasi atau WhatsApp, serta menangani laporan bug.</>],
  ['Dasar pemrosesan', <>Persetujuan Anda saat mendaftar dan pelaksanaan perjanjian layanan (Pasal 20 UU No. 27 Tahun 2022 tentang Pelindungan Data Pribadi).</>],
  ['Pihak yang memproses data', <>Midtrans (pembayaran: nama dan email), penyedia server tempat aplikasi dan basis data berjalan, serta penyedia model AI bila Anda memakai fitur AI: ringkasan angka laporan untuk analisis, keterangan mutasi bank untuk pengelompokan (angka panjang disamarkan), serta foto nota dan halaman mutasi hasil scan yang Anda unggah untuk dibaca menjadi isian transaksi. Data tidak dijual kepada pihak mana pun.</>],
  ['Penyimpanan', <>Data disimpan selama akun aktif. Foto nota yang disimpan sebagai bukti ikut terhapus bila catatannya dihapus. Setelah akun dihapus, data dihapus kecuali yang wajib disimpan menurut peraturan perpajakan (dokumen pembukuan 10 tahun, UU KUP Pasal 28).</>],
  ['Hak Anda', <>Mendapat informasi, mengakses dan meminta salinan data, membetulkan data, meminta penghapusan, menarik persetujuan, dan mengajukan keberatan (Pasal 5–13 UU PDP). Ajukan lewat {KONTAK_PRIVASI}; kami tanggapi paling lambat 3 × 24 jam.</>],
  ['Keamanan', <>Koneksi terenkripsi (HTTPS), password di-hash, akses dibatasi per usaha dan per peran. Bila terjadi kegagalan pelindungan data, kami memberi tahu Anda paling lambat 3 × 24 jam (Pasal 46 UU PDP).</>],
]

/** Dialog Kebijakan Privasi; `children` menjadi pemicunya (tautan/tombol). */
export function KebijakanPrivasi({ children }: { children: ReactNode }) {
  return (
    <Dialog>
      <DialogTrigger asChild>{children}</DialogTrigger>
      <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Kebijakan Privasi</DialogTitle>
          <DialogDescription>Master Pajak · versi {VERSI_PDP}</DialogDescription>
        </DialogHeader>
        <dl className="grid gap-4 text-sm">
          {BAGIAN.map(([judul, isi]) => (
            <div key={judul} className="grid gap-1">
              <dt className="font-semibold">{judul}</dt>
              <dd className="text-muted-foreground">{isi}</dd>
            </div>
          ))}
        </dl>
      </DialogContent>
    </Dialog>
  )
}
