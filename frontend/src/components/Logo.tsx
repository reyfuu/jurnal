// Logo Master Pajak (Brand Guideline v1.0 §02, docs/brand/). Monogram selalu dipakai bersama
// wordmark di materi publik; monogram sendiri hanya untuk ikon aplikasi, favicon, dan avatar.
// Warna dari token --logo-*: versi utama di latar terang, versi reverse di dalam .dark (latar navy).

/** Konstruksi hlm. 8 di grid 100: tebal garis 13, kaki di x 20 & 80, puncak y 24, lembah (50, 56).
 *  viewBox dipotong ke tepi goresan sehingga `size` = tinggi monogram. */
export function Monogram({ size = 24 }: { size?: number }) {
  return (
    <svg width={(size * 73) / 67} height={size} viewBox="13.5 17.5 73 67" fill="none" strokeWidth="13"
      strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" style={{ flexShrink: 0 }}>
      <path d="M20 78V24l30 32" stroke="var(--ink)" />
      <path d="M50 56l30-32v54" stroke="var(--yellow-500)" />
    </svg>
  )
}

/** `tinggi` = tinggi monogram S (minimum 24 px di layar). Jarak ke wordmark 0,22 S, wordmark Rubik Bold. */
export function Logo({ tinggi = 24 }: { tinggi?: number }) {
  return (
    <span role="img" aria-label="Master Pajak" style={{ display: 'inline-flex', alignItems: 'center', gap: tinggi * 0.22 }}>
      <Monogram size={tinggi} />
      <span aria-hidden="true" style={{ fontFamily: 'var(--font-heading)', fontWeight: 700, fontSize: tinggi * 0.95,
        lineHeight: 1, letterSpacing: '-0.01em', color: 'var(--ink)' }}>
        master<span style={{ color: 'var(--logo-pajak)' }}>pajak</span>
      </span>
    </span>
  )
}
