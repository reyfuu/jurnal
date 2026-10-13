<?php

namespace App\Support;

use Symfony\Component\HttpFoundation\StreamedResponse;

/** CSV yang dibuka benar oleh Excel Indonesia: BOM UTF-8, pemisah ";", aman dari formula injection. */
final class Csv
{
    /**
     * @param  array<int,array<int,mixed>>  $preamble  baris keterangan di atas tabel
     * @param  iterable<int,array<int,mixed>>  $rows
     */
    public static function download(string $filename, array $preamble, array $header, iterable $rows): StreamedResponse
    {
        return response()->streamDownload(function () use ($preamble, $header, $rows) {
            $fh = fopen('php://output', 'w');
            fwrite($fh, "\xEF\xBB\xBF");
            foreach ([...$preamble, $header] as $line) {
                fputcsv($fh, array_map(self::safe(...), $line), ';');
            }
            foreach ($rows as $line) {
                fputcsv($fh, array_map(self::safe(...), $line), ';');
            }
            fclose($fh);
        }, $filename, ['Content-Type' => 'text/csv; charset=UTF-8']);
    }

    /** Sel yang diawali = + - @ diberi petik supaya Excel tidak menjalankannya sebagai formula. */
    public static function safe(mixed $value): mixed
    {
        return is_string($value) && $value !== '' && in_array($value[0], ['=', '+', '-', '@', "\t", "\r"], true) ? "'".$value : $value;
    }
}
