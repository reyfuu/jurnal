package coa

import "strings"

// AkunPersediaan: akun persediaan (subtype "inventory") dalam urutan aslinya. Akun kontra
// (inventory_contra, mis. 1309) tidak dihitung.
func AkunPersediaan(akun []Account) []Account {
	var out []Account
	for _, a := range akun {
		if a.Subtype == "inventory" {
			out = append(out, a)
		}
	}
	return out
}

// AkunPersediaanBarang menebak kode akun persediaan dari teks kategori barang lama. Yang
// dicocokkan hanya akun dari AkunPersediaan; tak ada yang cocok -> akun peran persediaan.
func AkunPersediaanBarang(kategori string, akun []Account, peta Peta) string {
	kat := strings.ToLower(kategori)
	cari := func(kata ...string) string {
		for _, a := range AkunPersediaan(akun) {
			nama := strings.ToLower(a.Nama)
			for _, k := range kata {
				if strings.Contains(nama, k) {
					return a.Kode
				}
			}
		}
		return ""
	}
	var kode string
	switch {
	case strings.Contains(kat, "setengah jadi"):
		kode = cari("setengah jadi")
	case strings.Contains(kat, "dalam proses"):
		kode = cari("dalam proses")
	case strings.Contains(kat, "jadi"):
		kode = cari("barang jadi")
	case strings.Contains(kat, "penolong") || strings.Contains(kat, "kemasan"):
		kode = cari("penolong", "kemasan")
	case strings.Contains(kat, "bahan baku"):
		kode = cari("bahan baku")
	}
	if kode == "" {
		return peta.Kode(PeranPersediaan)
	}
	return kode
}
