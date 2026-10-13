<?php

namespace App\Jobs;

use App\Models\Document;
use App\Services\Ai\AiException;
use App\Services\Intake\DocumentExtractor;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Queue\Queueable;

/**
 * Baca dokumen di latar belakang (AI_QUEUE=true). Gagal baca bukan kegagalan
 * job: alasannya sudah dicatat di dokumen oleh extractor, user tinggal klik
 * "Baca ulang". Retry hanya untuk error tak terduga (mis. DB terkunci).
 */
class ExtractDocument implements ShouldQueue
{
    use Queueable;

    public int $tries = 2;

    public int $backoff = 30;

    public function __construct(public Document $document) {}

    public function handle(DocumentExtractor $extractor): void
    {
        try {
            $extractor->extract($this->document);
        } catch (AiException) {
            // sudah tercatat di document.status/error
        }
    }
}
