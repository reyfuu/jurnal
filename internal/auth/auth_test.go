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
	u, err = s.Update(1, u.ID, auth.Input{Name: "Rina S", Email: "rina@kopi.id", Role: auth.Admin, Active: true})
	if err != nil || u.Role != auth.Admin || u.Name != "Rina S" {
		t.Fatalf("update: %+v %v", u, err)
	}
	token, _, err := s.Login("rina@kopi.id", "rahasia123")
	if err != nil {
		t.Fatalf("login after update: %v", err)
	}

	// Nonaktif: sesi lama mati dan tidak bisa login.
	if _, err := s.Update(1, u.ID, auth.Input{Name: "Rina S", Email: "rina@kopi.id", Role: auth.Admin, Active: false}); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.UserBySession(token); ok {
		t.Fatal("inactive user session still valid")
	}
	if _, _, err := s.Login("rina@kopi.id", "rahasia123"); !errors.Is(err, auth.ErrBadCredentials) {
		t.Fatalf("inactive login: want ErrBadCredentials, got %v", err)
	}

	if err := s.Delete(1, u.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(1, u.ID); !errors.Is(err, auth.ErrNotFound) {
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
	if err := s.Delete(1, owner.ID); !errors.Is(err, auth.ErrLastOwner) {
		t.Fatalf("delete last owner: want ErrLastOwner, got %v", err)
	}
	if _, err := s.Update(1, owner.ID, auth.Input{Name: owner.Name, Email: owner.Email, Role: auth.Kasir, Active: true}); !errors.Is(err, auth.ErrLastOwner) {
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
	if _, err := s.Update(1, u.ID, auth.Input{Name: u.Name, Email: u.Email, Role: u.Role, Active: true, Password: "passwordbaru1"}); err != nil {
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

func TestSnapshotRestore(t *testing.T) {
	s := auth.NewStore()
	token, _, _ := s.Login("owner@kedaikopisenja.id", auth.DemoPassword)
	u, err := s.Create(auth.Input{Name: "Tono", Email: "tono@x.id", Role: auth.Kasir, Active: true, Password: "rahasia123"})
	if err != nil {
		t.Fatal(err)
	}

	restored := auth.Restore(s.Snapshot())
	if _, _, err := restored.Login("tono@x.id", "rahasia123"); err != nil {
		t.Fatalf("password hash must survive restore: %v", err)
	}
	if got, ok := restored.UserBySession(token); !ok || got.Email != "owner@kedaikopisenja.id" {
		t.Fatal("session must survive restore")
	}
	// ID baru tidak boleh bentrok dengan user yang dipulihkan
	next, err := restored.Create(auth.Input{Name: "Ani", Email: "ani@x.id", Role: auth.Kasir, Active: true, Password: "rahasia123"})
	if err != nil || next.ID <= u.ID {
		t.Fatalf("next ID %d must be > %d (%v)", next.ID, u.ID, err)
	}
}

func TestNormalizeWA(t *testing.T) {
	for _, tc := range []struct {
		in, want string
		ok       bool
	}{
		{"0812-3456-7890", "6281234567890", true},
		{"+62 812 3456 7890", "6281234567890", true},
		{"6281234567890", "6281234567890", true},
		{"0812.3456.789", "628123456789", true},
		{"812345", "", false},            // terlalu pendek
		{"0212345678", "", false},        // bukan nomor seluler
		{"08123456789012345", "", false}, // terlalu panjang
		{"0812abc45678", "", false},
		{"", "", false},
	} {
		got, err := auth.NormalizeWA(tc.in)
		if (err == nil) != tc.ok || got != tc.want {
			t.Errorf("NormalizeWA(%q) = %q, %v; want %q ok=%v", tc.in, got, err, tc.want, tc.ok)
		}
	}
}

func TestMP_TanpaUsaha(t *testing.T) {
	s := auth.NewStore()
	u, err := s.Create(auth.Input{Name: "Andi", Email: "andi@masterpajak.id", Role: auth.MP, Active: true, Password: "rahasia123"})
	if err != nil {
		t.Fatal(err)
	}
	if u.TenantID != 0 {
		t.Errorf("akun tim auth.MP tidak boleh punya usaha, tenant_id = %d", u.TenantID)
	}
	if len(s.List(1)) != 3 || len(s.List(0)) != 1 {
		t.Errorf("List usaha 1 = %d user, List tim = %d", len(s.List(1)), len(s.List(0)))
	}
	// Update akun tim: tidak terkena aturan Owner terakhir.
	if _, err := s.Update(0, u.ID, auth.Input{Name: "Andi P", Email: u.Email, Role: auth.MP, Active: false}); err != nil {
		t.Fatalf("nonaktifkan akun tim: %v", err)
	}
	snap := s.Snapshot()
	if r := auth.Restore(snap); r.List(0)[0].TenantID != 0 {
		t.Error("auth.Restore mengubah tenant_id akun tim jadi usaha 1")
	}
}

func TestMP_TidakBisaDibuatDariUsaha(t *testing.T) {
	s := auth.NewStore()
	if _, err := s.Create(auth.Input{Name: "X", Email: "x@contoh.id", Role: auth.MP, Active: true, Password: "rahasia123", TenantID: 1}); !errors.Is(err, auth.ErrInvalid) {
		t.Errorf("Owner usaha membuat akun tim: err = %v", err)
	}
	if _, err := s.Update(1, 2, auth.Input{Name: "Dimas", Email: "admin@kedaikopisenja.id", Role: auth.MP, Active: true}); !errors.Is(err, auth.ErrInvalid) {
		t.Errorf("ubah user usaha jadi tim: err = %v", err)
	}
	if auth.MP.Can("dashboard") || !auth.MP.Can("panel-master") {
		t.Error("role mp hanya boleh view panel-master")
	}
}

func TestTouch_LastSeen(t *testing.T) {
	s := auth.NewStore()
	t0 := time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)
	if !s.Touch(1, t0) {
		t.Fatal("sentuhan pertama harus mencatat")
	}
	if s.Touch(1, t0.Add(4*time.Minute)) {
		t.Error("di bawah 5 menit tidak perlu dicatat ulang")
	}
	if !s.Touch(1, t0.Add(5*time.Minute)) {
		t.Error("5 menit kemudian harus tercatat")
	}
	if got := s.List(1)[0].LastSeen; !got.Equal(t0.Add(5 * time.Minute)) {
		t.Errorf("last_seen = %v", got)
	}
	if s.Touch(99, t0) {
		t.Error("user tidak ada")
	}
}

func TestFindByEmail(t *testing.T) {
	s := auth.NewStore()
	if u, ok := s.FindByEmail(" Owner@KedaiKopiSenja.id "); !ok || u.ID != 1 {
		t.Errorf("FindByEmail = %+v %v", u, ok)
	}
	if _, ok := s.FindByEmail("tidak@ada.id"); ok {
		t.Error("email tak ada ditemukan")
	}
}

// Role lihat hanya dipakai server untuk mode lihat-sebagai: tidak bisa diberikan ke user.
func TestLihat_TidakBisaDiberikan(t *testing.T) {
	s := auth.NewStore()
	if _, err := s.Create(auth.Input{Name: "X", Email: "x@contoh.id", Role: auth.Lihat, Active: true, Password: "rahasia123", TenantID: 1}); !errors.Is(err, auth.ErrInvalid) {
		t.Errorf("buat user role lihat: err = %v", err)
	}
	if _, err := s.Update(1, 2, auth.Input{Name: "Dimas", Email: "admin@kedaikopisenja.id", Role: auth.Lihat, Active: true}); !errors.Is(err, auth.ErrInvalid) {
		t.Errorf("ubah user jadi lihat: err = %v", err)
	}
	for _, v := range []string{"transaksi-kelola", "impor-data", "pengguna", "langganan", "lapor-bug", "kalkulator-pajak"} {
		if auth.Lihat.Can(v) {
			t.Errorf("role lihat tidak boleh punya %s", v)
		}
	}
	if !auth.Lihat.Can("laporan-keuangan") || !auth.Lihat.Can("transaksi") {
		t.Error("role lihat harus bisa membaca laporan & transaksi")
	}
}
