<?php

namespace App\Http\Controllers;

use App\Models\Client;
use App\Models\Journal;
use App\Services\Reporting\ReportService;
use App\Support\Csv;
use Illuminate\Http\Request;

class ReportController extends Controller
{
    public function __construct(private ReportService $reports) {}

    public function trialBalance(Client $client, Request $request)
    {
        [$period, $periods] = $this->period($client, $request);

        return view('reports.trial_balance', [
            'client' => $client, 'period' => $period, 'periods' => $periods,
            'report' => $this->reports->trialBalance($client->id, $period),
        ]);
    }

    public function incomeStatement(Client $client, Request $request)
    {
        [$period, $periods] = $this->period($client, $request);

        return view('reports.income_statement', [
            'client' => $client, 'period' => $period, 'periods' => $periods,
            'report' => $this->reports->incomeStatement($client->id, $period),
        ]);
    }

    public function balanceSheet(Client $client, Request $request)
    {
        [$period, $periods] = $this->period($client, $request);

        return view('reports.balance_sheet', [
            'client' => $client, 'period' => $period, 'periods' => $periods,
            'report' => $this->reports->balanceSheet($client->id, $period),
        ]);
    }

    public function cashFlow(Client $client, Request $request)
    {
        [$period, $periods] = $this->period($client, $request);

        return view('reports.cash_flow', [
            'client' => $client, 'period' => $period, 'periods' => $periods,
            'report' => $this->reports->cashFlow($client->id, $period),
        ]);
    }

    /** Laporan → CSV (dibuka di Excel). PDF cukup lewat tombol Cetak + print CSS. */
    public function export(Client $client, string $jenis, Request $request)
    {
        [$period] = $this->period($client, $request);
        $line = fn (array $r) => [$r['account']->code, $r['account']->name, $r['value']];
        $total = fn (string $label, float $v) => ['', $label, $v];

        [$title, $header, $rows] = match ($jenis) {
            'neraca-saldo' => (function () use ($client, $period) {
                $r = $this->reports->trialBalance($client->id, $period);

                return ['Neraca Saldo', ['Kode', 'Akun', 'Debit', 'Kredit', 'Saldo'], [
                    ...array_map(fn ($x) => [$x['account']->code, $x['account']->name, $x['debit'], $x['credit'], $x['balance']], $r['rows']),
                    ['', 'TOTAL', $r['total_debit'], $r['total_credit'], ''],
                ]];
            })(),
            'laba-rugi' => (function () use ($client, $period, $line, $total) {
                $r = $this->reports->incomeStatement($client->id, $period);

                return ['Laba Rugi', ['Kode', 'Akun', 'Nilai'], [
                    ...array_map($line, $r['groups']['revenue']), $total('Total Pendapatan', $r['totals']['revenue']),
                    ...array_map($line, $r['groups']['cogs']), $total('Total HPP', $r['totals']['cogs']),
                    $total('LABA KOTOR', $r['gross_profit']),
                    ...array_map($line, $r['groups']['opex']), $total('Total Beban Operasional', $r['totals']['opex']),
                    $total('LABA OPERASIONAL', $r['operating_income']),
                    ...array_map($line, $r['groups']['other_income']), $total('Total Pendapatan Lain-lain', $r['totals']['other_income']),
                    ...array_map($line, $r['groups']['other']), $total('Total Beban Lain-lain', $r['totals']['other']),
                    $total('LABA (RUGI) BERSIH', $r['net_income']),
                ]];
            })(),
            'neraca' => (function () use ($client, $period, $line, $total) {
                $r = $this->reports->balanceSheet($client->id, $period);

                return ['Neraca', ['Kode', 'Akun', 'Nilai'], [
                    ...array_map($line, $r['groups']['asset']), $total('TOTAL AKTIVA', $r['total_active']),
                    ...array_map($line, $r['groups']['liability']), $total('Total Liabilitas', $r['totals']['liability']),
                    ...array_map($line, $r['groups']['equity']), $total('Laba ditahan & berjalan', $r['retained_earnings']),
                    $total('Total Ekuitas', $r['total_equity']), $total('TOTAL PASIVA', $r['total_passive']),
                ]];
            })(),
            'arus-kas' => (function () use ($client, $period, $line, $total) {
                $r = $this->reports->cashFlow($client->id, $period);

                return ['Arus Kas', ['Kode', 'Akun', 'Nilai'], [
                    ...array_map($line, $r['groups']['operating']), $total('Kas bersih operasi', $r['totals']['operating']),
                    ...array_map($line, $r['groups']['investing']), $total('Kas bersih investasi', $r['totals']['investing']),
                    ...array_map($line, $r['groups']['financing']), $total('Kas bersih pendanaan', $r['totals']['financing']),
                    $total('Kenaikan (penurunan) kas', $r['net']), $total('Kas awal', $r['opening']), $total('Kas akhir', $r['closing']),
                ]];
            })(),
            default => abort(404),
        };

        return Csv::download("{$jenis}-{$client->slug}-{$period}.csv", [[$title, $client->name, "Periode {$period}"]], $header, $rows);
    }

    /** @return array{0:string,1:array<int,string>} */
    private function period(Client $client, Request $request): array
    {
        $periods = Journal::periodsFor($client->id);
        $requested = $request->string('period')->toString();
        // Periode bebas diisi lewat query (mis. bookmark bulan lama) — validasi
        // bentuknya saja, tidak perlu ada jurnalnya (laporan kosong itu jawaban sah).
        $period = preg_match('/^\d{4}-\d{2}$/', $requested) === 1 ? $requested : $periods[0];

        return [$period, $periods];
    }
}
