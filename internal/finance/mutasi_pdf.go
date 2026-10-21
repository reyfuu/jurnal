package finance

import (
	"regexp"
	"strconv"
	"strings"
)

// Mutasi dari PDF e-statement / rekening koran. PDF tidak punya sel tabel, hanya baris teks,
// jadi tiap baris dibaca sendiri: baris yang diawali tanggal adalah transaksi, angka uang di
// ujung kanannya adalah mutasi dan saldo, dan baris tanpa tanggal di bawahnya lanjutan
// keterangan. Hasilnya tabel biasa (Tanggal, Keterangan, Debet, Kredit, Saldo) yang diperiksa
// ParseMutasi persis seperti berkas .xlsx / .csv.

var (
	polaAwalTanggal = regexp.MustCompile(`(?i)^(\d{1,2}[/-]\d{1,2}(?:[/-]\d{2,4})?|\d{1,2}[ -](?:jan|feb|mar|apr|mei|may|jun|jul|agu|agt|aug|sep|okt|oct|nov|des|dec)[a-z]*(?:[ -]\d{2,4})?)\b`)
	polaJam         = regexp.MustCompile(`^\d{1,2}[:.]\d{2}(?:[:.]\d{2})?$`)
	polaUang        = regexp.MustCompile(`^[-(]?(?:Rp\.?)?(?:\d{1,3}(?:[.,]\d{3})+(?:[.,]\d{2})?|\d+[.,]\d{2})\)?-?$`)
)

// barisPenutup: baris ringkasan cetakan bank; bukan transaksi dan bukan lanjutan keterangan.
var barisPenutup = []string{"saldo awal", "saldo akhir", "mutasi cr", "mutasi db", "mutasi kredit", "mutasi debet", "total", "halaman", "bersambung", "page "}

var bulanTeks = map[string]string{"jan": "01", "feb": "02", "mar": "03", "apr": "04", "mei": "05", "may": "05", "jun": "06",
	"jul": "07", "agu": "08", "agt": "08", "aug": "08", "sep": "09", "okt": "10", "oct": "10", "nov": "11", "des": "12", "dec": "12"}

// angkaTanggal: "02 Okt 2026" → "02/10/2026", supaya nama bulan Indonesia pun terbaca.
func angkaTanggal(s string) string {
	f := strings.FieldsFunc(s, func(r rune) bool { return r == ' ' || r == '-' })
	if len(f) < 2 {
		return s
	}
	b, ok := bulanTeks[strings.ToLower(f[1][:min(3, len(f[1]))])]
	if !ok {
		return s
	}
	return strings.Join(append([]string{f[0], b}, f[2:]...), "/")
}

type uangPDF struct {
	nilai int64
	tanda string // "masuk" / "keluar" bila ada penanda CR/DB di sebelah angkanya
	minus bool
}

// uangTeks membaca nominal cetakan bank dalam dua kebiasaan: 1,500,000.00 dan 1.500.000,00.
// Pemisah desimal adalah pemisah terakhir yang diikuti tepat dua angka; sen dibulatkan.
func uangTeks(s string) (int64, bool) {
	s = strings.Trim(strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(s), "-"), "("), "()-")
	s = strings.TrimPrefix(strings.TrimPrefix(s, "Rp."), "Rp")
	if s == "" {
		return 0, false
	}
	sen := int64(0)
	if i := strings.LastIndexAny(s, ".,"); i >= 0 && len(s)-i == 3 {
		sen, _ = strconv.ParseInt(s[i+1:], 10, 64)
		s = s[:i]
	}
	s = strings.NewReplacer(".", "", ",", "").Replace(s)
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, false
	}
	if sen >= 50 {
		n++
	}
	return n, true
}

// ujungUang memisahkan angka uang (dan penanda CR/DB) di ujung kanan baris dari teks di kirinya.
func ujungUang(kata []string) ([]string, []uangPDF) {
	var uang []uangPDF
	tandaTertunda := ""
	i := len(kata)
	for i > 0 {
		k := kata[i-1]
		if a := tandaArah(k); a != "" && len(k) <= 2 {
			tandaTertunda = a
			i--
			continue
		}
		// Penanda menempel: "1,500,000.00CR" / "150.000DB".
		tempel := ""
		for _, suf := range []string{"CR", "DB", "Cr", "Db", "cr", "db"} {
			if strings.HasSuffix(k, suf) && polaUang.MatchString(strings.TrimSuffix(k, suf)) {
				tempel, k = tandaArah(suf), strings.TrimSuffix(k, suf)
				break
			}
		}
		if !polaUang.MatchString(k) {
			break
		}
		n, ok := uangTeks(k)
		if !ok {
			break
		}
		u := uangPDF{nilai: n, tanda: tempel, minus: strings.HasPrefix(k, "-") || strings.HasSuffix(k, "-") || strings.HasPrefix(k, "(")}
		if u.tanda == "" {
			u.tanda = tandaTertunda
		}
		tandaTertunda = ""
		uang = append([]uangPDF{u}, uang...)
		i--
	}
	return kata[:i], uang
}

type transaksiPDF struct {
	tanggal, keterangan string
	uang                []uangPDF
	lanjutan            int
}

// arahDariKeterangan: cetakan BCA menulis arahnya di keterangan ("TRSF E-BANKING CR").
func arahDariKeterangan(ket string) string {
	for _, k := range strings.Fields(strings.ToUpper(ket)) {
		switch k {
		case "CR", "KR", "KREDIT", "CREDIT", "SETORAN", "BUNGA":
			return "masuk"
		case "DB", "DR", "DEBET", "DEBIT", "TARIKAN", "BIAYA", "PAJAK":
			return "keluar"
		}
	}
	return ""
}

// MutasiDariTeks mengubah baris teks PDF mutasi bank menjadi tabel untuk ParseMutasi.
// nil bila tidak ada satu pun baris transaksi.
func MutasiDariTeks(baris []string) [][]string {
	var txs []transaksiPDF
	var saldoAwal *int64
	var cur *transaksiPDF
	for _, b := range baris {
		b = strings.Join(strings.Fields(b), " ")
		if b == "" {
			continue
		}
		low := strings.ToLower(b)
		m := polaAwalTanggal.FindString(b)
		if m == "" {
			penutup := false
			for _, p := range barisPenutup {
				if strings.Contains(low, p) {
					penutup = true
				}
			}
			kata, uang := ujungUang(strings.Fields(b))
			if strings.Contains(low, "saldo awal") && len(uang) > 0 && saldoAwal == nil && len(txs) == 0 {
				v := uang[len(uang)-1].nilai
				saldoAwal = &v
			}
			switch {
			case penutup:
				cur = nil
			case cur != nil && len(cur.uang) == 0 && len(uang) > 0:
				// Tanggal & keterangan di satu baris, angkanya turun ke baris berikut.
				cur.uang = uang
				cur.keterangan += " " + strings.Join(kata, " ")
			case cur != nil && len(uang) == 0 && cur.lanjutan < 3:
				cur.keterangan += " " + b
				cur.lanjutan++
			default:
				cur = nil
			}
			continue
		}
		rest := strings.Fields(strings.TrimSpace(b[len(m):]))
		for len(rest) > 0 && polaJam.MatchString(rest[0]) {
			rest = rest[1:] // jam transaksi setelah tanggal
		}
		kata, uang := ujungUang(rest)
		ket := strings.Join(kata, " ")
		if strings.Contains(strings.ToLower(ket), "saldo awal") {
			if len(uang) > 0 && len(txs) == 0 {
				v := uang[len(uang)-1].nilai
				saldoAwal = &v
			}
			cur = nil
			continue
		}
		txs = append(txs, transaksiPDF{tanggal: angkaTanggal(m), keterangan: ket, uang: uang})
		cur = &txs[len(txs)-1]
	}
	if len(txs) == 0 {
		return nil
	}

	// Mutasi + DB/CR "?" hanya terisi bila arahnya tidak bisa dipastikan, supaya ParseMutasi
	// melaporkan baris itu alih-alih menganggapnya uang masuk.
	out := [][]string{{"Tanggal", "Keterangan", "Debet", "Kredit", "Mutasi", "DB/CR", "Saldo"}}
	prev := saldoAwal
	for _, t := range txs {
		if len(t.uang) == 0 {
			// Baris bertanggal tanpa angka uang sama sekali: jam cetak di kaki halaman
			// ("07/18/2025 08:58:23") atau kop, bukan transaksi. Ikut dibaca justru berbahaya:
			// tanggal bulan/hari seperti itu membuat semua tanggal lain dibaca terbalik.
			continue
		}
		debet, kredit, saldo := "", "", ""
		var mut *uangPDF
		var sal *int64
		switch u := t.uang; {
		case len(u) == 0:
		case len(u) >= 3 && (u[len(u)-3].nilai == 0 || u[len(u)-2].nilai == 0):
			// Kolom Debet, Kredit, Saldo terpisah; salah satunya nol.
			if u[len(u)-3].nilai > 0 {
				debet = strconv.FormatInt(u[len(u)-3].nilai, 10)
			} else {
				kredit = strconv.FormatInt(u[len(u)-2].nilai, 10)
			}
			sal = &u[len(u)-1].nilai
		case len(u) == 1:
			mut = &u[0]
		default:
			mut, sal = &u[len(u)-2], &u[len(u)-1].nilai
			if mut.tanda == "" && u[len(u)-1].tanda != "" {
				mut, sal = &u[len(u)-1], &u[len(u)-2].nilai // penanda menempel di angka terakhir: saldo di depan
			}
		}
		if mut != nil {
			arah := mut.tanda
			if arah == "" && mut.minus {
				arah = "keluar"
			}
			if arah == "" && prev != nil && sal != nil {
				switch *sal {
				case *prev + mut.nilai:
					arah = "masuk"
				case *prev - mut.nilai:
					arah = "keluar"
				}
			}
			if arah == "" {
				arah = arahDariKeterangan(t.keterangan)
			}
			switch arah {
			case "masuk":
				kredit = strconv.FormatInt(mut.nilai, 10)
			case "keluar":
				debet = strconv.FormatInt(mut.nilai, 10)
			}
		}
		if sal != nil {
			saldo = strconv.FormatInt(*sal, 10)
			prev = sal
		} else {
			prev = nil
		}
		if mut != nil && debet == "" && kredit == "" {
			out = append(out, []string{t.tanggal, t.keterangan, "", "", strconv.FormatInt(mut.nilai, 10), "?", saldo})
			continue
		}
		out = append(out, []string{t.tanggal, t.keterangan, debet, kredit, "", "", saldo})
	}
	return out
}
