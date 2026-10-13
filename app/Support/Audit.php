<?php

namespace App\Support;

use App\Models\AuditLog;

final class Audit
{
    /** Catat satu perubahan. Nilai sensitif (NPWP) wajib sudah dimasking oleh pemanggil. */
    public static function log(string $action, string $subjectType, ?int $subjectId, ?int $clientId, ?array $before, ?array $after): void
    {
        AuditLog::create([
            'user_id' => auth()->id(),
            'client_id' => $clientId,
            'subject_type' => $subjectType,
            'subject_id' => $subjectId,
            'action' => $action,
            'before' => $before,
            'after' => $after,
        ]);
    }
}
