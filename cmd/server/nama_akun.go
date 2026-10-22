package main

import (
	"net/http"
	"strings"

	"pajak/internal/coa"
	"pajak/internal/finance"
)

// namaAkunLaporan: kunci baris laporan → nama akun di CoA usaha, lewat peran akun. Dengan
// begitu laporan memakai istilah yang sama dengan Daftar Akun ("Piutang Usaha", "Utang Bank",
// "Beban Sewa") dan ikut berubah bila template CoA mengganti nama akunnya. Kosong bila usaha
// belum punya CoA (laporan memakai label bawaan).
func (s *tenant) namaAkunLaporan() map[string]string {
	all, _, p := s.coaAkun()
	if all == nil {
		return map[string]string{}
	}
	peta := coa.PetaDari(all)
	nama := func(kode string) string {
		if a, ok := coa.Find(all, kode); ok {
			return a.Nama
		}
		return ""
	}
	peran := func(id string) string { return nama(peta.Kode(id)) }
	out := map[string]string{
		"piutang":         peran(coa.PeranPiutang),
		"persediaan":      peran(coa.PeranPersediaan),
		"aset_lain":       peran(coa.PeranDibayarDimuka),
		"utang_usaha":     peran(coa.PeranUtangUsaha),
		"pinjaman":        peran(coa.PeranPinjaman),
		"kewajiban_lain":  peran(coa.PeranUtangLain),
		"modal_awal":      nama(coa.AutoAccount("Setor Modal", p, peta)),
		"omzet":           peran(coa.PeranPenjualan),
		"potongan":        peran(coa.PeranPotonganAplikasi),
		"hpp":             peran(coa.PeranHPP),
		"penyusutan":      peran(coa.PeranBebanPenyusutan),
		"amortisasi":      peran(coa.PeranBebanAmortisasi),
		"pendapatan_lain": peran(coa.PeranPendapatanLain),
	}
	// kas di neraca = kas + bank (+ saldo marketplace) → "Kas & Bank"
	var kas []string
	for _, id := range []string{coa.PeranKas, coa.PeranBank} {
		if n := peran(id); n != "" {
			kas = append(kas, n)
		}
	}
	out["kas"] = strings.Join(kas, " & ")
	// baris beban dikelompokkan per jenis transaksi → nama akun bebannya
	for _, j := range finance.JenisList {
		if j.Pos == "biaya" {
			out["beban:"+j.Nama] = nama(coa.AutoAccount(j.Nama, p, peta))
		}
	}
	for k, v := range out {
		if v == "" {
			delete(out, k)
		}
	}
	return out
}

func (s *tenant) handleNamaAkunLaporan(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, s.namaAkunLaporan())
}
