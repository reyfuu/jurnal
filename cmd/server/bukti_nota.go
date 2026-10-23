package main

import (
	"cmp"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"time"

	"pajak/internal/ai"
	"pajak/internal/finance"
)

// Bukti foto nota (handoff §15: hasil OCR hanya draf, sumber & jejak persetujuan disimpan).
// Foto yang baru dibaca AI ditahan di memori per usaha (tidak ke disk). Begitu catatannya
// disimpan, foto + jejaknya (saran AI, siapa yang menyimpan, kapan) ditulis sebagai bukti grup:
// $DATA_DIR/bukti/<usaha>/<grup>.<ext> dan <grup>.json. Tanpa DATA_DIR (test, dev lokal) bukti
// disimpan di memori. Kuncinya grup, karena ubah catatan mempertahankan nomor grup tetapi
// mengganti ID transaksinya. Bukti ikut terhapus bersama catatannya.

const (
	tundaNotaTTL    = 2 * time.Hour
	maxTundaNota    = 20 // foto menunggu disimpan, per usaha
	maxDuplikatNota = 3
)

type notaTunda struct {
	foto  []byte
	mime  string
	saran ai.SaranNota
	waktu time.Time
}

// buktiMeta: jejak bukti satu catatan — apa yang dibaca AI dan siapa yang mengonfirmasi simpan.
type buktiMeta struct {
	Grup          int          `json:"grup"`
	Hash          string       `json:"hash"` // sha256 isi foto
	Mime          string       `json:"mime"`
	Ukuran        int          `json:"ukuran"`
	Saran         ai.SaranNota `json:"saran"`
	DibacaPada    time.Time    `json:"dibaca_pada"`
	DisimpanOleh  string       `json:"disimpan_oleh"`
	DisimpanEmail string       `json:"disimpan_email"`
	DisimpanPada  time.Time    `json:"disimpan_pada"`
}

// fotoNotaInfo: ringkasan bukti untuk halaman Catat transaksi.
type fotoNotaInfo struct {
	DibacaPada   time.Time `json:"dibaca_pada"`
	DisimpanOleh string    `json:"disimpan_oleh"`
	DisimpanPada time.Time `json:"disimpan_pada"`
	Yakin        string    `json:"yakin,omitempty"`
}

func (m buktiMeta) info() *fotoNotaInfo {
	return &fotoNotaInfo{m.DibacaPada, m.DisimpanOleh, m.DisimpanPada, m.Saran.Yakin}
}

// duplikatNota: catatan yang fotonya sama dengan nota yang baru dibaca.
type duplikatNota struct {
	Grup       int    `json:"grup"`
	Tanggal    string `json:"tanggal"`
	Nominal    int64  `json:"nominal"`
	Pihak      string `json:"pihak,omitempty"`
	Keterangan string `json:"keterangan,omitempty"`
	Alasan     string `json:"alasan"`
}

// hasilNota: balasan baca foto. Bukti = token untuk menyimpan fotonya bersama catatan.
type hasilNota struct {
	ai.SaranNota
	Bukti    string         `json:"bukti"`
	Duplikat []duplikatNota `json:"duplikat,omitempty"`
}

type buktiMem struct {
	meta buktiMeta
	foto []byte
}

func buktiDir(tenantID int) string {
	d := dataPath("bukti")
	if d == "" {
		return ""
	}
	return filepath.Join(d, strconv.Itoa(tenantID))
}

// tundaNotaLocked menahan foto yang baru dibaca sampai catatannya disimpan. s.mu dipegang.
func (s *tenant) tundaNotaLocked(hash string, n notaTunda) {
	if s.notaTunda == nil {
		s.notaTunda = map[string]notaTunda{}
	}
	tertua := ""
	for h, x := range s.notaTunda {
		if time.Since(x.waktu) > tundaNotaTTL {
			delete(s.notaTunda, h)
		} else if tertua == "" || x.waktu.Before(s.notaTunda[tertua].waktu) {
			tertua = h
		}
	}
	if len(s.notaTunda) >= maxTundaNota {
		delete(s.notaTunda, tertua)
	}
	s.notaTunda[hash] = n
}

// semuaBuktiLocked: jejak semua bukti usaha ini. s.mu dipegang.
func (s *tenant) semuaBuktiLocked() []buktiMeta {
	dir := buktiDir(s.id)
	var out []buktiMeta
	if dir == "" {
		for _, b := range s.buktiMem {
			out = append(out, b.meta)
		}
		return out
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	for _, f := range files {
		var m buktiMeta
		if raw, err := os.ReadFile(f); err == nil && json.Unmarshal(raw, &m) == nil {
			out = append(out, m)
		}
	}
	return out
}

func (s *tenant) buktiLocked(grup int) (buktiMeta, bool) {
	dir := buktiDir(s.id)
	if dir == "" {
		b, ok := s.buktiMem[grup]
		return b.meta, ok
	}
	var m buktiMeta
	raw, err := os.ReadFile(filepath.Join(dir, strconv.Itoa(grup)+".json"))
	return m, err == nil && json.Unmarshal(raw, &m) == nil
}

func (s *tenant) fotoBuktiLocked(m buktiMeta) ([]byte, error) {
	dir := buktiDir(s.id)
	if dir == "" {
		return s.buktiMem[m.Grup].foto, nil
	}
	return os.ReadFile(filepath.Join(dir, strconv.Itoa(m.Grup)+screenshotExt[m.Mime]))
}

func (s *tenant) simpanBuktiLocked(m buktiMeta, foto []byte) error {
	dir := buktiDir(s.id)
	if dir == "" {
		if s.buktiMem == nil {
			s.buktiMem = map[int]buktiMem{}
		}
		s.buktiMem[m.Grup] = buktiMem{m, foto}
		return nil
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return err
	}
	base := filepath.Join(dir, strconv.Itoa(m.Grup))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(base+screenshotExt[m.Mime], foto, 0o600); err != nil {
		return err
	}
	return os.WriteFile(base+".json", raw, 0o600)
}

// hapusBuktiLocked membuang bukti grup: catatannya dihapus, atau nomor grupnya dipakai ulang.
func (s *tenant) hapusBuktiLocked(grup int) {
	dir := buktiDir(s.id)
	if dir == "" {
		delete(s.buktiMem, grup)
		return
	}
	lama, _ := filepath.Glob(filepath.Join(dir, strconv.Itoa(grup)+".*"))
	for _, f := range lama {
		if err := os.Remove(f); err != nil {
			log.Printf("hapus bukti %s: %v", f, err)
		}
	}
}

// lampirkanBuktiLocked menyimpan foto tunda sebagai bukti grup yang baru dicatat. Token asing atau
// kedaluwarsa tidak menggagalkan catatan; catatannya hanya tidak punya foto. s.mu dipegang.
func (s *tenant) lampirkanBuktiLocked(r *http.Request, grup int, token string) *fotoNotaInfo {
	s.hapusBuktiLocked(grup) // jangan warisi bukti grup lama yang nomornya dipakai ulang
	n, ok := s.notaTunda[token]
	if token == "" || !ok || time.Since(n.waktu) > tundaNotaTTL {
		return nil
	}
	u := currentUser(r)
	m := buktiMeta{Grup: grup, Hash: token, Mime: n.mime, Ukuran: len(n.foto), Saran: n.saran, DibacaPada: n.waktu,
		DisimpanOleh: cmp.Or(u.Name, u.Email), DisimpanEmail: u.Email, DisimpanPada: time.Now().UTC()}
	if err := s.simpanBuktiLocked(m, n.foto); err != nil {
		log.Printf("simpan bukti nota usaha %d grup %d: %v", s.id, grup, err)
		return nil
	}
	delete(s.notaTunda, token)
	return m.info()
}

// duplikatNotaLocked: catatan yang fotonya sama persis dengan nota ini. Tanggal & nominal sama
// tanpa foto dicegat cek dobel saat simpan (finance.Mirip), bukan di sini. s.mu dipegang.
func (s *tenant) duplikatNotaLocked(hash string) []duplikatNota {
	var out []duplikatNota
	for _, m := range s.semuaBuktiLocked() {
		// catatan yang sudah dibatalkan tidak dihitung dobel (fotonya hanya disimpan sebagai riwayat)
		aktif := slices.ContainsFunc(s.txs, func(t finance.Transaction) bool { return t.Grup == m.Grup })
		if m.Hash == hash && aktif && len(out) < maxDuplikatNota {
			out = append(out, duplikatNota{Grup: m.Grup, Tanggal: m.Saran.Tanggal, Nominal: m.Saran.Nominal, Pihak: m.Saran.Pihak,
				Keterangan: m.Saran.Keterangan, Alasan: "foto yang sama sudah pernah dicatat"})
		}
	}
	return out
}

// handleBuktiCatat menyajikan foto nota yang menjadi bukti satu catatan (hanya usaha pemiliknya).
func (s *tenant) handleBuktiCatat(w http.ResponseWriter, r *http.Request) {
	grup, _ := strconv.Atoi(r.PathValue("grup"))
	s.mu.RLock()
	m, ok := s.buktiLocked(grup)
	var foto []byte
	if ok {
		foto, _ = s.fotoBuktiLocked(m)
	}
	s.mu.RUnlock()
	if len(foto) == 0 {
		http.Error(w, "foto nota tidak ditemukan", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", m.Mime)
	_, _ = w.Write(foto)
}
