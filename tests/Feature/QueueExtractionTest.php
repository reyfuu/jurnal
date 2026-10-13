<?php

namespace Tests\Feature;

use App\Jobs\ExtractDocument;
use App\Models\Client;
use App\Models\Document;
use App\Models\User;
use App\Services\Accounting\ChartOfAccounts;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Queue;
use Tests\TestCase;

/** AI_QUEUE=true: upload tidak menunggu AI; job membaca dan mencatat hasil/gagalnya di dokumen. */
class QueueExtractionTest extends TestCase
{
    use RefreshDatabase;

    public function test_upload_masuk_antrian_lalu_job_membaca(): void
    {
        config(['ai.queue' => true, 'ai.provider' => 'openai', 'ai.openai.key' => 'sk-test', 'ai.model' => 'gpt-4o-mini', 'ai.backup.key' => null]);
        $client = Client::create(['name' => 'Uji', 'type' => Client::TYPE_EXTERNAL]);
        app(ChartOfAccounts::class)->seedFor($client);
        $user = User::create(['name' => 'A', 'email' => 'a@t.id', 'password' => 'secret123', 'role' => User::ROLE_ADMIN, 'is_active' => true]);

        Queue::fake();
        Http::fake(['*/chat/completions' => Http::sequence()
            ->push(['choices' => [['message' => ['content' => json_encode([
                'flow' => 'expense', 'document_date' => '2026-10-08', 'payment_method' => 'bank', 'total' => 500000,
                'items' => [['description' => 'Iklan', 'amount' => 500000, 'account_code' => '6102', 'confidence' => 0.9]],
            ])]]]])
            ->pushStatus(500)]);
        $this->actingAs($user)->post(route('documents.store', $client), ['kind' => Document::KIND_TEXT, 'raw_text' => 'iklan 500rb'])
            ->assertRedirect();
        $document = Document::sole();

        Queue::assertPushed(ExtractDocument::class);
        Http::assertNothingSent();
        $this->assertSame(Document::STATUS_UPLOADED, $document->status);
        $this->actingAs($user)->get(route('documents.show', [$client, $document]))->assertSee('http-equiv="refresh"', false);

        (new ExtractDocument($document))->handle(app(\App\Services\Intake\DocumentExtractor::class));
        $this->assertSame(Document::STATUS_EXTRACTED, $document->fresh()->status);

        // Gagal baca tidak melempar dari job — alasannya tercatat di dokumen.
        (new ExtractDocument($document))->handle(app(\App\Services\Intake\DocumentExtractor::class));
        $this->assertSame(Document::STATUS_FAILED, $document->fresh()->status);
    }
}
