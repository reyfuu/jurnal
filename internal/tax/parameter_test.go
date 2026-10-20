package tax

import (
	"errors"
	"testing"
)

// pakaiParameter memasang p selama satu tes, lalu mengembalikan bawaan.
func pakaiParameter(t *testing.T, ubah func(p *Parameter)) {
	t.Helper()
	p := ParameterBawaan()
	ubah(&p)
	if err := p.Validasi(); err != nil {
		t.Fatalf("parameter uji tidak valid: %v", err)
	}
	AktifkanParameter(p)
	t.Cleanup(func() { AktifkanParameter(ParameterBawaan()) })
}

func TestParameterBawaan_ValidDanSamaDenganAturanLama(t *testing.T) {
	p := ParameterBawaan()
	if err := p.Validasi(); err != nil {
		t.Fatal(err)
	}
	if got := p.PPNEfektifBps(); got != 1100 {
		t.Errorf("PPN efektif 12%% x 11/12 = %d bps, want 1100", got)
	}
	if p.PTKP["K/3"] != 72_000_000 || p.BadanTarifBps != 2200 || p.UMKMTarifBps != 50 || p.BatasPKP != 4_800_000_000 {
		t.Errorf("bawaan berubah: %+v", p)
	}
	if len(p.Norma) < 1000 || len(p.Objek) != len(JenisPajakList) {
		t.Errorf("norma %d baris, objek %d jenis", len(p.Norma), len(p.Objek))
	}
}

func TestAktifkanParameter_DipakaiSemuaHitungan(t *testing.T) {
	pakaiParameter(t, func(p *Parameter) {
		p.Pasal17[0].TarifBps = 1000 // lapis pertama 10%
		p.UMKMTarifBps = 100
		p.BatasPKP = 1_000_000_000
		p.PTKP["TK/0"] = 60_000_000
		p.Objek[0].Objek[5].TarifBps = 300 // 23-jasa 3%
	})
	if got := Pasal17(60_000_000); got != 6_000_000 {
		t.Errorf("Pasal17 = %d, want 6 jt", got)
	}
	if got := ppFinalOP(1_500_000_000); got != 10_000_000 {
		t.Errorf("PPh final OP = %d, want 1%% x 1 M", got)
	}
	if !CekPKP(1_200_000_000).WajibPKP {
		t.Error("batas PKP 1 M tidak dipakai")
	}
	r, _ := HitungPPh21(PPh21Input{PenghasilanBruto: 10_000_000, PTKPStatus: "TK/0", PunyaNPWP: true})
	if r.PTKP != 60_000_000 {
		t.Errorf("PTKP = %d", r.PTKP)
	}
	pr, _ := HitungPotong(PotongInput{Jenis: "pph23", Objek: "23-jasa", NilaiBruto: 1_000_000, PunyaNPWP: true})
	if pr.Pajak != 30_000 {
		t.Errorf("PPh 23 jasa = %d, want 30.000", pr.Pajak)
	}
}

func TestParameter_ValidasiMenolakAngkaNgawur(t *testing.T) {
	for nama, ubah := range map[string]func(p *Parameter){
		"lapis tidak naik":         func(p *Parameter) { p.Pasal17[1].Sampai = p.Pasal17[0].Sampai },
		"lapis terakhir terbatas":  func(p *Parameter) { p.Pasal17[len(p.Pasal17)-1].Sampai = 9_000_000_000 },
		"tarif di atas 100%":       func(p *Parameter) { p.BadanTarifBps = 12_000 },
		"PTKP hilang":              func(p *Parameter) { delete(p.PTKP, "K/1") },
		"DPP PPN nol":              func(p *Parameter) { p.PPNDPPPenyebut = 0 },
		"DPP PPN di atas 100%":     func(p *Parameter) { p.PPNDPPPembilang = 13 },
		"norma kembar":             func(p *Parameter) { p.Norma = append(p.Norma, p.Norma[0]) },
		"norma di atas 100%":       func(p *Parameter) { p.Norma[0].DaerahLainBps = 10_001 },
		"objek kembar":             func(p *Parameter) { p.Objek[0].Objek[1] = p.Objek[0].Objek[0] },
		"batas fasilitas terbalik": func(p *Parameter) { p.Batas31EPenuh = p.Batas31E + 1 },
		"TER tidak naik":           func(p *Parameter) { p.TER.Lapisan["A"][1].BatasBawah = 0 },
	} {
		p := ParameterBawaan()
		ubah(&p)
		if err := p.Validasi(); !errors.Is(err, ErrParameter) {
			t.Errorf("%s: err = %v, want ErrParameter", nama, err)
		}
	}
}

func TestParameterBawaan_SalinanTerpisah(t *testing.T) {
	a := ParameterBawaan()
	a.PTKP["TK/0"] = 1
	a.Pasal17[0].TarifBps = 1
	a.Objek[0].Objek[0].TarifBps = 1
	if b := ParameterBawaan(); b.PTKP["TK/0"] == 1 || b.Pasal17[0].TarifBps == 1 || b.Objek[0].Objek[0].TarifBps == 1 {
		t.Fatal("ParameterBawaan berbagi map/slice dengan pemanggil sebelumnya")
	}
}

func TestCariNorma(t *testing.T) {
	n, ok := CariNorma("01111")
	if !ok || n.Uraian != "PERTANIAN TANAMAN JAGUNG" || n.Bps(WilayahIbukota10) != 1500 {
		t.Fatalf("got %+v %v", n, ok)
	}
	if n, _ := CariNorma("01113"); n.Bps(WilayahIbukotaLain) != -1 {
		t.Error("sel kosong di sumber harus tetap tidak tersedia, bukan ditebak")
	}
	if _, ok := CariNorma("99999"); ok {
		t.Error("KLU tak dikenal ditemukan")
	}
}

func TestSelaraskan_ObjekIkutTarifUtama(t *testing.T) {
	pakaiParameter(t, func(p *Parameter) {
		p.UMKMTarifBps = 100
		p.PPNTarifBps = 1100
		p.PPNDPPPembilang, p.PPNDPPPenyebut = 1, 1
	})
	umkm, _ := HitungPotong(PotongInput{Jenis: "pph4-2", Objek: "42-umkm", NilaiBruto: 1_000_000, PunyaNPWP: true})
	ppn, _ := HitungPotong(PotongInput{Jenis: "ppn", Objek: "ppn-umum", NilaiBruto: 1_000_000, PunyaNPWP: true})
	mewah, _ := HitungPotong(PotongInput{Jenis: "ppn", Objek: "ppn-mewah", NilaiBruto: 1_000_000, PunyaNPWP: true})
	if umkm.Pajak != 10_000 || ppn.Pajak != 110_000 || ppn.DPP != 1_000_000 || mewah.Pajak != 110_000 {
		t.Fatalf("umkm %d, ppn %d (DPP %d), mewah %d", umkm.Pajak, ppn.Pajak, ppn.DPP, mewah.Pajak)
	}
}
