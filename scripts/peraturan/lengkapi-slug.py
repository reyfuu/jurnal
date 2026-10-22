#!/usr/bin/env python3
"""Lengkapi `jdih_slug` yang kosong di manifest dengan menebak pola slug JDIH.

Sebagian dokumen di Primbon hanya menautkan Google Drive (yang ditolak pemilik)
atau tidak menautkan sumber apa pun. Untuk itu slug JDIH ditebak dari kodenya,
lalu divalidasi dengan membandingkan judul resmi di <title> halaman JDIH.

Pemakaian:
    python3 scripts/peraturan/lengkapi-slug.py            # semua yang kosong
    python3 scripts/peraturan/lengkapi-slug.py --batas 20
"""
from __future__ import annotations

import argparse
import json
import re
import time
import urllib.parse
import urllib.request
from pathlib import Path

MANIFEST = Path(__file__).parent / "manifest-primbon.json"
JDIH = "https://jdih.kemenkeu.go.id"


def teks(url: str, timeout: int = 30) -> str:
    req = urllib.request.Request(url, headers={"User-Agent": "Mozilla/5.0"})
    with urllib.request.urlopen(req, timeout=timeout) as r:
        return r.read().decode("utf-8", "replace")


def kandidat_slug(kode: str, tahun: str) -> list[str]:
    """Bangun kandidat slug dari kode peraturan.

    Pola yang terbukti dipakai JDIH:
      PMK 4 TAHUN 2026      -> pmk-4-tahun-2026
      196/PMK.03/2021       -> 196-pmk-03-2021
      PER-11/PJ/2025        -> per-11pj2025 atau per-11-pj-2025
      KMK 49/MK/EF.2/2026   -> kmk-49-mk-ef-2-2026
    """
    k = kode.strip()
    out: list[str] = []

    # PER-11/PJ/2025, PENG-6/PJ/2026, SE-xx/PJ/yyyy
    m = re.match(r"^(PER|PENG|SE)-0*(\d+)/([A-Z]+)/(\d{4})$", k, re.I)
    if m:
        pre, no, unit, thn = m.group(1).lower(), m.group(2), m.group(3).lower(), m.group(4)
        out += [f"{pre}-{no}{unit}{thn}", f"{pre}-{no}-{unit}-{thn}",
                f"{pre}-0{no}-{unit}-{thn}" if len(no) == 1 else f"{pre}-{no}-{unit}-{thn}"]

    # 196/PMK.03/2021
    m = re.match(r"^(?:PMK\s*(?:Nomor|No\.?)?\s*)?(\d+)/([A-Z]+)\.(\d+)/(\d{4})$", k, re.I)
    if m:
        no, jenis, unit, thn = m.group(1), m.group(2).lower(), m.group(3), m.group(4)
        out += [f"{no}-{jenis}-{unit}-{thn}", f"{jenis}-{no}-tahun-{thn}"]

    # PMK/PP/PERPRES/UU Nomor N Tahun YYYY
    m = re.match(r"^(PMK|PP|PERPRES|UU|KMK|PERPPU|SEMA)\b[^0-9]*(\d+)\b.*?(\d{4})\s*$", k, re.I)
    if m:
        jenis, no, thn = m.group(1).lower(), m.group(2), m.group(3)
        out += [f"{jenis}-{no}-tahun-{thn}", f"{jenis}-{no}-{thn}"]
        # PMK sebelum 2023 memakai bentuk lama "62-pmk-03-2022" dengan kode unit
        # yang berbeda-beda per dokumen, jadi beberapa unit lazim dicoba.
        if jenis == "pmk" and thn.isdigit() and int(thn) <= 2023:
            out += [f"{no}-pmk-{u}-{thn}" for u in ("03", "010", "04", "05", "02", "06", "01", "011")]
        if jenis == "kmk" and thn.isdigit() and int(thn) <= 2023:
            out += [f"{no}-kmk-{u}-{thn}" for u in ("03", "010", "04", "02")]

    # Peraturan Presiden / Peraturan Pemerintah ditulis panjang
    m = re.match(r"^Peraturan\s+(Presiden|Pemerintah)\s+(?:Nomor|No\.?)\s*(\d+)\s+Tahun\s+(\d{4})", k, re.I)
    if m:
        jenis = "perpres" if m.group(1).lower() == "presiden" else "pp"
        out += [f"{jenis}-{m.group(2)}-tahun-{m.group(3)}"]

    # KMK 49/MK/EF.2/2026
    m = re.match(r"^KMK\s*(?:Nomor|No\.?)?\s*0*(\d+)/([A-Z]+)/([A-Z]+)\.?(\d*)/(\d{4})$", k, re.I)
    if m:
        no, a, b, c, thn = m.groups()
        inti = f"kmk-{no}-{a.lower()}-{b.lower()}" + (f"-{c}" if c else "")
        out += [f"{inti}-{thn}"]
        # Bentuk rapat yang terbukti di JDIH untuk Kurs KMK, prefiksnya tidak
        # konsisten: kmk-49mkef22026, nomor-15mkef22026, kmk-nomor-11mkef22026.
        rapat = f"{a.lower()}{b.lower()}{c}{thn}"
        for n in dict.fromkeys([no, no.zfill(2), no.zfill(3)]):
            out += [f"kmk-{n}{rapat}", f"nomor-{n}{rapat}", f"kmk-nomor-{n}{rapat}"]

    # fallback umum
    if tahun:
        datar = re.sub(r"[^a-z0-9]+", "-", k.lower()).strip("-")
        out.append(datar)

    # unik, pertahankan urutan
    seen, urut = set(), []
    for c in out:
        c = re.sub(r"-+", "-", c).strip("-")
        if c and c not in seen:
            seen.add(c)
            urut.append(c)
    return urut


def cocok(judul_jdih: str, kode: str, tahun: str) -> bool:
    """Judul halaman JDIH harus memuat nomor dan tahun yang sama.

    Bandingkan angka utuh tanpa nol di depan: 'PER-01' setara judul 'PER-1',
    dan nomor 1 tidak boleh cocok dengan '11' (perbandingan substring dulu)."""
    j = {a.lstrip("0") or "0" for a in re.findall(r"\d+", judul_jdih)}
    angka = re.findall(r"\d+", kode)
    no = (angka[0].lstrip("0") or "0") if angka else ""
    return bool(no) and no in j and (not tahun or tahun in j)


JENIS = {"KMK", "PMK", "PP", "UU", "PERPRES", "PERPPU", "SEMA", "NOMOR", "NO", "TAHUN"}


def token_kode(s: str) -> list[str]:
    """'KMK Nomor 019/MK/EF.2/2026' -> ['19','MK','EF','2','2026'].

    Kata jenis dan 'Nomor/Tahun' dibuang, nol di depan angka dibuang, supaya
    'Nomor 15/MK/EF.2/2026' dan 'KMK 15/MK/EF.2/2026' setara.
    """
    tok = re.findall(r"[A-Z]+|\d+", s.upper())
    return [t.lstrip("0") or "0" if t.isdigit() else t for t in tok if t not in JENIS]


def jenis_dari(s: str) -> str:
    """Jenis peraturan di awal teks, dinormalkan: 'Peraturan Presiden' -> PERPRES."""
    t = re.sub(r"\s+", " ", s.upper()).strip()
    t = t.replace("PERATURAN PRESIDEN", "PERPRES").replace("PERATURAN PEMERINTAH", "PP")
    t = t.replace("UNDANG-UNDANG", "UU")
    m = re.match(r"^(PMK|KMK|PP|UU|PERPRES|PERPPU|SEMA|PER|SE|KEP|PENG)\b", t)
    return m.group(1) if m else ""


def cocok_ketat(judul_jdih: str, kode: str, tahun: str) -> bool:
    """Hasil pencarian JDIH berisi banyak dokumen mirip (PMK 49 vs KMK 49),
    jadi syaratnya lebih ketat dari `cocok`: urutan token kode harus muncul
    utuh di bagian nomor judul (sebelum ' - ')."""
    nomor = judul_jdih.split(" - ", 1)[0]
    a, b = token_kode(kode), token_kode(nomor)
    if not a:
        return False
    # Jenis dibuang dari token, jadi wajib dicek terpisah: tanpa ini
    # 'SEMA Nomor 2 Tahun 2024' cocok dengan 'PMK 2 Tahun 2024'.
    jk, jj = jenis_dari(kode), jenis_dari(nomor)
    if jk and jj and jk != jj:
        return False
    for i in range(len(b) - len(a) + 1):
        if b[i:i + len(a)] == a:
            return True
    # 'PMK Nomor 2 Tahun 2010' vs judul '2/PMK.011/2010': nomor pertama dan tahun
    jenis = re.match(r"^\s*(PMK|KMK|PP|UU|PERPRES)\b", kode, re.I)
    angka_b = [t for t in b if t.isdigit()]
    if jenis and len(a) == 2 and angka_b and angka_b[0] == a[0] and a[1] in b \
            and jenis.group(1).upper() in nomor.upper():
        return True
    return False


def cari_jdih(kode: str, tahun: str, jeda: float) -> str | None:
    """Cari lewat halaman pencarian JDIH. Slug JDIH tidak konsisten
    (kmk-49mkef22026, nomor-15mkef22026, kmk-nomor-11mkef22026), jadi untuk
    sisa yang gagal ditebak, ini jalur yang andal."""
    q = re.sub(r"^(KMK|PMK)\s+Nomor\s+", "", kode.strip(), flags=re.I)
    try:
        html = teks(f"{JDIH}/search?q={urllib.parse.quote(q)}")
    except Exception:  # noqa: BLE001
        return None
    slugs = list(dict.fromkeys(re.findall(r"/dok/([a-zA-Z0-9-]+)", html)))
    for s in slugs[:12]:
        try:
            h = teks(f"{JDIH}/dok/{s}")
        except Exception:  # noqa: BLE001
            continue
        m = re.search(r"<title>([^<]*)", h)
        judul = (m.group(1) if m else "").split(" | ")[0]
        time.sleep(jeda)
        if judul and cocok_ketat(judul, kode, tahun):
            return s
    return None


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--batas", type=int)
    ap.add_argument("--jeda", type=float, default=0.4)
    a = ap.parse_args()

    man = json.loads(MANIFEST.read_text(encoding="utf-8"))
    kosong = [d for d in man["dokumen"] if not d["jdih_slug"]]
    if a.batas:
        kosong = kosong[: a.batas]

    dapat = 0
    for i, d in enumerate(kosong, 1):
        for c in kandidat_slug(d["kode"], d.get("tahun", "")):
            try:
                html = teks(f"{JDIH}/dok/{c}")
            except Exception:  # noqa: BLE001
                continue
            m = re.search(r"<title>([^<]*)", html)
            judul = m.group(1) if m else ""
            if not judul.strip() or "tidak ditemukan" in judul.lower():
                continue
            if cocok(judul, d["kode"], d.get("tahun", "")):
                d["jdih_slug"] = c
                dapat += 1
                print(f"[{i}/{len(kosong)}] {d['kode']} -> {c}")
                break
            time.sleep(a.jeda)
        else:
            s = cari_jdih(d["kode"], d.get("tahun", ""), a.jeda)
            if s:
                d["jdih_slug"] = s
                dapat += 1
                print(f"[{i}/{len(kosong)}] {d['kode']} -> {s} (pencarian)")
            else:
                print(f"[{i}/{len(kosong)}] {d['kode']}: tidak ketemu")
        time.sleep(a.jeda)

        if i % 20 == 0:
            man["tanpaSlugJdih"] = sum(1 for x in man["dokumen"] if not x["jdih_slug"])
            MANIFEST.write_text(json.dumps(man, ensure_ascii=False, indent=1), encoding="utf-8")

    man["tanpaSlugJdih"] = sum(1 for x in man["dokumen"] if not x["jdih_slug"])
    MANIFEST.write_text(json.dumps(man, ensure_ascii=False, indent=1), encoding="utf-8")
    print(f"\nslug baru: {dapat} dari {len(kosong)} | sisa tanpa slug: {man['tanpaSlugJdih']}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
