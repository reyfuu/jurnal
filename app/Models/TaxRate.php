<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;

/** Satu versi tarif pajak, berlaku dalam rentang tanggal. Tarif = data, bukan konstanta. */
class TaxRate extends Model
{
    protected $guarded = ['id'];

    protected function casts(): array
    {
        return ['effective_from' => 'date', 'effective_to' => 'date', 'verified_at' => 'datetime'];
    }

    /** "12% × 11/12", "2% (4% tanpa NPWP)" — untuk tampilan. */
    public function label(): string
    {
        $pct = fn (int $bp) => rtrim(rtrim(number_format($bp / 100, 2, ',', '.'), '0'), ',').'%';
        $out = $pct($this->rate_bp);
        if ($this->dpp_numerator !== $this->dpp_denominator) {
            $out .= " × {$this->dpp_numerator}/{$this->dpp_denominator}";
        }
        if ($this->rate_no_tax_id_bp) {
            $out .= ' ('.$pct($this->rate_no_tax_id_bp).' tanpa NPWP)';
        }

        return $out;
    }
}
