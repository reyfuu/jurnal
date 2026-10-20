package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

type aiReportBody struct {
	Facts struct {
		Periode  string `json:"periode"`
		LabaRugi struct {
			Omzet      int64 `json:"omzet"`
			LabaBersih int64 `json:"laba_bersih"`
			MarginBps  int   `json:"margin_bps"`
		} `json:"laba_rugi"`
		Neraca struct {
			TotalAktiva int64 `json:"total_aktiva"`
			TotalPasiva int64 `json:"total_pasiva"`
			Seimbang    bool  `json:"seimbang"`
		} `json:"neraca"`
		ArusKas struct {
			KasBersih  int64 `json:"kas_bersih"`
			SaldoAkhir int64 `json:"saldo_akhir"`
		} `json:"arus_kas"`
		Aset struct {
			NilaiBuku     int64 `json:"nilai_buku"`
			BebanBulanIni int64 `json:"beban_bulan_ini"`
		} `json:"aset"`
		Tagihan struct {
			InvoiceBelumDibayar int64 `json:"invoice_belum_dibayar"`
		} `json:"tagihan"`
		Tren []struct {
			Bulan string `json:"bulan"`
		} `json:"tren"`
		Peringatan []string `json:"peringatan"`
	} `json:"facts"`
	Analysis struct {
		Tersedia bool   `json:"tersedia"`
		Alasan   string `json:"alasan"`
	} `json:"analysis"`
	LLM struct {
		Aktif    bool   `json:"aktif"`
		Petunjuk string `json:"petunjuk"`
	} `json:"llm"`
}

// Inti fitur: tanpa LLM_API_KEY, rincian laporan tetap keluar lengkap dan hanya bagian
// analisis yang kosong. Halaman tidak boleh jadi tidak berguna karena kunci belum diisi.
func TestAIReport_TanpaKunciLaporanTetapLengkap(t *testing.T) {
	t.Setenv("LLM_API_KEY", "")
	s := NewServer()
	owner := login(t, s, "owner@kedaikopisenja.id")

	var b aiReportBody
	rec := doRaw(t, s, owner, http.MethodGet, "/api/v1/reports/ai?period=2026-08", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
		t.Fatalf("balasan tidak terbaca: %v", err)
	}

	if b.Analysis.Tersedia {
		t.Error("tanpa kunci, analisis harus ditandai tidak tersedia")
	}
	if b.LLM.Aktif {
		t.Error("status LLM harus tidak aktif")
	}
	if !strings.Contains(b.LLM.Petunjuk, "LLM_API_KEY") {
		t.Error("petunjuk harus menyebut variabel yang perlu diisi")
	}

	// Angka laporan harus persis sama dengan laporan keuangan periode yang sama.
	f := b.Facts
	if f.Periode != "2026-08" {
		t.Errorf("periode = %q", f.Periode)
	}
	if f.LabaRugi.LabaBersih != 18_450_000 {
		t.Errorf("laba bersih = %d, want 18.450.000", f.LabaRugi.LabaBersih)
	}
	if f.LabaRugi.MarginBps != 2005 {
		t.Errorf("margin = %d bps, want 2005", f.LabaRugi.MarginBps)
	}
	if f.Neraca.TotalAktiva != 316_775_000 || !f.Neraca.Seimbang {
		t.Errorf("neraca: aktiva %d seimbang %v", f.Neraca.TotalAktiva, f.Neraca.Seimbang)
	}
	if f.Neraca.TotalAktiva != f.Neraca.TotalPasiva {
		t.Errorf("aktiva %d != pasiva %d", f.Neraca.TotalAktiva, f.Neraca.TotalPasiva)
	}
	if f.ArusKas.KasBersih != 17_000_000 {
		t.Errorf("kas bersih = %d, want 17.000.000", f.ArusKas.KasBersih)
	}
	if f.Aset.BebanBulanIni != 6_550_000 {
		t.Errorf("beban penyusutan = %d, want 6.550.000", f.Aset.BebanBulanIni)
	}
	if f.Tagihan.InvoiceBelumDibayar != 11_000_000 {
		t.Errorf("invoice belum dibayar = %d, want 11.000.000", f.Tagihan.InvoiceBelumDibayar)
	}
	if len(f.Tren) == 0 {
		t.Error("tren bulanan harus ikut dikirim")
	}
}

// Peringatan adalah fakta objektif, bukan analisis — harus muncul walau AI mati.
func TestAIReport_PeringatanMunculTanpaAI(t *testing.T) {
	t.Setenv("LLM_API_KEY", "")
	s := NewServer()
	owner := login(t, s, "owner@kedaikopisenja.id")

	var b aiReportBody
	doJSON(t, s, owner, http.MethodGet, "/api/v1/reports/ai?period=2026-08", nil, http.StatusOK, &b)
	gabung := strings.Join(b.Facts.Peringatan, " | ")
	if !strings.Contains(gabung, "PMK 72/2023") {
		t.Errorf("pemetaan harta yang belum diverifikasi harus diperingatkan, got %q", gabung)
	}
}

func TestAIReport_KasirTidakBolehMelihat(t *testing.T) {
	s := NewServer()
	kasir := login(t, s, "kasir@kedaikopisenja.id")
	if rec := doRaw(t, s, kasir, http.MethodGet, "/api/v1/reports/ai?period=2026-08", nil); rec.Code != http.StatusForbidden {
		t.Errorf("Kasir tidak punya hak laporan-keuangan: want 403, got %d", rec.Code)
	}
}

// Usaha tanpa transaksi (baru daftar, belum berlangganan): Laporan AI memakai data form sign-up.
func TestAIReport_TanpaTransaksiPakaiProfil(t *testing.T) {
	srv := NewServer()
	qa := register(t, srv, registerBody)
	do(srv, qa, http.MethodPut, "/api/v1/business", `{"nama_usaha":"Toko QA","jenis_usaha":"perdagangan","bentuk_usaha":"op","ptkp":"K/3","pkp":false,"metode_persediaan":"periodik","klu":"471","omzet_tahunan":2400000000,"laba_sebelum_pajak":168000000}`)
	rec := do(srv, qa, http.MethodGet, "/api/v1/reports/ai?period=2026-08", "")
	var body struct {
		Sumber string `json:"sumber"`
		Profil struct {
			OmzetSetahun int64 `json:"omzet_setahun"`
			Saran        *struct {
				Judul string `json:"judul"`
			} `json:"saran_skema_pajak"`
		} `json:"profil"`
		ProfilUsaha struct {
			NamaUsaha string `json:"nama_usaha"`
		} `json:"profil_usaha"`
		Analysis struct {
			Tersedia bool `json:"tersedia"`
		} `json:"analysis"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&body)
	if rec.Code != http.StatusOK || body.Sumber != "profil" || body.Profil.OmzetSetahun != 2_400_000_000 || body.Profil.Saran == nil || body.ProfilUsaha.NamaUsaha != "Toko QA" {
		t.Fatalf("laporan AI tanpa transaksi: %d %+v", rec.Code, body)
	}
	// usaha dengan transaksi tetap memakai laporan keuangan
	var demo struct {
		Sumber string `json:"sumber"`
	}
	_ = json.NewDecoder(do(srv, login(t, srv, "owner@kedaikopisenja.id"), http.MethodGet, "/api/v1/reports/ai?period=2026-08", "").Body).Decode(&demo)
	if demo.Sumber != "transaksi" {
		t.Errorf("demo: sumber %q", demo.Sumber)
	}
}
