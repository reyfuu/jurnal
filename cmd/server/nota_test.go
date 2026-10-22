package main

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pajak/internal/ai"
	"pajak/internal/llm"
)

func unggahFoto(t *testing.T, srv *Server, c *http.Cookie, isi []byte) *httptest.ResponseRecorder {
	t.Helper()
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	fw, _ := mw.CreateFormFile("foto", "nota.png")
	_, _ = fw.Write(isi)
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/foto", &b)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.AddCookie(c)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	return rec
}

// Foto nota dikirim ke model sebagai image_url (data URI); balasannya dirapikan jadi pratinjau
// transaksi — tidak ada yang tersimpan.
func TestBacaNota_PratinjauDariFoto(t *testing.T) {
	var diterima string
	palsu := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		diterima = string(b)
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []map[string]any{{"message": map[string]string{"role": "assistant",
			"content": `{"tanggal":"2026-01-05","jenis":"Beli Bahan / Stok","nominal":125000,"pihak":"Toko Sumber Rejeki","keterangan":"gula 5 kg","lunas":true,"yakin":"tinggi","alasan":"nota pembelian bahan"}`}}}})
	}))
	defer palsu.Close()
	srv := NewServer()
	srv.llmGambar = llm.New(llm.Config{APIKey: "uji", BaseURL: palsu.URL, Model: "gpt-4o-mini"})
	owner := login(t, srv, "owner@kedaikopisenja.id")
	sebelum := len(srv.tenants[1].txs)

	rec := unggahFoto(t, srv, owner, png1x1)
	if rec.Code != http.StatusOK {
		t.Fatalf("baca nota: %d %s", rec.Code, rec.Body)
	}
	var s ai.SaranNota
	_ = json.Unmarshal(rec.Body.Bytes(), &s)
	if s.Jenis != "Beli Bahan / Stok" || s.Nominal != 125_000 || s.Tanggal != "2026-01-05" || s.Pihak != "Toko Sumber Rejeki" {
		t.Fatalf("saran %+v", s)
	}
	if !strings.Contains(diterima, `"image_url"`) || !strings.Contains(diterima, "data:image/png;base64,") || !strings.Contains(diterima, `"gpt-4o-mini"`) {
		t.Fatalf("permintaan ke model tidak memuat gambar: %.300s", diterima)
	}
	if len(srv.tenants[1].txs) != sebelum {
		t.Fatal("baca nota tidak boleh menyimpan transaksi")
	}
	if rec := unggahFoto(t, srv, owner, []byte("bukan gambar")); rec.Code != http.StatusBadRequest {
		t.Fatalf("berkas bukan gambar harus ditolak: %d", rec.Code)
	}
}

func TestRapikanNota_JenisDanTanggalAman(t *testing.T) {
	s := ai.RapikanNota(ai.SaranNota{Jenis: "Belanja Aneh", Tanggal: "2099-01-01", Nominal: -5}, "2026-10-22")
	if s.Jenis != "" || s.Tanggal != "2026-10-22" || s.Nominal != 0 || s.Yakin != "sedang" || len(s.Catatan) != 3 {
		t.Fatalf("%+v", s)
	}
}
