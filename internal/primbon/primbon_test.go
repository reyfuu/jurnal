package primbon

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

const fixture = "testdata/katalog"

func muat(t *testing.T) *Data {
	t.Helper()
	d, err := Muat(fixture)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func ambil(t *testing.T, d *Data, uuid string) Peraturan {
	t.Helper()
	p, ok := d.Ambil(uuid)
	if !ok {
		t.Fatalf("%s tidak ada", uuid)
	}
	return p
}

func uuids(items []Peraturan) []string {
	var out []string
	for _, p := range items {
		out = append(out, p.UUID)
	}
	return out
}

func TestMuatMenormalkanData(t *testing.T) {
	d := muat(t)
	if d.Total() != 8 {
		t.Fatalf("total: %d", d.Total())
	}
	cek := []struct {
		uuid string
		ok   func(Peraturan) bool
		desc string
	}{
		{"a1", func(p Peraturan) bool {
			return p.Jenis == "PMK" && p.JenisLabel == "Peraturan Menteri Keuangan" && slices.Equal(p.Kategori, []string{"KUP", "PPh"}) &&
				slices.Equal(p.KategoriLabel, []string{"Ketentuan Umum Perpajakan", "Pajak Penghasilan"}) && p.Format == "pdf" && p.Tahun == 2024 &&
				p.Ditetapkan == "2024-02-12" && p.Halaman == 13 && p.SumberURL == "https://jdih.kemenkeu.go.id/dok/pmk-10-tahun-2024"
		}, "PMK, dua kategori, PDF tersedia, sumber halaman JDIH"},
		{"a2", func(p Peraturan) bool {
			return p.Jenis == "PERDIRJEN" && p.Format == "htm" && p.Lampiran && p.Status == "Diubah" &&
				p.SumberURL == "https://www.pajak.go.id/id/peraturan/faktur-pajak"
		}, "naskah HTML + lampiran PDF, sumber pajak.go.id"},
		{"a3", func(p Peraturan) bool {
			return p.Jenis == "UU" && p.Status == "Berlaku" && p.Diundangkan == "" && p.Format == ""
		}, "status huruf kecil dirapikan, tanggal 1900 dibuang, berkas hilang = belum tersedia"},
		{"a4", func(p Peraturan) bool { return p.Jenis == "PERDIRJEN" && p.SumberLN == "" && p.Format == "" },
			"jenis kosong ditebak dari kode KEP-, catatan pipeline bukan sumber LN"},
		{"a5", func(p Peraturan) bool { return p.Tahun == 2024 && p.Status == "" && p.Format == "" },
			"tahun dari tanggal penetapan, path ../ ditolak"},
		{"a6", func(p Peraturan) bool { return p.Jenis == "PP" && p.Abstrak && p.Status == "Dicabut" }, "hanya abstrak"},
		{"a7", func(p Peraturan) bool { return p.Jenis == "KMK" }, "Keputusan Menteri = KMK"},
		{"a8", func(p Peraturan) bool { return p.Jenis == "SE_DJP" && p.Tahun == 2022 }, "tahun dari kode"},
	}
	for _, c := range cek {
		if p := ambil(t, d, c.uuid); !c.ok(p) {
			t.Errorf("%s (%s): %+v", c.uuid, c.desc, p)
		}
	}
}

func TestFacetSeluruhKatalog(t *testing.T) {
	f := muat(t).Cari(Query{Kategori: []string{"KURS"}}).Facet
	jumlah := func(opsi []Opsi, kode string) int {
		for _, o := range opsi {
			if o.Kode == kode {
				return o.Jumlah
			}
		}
		return -1
	}
	if jumlah(f.Kategori, "PPh") != 3 || jumlah(f.Kategori, "KUP") != 3 || jumlah(f.Kategori, "PPN") != 1 {
		t.Errorf("facet kategori dihitung dari seluruh katalog, bukan hasil filter: %+v", f.Kategori)
	}
	if f.Kategori[0].Kode != "PPN" || f.Kategori[0].Label != "Pajak Pertambahan Nilai" {
		t.Errorf("urutan & label kategori mengikuti daftar baku: %+v", f.Kategori[0])
	}
	if jumlah(f.Jenis, "PERDIRJEN") != 2 || f.Jenis[0].Kode != "UU" {
		t.Errorf("facet jenis: %+v", f.Jenis)
	}
	if jumlah(f.Status, "Berlaku") != 5 || jumlah(f.Status, "") != -1 || f.Status[0].Kode != "Berlaku" {
		t.Errorf("facet status (tanpa status kosong, Berlaku dulu): %+v", f.Status)
	}
	if f.TahunMin != 2021 || f.TahunMax != 2026 {
		t.Errorf("rentang tahun: %d–%d", f.TahunMin, f.TahunMax)
	}
}

func TestCariFilterDanPencarian(t *testing.T) {
	d := muat(t)
	cases := []struct {
		q    Query
		want []string
	}{
		{Query{Kategori: []string{"PPN", "KURS"}}, []string{"a7", "a2"}},                      // OR di dalam grup
		{Query{Kategori: []string{"PPh"}, Status: []string{"Berlaku"}}, []string{"a1", "a3"}}, // AND antar grup
		{Query{Jenis: []string{"PERDIRJEN", "SE_DJP"}}, []string{"a4", "a2", "a8"}},           // terbaru; tanpa tanggal di akhir tahunnya
		{Query{Dari: 2022, Sampai: 2022}, []string{"a6", "a2", "a8"}},                         // rentang tahun
		{Query{Q: "pmk 2024"}, []string{"a5", "a1"}},                                          // semua kata harus muncul
		{Query{Q: "  FAKTUR  "}, []string{"a2"}},                                              // tanpa beda huruf besar
		{Query{Q: "per-03/pj"}, []string{"a2"}},                                               // cari nomor
		{Query{Q: "undang-undang"}, []string{"a3"}},
		{Query{Q: "pajak penghasilan"}, []string{"a1", "a6", "a3"}},                     // label kategori ikut dicari
		{Query{Q: "pph harmonisasi"}, []string{"a3"}},                                   // kode kategori ikut dicari
		{Query{Q: "10"}, []string{"a1"}},                                                // angka utuh: bukan 010, 105, atau 2010
		{Query{Q: "pmk 105"}, []string{"a5"}},                                           // label jenis ikut dicari
		{Query{Urut: "abjad", Kategori: []string{"PPh"}}, []string{"a3", "a6", "a1"}},   // judul A–Z
		{Query{Urut: "terlama", Kategori: []string{"KUP"}}, []string{"a8", "a1", "a4"}}, // tahun naik
		{Query{Kategori: []string{"TIDAK_ADA"}}, nil},
	}
	for _, c := range cases {
		h := d.Cari(c.q)
		if got := uuids(h.Items); !slices.Equal(got, c.want) || h.Total != len(c.want) {
			t.Errorf("%+v: got %v (total %d), want %v", c.q, got, h.Total, c.want)
		}
	}
	if got := uuids(d.Cari(Query{}).Items); !slices.Equal(got, []string{"a7", "a4", "a5", "a1", "a6", "a2", "a8", "a3"}) {
		t.Errorf("terbaru: tahun lalu tanggal, menurun: %v", got)
	}
}

func TestCariPaginasi(t *testing.T) {
	dir := t.TempDir()
	var b strings.Builder
	b.WriteString("uuid,kategori,kode,judul,jenis,tahun,status,file\n")
	for i := range 45 {
		fmt.Fprintf(&b, "u%02d,KUP,PMK %d,Judul %02d,Peraturan Menteri,%d,Berlaku,\n", i, i, i, 2000+i)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.csv"), []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	d, err := Muat(dir)
	if err != nil {
		t.Fatal(err)
	}
	h := d.Cari(Query{Halaman: 3})
	if h.Total != 45 || h.TotalHalaman != 3 || h.Halaman != 3 || len(h.Items) != 5 || h.Items[0].UUID != "u04" {
		t.Fatalf("halaman 3: total %d, %d/%d, %d item, pertama %s", h.Total, h.Halaman, h.TotalHalaman, len(h.Items), h.Items[0].UUID)
	}
	if h := d.Cari(Query{Halaman: 99}); h.Halaman != 3 {
		t.Errorf("halaman melebihi batas dijepit ke terakhir: %d", h.Halaman)
	}
	if h := d.Cari(Query{Halaman: -1}); h.Halaman != 1 || len(h.Items) != PerHalaman {
		t.Errorf("halaman < 1: %d, %d item", h.Halaman, len(h.Items))
	}
	if h := d.Cari(Query{Q: "tidak ada"}); h.TotalHalaman != 1 || h.Halaman != 1 || h.Items == nil {
		t.Errorf("hasil kosong tetap 1 halaman dan items [] bukan null: %+v", h)
	}
}

func TestBerkasAmanDanLampiran(t *testing.T) {
	d := muat(t)
	cases := []struct {
		uuid     string
		lampiran bool
		akhiran  string // "" = tidak boleh ada berkas
	}{
		{"a1", false, filepath.Join("kup", "pmk-10-2024.pdf")},
		{"a1", true, ""},
		{"a2", false, filepath.Join("ppn", "per-03-pj-2022.htm")},
		{"a2", true, filepath.Join("ppn", "per-03-pj-2022-lampiran.pdf")},
		{"a3", false, ""}, // berkas belum ada
		{"a4", false, ""}, // tanpa-pdf
		{"a5", false, ""}, // ../rahasia.pdf
		{"zz", false, ""},
	}
	for _, c := range cases {
		p, ok := d.Berkas(c.uuid, c.lampiran)
		if c.akhiran == "" {
			if ok {
				t.Errorf("%s lampiran=%v: tidak boleh ada berkas, dapat %s", c.uuid, c.lampiran, p)
			}
			continue
		}
		if !ok || !strings.HasSuffix(p, c.akhiran) {
			t.Errorf("%s lampiran=%v: %q %v", c.uuid, c.lampiran, p, ok)
		}
	}
}

func TestKatalogMuatUlangSaatIndexBerubah(t *testing.T) {
	if _, err := Buka(filepath.Join(t.TempDir(), "kosong")).Data(); err == nil {
		t.Fatal("folder tanpa index.csv harus error")
	}
	dir := t.TempDir()
	csv := filepath.Join(dir, "index.csv")
	isi := "uuid,kode,judul,jenis,tahun,status,file\nb1,PMK 1,Satu,PMK,2024,Berlaku,\n"
	if err := os.WriteFile(csv, []byte(isi), 0o600); err != nil {
		t.Fatal(err)
	}
	k := Buka(dir)
	d, err := k.Data()
	if err != nil || d.Total() != 1 {
		t.Fatalf("muat awal: %v %v", d, err)
	}
	if d2, _ := k.Data(); d2 != d {
		t.Error("index tidak berubah: data yang sama dipakai ulang")
	}
	if err := os.WriteFile(csv, []byte(isi+"b2,PMK 2,Dua,PMK,2025,Berlaku,\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(time.Minute)
	if err := os.Chtimes(csv, later, later); err != nil {
		t.Fatal(err)
	}
	if d, err := k.Data(); err != nil || d.Total() != 2 {
		t.Fatalf("index baru harus termuat tanpa restart: %v %v", d, err)
	}
}
