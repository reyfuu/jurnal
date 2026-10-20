package auth

import (
	"errors"
	"testing"
)

func TestKelolaUser_UbahKontakResetHapusUsaha(t *testing.T) {
	s := NewStore()
	a, _ := s.Create(Input{Name: "Ani", Email: "ani@x.id", Role: Owner, Active: true, Password: "rahasia123", TenantID: 5, WA: "0811"})
	b, _ := s.Create(Input{Name: "Budi", Email: "budi@x.id", Role: Kasir, Active: true, Password: "rahasia123", TenantID: 5})
	c, _ := s.Create(Input{Name: "Cici", Email: "cici@x.id", Role: Owner, Active: true, Password: "rahasia123", TenantID: 6})
	tok, _, err := s.Login("ani@x.id", "rahasia123")
	if err != nil {
		t.Fatal(err)
	}

	u, err := s.UbahKontak(a.ID, "Ani S", "ani.baru@x.id", " 0812 ")
	if err != nil || u.Name != "Ani S" || u.WA != "0812" || u.Role != Owner {
		t.Fatalf("ubah kontak: %+v %v", u, err)
	}
	if _, ok := s.UserBySession(tok); ok {
		t.Error("sesi lama tetap hidup setelah email diganti")
	}
	if _, err := s.UbahKontak(a.ID, "Ani", "cici@x.id", ""); !errors.Is(err, ErrEmailTaken) {
		t.Errorf("email dipakai user lain: %v", err)
	}

	pw, err := s.ResetPassword(b.ID)
	if err != nil || len(pw) != 12 {
		t.Fatalf("reset: %q %v", pw, err)
	}
	if _, _, err := s.Login("budi@x.id", "rahasia123"); err == nil {
		t.Error("password lama masih berlaku")
	}
	if _, _, err := s.Login("budi@x.id", pw); err != nil {
		t.Errorf("password sementara: %v", err)
	}

	if n := s.HapusUsaha(5); n != 2 {
		t.Fatalf("hapus usaha 5: %d user", n)
	}
	if _, ok := s.Get(a.ID); ok {
		t.Error("user usaha 5 masih ada")
	}
	if _, ok := s.Get(c.ID); !ok || len(s.List(5)) != 0 || len(s.List(1)) == 0 {
		t.Error("user usaha lain ikut terhapus")
	}
	if s.HapusUsaha(0) != 0 {
		t.Error("akun tim (usaha 0) tidak boleh terhapus lewat HapusUsaha")
	}
}
