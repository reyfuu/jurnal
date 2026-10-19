package finance

import "strings"

// Terbilang mengeja jumlah Rupiah jadi kata, seperti lazimnya tertera di faktur Indonesia.
// Mis. 5_994_000 → "lima juta sembilan ratus sembilan puluh empat ribu rupiah".
//
// Mengikuti kaidah bahasa Indonesia: 1 di depan "belas", "puluh", "ratus" jadi "se-", dan
// "seribu" bukan "satu ribu". Pecahan sen diabaikan karena seluruh nominal di aplikasi ini
// bilangan bulat Rupiah.
func Terbilang(n int64) string {
	if n == 0 {
		return "nol rupiah"
	}
	s := eja(abs(n))
	if n < 0 {
		s = "minus " + s
	}
	return s + " rupiah"
}

var satuan = [...]string{"", "satu", "dua", "tiga", "empat", "lima", "enam", "tujuh", "delapan", "sembilan"}

// skala disusun dari yang terbesar supaya eja bisa memotong dari kiri.
var skala = []struct {
	nilai int64
	nama  string
}{
	{1_000_000_000_000, "triliun"},
	{1_000_000_000, "miliar"},
	{1_000_000, "juta"},
	{1_000, "ribu"},
}

func eja(n int64) string {
	switch {
	case n == 0:
		return ""
	case n < 10:
		return satuan[n]
	case n < 20:
		if n == 10 {
			return "sepuluh"
		}
		if n == 11 {
			return "sebelas"
		}
		return satuan[n-10] + " belas"
	case n < 100:
		return gabung(prefiksSe(n/10, "puluh"), eja(n%10))
	case n < 1000:
		return gabung(prefiksSe(n/100, "ratus"), eja(n%100))
	}
	for _, sk := range skala {
		if n >= sk.nilai {
			// "seribu", tapi "satu juta" — hanya ribu yang memakai bentuk se-.
			depan := eja(n / sk.nilai)
			if sk.nama == "ribu" && n/sk.nilai == 1 {
				depan = "se"
				return gabung(depan+sk.nama, eja(n%sk.nilai))
			}
			return gabung(depan+" "+sk.nama, eja(n%sk.nilai))
		}
	}
	return ""
}

// prefiksSe: 1 puluh → sepuluh, 1 ratus → seratus; selain itu "dua puluh" dst.
func prefiksSe(n int64, nama string) string {
	if n == 1 {
		return "se" + nama
	}
	return satuan[n] + " " + nama
}

func gabung(a, b string) string {
	if b == "" {
		return a
	}
	return strings.TrimSpace(a) + " " + b
}

func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
