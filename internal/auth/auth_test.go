package auth_test

import (
	"errors"
	"testing"
	"time"

	"pajak/internal/auth"
)

func TestLogin_SeedOwner(t *testing.T) {
	s := auth.NewStore()
	token, u, err := s.Login("OWNER@kedaikopisenja.id ", auth.DemoPassword)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if u.Role != auth.Owner || token == "" {
		t.Fatalf("want owner + token, got %+v %q", u, token)
	}
	got, ok := s.UserBySession(token)
	if !ok || got.ID != u.ID {
		t.Fatalf("session lookup failed: %+v %v", got, ok)
	}
	s.Logout(token)
	if _, ok := s.UserBySession(token); ok {
		t.Fatal("session still valid after logout")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	s := auth.NewStore()
	if _, _, err := s.Login("owner@kedaikopisenja.id", "salah-password"); !errors.Is(err, auth.ErrBadCredentials) {
		t.Fatalf("want ErrBadCredentials, got %v", err)
	}
}

func TestCreateUpdateDelete(t *testing.T) {
	s := auth.NewStore()
	u, err := s.Create(auth.Input{Name: "Rina", Email: "rina@kopi.id", Role: auth.Kasir, Active: true, Password: "rahasia123"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := s.Create(auth.Input{Name: "Rina 2", Email: "RINA@kopi.id", Role: auth.Kasir, Active: true, Password: "rahasia123"}); !errors.Is(err, auth.ErrEmailTaken) {
		t.Fatalf("duplicate email: want ErrEmailTaken, got %v", err)
	}

	// Update tanpa password: password lama tetap berlaku.
	u, err = s.Update(u.ID, auth.Input{Name: "Rina S", Email: "rina@kopi.id", Role: auth.Admin, Active: true})
	if err != nil || u.Role != auth.Admin || u.Name != "Rina S" {
		t.Fatalf("update: %+v %v", u, err)
	}
	token, _, err := s.Login("rina@kopi.id", "rahasia123")
	if err != nil {
		t.Fatalf("login after update: %v", err)
	}

	// Nonaktif: sesi lama mati dan tidak bisa login.
	if _, err := s.Update(u.ID, auth.Input{Name: "Rina S", Email: "rina@kopi.id", Role: auth.Admin, Active: false}); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.UserBySession(token); ok {
		t.Fatal("inactive user session still valid")
	}
	if _, _, err := s.Login("rina@kopi.id", "rahasia123"); !errors.Is(err, auth.ErrBadCredentials) {
		t.Fatalf("inactive login: want ErrBadCredentials, got %v", err)
	}

	if err := s.Delete(u.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(u.ID); !errors.Is(err, auth.ErrNotFound) {
		t.Fatalf("delete twice: want ErrNotFound, got %v", err)
	}
}

func TestValidation(t *testing.T) {
	s := auth.NewStore()
	bad := []auth.Input{
		{Name: "", Email: "a@b.id", Role: auth.Kasir, Password: "rahasia123"},
		{Name: "A", Email: "bukan-email", Role: auth.Kasir, Password: "rahasia123"},
		{Name: "A", Email: "a@b.id", Role: "superadmin", Password: "rahasia123"},
		{Name: "A", Email: "a@b.id", Role: auth.Kasir, Password: "pendek"},
	}
	for i, in := range bad {
		if _, err := s.Create(in); !errors.Is(err, auth.ErrInvalid) {
			t.Errorf("case %d: want ErrInvalid, got %v", i, err)
		}
	}
}

func TestLastOwnerProtected(t *testing.T) {
	s := auth.NewStore()
	_, owner, _ := s.Login("owner@kedaikopisenja.id", auth.DemoPassword)
	if err := s.Delete(owner.ID); !errors.Is(err, auth.ErrLastOwner) {
		t.Fatalf("delete last owner: want ErrLastOwner, got %v", err)
	}
	if _, err := s.Update(owner.ID, auth.Input{Name: owner.Name, Email: owner.Email, Role: auth.Kasir, Active: true}); !errors.Is(err, auth.ErrLastOwner) {
		t.Fatalf("demote last owner: want ErrLastOwner, got %v", err)
	}
}

func TestCan(t *testing.T) {
	if !auth.Owner.Can("pengguna") || auth.Admin.Can("pengguna") {
		t.Fatal("only owner manages users")
	}
	if !auth.Kasir.Can("transaksi") || auth.Kasir.Can("laporan-keuangan") {
		t.Fatal("kasir: transaksi yes, laporan no")
	}
}

func TestSession_Expires(t *testing.T) {
	s := auth.NewStore()
	now := time.Now()
	s.Now = func() time.Time { return now }
	token, _, err := s.Login("owner@kedaikopisenja.id", auth.DemoPassword)
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(auth.SessionTTL - time.Minute)
	if _, ok := s.UserBySession(token); !ok {
		t.Fatal("session should still be valid before TTL")
	}
	now = now.Add(2 * time.Minute)
	if _, ok := s.UserBySession(token); ok {
		t.Fatal("session should expire after TTL")
	}
}

func TestSession_RevokedOnPasswordChange(t *testing.T) {
	s := auth.NewStore()
	token, u, _ := s.Login("kasir@kedaikopisenja.id", auth.DemoPassword)
	if _, err := s.Update(u.ID, auth.Input{Name: u.Name, Email: u.Email, Role: u.Role, Active: true, Password: "passwordbaru1"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.UserBySession(token); ok {
		t.Fatal("old session must be revoked after password change")
	}
}

func TestNewStoreWithPassword(t *testing.T) {
	s := auth.NewStoreWithPassword("rahasia-prod-123")
	if _, _, err := s.Login("owner@kedaikopisenja.id", auth.DemoPassword); err == nil {
		t.Fatal("demo password must not work when seed password is set")
	}
	if _, _, err := s.Login("owner@kedaikopisenja.id", "rahasia-prod-123"); err != nil {
		t.Fatalf("seed password login: %v", err)
	}
}
