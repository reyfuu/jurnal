<?php

namespace App\Support;

use RuntimeException;
use SimpleXMLElement;
use Symfony\Component\HttpFoundation\BinaryFileResponse;
use ZipArchive;

/**
 * Excel .xlsx minimal tanpa library: .xlsx = zip berisi XML (Office Open XML).
 * Tulis: satu sheet, sel teks inline (bukan formula → aman dari formula injection).
 * Baca: sheet pertama, nilai sebagai teks; juga menerima .csv (pemisah ; atau ,).
 *
 * ponytail: cukup untuk tabel datar (COA, daftar). Pakai phpoffice/phpspreadsheet
 * kalau butuh banyak sheet, format angka, rumus, atau merge cell.
 */
final class Xlsx
{
    /** Batas ukuran XML sheet setelah diekstrak — pagar zip bomb. */
    private const MAX_XML_BYTES = 20 * 1024 * 1024;

    /**
     * @param  array<int,string>  $header
     * @param  iterable<int,array<int,mixed>>  $rows
     */
    public static function download(string $filename, string $sheet, array $header, iterable $rows): BinaryFileResponse
    {
        $path = tempnam(sys_get_temp_dir(), 'xlsx');
        self::write($path, $sheet, $header, $rows);

        return response()->download($path, $filename, [
            'Content-Type' => 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
        ])->deleteFileAfterSend();
    }

    /**
     * @param  array<int,string>  $header
     * @param  iterable<int,array<int,mixed>>  $rows
     */
    public static function write(string $path, string $sheet, array $header, iterable $rows): void
    {
        $esc = fn ($v) => htmlspecialchars((string) $v, ENT_XML1 | ENT_QUOTES, 'UTF-8');
        $cell = function ($value, int $col, int $row, bool $bold = false) use ($esc) {
            $ref = self::column($col).$row;
            $style = $bold ? ' s="1"' : '';
            if (is_int($value) || is_float($value)) {
                return "<c r=\"{$ref}\"{$style}><v>{$value}</v></c>";
            }

            return "<c r=\"{$ref}\" t=\"inlineStr\"{$style}><is><t xml:space=\"preserve\">{$esc($value)}</t></is></c>";
        };

        $xml = '';
        $r = 1;
        foreach ([$header, ...$rows] as $line) {
            $xml .= "<row r=\"{$r}\">";
            foreach (array_values($line) as $i => $value) {
                if ($value !== null && $value !== '') {
                    $xml .= $cell($value, $i, $r, $r === 1);
                }
            }
            $xml .= '</row>';
            $r++;
        }
        $cols = implode('', array_map(fn ($i) => '<col min="'.($i + 1).'" max="'.($i + 1).'" width="'.($i === 1 ? 42 : 16).'" customWidth="1"/>', array_keys($header)));

        $zip = new ZipArchive;
        if ($zip->open($path, ZipArchive::CREATE | ZipArchive::OVERWRITE) !== true) {
            throw new RuntimeException('Tidak bisa membuat file Excel.');
        }
        $ns = 'http://schemas.openxmlformats.org';
        $zip->addFromString('[Content_Types].xml', '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>'
            ."<Types xmlns=\"{$ns}/package/2006/content-types\">"
            .'<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>'
            .'<Default Extension="xml" ContentType="application/xml"/>'
            .'<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>'
            .'<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>'
            .'<Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>'
            .'</Types>');
        $zip->addFromString('_rels/.rels', '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>'
            ."<Relationships xmlns=\"{$ns}/package/2006/relationships\">"
            ."<Relationship Id=\"rId1\" Type=\"{$ns}/officeDocument/2006/relationships/officeDocument\" Target=\"xl/workbook.xml\"/>"
            .'</Relationships>');
        $zip->addFromString('xl/workbook.xml', '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>'
            ."<workbook xmlns=\"{$ns}/spreadsheetml/2006/main\" xmlns:r=\"{$ns}/officeDocument/2006/relationships\">"
            .'<sheets><sheet name="'.$esc(mb_substr($sheet, 0, 31)).'" sheetId="1" r:id="rId1"/></sheets></workbook>');
        $zip->addFromString('xl/_rels/workbook.xml.rels', '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>'
            ."<Relationships xmlns=\"{$ns}/package/2006/relationships\">"
            ."<Relationship Id=\"rId1\" Type=\"{$ns}/officeDocument/2006/relationships/worksheet\" Target=\"worksheets/sheet1.xml\"/>"
            ."<Relationship Id=\"rId2\" Type=\"{$ns}/officeDocument/2006/relationships/styles\" Target=\"styles.xml\"/>"
            .'</Relationships>');
        $zip->addFromString('xl/styles.xml', '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>'
            ."<styleSheet xmlns=\"{$ns}/spreadsheetml/2006/main\">"
            .'<fonts count="2"><font><sz val="11"/><name val="Calibri"/></font><font><b/><sz val="11"/><name val="Calibri"/></font></fonts>'
            .'<fills count="2"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill></fills>'
            .'<borders count="1"><border><left/><right/><top/><bottom/><diagonal/></border></borders>'
            .'<cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>'
            .'<cellXfs count="2"><xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/><xf numFmtId="0" fontId="1" fillId="0" borderId="0" xfId="0" applyFont="1"/></cellXfs>'
            .'<cellStyles count="1"><cellStyle name="Normal" xfId="0" builtinId="0"/></cellStyles></styleSheet>');
        $zip->addFromString('xl/worksheets/sheet1.xml', '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>'
            ."<worksheet xmlns=\"{$ns}/spreadsheetml/2006/main\">"
            .'<sheetViews><sheetView workbookViewId="0"><pane ySplit="1" topLeftCell="A2" state="frozen"/></sheetView></sheetViews>'
            ."<cols>{$cols}</cols><sheetData>{$xml}</sheetData></worksheet>");
        $zip->close();
    }

    /**
     * Baris-baris sheet pertama (.xlsx) atau file .csv, semua sel sebagai teks ter-trim.
     *
     * @return array<int,array<int,string>>
     *
     * @throws RuntimeException file bukan xlsx/csv yang bisa dibaca
     */
    public static function read(string $path, string $extension): array
    {
        return strtolower($extension) === 'csv' || strtolower($extension) === 'txt'
            ? self::readCsv($path)
            : self::readXlsx($path);
    }

    /** @return array<int,array<int,string>> */
    private static function readCsv(string $path): array
    {
        $raw = (string) file_get_contents($path);
        $raw = preg_replace('/^\xEF\xBB\xBF/', '', $raw); // BOM dari Excel
        $first = strtok($raw, "\n") ?: '';
        $sep = substr_count($first, ';') >= substr_count($first, ',') ? ';' : ',';

        $rows = [];
        foreach (preg_split('/\r\n|\n|\r/', $raw) as $line) {
            if (trim($line) !== '') {
                $rows[] = array_map(fn ($v) => trim((string) $v), str_getcsv($line, $sep, '"', ''));
            }
        }

        return $rows;
    }

    /** @return array<int,array<int,string>> */
    private static function readXlsx(string $path): array
    {
        $zip = new ZipArchive;
        if ($zip->open($path, ZipArchive::RDONLY) !== true) {
            throw new RuntimeException('File bukan Excel .xlsx yang valid.');
        }

        try {
            $xml = function (string $name) use ($zip): ?SimpleXMLElement {
                $stat = $zip->statName($name);
                if ($stat === false) {
                    return null;
                }
                if ($stat['size'] > self::MAX_XML_BYTES) {
                    throw new RuntimeException('Isi file Excel terlalu besar.');
                }

                return simplexml_load_string((string) $zip->getFromName($name), SimpleXMLElement::class, LIBXML_NONET) ?: null;
            };

            // Sheet pertama sesuai urutan workbook, bukan asal sheet1.xml.
            $sheetPath = 'xl/worksheets/sheet1.xml';
            $workbook = $xml('xl/workbook.xml');
            $rels = $xml('xl/_rels/workbook.xml.rels');
            if ($workbook && $rels) {
                $rid = (string) ($workbook->sheets->sheet[0]?->attributes('http://schemas.openxmlformats.org/officeDocument/2006/relationships')['id'] ?? '');
                foreach ($rels->Relationship as $rel) {
                    if ((string) $rel['Id'] === $rid) {
                        $target = ltrim((string) $rel['Target'], '/');
                        $sheetPath = str_starts_with($target, 'xl/') ? $target : 'xl/'.$target;
                    }
                }
            }

            $shared = [];
            foreach ($xml('xl/sharedStrings.xml')?->si ?? [] as $si) {
                // Teks kaya (<r><t>) dipecah per potongan — gabungkan semua <t>.
                $shared[] = isset($si->t) ? (string) $si->t : implode('', array_map('strval', $si->xpath('.//*[local-name()="t"]') ?: []));
            }

            $sheet = $xml($sheetPath) ?? throw new RuntimeException('Sheet pertama tidak ditemukan di file Excel.');
            $rows = [];
            foreach ($sheet->sheetData->row ?? [] as $row) {
                $line = [];
                foreach ($row->c as $c) {
                    $col = self::columnIndex(preg_replace('/\d+/', '', (string) $c['r']));
                    $value = match ((string) $c['t']) {
                        's' => $shared[(int) $c->v] ?? '',
                        'inlineStr' => implode('', array_map('strval', $c->is->xpath('.//*[local-name()="t"]') ?: [])),
                        'b' => (string) $c->v === '1' ? '1' : '0',
                        default => (string) $c->v,
                    };
                    $line[$col] = trim($value);
                }
                if ($line !== [] && implode('', $line) !== '') {
                    $max = max(array_keys($line));
                    $rows[] = array_map(fn ($i) => $line[$i] ?? '', range(0, $max));
                }
            }

            return $rows;
        } finally {
            $zip->close();
        }
    }

    /** 0 → A, 25 → Z, 26 → AA. */
    private static function column(int $index): string
    {
        $name = '';
        for ($i = $index + 1; $i > 0; $i = intdiv($i - 1, 26)) {
            $name = chr(65 + ($i - 1) % 26).$name;
        }

        return $name;
    }

    /** A → 0, AA → 26. */
    private static function columnIndex(string $letters): int
    {
        $n = 0;
        foreach (str_split(strtoupper($letters)) as $ch) {
            $n = $n * 26 + (ord($ch) - 64);
        }

        return max(0, $n - 1);
    }
}
