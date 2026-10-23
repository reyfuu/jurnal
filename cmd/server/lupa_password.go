package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"pajak/internal/auth"
)

// Lupa password (checklist tester No 15). Balasan permintaan selalu sama supaya keberadaan akun
// tidak bocor. Tautan memakai APP_URL (bukan header Host, supaya tautan tidak bisa dibelokkan
// ke domain lain); tanpa SMTP/APP_URL tautan hanya dicatat ke log (pengembangan lokal).

const pesanLupa = "Bila email itu terdaftar, tautan atur ulang password sudah dikirim. Tautan berlaku 30 menit."

func (s *Server) handleLupaPassword(w http.ResponseWriter, r *http.Request) {
	var req struct{ Email string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !strings.Contains(req.Email, "@") || len(req.Email) > 254 {
		http.Error(w, "isi email yang valid", http.StatusBadRequest)
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if !s.reserveAttempt("lupa|" + clientIP(r)) {
		http.Error(w, "terlalu banyak permintaan, coba lagi dalam 15 menit", http.StatusTooManyRequests)
		return
	}
	// Batas per email tidak memberi 429 (supaya tidak bisa dipakai mengunci reset orang lain atau
	// menebak akun): melewati batas = balasan sama, email tidak dikirim.
	if s.reserveAttempt("lupa|" + email) {
		if token, u, ok := s.users.MintaReset(email); ok {
			go s.kirimTautanReset(u, token) // dikirim di belakang: waktu balasan sama untuk email terdaftar atau tidak
		}
	}
	w.WriteHeader(http.StatusAccepted)
	writeJSON(w, map[string]string{"pesan": pesanLupa})
}

// kirimTautanReset: tautan lengkap hanya dicatat ke log di pengembangan lokal (APP_URL kosong /
// localhost); di produksi kegagalan dicatat tanpa token.
func (s *Server) kirimTautanReset(u auth.User, token string) {
	appURL := strings.TrimRight(os.Getenv("APP_URL"), "/")
	tautan := appURL + "/#/atur-password?token=" + url.QueryEscape(token)
	isi := "Halo " + u.Name + ",\n\nKami menerima permintaan atur ulang password akun MasterPajak AI Anda.\n" +
		"Buka tautan berikut (berlaku 30 menit, sekali pakai):\n\n" + tautan +
		"\n\nBila Anda tidak meminta ini, abaikan email ini; password Anda tidak berubah.\n"
	err := s.kirimSurel(u.Email, "Atur ulang password MasterPajak AI", isi)
	switch {
	case err == nil:
	case appURL == "" || strings.Contains(appURL, "localhost") || strings.Contains(appURL, "127.0.0.1"):
		log.Printf("lupa password %s (pengembangan, email tidak terkirim: %v): %s", u.Email, err, tautan)
	default:
		log.Printf("PERINGATAN: email atur ulang password ke %s gagal: %v", u.Email, err)
	}
}

func (s *Server) handleAturPassword(w http.ResponseWriter, r *http.Request) {
	var req struct{ Token, Password string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Token == "" {
		http.Error(w, "permintaan tidak valid", http.StatusBadRequest)
		return
	}
	if !s.reserveAttempt("atur|" + clientIP(r)) {
		http.Error(w, "terlalu banyak percobaan, coba lagi dalam 15 menit", http.StatusTooManyRequests)
		return
	}
	switch err := s.users.ResetDenganToken(req.Token, req.Password); {
	case errors.Is(err, auth.ErrInvalid):
		http.Error(w, "password minimal 8 karakter", http.StatusBadRequest)
		return
	case err != nil:
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.saveFor(nil); err != nil { // tanpa sesi, persisting tidak menyimpan; simpan sendiri
		http.Error(w, "Gagal menyimpan password, coba lagi sebentar lagi.", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
