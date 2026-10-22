"""Uji cocok() di lengkapi-slug.py. Jalankan: python3 scripts/peraturan/test_lengkapi_slug.py"""
import importlib.util
from pathlib import Path

_s = importlib.util.spec_from_file_location("ls", Path(__file__).parent / "lengkapi-slug.py")
ls = importlib.util.module_from_spec(_s)
_s.loader.exec_module(ls)


def test_nol_di_depan_nomor():
    # Primbon menulis PER-01/PJ/2025, judul JDIH menulis PER-1/PJ/2025
    assert ls.cocok("PER-1/PJ/2025 - Petunjuk Teknis", "PER-01/PJ/2025", "2025")


def test_nomor_tetap_utuh():
    assert ls.cocok("PER-11/PJ/2025 - x", "PER-11/PJ/2025", "2025")
    # nomor 1 tidak boleh cocok dengan judul bernomor 11
    assert not ls.cocok("PER-11/PJ/2025 - x", "PER-1/PJ/2025", "2025")


if __name__ == "__main__":
    for n, f in list(globals().items()):
        if n.startswith("test_"):
            f()
            print("ok", n)
