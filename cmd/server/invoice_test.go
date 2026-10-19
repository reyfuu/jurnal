package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestInvoiceEndpoints_DaftarDanCetakPDF(t *testing.T) {
	s := NewServer()
	owner := login(t, s, "owner@kedaikopisenja.id")

	// Daftar: faktur seed ikut terbawa, lengkap dengan total turunannya.
	var list struct {
		Summary struct {
			Invoices []struct {
				ID     int                             `json:"id"`
				Nomor  string                          `json:"nomor"`
				NoPO   string                          `json:"no_po"`
				Totals struct{ Subtotal, Total int64 } `json:"totals"`
			} `json:"invoices"`
			BelumDibayar int64 `json:"belum_dibayar"`
		} `json:"summary"`
		NomorBerikutnya string   `json:"nomor_berikutnya"`
		ProfilLengkap   []string `json:"profil_lengkap"`
	}
	doJSON(t, s, owner, http.MethodGet, "/api/v1/invoice", nil, http.StatusOK, &list)
	if len(list.Summary.Invoices) != 3 {
		t.Fatalf("faktur seed: want 3, got %d", len(list.Summary.Invoices))
	}
	if list.Summary.Invoices[0].Totals.Total != 5_400_000 {
		t.Errorf("total faktur pertama = %d, want 5.400.000", list.Summary.Invoices[0].Totals.Total)
	}
	if list.Summary.Invoices[0].NoPO == "" {
		t.Error("nomor PO faktur seed hilang")
	}
	if list.NomorBerikutnya == "" {
		t.Error("nomor faktur berikutnya harus disediakan untuk form")
	}

	// Cetak PDF faktur pertama.
	id := list.Summary.Invoices[0].ID
	rec := doRaw(t, s, owner, http.MethodGet, "/api/v1/invoice/"+itoa(id)+"/pdf", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("cetak PDF: status %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/pdf" {
		t.Errorf("Content-Type = %q, want application/pdf", ct)
	}
	if cd := rec.Header().Get("Content-Disposition"); !strings.Contains(cd, "INV-2026-061.pdf") {
		t.Errorf("nama berkas unduhan harus memakai nomor faktur, got %q", cd)
	}
	if !bytes.HasPrefix(rec.Body.Bytes(), []byte("%PDF-1.4")) {
		t.Error("isi balasan bukan berkas PDF")
	}
	if !strings.Contains(rec.Body.String(), "PO-KDM-2026-0317") {
		t.Error("nomor PO tidak ikut tercetak di PDF")
	}
}

func TestInvoiceEndpoints_BuatUbahHapus(t *testing.T) {
	s := NewServer()
	owner := login(t, s, "owner@kedaikopisenja.id")

	body := `{"pelanggan_nama":"Toko Mitra Jaya","tanggal":"2026-09-01","jatuh_tempo":"2026-10-01",
	  "no_po":"PO-TMJ-77","ppn_bps":1100,
	  "items":[{"deskripsi":"Biji kopi 1kg","qty":10,"satuan":"kg","harga_satuan":250000}]}`
	var dibuat struct {
		ID    int    `json:"id"`
		Nomor string `json:"nomor"`
	}
	doJSON(t, s, owner, http.MethodPost, "/api/v1/invoice", strings.NewReader(body), http.StatusCreated, &dibuat)
	// Nomor diisi otomatis meneruskan deret seed (INV-2026-089 → INV-2026-090).
	if dibuat.Nomor != "INV-2026-090" {
		t.Errorf("nomor otomatis = %q, want INV-2026-090", dibuat.Nomor)
	}

	// Nomor yang sama ditolak, bukan diam-diam membuat faktur kembar.
	dup := `{"nomor":"INV-2026-090","pelanggan_nama":"Toko Lain","tanggal":"2026-09-02","jatuh_tempo":"2026-10-02",
	  "items":[{"deskripsi":"Apa saja","qty":1,"harga_satuan":1000}]}`
	if rec := doRaw(t, s, owner, http.MethodPost, "/api/v1/invoice", strings.NewReader(dup)); rec.Code != http.StatusConflict {
		t.Errorf("nomor kembar harus ditolak 409, got %d", rec.Code)
	}

	// Ubah status jadi lunas.
	ubah := `{"nomor":"INV-2026-090","status":"lunas","pelanggan_nama":"Toko Mitra Jaya",
	  "tanggal":"2026-09-01","jatuh_tempo":"2026-10-01","ppn_bps":1100,
	  "items":[{"deskripsi":"Biji kopi 1kg","qty":10,"satuan":"kg","harga_satuan":250000}]}`
	var diubah struct {
		Status string `json:"status"`
	}
	doJSON(t, s, owner, http.MethodPut, "/api/v1/invoice/"+itoa(dibuat.ID), strings.NewReader(ubah), http.StatusOK, &diubah)
	if diubah.Status != "lunas" {
		t.Errorf("status = %q, want lunas", diubah.Status)
	}

	if rec := doRaw(t, s, owner, http.MethodDelete, "/api/v1/invoice/"+itoa(dibuat.ID), nil); rec.Code != http.StatusNoContent {
		t.Errorf("hapus faktur: status %d", rec.Code)
	}
	if rec := doRaw(t, s, owner, http.MethodGet, "/api/v1/invoice/"+itoa(dibuat.ID)+"/pdf", nil); rec.Code != http.StatusNotFound {
		t.Errorf("faktur terhapus harusnya 404, got %d", rec.Code)
	}
}

func TestInvoiceEndpoints_TolakDataTidakMasukAkal(t *testing.T) {
	s := NewServer()
	owner := login(t, s, "owner@kedaikopisenja.id")
	for _, c := range []struct{ nama, body string }{
		{"tanpa pelanggan", `{"tanggal":"2026-09-01","jatuh_tempo":"2026-10-01","items":[{"deskripsi":"x","qty":1,"harga_satuan":1}]}`},
		{"tanpa baris", `{"pelanggan_nama":"A","tanggal":"2026-09-01","jatuh_tempo":"2026-10-01","items":[]}`},
		{"qty nol", `{"pelanggan_nama":"A","tanggal":"2026-09-01","jatuh_tempo":"2026-10-01","items":[{"deskripsi":"x","qty":0,"harga_satuan":1}]}`},
		{"jatuh tempo mendahului terbit", `{"pelanggan_nama":"A","tanggal":"2026-10-01","jatuh_tempo":"2026-09-01","items":[{"deskripsi":"x","qty":1,"harga_satuan":1}]}`},
	} {
		rec := doRaw(t, s, owner, http.MethodPost, "/api/v1/invoice", strings.NewReader(c.body))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: want 400, got %d (%s)", c.nama, rec.Code, strings.TrimSpace(rec.Body.String()))
		}
	}
}

func itoa(n int) string { return strconv.Itoa(n) }

func doRaw(t *testing.T, s *Server, c *http.Cookie, method, path string, body *strings.Reader) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body == nil {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, body)
		r.Header.Set("Content-Type", "application/json")
	}
	r.AddCookie(c)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, r)
	return rec
}

func doJSON(t *testing.T, s *Server, c *http.Cookie, method, path string, body *strings.Reader, wantCode int, out any) {
	t.Helper()
	rec := doRaw(t, s, c, method, path, body)
	if rec.Code != wantCode {
		t.Fatalf("%s %s: status %d, want %d (%s)", method, path, rec.Code, wantCode, strings.TrimSpace(rec.Body.String()))
	}
	if out != nil {
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			t.Fatalf("balasan tidak terbaca: %v", err)
		}
	}
}
