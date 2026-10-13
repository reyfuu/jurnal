<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Support\Str;

class Client extends Model
{
    public const TYPE_INTERNAL = 'internal';

    public const TYPE_EXTERNAL = 'external';

    public const TYPES = [
        self::TYPE_INTERNAL => 'Brand / unit bisnis sendiri',
        self::TYPE_EXTERNAL => 'Klien jasa pembukuan',
    ];

    protected $fillable = [
        'name', 'slug', 'type', 'legal_name', 'npwp', 'contact_name',
        'phone', 'email', 'address', 'fiscal_year_start', 'notes', 'is_active',
        'taxpayer_type', 'tax_id', 'is_pkp', 'pkp_since', 'pph_regime',
    ];

    public const TAXPAYER_TYPES = ['orang_pribadi' => 'Orang pribadi', 'badan' => 'Badan (PT/CV/dll.)'];

    public const REGIMES = ['normal' => 'Normal (tarif umum)', 'final_umkm' => 'PPh Final UMKM'];

    protected function casts(): array
    {
        return [
            'is_active' => 'boolean',
            'is_pkp' => 'boolean',
            'pkp_since' => 'date',
            'tax_id' => 'encrypted',
        ];
    }

    protected static function booted(): void
    {
        static::saving(function (self $client) {
            if (blank($client->slug)) {
                $client->slug = static::uniqueSlug($client->name, $client->id);
            }
        });
    }

    /** Slug yang dijamin unik — nambah sufiks angka kalau sudah dipakai klien lain. */
    public static function uniqueSlug(string $name, ?int $ignoreId = null): string
    {
        $base = Str::slug($name) ?: 'klien';
        $slug = $base;
        $n = 2;
        while (static::where('slug', $slug)->when($ignoreId, fn ($q) => $q->whereKeyNot($ignoreId))->exists()) {
            $slug = $base.'-'.$n++;
        }

        return $slug;
    }

    public function scopeActive($query)
    {
        return $query->where('is_active', true);
    }

    public function users()
    {
        return $this->belongsToMany(User::class);
    }

    /** Klien yang boleh dilihat user ini (admin/supervisor: semua). */
    public function scopeVisibleTo($query, User $user)
    {
        return $user->may('clients.all') ? $query : $query->whereHas('users', fn ($q) => $q->whereKey($user->id));
    }

    public function accounts()
    {
        return $this->hasMany(Account::class);
    }

    public function journals()
    {
        return $this->hasMany(Journal::class);
    }

    public function documents()
    {
        return $this->hasMany(Document::class);
    }

    public function taxEntries()
    {
        return $this->hasMany(TaxEntry::class);
    }

    /**
     * Status PKP pada tanggal tertentu. Profil kosong (is_pkp null) diperlakukan
     * PKP — sama dengan perilaku sebelum fitur pajak, supaya data lama tidak
     * berubah arti (TRD Q2).
     */
    public function isPkpOn(?string $date = null): bool
    {
        if ($this->is_pkp === null) {
            return true;
        }

        return $this->is_pkp && (! $this->pkp_since || ! $date || $this->pkp_since->toDateString() <= $date);
    }

    public function isFinalUmkm(): bool
    {
        return $this->pph_regime === 'final_umkm';
    }

    public function hasTaxProfile(): bool
    {
        return $this->is_pkp !== null;
    }

    public function typeLabel(): string
    {
        return self::TYPES[$this->type] ?? $this->type;
    }
}
