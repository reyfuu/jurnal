<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;

/** Catatan pajak per jurnal. Dibuat HANYA oleh JournalPoster, di transaksi yang sama dengan jurnalnya. */
class TaxEntry extends Model
{
    protected $guarded = ['id'];

    protected function casts(): array
    {
        return ['withheld' => 'boolean', 'noncreditable' => 'boolean', 'counterparty_tax_id' => 'encrypted'];
    }

    public function journal()
    {
        return $this->belongsTo(Journal::class);
    }

    public function codeName(): string
    {
        return config("pajak.codes.{$this->tax_code}.name", $this->tax_code);
    }
}
