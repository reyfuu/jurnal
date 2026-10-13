<?php

namespace App\Services\Ai;

use App\Models\Setting;

/**
 * Satu-satunya tempat yang tahu "lagi pakai otak apa". Key Gemini yang disimpan
 * admin di halaman Pengaturan menang; kalau kosong, provider & model dari
 * config/ai.php (.env), plus rantai cadangan yang aktif hanya kalau key & model
 * cadangan benar-benar terisi.
 */
class AiProviderFactory
{
    public static function make(): AiProvider
    {
        $primary = self::primary();
        $backups = self::backups();

        return $backups === [] ? $primary : new FailoverAiProvider([$primary, ...$backups]);
    }

    /** true kalau sistem punya minimal satu otak siap pakai — dipakai UI untuk memberi peringatan dini. */
    public static function configured(): bool
    {
        try {
            self::primary();

            return true;
        } catch (AiException) {
            return self::backups() !== [];
        }
    }

    private static function primary(): AiProvider
    {
        $provider = (string) config('ai.provider', 'openai');
        $model = (string) config('ai.model');
        $maxTokens = (int) config('ai.max_output_tokens', 4000);
        $timeout = (int) config('ai.request_timeout', 120);
        $connect = (int) config('ai.connect_timeout', 10);

        // Gemini lewat endpoint OpenAI-compatible-nya — cukup reuse OpenAiProvider.
        $geminiKey = (string) (Setting::get('gemini_key') ?: config('ai.gemini.key'));
        if ($geminiKey !== '' || $provider === 'gemini') {
            if ($geminiKey === '') {
                throw new AiException('API key Gemini belum diisi — buka menu Pengaturan.');
            }

            return new OpenAiProvider(
                $geminiKey, (string) config('ai.gemini.base'), self::geminiModel(), $maxTokens, $timeout, $connect,
            );
        }

        if ($provider === 'anthropic') {
            $key = (string) config('ai.anthropic.key');
            if ($key === '') {
                throw new AiException('ANTHROPIC_API_KEY belum diisi di .env.');
            }

            return new AnthropicProvider(
                $key, (string) config('ai.anthropic.base'), $model,
                $maxTokens, $timeout, $connect, (string) config('ai.anthropic.version'),
            );
        }

        $key = (string) config('ai.openai.key');
        if ($key === '') {
            throw new AiException('OPENAI_API_KEY belum diisi di .env.');
        }

        return new OpenAiProvider(
            $key, (string) config('ai.openai.base'), $model, $maxTokens, $timeout, $connect,
            (string) config('ai.token_param', 'max_tokens'),
        );
    }

    public static function geminiModel(): string
    {
        return (string) (Setting::get('gemini_model') ?: config('ai.gemini.model'));
    }

    /**
     * Slot cadangan yang siap pakai. Endpoint OpenAI-compatible, jadi cukup
     * reuse OpenAiProvider dengan key/base/model sendiri.
     *
     * @return array<int,AiProvider>
     */
    private static function backups(): array
    {
        $slot = (array) config('ai.backup', []);
        $key = (string) ($slot['key'] ?? '');
        $model = (string) ($slot['model'] ?? '');
        if ($key === '' || $model === '') {
            return [];
        }

        return [new OpenAiProvider(
            $key,
            (string) ($slot['base'] ?? 'https://openrouter.ai/api/v1'),
            $model,
            (int) config('ai.max_output_tokens', 4000),
            (int) ($slot['timeout'] ?? 120),
            (int) config('ai.connect_timeout', 10),
            (string) config('ai.token_param', 'max_tokens'),
        )];
    }
}
