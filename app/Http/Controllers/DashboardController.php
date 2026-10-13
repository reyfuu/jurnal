<?php

namespace App\Http\Controllers;

use App\Models\Client;
use App\Models\Document;
use App\Models\Journal;
use App\Services\Ai\AiProviderFactory;
use App\Services\Reporting\ReportService;

class DashboardController extends Controller
{
    public function __construct(private ReportService $reports) {}

    public function index()
    {
        $user = auth()->user();
        // Anggota tidak melihat angka keuangan — langsung ke dokumen klien pertamanya.
        if (! $user->may('books.view')) {
            $first = Client::active()->visibleTo($user)->orderBy('name')->first();

            return $first ? redirect()->route('documents.index', $first) : view('dashboard-empty');
        }

        $period = now()->format('Y-m');

        $clients = Client::active()->visibleTo($user)->orderBy('type')->orderBy('name')->get()
            ->map(fn (Client $client) => [
                'client' => $client,
                'snapshot' => $this->reports->snapshot($client->id, $period),
                'pending' => $client->documents()
                    ->whereIn('status', [Document::STATUS_UPLOADED, Document::STATUS_EXTRACTED])
                    ->count(),
            ]);

        // Kalender pajak lintas klien: kewajiban masa lalu, jatuh tempo bulan ini.
        // ponytail: N kueri kecil per klien; satukan kalau klien sudah ratusan.
        $masa = now()->subMonthNoOverflow()->format('Y-m');
        $recap = app(\App\Services\Tax\TaxRecap::class);
        $taxDue = $clients->pluck('client')->filter->hasTaxProfile()
            ->flatMap(fn (Client $c) => array_map(fn ($o) => $o + ['client' => $c, 'period' => $masa], $recap->obligations($c, $masa)))
            ->reject(fn ($o) => in_array($o['status'], ['lapor', 'nihil'], true))
            ->sortBy('pay_due')->values();

        return view('dashboard', [
            'period' => $period,
            'rows' => $clients,
            'aiReady' => AiProviderFactory::configured(),
            'taxDue' => $taxDue,
            'totals' => [
                'clients' => $clients->count(),
                'documents' => Document::whereIn('client_id', $clients->pluck('client.id'))->count(),
                'journals' => Journal::posted()->whereIn('client_id', $clients->pluck('client.id'))->count(),
                'pending' => $clients->sum('pending'),
            ],
        ]);
    }
}
