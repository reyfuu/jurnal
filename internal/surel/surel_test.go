package surel

import "testing"

func TestKirimMenolakTanpaKonfigurasiDanInjeksiHeader(t *testing.T) {
	if err := (Config{}).Kirim("a@b.id", "s", "i"); err == nil {
		t.Fatal("tanpa SMTP harus gagal")
	}
	c := Config{Host: "127.0.0.1", Port: "1", From: "x@y.id"}
	for _, ke := range []string{"a@b.id\r\nBcc: c@d.id", "a@b.id\nX: y"} {
		if err := c.Kirim(ke, "s", "i"); err == nil || err.Error() != "alamat atau subjek tidak valid" {
			t.Fatalf("injeksi header %q: %v", ke, err)
		}
	}
}
