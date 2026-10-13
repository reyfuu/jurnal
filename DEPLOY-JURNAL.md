# Deploy Jurnal AI ke `jurnal.skinku.id`

Berdiri sendiri di samping portal B2B: **folder terpisah, database terpisah,
subdomain terpisah**. `system.skinku.id` dan WordPress `skinku.id` tidak
tersentuh sama sekali.

Server (sama dengan portal B2B):

| | |
|---|---|
| SSH | `ssh -p 65002 u864765086@153.92.11.179` |
| PHP 8.3 | `/opt/alt/php83/usr/bin/php` |
| Folder aplikasi | `~/jurnal-src` (di luar `public_html`) |
| Document root | `/home/u864765086/jurnal-src/jurnal-ai/public` |
| Database | `u864765086_jurnal` |

> Hanya folder `public/` yang boleh diekspos web. Kalau document root diarahkan
> ke folder aplikasi, `.env` ikut bisa diunduh orang — kunci API dan sandi
> database bocor seketika.

---

## 1. Subdomain (hPanel)

hPanel → **Domain → Subdomain** → buat `jurnal` untuk `skinku.id`.
Catat document root bawaannya; akan diganti di langkah 5.

## 2. Ambil kode (SSH)

```bash
ssh -p 65002 u864765086@153.92.11.179
cd ~
git clone https://github.com/raventcreative/skinku-b2b-portal.git jurnal-src
cd jurnal-src
git checkout claude/quirky-cori-kbvckj
cd jurnal-ai
```

## 3. Dependency

```bash
/opt/alt/php83/usr/bin/php $(which composer) install --no-dev --optimize-autoloader
```

Kalau `composer` tidak ada di PATH:

```bash
/opt/alt/php83/usr/bin/php ~/composer.phar install --no-dev --optimize-autoloader
```

`npm` tidak perlu — hasil build CSS/JS sudah ikut di repo.

## 4. Konfigurasi

```bash
cp .env.production.example .env
nano .env
```

Isi: `DB_PASSWORD`, `SEED_ADMIN_EMAIL`, `SEED_ADMIN_PASSWORD`, `OPENAI_API_KEY`,
`AI_MODEL`. Lalu:

```bash
/opt/alt/php83/usr/bin/php artisan key:generate --force
/opt/alt/php83/usr/bin/php artisan migrate --force
/opt/alt/php83/usr/bin/php artisan db:seed --force

chmod -R 775 storage bootstrap/cache

/opt/alt/php83/usr/bin/php artisan config:cache
/opt/alt/php83/usr/bin/php artisan route:cache
/opt/alt/php83/usr/bin/php artisan view:cache
```

Seeder akan **menolak jalan** kalau `SEED_ADMIN_PASSWORD` kosong, kurang dari
12 karakter, atau memakai sandi umum. Itu disengaja.

`storage:link` TIDAK diperlukan: dokumen yang diupload sengaja disimpan di
disk privat dan dilayani lewat route ber-autentikasi, bukan folder publik.

## 5. Arahkan document root

hPanel → Subdomain `jurnal.skinku.id` → **Document root** →
`/home/u864765086/jurnal-src/jurnal-ai/public`

Kalau hPanel memaksa folder di dalam `public_html`, buat symlink:

```bash
ln -s /home/u864765086/jurnal-src/jurnal-ai/public ~/public_html/jurnal
```

## 6. SSL

hPanel → **SSL** → terbitkan sertifikat gratis untuk `jurnal.skinku.id`.
Wajib: `SESSION_SECURE_COOKIE=true` membuat sesi hanya dikirim lewat HTTPS,
jadi tanpa SSL login tidak akan nyangkut.

## 7. Cek

Buka `https://jurnal.skinku.id` → login dengan `SEED_ADMIN_EMAIL` dan
`SEED_ADMIN_PASSWORD` dari `.env`.

Verifikasi cepat:

- Buat satu klien → COA 50 akun terbentuk otomatis.
- Upload satu foto struk → AI membacanya → review → posting.
- Buka Neraca → keterangan hijau "Aktiva = Pasiva".

---

## Update berikutnya

```bash
cd ~/jurnal-src
git pull
cd jurnal-ai
/opt/alt/php83/usr/bin/php $(which composer) install --no-dev --optimize-autoloader
/opt/alt/php83/usr/bin/php artisan migrate --force
/opt/alt/php83/usr/bin/php artisan optimize:clear
/opt/alt/php83/usr/bin/php artisan config:cache
/opt/alt/php83/usr/bin/php artisan route:cache
/opt/alt/php83/usr/bin/php artisan view:cache
```

`optimize:clear` sebelum cache ulang itu wajib — tanpa itu konfigurasi lama
tetap dipakai dan perubahan `.env` seolah tidak berefek.

---

## Kalau bermasalah

| Gejala | Sebabnya biasanya |
|---|---|
| Halaman putih / error 500 | `storage` & `bootstrap/cache` belum writable (`chmod -R 775`) |
| Isi folder aplikasi terlihat di browser | Document root salah — harus menunjuk ke `.../jurnal-ai/public` |
| Login selalu balik ke form | SSL belum aktif, padahal `SESSION_SECURE_COOKIE=true` |
| Perubahan `.env` tidak berefek | Cache konfigurasi masih lama → `artisan optimize:clear` |
| `could not find driver` | PHP yang dipakai bukan 8.3, atau ekstensi `pdo_mysql` mati |
| Seeder menolak jalan | `SEED_ADMIN_PASSWORD` belum diisi / terlalu lemah. Itu memang pagarnya |

## Yang TIDAK boleh dilakukan di server

- `migrate:fresh` atau `migrate:refresh` — menghapus seluruh pembukuan.
- `db:seed --class=DemoSeeder` — menyuntik 18 jurnal karangan ke buku asli.
- Mengarahkan `DB_DATABASE` ke database portal B2B.
- `APP_DEBUG=true` — halaman error akan menampilkan isi `.env`, termasuk kunci API.
