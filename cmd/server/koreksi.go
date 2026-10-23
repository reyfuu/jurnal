package main

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"pajak/internal/finance"
	"pajak/internal/master"
)

// Usulan koreksi: tim Master Pajak mengusulkan ubah/hapus transaksi klien; pemilik usaha yang
// memutuskan. Pembukuan baru berubah setelah disetujui (finance.TerapkanKoreksi).

type koreksiRow struct {
	finance.Koreksi
	Nomor     string                  `json:"nomor"`
	KlienID   int                     `json:"klien_id"`
	KlienNama string                  `json:"klien_nama"`
	Dampak    []finance.DampakKoreksi `json:"dampak"`
}

func nomorKoreksi(id int) string { return fmt.Sprintf("KR-%03d", id) }

// koreksiRows: usulan satu usaha, terbaru dulu; pemanggil TIDAK memegang t.mu.
func koreksiRows(t *tenant) []koreksiRow {
	t.mu.RLock()
	defer t.mu.RUnlock()
	nama := ""
	if t.profile != nil {
		nama = t.profile.NamaUsaha
	}
	nama = namaKlien(t, nama)
	out := make([]koreksiRow, 0, len(t.koreksi))
	for i := len(t.koreksi) - 1; i >= 0; i-- {
		k := t.koreksi[i]
		out = append(out, koreksiRow{k, nomorKoreksi(k.ID), t.id, nama, finance.HitungDampak(t.txs, k)})
	}
	return out
}

func (s *Server) handleUsulKoreksi(w http.ResponseWriter, r *http.Request) {
	t := s.klienDari(w, r)
	if t == nil {
		return
	}
	var k finance.Koreksi
	if !decodeBody(w, r, &k) {
		return
	}
	if k.Usulan != nil && !t.validAkun(*k.Usulan) {
		http.Error(w, "akun "+k.Usulan.Akun+" tidak ada di daftar akun usaha ini", http.StatusBadRequest)
		return
	}
	u := currentUser(r)
	t.mu.Lock()
	if err := finance.SiapkanKoreksi(&k, t.txs); err != nil {
		t.mu.Unlock()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	k.ID, k.OlehID, k.Oleh, k.Waktu = t.nextKoreksiID, u.ID, u.Name, time.Now().UTC().Truncate(time.Second)
	k.Catatan, k.DiputuskanOleh, k.DiputuskanPada = "", "", time.Time{}
	t.nextKoreksiID++
	t.koreksi = append(t.koreksi, k)
	t.mu.Unlock()
	if err := s.saveFor(t); err != nil {
		t.mu.Lock()
		t.koreksi, t.nextKoreksiID = t.koreksi[:len(t.koreksi)-1], t.nextKoreksiID-1
		t.mu.Unlock()
		writeMasterErr(w, err)
		return
	}
	if err := s.catat(r, "Usulkan koreksi "+nomorKoreksi(k.ID), t.id, k.Alasan); err != nil {
		writeMasterErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, koreksiRows(t)[0])
}

// handleMasterKoreksi: semua usulan di klien yang boleh dilihat anggota, terbaru dulu.
func (s *Server) handleMasterKoreksi(w http.ResponseWriter, r *http.Request) {
	out := []koreksiRow{}
	for _, k := range s.klienRows(aksesOf(r), time.Now(), false) { // koreksi hanya butuh ID usaha
		s.tenantsMu.RLock()
		t := s.tenants[k.ID]
		s.tenantsMu.RUnlock()
		out = append(out, koreksiRows(t)...)
	}
	slices.SortStableFunc(out, func(a, b koreksiRow) int { return b.Waktu.Compare(a.Waktu) })
	writeJSON(w, out)
}

// handleListKoreksi: usulan untuk usaha milik user (menu Usulan Koreksi, pemilik usaha).
func (t *tenant) handleListKoreksi(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, koreksiRows(t))
}

func (s *Server) handleSetujuiKoreksi(w http.ResponseWriter, r *http.Request) {
	s.putuskanKoreksi(w, r, true)
}

func (s *Server) handleTolakKoreksi(w http.ResponseWriter, r *http.Request) {
	s.putuskanKoreksi(w, r, false)
}

func (s *Server) putuskanKoreksi(w http.ResponseWriter, r *http.Request, setuju bool) {
	t := s.tenantOf(r)
	id, _ := strconv.Atoi(r.PathValue("id"))
	var req struct{ Catatan string }
	if !decodeBody(w, r, &req) {
		return
	}
	u := currentUser(r)
	now := time.Now().UTC().Truncate(time.Second)
	t.mu.Lock()
	i := slices.IndexFunc(t.koreksi, func(k finance.Koreksi) bool { return k.ID == id })
	if i < 0 {
		t.mu.Unlock()
		http.Error(w, "usulan tidak ditemukan", http.StatusNotFound)
		return
	}
	k := &t.koreksi[i]
	if k.Status != finance.KoreksiMenunggu {
		t.mu.Unlock()
		http.Error(w, "usulan ini sudah diputuskan", http.StatusConflict)
		return
	}
	if setuju {
		tgl := tanggalTx(k.Sebelum)
		if k.Usulan != nil {
			tgl = append(tgl, tanggalTx(*k.Usulan)...)
		}
		if err := t.cekTerbukaLocked(tgl...); err != nil {
			t.mu.Unlock()
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
	}
	k.Catatan, k.DiputuskanOleh, k.DiputuskanPada = clip(strings.TrimSpace(req.Catatan), 500), u.Name, now
	basi := false
	if setuju {
		txs, err := finance.TerapkanKoreksi(t.txs, *k)
		switch {
		case errors.Is(err, finance.ErrKoreksiBasi):
			k.Status, basi = finance.KoreksiKedaluwarsa, true
		case k.Jenis == finance.KoreksiHapus:
			// hapus = batalkan: nomor & riwayatnya tetap tercatat
			k.Status = finance.KoreksiDisetujui
			id := k.TransaksiID
			t.batalkanLocked(func(x finance.Transaction) bool { return x.ID == id }, "transaksi", u.Name, "Usulan koreksi konsultan disetujui: "+k.Alasan)
		default:
			k.Status, t.txs = finance.KoreksiDisetujui, txs
		}
	} else {
		k.Status = finance.KoreksiDitolak
	}
	kk := *k
	t.mu.Unlock()
	if basi {
		// persisting tidak menyimpan respons gagal, padahal status usulan berubah: simpan sendiri.
		if err := s.saveFor(t); err != nil {
			writeMasterErr(w, err)
			return
		}
		http.Error(w, "Transaksi ini sudah berubah sejak usulan dibuat, jadi usulan tidak diterapkan. Minta konsultan mengirim usulan baru bila masih perlu.", http.StatusConflict)
		return
	}
	aksi := "Tolak koreksi "
	if setuju {
		aksi = "Setujui koreksi "
	}
	err := s.ubahMaster(func(st *master.State) error {
		st.Catat(master.Catatan{UserID: u.ID, Nama: u.Name, Peran: "Owner klien", Aksi: aksi + nomorKoreksi(kk.ID), KlienID: t.id, Keterangan: kk.Catatan})
		return nil
	})
	if err != nil {
		writeMasterErr(w, err)
		return
	}
	writeJSON(w, kk)
}

// koreksiMenunggu: jumlah usulan yang belum diputuskan (badge untuk pemilik usaha).
func (t *tenant) koreksiMenunggu() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	n := 0
	for _, k := range t.koreksi {
		if k.Status == finance.KoreksiMenunggu {
			n++
		}
	}
	return n
}
