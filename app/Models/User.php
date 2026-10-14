<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Foundation\Auth\User as Authenticatable;
use Illuminate\Notifications\Notifiable;

class User extends Authenticatable
{
    /** @use HasFactory<\Database\Factories\UserFactory> */
    use HasFactory, Notifiable;

    public const ROLE_ADMIN = 'admin';

    public const ROLE_SUPERVISOR = 'supervisor';

    public const ROLE_AKUNTAN = 'akuntan';

    public const ROLE_ANGGOTA = 'anggota';

    public const ROLES = [
        self::ROLE_ADMIN => 'Admin',
        self::ROLE_SUPERVISOR => 'Supervisor',
        self::ROLE_AKUNTAN => 'Akuntan',
        self::ROLE_ANGGOTA => 'Anggota',
    ];

    public const ROLE_HINTS = [
        self::ROLE_ADMIN => 'Pemilik sistem: semua akses + kelola pengguna & tarif pajak.',
        self::ROLE_SUPERVISOR => 'Maks. 3 perusahaan yang ditugaskan. Kelola klien, COA, profil pajak, audit. Review & posting.',
        self::ROLE_AKUNTAN => 'Maks. 5 perusahaan yang ditugaskan. Review, posting, void, laporan, pajak.',
        self::ROLE_ANGGOTA => 'Maks. 10 perusahaan yang ditugaskan. Hanya upload & melihat dokumen — tidak bisa posting atau melihat laporan.',
    ];

    /** Batas jumlah perusahaan yang boleh dipegang per peran. Admin tanpa batas (melihat semua). */
    public const CLIENT_LIMITS = [
        self::ROLE_SUPERVISOR => 3,
        self::ROLE_AKUNTAN => 5,
        self::ROLE_ANGGOTA => 10,
    ];

    /**
     * Hak per kemampuan. Satu tempat — route (middleware `ability`) dan tampilan
     * membaca dari sini. ponytail: peran tetap di kode, bukan tabel role dinamis;
     * buat tabel kalau admin perlu merakit peran sendiri.
     */
    private const ABILITIES = [
        'users.manage' => [self::ROLE_ADMIN],
        'tax.rates' => [self::ROLE_ADMIN],
        'settings.manage' => [self::ROLE_ADMIN],
        'records.delete' => [self::ROLE_ADMIN],
        'clients.all' => [self::ROLE_ADMIN],
        'clients.manage' => [self::ROLE_ADMIN, self::ROLE_SUPERVISOR],
        'books.view' => [self::ROLE_ADMIN, self::ROLE_SUPERVISOR, self::ROLE_AKUNTAN],
        'books.post' => [self::ROLE_ADMIN, self::ROLE_SUPERVISOR, self::ROLE_AKUNTAN],
        'documents.upload' => [self::ROLE_ADMIN, self::ROLE_SUPERVISOR, self::ROLE_AKUNTAN, self::ROLE_ANGGOTA],
    ];

    protected $fillable = ['name', 'email', 'password', 'role', 'is_active'];

    protected $hidden = ['password', 'remember_token'];

    protected function casts(): array
    {
        return [
            'email_verified_at' => 'datetime',
            'password' => 'hashed',
            'is_active' => 'boolean',
        ];
    }

    public function clients()
    {
        return $this->belongsToMany(Client::class);
    }

    public function isAdmin(): bool
    {
        return $this->role === self::ROLE_ADMIN;
    }

    /** Boleh melakukan kemampuan ini? (Nama `may`, bukan `can`, supaya tidak bentrok dengan Gate Laravel.) */
    public function may(string $ability): bool
    {
        return in_array($this->role, self::ABILITIES[$ability] ?? [], true);
    }

    public function canAccessClient(Client $client): bool
    {
        return $this->may('clients.all') || $this->clients->contains($client);
    }

    /** Batas perusahaan untuk peran ini; null = tanpa batas (admin). */
    public function clientLimit(): ?int
    {
        return self::CLIENT_LIMITS[$this->role] ?? null;
    }

    public function roleLabel(): string
    {
        return self::ROLES[$this->role] ?? $this->role;
    }
}
