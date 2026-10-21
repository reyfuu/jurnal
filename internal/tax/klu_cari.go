package tax

import (
	"cmp"
	"math"
	"slices"
	"strings"
	"unicode"
)

// Pencarian KLU dari bahasa sehari-hari ("jual kopi susu di kafe", "laundry kiloan"). Nama KLU
// 3 digit sering terlalu umum bagi orang awam, jadi yang dicari juga uraian KBLI 5 digit di
// bawahnya (tabel norma), dengan bobot kata langka (IDF) supaya "kafe" lebih menentukan
// daripada "minuman". Hasilnya kandidat; saran AI memilih dari daftar penuh di internal/ai.

type KandidatKLU struct {
	KLUBenchmark
	Jenis  string   `json:"jenis"`
	Contoh []string `json:"contoh"`
	Skor   float64  `json:"-"`
}

// sinonim: kata sehari-hari → kata yang dipakai uraian KBLI.
var sinonim = map[string][]string{
	"laundry": {"binatu"}, "londri": {"binatu"}, "cuci": {"binatu", "pencucian"},
	"cafe": {"kafe", "minuman"}, "kafe": {"minuman"}, "coffee": {"kafe", "minuman"}, "ngopi": {"kafe", "minuman"}, "kopi": {"minuman"},
	"skincare": {"kosmetik"}, "makeup": {"kosmetik"}, "kosmetika": {"kosmetik"}, "serum": {"kosmetik"},
	"parfum": {"kosmetik"}, "kecantikan": {"kosmetik"},
	"catering": {"boga"}, "katering": {"boga", "catering"},
	"bengkel": {"reparasi"}, "servis": {"reparasi"}, "service": {"reparasi"},
	"barbershop": {"pangkas", "rambut"}, "barber": {"pangkas", "rambut"}, "potong": {"pangkas"},
	"warung": {"kedai", "eceran"}, "baju": {"pakaian"}, "fashion": {"pakaian"}, "kue": {"roti"},
	"homestay": {"pondok"}, "konsultan": {"konsultasi"},
}

var kataUmum = map[string]bool{
	"di": true, "dan": true, "yang": true, "untuk": true, "ke": true, "dari": true, "saya": true, "kami": true,
	"jual": true, "jualan": true, "menjual": true, "usaha": true, "bisnis": true, "kecil": true, "dengan": true,
	"atau": true, "buat": true, "acara": true, "punya": true, "kiloan": true,
}

// jualOnline: kata yang langsung menunjuk KLU 479 (eceran melalui media / bukan di toko).
// Tidak lewat sinonim "media", karena "media" juga ada di "reproduksi media rekaman".
var jualOnline = []string{"online", "olshop", "shopee", "tokopedia", "tiktok", "lazada", "bukalapak", "marketplace", "ecommerce", "daring", "instagram", "whatsapp"}

// produksi: kata yang membuat KLU industri/pengolahan (golongan 10–33) masuk akal.
var produksi = []string{"produksi", "pabrik", "bikin", "membuat", "olah", "mengolah", "industri", "konveksi", "jahit", "masak", "roasting", "sangrai", "cetak", "rakit"}

// jasaKata: kata yang menunjukkan yang dijual adalah tenaga / keahlian.
var jasaKata = []string{"jasa", "servis", "service", "reparasi", "perbaikan", "bengkel", "binatu", "laundry", "cuci", "salon", "pangkas",
	"konsultan", "konsultasi", "les", "kursus", "privat", "desain", "sewa", "rental", "pijat", "spa", "fotografi", "foto", "travel",
	"ekspedisi", "kurir", "ojek", "agensi", "event", "wedding", "arsitek", "akuntan", "klinik", "terapi"}

// sajian: masak untuk disajikan (kafe, restoran, katering) termasuk jasa makanan, bukan industri.
var sajian = []string{"kafe", "cafe", "coffee", "restoran", "resto", "kedai", "catering", "katering", "boga", "angkringan", "warung", "gofood", "grabfood"}

// grosir: kata yang membuat KLU perdagangan besar (46x) masuk akal.
var grosir = []string{"grosir", "besar", "distributor", "agen", "partai"}

func kata(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
}

func kalimat(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// ContohKLU: contoh kegiatan (uraian KBLI 5 digit) di bawah KLU 3 digit, huruf kalimat.
func ContohKLU(kode string, n int) []string {
	out := []string{}
	for _, x := range P().Norma {
		if strings.HasPrefix(x.KLU, kode) && len(kode) == 3 {
			if out = append(out, kalimat(x.Uraian)); len(out) == n {
				break
			}
		}
	}
	return out
}

func cocok(t string, ws []string) bool {
	for _, w := range ws {
		if w == t || (len(t) >= 4 && strings.HasPrefix(w, t)) || (len(w) >= 4 && strings.HasPrefix(t, w)) {
			return true
		}
	}
	return false
}

// CariKLU: kandidat KLU untuk deskripsi usaha, paling cocok dulu. jenis (perdagangan/jasa/
// manufaktur, boleh kosong) memberi sedikit bonus.
func CariKLU(teks, jenis string, n int) []KandidatKLU {
	var istilah []string
	for _, w := range kata(teks) {
		if len([]rune(w)) < 3 || kataUmum[w] {
			continue
		}
		istilah = append(istilah, w)
		istilah = append(istilah, sinonim[w]...)
	}
	slices.Sort(istilah)
	istilah = slices.Compact(istilah)
	if len(istilah) == 0 {
		return nil
	}

	klu := BenchmarkAktif()
	type dok struct{ nama, contoh []string }
	docs := make([]dok, len(klu))
	for i, k := range klu {
		docs[i].nama = kata(k.Nama)
		for _, x := range P().Norma {
			if strings.HasPrefix(x.KLU, k.Kode) {
				docs[i].contoh = append(docs[i].contoh, kata(x.Uraian)...)
			}
		}
	}
	idf := map[string]float64{}
	for _, t := range istilah {
		df := 0
		for _, d := range docs {
			if cocok(t, d.nama) || cocok(t, d.contoh) {
				df++
			}
		}
		if df > 0 {
			idf[t] = math.Log(float64(len(docs)) / float64(df))
		}
	}
	ada := func(daftar []string) bool {
		return slices.ContainsFunc(istilah, func(t string) bool { return slices.Contains(daftar, t) })
	}
	mauGrosir, online, buatSendiri := ada(grosir), ada(jualOnline), ada(produksi)
	// Jenis belum dipilih (mis. saran di langkah "usaha Anda paling mirip yang mana"): tebak dari
	// kata kerjanya, supaya "bikin kue sendiri" tidak jatuh ke toko kue.
	bonusJenis := 1.0
	if jenis == "" {
		switch {
		case buatSendiri && !ada(sajian):
			jenis, bonusJenis = "manufaktur", 3
		case ada(jasaKata):
			jenis, bonusJenis = "jasa", 3
		}
	}

	var out []KandidatKLU
	for i, k := range klu {
		var skor float64
		for _, t := range istilah {
			switch {
			case cocok(t, docs[i].nama):
				skor += 1.5 * idf[t]
			case cocok(t, docs[i].contoh):
				skor += idf[t]
			}
		}
		if online && k.Kode == "479" {
			skor += 4
		}
		if skor == 0 {
			continue
		}
		j := JenisKLU(k.Kode)
		switch {
		case jenis != "" && j == jenis:
			skor += bonusJenis
		case bonusJenis > 1 && j != jenis:
			skor -= bonusJenis // jenis hasil tebakan: kategori jenis lain mundur
		}
		if strings.HasPrefix(k.Kode, "46") && !mauGrosir {
			skor -= 6
		}
		if j == "manufaktur" && !buatSendiri && jenis != "manufaktur" {
			skor -= 3
		}
		if skor > 0 {
			out = append(out, KandidatKLU{KLUBenchmark: k, Jenis: j, Skor: skor})
		}
	}
	slices.SortStableFunc(out, func(a, b KandidatKLU) int { return cmp.Compare(b.Skor, a.Skor) })
	// Kandidat yang jauh di bawah teratas biasanya cocok kebetulan ("antar" ~ "antarkota").
	if len(out) > 0 {
		batas := out[0].Skor * 0.6
		out = slices.DeleteFunc(out, func(k KandidatKLU) bool { return k.Skor < batas })
	}
	out = out[:min(n, len(out))]
	for i := range out {
		out[i].Contoh = ContohKLU(out[i].Kode, 4)
	}
	return out
}
