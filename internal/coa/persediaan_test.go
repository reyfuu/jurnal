package coa

import "testing"

func TestAkunPersediaanBarang_Manufaktur(t *testing.T) {
	peta := PetaDari(Manufaktur)
	for kat, mau := range map[string]string{
		"Bahan Baku": "1301", "bahan penolong": "1302", "Kemasan": "1302",
		"Barang Setengah Jadi": "1303", "Barang Dalam Proses": "1304",
		"Produk Jadi": "1305", "Barang Jadi": "1305", "Barang": "1301", "": "1301",
	} {
		if got := AkunPersediaanBarang(kat, Manufaktur, peta); got != mau {
			t.Errorf("%q → %s, mau %s", kat, got, mau)
		}
	}
	if n := len(AkunPersediaan(Manufaktur)); n != 7 { // 1301 1302 1303 1304 1305 1306 1307
		t.Errorf("akun persediaan manufaktur %d, mau 7", n)
	}
}

func TestAkunPersediaanBarang_Dagang(t *testing.T) {
	peta := PetaDari(Master)
	for _, kat := range []string{"Bahan Baku", "Produk Jadi", "Kemasan", "Barang"} {
		if got := AkunPersediaanBarang(kat, Master, peta); got != peta.Kode(PeranPersediaan) {
			t.Errorf("dagang %q → %s, mau akun peran", kat, got)
		}
	}
}
