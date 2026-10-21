package main

import (
	"encoding/json"
	"net/http"
	"pajak/internal/coa"
	"pajak/internal/finance"
	"slices"
	"strings"
	"testing"

	"pajak/internal/xlsx"
)

const dagangCV = `{"nama_usaha":"Toko Senja","jenis_usaha":"perdagangan","bentuk_usaha":"cv","pkp":false,"metode_persediaan":"periodik"}`

func TestCoAEndpoints(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")

	var c struct {
		Supported bool                `json:"supported"`
		Accounts  []map[string]any    `json:"accounts"`
		Mapping   []map[string]string `json:"mapping"`
	}
	// belum ada profil → tidak didukung
	_ = json.NewDecoder(do(srv, owner, http.MethodGet, "/api/v1/coa", "").Body).Decode(&c)
	if c.Supported {
		t.Fatal("tanpa profil perdagangan CoA belum didukung")
	}

	do(srv, owner, http.MethodPut, "/api/v1/business", dagangCV)
	_ = json.NewDecoder(do(srv, owner, http.MethodGet, "/api/v1/coa", "").Body).Decode(&c)
	if !c.Supported || len(c.Accounts) < 60 || len(c.Mapping) != 18 {
		t.Fatalf("CoA perdagangan: supported=%v akun=%d mapping=%d", c.Supported, len(c.Accounts), len(c.Mapping))
	}

	var rep struct {
		Supported       bool `json:"supported"`
		IncomeStatement struct {
			LabaBersih int64 `json:"laba_bersih"`
		} `json:"income_statement"`
		BalanceSheet struct {
			Seimbang bool `json:"seimbang"`
		} `json:"balance_sheet"`
		TrialBalance []any `json:"trial_balance"`
	}
	rec := do(srv, owner, http.MethodGet, "/api/v1/reports/coa?period=2026-08", "")
	_ = json.NewDecoder(rec.Body).Decode(&rep)
	if rec.Code != http.StatusOK || !rep.Supported || !rep.BalanceSheet.Seimbang || rep.IncomeStatement.LabaBersih == 0 || len(rep.TrialBalance) == 0 {
		t.Fatalf("laporan CoA: %d %+v", rec.Code, rep)
	}

	rec = do(srv, owner, http.MethodGet, "/api/v1/reports/coa.xlsx?period=2026-08", "")
	rows, err := xlsx.Read(rec.Body.Bytes())
	if rec.Code != http.StatusOK || err != nil || len(rows) < 5 || !strings.Contains(rec.Header().Get("Content-Disposition"), ".xlsx") {
		t.Fatalf("ekspor xlsx: %d %v rows=%d", rec.Code, err, len(rows))
	}
}

func TestTransactionAkunOverride(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	do(srv, owner, http.MethodPut, "/api/v1/business", dagangCV)

	ok := `{"tanggal":"2026-08-21","jenis":"Biaya Lain","nominal":250000,"keterangan":"Langganan Canva","lunas":true,"akun":"6114"}`
	if rec := do(srv, owner, http.MethodPost, "/api/v1/transactions", ok); rec.Code != http.StatusCreated {
		t.Fatalf("akun valid: %d %s", rec.Code, rec.Body)
	}
	bad := strings.Replace(ok, `"6114"`, `"2115"`, 1) // Utang Dividen tidak aktif untuk CV
	if rec := do(srv, owner, http.MethodPost, "/api/v1/transactions", bad); rec.Code != http.StatusBadRequest {
		t.Fatalf("akun tidak aktif harus 400, got %d", rec.Code)
	}
}

// Usaha manufaktur otomatis memakai CoA manufaktur (penomoran sendiri) lewat peran akun:
// kas di 1102 Kas Besar, bukan 1101 seperti CoA dagang, dan neraca saldonya tetap seimbang.
func TestCoA_ManufakturMemakaiTemplateSendiri(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	if rec := do(srv, owner, http.MethodPut, "/api/v1/business", profileBody); rec.Code != http.StatusOK {
		t.Fatalf("profil manufaktur: %d %s", rec.Code, rec.Body)
	}
	rep := getJSON[coaReport](t, srv, owner, "/api/v1/reports/coa?period=2026-08")
	if !rep.Supported {
		t.Fatal("usaha manufaktur harus punya CoA")
	}
	var d, k int64
	saldo := map[string]int64{}
	for _, b := range rep.TrialBalance {
		d, k = d+b.Debit, k+b.Kredit
		saldo[b.Kode] = b.Saldo
	}
	if d != k {
		t.Fatalf("neraca saldo tidak seimbang: debit %d kredit %d", d, k)
	}
	if saldo["1102"] == 0 || rep.BalanceSheet == nil {
		t.Fatalf("kas manufaktur harus di 1102 Kas Besar: %+v", saldo)
	}
	if acc, ok := coa.Find(coa.Manufaktur, "1102"); !ok || acc.Nama != "Kas Besar" {
		t.Fatalf("akun 1102: %+v", acc)
	}
	if bs := rep.BalanceSheet; bs.TotalAset != bs.TotalLiabilitas+bs.TotalEkuitas {
		t.Fatalf("neraca per akun tidak seimbang: %+v", bs)
	}
}

// Buku Besar: total debit = kredit, dan saldo akhir tiap akun = neraca saldo laporan per akun,
// baik untuk CoA dagang maupun manufaktur.
func TestBukuBesar_SamaDenganNeracaSaldo(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	cek := func(nama string) {
		bb := getJSON[bukuBesarResp](t, srv, owner, "/api/v1/reports/buku-besar?period=2026-08")
		rep := getJSON[coaReport](t, srv, owner, "/api/v1/reports/coa?period=2026-08")
		if !bb.Supported || !bb.Seimbang || bb.TotalDebit == 0 {
			t.Fatalf("%s: buku besar supported=%v seimbang=%v debit=%d kredit=%d akun=%d", nama, bb.Supported, bb.Seimbang, bb.TotalDebit, bb.TotalKredit, len(bb.Akun))
		}
		tb := map[string]int64{}
		for _, b := range rep.TrialBalance {
			tb[b.Kode] = b.Saldo
		}
		adaBaris := false
		for _, a := range bb.Akun {
			if a.SaldoAkhir != tb[a.Kode] {
				t.Errorf("%s: akun %s %s saldo akhir %d, neraca saldo %d", nama, a.Kode, a.Nama, a.SaldoAkhir, tb[a.Kode])
			}
			if len(a.Baris) > 0 {
				adaBaris = true
				if last := a.Baris[len(a.Baris)-1]; last.Saldo != a.SaldoAkhir || last.Keterangan == "" {
					t.Errorf("%s: baris terakhir akun %s: %+v", nama, a.Kode, last)
				}
			}
		}
		if !adaBaris {
			t.Errorf("%s: tidak ada mutasi", nama)
		}
	}
	if rec := do(srv, owner, http.MethodPut, "/api/v1/business", strings.Replace(profileBody, "manufaktur", "perdagangan", 1)); rec.Code != http.StatusOK {
		t.Fatalf("profil dagang: %d", rec.Code)
	}
	cek("dagang")
	if rec := do(srv, owner, http.MethodPut, "/api/v1/business", profileBody); rec.Code != http.StatusOK {
		t.Fatalf("profil manufaktur: %d", rec.Code)
	}
	cek("manufaktur")
	if rec := do(srv, owner, http.MethodGet, "/api/v1/reports/buku-besar.xlsx?period=2026-08", ""); rec.Code != http.StatusOK {
		t.Fatalf("ekspor: %d %s", rec.Code, rec.Body)
	}
}

// Buku besar rentang lintas bulan & riwayat satu akun: saldo menyambung antar bulan, dan
// rentang tengah bulan memasukkan mutasi sebelum tanggal dari ke saldo awal.
func TestBukuBesar_RentangLintasBulan(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	if rec := do(srv, owner, http.MethodPut, "/api/v1/business", strings.Replace(profileBody, "manufaktur", "perdagangan", 1)); rec.Code != http.StatusOK {
		t.Fatalf("profil: %d", rec.Code)
	}
	ambil := func(q string) bukuBesarResp {
		return getJSON[bukuBesarResp](t, srv, owner, "/api/v1/reports/buku-besar?"+q)
	}
	jul, agu := ambil("period=2026-07"), ambil("period=2026-08")
	dua := ambil("dari=2026-07-01&sampai=2026-08-31")
	// Akun neraca menyambung antarbulan; akun laba rugi ditutup ke Laba Ditahan tiap bulan
	// (mulai 0), jadi di rentang dua bulan nilainya = mutasi Juli + Agustus.
	nominal := func(k coa.Kelompok) bool {
		return slices.Contains([]coa.Kelompok{coa.PendapatanUsaha, coa.HPP, coa.BebanOperasional, coa.PendapatanLain, coa.BebanLain}, k)
	}
	akhirJul, mutJul, awalAgu, akhirAgu, mutAgu := map[string]int64{}, map[string]int64{}, map[string]int64{}, map[string]int64{}, map[string]int64{}
	for _, a := range jul.Akun {
		akhirJul[a.Kode], mutJul[a.Kode] = a.SaldoAkhir, a.SaldoAkhir-a.SaldoAwal
	}
	for _, a := range agu.Akun {
		awalAgu[a.Kode], akhirAgu[a.Kode], mutAgu[a.Kode] = a.SaldoAwal, a.SaldoAkhir, a.SaldoAkhir-a.SaldoAwal
	}
	if !dua.Seimbang || dua.PeriodLabel == "" {
		t.Fatalf("rentang 2 bulan: %+v", dua.Seimbang)
	}
	for _, a := range dua.Akun {
		switch {
		case a.Kode == "3201":
			// laba ditahan: penutupan Juli terjadi di dalam rentang, tidak dijurnal di sini
		case nominal(a.Kelompok):
			if a.SaldoAkhir-a.SaldoAwal != mutJul[a.Kode]+mutAgu[a.Kode] {
				t.Errorf("akun %s: mutasi rentang %d ≠ Juli %d + Agustus %d", a.Kode, a.SaldoAkhir-a.SaldoAwal, mutJul[a.Kode], mutAgu[a.Kode])
			}
		default:
			if akhirJul[a.Kode] != awalAgu[a.Kode] {
				t.Errorf("akun %s: saldo akhir Juli %d ≠ saldo awal Agustus %d", a.Kode, akhirJul[a.Kode], awalAgu[a.Kode])
			}
			if a.SaldoAkhir != akhirAgu[a.Kode] {
				t.Errorf("akun %s: saldo akhir rentang %d ≠ Agustus %d", a.Kode, a.SaldoAkhir, akhirAgu[a.Kode])
			}
		}
	}
	// Riwayat satu akun, mulai tengah bulan.
	kas := ambil("dari=2026-08-10&sampai=2026-08-31&akun=1101")
	if len(kas.Akun) != 1 || kas.Akun[0].Kode != "1101" || len(kas.Pilihan) == 0 {
		t.Fatalf("riwayat kas: %+v", kas.Akun)
	}
	a := kas.Akun[0]
	for _, b := range a.Baris {
		if b.Tanggal < "2026-08-10" {
			t.Fatalf("baris sebelum tanggal dari: %+v", b)
		}
	}
	if a.SaldoAkhir != akhirAgu["1101"] || a.SaldoAwal+a.Debit-a.Kredit != a.SaldoAkhir {
		t.Fatalf("kas tengah bulan: awal %d debit %d kredit %d akhir %d (Agustus %d)", a.SaldoAwal, a.Debit, a.Kredit, a.SaldoAkhir, akhirAgu["1101"])
	}
	if rec := do(srv, owner, http.MethodGet, "/api/v1/reports/buku-besar?dari=2026-08-31&sampai=2026-08-01", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("rentang terbalik: %d", rec.Code)
	}
}

// Laporan transaksi: rentang tanggal bebas + filter; total = jumlah baris yang tampil.
func TestLaporanTransaksi_FilterRentang(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	semua := getJSON[laporanTransaksiResp](t, srv, owner, "/api/v1/reports/transaksi?dari=2026-08-05&sampai=2026-08-20")
	if semua.Jumlah == 0 {
		t.Fatal("tidak ada transaksi 5–20 Agustus di data demo")
	}
	var masuk, keluar int64
	for _, tx := range semua.Transaksi {
		if tx.Tanggal < "2026-08-05" || tx.Tanggal > "2026-08-20" {
			t.Fatalf("di luar rentang: %+v", tx)
		}
		if j, _ := finance.LookupJenis(tx.Jenis); j.Arah == "masuk" {
			masuk += tx.Nominal
		} else {
			keluar += tx.Nominal
		}
	}
	if masuk != semua.TotalMasuk || keluar != semua.TotalKeluar || len(semua.PerJenis) == 0 {
		t.Fatalf("total: %d/%d vs %d/%d", masuk, keluar, semua.TotalMasuk, semua.TotalKeluar)
	}
	jual := getJSON[laporanTransaksiResp](t, srv, owner, "/api/v1/reports/transaksi?dari=2026-08-05&sampai=2026-08-20&jenis=Penjualan&status=lunas")
	for _, tx := range jual.Transaksi {
		if tx.Jenis != "Penjualan" || !tx.Lunas {
			t.Fatalf("filter jenis/status: %+v", tx)
		}
	}
	if jual.TotalKeluar != 0 || jual.Jumlah > semua.Jumlah {
		t.Fatalf("filter: %+v", jual.Jumlah)
	}
	if rec := do(srv, owner, http.MethodGet, "/api/v1/reports/transaksi.xlsx?dari=2026-08-05&sampai=2026-08-20", ""); rec.Code != http.StatusOK {
		t.Fatalf("ekspor: %d %s", rec.Code, rec.Body)
	}
}
