package tax

import "errors"

// Batas masukan kalkulator & profil: margin laba (laba bersih ÷ omzet) di antara -100% dan
// 100%. Frontend memakai teks yang sama
// (frontend/src/lib/kalkulator.ts) supaya pengguna diminta merevisi sebelum menekan Hitung.
var (
	ErrLabaMelebihiOmzet = errors.New("laba bersih tidak boleh lebih besar dari omzet: margin laba harus di antara -100% dan 100%, periksa lagi angka laba dan omzet")
	ErrRugiMelebihiOmzet = errors.New("kerugian tidak boleh lebih besar dari omzet (margin di bawah -100%), periksa lagi angka laba dan omzet")
	ErrMargin            = errors.New("margin laba harus di antara -100% dan 100%")
)

// CekLabaOmzet: laba bersih (minus = rugi) harus di antara -omzet dan omzet.
func CekLabaOmzet(laba, omzet int64) error {
	switch {
	case laba > omzet:
		return ErrLabaMelebihiOmzet
	case laba < -omzet:
		return ErrRugiMelebihiOmzet
	}
	return nil
}

// CekMarginBps: margin laba dalam basis poin (10.000 = 100%).
func CekMarginBps(bps int64) error {
	if bps < -10_000 || bps > 10_000 {
		return ErrMargin
	}
	return nil
}
