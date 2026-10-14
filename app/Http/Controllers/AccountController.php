<?php

namespace App\Http\Controllers;

use App\Models\Account;
use App\Models\Client;
use App\Services\Accounting\ChartOfAccounts;
use App\Support\Xlsx;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Validator;
use Illuminate\Support\Str;
use Illuminate\Validation\Rule;
use Illuminate\Validation\ValidationException;
use Symfony\Component\HttpFoundation\BinaryFileResponse;

/**
 * COA per klien. Tambah/ubah lewat modal, impor/ekspor Excel. Semua jalur
 * (modal & impor) memakai aturan yang sama — rules() + guardType().
 */
class AccountController extends Controller
{
    /** Kolom file impor/ekspor. Urutan = urutan kolom di Excel. */
    private const COLUMNS = ['Kode', 'Nama akun', 'Tipe', 'Subtype', 'Saldo normal', 'Aktif'];

    public function index(Client $client)
    {
        return view('accounts.index', [
            'client' => $client,
            'accounts' => $client->accounts()->withExists('lines as used')->orderBy('code')->get()->groupBy('type'),
            'types' => Account::TYPES,
            'subtypes' => collect(array_keys(ChartOfAccounts::BUSINESS_TYPES))
                ->flatMap(fn ($j) => array_column(ChartOfAccounts::template($j), 'subtype'))->filter()->unique()->sort()->values(),
        ]);
    }

    public function store(Client $client, Request $request): RedirectResponse
    {
        $data = $this->validated($client, $request);
        $client->accounts()->create($data);

        return back()->with('status', "Akun {$data['code']} ditambahkan.");
    }

    public function update(Client $client, Account $account, Request $request): RedirectResponse
    {
        $this->assertOwned($client, $account);
        $account->update($this->validated($client, $request, $account));

        return back()->with('status', "Akun {$account->code} diperbarui.");
    }

    public function destroy(Client $client, Account $account): RedirectResponse
    {
        $this->assertOwned($client, $account);

        // Akun yang sudah dipakai jurnal tidak boleh hilang — saldo historisnya
        // akan menguap dan laporan periode lalu jadi salah. Nonaktifkan saja.
        if ($account->lines()->exists()) {
            $account->update(['is_active' => false]);

            return back()->with('status', "Akun {$account->code} sudah dipakai di jurnal, jadi dinonaktifkan (bukan dihapus).");
        }

        $code = $account->code;
        $account->delete();

        return back()->with('status', "Akun {$code} dihapus.");
    }

    public function export(Client $client): BinaryFileResponse
    {
        $rows = $client->accounts()->orderBy('code')->get()->map(fn (Account $a) => [
            $a->code, $a->name, Account::TYPES[$a->type] ?? $a->type, $a->subtype, $a->normal_balance, $a->is_active ? 'ya' : 'tidak',
        ]);

        return Xlsx::download('COA '.Str::slug($client->name).' '.now()->format('Y-m-d').'.xlsx', 'COA', self::COLUMNS, $rows);
    }

    /**
     * Impor per KODE, hanya untuk klien ini: kode baru → dibuat, kode yang ada →
     * diperbarui, akun yang tidak ada di file → dibiarkan. Satu baris salah =
     * seluruh impor batal (tidak ada COA setengah jadi).
     */
    public function import(Client $client, Request $request): RedirectResponse
    {
        $request->validate(['file' => ['required', 'file', 'max:2048', 'mimes:xlsx,csv,txt']]);
        $file = $request->file('file');

        try {
            $rows = Xlsx::read($file->getRealPath(), $file->getClientOriginalExtension());
        } catch (\Throwable) {
            return back()->withErrors(['file' => 'File tidak bisa dibaca. Simpan sebagai Excel (.xlsx) atau CSV, lalu coba lagi.']);
        }

        $map = $this->headerMap(array_shift($rows) ?? []);
        if (! isset($map['code'], $map['name'], $map['type'])) {
            return back()->withErrors(['file' => 'Baris pertama harus judul kolom: '.implode(' · ', self::COLUMNS).'. Unduh "Ekspor Excel" sebagai contoh format.']);
        }

        $existing = $client->accounts()->get()->keyBy('code');
        $errors = [];
        $parsed = [];
        $seen = [];
        foreach ($rows as $i => $row) {
            $line = $i + 2; // +1 judul, +1 karena Excel mulai dari 1
            $cell = fn (string $key) => isset($map[$key]) ? trim((string) ($row[$map[$key]] ?? '')) : '';
            $data = [
                'code' => $cell('code'),
                'name' => $cell('name'),
                'type' => $this->typeFrom($cell('type')),
                'subtype' => $cell('subtype') ?: null,
                'normal_balance' => $this->balanceFrom($cell('normal_balance')),
                'is_active' => $this->boolFrom($cell('is_active')),
            ];
            if ($data['code'] === '' && $data['name'] === '') {
                continue;
            }
            if (isset($seen[$data['code']])) {
                $errors[] = "Baris {$line}: kode {$data['code']} dobel (juga di baris {$seen[$data['code']]}).";

                continue;
            }
            $seen[$data['code']] = $line;

            $account = $existing->get($data['code']);
            $validator = Validator::make($data, $this->rules($client, $account), [], self::attributes());
            $validator->after(fn ($v) => $this->guardType($v, $account, (string) $data['type']));
            if ($validator->fails()) {
                $errors[] = "Baris {$line}".($data['code'] !== '' ? " ({$data['code']})" : '').': '.implode(' ', $validator->errors()->all());

                continue;
            }
            $parsed[] = [$account, $this->finish($data)];
        }

        if ($errors !== []) {
            $more = count($errors) > 8 ? ['… dan '.(count($errors) - 8).' baris lain.'] : [];

            return back()->withErrors(['file' => ['Impor dibatalkan — tidak ada yang diubah. Perbaiki baris berikut lalu unggah ulang:', ...array_slice($errors, 0, 8), ...$more]]);
        }
        if ($parsed === []) {
            return back()->withErrors(['file' => 'File tidak berisi akun.']);
        }

        DB::transaction(function () use ($client, $parsed) {
            foreach ($parsed as [$account, $data]) {
                $account ? $account->update($data) : $client->accounts()->create($data);
            }
        });

        $baru = count(array_filter($parsed, fn ($p) => $p[0] === null));

        return back()->with('status', "Impor COA selesai: {$baru} akun baru, ".(count($parsed) - $baru).' diperbarui.');
    }

    private function assertOwned(Client $client, Account $account): void
    {
        abort_unless($account->client_id === $client->id, 404);
    }

    private function validated(Client $client, Request $request, ?Account $account = null): array
    {
        $validator = Validator::make($request->all(), $this->rules($client, $account), [], self::attributes());
        $validator->after(fn ($v) => $this->guardType($v, $account, (string) $request->input('type')));
        // Modal dibuka ulang dengan isian lama kalau validasi gagal (lihat accounts/index).
        $data = $validator->validate();
        $data['is_active'] = $request->boolean('is_active');

        return $this->finish($data);
    }

    /** @return array<string,array<int,mixed>> */
    private function rules(Client $client, ?Account $account): array
    {
        return [
            'code' => ['required', 'string', 'max:20', 'regex:/^[A-Za-z0-9.\-]+$/',
                Rule::unique('accounts', 'code')->where('client_id', $client->id)->ignore($account?->id)],
            'name' => ['required', 'string', 'max:160'],
            'type' => ['required', Rule::in(array_keys(Account::TYPES))],
            'subtype' => ['nullable', 'string', 'max:40'],
            'normal_balance' => ['nullable', Rule::in(['debit', 'credit'])],
            'is_active' => ['nullable', 'boolean'],
        ];
    }

    /** Tipe akun yang sudah dipakai jurnal tidak boleh diganti — laporan periode lalu akan berpindah pos. */
    private function guardType($validator, ?Account $account, string $type): void
    {
        if ($account && $type !== '' && $account->type !== $type && $account->lines()->exists()) {
            $validator->errors()->add('type', "Tipe akun {$account->code} tidak bisa diganti karena sudah dipakai jurnal. Buat akun baru, lalu nonaktifkan yang lama.");
        }
    }

    private function finish(array $data): array
    {
        $data['subtype'] = filled($data['subtype'] ?? null) ? $data['subtype'] : null;
        $data['normal_balance'] = ($data['normal_balance'] ?? null)
            ?: ChartOfAccounts::normalBalanceFor($data['code'], $data['type'], $data['subtype']);
        $data['is_active'] = (bool) ($data['is_active'] ?? true);

        return $data;
    }

    private static function attributes(): array
    {
        return ['code' => 'kode', 'name' => 'nama akun', 'type' => 'tipe', 'normal_balance' => 'saldo normal'];
    }

    /** Judul kolom → indeks. Toleran huruf besar/kecil & variasi nama. */
    private function headerMap(array $header): array
    {
        $alias = [
            'code' => ['kode', 'kode akun', 'code', 'no akun', 'nomor akun'],
            'name' => ['nama akun', 'nama', 'name', 'akun'],
            'type' => ['tipe', 'jenis', 'type', 'kelompok'],
            'subtype' => ['subtype', 'subtipe', 'sub tipe'],
            'normal_balance' => ['saldo normal', 'normal balance', 'saldo'],
            'is_active' => ['aktif', 'status', 'active'],
        ];
        $map = [];
        foreach ($header as $i => $title) {
            $title = Str::of((string) $title)->lower()->squish()->toString();
            foreach ($alias as $key => $names) {
                if (! isset($map[$key]) && in_array($title, $names, true)) {
                    $map[$key] = $i;
                }
            }
        }

        return $map;
    }

    /** "Aset" / "asset" / "harta" → asset. Tidak dikenal → dikembalikan apa adanya (ditolak validasi). */
    private function typeFrom(string $value): string
    {
        $v = Str::lower(trim($value));
        $labels = array_change_key_case(array_flip(Account::TYPES)) + [
            'harta' => 'asset', 'aktiva' => 'asset', 'kewajiban' => 'liability', 'utang' => 'liability', 'hutang' => 'liability',
            'modal' => 'equity', 'ekuitas' => 'equity', 'penghasilan' => 'revenue', 'biaya' => 'expense', 'beban' => 'expense',
        ];

        return isset(Account::TYPES[$v]) ? $v : ($labels[$v] ?? $value);
    }

    private function balanceFrom(string $value): ?string
    {
        return match (Str::lower(trim($value))) {
            '' => null,
            'debit', 'd', 'db' => 'debit',
            'credit', 'kredit', 'k', 'cr' => 'credit',
            default => $value,
        };
    }

    private function boolFrom(string $value): bool
    {
        return ! in_array(Str::lower(trim($value)), ['0', 'tidak', 'nonaktif', 'no', 'false', 'n', 'off'], true);
    }
}
