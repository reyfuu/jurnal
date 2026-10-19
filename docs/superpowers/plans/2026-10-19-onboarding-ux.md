# Rencana Implementasi UX Onboarding

> Eksekusi: `superpowers:executing-plans`, langsung di branch `feat/onboarding-ux` sesuai persetujuan pengguna.

**Goal:** Onboarding satu pertanyaan per layar dengan navigasi eksplisit, progres yang jelas, dan akses keyboard.

**Architecture:** Gunakan ulang `ChoiceGrid`, `Button`, `Input`, serta fungsi `steps`, `answered`, dan `toProfile`. Perubahan pada `ProfilUsaha.tsx` dan CSS yang dibatasi ke komponen profil/onboarding; kontrak API tetap.

**Tech Stack:** React, TypeScript, shadcn/ui, CSS, native radio inputs.

**Spec:** `docs/superpowers/specs/2026-10-19-onboarding-ux-design.md`

## Batasan

- Satu pertanyaan per layar, maju hanya dengan Lanjut.
- Pertahankan pertanyaan, percabangan, pemetaan profil, dan ringkasan.
- Gunakan token desain serta komponen yang sudah terpasang; tanpa dependensi baru.

## Fokus pemeriksaan

- Pilihan melalui mouse atau tombol panah tidak memajukan layar.
- Kembali dan Ubah jawaban mempertahankan nilai dan menghitung ulang langkah kondisional.
- Fokus pada judul pertanyaan baru; radio dan petunjuknya terbaca teknologi bantu.
- Penyimpanan gagal dapat dicoba ulang; navigasi dinonaktifkan selama menyimpan.
- Pertanyaan panjang dan ringkasan terbaca tanpa scroll horizontal di lebar 320 px.

## Implementasi dan pemeriksaan

- [x] Ubah `ChoiceGrid` menjadi radio native berlabel, dengan keadaan pilihan dan fokus yang jelas. Periksa juga pemakainya di form Profil Usaha.
- [x] Hilangkan auto-advance; tambahkan progres, fokus judul, dan navigasi eksplisit di `OnboardingWizard`. Pertahankan API dan fungsi pemetaan.
- [x] Rapikan hierarki teks, susunan pilihan, spasi, dan tombol di desktop/mobile dengan CSS terbatas.
- [x] Jalankan Vitest yang sudah ada dan build frontend. Lakukan pemeriksaan browser dengan assertion untuk navigasi, percabangan, fokus, error/retry, simpan, dan batas viewport.
- [x] Review diff dan perbarui `HANDOFF.md` dengan hasil pemeriksaan.

Pengiriman: commit perubahan, push branch `feat/onboarding-ux`, dan buat PR ke `main`.

## Catatan eksekusi

Persetujuan desain dan instruksi “lanjut” sudah mencakup implementasi. Rencana dijalankan langsung tanpa gerbang persetujuan tambahan. Perubahan interaksi diverifikasi lewat browser nyata; tidak menambah library tes komponen.
