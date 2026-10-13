<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;

/**
 * Pengaturan key/value dari UI admin. Nilai dienkripsi dengan APP_KEY —
 * ganti APP_KEY = isi ulang key AI di halaman Pengaturan.
 */
class Setting extends Model
{
    protected $primaryKey = 'key';

    protected $keyType = 'string';

    public $incrementing = false;

    protected $fillable = ['key', 'value'];

    protected function casts(): array
    {
        return ['value' => 'encrypted'];
    }

    public static function get(string $key): ?string
    {
        try {
            return static::find($key)?->value;
        } catch (\Throwable) {
            return null; // tabel belum dimigrasi / APP_KEY berubah → anggap kosong
        }
    }

    public static function put(string $key, ?string $value): void
    {
        blank($value)
            ? static::whereKey($key)->delete()
            : static::updateOrCreate(['key' => $key], ['value' => $value]);
    }
}
