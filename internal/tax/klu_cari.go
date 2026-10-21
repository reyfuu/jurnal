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
	"barbershop": {"pangkas", "rambut"}, "barber": {"pangkas", "rambut"},
	"warung": {"kedai", "eceran"}, "baju": {"pakaian"}, "fashion": {"pakaian"}, "kue": {"roti"},
	"homestay": {"pondok", "akomodasi"}, "konsultan": {"konsultasi"},
	"hp": {"telepon", "komunikasi"}, "handphone": {"telepon", "komunikasi"}, "ponsel": {"telepon", "komunikasi"}, "smartphone": {"telepon", "komunikasi"},
	"gadget": {"komunikasi"}, "emas": {"perhiasan"}, "apotek": {"farmasi", "obat"}, "apotik": {"farmasi", "obat"},
	"petshop": {"hewan", "peliharaan"}, "travel": {"perjalanan"}, "umroh": {"perjalanan"}, "umrah": {"perjalanan"}, "tour": {"perjalanan"},
	"kos": {"akomodasi"}, "kosan": {"akomodasi"}, "indekos": {"akomodasi"}, "penginapan": {"akomodasi"},
	"frozen": {"beku"}, "food": {"makanan"}, "percetakan": {"pencetakan"}, "undangan": {"pencetakan"}, "sablon": {"pencetakan"},
	"desain": {"perancangan"}, "grafis": {"perancangan"}, "fotografer": {"fotografi"}, "foto": {"fotografi"},
	"tambak": {"budidaya"}, "kontraktor": {"konstruksi"}, "renovasi": {"konstruksi"}, "cleaning": {"kebersihan"},
	"kerang": {"perikanan", "biota"}, "mutiara": {"perikanan"}, "lele": {"ikan", "perikanan"}, "nila": {"ikan", "perikanan"},
	"udang": {"perikanan"}, "kepiting": {"perikanan"}, "bandeng": {"ikan", "perikanan"}, "gurame": {"ikan", "perikanan"},
	"sembako": {"berbagai"}, "laptop": {"komputer"}, "pc": {"komputer"}, "rental": {"persewaan", "sewa"},
	"buku": {"budaya", "rekreasi"}, "musik": {"budaya", "rekreasi"}, "olahraga": {"rekreasi"}, "kulkas": {"rumah", "tangga"},
	"sambal": {"bumbu", "makanan"}, "bensin": {"bahan", "bakar"}, "pertamini": {"bahan", "bakar"}, "elpiji": {"gas"}, "kelontong": {"berbagai"}, "kaos": {"pakaian"}, "hijab": {"pakaian"}, "sepatu": {"alas"},
}

var kataUmum = map[string]bool{
	"di": true, "dan": true, "yang": true, "untuk": true, "ke": true, "dari": true, "saya": true, "kami": true,
	"jual": true, "jualan": true, "menjual": true, "usaha": true, "bisnis": true, "kecil": true, "dengan": true,
	"atau": true, "buat": true, "acara": true, "punya": true, "kiloan": true, "jasa": true, "beli": true, "lagi": true,
	"alat": true, "toko": true, "sendiri": true, "aksesoris": true, "kemasan": true, "fotografer": true, // "fotografer" ~ "foto kopi"; padanannya lewat sinonim
}

// jualOnline: kata yang langsung menunjuk KLU 479 (eceran melalui media / bukan di toko).
// Tidak lewat sinonim "media", karena "media" juga ada di "reproduksi media rekaman".
var jualOnline = []string{"online", "olshop", "shopee", "tokopedia", "tiktok", "lazada", "bukalapak", "marketplace", "ecommerce", "daring", "instagram", "whatsapp"}

// produksi: kata yang membuat KLU industri/pengolahan (golongan 10–33) masuk akal.
var produksi = []string{"produksi", "pabrik", "bikin", "membuat", "olah", "mengolah", "industri", "konveksi", "jahit", "masak", "roasting", "sangrai", "cetak", "rakit",
	// budidaya sendiri: pertanian, peternakan, perikanan (golongan 01–03, pembukuan produksi)
	"percetakan", "printing", "ternak", "beternak", "tanam", "menanam", "budidaya", "membudidayakan", "panen", "tambak", "berkebun", "kebun", "hidroponik"}

// dagangKata: kata yang menunjukkan barang dibeli lalu dijual (perdagangan).
var dagangKata = []string{"jual", "jualan", "menjual", "dagang", "berdagang", "pedagang", "toko", "dealer", "distributor", "grosir",
	"agen", "warung", "reseller", "dropship", "dropshipper", "kulakan", "minimarket", "kios", "apotek", "apotik", "petshop", "olshop", "eceran",
	"supplier", "importir", "impor"}

// NiatUsaha menebak jenis usaha dari kalimat bebas: produksi sendiri, jasa (termasuk jasa
// makanan & minuman), atau jual-beli. "" bila tidak ada petunjuk.
func NiatUsaha(teks string) string {
	w := kata(teks)
	// "pakan ternak", "alat tani": kata budidaya di situ menerangkan barang, bukan kegiatan.
	for i := 1; i < len(w); i++ {
		if slices.Contains([]string{"pakan", "obat", "alat", "perlengkapan", "kebutuhan", "bibit"}, w[i-1]) && slices.Contains(produksi, w[i]) {
			w[i] = ""
		}
	}
	low := " " + strings.Join(w, " ") + " "
	ada := func(daftar []string) bool {
		return slices.ContainsFunc(w, func(t string) bool { return slices.Contains(daftar, t) })
	}
	frasa := func(daftar []string) bool {
		return slices.ContainsFunc(daftar, func(f string) bool { return strings.Contains(low, " "+f+" ") })
	}
	switch {
	case frasa(jualLagi) && !ada(produksi):
		return "perdagangan"
	case ada(produksi) && !frasa(sajian):
		return "manufaktur"
	case frasa(sajian) || ada(jasaKata):
		return "jasa"
	case ada(dagangKata):
		return "perdagangan"
	}
	return ""
}

// budidayaKata: kegiatan memelihara / menanam sendiri; KLU-nya golongan 01–03 (pertanian,
// peternakan, perikanan), bukan industri pengolahan walau barangnya mirip ("kerang mutiara" ≠
// industri perhiasan, "ternak lele" ≠ pengolahan ikan).
var budidayaKata = []string{"ternak", "beternak", "peternak", "budidaya", "membudidayakan", "tanam", "menanam", "panen", "tambak", "berkebun", "kebun", "hidroponik", "kolam"}

// airKata: budidaya di air → perikanan budidaya (032).
var airKata = []string{"ikan", "lele", "nila", "udang", "kerang", "mutiara", "kepiting", "bandeng", "gurame", "tambak", "kolam", "rumput", "laut"}

// jualLagi: tanda membeli barang jadi untuk dijual kembali (perdagangan), dicari sebagai frasa.
var jualLagi = []string{"jual lagi", "dijual lagi", "jual kembali", "dijual kembali", "kulak", "reseller", "dropship", "beli dari", "ambil dari"}

// jasaKata: kata yang menunjukkan yang dijual adalah tenaga / keahlian.
var jasaKata = []string{"jasa", "servis", "service", "reparasi", "perbaikan", "bengkel", "binatu", "laundry", "cuci", "salon", "pangkas",
	"konsultan", "konsultasi", "les", "kursus", "privat", "desain", "sewa", "rental", "pijat", "spa", "fotografi", "foto", "travel",
	"ekspedisi", "kurir", "ojek", "agensi", "event", "wedding", "arsitek", "akuntan", "klinik", "terapi", "fotografer",
	"kontraktor", "renovasi", "cleaning", "umroh", "umrah", "kos", "kosan", "indekos", "homestay", "penginapan", "bimbel", "tutor",
	"dokter", "bidan", "praktik", "notaris", "pengacara", "videografer", "editing", "freelance", "pemrograman", "programmer", "konsultasi"}

// sajian: masak untuk disajikan (kafe, restoran, katering) termasuk jasa makanan, bukan industri.
var sajian = []string{"kafe", "cafe", "coffee shop", "restoran", "resto", "kedai kopi", "kedai makan", "catering", "katering", "boga", "angkringan",
	"warung makan", "warung kopi", "warkop", "rumah makan", "gofood", "grabfood", "shopeefood", "nasi kotak", "prasmanan"}

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
		// Sinonim dulu: kata pendek / umum ("hp") tetap menyumbang padanannya.
		istilah = append(istilah, sinonim[w]...)
		if len([]rune(w)) < 3 || kataUmum[w] {
			continue
		}
		istilah = append(istilah, w)
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
	// Jenis belum dipilih (saran di langkah "usaha Anda paling mirip yang mana"): niat usahanya
	// ditebak dari kata kerja seluruh kalimat dan MEMBATASI kategori yang boleh muncul. Tanpa
	// itu kata benda yang menentukan: "jual metal detector" jatuh ke industri logam, "toko buku"
	// ke penerbitan.
	tebakan := jenis == ""
	if tebakan {
		jenis = NiatUsaha(teks)
	}
	kataAsli := kata(teks)
	adaAsli := func(daftar []string) bool {
		return slices.ContainsFunc(kataAsli, func(w string) bool { return slices.Contains(daftar, w) })
	}
	budidaya := tebakan && jenis == "manufaktur" && adaAsli(budidayaKata)

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
		if budidaya && k.Kode == "032" && adaAsli(airKata) {
			skor += 4
		}
		if skor == 0 {
			continue
		}
		j := JenisKLU(k.Kode)
		// Bengkel: KLU 454 menggabungkan dagang & reparasi motor; bagi yang datang dengan niat
		// jasa, itu usaha jasa.
		if jenis == "jasa" && strings.Contains(k.Nama, "Reparasi") {
			j = "jasa"
		}
		switch {
		case tebakan && jenis != "" && j != jenis:
			continue
		case budidaya && k.Kode >= "100":
			continue // budidaya: hanya pertanian / peternakan / perikanan (golongan 01–03)
		case budidaya && k.Kode == "031" && !adaAsli([]string{"tangkap", "nelayan", "melaut", "menangkap"}):
			continue // dipelihara sendiri = perikanan budidaya, bukan tangkap
		case jenis != "" && j == jenis:
			skor++
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
	// Niat jelas tapi barangnya tidak ada di uraian KBLI (mis. metal detector): pakai kategori
	// umum jenis itu, daripada tidak memberi saran sama sekali.
	if len(out) == 0 && budidaya {
		umum := "014" // peternakan
		switch {
		case adaAsli(airKata):
			umum = "032"
		case adaAsli([]string{"tanam", "menanam", "panen", "kebun", "berkebun", "hidroponik"}):
			umum = "011"
		}
		if i := slices.IndexFunc(klu, func(k KLUBenchmark) bool { return k.Kode == umum }); i >= 0 {
			out = append(out, KandidatKLU{KLUBenchmark: klu[i], Jenis: jenis, Skor: 1})
		}
	}
	if len(out) == 0 && tebakan {
		umum := map[string]string{"perdagangan": "477", "jasa": "969"}[jenis]
		switch {
		case jenis == "perdagangan" && online:
			umum = "479"
		case jenis == "perdagangan" && mauGrosir:
			umum = "469"
		}
		if i := slices.IndexFunc(klu, func(k KLUBenchmark) bool { return k.Kode == umum }); i >= 0 {
			out = append(out, KandidatKLU{KLUBenchmark: klu[i], Jenis: jenis, Skor: 1})
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
