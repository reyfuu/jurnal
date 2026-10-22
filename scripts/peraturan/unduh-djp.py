#!/usr/bin/env python3
"""Unduh dokumen `tanpa-pdf` dari pajak.go.id (situs resmi DJP).

JDIH Kemenkeu tidak memuat banyak PER/KEP/PENG Dirjen Pajak (contoh
PER-16/PJ/2021: JDIH "tidak ditemukan"), padahal pajak.go.id punya salinannya.
Pencarian berdasarkan nomor di pajak.go.id tidak berfungsi, jadi dicari lewat
kata awal judul, lalu cadangannya menyisir daftar per tahun + jenis. Hasil
selalu dicocokkan dengan nomor peraturan, lalu PDF lewat verifikasi yang sama
dengan unduh.py.

Pemakaian:
    python3 scripts/peraturan/unduh-djp.py            # semua baris tanpa-pdf
    python3 scripts/peraturan/unduh-djp.py --batas 3 --dry-run
"""
from __future__ import annotations

import argparse
import hashlib
import importlib.util
import json
import re
import subprocess
import sys
import time
import urllib.parse
from pathlib import Path

_s = importlib.util.spec_from_file_location("unduh", Path(__file__).parent / "unduh.py")
U = importlib.util.module_from_spec(_s)
_s.loader.exec_module(U)

DJP = "https://www.pajak.go.id"
# kode jenis dokumen di filter /id/peraturan
JENIS_DJP = {"PER": "13894", "KEP": "13877", "PENG": "13889", "KMK": "13881",
             "PMK": "13899", "PP": "13902", "PERPRES": "13903", "UU": "13926"}
BUANG = {"NOMOR", "NO", "TAHUN", "KMK", "PMK", "PP", "UU", "PERPRES", "SEMA"}


def token(nomor: str) -> list[str]:
    tok = re.findall(r"[A-Z]+|\d+", nomor.upper())
    return [t.lstrip("0") or "0" if t.isdigit() else t for t in tok if t not in BUANG]


def nomor_sama(a: str, b: str) -> bool:
    """'PER-01/PJ/2025' setara 'PER - 1/PJ/2025'; SE-15 tidak setara PER-15."""
    ta = token(a)
    return bool(ta) and ta == token(b)


def jenis_kode(kode: str) -> str:
    k = kode.upper()
    if re.match(r"^PERATURAN PEMERINTAH|^PP\b", k):
        return "PP"
    if re.match(r"^PERPRES|^PERATURAN PRESIDEN", k):
        return "PERPRES"
    m = re.match(r"^(PER|KEP|PENG|SE|KMK|PMK|UU)\b", k)
    return m.group(1) if m else ""


def parse_hasil(html: str) -> list[tuple[str, str]]:
    """Baris hasil /id/peraturan -> [(nomor, href_detail)]."""
    return [(n.strip(), h) for h, n in re.findall(
        r'views-field-field-nomor-dokumen"><h3[^>]*><a href="([^"]+)"[^>]*>([^<]+)</a>', html)]


def pdf_lampiran(html: str) -> list[str]:
    """PDF di halaman detail; berkas 'Salinan' (naskah utama) didahulukan."""
    url = [urllib.parse.urljoin(DJP, h) for h in dict.fromkeys(
        re.findall(r'href="([^"]*/sites/default/files/[^"]+\.pdf)"', html, re.I))]
    return sorted(url, key=lambda u: 0 if "salinan" in u.lower() else 1)


def naskah_html(html: str) -> str:
    """Isi div `field--name-field-body-dalam-html` di halaman detail DJP.

    DJP menaruh batang tubuh (Menimbang s.d. Pasal penutup) sebagai HTML,
    sedangkan PDF di halaman sering hanya lampiran. Div bersarang, jadi
    ditutup dengan menghitung kedalaman. Isi pendek (mis. 'Lihat lampiran')
    dianggap tidak ada naskah.
    """
    m = re.search(r'<div[^>]*class="[^"]*field--name-field-body-dalam-html[^"]*"[^>]*>', html)
    if not m:
        return ""
    dalam = 1
    for t in re.finditer(r"<(/?)div\b", html[m.end():]):
        dalam += -1 if t.group(1) else 1
        if dalam == 0:
            isi = html[m.end():m.end() + t.start()]
            break
    else:
        return ""
    teks = re.sub(r"\s+", " ", re.sub(r"<[^>]+>", " ", isi))
    return isi.strip() if len(teks) >= 800 else ""


def adalah_lampiran(url: str, hal1: str) -> bool:
    """PDF hanya lampiran bila nama berkasnya memuat 'Lampiran' atau halaman
    pertamanya diawali 'LAMPIRAN' (pdftotext kadang menaruh '-6' di depan)."""
    nama = urllib.parse.unquote(url.rsplit("/", 1)[-1]).lower()
    awal = re.sub(r"^[^A-Za-z]+", "", hal1).upper()
    return "lampiran" in nama or awal.startswith("LAMPIRAN")


def hal_pertama(p: Path) -> str:
    return subprocess.run(["pdftotext", "-l", "1", str(p), "-"], capture_output=True, text=True).stdout


def simpan_naskah(p: Path, kode: str, url: str, isi: str) -> None:
    isi = re.sub(r"<script\b.*?</script>", "", isi, flags=re.S | re.I)
    p.write_text(
        f'<!doctype html>\n<html lang="id"><head><meta charset="utf-8"><title>{kode}</title></head>\n'
        f'<body>\n<p><small>Sumber: <a href="{url}">{url}</a> (naskah HTML resmi DJP)</small></p>\n'
        f"{isi}\n</body></html>\n", encoding="utf-8")


BPK = "https://peraturan.bpk.go.id"


def slug_bpk(kode: str) -> str:
    """Slug detail JDIH BPK: 'PP Nomor 18 Tahun 2025' -> 'pp-no-18-tahun-2025'.
    Hanya PP/Perpres/UU; PER/SE Dirjen Pajak tidak dimuat BPK."""
    j = jenis_kode(kode)
    m = re.search(r"(\d+)\D+(\d{4})\s*$", kode)
    if j not in ("PP", "PERPRES", "UU") or not m:
        return ""
    return f"{j.lower()}-no-{m.group(1).lstrip('0')}-tahun-{m.group(2)}"


def cari_bpk(d: dict) -> str | None:
    """URL PDF di JDIH BPK. Pencarian BPK berbasis kata, jadi kueri = kode +
    potongan judul; hasil disaring dengan slug persis."""
    slug = slug_bpk(d["kode"])
    if not slug:
        return None
    # pencarian BPK hanya kenal nama jenis panjang: "PP Nomor 18" tidak ketemu,
    # "Peraturan Pemerintah Nomor 18 Tahun 2025" ketemu
    panjang = {"pp": "Peraturan Pemerintah", "perpres": "Peraturan Presiden", "uu": "Undang-Undang"}
    j, no, th = re.match(r"([a-z]+)-no-(\d+)-tahun-(\d{4})", slug).groups()
    for q in (f"{panjang[j]} Nomor {no} Tahun {th}", d["kode"]):
        html = U.teks(f"{BPK}/Search?{urllib.parse.urlencode({'keywords': q})}")
        for det in dict.fromkeys(re.findall(rf'href="(/Details/\d+/{slug})"', html)):
            dl = re.findall(r'href="(/Download/[^"]+\.pdf)"', U.teks(BPK + det))
            if dl:
                return BPK + dl[0]
    return None


def kata_kunci(judul: str) -> list[str]:
    """Filter judul pajak.go.id berupa 'mengandung', jadi pakai potongan awal judul.
    Judul Primbon kadang diringkas, maka potongan pendek dicoba belakangan."""
    kata = re.sub(r"[^\w\s/-]", " ", judul).split()
    out = [" ".join(kata[:n]) for n in (8, 5) if kata]
    return list(dict.fromkeys(out))


def cari(d: dict, jeda: float) -> list[str]:
    """Kembalikan href detail yang nomornya cocok."""
    kode, tahun = d["kode"], d.get("tahun", "")
    cocok: list[str] = []

    def sisir(q: dict, halaman: int) -> None:
        for p in range(halaman):
            html = U.teks(f"{DJP}/id/peraturan?{urllib.parse.urlencode({**q, 'page': p})}")
            baris = parse_hasil(html)
            cocok.extend(h for n, h in baris if nomor_sama(kode, n) and h not in cocok)
            time.sleep(jeda)
            if not baris or f"page={p + 1}" not in html:
                return

    for k in kata_kunci(d["judul"]):
        sisir({"title": k}, 3)
        if cocok:
            return cocok
    j = JENIS_DJP.get(jenis_kode(kode))
    if j and tahun.isdigit() and 1983 <= int(tahun) <= 2026:
        sisir({"field_jenis_dokumen_target_id": j, "field_tahun_peraturan_value": str(2027 - int(tahun))}, 40)
    return cocok


def proses(d: dict, r: dict, jeda: float, dry: bool) -> str:
    """Isi baris index `r` dari pajak.go.id. Kembalikan ringkasan untuk log.

    Urutan pilihan berkas utama: PDF naskah > naskah HTML halaman DJP > PDF
    lampiran saja. Bila naskah HTML dipakai dan ada PDF lampiran, lampiran
    disimpan berdampingan sebagai '<nama>-lampiran.pdf'.
    """
    kode = r["kode"]
    subdir = U.SLUG_KATEGORI.get(d["kategori"][0], d["kategori"][0].lower())
    dasar = U.TUJUAN / subdir / U.nama_berkas(kode, r.get("tahun", ""))
    dasar.parent.mkdir(parents=True, exist_ok=True)
    detail = cari(d, jeda)
    if not detail:
        u = cari_bpk(d)
        if not u:
            return "tidak ada di pajak.go.id maupun JDIH BPK"
        if dry:
            return f"BPK {u}"
        p = dasar.with_suffix(".pdf")
        p.write_bytes(U.ambil(u, timeout=180))
        ok, hal, cat = U.verifikasi(p, kode)
        if not ok:
            p.unlink(missing_ok=True)
            return f"PDF BPK tidak sah: {cat}"
        isi = p.read_bytes()
        r.update({
            "sumber_ln": "" if r.get("sumber_ln", "").startswith(("tanpa slug", "semua sumber")) else r.get("sumber_ln", ""),
            "halaman": hal, "ukuran_byte": str(len(isi)),
            "sha256": hashlib.sha256(isi).hexdigest()[:16],
            "sumber_url": u, "file": str(p.relative_to(U.TUJUAN)),
            "catatan": "dari JDIH BPK (peraturan.bpk.go.id), JDIH Kemenkeu dan pajak.go.id tidak memuat"
                       + (f"; {cat}" if cat else ""),
        })
        return f"OK {r['file']} (BPK, PDF {hal} hal)"
    naskah = utama = lamp = None  # (url_detail, isi) / (url, bytes, hal)
    for h in detail:
        url_detail = urllib.parse.urljoin(DJP, h)
        html = U.teks(url_detail)
        naskah = naskah or ((url_detail, naskah_html(html), html) if naskah_html(html) else None)
        for u in pdf_lampiran(html):
            if dry:
                lamp = lamp or (u, b"", "")
                continue
            tmp = dasar.with_suffix(".tmp.pdf")
            tmp.write_bytes(U.ambil(u))
            ok, hal, cat = U.verifikasi(tmp, kode)
            isi = tmp.read_bytes()
            lampiran = ok and adalah_lampiran(u, hal_pertama(tmp))
            tmp.unlink(missing_ok=True)
            if not ok or cat == "abstrak":
                continue
            if lampiran:
                lamp = lamp or (u, isi, hal)
            else:
                utama = utama or (u, isi, hal, url_detail, html)
        if utama:
            break
    if dry:
        return f"naskah_html={bool(naskah)} pdf={lamp[0] if lamp else None}"
    if utama:
        u, isi, hal, url_detail, html = utama
        rel, src, cat = f"{dasar.relative_to(U.TUJUAN)}.pdf", u, ""
        dasar.with_suffix(".pdf").write_bytes(isi)
    elif naskah:
        url_detail, isi_html, html = naskah
        rel, src, hal = f"{dasar.relative_to(U.TUJUAN)}.htm", url_detail, ""
        simpan_naskah(dasar.with_suffix(".htm"), kode, url_detail, isi_html)
        isi = dasar.with_suffix(".htm").read_bytes()
        cat = "naskah HTML resmi DJP"
        if lamp:
            Path(f"{dasar}-lampiran.pdf").write_bytes(lamp[1])
            cat += f"; lampiran PDF {lamp[2]} hal: {dasar.name}-lampiran.pdf ({lamp[0]})"
    elif lamp:
        u, isi, hal = lamp
        url_detail, html = urllib.parse.urljoin(DJP, detail[0]), ""
        rel, src, cat = f"{dasar.relative_to(U.TUJUAN)}.pdf", u, "hanya lampiran, batang tubuh tidak ada di DJP"
        dasar.with_suffix(".pdf").write_bytes(isi)
    else:
        return f"halaman DJP ada, tanpa naskah maupun PDF ({len(detail)} halaman)"
    # PDF lampiran lama dari versi skrip sebelumnya bisa tertinggal
    if rel.endswith(".htm"):
        dasar.with_suffix(".pdf").unlink(missing_ok=True)
    tgl = re.search(r'field-tanggal-peraturan.*?datetime="(\d{4}-\d{2}-\d{2})', html, re.S)
    r.update({
        "tanggal_penetapan": r.get("tanggal_penetapan") or (tgl.group(1) if tgl else ""),
        "sumber_ln": "" if r.get("sumber_ln", "").startswith(("tanpa slug", "semua sumber")) else r.get("sumber_ln", ""),
        "halaman": hal, "ukuran_byte": str(len(isi)),
        "sha256": hashlib.sha256(isi).hexdigest()[:16],
        "sumber_url": src, "file": rel,
        "catatan": f"dari pajak.go.id (DJP), JDIH tidak memuat; {cat or 'PDF naskah'}; halaman {url_detail}",
    })
    return f"OK {rel} ({cat or f'PDF {hal} hal'})"


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--batas", type=int)
    ap.add_argument("--jeda", type=float, default=1.0)
    ap.add_argument("--dry-run", action="store_true")
    ap.add_argument("--ulang", action="store_true", help="proses ulang hasil pajak.go.id sebelumnya")
    a = ap.parse_args()

    index = U.baca_index()
    man = {d["uuid"]: d for d in json.loads(U.MANIFEST.read_text(encoding="utf-8"))["dokumen"]}
    sisa = [r for r in index.values()
            if r.get("file") in ("", "tanpa-pdf")
            or (a.ulang and r.get("catatan", "").startswith("dari pajak.go.id"))]
    if a.batas:
        sisa = sisa[: a.batas]
    dapat = 0
    for i, r in enumerate(sisa, 1):
        d = man.get(r["uuid"]) or {"kode": r["kode"], "judul": r["judul"], "tahun": r["tahun"],
                                    "kategori": r["kategori"].split(";")}
        try:
            log = proses(d, r, a.jeda, a.dry_run)
            if log.startswith("OK"):
                dapat += 1
                U.tulis_index(index)
        except Exception as e:  # noqa: BLE001
            log = f"GAGAL {e}"
        print(f"[{i}/{len(sisa)}] {r['kode']}: {log}", flush=True)
    print(f"\nterisi dari DJP: {dapat} dari {len(sisa)}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
