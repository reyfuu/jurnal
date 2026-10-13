<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;

/** Jejak perubahan profil pajak, tarif, dan status setor/lapor. Append-only. */
class AuditLog extends Model
{
    public const UPDATED_AT = null;

    protected $guarded = ['id'];

    protected function casts(): array
    {
        return ['before' => 'array', 'after' => 'array'];
    }

    public function user()
    {
        return $this->belongsTo(User::class);
    }
}
