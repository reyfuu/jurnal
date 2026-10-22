package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"
)

// Lupa password (checklist tester No 15): token sekali pakai, berlaku 30 menit, dikirim lewat
// email. Yang disimpan hanya hash token, di memori: bila server dimulai ulang, pengguna cukup
// meminta tautan baru. Berhasil reset = semua sesi lama berakhir.

const umurTokenReset = 30 * time.Minute

var ErrTokenReset = errors.New("tautan atur ulang password tidak berlaku lagi; minta tautan baru")

type tokenReset struct {
	userID int
	sampai time.Time
}

func hashToken(t string) string { h := sha256.Sum256([]byte(t)); return hex.EncodeToString(h[:]) }

// MintaReset membuat token untuk email akun aktif; ok=false bila tidak ada (pemanggil tetap
// membalas sama supaya keberadaan akun tidak bocor). Token lama milik akun itu dibatalkan.
func (s *Store) MintaReset(email string) (token string, u User, ok bool) {
	email = normEmail(email)
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range s.users {
		if r.Email != email || !r.Active {
			continue
		}
		if s.reset == nil {
			s.reset = map[string]tokenReset{}
		}
		s.batalkanResetLocked(r.ID)
		token = rand.Text()
		s.reset[hashToken(token)] = tokenReset{r.ID, s.Now().Add(umurTokenReset)}
		return token, r.User, true
	}
	return "", User{}, false
}

// batalkanResetLocked: token lupa password milik user ini (dan yang sudah kedaluwarsa) gugur.
// s.mu dipegang.
func (s *Store) batalkanResetLocked(userID int) {
	kini := s.Now()
	for h, tr := range s.reset {
		if tr.userID == userID || kini.After(tr.sampai) {
			delete(s.reset, h)
		}
	}
}

// ResetDenganToken mengganti password lewat token yang masih berlaku, lalu mengakhiri semua sesi.
func (s *Store) ResetDenganToken(token, password string) error {
	if len(password) < 8 {
		return ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	h := hashToken(token)
	tr, ok := s.reset[h]
	if !ok || s.Now().After(tr.sampai) {
		delete(s.reset, h)
		return ErrTokenReset
	}
	delete(s.reset, h)
	r, ok := s.users[tr.userID]
	if !ok || !r.Active {
		return ErrTokenReset
	}
	r.hash = hashPassword(password, r.salt)
	s.revoke(r.ID)
	return nil
}
