package finance

import "testing"

// Catat dulu lalu impor mutasi (atau sebaliknya): transaksi lunas yang sama arah & nominal dalam
// ±3 hari dianggap mirip; transaksi belum lunas (termasuk invoice DPP + PPN) jadi calon pelunasan.
func TestCariMirip(t *testing.T) {
	txs := []Transaction{
		{ID: 1, Tanggal: "2026-10-10", Jenis: "Beli Bahan / Stok", Nominal: 125_000, Lunas: true, Sumber: "foto"},
		{ID: 2, Tanggal: "2026-10-01", Jenis: "Beli Bahan / Stok", Nominal: 125_000, Lunas: true},
		{ID: 3, Tanggal: "2026-10-10", Jenis: "Penjualan", Nominal: 125_000, Lunas: true},
		{ID: 4, Tanggal: "2026-09-20", Jenis: "Penjualan", Nominal: 1_000_000, InvoiceID: 7},
		{ID: 5, Tanggal: "2026-09-20", Jenis: "PPN Keluaran", Nominal: 110_000, InvoiceID: 7},
		{ID: 6, Tanggal: "2026-09-25", Jenis: "Beli Bahan / Stok", Nominal: 400_000},
	}
	m := CariMirip(txs, "2026-10-12", 125_000, "keluar", nil)
	if len(m) != 1 || m[0].ID != 1 || m[0].BelumLunas || m[0].Sumber != "foto" {
		t.Fatalf("lunas ±3 hari: %+v", m)
	}
	if m := CariMirip(txs, "2026-10-12", 125_000, "keluar", map[int]bool{1: true}); len(m) != 0 {
		t.Fatalf("kecuali: %+v", m)
	}
	if m := CariMirip(txs, "2026-10-15", 1_110_000, "masuk", nil); len(m) != 1 || m[0].InvoiceID != 7 || !m[0].BelumLunas {
		t.Fatalf("invoice DPP+PPN: %+v", m)
	}
	if m := CariMirip(txs, "2026-10-05", 400_000, "keluar", nil); len(m) != 1 || m[0].ID != 6 || !m[0].BelumLunas {
		t.Fatalf("utang belum lunas: %+v", m)
	}
	if m := CariMirip(txs, "2026-09-01", 400_000, "keluar", nil); len(m) != 0 {
		t.Fatalf("tagihan sesudah tanggal mutasi bukan calon: %+v", m)
	}
	p := PasanganMirip(append(txs, Transaction{ID: 8, Tanggal: "2026-10-11", Jenis: "Biaya Lain", Nominal: 125_000, Lunas: true, Sumber: "mutasi"}), "2026-10")
	if len(p) != 1 || p[0][0].ID+p[0][1].ID != 9 {
		t.Fatalf("pasangan dobel: %+v", p)
	}
}

func TestParseMutasi_TagihanJadiPelunasan(t *testing.T) {
	h := ParseMutasi([][]string{{"Tanggal", "Keterangan", "Debet", "Kredit"}, {"05/10/2026", "TRSF KE CV MAJU", "400000", ""}}, 2026,
		[]Transaction{{ID: 6, Tanggal: "2026-09-25", Jenis: "Beli Bahan / Stok", Nominal: 400_000, Pihak: "CV Maju"}})
	if len(h.Rows) != 1 || h.Rows[0].Lunasi != 6 || !h.Rows[0].Pilih || h.Rows[0].Mirip == nil {
		t.Fatalf("baris: %+v", h.Rows)
	}
	if txs, _ := MutasiKeTransaksi(h.Rows); len(txs) != 0 {
		t.Fatalf("baris pelunasan tidak boleh jadi transaksi baru: %+v", txs)
	}
}
