package auth

import (
	"testing"
	"time"
)

// Checklist tester No 15: lupa password lewat tautan sekali pakai yang kedaluwarsa 30 menit.
func TestResetLewatToken(t *testing.T) {
	s := NewStore()
	kini := time.Date(2026, 10, 22, 10, 0, 0, 0, time.UTC)
	s.Now = func() time.Time { return kini }
	sesi, _, err := s.Login("owner@kedaikopisenja.id", DemoPassword)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, ok := s.MintaReset("tidak-ada@x.id"); ok {
		t.Fatal("email tak terdaftar tidak boleh mendapat token")
	}
	token, u, ok := s.MintaReset(" Owner@KedaiKopiSenja.id ")
	if !ok || token == "" || u.Email != "owner@kedaikopisenja.id" {
		t.Fatalf("minta reset: %q %+v %v", token, u, ok)
	}
	if err := s.ResetDenganToken(token, "pendek"); err == nil {
		t.Fatal("password < 8 karakter harus ditolak")
	}
	if err := s.ResetDenganToken(token, "rahasia-baru-1"); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.UserBySession(sesi); ok {
		t.Fatal("sesi lama harus berakhir")
	}
	if _, _, err := s.Login("owner@kedaikopisenja.id", "rahasia-baru-1"); err != nil {
		t.Fatalf("login password baru: %v", err)
	}
	if err := s.ResetDenganToken(token, "lagi-lagi-123"); err == nil {
		t.Fatal("token hanya sekali pakai")
	}
	token2, _, _ := s.MintaReset("owner@kedaikopisenja.id")
	kini = kini.Add(31 * time.Minute)
	if err := s.ResetDenganToken(token2, "lagi-lagi-123"); err == nil {
		t.Fatal("token kedaluwarsa harus ditolak")
	}
}
