package tax

import (
	"slices"
	"testing"
)

func kodeKandidat(ks []KandidatKLU) []string {
	out := make([]string, len(ks))
	for i, k := range ks {
		out[i] = k.Kode
	}
	return out
}

func TestContohKLU_DariUraianLimaDigit(t *testing.T) {
	c := ContohKLU("563", 3)
	if len(c) != 3 || c[0] != "Bar" || c[1] != "Rumah minum/kafe" {
		t.Fatalf("contoh 563 = %v", c)
	}
	if len(ContohKLU("999", 3)) != 0 {
		t.Error("KLU tak dikenal punya contoh")
	}
}

func TestCariKLU_BahasaSehariHari(t *testing.T) {
	for _, c := range []struct{ teks, jenis, want string }{
		{"jual kopi susu di kafe kecil", "", "563"},
		{"laundry kiloan", "jasa", "962"},
		{"jualan baju di shopee", "perdagangan", "479"},
		{"catering nasi kotak untuk acara", "", "562"},
		{"bengkel mobil", "jasa", "452"},
		{"barbershop", "", "961"},
		{"saya jual skincare di marketplace", "perdagangan", "479"},
		{"jual kopi susu di kafe kecil", "perdagangan", "563"},
		{"distributor kopi ke kafe-kafe", "perdagangan", "463"},
		{"jual skincare di toko sendiri", "perdagangan", "477"},
		{"bikin kue kering untuk dijual", "manufaktur", "107"},
	} {
		got := kodeKandidat(CariKLU(c.teks, c.jenis, 5))
		if !slices.Contains(got[:min(3, len(got))], c.want) {
			t.Errorf("%q → %v, want %s di 3 teratas", c.teks, got, c.want)
		}
	}
	if got := kodeKandidat(CariKLU("laundry kiloan dekat kampus, bisa antar jemput", "jasa", 5)); len(got) != 1 || got[0] != "962" {
		t.Errorf("kecocokan kebetulan ikut tampil: %v", got)
	}
	for _, x := range kodeKandidat(CariKLU("saya jual skincare di marketplace", "perdagangan", 5)) {
		if x == "182" || x == "268" {
			t.Errorf("jual online tidak boleh jatuh ke reproduksi/industri media: %s", x)
		}
	}
	if got := CariKLU("di dan yang", "", 5); len(got) != 0 {
		t.Errorf("kata umum saja memberi hasil %v", kodeKandidat(got))
	}
}

func TestCariKLU_TanpaJenisDitebakDariKataKerja(t *testing.T) {
	for teks, want := range map[string]string{
		"saya bikin kue kering dan bolu sendiri lalu jual di toko dan online": "manufaktur",
		"bikin kopi susu sendiri, dijual di kafe":                             "jasa",
		"saya ternak ayam dan jual ayam":                                      "manufaktur",
		"tanam sayur hidroponik":                                              "manufaktur",
		"budidaya ikan lele":                                                  "manufaktur",
		"beli ayam dari peternak lalu jual lagi":                              "perdagangan",
		"jual ayam potong di pasar":                                           "perdagangan",
		"jasa servis AC dan kulkas panggilan":                                 "jasa",
	} {
		k := CariKLU(teks, "", 3)
		if len(k) == 0 || k[0].Jenis != want {
			t.Errorf("CariKLU(%q) = %+v; want jenis %s", teks, k, want)
		}
	}
}

// Saran tanpa jenis untuk berbagai macam usaha: kode teratas harus salah satu dari want, dan
// jenisnya harus jenis itu. Daftar ini sengaja lebar, karena kesalahan pencarian kata biasanya
// muncul di bidang yang tidak terpikirkan saat aturan ditulis.
func TestCariKLU_BerbagaiIndustri(t *testing.T) {
	for _, c := range []struct {
		teks, jenis string
		kode        []string
	}{
		// perdagangan
		{"jual beli metal detector", "perdagangan", []string{"477"}},
		{"jual metal detector online", "perdagangan", []string{"479"}},
		{"toko alat listrik", "perdagangan", []string{"475", "477"}},
		{"jual hp dan aksesoris", "perdagangan", []string{"474"}},
		{"jual sparepart motor", "perdagangan", []string{"454"}},
		{"jual pakaian di shopee", "perdagangan", []string{"479"}},
		{"reseller skincare", "perdagangan", []string{"477", "479"}},
		{"toko bangunan", "perdagangan", []string{"475", "471"}},
		{"jual beli mobil bekas", "perdagangan", []string{"451"}},
		{"dealer motor", "perdagangan", []string{"454"}},
		{"apotek", "perdagangan", []string{"477"}},
		{"toko obat", "perdagangan", []string{"477"}},
		{"warung sembako", "perdagangan", []string{"471", "472", "478"}},
		{"minimarket", "perdagangan", []string{"471"}},
		{"toko emas", "perdagangan", []string{"477"}},
		{"jual beli emas", "perdagangan", []string{"477"}},
		{"jual furniture", "perdagangan", []string{"475"}},
		{"jual kaos sablon", "perdagangan", []string{"477", "471", "478", "479"}},
		{"jual frozen food", "perdagangan", []string{"472", "471", "478"}},
		{"jual beras grosir", "perdagangan", []string{"463", "462"}},
		{"distributor minuman", "perdagangan", []string{"463"}},
		{"jual pupuk dan bibit", "perdagangan", []string{"477", "475"}},
		{"jual mainan anak", "perdagangan", []string{"476"}},
		{"toko buku", "perdagangan", []string{"476"}},
		{"jual ikan hias", "perdagangan", []string{"477", "478"}},
		{"jual hewan peliharaan kucing", "perdagangan", []string{"477", "478"}},
		{"petshop", "perdagangan", []string{"477"}},
		{"jual tanaman hias", "perdagangan", []string{"477", "478"}},
		{"jual alat musik", "perdagangan", []string{"476"}},
		{"jual komputer laptop", "perdagangan", []string{"474"}},
		{"jual pulsa dan token listrik", "perdagangan", []string{"474", "477", "475"}},
		{"beli ayam dari peternak lalu jual lagi", "perdagangan", []string{"472", "478", "477", "462"}},
		{"jual ayam potong di pasar", "perdagangan", []string{"478"}},
		{"jual pakan ternak", "perdagangan", []string{"477", "475", "478"}},
		{"pom mini bensin eceran", "perdagangan", []string{"473"}},
		// produksi / budidaya
		{"bikin furniture kayu jati", "manufaktur", []string{"310"}},
		{"konveksi kaos sablon", "manufaktur", []string{"141", "143"}},
		{"produksi keripik pisang", "manufaktur", []string{"107", "103"}},
		{"saya bikin kue kering dan bolu sendiri lalu jual di toko dan online", "manufaktur", []string{"107"}},
		{"percetakan undangan", "manufaktur", []string{"181"}},
		{"bikin sambal kemasan", "manufaktur", []string{"107"}},
		{"saya ternak ayam dan jual ayam", "manufaktur", []string{"014"}},
		{"tanam sayur hidroponik", "manufaktur", []string{"011"}},
		{"budidaya ikan lele", "manufaktur", []string{"032"}},
		{"tambak udang", "manufaktur", []string{"032"}},
		{"kebun sawit", "manufaktur", []string{"012"}},
		// jasa
		{"bengkel motor", "jasa", []string{"454"}},
		{"service laptop", "jasa", []string{"951"}},
		{"jasa desain grafis", "jasa", []string{"741"}},
		{"fotografer pernikahan", "jasa", []string{"742"}},
		{"travel umroh", "jasa", []string{"791"}},
		{"jasa ekspedisi", "jasa", []string{"522", "532"}},
		{"rental mobil", "jasa", []string{"771"}},
		{"kos-kosan", "jasa", []string{"551", "559"}},
		{"klinik gigi", "jasa", []string{"862"}},
		{"salon kecantikan", "jasa", []string{"961", "963"}},
		{"laundry kiloan", "jasa", []string{"962"}},
		{"kafe kopi", "jasa", []string{"563"}},
		{"bikin kopi susu sendiri, dijual di kafe", "jasa", []string{"563"}},
		{"warung makan padang", "jasa", []string{"561"}},
		{"catering nasi kotak", "jasa", []string{"562"}},
		{"jasa kontraktor bangunan", "jasa", []string{"410", "429", "433"}},
		{"jasa cleaning service", "jasa", []string{"812"}},
		{"software house aplikasi", "jasa", []string{"582", "620"}},
		{"jasa servis AC dan kulkas panggilan", "jasa", []string{"952", "432", "331"}},
	} {
		k := CariKLU(c.teks, "", 3)
		if len(k) == 0 || k[0].Jenis != c.jenis || !slices.Contains(c.kode, k[0].Kode) {
			t.Errorf("%-40q → %v; want %s %v", c.teks, ringkasKandidat(k), c.jenis, c.kode)
		}
	}
}

func ringkasKandidat(ks []KandidatKLU) []string {
	out := make([]string, len(ks))
	for i, k := range ks {
		out[i] = k.Kode + " " + k.Jenis + " " + k.Nama
	}
	return out
}
