package finance

import "testing"

func TestBeriRef(t *testing.T) {
	txs := []Transaction{
		{ID: 1, Tanggal: "2026-10-01"},          // sebelum fitur: tanpa nomor
		{ID: 5, Tanggal: "2026-10-02"},          // TRX-2610-0001
		{ID: 6, Tanggal: "2026-10-03", Grup: 3}, // satu catatan 3 baris = satu nomor
		{ID: 7, Tanggal: "2026-10-03", Grup: 3},
		{ID: 8, Tanggal: "2026-10-03", Grup: 3},
		{ID: 9, Tanggal: "2026-09-28"},                 // backdate: nomor berikutnya di September
		{ID: 10, Tanggal: "2026-10-06", InvoiceID: 12}, // DPP + PPN invoice = satu nomor
		{ID: 11, Tanggal: "2026-10-06", InvoiceID: 12},
		{ID: 12, Tanggal: "2026-10-08"},
	}
	batal := []Transaction{{ID: 4, Tanggal: "2026-09-10", Ref: "TRX-2609-0019"}}
	BeriRef(txs, batal, 5)
	mau := []string{"", "TRX-2610-0001", "TRX-2610-0002", "TRX-2610-0002", "TRX-2610-0002", "TRX-2609-0020", "TRX-2610-0003", "TRX-2610-0003", "TRX-2610-0004"}
	for i, m := range mau {
		if txs[i].Ref != m {
			t.Errorf("tx %d ref %q, mau %q", txs[i].ID, txs[i].Ref, m)
		}
	}
	// nomor yang ada tidak berubah; baris catatan yang ditambah belakangan ikut nomor catatannya
	txs = append(txs, Transaction{ID: 13, Tanggal: "2026-10-03", Grup: 3}, Transaction{ID: 14, Tanggal: "2026-10-09"})
	BeriRef(txs, batal, 5)
	if txs[9].Ref != "TRX-2610-0002" || txs[10].Ref != "TRX-2610-0005" || txs[1].Ref != "TRX-2610-0001" {
		t.Errorf("tambahan: %q %q", txs[9].Ref, txs[10].Ref)
	}
	if RefBerikut(nil, "2027-01-15") != "TRX-2701-0001" {
		t.Error("awal bulan baru")
	}
}
