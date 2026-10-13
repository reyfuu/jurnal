<?php

namespace App\Support;

/**
 * Aturan sandi yang dipakai bersama oleh perintah ganti sandi dan seeder,
 * supaya "cukup kuat" berarti hal yang sama di semua pintu masuk.
 *
 * Daftar blokir memeriksa KANDUNGAN, bukan kesamaan persis: "password1234"
 * panjangnya 12 karakter dan lolos pencocokan persis, padahal sama saja
 * buruknya dengan "password".
 */
class SandiLemah
{
    public const PANJANG_MINIMAL = 12;

    /** Potongan kata yang bikin sandi gampang ditebak, dicek sebagai substring. */
    private const KATA_TERLARANG = [
        'password', 'passw0rd', 'sandi', 'rahasia', 'qwerty', 'admin',
        '123456', 'abcdef', 'jurnal', 'skinku', 'letmein', 'welcome',
    ];

    /** Sandi bawaan yang pernah dipakai contoh/dokumentasi — untuk dicek lewat Hash::check. */
    public const BAWAAN = ['password123', 'password', 'admin', 'rahasia', '12345678', 'qwerty123'];

    /**
     * Alasan kenapa sandi ditolak, atau null kalau sandinya boleh dipakai.
     */
    public static function alasanTolak(?string $sandi): ?string
    {
        $sandi = (string) $sandi;

        if (mb_strlen($sandi) < self::PANJANG_MINIMAL) {
            return 'Sandi minimal '.self::PANJANG_MINIMAL.' karakter.';
        }

        $kecil = mb_strtolower($sandi);
        foreach (self::KATA_TERLARANG as $kata) {
            if (str_contains($kecil, $kata)) {
                return "Sandi mengandung kata yang gampang ditebak (\"{$kata}\"). Pilih yang lain.";
            }
        }

        if (ctype_digit($sandi)) {
            return 'Sandi tidak boleh angka semua.';
        }

        // Sandi seperti "aaaaaaaaaaaa" panjangnya cukup tapi ragamnya nyaris nol.
        if (count(array_unique(mb_str_split($kecil))) < 5) {
            return 'Sandi terlalu sedikit ragam karakternya.';
        }

        return null;
    }
}
