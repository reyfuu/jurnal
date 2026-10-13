<?php

namespace App\Support;

/** NPWP / NIK / NITKU: hanya digit. 15 digit = NPWP format lama. */
final class TaxId
{
    /** @return array{0:?string,1:?string} [digit atau null, alasan tolak] */
    public static function normalize(?string $value): array
    {
        $digits = preg_replace('/\D/', '', (string) $value) ?? '';
        if ($digits === '') {
            return [null, null];
        }
        if (! in_array(strlen($digits), [15, 16, 22], true)) {
            return [null, 'NPWP/NIK harus 16 digit (atau 15 digit format lama, 22 digit NITKU).'];
        }

        return [$digits, null];
    }

    public static function mask(?string $digits): string
    {
        if (blank($digits)) {
            return '—';
        }

        return str_repeat('•', max(0, strlen($digits) - 4)).substr($digits, -4);
    }
}
