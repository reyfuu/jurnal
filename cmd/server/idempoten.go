package main

import (
	"errors"
	"net/http"
	"strings"
)

// Idempotency key: klien membuat satu kunci per niat (satu klik Bayar, satu buka dialog
// pembayaran) dan memakainya ulang saat mengulang permintaan yang sama, supaya klik ganda,
// tab ganda, atau retry jaringan tidak membuat dampak dua kali. Kunci disimpan pada dokumen
// yang dibuatnya (order langganan, pembayaran) dan dicek di bawah kunci tenant.

var errKunciTidakSah = errors.New("Idempotency-Key tidak sah: maksimal 64 karakter huruf, angka, - atau _")

// kunciIdempoten membaca header Idempotency-Key; "" = klien tidak mengirim (tanpa jaminan).
func kunciIdempoten(r *http.Request) (string, error) {
	k := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if len(k) > 64 || strings.ContainsFunc(k, func(c rune) bool {
		return !(c == '-' || c == '_' || '0' <= c && c <= '9' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z')
	}) {
		return "", errKunciTidakSah
	}
	return k, nil
}
