<?php

namespace App\Http\Middleware;

use App\Models\Client;
use Closure;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;

/**
 * Pagar isolasi per user: route ber-{client} hanya untuk user yang ditugaskan
 * ke klien itu (atau admin/supervisor). 404, bukan 403, supaya keberadaan
 * klien lain tidak terkonfirmasi.
 */
class EnsureClientAccess
{
    public function handle(Request $request, Closure $next): Response
    {
        $client = $request->route('client');
        if ($client instanceof Client) {
            abort_unless($request->user()?->canAccessClient($client), 404);
        }

        return $next($request);
    }
}
