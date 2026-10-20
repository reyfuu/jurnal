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
	"online": {"media"}, "olshop": {"media"}, "shopee": {"media"}, "tokopedia": {"media"}, "tiktok": {"media"},
	"lazada": {"media"}, "marketplace": {"media"}, "ecommerce": {"media"},
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
	mauGrosir := slices.ContainsFunc(istilah, func(t string) bool { return slices.Contains(grosir, t) })

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
		if skor == 0 {
			continue
		}
		j := JenisKLU(k.Kode)
		if jenis != "" && j == jenis {
			skor += 1
		}
		if strings.HasPrefix(k.Kode, "46") && !mauGrosir {
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
