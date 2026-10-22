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

// Jurnal & laporan tidak boleh menulis kode akun langsung: semuanya lewat peran, supaya CoA
// dengan penomoran lain (manufaktur) tetap dijurnal ke akun yang benar.
func TestTidakAdaKodeAkunTetap(t *testing.T) {
	re := regexp.MustCompile(`"([0-9]{4})"`)
	for _, f := range []string{"journal.go", "reports.go"} {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range re.FindAllStringSubmatch(string(src), -1) {
			t.Errorf("%s menulis kode akun %s langsung; pakai peran (peran.go)", f, m[1])
		}
	}
	if kurang := PeranKurang(Master); len(kurang) > 0 {
		t.Errorf("Master tidak memegang peran: %v", kurang)
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

// Akun di template Saldo Awal harus akun CoA Master dengan nama yang sama.
func TestTemplateSaldoAwal_SesuaiCoA(t *testing.T) {
	for _, a := range finance.TemplateAkunSaldo() {
		if a.Kode == "" {
			continue // baris aset tetap (nilai buku dari sheet Aset)
		}
		acc, ok := Find(Master, a.Kode)
		if !ok || acc.Nama != a.Nama {
			t.Errorf("akun %s %q tidak sama dengan CoA Master (%q, ada=%v)", a.Kode, a.Nama, acc.Nama, ok)
		}
	}
}

// CoA manufaktur lolos validasi template dan setiap peran menunjuk akun yang memang ada.
func TestManufaktur_TemplateSah(t *testing.T) {
	if err := ValidateTemplate(Manufaktur); err != nil {
		t.Fatal(err)
	}
	peta := PetaDari(Manufaktur)
	for _, c := range []struct{ peran, kode string }{{PeranKas, "1102"}, {PeranBank, "1111"}, {PeranPinjaman, "2131"}, {PeranBebanAmortisasi, "6124"}} {
		if peta.Kode(c.peran) != c.kode {
			t.Errorf("peran %s = %s, want %s", c.peran, peta.Kode(c.peran), c.kode)
		}
	}
	for kode, peran := range peranManufaktur {
		if _, ok := Find(Manufaktur, kode); !ok {
			t.Errorf("peran %s menunjuk akun %s yang tidak ada", peran, kode)
		}
	}
	// 155 akun dari "Daftar COA Manufaktur.xlsx" + 3299 penampung selisih saldo awal.
	if len(Manufaktur) != 156 {
		t.Errorf("jumlah akun %d, file 155 + 3299", len(Manufaktur))
	}
}
