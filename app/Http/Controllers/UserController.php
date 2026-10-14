<?php

namespace App\Http\Controllers;

use App\Models\Client;
use App\Models\User;
use App\Support\Audit;
use App\Support\SandiLemah;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Validation\Rule;
use Illuminate\Validation\ValidationException;

/**
 * Kelola pengguna (khusus admin). Tidak ada hapus: pengguna yang sudah pernah
 * memposting meninggalkan jejak di jurnal — nonaktifkan saja.
 */
class UserController extends Controller
{
    public function index()
    {
        return view('users.index', [
            'users' => User::with('clients:id,name')->orderByDesc('is_active')->orderBy('role')->orderBy('name')->get(),
        ]);
    }

    public function create()
    {
        return view('users.form', ['user' => new User(['role' => User::ROLE_AKUNTAN, 'is_active' => true]), 'clients' => $this->clients()]);
    }

    public function store(Request $request): RedirectResponse
    {
        $data = $this->validated($request, new User);
        $user = User::create($data + ['is_active' => true]);
        $user->clients()->sync($data['clients']);
        Audit::log('user.create', 'user', $user->id, null, null, $this->snapshot($user));

        return redirect()->route('users.index')->with('status', "Pengguna {$user->name} dibuat sebagai {$user->roleLabel()}.");
    }

    public function edit(User $user)
    {
        return view('users.form', ['user' => $user->load('clients:id'), 'clients' => $this->clients()]);
    }

    public function update(User $user, Request $request): RedirectResponse
    {
        $data = $this->validated($request, $user);
        if ($user->is($request->user()) && $data['role'] !== $user->role) {
            return back()->withInput()->withErrors(['role' => 'Tidak bisa mengubah peranmu sendiri — minta admin lain.']);
        }

        $before = $this->snapshot($user);
        $user->update(array_filter($data, fn ($v, $k) => $k !== 'password' || filled($v), ARRAY_FILTER_USE_BOTH));
        $user->clients()->sync($data['clients']);
        Audit::log('user.update', 'user', $user->id, null, $before, $this->snapshot($user->refresh()));

        return redirect()->route('users.index')->with('status', "Pengguna {$user->name} diperbarui.");
    }

    public function toggle(User $user, Request $request): RedirectResponse
    {
        abort_if($user->is($request->user()), 422, 'Tidak bisa menonaktifkan akunmu sendiri.');
        $user->update(['is_active' => ! $user->is_active]);
        Audit::log($user->is_active ? 'user.activate' : 'user.deactivate', 'user', $user->id, null, null, ['is_active' => $user->is_active]);

        return back()->with('status', "{$user->name} ".($user->is_active ? 'diaktifkan kembali.' : 'dinonaktifkan — langsung keluar dari sesinya.'));
    }

    /** @return array<string,mixed> */
    private function validated(Request $request, User $user): array
    {
        $data = $request->validate([
            'name' => ['required', 'string', 'max:120'],
            'email' => ['required', 'email', 'max:190', Rule::unique('users', 'email')->ignore($user->id)],
            'role' => ['required', Rule::in(array_keys(User::ROLES))],
            'password' => [$user->exists ? 'nullable' : 'required', 'string', 'max:190', function ($attr, $value, $fail) {
                if (filled($value) && $reason = SandiLemah::alasanTolak($value)) {
                    $fail($reason);
                }
            }],
            'clients' => ['nullable', 'array'],
            'clients.*' => ['integer', 'exists:clients,id'],
        ]);
        $data['clients'] = array_values(array_unique(array_map('intval', $data['clients'] ?? [])));

        $limit = User::CLIENT_LIMITS[$data['role']] ?? null;
        if ($limit === null) {
            $data['clients'] = []; // admin melihat semua — penugasan tidak berarti
        } elseif (count($data['clients']) > $limit) {
            throw ValidationException::withMessages([
                'clients' => sprintf('%s maksimal memegang %d perusahaan — %d dipilih.', User::ROLES[$data['role']], $limit, count($data['clients'])),
            ]);
        }

        return $data;
    }

    private function snapshot(User $user): array
    {
        return ['name' => $user->name, 'email' => $user->email, 'role' => $user->role, 'clients' => $user->clients()->pluck('clients.id')->all()];
    }

    private function clients()
    {
        return Client::orderBy('name')->get(['id', 'name', 'is_active']);
    }
}
