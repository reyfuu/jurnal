# Tag Primbon tanpa memindahkan naskah VPS

PDF/HTML tetap di `REGULASI_DIR` pada VPS. Kode dan metadata kecil dapat diperbarui terpisah; tidak perlu menyalin ulang 6,6 GB naskah untuk mengembangkan tag.

Tambahkan `metadata.json` di samping `index.csv` pada folder katalog VPS. Kunci adalah UUID dari index.csv, bukan nomor urut atau judul. Contoh berikut hanya format pengisian; ganti UUID dan isinya berdasarkan peraturan yang diperiksa tim:

```json
{
  "UUID-DARI-INDEX-CSV": {
    "tag": ["PPh 21", "Penggajian"],
    "status": "",
    "perubahan": "",
    "mulai_berlaku": "",
    "status_pada": "",
    "diverifikasi_pada": "",
    "sumber_verifikasi": ""
  }
}
```

- Semua field opsional. `status` dan `mulai_berlaku` yang terisi menimpa metadata CSV; nilai kosong mempertahankan CSV. Jangan mengisi status/tanggal tanpa bukti.
- Tanggal menggunakan `YYYY-MM-DD`. `status_pada` adalah tanggal acuan metadata, bukan pilihan untuk menghitung status hukum pada sembarang tanggal. `diverifikasi_pada` mencatat kapan tim memeriksa metadata; sumber verifikasi harus URL HTTPS.
- Maksimal 20 tag per peraturan, masing-masing 80 karakter. Tag kosong dan duplikat tanpa membedakan kapital dihapus. Pencarian katalog juga mencari tag.
- UUID yang belum ada di katalog tidak ditampilkan. Metadata tidak mencocokkan otomatis katalog referensi 20.798 entri dengan katalog 1.512 entri.
- `metadata.json` tidak memuat isi naskah. Penghapusan file metadata mengembalikan katalog ke metadata CSV; tidak menghapus PDF/HTML atau bookmark pengguna.
- Server memuat ulang saat CSV atau metadata berubah tanpa restart. JSON/tanggal/URL yang tidak valid mempertahankan snapshot katalog terakhir; pada boot tanpa snapshot, metadata invalid menghasilkan katalog tidak tersedia. Perbaiki file, lalu request berikutnya mencoba lagi.

Unggah file metadata ke nama sementara pada folder yang sama, lalu rename menjadi `metadata.json` agar pembaca tidak melihat berkas separuh ditulis. Kirim hanya file metadata yang berubah dan aplikasi yang sudah dibangun; jangan memakai sinkronisasi dengan penghapusan terhadap folder naskah.

Setelah metadata berubah, jalankan kembali worker impor RAG untuk menyegarkan fingerprint, tag pencarian, dan metadata kutipan. File naskah tetap berada di VPS; proses ini tidak mengunduh atau memindahkannya. API RAG baru menampilkan metadata terbaru setelah impor tersebut, sedangkan katalog memuat metadata baru langsung.

Dalam pengembangan lokal cukup gunakan `internal/primbon/testdata/katalog` dan contoh metadata kecil, atau salinan `index.csv` saja. Jika naskah tidak ada lokal, tag/detail tetap dapat ditampilkan dan ketersediaan naskah dijelaskan dengan jujur. Jangan menandai seluruh koleksi sebagai terverifikasi hanya karena metadata berhasil diimpor.
