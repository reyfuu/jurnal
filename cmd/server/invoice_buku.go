package main

import (
	"errors"
	"net/http"
	"slices"
	"time"

	"pajak/internal/finance"
)

// Invoice ↔ pembukuan. Invoice yang dibukukan (Dibukukan) dan sudah terkirim/lunas dijaga
// punya dua transaksi: Penjualan sebesar DPP dan PPN Keluaran sebesar PPN (bila ada), atas nama
// pelanggan dengan jatuh tempo invoice. Draft tidak dibukukan. Dengan begitu invoice yang
// dikirim langsung menjadi omzet & piutang, dan pelunasannya menambah kas — tidak dicatat dua
// kali lewat menu Transaksi.

var errTxInvoice = errors.New("transaksi ini dibuat otomatis dari invoice; ubah atau hapus lewat menu Invoice supaya invoice dan pembukuan tetap sama")

// sinkronInvoiceLocked menyesuaikan transaksi milik inv (lama = keadaan sebelumnya, nil saat
// dibuat / dihapus = inv dengan status draft). Dipanggil dengan s.mu terkunci; inv.ID sudah ada.
func (s *tenant) sinkronInvoiceLocked(inv *finance.Invoice) error {
	if !inv.Dibukukan {
		return nil
	}
	tot := inv.Totals()
	aktif := inv.Status != finance.StatusDraft
	lunas := inv.Status == finance.StatusLunas
	jual := finance.Transaction{
		Tanggal: inv.Tanggal, Jenis: "Penjualan", Nominal: tot.DPP, Keterangan: "Invoice " + inv.Nomor,
		Pihak: inv.PelangganNama, Lunas: lunas, TanggalLunas: inv.TanggalLunas, InvoiceID: inv.ID, JatuhTempo: inv.JatuhTempo,
	}
	jual.Rincian, jual.BukanBarang = inv.BarangPenjualan()
	var awal *finance.GerakStok
	if aktif && tot.DPP > 0 {
		var err error
		if awal, err = s.cekStokInvoiceLocked(inv.TransaksiID, jual); err != nil {
			return err
		}
	}
	if err := s.jagaTxInvoiceLocked(&inv.TransaksiID, aktif && tot.DPP > 0, jual); err != nil {
		return err
	}
	if err := s.jagaTxInvoiceLocked(&inv.TransaksiPPNID, aktif && tot.PPN > 0, finance.Transaction{
		Tanggal: inv.Tanggal, Jenis: "PPN Keluaran", Nominal: tot.PPN, Keterangan: "PPN invoice " + inv.Nomor,
		Pihak: inv.PelangganNama, Lunas: lunas, TanggalLunas: inv.TanggalLunas, InvoiceID: inv.ID, JatuhTempo: inv.JatuhTempo,
	}); err != nil {
		return err
	}
	if awal != nil { // terakhir: gagal di atas → kartu belum dimulai (pemanggil memulihkan transaksi)
		if err := s.simpanStokAwalLocked(awal, "Invoice "+inv.Nomor, time.Now().UTC()); err != nil {
			return errors.New("gagal menyimpan metode nilai stok")
		}
	}
	return nil
}

// cekStokInvoiceLocked: stok tidak minus bila transaksi Penjualan invoice (id lama; 0 = baru)
// diganti jual. Kartu yang belum dimulai dimulai otomatis oleh invoice berbarang pertama:
// usulan stok awalnya dikembalikan (nil bila tidak perlu) untuk disimpan pemanggil. Dasar
// pembanding = sekarang + stok awal itu, tanpa perubahan invoice ini (lihat cekSimpanTxLocked),
// jadi invoice lama yang tidak berubah tidak terblokir minus lama. s.mu dipegang.
func (s *tenant) cekStokInvoiceLocked(id int, jual finance.Transaction) (*finance.GerakStok, error) {
	if !s.perpetualLocked() {
		return nil, nil
	}
	awal := s.stokAwalTxLocked(jual)
	txs := slices.Clone(s.txs)
	if i := slices.IndexFunc(txs, func(x finance.Transaction) bool { return id != 0 && x.ID == id }); i >= 0 {
		jual.ID = id
		txs[i] = jual
	} else {
		jual.ID = s.nextTxID
		txs = append(txs, jual)
	}
	gerak := s.tambahGerakUsulan(awal)
	return awal, s.cekTanpaMinusDariLocked(s.txs, gerak, txs, gerak)
}

// jagaTxInvoiceLocked membuat, memperbarui, atau menghapus satu transaksi milik invoice.
// Channel & akun koreksi pengguna dipertahankan; bulan yang sudah tutup buku ditolak.
func (s *tenant) jagaTxInvoiceLocked(id *int, ada bool, t finance.Transaction) error {
	i := slices.IndexFunc(s.txs, func(x finance.Transaction) bool { return *id != 0 && x.ID == *id })
	if !ada {
		if i >= 0 {
			if err := s.cekTerbukaLocked(tanggalTx(s.txs[i])...); err != nil {
				return err
			}
			s.txs = slices.Delete(s.txs, i, i+1)
		}
		*id = 0
		return nil
	}
	if t.TanggalLunas == t.Tanggal {
		t.TanggalLunas = ""
	}
	if i >= 0 {
		t.Channel, t.Akun = s.txs[i].Channel, s.txs[i].Akun
		if err := s.cekTerbukaLocked(tanggalTx(s.txs[i])...); err != nil {
			return err
		}
	}
	if err := finance.Validate(&t); err != nil {
		return err
	}
	if err := s.cekTxLocked(t); err != nil {
		return err
	}
	lama := ""
	if i >= 0 {
		lama = s.txs[i].Lawan
	}
	s.pakaiRekeningBawaanLocked(&t, lama)
	if i >= 0 {
		t.ID = s.txs[i].ID
		t.NoJurnal = s.txs[i].NoJurnal
		s.txs[i] = t
	} else {
		t.ID = s.nextTxID
		s.nextTxID++
		s.txs = append(s.txs, t)
	}
	*id = t.ID
	return nil
}

// lunasiInvoiceLocked: transaksi invoice dilunasi dari Piutang & Utang → invoice lunas pada
// tanggal itu dan transaksi pasangannya (DPP / PPN) ikut lunas.
func (s *tenant) lunasiInvoiceLocked(invoiceID int, tanggal string) error {
	i := slices.IndexFunc(s.invoices, func(x finance.Invoice) bool { return x.ID == invoiceID })
	if i < 0 {
		return nil
	}
	inv := s.invoices[i]
	inv.Status, inv.TanggalLunas = finance.StatusLunas, tanggal
	if err := s.sinkronInvoiceLocked(&inv); err != nil {
		return err
	}
	s.invoices[i] = inv
	return nil
}

// handleBukukanInvoice: semua invoice yang belum dibukukan (dibuat sebelum invoice tersambung
// ke pembukuan) dibukukan sekaligus. Draft ikut ditandai supaya langsung tercatat begitu
// dikirim. Invoice yang gagal (mis. bulannya sudah tutup buku) dilewati dan dilaporkan.
func (s *tenant) handleBukukanInvoice(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := struct {
		Dibukukan int      `json:"dibukukan"`
		Gagal     []string `json:"gagal"`
	}{Gagal: []string{}}
	for i := range s.invoices {
		if s.invoices[i].Dibukukan {
			continue
		}
		inv := s.invoices[i]
		inv.Dibukukan = true
		txs, nextID := slices.Clone(s.txs), s.nextTxID
		if err := s.sinkronInvoiceLocked(&inv); err != nil {
			s.txs, s.nextTxID = txs, nextID // invoice ini tidak diubah sama sekali
			out.Gagal = append(out.Gagal, inv.Nomor+": "+err.Error())
			continue
		}
		s.invoices[i] = inv
		if inv.Status != finance.StatusDraft {
			out.Dibukukan++
		}
	}
	writeJSON(w, out)
}
