package finance

import "strings"

// EjaUtang: ejaan baku KBBI "utang" untuk nama akun lama yang masih memakai "hutang"
// (saldo awal tersimpan, template CoA terbitan). Kata "piutang" tidak tersentuh.
func EjaUtang(s string) string { return ejaUtang.Replace(s) }

var ejaUtang = strings.NewReplacer("HUTANG", "UTANG", "Hutang", "Utang", "hutang", "utang")
