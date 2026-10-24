import type { Mirip } from '@/lib/api'

// Tindakan baris mutasi yang diduga sudah tercatat (Mirip): lewati, catat sebagai pelunasannya,
// "sama — pindah akun bayar" ke rekening mutasi ini, atau tetap catat sebagai transaksi baru.

export type AksiMutasi = 'lewati' | 'lunasi' | 'pindah' | 'baru'

type BarisAksi = { pilih: boolean; lunasi?: number; pindah?: number }

/** Akun bayar transaksi yang cocok boleh dipindah ke rekening: transaksi lunas biasa (bukan
 *  invoice, catatan multi-akun, atau Pindah Dana) yang akun bayarnya berbeda (aturan server sama). */
export const bolehPindah = (m: Mirip | undefined, rekening: string) =>
  !!m && !!rekening && !m.belum_lunas && !m.invoice_id && !m.grup && m.jenis !== 'Pindah Dana' && m.lawan !== rekening

export const aksiMutasi = (r: BarisAksi): AksiMutasi => (!r.pilih ? 'lewati' : r.lunasi ? 'lunasi' : r.pindah ? 'pindah' : 'baru')

/** Perubahan baris untuk aksi v atas transaksi m. */
export const patchAksi = (v: AksiMutasi, m: Mirip) => ({
  pilih: v !== 'lewati',
  lunasi: v === 'lunasi' ? m.id : 0,
  pindah: v === 'pindah' ? m.id : 0,
})

/** Rekening diganti: baris "pindah" yang tidak berlaku lagi kembali dilewati. */
export function sesuaikanRekening<T extends BarisAksi & { mirip?: Mirip }>(rows: T[], rekening: string): T[] {
  if (!rows.some((r) => r.pindah && !bolehPindah(r.mirip, rekening))) return rows
  return rows.map((r) => (r.pindah && !bolehPindah(r.mirip, rekening) ? { ...r, pilih: false, pindah: 0 } : r))
}
