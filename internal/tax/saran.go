package tax

import (
	"cmp"
	"fmt"
	"slices"
)

// SaranBentuk: kesimpulan sheet "Bentuk Usaha" (Tax Planning Calculator V2.1) dalam bahasa awam —
// skema pajak termurah untuk omzet & laba usaha, alasannya, dan syaratnya. Dipakai di ringkasan
// sign-up dan Perencanaan Pajak (lewat CekAwal).
type SaranBentuk struct {
	Termurah string   `json:"termurah"`
	Sekarang string   `json:"sekarang"` // skema bentuk usaha yang dipilih sekarang
	Judul    string   `json:"judul"`
	Asumsi   string   `json:"asumsi"` // dasar perhitungan: usaha di tahun pertama
	Alasan   []string `json:"alasan"`
	Catatan  []string `json:"catatan"`
	Hemat    int64    `json:"hemat"`     // per tahun dibanding skema sekarang
	HematBps int64    `json:"hemat_bps"` // hemat / pajak skema sekarang (8500 = 85% lebih kecil)
	Omzet    int64    `json:"omzet"`     // untuk persen pajak terhadap omzet per skema
	Skema    []Skema  `json:"skema"`
	Laba     int64    `json:"laba"`
	PTKP     int64    `json:"ptkp"`
}

func skemaPribadi(id string) bool { return id == SkemaPPOP || id == SkemaPasal17OP || id == SkemaNPPN }

// SaranBentukUsaha: nil bila omzet belum diisi. Badan usaha dihitung dengan PTKP TK/0 untuk
// skema orang pribadi (pemiliknya belum ditanya status keluarga).
func SaranBentukUsaha(bentuk, ptkpCode string, omzet, laba int64) *SaranBentuk {
	if omzet <= 0 {
		return nil
	}
	pribadi := bentuk == "op"
	if !pribadi || ptkpCode == "" {
		ptkpCode = "TK/0"
	}
	bu, err := compareLaba(omzet, laba, ptkpCode, true)
	if err != nil {
		return nil
	}
	pr := P()
	cari := func(id string) Skema {
		return bu.Skema[slices.IndexFunc(bu.Skema, func(s Skema) bool { return s.ID == id })]
	}
	best, now := cari(bu.Termurah), cari(skemaSaatIni(bentuk, omzet, true))
	s := &SaranBentuk{Termurah: best.ID, Sekarang: now.ID, Hemat: now.Pajak - best.Pajak, Skema: bu.Skema, Laba: laba, PTKP: bu.PTKP, Omzet: omzet}
	if s.Hemat <= 0 { // seri dengan skema sekarang: tetap di bentuk sekarang
		s.Termurah, s.Hemat, best = now.ID, 0, now
	}
	if now.Pajak > 0 {
		s.HematBps = s.Hemat * 10_000 / now.Pajak
	}
	umkm := persenBps(int(pr.UMKMTarifBps))
	margin := pct(laba * 10_000 / omzet)

	s.Asumsi = "Asumsi: usaha di tahun pertama, dengan omzet dan laba setahun sesuai perkiraan yang Anda isi."
	if omzet > pr.BatasOmzetUMKM {
		s.Asumsi += fmt.Sprintf(" Karena itu PPh final %s masih dibandingkan walau omzet di atas %s; tahun berikutnya skema ini tidak berlaku lagi.", umkm, rp(pr.BatasOmzetUMKM))
	}
	if s.Termurah == s.Sekarang {
		s.Judul = "Skema pajak Anda sekarang sudah paling hemat: " + best.Nama
	} else {
		s.Judul = "Anda disarankan memakai " + best.Nama
	}

	switch best.ID {
	case SkemaPPOP:
		if omzet <= pr.UMKMBebasOP {
			s.Alasan = append(s.Alasan, fmt.Sprintf("Omzet %s masih di bawah %s per tahun, jadi sebagai usaha pribadi dengan PPh final %s Anda belum terutang PPh.", rp(omzet), rp(pr.UMKMBebasOP), umkm))
		} else {
			s.Alasan = append(s.Alasan, fmt.Sprintf("Dengan PPh final %s, omzet %s pertama bebas pajak; pajak hanya atas kelebihannya (%s), yaitu %s per tahun.", umkm, rp(pr.UMKMBebasOP), rp(omzet-pr.UMKMBebasOP), rp(best.Pajak)))
		}
	case SkemaPPBadan:
		s.Alasan = append(s.Alasan, fmt.Sprintf("PPh final %s dari seluruh omzet hanya %s per tahun. Margin laba Anda %s cukup tinggi, sehingga pajak yang dihitung dari laba akan lebih besar.", umkm, rp(best.Pajak), margin))
	case SkemaPasal17OP:
		if laba <= bu.PTKP {
			s.Alasan = append(s.Alasan, fmt.Sprintf("Laba %s tidak melebihi PTKP %s, jadi dengan pembukuan tidak ada PPh terutang, sedangkan PPh final tetap dihitung dari omzet.", rp(laba), rp(bu.PTKP)))
		} else {
			s.Alasan = append(s.Alasan, fmt.Sprintf("Margin laba Anda %s relatif tipis. Dengan pembukuan, pajak dihitung dari laba %s dikurangi PTKP %s memakai tarif progresif Pasal 17, yaitu %s per tahun — lebih kecil daripada pajak yang dihitung dari omzet.", margin, rp(laba), rp(bu.PTKP), rp(best.Pajak)))
		}
	case SkemaPasal31E:
		s.Alasan = append(s.Alasan, fmt.Sprintf("Sebagai badan, tarif %s atas laba mendapat pengurangan %s untuk omzet sampai %s. Atas laba %s pajaknya %s per tahun.", persenBps(int(pr.BadanTarifBps)), persenBps(int(pr.Fasilitas31EBps)), rp(pr.Batas31EPenuh), rp(laba), rp(best.Pajak)))
	case SkemaBadan22:
		s.Alasan = append(s.Alasan, fmt.Sprintf("Tarif umum badan %s atas laba %s: %s per tahun.", persenBps(int(pr.BadanTarifBps)), rp(laba), rp(best.Pajak)))
	case SkemaNPPN:
		s.Alasan = append(s.Alasan, fmt.Sprintf("Dengan norma, penghasilan neto dianggap %s dari omzet (dikurangi PTKP %s), pajaknya %s per tahun — lebih kecil karena laba Anda sebenarnya lebih tinggi dari norma.", persenBps(int(pr.NormaPerbandinganBps)), rp(bu.PTKP), rp(best.Pajak)))
	}

	if s.Termurah != s.Sekarang {
		s.Alasan = append(s.Alasan, fmt.Sprintf("Dibanding skema Anda sekarang (%s, %s per tahun), Anda bisa hemat %s per tahun — pajaknya %s lebih kecil.", now.Nama, rp(now.Pajak), rp(s.Hemat), pct(s.HematBps)))
		switch {
		case pribadi && !skemaPribadi(best.ID):
			s.Alasan = append(s.Alasan, "Artinya usaha dijalankan lewat badan usaha (PT/CV). Pertimbangkan biaya pendirian dan administrasinya.")
		case !pribadi && skemaPribadi(best.ID):
			s.Alasan = append(s.Alasan, "Artinya usaha dijalankan atas nama pribadi, bukan badan usaha.")
		}
	} else {
		lain := slices.DeleteFunc(slices.Clone(bu.Skema), func(x Skema) bool { return x.ID == best.ID })
		if len(lain) > 0 {
			next := slices.MinFunc(lain, func(a, b Skema) int { return cmp.Compare(a.Pajak, b.Pajak) })
			lebih := ""
			if best.Pajak > 0 {
				lebih = ", " + pct((next.Pajak-best.Pajak)*10_000/best.Pajak)
			}
			s.Alasan = append(s.Alasan, fmt.Sprintf("Pilihan terdekat berikutnya, %s, %s per tahun (%s lebih mahal%s).", next.Nama, rp(next.Pajak), rp(next.Pajak-best.Pajak), lebih))
		}
	}

	switch best.ID {
	case SkemaPPOP, SkemaPPBadan:
		if omzet > pr.BatasOmzetUMKM {
			s.Alasan = append(s.Alasan, fmt.Sprintf("Omzet di atas %s: PPh final hanya berlaku sampai akhir tahun pajak saat omzet pertama kali melewati batas (tahun pertama). Mulai tahun berikutnya wajib pembukuan dan tarif umum.", rp(pr.BatasOmzetUMKM)))
		}
		s.Alasan = append(s.Alasan, fmt.Sprintf("Syaratnya omzet setahun tidak melebihi %s, dan PPh final punya batas waktu pemakaian (%s).", rp(pr.BatasOmzetUMKM), pr.DasarUMKM))
	case SkemaPasal17OP:
		s.Alasan = append(s.Alasan, "Syaratnya menyelenggarakan pembukuan yang rapi (aplikasi ini menyiapkannya) dan memberi tahu DJP bahwa Anda tidak memakai PPh final.")
	case SkemaPasal31E, SkemaBadan22:
		s.Alasan = append(s.Alasan, fmt.Sprintf("Bila laba PT dibagikan ke pemilik, ada PPh dividen %s, kecuali memenuhi syarat dikecualikan (diinvestasikan kembali di Indonesia).", persenBps(int(pr.DividenBps))))
	case SkemaNPPN:
		s.Alasan = append(s.Alasan, fmt.Sprintf("Syaratnya omzet di bawah %s dan memberi tahu DJP dalam 3 bulan pertama tahun pajak. Norma sebenarnya mengikuti KLU dan wilayah; hitung di Kalkulator Pajak → NPPN.", rp(pr.BatasNPPN)))
	}

	s.Catatan = []string{fmt.Sprintf("Perkiraan dari omzet %s dan laba %s per tahun yang Anda isi; angka sebenarnya mengikuti pembukuan.", rp(omzet), rp(laba))}
	if !pribadi {
		s.Catatan = append(s.Catatan, "Skema orang pribadi dihitung dengan PTKP TK/0 (belum menikah, tanpa tanggungan).")
	}
	s.Catatan = append(s.Catatan, "Bandingkan rinci di menu Perencanaan Pajak, dan diskusikan dengan konsultan sebelum mengubah bentuk usaha.")
	return s
}
