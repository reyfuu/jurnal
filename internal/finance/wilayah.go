package finance

import (
	_ "embed"
	"encoding/json"
	"errors"
	"slices"
)

// Data wilayah administrasi: Kepmendagri No. 300.2.2-2138 Tahun 2025 (38 provinsi, 514
// kabupaten/kota, 7.285 kecamatan), diolah dari github.com/yonatanyl/KODE-WILAYAH-KEPMENDAGRI-2025
// (CC BY 4.0). Dibuat ulang dengan scripts/gen_wilayah.py.

//go:embed wilayah.json
var wilayahJSON []byte

type Kota struct {
	Nama      string   `json:"nama"`
	Kecamatan []string `json:"kecamatan"`
}

var wilayah = func() map[string][]Kota {
	var m map[string][]Kota
	if err := json.Unmarshal(wilayahJSON, &m); err != nil {
		panic("wilayah.json rusak: " + err.Error())
	}
	return m
}()

// KotaDi: kabupaten/kota di provinsi (urut nama); nil bila provinsi tidak dikenal.
func KotaDi(provinsi string) []Kota { return wilayah[provinsi] }

// ValidateWilayah: kota harus ada di provinsi dan kecamatan di kota; boleh kosong. Dipisah dari
// Validate supaya profil lama berisi teks bebas tetap bisa dimuat, dan baru wajib dipilih ulang
// dari daftar saat disimpan.
func (p BusinessProfile) ValidateWilayah() error {
	if p.Kota == "" {
		if p.Kecamatan != "" {
			return errors.New("pilih kota/kabupaten dulu sebelum kecamatan")
		}
		return nil
	}
	i := slices.IndexFunc(KotaDi(p.Provinsi), func(k Kota) bool { return k.Nama == p.Kota })
	if i < 0 {
		return errors.New("pilih kota/kabupaten dari daftar provinsi " + p.Provinsi)
	}
	if p.Kecamatan != "" && !slices.Contains(wilayah[p.Provinsi][i].Kecamatan, p.Kecamatan) {
		return errors.New("pilih kecamatan dari daftar " + p.Kota)
	}
	return nil
}
