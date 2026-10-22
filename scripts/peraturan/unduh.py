#!/usr/bin/env python3
"""Unduh PDF peraturan dari JDIH Kemenkeu berdasarkan manifest, resumable.

Tidak butuh login: slug JDIH di manifest cukup, URL fulltext diambil dari
halaman JDIH yang publik.

Pemakaian:
    python3 scripts/peraturan/unduh.py                      # semua di manifest
    python3 scripts/peraturan/unduh.py --kategori FAKTUR PPS
    python3 scripts/peraturan/unduh.py --dry-run
    python3 scripts/peraturan/unduh.py --jeda 5

Resumable: dokumen yang PDF-nya sudah ada dan lolos verifikasi dilewati.
"""
from __future__ import annotations

import argparse
import csv
import hashlib
import json
import re
import subprocess
import sys
import time
import urllib.parse
import urllib.request
from pathlib import Path

AKAR = Path(__file__).resolve().parents[2]
MANIFEST = Path(__file__).parent / "manifest-primbon.json"
TUJUAN = AKAR / "docs" / "peraturan"
INDEX = TUJUAN / "index.csv"
JDIH = "https://jdih.kemenkeu.go.id"

KOLOM = [
    "uuid", "kategori", "kode", "judul", "jenis", "penerbit", "tahun",
    "tanggal_penetapan", "tanggal_pengundangan", "berlaku", "sumber_ln",
    "status", "halaman", "ukuran_byte", "sha256", "primbon_url", "jdih_url",
    "sumber_url", "file", "catatan",
]

SLUG_KATEGORI = {
    "PPN": "ppn", "PPnBM": "ppnbm", "PPh": "pph", "KUP": "kup", "PBB": "pbb",
    "BEA_MATERAI": "bea-materai", "BEA_CUKAI": "bea-cukai",
    "PAJAK_DAERAH": "pajak-daerah", "UMKM": "umkm", "FAKTUR": "faktur",
    "NPWP": "npwp", "PPS": "pps", "CARBON_TAX": "pajak-karbon",
    "TRANSFER_PRICING": "transfer-pricing", "TINDAK_PIDANA": "tindak-pidana",
    "SENGKETA_PAJAK": "sengketa-pajak", "KURS": "kurs", "KEUNEG": "keuangan-negara",
    "PENG_SP_DJP": "pengumuman-djp",
}


def ambil(url: str, timeout: int = 60, coba: int = 3) -> bytes:
    """GET dengan percobaan ulang. JDIH sesekali membalas 504 sementara."""
    galat: Exception | None = None
    for n in range(coba):
        try:
            req = urllib.request.Request(url, headers={"User-Agent": "Mozilla/5.0"})
            with urllib.request.urlopen(req, timeout=timeout) as r:
                return r.read()
        except Exception as e:  # noqa: BLE001
            galat = e
            if n < coba - 1:
                time.sleep(3 * (n + 1))
    raise galat  # type: ignore[misc]


def teks(url: str) -> str:
    return ambil(url).decode("utf-8", "replace")


def nama_berkas(kode: str, tahun: str) -> str:
    """PER-11/PJ/2025 -> per-11-pj-2025 ; PMK Nomor 196/PMK.03/2021 -> pmk-196-2021"""
    s = kode.lower()
    s = re.sub(r"\b(nomor|no\.?|tahun)\b", " ", s)
    # "196/PMK.03/2021" -> "196/2021": buang kode unit bertitik di tengah.
    # Pola tanpa titik seperti "/PJ/" dipertahankan: nama bakunya per-11-pj-2025.
    s = re.sub(r"/[a-z]+\.\d+/", "/", s)
    s = re.sub(r"[^a-z0-9]+", "-", s).strip("-")
    s = re.sub(r"-+", "-", s)
    # buang pengulangan penanda jenis, misalnya "pmk-196-pmk-2021"
    bagian, hasil = s.split("-"), []
    for b in bagian:
        if b.isalpha() and b in hasil:
            continue
        hasil.append(b)
    s = "-".join(hasil)
    if tahun and tahun not in s:
        s = f"{s}-{tahun}"
    return s or "tanpa-kode"


def url_fulltext(slug: str) -> tuple[list[str], str]:
    """Kembalikan (daftar_url_pdf_berurut, jdih_url).

    JDIH punya dua bentuk tautan:
      - dokumen baru : /api/download/<uuid>/<FILE>.pdf
      - dokumen lama : /api/download/fulltext/<tahun>/<FILE>.pdf   (IsNew:false)
    Nama berkas bisa memuat spasi, misalnya "37 Tahun 2025.pdf", jadi pola
    dibaca sampai kutip penutup atribut href lalu di-encode.

    Mengembalikan daftar, bukan satu URL, karena berkas "Fulltext PDF" sesekali
    404 di server JDIH sementara salinan lain di halaman yang sama hidup
    (contoh: PMK 37 Tahun 2025). Berkas bersufiks 'abs' adalah abstrak, jadi
    ditaruh paling belakang. Dokumen yang hanya punya "Fulltext HTML" akan
    menghasilkan daftar kosong.
    """
    jdih_url = f"{JDIH}/dok/{slug}"
    html = teks(jdih_url)
    # Label tombol adalah sumber kebenaran, bukan nama berkas: untuk PMK 8/2026
    # berkas '8 Tahun 2026.pdf' berlabel Abstrak, fulltext-nya '2026pmkeuangan008.pdf'.
    label: dict[str, str] = {}
    for m in re.finditer(r'<a[^>]+href="(/api/download/[^"]+?\.pdf)"[^>]*>(.*?)</a>', html, re.S):
        label.setdefault(m.group(1), re.sub(r"<[^>]+>", " ", m.group(2)).lower())
    kandidat = re.findall(r'href="(/api/download/[^"]+?\.pdf)"', html)
    urut = list(dict.fromkeys(kandidat))

    def peringkat(c: str) -> int:
        lb = label.get(c, "")
        if "abstrak" in lb or re.search(r"abs\.pdf$", c):
            return 2
        return 0 if "fulltext" in lb else 1

    pilih = sorted(urut, key=peringkat)  # sorted stabil: urutan halaman dipertahankan
    return [JDIH + urllib.parse.quote(c, safe="/:") for c in pilih], jdih_url


def url_html(slug: str) -> list[str]:
    """Fulltext HTML resmi JDIH (peraturan lama tanpa PDF), mis.
    /api/download/fulltext/2008/51TAHUN2008PP.htm"""
    html = teks(f"{JDIH}/dok/{slug}")
    c = list(dict.fromkeys(re.findall(r'href="(/api/download/[^"]+?\.html?)"', html)))
    return [JDIH + urllib.parse.quote(x, safe="/:") for x in c]


def simpan_html(index: dict, d: dict, p_pdf: Path, rel_pdf: str, i: int, n: int) -> bool:
    """Simpan fulltext HTML resmi bila tidak ada PDF sah. True kalau berhasil."""
    for u in url_html(d["jdih_slug"]):
        try:
            isi = ambil(u, timeout=120, coba=2)
        except Exception:  # noqa: BLE001
            continue
        teks_polos = re.sub(r"<[^>]+>", " ", isi.decode("latin-1")).upper()
        angka = re.findall(r"\d+", d["kode"])
        if len(isi) < 2000 or (angka and angka[0] not in teks_polos):
            continue
        p = p_pdf.with_suffix(".htm")
        p.parent.mkdir(parents=True, exist_ok=True)
        p.write_bytes(isi)
        m = meta_jdih(d["jdih_slug"])
        index[d["uuid"]] = {
            "uuid": d["uuid"], "kategori": ";".join(d["kategori"]), "kode": d["kode"],
            "judul": m["judul_resmi"] or d["judul"], "jenis": m["jenis"] or d.get("jenis", ""),
            "penerbit": "", "tahun": d.get("tahun", ""),
            "tanggal_penetapan": m["tanggal_penetapan"],
            "tanggal_pengundangan": m["tanggal_pengundangan"],
            "berlaku": m["berlaku"], "sumber_ln": m["sumber_ln"],
            "status": d.get("status", ""), "halaman": "", "ukuran_byte": str(len(isi)),
            "sha256": hashlib.sha256(isi).hexdigest()[:16],
            "primbon_url": f"https://primbonpajak.id/regulations/{d['uuid']}",
            "jdih_url": f"{JDIH}/dok/{d['jdih_slug']}", "sumber_url": u,
            "file": rel_pdf[:-4] + ".htm", "catatan": "fulltext HTML resmi, PDF tidak tersedia",
        }
        print(f"[{i}/{n}] {d['kode']}: OK fulltext HTML, {len(isi)//1024} KB")
        return True
    return False


def url_cadangan(kode: str, tahun: str) -> str | None:
    """Cadangan peraturan.go.id untuk UU/PP lama yang di JDIH hanya tersedia HTML.

    Pola berkasnya `uu14-2002.pdf`, `pp46-2017.pdf`. Hanya dipakai kalau JDIH
    tidak punya PDF, dan hasilnya tetap lewat verifikasi yang sama. Kalau
    peraturan.go.id juga tidak punya PDF, ia membalas halaman HTML dengan
    status 200, jadi verifikasi header `%PDF` yang menangkapnya.
    """
    k = kode.strip()
    # bentuk pendek: "PP Nomor 46 Tahun 2017", "UU 10 TAHUN 2020"
    m = re.match(r"^(uu|pp|perpres|perppu)\b\D*(\d+)", k, re.I)
    if not m:
        # bentuk panjang: "Peraturan Pemerintah Nomor 31 Tahun 2007"
        m2 = re.match(r"^Peraturan\s+(Presiden|Pemerintah)\s+(?:Nomor|No\.?)\s*(\d+)", k, re.I)
        if not m2:
            return None
        jenis = "perpres" if m2.group(1).lower() == "presiden" else "pp"
        no = m2.group(2)
    else:
        jenis, no = m.group(1).lower(), m.group(2)
    if not tahun:
        return None
    return f"https://peraturan.go.id/files/{jenis}{no}-{tahun}.pdf"


def meta_jdih(slug: str) -> dict:
    html = teks(f"{JDIH}/dok/{slug}")

    def satu(key: str) -> str:
        # Payload RSC di-escape: \"Sumber\":\"BNRI 2026 (138)\". Ambil sampai \"
        # pertama. Pola lama ikut menelan \" sehingga merembet ke field berikutnya.
        m = re.search(r'\\"' + key + r'\\":\\"(.*?)\\"', html)
        v = m.group(1).replace("\\\\", "").strip() if m else ""
        # Tanggal di payload RSC berprefiks '$D': "$D2026-02-27T12:00:00.000Z"
        return v[2:] if v.startswith("$D") else v

    judul_resmi = ""
    m = re.search(r"<title>([^<]*)", html)
    if m:
        judul_resmi = m.group(1).split(" | ")[0]
        if " - " in judul_resmi:
            judul_resmi = judul_resmi.split(" - ", 1)[1]
    jenis = ""
    m = re.search(r'\\"Keterangan\\":\\"([^\\"]+)\\",\\"Singkatan\\"', html)
    if m:
        jenis = m.group(1)
    return {
        "judul_resmi": judul_resmi,
        "jenis": jenis,
        "tanggal_penetapan": satu("TglPenetapan")[:10],
        "tanggal_pengundangan": satu("TglPengundangan")[:10],
        "berlaku": satu("TglMulaiBerlaku")[:10],
        "sumber_ln": satu("Sumber"),
    }


def adalah_abstrak(p: Path) -> bool:
    """JDIH melampirkan abstrak 1 halaman yang diawali 'ABSTRAK PERATURAN'."""
    t = subprocess.run(["pdftotext", "-l", "1", str(p), "-"],
                       capture_output=True, text=True).stdout
    return t.lstrip().upper().startswith("ABSTRAK PERATURAN")


def verifikasi(p: Path, kode: str) -> tuple[bool, str, str]:
    """(ok, halaman, alasan). Alasan 'abstrak' = PDF sah tapi bukan fulltext."""
    if not p.exists() or p.stat().st_size == 0:
        return False, "", "kosong"
    with p.open("rb") as f:
        if f.read(5)[:4] != b"%PDF":
            return False, "", "bukan PDF (mungkin halaman HTML)"
    r = subprocess.run(["pdfinfo", str(p)], capture_output=True, text=True)
    if r.returncode != 0:
        return False, "", "pdfinfo gagal"
    hal = ""
    for ln in r.stdout.splitlines():
        if ln.startswith("Pages:"):
            hal = ln.split()[1]
    if adalah_abstrak(p):
        return True, hal, "abstrak"
    # nomor peraturan harus muncul di halaman awal
    t = subprocess.run(["pdftotext", "-l", "4", str(p), "-"],
                       capture_output=True, text=True).stdout
    angka = re.findall(r"\d+", kode)
    if angka:
        # OCR kadang mengubah 0 jadi O, jadi cocokkan longgar
        datar = re.sub(r"[^0-9A-Za-z]", "", t.upper()).replace("O", "0")
        if not any(a in datar for a in angka[:1]):
            return True, hal, "peringatan: nomor tidak terbaca di 4 halaman awal"
    return True, hal, ""


def baca_index() -> dict[str, dict]:
    if not INDEX.exists():
        return {}
    with INDEX.open(encoding="utf-8") as f:
        rows = list(csv.DictReader(f))
    out = {}
    for r in rows:
        kunci = r.get("uuid") or r.get("primbon_url", "").rsplit("/", 1)[-1]
        out[kunci] = r
    return out


def tulis_index(baris: dict[str, dict]) -> None:
    urut = sorted(baris.values(), key=lambda r: (r.get("kategori", ""), r.get("kode", "")))
    # tulis ke berkas sementara lalu rename: proses yang dihentikan di tengah
    # tidak boleh meninggalkan index.csv setengah jadi
    tmp = INDEX.with_suffix(".csv.tmp")
    with tmp.open("w", newline="", encoding="utf-8") as f:
        w = csv.DictWriter(f, fieldnames=KOLOM)
        w.writeheader()
        for r in urut:
            w.writerow({k: r.get(k, "") for k in KOLOM})
    tmp.replace(INDEX)


def tercemar(r: dict) -> bool:
    """Baris dari versi lama yang regex metadatanya merembet ke JSON."""
    return any("\\" in r.get(k, "") or len(r.get(k, "")) > 120 or r.get(k, "").startswith("$D")
               for k in ("sumber_ln", "tanggal_penetapan", "berlaku"))


def catat_tanpa_pdf(index: dict, d: dict, alasan: str) -> None:
    """Catat dokumen yang memang tidak punya PDF, supaya tidak dicoba terus."""
    uuid = d["uuid"]
    index[uuid] = {
        "uuid": uuid, "kategori": ";".join(d["kategori"]), "kode": d["kode"],
        "judul": d["judul"], "jenis": d.get("jenis", ""), "penerbit": "",
        "tahun": d.get("tahun", ""), "status": d.get("status", ""),
        "primbon_url": f"https://primbonpajak.id/regulations/{uuid}",
        "jdih_url": f"{JDIH}/dok/{d['jdih_slug']}" if d.get("jdih_slug") else "",
        "sumber_ln": alasan, "file": "tanpa-pdf",
    }


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--kategori", nargs="*", help="batasi kategori")
    ap.add_argument("--jeda", type=float, default=3.0, help="detik antar dokumen")
    ap.add_argument("--dry-run", action="store_true")
    ap.add_argument("--batas", type=int, help="maksimal dokumen diproses")
    a = ap.parse_args()

    man = json.loads(MANIFEST.read_text(encoding="utf-8"))
    dok = man["dokumen"]
    if a.kategori:
        pilih = set(a.kategori)
        dok = [d for d in dok if pilih & set(d["kategori"])]
    if a.batas:
        dok = dok[: a.batas]

    index = baca_index()
    stat = {"lewat": 0, "unduh": 0, "gagal": 0, "tanpa_pdf": 0}

    for i, d in enumerate(dok, 1):
        uuid, kode = d["uuid"], d["kode"]
        kat_utama = d["kategori"][0]
        subdir = SLUG_KATEGORI.get(kat_utama, kat_utama.lower())
        rel = f"{subdir}/{nama_berkas(kode, d.get('tahun',''))}.pdf"
        p = TUJUAN / rel

        lama = index.get(uuid)
        if lama and lama.get("file", "").endswith(".htm") and (TUJUAN / lama["file"]).exists():
            print(f"[{i}/{len(dok)}] {kode}: lewat (sudah lengkap)")
            stat["lewat"] += 1
            continue
        if lama and lama.get("file") not in (None, "", "tanpa-pdf"):
            ok, _, alasan = verifikasi(TUJUAN / lama["file"], kode)
            if ok and alasan == "abstrak" and d.get("jdih_slug") \
                    and not lama.get("catatan", "").startswith("hanya abstrak"):
                print(f"[{i}/{len(dok)}] {kode}: berkas lama hanya abstrak, unduh ulang")
                ok = False
            if ok:
                kats = set(filter(None, lama.get("kategori", "").split(";")))
                if not set(d["kategori"]) <= kats:
                    lama["kategori"] = ";".join(sorted(kats | set(d["kategori"])))
                    print(f"[{i}/{len(dok)}] {kode}: sudah ada, kategori diperbarui")
                else:
                    print(f"[{i}/{len(dok)}] {kode}: lewat (sudah lengkap)")
                if tercemar(lama) and d.get("jdih_slug") and not a.dry_run:
                    try:
                        m = meta_jdih(d["jdih_slug"])
                        for k in ("tanggal_penetapan", "tanggal_pengundangan", "berlaku", "sumber_ln"):
                            lama[k] = m[k]
                        print(f"[{i}/{len(dok)}] {kode}: metadata diperbaiki")
                        tulis_index(index)
                    except Exception as e:  # noqa: BLE001
                        print(f"[{i}/{len(dok)}] {kode}: gagal perbaiki metadata {e}", file=sys.stderr)
                stat["lewat"] += 1
                continue

        if not d.get("jdih_slug"):
            print(f"[{i}/{len(dok)}] {kode}: TANPA SLUG JDIH, dicatat tanpa-pdf")
            catat_tanpa_pdf(index, d, "tanpa slug JDIH")
            stat["tanpa_pdf"] += 1
            continue

        if a.dry_run:
            print(f"[{i}/{len(dok)}] {kode} -> {rel} (dry-run)")
            continue

        try:
            sumber, jdih_url = url_fulltext(d["jdih_slug"])
            cadangan = url_cadangan(kode, d.get("tahun", ""))
            if cadangan:
                sumber = sumber + [cadangan]
            if not sumber:
                if simpan_html(index, d, p, rel, i, len(dok)):
                    stat["unduh"] += 1
                    tulis_index(index)
                    time.sleep(a.jeda)
                    continue
                catat_tanpa_pdf(index, d, "JDIH hanya HTML, tanpa pola cadangan")
                stat["tanpa_pdf"] += 1
                print(f"[{i}/{len(dok)}] {kode}: hanya HTML, dicatat tanpa-pdf")
                tulis_index(index)
                time.sleep(a.jeda)
                continue

            # Coba kandidat berurutan: berkas "Fulltext PDF" sesekali 404
            # padahal salinan lain di halaman yang sama bisa diunduh.
            p.parent.mkdir(parents=True, exist_ok=True)
            src = None
            ok = False
            hal = catatan = ""
            galat_akhir = ""
            abstrak: tuple[bytes, str, str] | None = None
            for kandidat in sumber:
                try:
                    isi_unduh = ambil(kandidat, timeout=180, coba=2)
                except Exception as e:  # noqa: BLE001
                    galat_akhir = str(e)
                    continue
                p.write_bytes(isi_unduh)
                ok, hal, catatan = verifikasi(p, kode)
                if ok and catatan == "abstrak":
                    # simpan sebagai cadangan, terus cari fulltext
                    abstrak = abstrak or (isi_unduh, hal, kandidat)
                    ok = False
                    p.unlink(missing_ok=True)
                    continue
                if ok:
                    src = kandidat
                    break
                p.unlink(missing_ok=True)
                galat_akhir = catatan

            if not ok and abstrak:
                isi_unduh, hal, src = abstrak
                p.write_bytes(isi_unduh)
                ok, catatan = True, "hanya abstrak, fulltext tidak tersedia"

            if not ok:
                if simpan_html(index, d, p, rel, i, len(dok)):
                    stat["unduh"] += 1
                    tulis_index(index)
                    time.sleep(a.jeda)
                    continue
                if "bukan PDF" in galat_akhir or galat_akhir == "kosong" or not galat_akhir:
                    catat_tanpa_pdf(index, d, "semua sumber hanya HTML atau tidak terbaca")
                    stat["tanpa_pdf"] += 1
                    print(f"[{i}/{len(dok)}] {kode}: tidak ada PDF sah, tanpa-pdf")
                    tulis_index(index)
                    time.sleep(a.jeda)
                    continue
                raise RuntimeError(galat_akhir)

            m = meta_jdih(d["jdih_slug"])
            isi = p.read_bytes()
            index[uuid] = {
                "uuid": uuid, "kategori": ";".join(d["kategori"]), "kode": kode,
                "judul": m["judul_resmi"] or d["judul"],
                "jenis": m["jenis"] or d.get("jenis", ""), "penerbit": "",
                "tahun": d.get("tahun", ""),
                "tanggal_penetapan": m["tanggal_penetapan"],
                "tanggal_pengundangan": m["tanggal_pengundangan"],
                "berlaku": m["berlaku"], "sumber_ln": m["sumber_ln"],
                "status": d.get("status", ""), "halaman": hal,
                "ukuran_byte": str(len(isi)),
                "sha256": hashlib.sha256(isi).hexdigest()[:16],
                "primbon_url": f"https://primbonpajak.id/regulations/{uuid}",
                "jdih_url": jdih_url, "sumber_url": src, "file": rel,
                "catatan": catatan if catatan.startswith("hanya abstrak") else "",
            }
            stat["unduh"] += 1
            tanda = f" ({catatan})" if catatan else ""
            print(f"[{i}/{len(dok)}] {kode}: OK {hal} hal, {len(isi)//1024} KB{tanda}")
        except Exception as e:  # noqa: BLE001
            stat["gagal"] += 1
            print(f"[{i}/{len(dok)}] {kode}: GAGAL {e}", file=sys.stderr)

        tulis_index(index)
        time.sleep(a.jeda)

    if not a.dry_run:
        tulis_index(index)
    print(f"\nringkasan: {stat} | total baris index: {len(index)}")
    return 1 if stat["gagal"] else 0


if __name__ == "__main__":
    raise SystemExit(main())
