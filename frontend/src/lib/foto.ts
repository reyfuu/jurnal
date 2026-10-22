// Foto nota dikecilkan di browser sebelum dikirim ke AI: cepat diunggah dan hemat kuota.

const SISI_MAKS = 1600

/** Perkecil foto (maks 1600 px, JPEG) supaya cepat diunggah dan hemat kuota AI. */
export async function kecilkan(file: File): Promise<Blob> {
  const url = URL.createObjectURL(file)
  try {
    const img = await new Promise<HTMLImageElement>((ok, gagal) => {
      const i = new Image()
      i.onload = () => ok(i)
      i.onerror = () => gagal(new Error('Foto tidak bisa dibuka. Pakai JPG atau PNG.'))
      i.src = url
    })
    const skala = Math.min(1, SISI_MAKS / Math.max(img.width, img.height))
    const c = document.createElement('canvas')
    c.width = Math.round(img.width * skala)
    c.height = Math.round(img.height * skala)
    c.getContext('2d')!.drawImage(img, 0, 0, c.width, c.height)
    return await new Promise<Blob>((ok, gagal) => c.toBlob((b) => (b ? ok(b) : gagal(new Error('Foto gagal diproses'))), 'image/jpeg', 0.85))
  } finally {
    URL.revokeObjectURL(url)
  }
}

