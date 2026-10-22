package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"pajak/internal/llm"
)

// llmNota: model palsu yang selalu membaca foto sebagai nota belanja 10 Jan 2026 senilai 40.500.
func llmNota(t *testing.T) *llm.Client {
	t.Helper()
	palsu := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []map[string]any{{"message": map[string]string{"role": "assistant",
			"content": `{"tanggal":"2026-01-10","jenis":"Biaya Lain","nominal":40500,"pihak":"Toko Sumber Rejeki","keterangan":"ongkir","lunas":true,"yakin":"tinggi","alasan":"nota belanja"}`}}}})
	}))
	t.Cleanup(palsu.Close)
	return llm.New(llm.Config{APIKey: "uji", BaseURL: palsu.URL, Model: "model-uji"})
}

func bacaFotoNota(t *testing.T, srv *Server, c *http.Cookie) hasilNota {
	t.Helper()
	rec := unggahFoto(t, srv, c, png1x1)
	if rec.Code != http.StatusOK {
		t.Fatalf("baca nota: %d %s", rec.Code, rec.Body)
	}
	var h hasilNota
	if err := json.Unmarshal(rec.Body.Bytes(), &h); err != nil {
		t.Fatal(err)
	}
	return h
}

const catatNota = `{"tanggal":"2026-01-10","pihak":"Toko Sumber Rejeki","keterangan":"ongkir","bukti":%q,"baris":[{"akun":"6190","debit":40500},{"akun":"1101","kredit":40500}]}`

func catatDenganBukti(t *testing.T, srv *Server, c *http.Cookie, bukti string) int {
	t.Helper()
	rec := do(srv, c, http.MethodPost, "/api/v1/catat", fmt.Sprintf(catatNota, bukti))
	if rec.Code != http.StatusCreated {
		t.Fatalf("catat: %d %s", rec.Code, rec.Body)
	}
	var r catatResp
	_ = json.Unmarshal(rec.Body.Bytes(), &r)
	return r.Grup
}

func usahaNota(t *testing.T) (*Server, *http.Cookie) {
	t.Helper()
	srv, qa := usahaBaru(t)
	if rec := do(srv, qa, http.MethodPut, "/api/v1/opening", `{"tanggal":"2026-01-01","rincian":[{"nama":"Kas","kategori":"kas","nominal":1000000},{"nama":"Modal Pemilik","kategori":"modal","nominal":1000000}]}`); rec.Code != http.StatusOK {
		t.Fatalf("saldo awal: %d %s", rec.Code, rec.Body)
	}
	srv.llmGambar = llmNota(t)
	return srv, qa
}

// Pesan gagal untuk pengguna tidak boleh membocorkan alamat penyedia AI, nama model, atau nama env var.
func TestBacaNota_PesanGagalTanpaDetailInternal(t *testing.T) {
	palsu := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":{"message":"[rahasia/model-x] upstream 404"}}`))
	}))
	defer palsu.Close()
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")

	srv.llmGambar = llm.New(llm.Config{APIKey: "uji", BaseURL: palsu.URL, Model: "model-x"})
	rec := unggahFoto(t, srv, owner, png1x1)
	body := rec.Body.String()
	if rec.Code != http.StatusBadGateway || !strings.Contains(body, "AI gagal membaca foto") {
		t.Fatalf("gagal AI: %d %s", rec.Code, body)
	}
	for _, bocor := range []string{palsu.URL, "model-x", "rahasia", "127.0.0.1"} {
		if strings.Contains(body, bocor) {
			t.Fatalf("pesan membocorkan %q: %s", bocor, body)
		}
	}

	srv.llmGambar = llm.New(llm.Config{})
	if rec := unggahFoto(t, srv, owner, png1x1); rec.Code != http.StatusServiceUnavailable || strings.Contains(rec.Body.String(), "LLM_") {
		t.Fatalf("belum aktif: %d %s", rec.Code, rec.Body)
	}
}

// Foto nota yang dicatat tersimpan sebagai bukti grup beserta jejaknya (saran AI, siapa, kapan),
// bisa dilihat ulang, memicu peringatan bila difoto lagi, dan ikut terhapus bersama catatannya.
func TestBuktiNota_SimpanLihatDuplikatHapus(t *testing.T) {
	srv, qa := usahaNota(t)
	h := bacaFotoNota(t, srv, qa)
	if len(h.Bukti) != 64 || h.Nominal != 40_500 || len(h.Duplikat) != 0 {
		t.Fatalf("hasil baca pertama: %+v", h)
	}
	grup := catatDenganBukti(t, srv, qa, h.Bukti)

	tid := currentTenantID(t, srv, qa)
	dir := filepath.Join(os.Getenv("DATA_DIR"), "bukti", strconv.Itoa(tid))
	foto, err := os.ReadFile(filepath.Join(dir, strconv.Itoa(grup)+".png"))
	if err != nil || !bytes.Equal(foto, png1x1) {
		t.Fatalf("foto bukti tidak tersimpan: %v", err)
	}
	var m buktiMeta
	raw, _ := os.ReadFile(filepath.Join(dir, strconv.Itoa(grup)+".json"))
	if err := json.Unmarshal(raw, &m); err != nil || m.Hash != h.Bukti || m.Saran.Pihak != "Toko Sumber Rejeki" ||
		m.Saran.Model != "model-uji" || m.DisimpanEmail == "" || m.DisimpanPada.IsZero() || m.DibacaPada.IsZero() {
		t.Fatalf("jejak bukti: %v %+v", err, m)
	}

	url := "/api/v1/catat/" + strconv.Itoa(grup)
	if g := getJSON[catatResp](t, srv, qa, url); g.FotoNota == nil || g.FotoNota.DisimpanOleh == "" || g.FotoNota.Yakin != "tinggi" {
		t.Fatalf("info foto nota di catatan: %+v", g.FotoNota)
	}
	rec := do(srv, qa, http.MethodGet, url+"/bukti", "")
	if rec.Code != http.StatusOK || !bytes.Equal(rec.Body.Bytes(), png1x1) || rec.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("lihat bukti: %d %q", rec.Code, rec.Header().Get("Content-Type"))
	}

	lagi := bacaFotoNota(t, srv, qa)
	if len(lagi.Duplikat) != 1 || lagi.Duplikat[0].Grup != grup || !strings.Contains(lagi.Duplikat[0].Alasan, "foto yang sama") {
		t.Fatalf("foto yang sama harus diperingatkan sekali per catatan: %+v", lagi.Duplikat)
	}

	if rec := do(srv, qa, http.MethodDelete, url, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("hapus: %d %s", rec.Code, rec.Body)
	}
	if _, err := os.Stat(filepath.Join(dir, strconv.Itoa(grup)+".png")); !os.IsNotExist(err) {
		t.Fatal("foto bukti harus ikut terhapus")
	}
	if rec := do(srv, qa, http.MethodGet, url+"/bukti", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("bukti catatan terhapus: %d", rec.Code)
	}
}

// Token bukti asing tidak menggagalkan penyimpanan, catatannya hanya tidak punya foto. Catatan
// dengan tanggal & nominal sama tanpa foto bukan urusan baca foto: itu dicegat cek dobel
// (finance.Mirip / cariMirip) saat menyimpan, jadi di sini tidak ada peringatan kedua.
func TestBuktiNota_TokenAsingTanpaPeringatanGanda(t *testing.T) {
	srv, qa := usahaNota(t)
	grup := catatDenganBukti(t, srv, qa, "token-asing")
	if g := getJSON[catatResp](t, srv, qa, "/api/v1/catat/"+strconv.Itoa(grup)); g.FotoNota != nil {
		t.Fatalf("token asing tidak boleh menghasilkan bukti: %+v", g.FotoNota)
	}
	if h := bacaFotoNota(t, srv, qa); len(h.Duplikat) != 0 {
		t.Fatalf("tanggal+nominal sama sudah ditangani cek dobel saat simpan: %+v", h.Duplikat)
	}
}

// Bukti hanya bisa dibuka oleh usaha pemiliknya.
func TestBuktiNota_UsahaLainTidakBisaMelihat(t *testing.T) {
	srv, qa := usahaNota(t)
	grup := catatDenganBukti(t, srv, qa, bacaFotoNota(t, srv, qa).Bukti)
	lain := login(t, srv, "owner@kedaikopisenja.id")
	if rec := do(srv, lain, http.MethodGet, "/api/v1/catat/"+strconv.Itoa(grup)+"/bukti", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("usaha lain: %d", rec.Code)
	}
}

// Tanpa DATA_DIR bukti disimpan di memori; Kasir yang boleh mencatat juga boleh melihat buktinya.
func TestBuktiNota_TanpaDataDirDanKasirBisaLihat(t *testing.T) {
	t.Setenv("DATA_DIR", "")
	srv := NewServer()
	srv.llmGambar = llmNota(t)
	kasir := login(t, srv, "kasir@kedaikopisenja.id")
	grup := catatDenganBukti(t, srv, kasir, bacaFotoNota(t, srv, kasir).Bukti)
	rec := do(srv, kasir, http.MethodGet, "/api/v1/catat/"+strconv.Itoa(grup)+"/bukti", "")
	if rec.Code != http.StatusOK || !bytes.Equal(rec.Body.Bytes(), png1x1) {
		t.Fatalf("kasir lihat bukti: %d", rec.Code)
	}
}

func currentTenantID(t *testing.T, srv *Server, c *http.Cookie) int {
	t.Helper()
	me := getJSON[meResp](t, srv, c, "/api/v1/auth/me")
	if me.User.TenantID == 0 {
		t.Fatal("tenant_id tidak terbaca dari /auth/me")
	}
	return me.User.TenantID
}
