package finance

import (
	"fmt"
	"strings"
)

// Data master (barang di Stok & Gudang, produk, supplier) tidak boleh kembar: SKU sama atau
// nama sama — salah satunya saja sudah ditolak — supaya kartu stok, resep, dan transaksi tidak
// menunjuk ke barang yang salah. Perbandingan tidak membedakan huruf besar/kecil dan spasi ganda.

// KunciNama: "  Kopi   Susu " → "kopi susu".
func KunciNama(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), " ")) }

// CekUnik: galat bila sku (bila diisi) atau nama sudah dipakai entri lain di list. id = entri
// yang sedang diubah (dilewati); 0 untuk entri baru. jenis dipakai di pesan ("barang").
func CekUnik[T any](list []T, info func(T) (id int, sku, nama string), id int, sku, nama, jenis string) error {
	ks, kn := KunciNama(sku), KunciNama(nama)
	for _, x := range list {
		xi, xs, xn := info(x)
		if id != 0 && xi == id {
			continue
		}
		if ks != "" && KunciNama(xs) == ks {
			return fmt.Errorf("SKU %s sudah dipakai %s %q; SKU tidak boleh sama", strings.TrimSpace(sku), jenis, xn)
		}
		if kn != "" && KunciNama(xn) == kn {
			return fmt.Errorf("%s bernama %q sudah ada; nama tidak boleh sama", jenis, strings.TrimSpace(xn))
		}
	}
	return nil
}

func InfoStok(s StokItem) (int, string, string)     { return s.ID, s.SKU, s.Nama }
func InfoProduk(p Produk) (int, string, string)     { return p.ID, p.SKU, p.Nama }
func InfoSupplier(s Supplier) (int, string, string) { return s.ID, "", s.Nama }
