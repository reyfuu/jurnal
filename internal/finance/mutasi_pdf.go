package finance

import (
	"fmt"
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
	// Tanggal di awal baris: 2026-08-31, 31/08, 31/08/26, 31-08-2026, 31 Agu 2026, 31-Aug-26.
	polaAwalTanggal = regexp.MustCompile(`(?i)^(\d{4}[/-]\d{1,2}[/-]\d{1,2}|\d{1,2}[/-]\d{1,2}(?:[/-]\d{2,4})?|\d{1,2}[ -](?:jan|feb|mar|apr|mei|may|jun|jul|agu|agt|aug|sep|okt|oct|nov|des|dec)[a-z]*(?:[ -]\d{2,4})?)\b`)
	polaISO         = regexp.MustCompile(`^(\d{4})[/-](\d{1,2})[/-](\d{1,2})$`)
	polaNoUrut      = regexp.MustCompile(`^\d{1,4}\.?$`)
	polaTahunKop    = regexp.MustCompile(`\b(20\d\d)\b`)
	polaJam         = regexp.MustCompile(`^\d{1,2}[:.]\d{2}(?:[:.]\d{2})?$`)
	polaUang        = regexp.MustCompile(`(?i)^[-+(]?(?:Rp\.?|IDR)?(?:\d{1,3}(?:[.,]\d{3})+(?:[.,]\d{2})?|\d+[.,]\d{2})\)?-?$`)
)

// barisPenutup: baris ringkasan cetakan bank; bukan transaksi dan bukan lanjutan keterangan.
var barisPenutup = []string{"saldo awal", "saldo akhir", "mutasi cr", "mutasi db", "mutasi kredit", "mutasi debet", "total", "halaman", "bersambung", "page ",
	"opening balance", "closing balance", "beginning balance", "ending balance", "saldo sebelumnya"}

// labelSaldoAwal: baris yang menyebut saldo awal rekening (dipakai menebak arah lewat saldo).
var labelSaldoAwal = []string{"saldo awal", "opening balance", "beginning balance", "saldo sebelumnya", "previous balance"}

func sebutSaldoAwal(low string) bool {
	for _, l := range labelSaldoAwal {
		if strings.Contains(low, l) {
			return true
		}
	}
	return false
}

// mataUang: kode mata uang di sela angka ("IDR 15,000.00"); dilewati, bukan pemutus.
func mataUang(k string) bool {
	switch strings.ToUpper(strings.TrimSuffix(k, ":")) {
	case "IDR", "RP", "RP.":
		return true
	}
	return false
}

var bulanTeks = map[string]string{"jan": "01", "feb": "02", "mar": "03", "apr": "04", "mei": "05", "may": "05", "jun": "06",
	"jul": "07", "agu": "08", "agt": "08", "aug": "08", "sep": "09", "okt": "10", "oct": "10", "nov": "11", "des": "12", "dec": "12"}

// angkaTanggal: "02 Okt 2026" → "02/10/2026", "2026-08-31" → "31/08/2026", supaya nama bulan
// Indonesia dan format ISO pun terbaca dengan aturan hari/bulan yang sama.
func angkaTanggal(s string) string {
	if m := polaISO.FindStringSubmatch(s); m != nil {
		return m[3] + "/" + m[2] + "/" + m[1]
	}
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
	s = strings.Trim(strings.TrimSpace(s), "+-()")
	for _, p := range []string{"Rp.", "Rp", "RP", "IDR", "idr"} {
		s = strings.TrimPrefix(s, p)
	}
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
		if mataUang(k) && len(uang)+len(tandaTertunda) > 0 {
			i--
			continue
		}
		if a := tandaArah(k); a != "" && len(k) <= 6 {
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
		if u.tanda == "" && strings.HasPrefix(k, "+") {
			u.tanda = "masuk" // "+Rp1.000.000" (bank digital)
		}
		if u.tanda == "" {
			u.tanda = tandaTertunda
		}
		tandaTertunda = ""
		uang = append([]uangPDF{u}, uang...)
		i--
	}
	// Penanda di KIRI angka ("DB 500.000,00"): tersisa sebagai penanda tertunda.
	if tandaTertunda != "" && len(uang) > 0 && uang[0].tanda == "" {
		uang[0].tanda = tandaTertunda
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
	rows, _ := MutasiDariTeksCek(baris)
	return rows
}

// tahunDariKop: tahun periode dari kop rekening koran ("PERIODE : SEPTEMBER 2026",
// "Date : 01 Aug 2026 - 31 Aug 2026"); "" bila tidak ada atau lebih dari satu tahun.
func tahunDariKop(baris []string) string {
	tahun := ""
	for _, b := range baris[:min(len(baris), 40)] {
		low := strings.ToLower(b)
		if !strings.Contains(low, "periode") && !strings.Contains(low, "period") && !strings.Contains(low, "date") && !strings.Contains(low, "tanggal") && !strings.Contains(low, "bulan") {
			continue
		}
		for _, t := range polaTahunKop.FindAllString(b, -1) {
			if tahun != "" && t != tahun {
				return ""
			}
			tahun = t
		}
	}
	return tahun
}

// MutasiDariTeksCek: seperti MutasiDariTeks, ditambah peringatan dari pemeriksaan saldo
// berjalan — saldo sebelumnya ± mutasi harus sama dengan saldo baris itu. Baris yang tidak
// cocok hampir pasti salah baca (angka terpotong, kolom tergeser), jadi pengguna diminta
// mencocokkannya sebelum membukukan.
func MutasiDariTeksCek(baris []string) ([][]string, []string) {
	var txs []transaksiPDF
	var saldoAwal *int64
	var cur *transaksiPDF
	tahunKop := tahunDariKop(baris)
	for _, b := range baris {
		b = strings.Join(strings.Fields(b), " ")
		if b == "" {
			continue
		}
		// Nomor urut di depan tanggal ("1 01/08/2026 …").
		if f := strings.Fields(b); len(f) > 2 && polaNoUrut.MatchString(f[0]) && polaAwalTanggal.MatchString(strings.Join(f[1:], " ")) {
			b = strings.Join(f[1:], " ")
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
			if sebutSaldoAwal(low) && len(uang) > 0 && saldoAwal == nil && len(txs) == 0 {
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
		if sebutSaldoAwal(strings.ToLower(ket)) {
			if len(uang) > 0 && len(txs) == 0 {
				v := uang[len(uang)-1].nilai
				saldoAwal = &v
			}
			cur = nil
			continue
		}
		tgl := angkaTanggal(m)
		if tahunKop != "" && strings.Count(tgl, "/")+strings.Count(tgl, "-") == 1 {
			tgl += "/" + tahunKop // "31/08" → "31/08/2026" dari kop periode
		}
		txs = append(txs, transaksiPDF{tanggal: tgl, keterangan: ket, uang: uang})
		cur = &txs[len(txs)-1]
	}
	if len(txs) == 0 {
		return nil, nil
	}
	var tidakCocok []string
	cekBaris := 0

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
		if prev != nil && sal != nil && (debet != "" || kredit != "") {
			d, _ := strconv.ParseInt(debet, 10, 64)
			k, _ := strconv.ParseInt(kredit, 10, 64)
			cekBaris++
			if *prev+k-d != *sal {
				tidakCocok = append(tidakCocok, t.tanggal)
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
	var peringatan []string
	// Banyak yang tidak cocok biasanya berarti urutannya terbalik (terbaru dulu), bukan salah
	// baca; itu tidak dilaporkan per baris.
	if n := len(tidakCocok); n > 0 && n*2 <= cekBaris {
		peringatan = append(peringatan, fmt.Sprintf("%d baris saldonya tidak cocok dengan mutasinya (tanggal %s). Kemungkinan ada angka yang salah terbaca; cocokkan baris itu dengan rekening koran sebelum membukukan.",
			n, strings.Join(tidakCocok[:min(5, n)], ", ")))
	}
	return out, peringatan
}
