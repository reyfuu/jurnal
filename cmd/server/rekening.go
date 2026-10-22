package main

import (
	"pajak/internal/coa"
	"pajak/internal/finance"
	"pajak/internal/katalog"
)

// Rekening bawaan: uang masuk / keluar yang tidak memilih rekening sendiri dicatat di Bank
// (bukan Kas): transaksi baru, pelunasan piutang & utang, invoice, aset, mutasi bank, produksi,
// dan pelunasan saldo awal. Kas tetap bisa dipilih. Transaksi lama tidak diubah.

// coaTemplateLocked: akun template CoA usaha; dipanggil dengan s.mu terkunci (lihat coaTemplate).
func (s *tenant) coaTemplateLocked() []coa.Account {
	st := katalogAktif.Load()
	if st == nil || s.profile == nil {
		return nil
	}
	p := *s.profile
	if t, ok := st.TemplateByID(s.coaTemplateID); ok && s.coaTemplateID > 0 && t.Jenis == p.JenisUsaha && t.Status != katalog.Draft {
		return t.Akun
	}
	t, _ := st.TemplateAktif(p.JenisUsaha)
	return t.Akun
}

// rekeningBawaanLocked: kode akun Bank usaha ("" bila CoA belum ada / tanpa akun bank).
func (s *tenant) rekeningBawaanLocked() string {
	all := s.coaTemplateLocked()
	if all == nil {
		return ""
	}
	return coa.PetaDari(all).Kode(coa.PeranBank)
}

// pakaiRekeningBawaanLocked mengisi Lawan transaksi dengan rekening bawaan bila kosong. lama =
// Lawan transaksi sebelum diubah (dipertahankan). Channel marketplace / B2B tetap ke akunnya.
func (s *tenant) pakaiRekeningBawaanLocked(t *finance.Transaction, lama string) {
	switch {
	case coa.ChannelKhusus(t.Channel):
		t.Lawan = ""
	case t.Lawan != "":
	case lama != "":
		t.Lawan = lama
	default:
		t.Lawan = s.rekeningBawaanLocked()
	}
}
