package tax

import (
	"errors"
	"fmt"
)

// PPh 22 marketplace (PMK 37/2025): marketplace memungut PPh 22 dari omzet pedagang dalam negeri.
// Bagi pemakai PPh final UMKM, pungutan itu bagian dari pelunasan PPh final; bagi yang memakai
// pembukuan (tarif umum), pungutan itu kredit pajak — bila margin laba kecil, kredit melebihi
// pajak terutang (lebih bayar) dan pedagang dapat mengajukan SKB supaya tidak dipungut.

type MarketplaceInput struct {
	OmzetMarketplace int64 `json:"omzet_marketplace"` // setahun, lewat marketplace
	OmzetLain        int64 `json:"omzet_lain"`        // setahun, di luar marketplace
	NPMBps           int64 `json:"npm_bps"`           // margin laba bersih
	// Laba: laba bersih setahun dalam rupiah bila pengguna mengisi nominal; bila ada, dipakai
	// persis (NPMBps hanya untuk keterangan) supaya tidak bergeser karena pembulatan persen.
	Laba   *int64 `json:"laba,omitempty"`
	Bentuk string `json:"bentuk"` // op | badan
	Skema  string `json:"skema"`  // final (PPh final UMKM) | umum (pembukuan)
	PTKP   string `json:"ptkp"`   // untuk op
}

type SaranPajak struct {
	Judul     string `json:"judul"`
	Isi       string `json:"isi"`
	Langganan bool   `json:"langganan"` // tampilkan ajakan berlangganan
}

type MarketplaceResult struct {
	OmzetTotal   int64 `json:"omzet_total"`
	Laba         int64 `json:"laba"`
	Final        bool  `json:"final"`
	Dipungut     int64 `json:"dipungut"`      // PPh 22 dipungut marketplace setahun
	Terutang     int64 `json:"terutang"`      // PPh terutang setahun menurut skema
	TerutangUmum int64 `json:"terutang_umum"` // pembanding: dengan pembukuan
	LebihBayar   int64 `json:"lebih_bayar"`   // dipungut melebihi terutang
	KurangBayar  int64 `json:"kurang_bayar"`  // sisa yang disetor sendiri
	// NPMImpasBps: margin laba di bawah ini PPh 22 dipungut > PPh terutang (pembukuan). -1 = tidak ada.
	NPMImpasBps int64        `json:"npm_impas_bps"`
	Status      string       `json:"status"` // bebas | lunas | kurang_bayar | lebih_bayar
	Rekomendasi []SaranPajak `json:"rekomendasi"`
	Langkah     []Langkah    `json:"langkah"`
	Catatan     []string     `json:"catatan"`
}

func HitungMarketplace(in MarketplaceInput) (MarketplaceResult, error) {
	switch {
	case in.OmzetMarketplace < 0 || in.OmzetLain < 0:
		return MarketplaceResult{}, errors.New("omzet tidak boleh negatif")
	case in.Bentuk != "op" && in.Bentuk != "badan":
		return MarketplaceResult{}, errors.New("bentuk usaha harus op atau badan")
	case in.Skema != "final" && in.Skema != "umum":
		return MarketplaceResult{}, errors.New("skema harus final (PPh final UMKM) atau umum (pembukuan)")
	case in.Laba == nil && (in.NPMBps < -10_000 || in.NPMBps > 10_000):
		return MarketplaceResult{}, errors.New("margin laba bersih di atas 100% berarti laba lebih besar dari omzet; periksa lagi margin atau omzetnya")
	}
	if in.Laba != nil {
		total := in.OmzetMarketplace + in.OmzetLain
		if *in.Laba > total || *in.Laba < -total {
			return MarketplaceResult{}, fmt.Errorf("laba bersih (%s) lebih besar dari total omzet (%s); laba tidak mungkin melebihi omzet, periksa lagi isiannya", rp(*in.Laba), rp(total))
		}
		if total > 0 {
			in.NPMBps = mulDiv(*in.Laba, 10_000, total)
		}
	}
	ptkp := int64(0)
	if in.Bentuk == "op" {
		var ok bool
		if ptkp, ok = PTKPOf(in.PTKP); !ok {
			return MarketplaceResult{}, fmt.Errorf("status PTKP %q tidak dikenal", in.PTKP)
		}
	}
	pr := P()
	r := MarketplaceResult{OmzetTotal: in.OmzetMarketplace + in.OmzetLain, Final: in.Skema == "final", NPMImpasBps: -1}
	if r.Final && r.OmzetTotal > pr.BatasOmzetUMKM {
		r.Final = false
		r.Catatan = append(r.Catatan, fmt.Sprintf("Omzet setahun di atas %s tidak bisa memakai PPh final UMKM, jadi dihitung dengan pembukuan (tarif umum).", rp(pr.BatasOmzetUMKM)))
	}
	r.Laba = mulDiv(r.OmzetTotal, in.NPMBps, 10_000)
	if in.Laba != nil {
		r.Laba = *in.Laba
	}
	tarif := persenBps(int(pr.MarketplaceTarifBps))

	umum := func(laba int64) int64 {
		if in.Bentuk == "badan" {
			return Pasal31E(r.OmzetTotal, laba)
		}
		return Pasal17(max(0, laba-ptkp) / 1000 * 1000)
	}
	final := func() int64 {
		if in.Bentuk == "op" {
			return ppFinalOP(r.OmzetTotal)
		}
		return r.OmzetTotal * pr.UMKMTarifBps / 10_000
	}
	r.TerutangUmum = umum(r.Laba)
	r.Terutang = r.TerutangUmum
	if r.Final {
		r.Terutang = final()
	}

	bebas := in.Bentuk == "op" && r.OmzetTotal <= pr.MarketplaceBebasOP
	if !bebas {
		r.Dipungut = mulDiv(in.OmzetMarketplace, pr.MarketplaceTarifBps, 10_000)
	}
	switch d := r.Dipungut - r.Terutang; {
	case bebas:
		r.Status = "bebas"
	case d > 0:
		r.Status, r.LebihBayar = "lebih_bayar", d
	case d < 0:
		r.Status, r.KurangBayar = "kurang_bayar", -d
	default:
		r.Status = "lunas"
	}
	if !r.Final && r.Dipungut > 0 {
		lo, hi := int64(0), int64(10_000) // NPM terkecil yang pajaknya menutup PPh 22 dipungut
		if umum(r.OmzetTotal) >= r.Dipungut {
			for lo < hi {
				mid := (lo + hi) / 2
				if umum(mulDiv(r.OmzetTotal, mid, 10_000)) >= r.Dipungut {
					hi = mid
				} else {
					lo = mid + 1
				}
			}
			r.NPMImpasBps = lo
		}
	}

	skema := "pembukuan (tarif umum)"
	if r.Final {
		skema = "PPh final UMKM (" + pr.DasarUMKM + ")"
	}
	r.Langkah = []Langkah{
		{"Omzet lewat marketplace", in.OmzetMarketplace, ""},
		{"Omzet di luar marketplace", in.OmzetLain, ""},
		{"PPh 22 dipungut marketplace", r.Dipungut, tarif + " x omzet marketplace (" + pr.DasarMarketplace + ")"},
		{"PPh terutang setahun", r.Terutang, skema},
	}
	if r.LebihBayar > 0 {
		r.Langkah = append(r.Langkah, Langkah{"Lebih bayar", r.LebihBayar, "dipungut melebihi pajak terutang"})
	} else if r.KurangBayar > 0 {
		r.Langkah = append(r.Langkah, Langkah{"Sisa disetor sendiri", r.KurangBayar, ""})
	}

	add := func(judul, isi string, langganan bool) {
		r.Rekomendasi = append(r.Rekomendasi, SaranPajak{judul, isi, langganan})
	}
	switch {
	case bebas:
		add("Sampaikan surat pernyataan omzet ke marketplace",
			fmt.Sprintf("Omzet setahun %s tidak melebihi %s. Sebagai orang pribadi, Anda tidak dipungut PPh 22 bila menyampaikan surat pernyataan omzet ke marketplace. Pantau omzet: begitu melewati batas, beri tahu marketplace.", rp(r.OmzetTotal), rp(pr.MarketplaceBebasOP)), false)
	case !r.Final && r.LebihBayar > 0:
		npm := ""
		if r.NPMImpasBps >= 0 {
			npm = fmt.Sprintf(" karena margin laba Anda (%s) di bawah ambang %s", pct(in.NPMBps), pct(r.NPMImpasBps))
		}
		add("Ajukan SKB PPh 22 supaya tidak dipungut marketplace",
			fmt.Sprintf("PPh 22 yang dipungut marketplace setahun (%s) lebih besar dari PPh terutang (%s)%s. Kelebihan %s menjadi lebih bayar di SPT Tahunan dan baru kembali lewat restitusi yang memakan waktu dan bisa diikuti pemeriksaan. Dengan Surat Keterangan Bebas (SKB) pemungutan PPh 22, marketplace tidak memungut sehingga kas usaha tidak tertahan.",
				rp(r.Dipungut), rp(r.Terutang), npm, rp(r.LebihBayar)), false)
		add("Siapkan pengajuan SKB bersama MasterPajak AI",
			"Pengajuan SKB perlu laporan keuangan dan perhitungan pajak yang rapi. Isi data usaha dan berlangganan: pembukuan, laporan laba rugi, dan perkiraan PPh tersusun otomatis dari transaksi Anda sebagai bahan pengajuan.", true)
	case !r.Final && r.KurangBayar > 0:
		add("PPh 22 marketplace mengurangi pajak tahunan",
			fmt.Sprintf("PPh 22 yang dipungut (%s) dikreditkan di SPT Tahunan. Sisa %s disetor sendiri, misalnya lewat angsuran PPh 25.", rp(r.Dipungut), rp(r.KurangBayar)), false)
	case r.Final && r.LebihBayar > 0:
		add("Ada pungutan melebihi PPh final",
			fmt.Sprintf("PPh 22 yang dipungut (%s) melebihi PPh final terutang (%s), antara lain karena omzet %s pertama bebas PPh final bagi orang pribadi. Simpan bukti pungutan dari marketplace; kelebihan %s dapat dimintakan kembali (pemindahbukuan/pengembalian) — konsultasikan prosedurnya.",
				rp(r.Dipungut), rp(r.Terutang), rp(pr.UMKMBebasOP), rp(r.LebihBayar)), false)
	case r.Final && r.KurangBayar > 0:
		add("Setor sendiri PPh final atas omzet di luar marketplace",
			fmt.Sprintf("PPh 22 marketplace (%s) menjadi bagian pelunasan PPh final. Sisa %s setahun disetor sendiri setiap bulan atas omzet yang tidak dipungut marketplace.", rp(r.Dipungut), rp(r.KurangBayar)), false)
	case r.Final:
		add("PPh 22 marketplace sudah melunasi PPh final", "Pungutan marketplace menjadi bagian pelunasan PPh final Anda; simpan bukti pungutannya untuk SPT Tahunan.", false)
	}
	if r.Final && r.TerutangUmum < r.Terutang {
		add("Bandingkan dengan pembukuan",
			fmt.Sprintf("Dengan pembukuan (tarif umum), pajak Anda %s setahun, lebih kecil dari PPh final %s. Bila beralih, PPh 22 marketplace menjadi kredit pajak; bila melebihi pajak terutang, SKB dapat diajukan.", rp(r.TerutangUmum), rp(r.Terutang)), true)
	}
	r.Catatan = append(r.Catatan,
		fmt.Sprintf("Mengikuti %s; minta tim pajak memverifikasi sebelum mengajukan SKB atau surat pernyataan.", pr.DasarMarketplace),
		"Perkiraan setahun dari omzet dan margin yang Anda isi.")
	return r, nil
}
