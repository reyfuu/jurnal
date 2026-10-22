"""Uji parser unduh-djp.py. Jalankan: python3 scripts/peraturan/test_unduh_djp.py"""
import importlib.util
from pathlib import Path

_s = importlib.util.spec_from_file_location("djp", Path(__file__).parent / "unduh-djp.py")
djp = importlib.util.module_from_spec(_s)
_s.loader.exec_module(djp)

# potongan asli halaman /id/peraturan?title=... (21 Okt 2026)
HASIL = (
    '<div class="peraturan-content views-row row-1"><div class="views-field views-field-field-nomor-dokumen">'
    '<h3 class="field-content title-custom text-link"><a href="/id/peraturan/dokumen-tertentu-2" hreflang="id">'
    'PER-16/PJ/2021</a></h3></div><div class="views-field views-field-title"><span class="field-content">X</span></div>'
    '<div class="peraturan-content views-row row-2"><div class="views-field views-field-field-nomor-dokumen">'
    '<h3 class="field-content title-custom text-link"><a href="/id/peraturan/dokumen-tertentu-1" hreflang="id">'
    'PER - 16/PJ/2021</a></h3></div>'
)
DETAIL = (
    '<link rel="icon" href="/sites/default/files/djp.ico" type="image/vnd.microsoft.icon" />'
    '<a href="https://www.pajak.go.id/sites/default/files/lampiran/PER%2016_Salinan.pdf" '
    'type="application/pdf; length=599768">PER 16_Salinan.pdf</a>'
    '<a href="/sites/default/files/2021-07/Lampiran%20I.pdf" type="application/pdf">Lampiran I.pdf</a>'
)


def test_nomor_setara():
    assert djp.nomor_sama("PER-01/PJ/2025", "PER-1/PJ/2025")
    assert djp.nomor_sama("PER-16/PJ/2021", "PER - 16/PJ/2021")
    assert djp.nomor_sama("KMK Nomor 09/MK/EF.2/2026", "9/MK/EF.2/2026")
    assert not djp.nomor_sama("PER-1/PJ/2025", "PER-11/PJ/2025")
    assert not djp.nomor_sama("PER-16/PJ/2021", "PER-16/PJ/2016")
    assert not djp.nomor_sama("SE-15/PJ/2018", "PER-15/PJ/2018")


def test_parse_hasil():
    assert djp.parse_hasil(HASIL) == [
        ("PER-16/PJ/2021", "/id/peraturan/dokumen-tertentu-2"),
        ("PER - 16/PJ/2021", "/id/peraturan/dokumen-tertentu-1"),
    ]


def test_pdf_lampiran_utamakan_salinan():
    assert djp.pdf_lampiran(DETAIL) == [
        "https://www.pajak.go.id/sites/default/files/lampiran/PER%2016_Salinan.pdf",
        "https://www.pajak.go.id/sites/default/files/2021-07/Lampiran%20I.pdf",
    ]


def test_kata_kunci_judul():
    # judul Primbon kadang ditambah "tentang ..." atau singkatan: pakai beberapa kata awal
    k = djp.kata_kunci("Perubahan Keempat Atas PER-34/PJ/2010 tentang Bentuk Formulir Surat Pemberitahuan")
    assert k[0].startswith("Perubahan Keempat Atas")
    assert len(k) >= 2 and len(k[-1].split()) <= 5


def test_naskah_html_ambil_div_body_utuh():
    pasal = "<p>Menimbang : a. bahwa ...</p>" + "<p>Pasal 1 isi</p>" * 80
    html = ('<style>.field--name-field-body-dalam-html{border:2px}</style>'
            '<div class="clearfix text-formatted field field--name-field-body-dalam-html field__item">'
            f'<div><p>PERATURAN DIREKTUR JENDERAL PAJAK</p></div>{pasal}</div><div class="lain">x</div>')
    n = djp.naskah_html(html)
    assert n.startswith("<div><p>PERATURAN") and n.endswith("isi</p>") and "lain" not in n


def test_naskah_html_kosong_bila_tanpa_pasal():
    html = '<div class="field field--name-field-body-dalam-html"><p>Lihat lampiran.</p></div>'
    assert djp.naskah_html(html) == ""


def test_lampiran_dari_nama_atau_halaman_awal():
    assert djp.adalah_lampiran("https://x/files/Lampiran%20PER03PJ2022.pdf", "")
    assert djp.adalah_lampiran("https://x/files/PER8.pdf", "-6LAMPIRAN\nPERATURAN DIREKTUR")
    assert not djp.adalah_lampiran("https://x/files/lampiran/PER%2016_Salinan.pdf", "")
    assert not djp.adalah_lampiran("https://x/files/KEP185.pdf", "KEMENTERIAN KEUANGAN\nKEPUTUSAN")


def test_slug_bpk():
    assert djp.slug_bpk("PP Nomor 18 Tahun 2025") == "pp-no-18-tahun-2025"
    assert djp.slug_bpk("Perpres No. 68 Tahun 2025") == "perpres-no-68-tahun-2025"
    assert djp.slug_bpk("PER-16/PJ/2021") == ""


if __name__ == "__main__":
    for n, f in list(globals().items()):
        if n.startswith("test_"):
            f()
            print("ok", n)
