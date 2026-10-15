<?php

namespace App\Http\Controllers;

use App\Models\AuditLog;
use App\Models\Client;
use App\Models\Journal;
use App\Models\TaxEntry;
use App\Models\TaxFiling;
use App\Models\TaxRate;
use App\Services\Accounting\ChartOfAccounts;
use App\Services\Tax\TaxRecap;
use App\Support\Audit;
use App\Support\Csv;
use App\Support\Rupiah;
use App\Support\TaxId;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Carbon;
use Illuminate\Support\Facades\DB;
use Illuminate\Validation\Rule;
use Symfony\Component\HttpFoundation\StreamedResponse;

/**
 * Perpajakan per klien (docs/perpajakan). Semua angka dari TaxRecap; manusia
 * hanya mengisi profil, tarif, dan status setor/lapor. Tidak ada yang dikirim
 * ke Coretax — ekspor CSV dibawa manusia.
 */
class TaxController extends Controller
{
    public function __construct(private TaxRecap $recap) {}

    public function index(Client $client, Request $request)
    {
        [$period, $periods] = $this->period($client, $request);
        $entries = $this->recap->entries($client, $period);

        return view('tax.index', [
            'client' => $client, 'period' => $period, 'periods' => $periods,
            'ppn' => $client->isPkpOn() ? $this->recap->ppn($client, $period, $entries) : null,
            'pph' => $this->recap->withholding($client, $period, $entries),
            'final' => $this->recap->finalUmkm($client, $period),
            'mismatch' => $this->recap->reconcile($client, $period, $entries),
            'obligations' => $this->recap->obligations($client, $period, $entries),
        ]);
    }

    public function storeFiling(Client $client, Request $request): RedirectResponse
    {
        $data = $request->validate([
            'tax_code' => ['required', Rule::in(array_keys(config('pajak.codes')))],
            'period' => ['required', 'regex:/^\d{4}-\d{2}$/'],
            'status' => ['required', Rule::in(array_keys(TaxFiling::STATUSES))],
            'paid_amount' => ['nullable', 'integer', 'min:0'],
            'paid_at' => ['nullable', 'date'],
            'ntpn' => ['nullable', 'string', 'max:16'],
            'filed_at' => ['nullable', 'date'],
            'receipt_number' => ['nullable', 'string', 'max:60'],
            'note' => ['nullable', 'string', 'max:500'],
        ]);
        if ($data['status'] === 'setor' && blank($data['ntpn'])) {
            return back()->withErrors(['ntpn' => 'Status "Sudah setor" wajib menyertakan NTPN dari bukti setor.']);
        }

        $filing = TaxFiling::firstOrNew(['client_id' => $client->id, 'tax_code' => $data['tax_code'], 'period' => $data['period']]);
        $before = $filing->exists ? $filing->only(array_keys($data)) : null;
        $filing->fill($data + ['updated_by' => $request->user()->id])->save();
        Audit::log('filing.update', 'tax_filing', $filing->id, $client->id, $before, $filing->only(array_keys($data)));

        return back()->with('status', config("pajak.codes.{$data['tax_code']}.name")." masa {$data['period']}: ".TaxFiling::STATUSES[$data['status']].'.');
    }

    /**
     * Draft jurnal setoran/akrual dari rekap → form jurnal manual terisi.
     * Tidak ada yang diposting di sini; manusia memeriksa lalu menekan Posting.
     */
    public function draft(Client $client, string $jenis, Request $request): RedirectResponse
    {
        [$period] = $this->period($client, $request);
        $akun = fn (string $code) => $client->accounts()->where('code', $code)->value('id');
        $date = Carbon::createFromFormat('Y-m', $period)->endOfMonth()->toDateString();
        $line = fn (string $code, int $debit, int $credit, string $memo) => ['account_id' => $akun($code), 'debit' => $debit ?: null, 'credit' => $credit ?: null, 'memo' => $memo];

        [$description, $lines] = match ($jenis) {
            'final-akrual' => (function () use ($client, $period, $line) {
                $tax = (int) ($this->recap->finalUmkm($client, $period)['tax'] ?? 0);
                $beban = $client->accounts()->whereIn('code', ChartOfAccounts::TAX_EXPENSE_CODES)->orderByDesc('code')->value('code') ?? '7103';

                return ["Akrual PPh Final UMKM masa {$period}", [$line($beban, $tax, 0, 'PPh Final 0,5% omzet'), $line('2110', 0, $tax, 'Utang PPh Final')]];
            })(),
            'final-setor' => (function () use ($client, $period, $line) {
                $tax = (int) ($this->recap->finalUmkm($client, $period)['tax'] ?? 0);

                return ["Setor PPh Final UMKM masa {$period}", [$line('2110', $tax, 0, 'Pelunasan utang PPh Final'), $line('1102', 0, $tax, 'Setor via bank (isi NTPN di referensi)')]];
            })(),
            'ppn-setor' => (function () use ($client, $period, $line) {
                $p = $this->recap->ppn($client, $period);
                abort_if($p['net'] <= 0, 404);

                return ["Setor PPN kurang bayar masa {$period}", [
                    $line('2106', $p['keluaran'], 0, 'Tutup PPN Keluaran'),
                    $line('1402', 0, $p['masukan'], 'Kreditkan PPN Masukan'),
                    $line('1102', 0, $p['net'], 'Setor kurang bayar (isi NTPN di referensi)'),
                ]];
            })(),
            'pph-setor' => (function () use ($client, $period, $line, $request) {
                $code = $request->string('kode')->toString();
                $g = $this->recap->withholding($client, $period)['groups']->first(fn ($g) => $g['tax_code'] === $code && $g['direction'] === 'out');
                abort_unless($g, 404);
                $account = app(\App\Services\Tax\TaxCalculator::class)->accountFor($client->accounts()->active()->get()->keyBy('code'), $code, 'out');

                return ['Setor '.config("pajak.codes.{$code}.name")." masa {$period}", [
                    $line($account->code, $g['amount'], 0, 'Pelunasan utang '.config("pajak.codes.{$code}.name")),
                    $line('1102', 0, $g['amount'], 'Setor via bank (isi NTPN di referensi)'),
                ]];
            })(),
            default => abort(404),
        };

        return redirect()->route('journals.create', $client)
            ->withInput(['date' => $date, 'description' => $description, 'type' => 'general', 'lines' => $lines])
            ->with('status', 'Draft jurnal terisi dari rekap pajak. Periksa akun bank & tanggal, lalu posting.');
    }

    public function profile(Client $client)
    {
        return view('tax.profile', ['client' => $client]);
    }

    public function updateProfile(Client $client, Request $request): RedirectResponse
    {
        $data = $request->validate([
            'taxpayer_type' => ['nullable', Rule::in(array_keys(Client::TAXPAYER_TYPES))],
            'tax_id' => ['nullable', 'string', 'max:30'],
            'is_pkp' => ['required', 'in:0,1'],
            'pkp_since' => ['nullable', 'date'],
            'pph_regime' => ['required', Rule::in(array_keys(Client::REGIMES))],
        ]);

        [$digits, $error] = TaxId::normalize($data['tax_id'] ?? null);
        if ($error) {
            return back()->withInput()->withErrors(['tax_id' => $error]);
        }
        if ($data['pph_regime'] === 'final_umkm' && blank($data['taxpayer_type'])) {
            return back()->withInput()->withErrors(['taxpayer_type' => 'Rezim PPh Final UMKM butuh jenis wajib pajak (orang pribadi/badan) — batas omzet bebas berbeda.']);
        }

        $snapshot = fn () => [
            'taxpayer_type' => $client->taxpayer_type, 'tax_id' => TaxId::mask($client->tax_id),
            'is_pkp' => $client->is_pkp, 'pkp_since' => $client->pkp_since?->toDateString(), 'pph_regime' => $client->pph_regime,
        ];
        $before = $snapshot();
        $client->update(['tax_id' => $digits, 'is_pkp' => (bool) $data['is_pkp']] + $data);
        Audit::log('profile.update', 'client', $client->id, $client->id, $before, $snapshot());

        return redirect()->route('tax.index', $client)->with('status', 'Profil pajak disimpan.');
    }

    /** CSV untuk konsultan / input Coretax manual. Pemisah ";" + BOM supaya Excel Indonesia membacanya benar. */
    public function export(Client $client, string $jenis, Request $request): StreamedResponse
    {
        abort_unless(in_array($jenis, ['ppn', 'pph', 'final'], true), 404);
        [$period] = $this->period($client, $request);
        $entries = $this->recap->entries($client, $period);

        $header = ['Tanggal', 'Jenis', 'Arah', 'Lawan transaksi', 'NPWP lawan', 'No. faktur/bukpot', 'Dasar (Rp)', 'Pajak (Rp)', 'Hitung sistem (Rp)', 'Status', 'Jurnal'];
        $rows = match ($jenis) {
            'ppn' => $this->recap->ppn($client, $period, $entries)['rows'],
            'pph' => $this->recap->withholding($client, $period, $entries)['rows'],
            'final' => collect(),
        };
        $lines = $rows->map(fn (TaxEntry $e) => [
            Carbon::parse($e->journal_date)->toDateString(), $e->codeName(),
            $e->direction === 'out' ? ($e->withheld ? 'Dipotong (wajib setor)' : 'Keluaran/dipungut') : ($e->withheld ? 'Dipotong pihak lain' : 'Masukan'),
            $e->counterparty_name, $e->counterparty_tax_id, $e->document_number,
            $e->base, $e->amount, $e->computed_amount, $e->match_status, '#'.$e->journal_id,
        ]);

        if ($jenis === 'final' && ($f = $this->recap->finalUmkm($client, $period))) {
            $header = ['Masa', 'Omzet bruto (Rp)', 'Omzet tahun berjalan sebelumnya (Rp)', 'Bagian bebas (Rp)', 'Dasar pengenaan (Rp)', 'Tarif', 'PPh Final (Rp)'];
            $lines = collect([[$period, $f['omzet'], $f['omzet_ytd_before'], $f['exempt'], $f['taxable'], $f['rate']?->label() ?? '—', $f['tax']]]);
        }

        Audit::log('export', 'tax_export', null, $client->id, null, ['jenis' => $jenis, 'period' => $period]);
        $name = sprintf('pajak-%s-%s-%s.csv', $client->slug, $jenis, $period);

        return Csv::download($name, [[config('pajak.disclaimer')], ["Klien: {$client->name}", "Masa: {$period}"]], $header, $lines);
    }

    // ── Admin: tarif & audit ─────────────────────────────────────────────────

    public function rates()
    {
        return view('tax.rates', [
            'rates' => TaxRate::orderBy('tax_code')->orderBy('applies_to')->orderByDesc('effective_from')->get(),
            'used' => TaxEntry::distinct()->pluck('tax_rate_id')->filter()->flip(),
        ]);
    }

    /** Tarif tidak pernah diedit — selalu versi baru yang menutup versi lama di hari sebelumnya. */
    public function storeRate(Request $request): RedirectResponse
    {
        $data = $request->validate([
            'tax_code' => ['required', Rule::in(array_keys(config('pajak.codes')))],
            'applies_to' => ['nullable', Rule::in(array_keys(Client::TAXPAYER_TYPES))],
            'rate_percent' => ['required', 'string', 'max:10'],
            'rate_no_tax_id_percent' => ['nullable', 'string', 'max:10'],
            'dpp_numerator' => ['required', 'integer', 'min:1', 'max:1000'],
            'dpp_denominator' => ['required', 'integer', 'min:1', 'max:1000'],
            'exempt_threshold' => ['nullable', 'integer', 'min:0'],
            'rounding' => ['required', 'in:floor,half_up'],
            'effective_from' => ['required', 'date'],
            'source' => ['required', 'string', 'max:190'],
        ]);
        $bp = fn (?string $pct) => blank($pct) ? null : (int) round((Rupiah::parse($pct) ?? 0) * 100);
        if (($rateBp = $bp($data['rate_percent'])) === null || $rateBp <= 0 || $rateBp > 10000) {
            return back()->withInput()->withErrors(['rate_percent' => 'Tarif harus antara 0 dan 100%.']);
        }

        $from = Carbon::parse($data['effective_from']);
        $sameKey = fn () => TaxRate::where('tax_code', $data['tax_code'])->where('applies_to', $data['applies_to'] ?? null);
        if ($sameKey()->whereDate('effective_from', '>=', $from)->exists()) {
            return back()->withInput()->withErrors(['effective_from' => 'Sudah ada versi tarif yang mulai di/setelah tanggal ini. Versi baru harus lebih baru.']);
        }

        $rate = DB::transaction(function () use ($data, $rateBp, $bp, $request, $from, $sameKey) {
            $sameKey()->whereNull('effective_to')->update(['effective_to' => $from->copy()->subDay()->toDateString()]);

            return TaxRate::create([
                'tax_code' => $data['tax_code'], 'applies_to' => $data['applies_to'] ?? null,
                'rate_bp' => $rateBp, 'rate_no_tax_id_bp' => $bp($data['rate_no_tax_id_percent'] ?? null),
                'dpp_numerator' => $data['dpp_numerator'], 'dpp_denominator' => $data['dpp_denominator'],
                'exempt_threshold' => $data['exempt_threshold'] ?? null, 'rounding' => $data['rounding'],
                'effective_from' => $from->toDateString(), 'source' => $data['source'], 'created_by' => $request->user()->id,
            ]);
        });
        Audit::log('rate.create', 'tax_rate', $rate->id, null, null, $rate->only(['tax_code', 'applies_to', 'rate_bp', 'effective_from']));

        return back()->with('status', 'Versi tarif baru disimpan. Versi sebelumnya ditutup otomatis.');
    }

    public function verifyRate(TaxRate $rate): RedirectResponse
    {
        $rate->update(['verified_at' => now()]);
        Audit::log('rate.verify', 'tax_rate', $rate->id, null, null, ['verified_at' => now()->toDateTimeString()]);

        return back()->with('status', 'Tarif ditandai sudah diverifikasi konsultan.');
    }

    public function audit(Request $request)
    {
        return view('tax.audit', [
            'logs' => AuditLog::with('user')->latest('id')
                ->when($request->integer('klien'), fn ($q, $id) => $q->where('client_id', $id))
                ->limit(200)->get(),
            'clients' => Client::pluck('name', 'id'),
        ]);
    }

    /** @return array{0:string,1:array<int,string>} */
    private function period(Client $client, Request $request): array
    {
        $periods = Journal::periodsFor($client->id);
        $requested = $request->string('period')->toString();

        return [preg_match('/^\d{4}-\d{2}$/', $requested) === 1 ? $requested : $periods[0], $periods];
    }
}
