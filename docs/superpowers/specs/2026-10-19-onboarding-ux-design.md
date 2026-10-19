# Desain UX Onboarding Profil Usaha

## Tujuan

Membuat onboarding profil usaha mudah dipahami, nyaman di layar kecil, dan dapat digunakan dengan keyboard serta pembaca layar. Isi pertanyaan, pemetaan jawaban, validasi backend, dan ringkasan pajak tetap mengikuti alur yang sudah ada.

## Desain

- Wizard tetap menampilkan satu pertanyaan pada satu layar.
- Pilihan jawaban hanya memilih opsi. Pengguna menekan **Lanjut** untuk berpindah, sehingga perubahan layar tidak terjadi tiba-tiba dan pilihan bisa diperiksa dahulu.
- Setiap layar menampilkan nomor langkah dan progres dari total pertanyaan yang berlaku untuk jawaban saat ini. Pertanyaan yang dilewati oleh kondisi usaha tidak dihitung.
- **Kembali** tersedia pada setiap langkah setelah langkah pertama dan mempertahankan jawaban sebelumnya.
- Saat pertanyaan berubah, fokus keyboard berpindah ke judul pertanyaan baru. Nama dan keadaan pilihan tersedia bagi teknologi bantu.
- Ringkasan akhir tetap memuat persiapan akun, stok, pajak, dan PPN. Tombol **Ubah jawaban** kembali ke awal seperti perilaku saat ini.
- Tampilan menggunakan token dan komponen yang sudah ada, dengan lebar konten yang nyaman di desktop dan padding yang cukup di ponsel.
- Status simpan, pesan gagal, dan perilaku tombol saat menyimpan tetap dipertahankan.

## Batasan

Perubahan ini tidak mengubah skema profil, pertanyaan, aturan percabangan, teks kalkulasi pajak, persistensi, atau akses berdasarkan peran. Tidak ada dependensi baru.

## Validasi

- Periksa alur perdagangan, jasa, dan manufaktur serta langkah kondisional OP dan badan usaha.
- Pastikan pilihan tidak maju otomatis; Lanjut dan Kembali menjaga jawaban.
- Pastikan indikator progres mencerminkan langkah yang tampil, termasuk pertanyaan kondisional.
- Pastikan alur ringkasan, penyimpanan, dan kegagalan simpan tetap berfungsi.
- Periksa keyboard, fokus, label teknologi bantu, ukuran ponsel, dan desktop.

## Keputusan

Perubahan ini memilih navigasi eksplisit dengan tombol **Lanjut** alih-alih perpindahan otomatis setelah memilih. Tujuannya memberi kontrol kepada pengguna dan membuat perilaku wizard lebih dapat diprediksi.
