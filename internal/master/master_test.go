package master

import (
	"errors"
	"slices"
	"testing"
	"time"
)

func TestDefaultRoles_MasterPunyaSemuaIzin(t *testing.T) {
	st := NewState()
	st.Anggota = []Anggota{{UserID: 1, Role: RoleMaster}}
	ak, ok := st.Akses(1)
	if !ok {
		t.Fatal("anggota master tidak dikenali")
	}
	for _, i := range DaftarIzin {
		if !ak.Bisa(i.ID) {
			t.Errorf("master tidak punya izin %s", i.ID)
		}
	}
	if !ak.BolehKlien(99) {
		t.Error("master harus melihat semua klien")
	}
}

func TestAkses_KonsultanHanyaKlienDitugaskan(t *testing.T) {
	st := NewState()
	st.Anggota = []Anggota{{UserID: 2, Role: "konsultan", Klien: []int{3, 5}}}
	ak, _ := st.Akses(2)
	if !ak.BolehKlien(3) || !ak.BolehKlien(5) || ak.BolehKlien(4) {
		t.Errorf("penugasan salah: %+v", ak)
	}
	if ak.Bisa(Langganan) || ak.Bisa(Tim) || ak.Bisa(Audit) {
		t.Error("konsultan tidak boleh atur langganan, tim, atau audit")
	}
	if !ak.Bisa(Tiket) || !ak.Bisa(KlienLihat) {
		t.Error("konsultan harus bisa lihat klien & tiket")
	}
}

func TestAkses_BukanAnggota(t *testing.T) {
	st := NewState()
	if _, ok := st.Akses(7); ok {
		t.Error("user tanpa keanggotaan tidak boleh punya akses")
	}
}

func TestSimpanRole_Baru(t *testing.T) {
	st := NewState()
	r, err := st.SimpanRole(Role{Nama: "Akuntan Senior", Izin: []Izin{KlienLihat, Tiket, Langganan}})
	if err != nil {
		t.Fatal(err)
	}
	if r.ID != "akuntan-senior" {
		t.Errorf("id = %q", r.ID)
	}
	st.Anggota = []Anggota{{UserID: 4, Role: r.ID, Klien: []int{1}}}
	ak, _ := st.Akses(4)
	if !ak.Bisa(Langganan) || ak.Bisa(Tim) {
		t.Errorf("izin role baru tidak terpakai: %+v", ak.Role)
	}
	// nama sama → id unik
	r2, err := st.SimpanRole(Role{Nama: "Akuntan Senior", Izin: []Izin{Tiket}})
	if err != nil || r2.ID == r.ID {
		t.Errorf("role kedua bernama sama harus dapat id lain, dapat %q (%v)", r2.ID, err)
	}
}

func TestSimpanRole_Ubah(t *testing.T) {
	st := NewState()
	if _, err := st.SimpanRole(Role{ID: "konsultan", Nama: "Konsultan Pajak", Izin: []Izin{KlienLihat, Tiket, Langganan}}); err != nil {
		t.Fatal(err)
	}
	st.Anggota = []Anggota{{UserID: 2, Role: "konsultan"}}
	if ak, _ := st.Akses(2); !ak.Bisa(Langganan) || ak.Role.Nama != "Konsultan Pajak" {
		t.Errorf("perubahan role tidak tersimpan: %+v", ak.Role)
	}
}

func TestSimpanRole_Tolak(t *testing.T) {
	st := NewState()
	cases := map[string]Role{
		"master terkunci":  {ID: RoleMaster, Nama: "Master", Izin: []Izin{Tiket}},
		"nama kosong":      {Nama: "  ", Izin: []Izin{Tiket}},
		"izin tak dikenal": {Nama: "X", Izin: []Izin{"hapus_semua"}},
		"id tak ada":       {ID: "tidak-ada", Nama: "X"},
	}
	for nama, r := range cases {
		if _, err := st.SimpanRole(r); !errors.Is(err, ErrTidakValid) {
			t.Errorf("%s: err = %v", nama, err)
		}
	}
}

func TestHapusRole(t *testing.T) {
	st := NewState()
	st.Anggota = []Anggota{{UserID: 2, Role: "konsultan"}}
	if err := st.HapusRole("konsultan"); !errors.Is(err, ErrRoleDipakai) {
		t.Errorf("role dipakai: err = %v", err)
	}
	if err := st.HapusRole(RoleMaster); !errors.Is(err, ErrTidakValid) {
		t.Errorf("hapus master: err = %v", err)
	}
	if err := st.HapusRole("cs"); err != nil {
		t.Fatal(err)
	}
	if slices.ContainsFunc(st.Roles, func(r Role) bool { return r.ID == "cs" }) {
		t.Error("role cs masih ada")
	}
}

func TestSetAnggota_MasterTerakhir(t *testing.T) {
	st := NewState()
	if err := st.SetAnggota(Anggota{UserID: 1, Role: RoleMaster}); err != nil {
		t.Fatal(err)
	}
	if err := st.SetAnggota(Anggota{UserID: 1, Role: "cs"}); !errors.Is(err, ErrMasterTerakhir) {
		t.Errorf("menurunkan master terakhir: err = %v", err)
	}
	if err := st.LepasAnggota(1); !errors.Is(err, ErrMasterTerakhir) {
		t.Errorf("melepas master terakhir: err = %v", err)
	}
	if err := st.SetAnggota(Anggota{UserID: 2, Role: RoleMaster}); err != nil {
		t.Fatal(err)
	}
	if err := st.SetAnggota(Anggota{UserID: 1, Role: "cs", Klien: []int{4, 4, 2}}); err != nil {
		t.Fatalf("sudah ada master lain: %v", err)
	}
	ak, _ := st.Akses(1)
	if !slices.Equal(ak.Klien, []int{2, 4}) {
		t.Errorf("klien harus unik & urut, dapat %v", ak.Klien)
	}
	if err := st.SetAnggota(Anggota{UserID: 3, Role: "tidak-ada"}); !errors.Is(err, ErrTidakValid) {
		t.Errorf("role tak ada: err = %v", err)
	}
}

func TestPenangan_TanpaYangMelihatSemua(t *testing.T) {
	st := NewState()
	st.Anggota = []Anggota{
		{UserID: 1, Role: RoleMaster},
		{UserID: 2, Role: "konsultan", Klien: []int{3}},
		{UserID: 3, Role: "cs", Klien: []int{3, 4}},
	}
	if got := st.Penangan(3); !slices.Equal(got, []int{2, 3}) {
		t.Errorf("penangan klien 3 = %v", got)
	}
	if got := st.Penangan(9); len(got) != 0 {
		t.Errorf("klien belum ditugaskan harus kosong, dapat %v", got)
	}
}

func TestCatat_IDBerurutDanTidakBerubah(t *testing.T) {
	st := NewState()
	a := st.Catat(Catatan{UserID: 1, Aksi: "Buka detail klien", KlienID: 3})
	b := st.Catat(Catatan{UserID: 1, Aksi: "Balas tiket BUG-0002", KlienID: 3})
	if a.ID != 1 || b.ID != 2 || len(st.Audit) != 2 {
		t.Errorf("id audit = %d,%d len %d", a.ID, b.ID, len(st.Audit))
	}
	if a.Waktu.IsZero() {
		t.Error("waktu kosong")
	}
}

func TestPerluPerhatian(t *testing.T) {
	now := time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)
	hari := func(n int) time.Time { return now.AddDate(0, 0, n) }
	cases := []struct {
		nama     string
		status   string
		sampai   time.Time
		terakhir time.Time
		want     bool
	}{
		{"aktif rajin", "aktif", hari(30), hari(-1), false},
		{"habis", "habis", hari(-3), hari(-1), true},
		{"trial habis 2 hari lagi", "trial", hari(2), hari(0), true},
		{"trial masih lama", "trial", hari(10), hari(0), false},
		{"tidak aktif 14 hari", "aktif", hari(60), hari(-14), true},
		{"tidak aktif 13 hari", "aktif", hari(60), hari(-13), false},
		{"belum pernah login", "trial", hari(10), time.Time{}, false},
	}
	for _, c := range cases {
		got, alasan := PerluPerhatian(c.status, c.sampai, c.terakhir, now)
		if got != c.want || got && alasan == "" {
			t.Errorf("%s: got %v (%q), want %v", c.nama, got, alasan, c.want)
		}
	}
}
