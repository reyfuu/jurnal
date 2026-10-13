<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;

/** Status setor/lapor satu kewajiban pajak per masa. Diisi manusia, tidak pernah otomatis. */
class TaxFiling extends Model
{
    public const STATUSES = ['belum' => 'Belum', 'setor' => 'Sudah setor', 'lapor' => 'Sudah lapor', 'nihil' => 'Nihil'];

    protected $guarded = ['id'];

    protected function casts(): array
    {
        return ['paid_at' => 'date', 'filed_at' => 'date'];
    }
}
