package main

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"pajak/internal/ai"
	"pajak/internal/finance"
)

// Input transaksi dari foto: foto nota/struk dibaca AI menjadi pratinjau isian transaksi. Tidak
// ada yang tersimpan di sini — pengguna memeriksa pratinjaunya lalu menyimpan lewat form biasa.
// Fotonya tidak disimpan di server.

const (
	maxFotoNota   = 6 << 20 // frontend sudah mengecilkan foto; ini batas aman
	maxBacaNotaAI = 100     // per usaha per 24 jam
)

var mimeNota = []string{"image/jpeg", "image/png", "image/webp", "image/gif"}

func (s *Server) handleBacaNota(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxFotoNota+(64<<10))
	f, _, err := r.FormFile("foto")
	if err != nil {
		http.Error(w, "foto nota tidak terbaca atau lebih dari 6 MB", http.StatusBadRequest)
		return
	}
	defer f.Close()
	foto, err := io.ReadAll(io.LimitReader(f, maxFotoNota+1))
	if err != nil || len(foto) == 0 || len(foto) > maxFotoNota {
		http.Error(w, "foto nota tidak terbaca atau lebih dari 6 MB", http.StatusBadRequest)
		return
	}
	mime := http.DetectContentType(foto)
	ok := false
	for _, m := range mimeNota {
		ok = ok || strings.HasPrefix(mime, m)
	}
	if !ok {
		http.Error(w, "kirim foto JPG, PNG, atau WebP", http.StatusBadRequest)
		return
	}
	if s.llmGambar == nil || !s.llmGambar.Enabled() {
		http.Error(w, "Baca foto belum aktif: isi LLM_API_KEY (atau LLM_VISION_*) di server. Sementara catat manual.", http.StatusServiceUnavailable)
		return
	}
	t := s.tenantOf(r)
	if !s.jatahAI("nota|"+strconv.Itoa(t.id), maxBacaNotaAI) {
		http.Error(w, "Kuota baca foto hari ini habis; catat manual dulu, besok bisa dipakai lagi.", http.StatusTooManyRequests)
		return
	}
	p := t.currentProfile()
	usaha := strings.TrimSpace(p.JenisUsaha + " · " + p.NamaUsaha)
	saran, err := ai.BacaNota(r.Context(), s.llmGambar, usaha, mime, foto, time.Now().In(finance.WIB).Format(time.DateOnly))
	if err != nil {
		http.Error(w, "AI gagal membaca foto ("+err.Error()+"). Coba foto yang lebih terang dan tegak, atau catat manual.", http.StatusBadGateway)
		return
	}
	writeJSON(w, saran)
}
