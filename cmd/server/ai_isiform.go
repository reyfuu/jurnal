package main

import (
	"context"
	"log"
	"strconv"
	"time"

	"pajak/internal/ai"
	"pajak/internal/auth"
	"pajak/internal/finance"
)

// maksIsiFormAI: batas permintaan isi form oleh AI per pengguna per 24 jam.
const maksIsiFormAI = 60

// jawabIsiForm: perintah "catat / buat / tambah …" dijawab usulan isian form. Server tidak
// menyimpan apa pun; bila AI tidak bisa membantu, pengguna diarahkan mengisi form sendiri.
func (s *Server) jawabIsiForm(ctx context.Context, u auth.User, form, query string) ai.QueryResponse {
	view, judul := ai.ViewForm[form], ai.JudulForm[form]
	if !u.Role.Can(view) {
		return ai.QueryResponse{Category: "isi-form", ActionSuggestions: []string{},
			Answer: "Form " + judul + " tidak tersedia untuk akun Anda. Minta Owner usaha membukakan aksesnya di Pengguna & Akses."}
	}
	manual := ai.QueryResponse{Category: "isi-form", ActionSuggestions: []string{}, Tautan: []ai.Tautan{{View: view, Label: "Buka " + judul}}}
	switch {
	case s.llm == nil || !s.llm.Enabled():
		manual.Answer = "Isi form otomatis butuh AI aktif. Sementara itu, isi form " + judul + " sendiri lewat tombol di bawah."
		return manual
	case !s.jatahAI("isiform|"+strconv.Itoa(u.ID), maksIsiFormAI):
		manual.Answer = "Batas isi form otomatis hari ini sudah tercapai. Isi form " + judul + " sendiri lewat tombol di bawah."
		return manual
	}
	h, err := ai.IsiForm(ctx, s.llm, form, query, time.Now().In(finance.WIB).Format(time.DateOnly))
	if err != nil {
		log.Printf("isi form AI (%s): %v", form, err)
		manual.Answer = "Saya belum bisa menyiapkan isiannya. Coba tulis lebih lengkap, mis. *catat penjualan kopi Rp2.000.000 tunai hari ini*, atau isi form " + judul + " sendiri."
		return manual
	}
	return ai.QueryResponse{Category: "isi-form", ActionSuggestions: []string{}, Aksi: h,
		Answer: "Saya siapkan isian form " + judul + ". Periksa pratinjaunya dulu."}
}
