package finance

import "testing"

func jualTempo(id int, tanggal string, nominal int64) Transaction {
	return Transaction{ID: id, Tanggal: tanggal, Jenis: "Penjualan", Nominal: nominal, Pihak: "Toko Maju"}
}

func bayarUji(id int, tanggal string, alokasi ...Alokasi) Pembayaran {
	p := Pembayaran{ID: id, Tanggal: tanggal, Jenis: BayarPiutang, Alokasi: alokasi}
	for _, a := range alokasi {
		p.Total += a.Nominal
	}
	return p
}

// Pembayaran sebagian: piutang berkurang sebesar yang dibayar pada tanggal bayar; sebelum
// tanggal itu piutang masih penuh (laporan historis tidak memakai sisa hari ini).
func TestBayar_SebagianDanPosisiPerTanggal(t *testing.T) {
	txs := []Transaction{jualTempo(1, "2026-01-05", 300_000)}
	txs, err := Bayar(txs, bayarUji(1, "2026-01-20", Alokasi{TransaksiID: 1, Nominal: 200_000}))
	if err != nil {
		t.Fatal(err)
	}
	if tx := txs[0]; tx.Lunas || tx.Sisa() != 100_000 || tx.Dibayar("2026-01-19") != 0 || tx.Dibayar("2026-01-20") != 200_000 {
		t.Fatalf("setelah bayar sebagian: %+v sisa %d", tx, tx.Sisa())
	}
	ob := OpeningBalance{Tanggal: "2026-01-01"}
	if p := PosisiPada(ob, txs, "2026-01-19"); p.Piutang != 300_000 || p.Kas != 0 {
		t.Fatalf("sebelum bayar: %+v", p)
	}
	if p := PosisiPada(ob, txs, "2026-01-31"); p.Piutang != 100_000 || p.Kas != 200_000 || p.PiutangOps != 100_000 {
		t.Fatalf("sesudah bayar: %+v", p)
	}
	// Pelunasan sisanya: transaksi lunas, tanggal lunas = cicilan terakhir.
	txs, err = Bayar(txs, bayarUji(2, "2026-02-03", Alokasi{TransaksiID: 1, Nominal: 100_000}))
	if err != nil || !txs[0].Lunas || txs[0].TanggalLunas != "2026-02-03" {
		t.Fatalf("lunas: %+v %v", txs[0], err)
	}
}

// Aturan alokasi: tidak boleh melebihi sisa, arah harus cocok, tanggal bayar tidak sebelum
// transaksi, transaksi lunas biasa tidak dicicil, dan dokumen lama tidak diubah di tempat.
func TestBayar_Ditolak(t *testing.T) {
	asal := []Transaction{
		jualTempo(1, "2026-01-05", 300_000),
		{ID: 2, Tanggal: "2026-01-06", Jenis: "Beli Bahan / Stok", Nominal: 50_000},
		{ID: 3, Tanggal: "2026-01-07", Jenis: "Penjualan", Nominal: 10_000, Lunas: true},
		{ID: 4, Tanggal: "2026-01-08", Jenis: "Terima Pinjaman", Nominal: 1_000_000},
	}
	for nama, p := range map[string]Pembayaran{
		"melebihi sisa":       bayarUji(1, "2026-01-20", Alokasi{1, 300_001}),
		"arah salah":          bayarUji(1, "2026-01-20", Alokasi{2, 1}),
		"sebelum transaksi":   bayarUji(1, "2026-01-04", Alokasi{1, 1}),
		"sudah lunas":         bayarUji(1, "2026-01-20", Alokasi{3, 1}),
		"bukan piutang usaha": bayarUji(1, "2026-01-20", Alokasi{4, 1}),
		"tidak ada":           bayarUji(1, "2026-01-20", Alokasi{9, 1}),
		"nol":                 bayarUji(1, "2026-01-20", Alokasi{1, 0}),
		"tanpa alokasi":       bayarUji(1, "2026-01-20"),
		"dua kali sama":       bayarUji(1, "2026-01-20", Alokasi{1, 200_000}, Alokasi{1, 200_000}),
	} {
		if _, err := Bayar(asal, p); err == nil {
			t.Errorf("%s: harus ditolak", nama)
		}
	}
	baru, err := Bayar(asal, bayarUji(1, "2026-01-20", Alokasi{1, 100_000}))
	if err != nil || len(asal[0].Cicilan) != 0 || len(baru[0].Cicilan) != 1 {
		t.Fatalf("copy-on-write: asal %+v baru %+v %v", asal[0], baru[0], err)
	}
}

// Batal: cicilan dicabut dan status lunas dihitung ulang.
func TestBatalBayar(t *testing.T) {
	txs, _ := Bayar([]Transaction{jualTempo(1, "2026-01-05", 300_000)}, bayarUji(1, "2026-01-20", Alokasi{1, 300_000}))
	if !txs[0].Lunas {
		t.Fatal("harus lunas")
	}
	txs = BatalBayar(txs, 1)
	if txs[0].Lunas || txs[0].TanggalLunas != "" || len(txs[0].Cicilan) != 0 || txs[0].Sisa() != 300_000 {
		t.Fatalf("setelah batal: %+v", txs[0])
	}
}

// Umur dihitung dari jatuh tempo (tanpa jatuh tempo: tanggal + 30 hari) dan bernilai sisa.
func TestUmurSaldo_SisaDanJatuhTempo(t *testing.T) {
	txs := []Transaction{
		{ID: 1, Tanggal: "2026-01-05", Jenis: "Penjualan", Nominal: 300_000, Pihak: "A", JatuhTempo: "2026-03-31"},
		jualTempo(2, "2026-01-10", 500_000), // jatuh tempo 9 Feb
	}
	txs, _ = Bayar(txs, bayarUji(1, "2026-01-20", Alokasi{1, 200_000}))
	u := UmurSaldo(OpeningBalance{}, txs, "2026-03-15", true)
	if u.Total != 600_000 || len(u.Kelompok) != 5 || u.Kelompok[0].Label != "Belum jatuh tempo" {
		t.Fatalf("umur: total %d kelompok %+v", u.Total, u.Kelompok)
	}
	if u.Kelompok[0].Nilai != 100_000 || u.Kelompok[2].Nilai != 500_000 {
		t.Fatalf("kelompok: %+v (mau 100rb belum jatuh tempo, 500rb lewat 31–60 hari)", u.Kelompok)
	}
	for _, b := range u.Baris {
		if b.JatuhTempo == "" {
			t.Errorf("jatuh tempo bawaan harus terisi: %+v", b)
		}
	}
	// Per tanggal sebelum pembayaran: masih penuh 300rb.
	if u := UmurSaldo(OpeningBalance{}, txs, "2026-01-19", true); u.Total != 800_000 {
		t.Fatalf("umur per 19 Jan: %d", u.Total)
	}
}

// Invarian laporan berjalan (neraca seimbang, kas langsung = tidak langsung, kas menutup)
// tetap berlaku dengan cicilan lintas bulan dan pelunasan bulan yang sama.
func TestLaporanBerjalan_InvarianCicilan(t *testing.T) {
	ob, txs, assets, stok, gerak := skenarioBerjalan()
	txs = append(txs,
		Transaction{ID: 20, Tanggal: "2026-01-10", Jenis: "Penjualan", Nominal: 900_000, Pihak: "B"},
		Transaction{ID: 21, Tanggal: "2026-02-02", Jenis: "Biaya Lain", Nominal: 600_000, Pihak: "C"},
	)
	var err error
	for _, p := range []Pembayaran{
		bayarUji(1, "2026-01-25", Alokasi{20, 300_000}),
		bayarUji(2, "2026-02-14", Alokasi{20, 400_000}),
		{ID: 3, Tanggal: "2026-02-20", Jenis: BayarUtang, Total: 250_000, Alokasi: []Alokasi{{21, 250_000}}},
	} {
		if txs, err = Bayar(txs, p); err != nil {
			t.Fatal(err)
		}
	}
	k := HitungKartu(stok, txs, gerak)
	for _, p := range []string{"2026-01", "2026-02", "2026-03"} {
		r := BuildLaporanBerjalan(txs, assets, p, ob, k)
		bs, cf := r.BalanceSheet, r.CashFlow
		if !bs.IsBalanced || cf.OperasiLangsung.Net != cf.OperasiTidakLangsung.Net || cf.ClosingBalance != bs.Assets.Cash {
			t.Errorf("%s invarian: seimbang %v, langsung %d tidak langsung %d, kas %d vs %d", p, bs.IsBalanced, cf.OperasiLangsung.Net, cf.OperasiTidakLangsung.Net, cf.ClosingBalance, bs.Assets.Cash)
		}
	}
	feb := BuildLaporanBerjalan(txs, assets, "2026-02", ob, k)
	if feb.BalanceSheet.Assets.Receivables != 1_800_000+200_000 {
		t.Errorf("piutang Feb %d, mau 2.000.000 (1,8 jt skenario + sisa 200rb)", feb.BalanceSheet.Assets.Receivables)
	}
}

// Kas harian: cicilan menggerakkan kas di tanggal bayarnya masing-masing.
func TestCashWarnings_Cicilan(t *testing.T) {
	txs := []Transaction{{ID: 1, Tanggal: "2026-01-02", Jenis: "Biaya Lain", Nominal: 500_000}}
	txs, _ = Bayar(txs, Pembayaran{ID: 1, Tanggal: "2026-01-03", Jenis: BayarUtang, Total: 300_000, Alokasi: []Alokasi{{1, 300_000}}})
	w := CashWarnings(txs, 100_000)
	if len(w) != 1 || w[0].Tanggal != "2026-01-03" || w[0].Saldo != -200_000 {
		t.Fatalf("peringatan kas: %+v", w)
	}
}

// Impor mutasi: transfer sebesar sisa piutang dicocokkan sebagai pelunasan, dan transfer yang
// sudah dicatat sebagai cicilan dikenali sebagai dobel.
func TestCariMirip_Cicilan(t *testing.T) {
	txs, _ := Bayar([]Transaction{jualTempo(1, "2026-01-05", 300_000)}, bayarUji(1, "2026-01-20", Alokasi{1, 200_000}))
	m := CariMirip(txs, "2026-01-28", 100_000, "masuk", nil)
	if len(m) != 1 || !m[0].BelumLunas {
		t.Fatalf("sisa 100rb: %+v", m)
	}
	m = CariMirip(txs, "2026-01-21", 200_000, "masuk", nil)
	if len(m) != 1 || m[0].BelumLunas {
		t.Fatalf("cicilan 200rb yang sudah dicatat: %+v", m)
	}
	if m := CariMirip(txs, "2026-01-21", 300_000, "masuk", nil); len(m) != 0 {
		t.Fatalf("nominal penuh tidak cocok lagi: %+v", m)
	}
}

// Koreksi tetap bisa membandingkan transaksi walau ada field slice; cicilan ikut dibandingkan.
func TestSamaTx(t *testing.T) {
	a := jualTempo(1, "2026-01-05", 300_000)
	b := a
	if !samaTx(a, b) {
		t.Fatal("sama")
	}
	b.Cicilan = []Cicilan{{PembayaranID: 1, Tanggal: "2026-01-20", Nominal: 1}}
	if samaTx(a, b) {
		t.Fatal("cicilan berbeda harus tidak sama")
	}
}
