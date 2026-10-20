package coa

import (
	"errors"
	"os"
	"regexp"
	"slices"
	"testing"

	"pajak/internal/finance"
)

func TestActiveFrom_SamaDenganActiveUntukMaster(t *testing.T) {
	for _, p := range []finance.BusinessProfile{profile("op", "periodik"), profile("pt", "perpetual")} {
		if got, want := ActiveFrom(Master, p), Active(p); !slices.Equal(got, want) {
			t.Errorf("%+v: ActiveFrom(Master) beda dengan Active", p)
		}
	}
}

// Setiap kode akun yang ditulis langsung di jurnal & laporan wajib ada di template terbit;
// kalau tidak, jurnal otomatis memakai akun yang tidak ada.
func TestKodeWajib_MencakupKodeDiKode(t *testing.T) {
	wajib := KodeWajib()
	re := regexp.MustCompile(`"([0-9]{4})"`)
	for _, f := range []string{"journal.go", "reports.go"} {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range re.FindAllStringSubmatch(string(src), -1) {
			if !slices.Contains(wajib, m[1]) {
				t.Errorf("%s memakai akun %s yang tidak ada di KodeWajib", f, m[1])
			}
		}
	}
	for _, k := range wajib {
		if _, ok := Find(Master, k); !ok {
			t.Errorf("kode wajib %s tidak ada di Master", k)
		}
	}
}

func TestValidateTemplate(t *testing.T) {
	if err := ValidateTemplate(Master); err != nil {
		t.Fatalf("Master harus valid: %v", err)
	}
	tanpaKas := slices.DeleteFunc(slices.Clone(Master), func(a Account) bool { return a.Kode == "1101" })
	dobel := append(slices.Clone(Master), Master[0])
	salahKelompok := slices.Clone(Master)
	salahKelompok[0].Kelompok = "Aset Ajaib"
	salahKode := append(slices.Clone(Master), Account{Kode: "12A", Nama: "X", Subtype: "opex", Normal: Debit, Kelompok: BebanOperasional, Kondisi: Selalu})
	tanpaNama := append(slices.Clone(Master), Account{Kode: "6199", Subtype: "opex", Normal: Debit, Kelompok: BebanOperasional, Kondisi: Selalu})
	for nama, accs := range map[string][]Account{"tanpa kas": tanpaKas, "dobel": dobel, "kelompok": salahKelompok, "kode": salahKode, "nama": tanpaNama} {
		if err := ValidateTemplate(accs); !errors.Is(err, ErrTemplate) {
			t.Errorf("%s: err = %v", nama, err)
		}
	}
	tambah := append(slices.Clone(Master), Account{Kode: "6119", Nama: "Beban Asuransi", Subtype: "opex", Normal: Debit, Kelompok: BebanOperasional, Kondisi: Selalu})
	if err := ValidateTemplate(tambah); err != nil {
		t.Errorf("tambah akun: %v", err)
	}
}

// Neraca saldo mengikuti urutan template, termasuk akun yang hanya ada di template.
func TestTrialBalanceFrom_AkunTemplate(t *testing.T) {
	tmpl := append(slices.Clone(Master), Account{Kode: "6119", Nama: "Beban Asuransi", Subtype: "opex", Normal: Debit, Kelompok: BebanOperasional, Kondisi: Selalu})
	p := profile("op", "periodik")
	tb := TrialBalanceFrom(tmpl, ActiveFrom(tmpl, p), nil, []Line{{Kode: "6119", Debit: 500}, {Kode: "1101", Kredit: 500}})
	i := slices.IndexFunc(tb, func(b Balance) bool { return b.Kode == "6119" })
	if i < 0 || tb[i].Saldo != 500 {
		t.Fatalf("akun template tidak muncul: %+v", tb)
	}
}
