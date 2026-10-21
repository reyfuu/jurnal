package main

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"pajak/internal/finance"
	"pajak/internal/xlsx"
)

// Unduh semua daftar: satu Excel berisi 6 daftar untuk periode itu (total piutang = umur piutang
// per akhir periode = neraca), satu PDF yang sama, dan hanya untuk yang boleh melihat laporan.
func TestDaftarUsaha_XlsxDanPDF(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	rec := do(srv, owner, http.MethodGet, "/api/v1/reports/daftar.xlsx?period=2026-08", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Header().Get("Content-Disposition"), "daftar-") {
		t.Fatalf("xlsx: %d %v", rec.Code, rec.Header())
	}
	sheets, err := xlsx.ReadAll(rec.Body.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	var nama []string
	for _, sh := range sheets {
		nama = append(nama, sh.Name)
	}
	if strings.Join(nama, "|") != "Piutang|Utang|Invoice|Barang|Pelanggan|Pemasok" {
		t.Fatalf("sheet: %v", nama)
	}
	umur := getJSON[finance.UmurSaldoResult](t, srv, owner, "/api/v1/reports/umur?jenis=piutang&per=2026-08-31")
	piutang := sheets[0].Rows
	total := piutang[len(piutang)-1]
	if total[0] != "Total" || total[len(total)-1] != strconv.FormatInt(umur.Total, 10) || umur.Total == 0 {
		t.Fatalf("total piutang %q, umur piutang %d", total, umur.Total)
	}
	if len(sheets[4].Rows) < 2 {
		t.Error("pelanggan Agustus kosong")
	}
	pdf := do(srv, owner, http.MethodGet, "/api/v1/reports/daftar.pdf?dari=2026-08-01&sampai=2026-08-31", "")
	if pdf.Code != http.StatusOK || !bytes.HasPrefix(pdf.Body.Bytes(), []byte("%PDF")) {
		t.Fatalf("pdf: %d", pdf.Code)
	}
	kasir := login(t, srv, "kasir@kedaikopisenja.id")
	if rec := do(srv, kasir, http.MethodGet, "/api/v1/reports/daftar.xlsx?period=2026-08", ""); rec.Code != http.StatusForbidden {
		t.Fatalf("kasir: %d", rec.Code)
	}
}
