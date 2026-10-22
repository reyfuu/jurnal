# AGENTS.md — Master Pajak (repo laporan-bisnis)

> Lihat dan patuhi seluruh instruksi di [agent.md](file:///home/reyfuu/pajak/agent.md). Setiap sesi WAJIB mengaktifkan: `rtk`, `graphify`, `leantoken`, `grillme`, `superpowers`, `antislop`, `taste`, `21dev`, `shadcn/ui`, `ponytail`, `ui ux pro max`, `kokonut ui`, `impeccable`, `grillme with docs`, `skillspector`, `agent browser`, `tdd`, `smokinggun` (hotspot: temuan wajib dibuktikan benchmark, lihat agent.md §1.B), `handoff`, `harness2` (Google Research), `rea` (reverse engineering, situasional: lihat agent.md §1.D), dan `improve-codebase-architecture` / `setup-matt-pocock-skills`.

## Atribusi commit dan PR

- Jangan tambahkan trailer `Co-authored-by` atau atribusi penulis untuk Claude, Codex, maupun agen AI lainnya pada commit atau PR.
- Commit dan PR hanya mencantumkan penulis manusia yang memang berkontribusi.

## Batas penggunaan browser

- Jangan membuka browser baru. Untuk uji di browser, pakai browser yang sudah berjalan (cek dulu, misalnya dengan `list_pages` di chrome-devtools). Buka browser baru hanya jika tidak ada browser yang berjalan.
- Mesin kerja memiliki RAM 8 GB. Gunakan browser seperlunya: pakai ulang satu sesi/jendela, batasi tab yang dibuka, dan hindari menjalankan beberapa browser atau proses berat bersamaan.
- Setelah selesai, tutup hanya tab atau jendela yang dibuka agen. Jangan tutup tab milik pengguna.
- Jika browser mulai lambat atau penggunaan memori tinggi, hentikan pekerjaan browser yang tidak perlu sebelum melanjutkan agar aplikasi tidak kehabisan RAM dan perlu dimulai ulang.

