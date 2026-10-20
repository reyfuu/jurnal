"""Buat internal/finance/wilayah.json: provinsi -> kabupaten/kota -> kecamatan.

Sumber: Kepmendagri No. 300.2.2-2138 Tahun 2025 lewat
https://github.com/yonatanyl/KODE-WILAYAH-KEPMENDAGRI-2025 (CC BY 4.0).

Pakai: python3 scripts/gen_wilayah.py <KODE-WILAYAH-KEPMENDAGRI-2025.xlsx>
Hanya stdlib; nama provinsi dipetakan ke finance.ProvinsiOptions (profile.go).
"""
import json
import re
import sys
import xml.etree.ElementTree as ET
import zipfile

OUT = "internal/finance/wilayah.json"
NS = "{http://schemas.openxmlformats.org/spreadsheetml/2006/main}"
# Nama di Kepmendagri 2025 -> nama di ProvinsiOptions (yang sudah dipakai profil tersimpan).
ALIAS = {"Daerah Khusus Ibukota Jakarta": "DKI Jakarta", "Daerah Istimewa Yogyakarta": "DI Yogyakarta"}

profile = open("internal/finance/profile.go", encoding="utf-8").read()
blok = re.search(r"var ProvinsiOptions = \[\]string\{(.*?)\n\}", profile, re.S).group(1)
provinsi_app = re.findall(r'"([^"]+)"', blok)

z = zipfile.ZipFile(sys.argv[1])
shared = ["".join(t.text or "" for t in si.iter(NS + "t")) for si in ET.fromstring(z.read("xl/sharedStrings.xml")).iter(NS + "si")]

wilayah = {}  # provinsi -> {kota: set(kecamatan)}
header = True
for _, el in ET.iterparse(z.open("xl/worksheets/sheet1.xml")):
    if el.tag != NS + "row":
        continue
    row = {}
    for c in el.iter(NS + "c"):
        v = c.find(NS + "v")
        x = v.text if v is not None else ""
        row[re.sub(r"\d", "", c.get("r"))] = shared[int(x)] if c.get("t") == "s" else x
    el.clear()
    if header:  # A..I: desa, kecamatan (C,D), kabupaten (E,F), provinsi (G,H), tipe
        assert row.get("D") == "NAMA KECAMATAN" and row.get("F") == "NAMA KABUPATEN" and row.get("H") == "NAMA PROVINSI", row
        header = False
        continue
    prov = ALIAS.get(row["H"].strip(), row["H"].strip())
    wilayah.setdefault(prov, {}).setdefault(row["F"].strip(), set()).add(row["D"].strip())

asing = sorted(set(wilayah) - set(provinsi_app))
hilang = sorted(set(provinsi_app) - set(wilayah))
if asing or hilang:
    sys.exit(f"provinsi tidak cocok dengan ProvinsiOptions; tambahkan ke ALIAS. asing={asing} hilang={hilang}")

out = {p: [{"nama": k, "kecamatan": sorted(kec)} for k, kec in sorted(wilayah[p].items())] for p in provinsi_app}
n_kota = sum(len(v) for v in out.values())
n_kec = sum(len(k["kecamatan"]) for v in out.values() for k in v)
if not (len(out) == 38 and n_kota == 514 and 7200 <= n_kec <= 7400):
    sys.exit(f"jumlah tidak wajar: provinsi={len(out)} kota={n_kota} kecamatan={n_kec}")

with open(OUT, "w", encoding="utf-8") as f:
    json.dump(out, f, ensure_ascii=False, separators=(",", ":"))
print(f"{OUT}: {len(out)} provinsi, {n_kota} kabupaten/kota, {n_kec} kecamatan")
