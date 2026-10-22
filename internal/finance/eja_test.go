package finance

import "testing"

// Bugslist 22 Okt #2: ejaan baku KBBI "utang", bukan "hutang", di semua nama akun yang tampil.
func TestEjaUtang(t *testing.T) {
	for in, mau := range map[string]string{
		"Hutang Bank":             "Utang Bank",
		"HUTANG USAHA":            "UTANG USAHA",
		"sisa hutang supplier":    "sisa utang supplier",
		"Piutang Usaha":           "Piutang Usaha",
		"Utang Lain-lain":         "Utang Lain-lain",
		"Hutang Lain-lain/Hutang": "Utang Lain-lain/Utang",
	} {
		if got := EjaUtang(in); got != mau {
			t.Errorf("EjaUtang(%q) = %q, mau %q", in, got, mau)
		}
	}
}

// Template saldo awal & Excel tidak lagi memakai kata "Hutang".
func TestTemplateTanpaHutang(t *testing.T) {
	for _, r := range TemplateAkunSaldo() {
		if EjaUtang(r.Nama) != r.Nama {
			t.Errorf("template saldo awal masih %q", r.Nama)
		}
	}
}
