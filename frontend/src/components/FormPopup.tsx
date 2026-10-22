import type { ReactNode } from 'react'
import { Dialog, DialogContent, DialogDescription, DialogTitle } from '@/components/ui/dialog'

/**
 * Form tambah/ubah sebagai jendela di tengah layar (latar digelapkan), bukan kotak di bawah
 * daftar yang mudah terlewat saat layar di-zoom. Judul tampil dari isi form; judul di sini hanya
 * untuk pembaca layar.
 */
export function FormPopup({ judul, onClose, lebar = 'sm:max-w-3xl', children }: { judul: string; onClose: () => void; lebar?: string; children: ReactNode }) {
  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent className={lebar}>
        <DialogTitle className="sr-only">{judul}</DialogTitle>
        <DialogDescription className="sr-only">Isi lalu simpan, atau tutup untuk batal.</DialogDescription>
        {children}
      </DialogContent>
    </Dialog>
  )
}
