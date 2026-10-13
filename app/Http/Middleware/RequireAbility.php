<?php

namespace App\Http\Middleware;

use Closure;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;

/** `->middleware('ability:books.post')` — hak diambil dari User::may(). */
class RequireAbility
{
    public function handle(Request $request, Closure $next, string $ability): Response
    {
        abort_unless($request->user()?->may($ability), 403, 'Peranmu tidak punya akses ke fitur ini.');

        return $next($request);
    }
}
