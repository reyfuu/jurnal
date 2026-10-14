<?php

namespace App\Http\Controllers;

use App\Models\Client;
use App\Models\Journal;
use App\Models\TaxRate;
use App\Models\User;
use App\Services\Ai\AiException;
use App\Services\Ai\AiProviderFactory;
use App\Services\Tax\TaxRecap;
use App\Support\Rupiah;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

/**
 * Bubble "Tanya pajak": AI menjawab pertanyaan pajak SATU klien, berbekal profil
 * & rekap pajak klien itu saja. Hanya membaca — tidak bisa mencatat apa pun.
 *
 * ponytail: riwayat percakapan disimpan di browser (sessionStorage) dan dikirim
 * ulang per pertanyaan, bukan tabel. Buat tabel kalau percakapan perlu diaudit.
 */
class TaxChatController extends Controller
{
    private const MAX_HISTORY = 10;

    public function __construct(private TaxRecap $recap) {}

    public function ask(Client $client, Request $request): JsonResponse
    {
        $data = $request->validate([
            'question' => ['required', 'string', 'max:1000'],
            'history' => ['nullable', 'array', 'max:'.self::MAX_HISTORY],
            // Hanya user/assistant — riwayat dari browser tidak boleh menyelipkan pesan system.
            'history.*.role' => ['required', 'in:user,assistant'],
            'history.*.content' => ['required', 'string', 'max:4000'],
        ]);

        $messages = [
            ['role' => 'system', 'content' => $this->context($client, $request->user())],
            ...array_map(fn ($m) => ['role' => $m['role'], 'content' => $m['content']], $data['history'] ?? []),
            ['role' => 'user', 'content' => $data['question']],
        ];

        try {
            $turn = AiProviderFactory::make()->chat($messages, ['max_tokens' => 1500]);
        } catch (AiException $e) {
            return response()->json(['error' => $e->getMessage()], 503);
        }

        return response()->json(['answer' => trim($turn->text)]);
    }

    /** Prompt system: aturan main + data pajak klien ini. Angka buku hanya untuk yang boleh melihat buku. */
    private function context(Client $client, User $user): string
    {
        $profil = $client->hasTaxProfile()
            ? sprintf('%s · %s · rezim PPh %s', $client->taxpayer_type ? Client::TAXPAYER_TYPES[$client->taxpayer_type] : 'jenis WP belum diisi',
                $client->isPkpOn() ? 'PKP'.($client->pkp_since ? ' sejak '.$client->pkp_since->format('d/m/Y') : '') : 'bukan PKP',
                Client::REGIMES[$client->pph_regime] ?? $client->pph_regime)
            : 'Profil pajak BELUM diisi — sarankan admin/akuntan mengisinya di menu Profil pajak.';

        $lines = [
            'Kamu asisten pajak di aplikasi pembukuan Jurnal AI. Kamu membantu SATU perusahaan saja:',
            "- Perusahaan: {$client->name} (jenis usaha {$client->businessTypeLabel()})",
            "- Profil pajak: {$profil}",
            '- NPWP: '.($client->tax_id ? 'sudah tercatat' : 'belum tercatat'),
            '- Hari ini: '.now()->translatedFormat('l, j F Y'),
            '',
            'Tarif yang tercatat di sistem:',
            ...$this->rates(),
        ];

        if ($user->may('books.view')) {
            foreach ([now()->format('Y-m'), now()->subMonthNoOverflow()->format('Y-m')] as $period) {
                array_push($lines, '', 'Masa '.Journal::periodLabel($period).':', ...$this->period($client, $period));
            }
        } else {
            $lines[] = '';
            $lines[] = 'Pengguna ini TIDAK punya akses ke angka buku. Jawab secara umum berdasarkan profil; jangan menyebut atau menebak angka transaksi perusahaan.';
        }

        array_push($lines, '',
            'Aturan menjawab:',
            '- Bahasa Indonesia yang sederhana, ringkas, praktis. Teks biasa tanpa markdown (boleh daftar dengan tanda "-").',
            '- Untuk angka perusahaan, pakai HANYA data di atas. Kalau datanya tidak ada, katakan terus terang — jangan mengarang.',
            '- Tarif berlabel "belum diverifikasi" wajib disebut belum diverifikasi konsultan.',
            '- Hanya bahas pajak & pembukuan perusahaan ini. Tolak sopan pertanyaan tentang perusahaan lain atau topik di luar itu.',
            '- Kamu tidak bisa mencatat, memposting, atau mengubah data. Arahkan ke menu yang tepat (Ringkasan pajak, Profil pajak, Jurnal).',
            '- Untuk keputusan penting (pembetulan SPT, sengketa, restitusi), sarankan konfirmasi ke konsultan pajak.',
        );

        return implode("\n", $lines);
    }

    /** @return array<int,string> */
    private function rates(): array
    {
        $today = now()->toDateString();
        $rates = TaxRate::where('effective_from', '<=', $today)
            ->where(fn ($q) => $q->whereNull('effective_to')->orWhere('effective_to', '>=', $today))
            ->orderBy('tax_code')->get();

        return $rates->isEmpty() ? ['- (belum ada tarif tercatat)'] : $rates->map(fn (TaxRate $r) => sprintf('- %s%s: %s%s · %s',
            config("pajak.codes.{$r->tax_code}.name", $r->tax_code),
            $r->applies_to ? ' ('.str_replace('_', ' ', $r->applies_to).')' : '',
            $r->label(),
            $r->exempt_threshold ? ', omzet bebas s/d Rp'.Rupiah::format($r->exempt_threshold).'/tahun' : '',
            $r->verified_at ? 'terverifikasi' : 'belum diverifikasi',
        ))->all();
    }

    /** @return array<int,string> */
    private function period(Client $client, string $period): array
    {
        $entries = $this->recap->entries($client, $period);
        $out = [];

        foreach ($this->recap->obligations($client, $period, $entries) as $o) {
            $out[] = sprintf('- Kewajiban %s: setor sebelum %s, lapor sebelum %s, status %s%s',
                $o['name'], date('d/m/Y', strtotime($o['pay_due'])), date('d/m/Y', strtotime($o['file_due'])),
                $o['status'], $o['overdue'] ? ' (TERLAMBAT)' : '');
        }
        if ($client->isPkpOn()) {
            $ppn = $this->recap->ppn($client, $period, $entries);
            $out[] = sprintf('- PPN keluaran Rp%s, PPN masukan Rp%s, selisih Rp%s (%s)', Rupiah::format($ppn['keluaran']), Rupiah::format($ppn['masukan']),
                Rupiah::format(abs($ppn['net'])), $ppn['net'] >= 0 ? 'kurang bayar' : 'lebih bayar');
        }
        foreach ($this->recap->withholding($client, $period, $entries)['groups'] as $g) {
            $out[] = sprintf('- %s %s: DPP Rp%s, pajak Rp%s (%d transaksi)', config("pajak.codes.{$g['tax_code']}.name", $g['tax_code']),
                $g['direction'] === 'out' ? 'yang kita potong/wajib setor' : 'dipotong pihak lain (kredit pajak kita)',
                Rupiah::format($g['base']), Rupiah::format($g['amount']), $g['count']);
        }
        if ($final = $this->recap->finalUmkm($client, $period)) {
            $out[] = sprintf('- PPh Final UMKM: omzet Rp%s, kena pajak Rp%s, pajak Rp%s', Rupiah::format($final['omzet']),
                Rupiah::format($final['taxable']), $final['tax'] === null ? '(tarif belum ada)' : Rupiah::format($final['tax']));
        }

        return $out ?: ['- Tidak ada transaksi pajak tercatat.'];
    }
}
