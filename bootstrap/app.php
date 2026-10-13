<?php

use App\Http\Middleware\AdminOnly;
use App\Http\Middleware\EnsureActiveUser;
use Illuminate\Foundation\Application;
use Illuminate\Foundation\Configuration\Exceptions;
use Illuminate\Foundation\Configuration\Middleware;

return Application::configure(basePath: dirname(__DIR__))
    ->withRouting(
        web: __DIR__.'/../routes/web.php',
        commands: __DIR__.'/../routes/console.php',
        health: '/up',
    )
    ->withMiddleware(function (Middleware $middleware): void {
        $middleware->alias([
            'admin' => AdminOnly::class,
            'active' => EnsureActiveUser::class,
        ]);
        $middleware->redirectGuestsTo('/login');

        // Saat diakses lewat tunnel (cloudflared/ngrok), TLS diputus di sisi
        // layanan tunnel lalu diteruskan ke sini sebagai http biasa. Tanpa ini
        // Laravel menyusun semua URL sebagai http sementara browser berada di
        // https — form dan aset diblokir browser sebagai mixed content.
        //
        // Yang dipercaya HANYA loopback: proses tunnel berjalan di mesin yang
        // sama. Penyerang dari luar tidak bisa membuat requestnya seolah datang
        // dari 127.0.0.1, jadi header X-Forwarded-* palsu tetap diabaikan.
        $middleware->trustProxies(at: ['127.0.0.1', '::1']);
    })
    ->withExceptions(function (Exceptions $exceptions): void {
        //
    })->create();
