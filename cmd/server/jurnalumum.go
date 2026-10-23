package main

import (
	"cmp"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"pajak/internal/xlsx"
)

// Jurnal Umum: seluruh entri jurnal urut tanggal, dikelompokkan per bukti (satu transaksi,
// pelunasan, atau entri sistem seperti penyusutan), dengan baris debit/kredit per akun. Dibentuk
// dari jurnal yang sama dengan Buku Besar & laporan per akun (jurnalPeriode), jadi totalnya sama.

type barisJurnalUmum struct {
	Kode   string `json:"kode"`
	Nama   string `json:"nama"`
	Debit  int64  `json:"debit"`
	Kredit int64  `json:"kredit"`
}

type entriJurnalUmum struct {
	Bukti      string            `json:"bukti"` // SA saldo awal, JU-yyyymm-n transaksi, BYR-id pelunasan, SIS-yyyymm-n entri sistem
	Tanggal    string            `json:"tanggal"`
	TxID       int               `json:"tx_id,omitempty"`
	Keterangan string            `json:"keterangan"`
	Baris      []barisJurnalUmum `json:"baris"`
	Debit      int64             `json:"debit"`
	Kredit     int64             `json:"kredit"`
}

type jurnalUmumResp struct {
	Supported   bool              `json:"supported"`
	PeriodLabel string            `json:"period_label"`
	Dari        string            `json:"dari"`
	Sampai      string            `json:"sampai"`
	Entri       []entriJurnalUmum `json:"entri"`
	TotalDebit  int64             `json:"total_debit"`
	TotalKredit int64             `json:"total_kredit"`
	Seimbang    bool              `json:"seimbang"`
	Pilihan     []pilihanAkun     `json:"pilihan"`
}

// jurnalUmum: akun ≠ "" = hanya bukti yang menyentuh akun itu; q = kata di keterangan, nomor
// bukti, atau nama akun. Disaring di server supaya tampilan sama dengan ekspor Excel.
func (s *tenant) jurnalUmum(dari, sampai, akun, q string) jurnalUmumResp {
	out := jurnalUmumResp{PeriodLabel: labelRentang(dari, sampai), Dari: dari, Sampai: sampai, Entri: []entriJurnalUmum{}, Pilihan: []pilihanAkun{}}
	s.mu.RLock()
	mulai := s.bukuAwalLocked().Tanggal
	s.mu.RUnlock()
	nb := s.penomoranBukti()
	q = strings.ToLower(strings.TrimSpace(q))
	for bulan := dari[:7]; bulan <= sampai[:7]; bulan = bulanBerikut(bulan) {
		j, ok := s.jurnalPeriode(bulan)
		if !ok {
			return out
		}
		out.Supported = true
		nama := make(map[string]string, len(j.all))
		for _, a := range j.all {
			nama[a.Kode] = a.Nama
			if bulan == dari[:7] {
				out.Pilihan = append(out.Pilihan, pilihanAkun{a.Kode, a.Nama})
			}
		}
		var bulanIni []entriJurnalUmum
		// Saldo awal pembukuan: satu bukti berisi saldo awal semua akun, urut CoA.
		if mulai[:7] == bulan && mulai >= dari && mulai <= sampai {
			e := entriJurnalUmum{Bukti: "SA", Tanggal: mulai, Keterangan: "Saldo awal pembukuan"}
			for _, a := range j.all {
				if v := j.saldoAwal[a.Kode]; v > 0 {
					e.Baris = append(e.Baris, barisJurnalUmum{Kode: a.Kode, Nama: a.Nama, Debit: v})
				} else if v < 0 {
					e.Baris = append(e.Baris, barisJurnalUmum{Kode: a.Kode, Nama: a.Nama, Kredit: -v})
				}
			}
			bulanIni = append(bulanIni, e)
		}
		grup, bukti := nb.kelompokkan(j.entri, bulan)
		awal := len(bulanIni) // setelah bukti saldo awal
		for x, l := range j.entri {
			i := awal + grup[x]
			if i == len(bulanIni) {
				bulanIni = append(bulanIni, entriJurnalUmum{Bukti: bukti[grup[x]], Tanggal: l.Tanggal, TxID: l.TxID, Keterangan: l.Ket})
			}
			// akun & sisi yang sama dalam satu bukti (mis. Kas dari beberapa transaksi satu catatan) digabung
			bs := bulanIni[i].Baris
			if x := slices.IndexFunc(bs, func(b barisJurnalUmum) bool { return b.Kode == l.Kode && (b.Debit > 0) == (l.Debit > 0) }); x >= 0 {
				bs[x].Debit += l.Debit
				bs[x].Kredit += l.Kredit
			} else {
				bulanIni[i].Baris = append(bs, barisJurnalUmum{Kode: l.Kode, Nama: nama[l.Kode], Debit: l.Debit, Kredit: l.Kredit})
			}
		}
		slices.SortStableFunc(bulanIni, func(a, b entriJurnalUmum) int { return cmp.Compare(a.Tanggal, b.Tanggal) })
		for _, e := range bulanIni {
			if e.Tanggal < dari || e.Tanggal > sampai || !cocokJurnal(e, akun, q) {
				continue
			}
			for _, b := range e.Baris {
				e.Debit += b.Debit
				e.Kredit += b.Kredit
			}
			out.TotalDebit += e.Debit
			out.TotalKredit += e.Kredit
			out.Entri = append(out.Entri, e)
		}
	}
	out.Seimbang = out.TotalDebit == out.TotalKredit
	return out
}

// penomoranBukti: nomor jurnal tetap & grup catatan per transaksi, dasar No. Bukti yang sama di
// Jurnal Umum dan Buku Besar.
type penomoranBukti struct {
	noJurnal map[int]string // nomor jurnal tetap per transaksi
	grupTx   map[int]int    // transaksi satu catatan Catat → satu bukti
}

func (s *tenant) penomoranBukti() penomoranBukti {
	s.mu.RLock()
	defer s.mu.RUnlock()
	nb := penomoranBukti{noJurnal: make(map[int]string, len(s.txs)), grupTx: make(map[int]int, len(s.txs))}
	for _, t := range s.txs {
		nb.noJurnal[t.ID], nb.grupTx[t.ID] = t.NoJurnal, t.Grup
	}
	return nb
}

// kelompokkan: baris jurnal satu transaksi / entri sistem bersebelahan di entri; dikelompokkan per
// (transaksi, tanggal, keterangan), jadi pelunasan bulan berikutnya jadi bukti sendiri. grup[i] =
// nomor urut bukti baris entri[i] (urut kemunculan), bukti[g] = No. Bukti-nya.
func (nb penomoranBukti) kelompokkan(entri []entriJurnal, bulan string) (grup []int, bukti []string) {
	type kunci struct {
		tx      int
		tanggal string
		ket     string
	}
	indeks := map[kunci]int{}
	grup = make([]int, len(entri))
	nSistem := 0
	for x, l := range entri {
		k := kunci{l.TxID, l.Tanggal, l.Ket}
		if g := nb.grupTx[l.TxID]; g > 0 && !strings.HasPrefix(l.Ket, "Pelunasan") {
			k.tx, k.ket = -g, "" // baris semua transaksi satu catatan digabung jadi satu jurnal
		}
		i, ada := indeks[k]
		if !ada {
			var b string
			switch {
			case l.TxID == 0:
				nSistem++
				b = fmt.Sprintf("SIS-%s-%d", strings.ReplaceAll(bulan, "-", ""), nSistem) // entri sistem; JU- untuk transaksi
			case strings.HasPrefix(l.Ket, "Pelunasan"):
				b = "BYR-" + strconv.Itoa(l.TxID)
			default:
				b = cmp.Or(nb.noJurnal[l.TxID], "TRX-"+strconv.Itoa(l.TxID))
			}
			i = len(bukti)
			indeks[k] = i
			bukti = append(bukti, b)
		}
		grup[x] = i
	}
	return grup, bukti
}

func cocokJurnal(e entriJurnalUmum, akun, q string) bool {
	if akun != "" && !slices.ContainsFunc(e.Baris, func(b barisJurnalUmum) bool { return b.Kode == akun }) {
		return false
	}
	// TRX-<id>: referensi lama sebelum ada nomor jurnal, tetap bisa dicari
	if q == "" || strings.Contains(strings.ToLower(e.Keterangan+" "+e.Bukti+" trx-"+strconv.Itoa(e.TxID)), q) {
		return true
	}
	return slices.ContainsFunc(e.Baris, func(b barisJurnalUmum) bool { return strings.Contains(strings.ToLower(b.Kode+" "+b.Nama), q) })
}

func (s *tenant) handleJurnalUmum(w http.ResponseWriter, r *http.Request) {
	dari, sampai, err := rentangDari(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, s.jurnalUmum(dari, sampai, r.URL.Query().Get("akun"), r.URL.Query().Get("q")))
}

func (s *tenant) handleJurnalUmumXlsx(w http.ResponseWriter, r *http.Request) {
	dari, sampai, err := rentangDari(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ju := s.jurnalUmum(dari, sampai, r.URL.Query().Get("akun"), r.URL.Query().Get("q"))
	if !ju.Supported {
		http.Error(w, "CoA untuk jenis usaha ini belum tersedia", http.StatusNotFound)
		return
	}
	// Tanggal & bukti diulang di tiap baris supaya filter Excel tetap utuh per bukti.
	rows := [][]string{{"Tanggal", "No. Bukti", "Keterangan", "Kode", "Akun", "Debit", "Kredit"}}
	for _, e := range ju.Entri {
		for _, b := range e.Baris {
			rows = append(rows, []string{e.Tanggal, e.Bukti, e.Keterangan, b.Kode, b.Nama, rp(b.Debit), rp(b.Kredit)})
		}
	}
	rows = append(rows, []string{"", "", "TOTAL", "", "", rp(ju.TotalDebit), rp(ju.TotalKredit)})
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="jurnal-umum-%s_%s.xlsx"`, dari, sampai))
	_ = xlsx.Write(w, []xlsx.Sheet{{Name: "Jurnal Umum", Rows: rows}})
}
