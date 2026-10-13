<!DOCTYPE html>
<html lang="id">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Masuk · {{ config('app.name') }}</title>
    @vite(['resources/css/app.css', 'resources/js/app.js'])
</head>
<body class="grid min-h-screen place-items-center px-4">
    <div class="w-full max-w-sm">
        <div class="mb-8">
            <span class="grid size-9 place-items-center rounded-md bg-brand-600 text-white" aria-hidden="true">
                <svg viewBox="0 0 16 16" class="size-4" fill="currentColor"><rect x="3" y="2" width="4" height="12" rx="1"/><rect x="9" y="2" width="4" height="12" rx="1" opacity=".55"/></svg>
            </span>
            <h1 class="mt-4 text-2xl font-semibold tracking-tight text-ink-900">Masuk ke {{ config('app.name') }}</h1>
            <p class="mt-1 text-sm text-ink-500">AI membaca bukti transaksi. Kamu yang memutuskan apa yang masuk buku.</p>
        </div>

        <form method="POST" action="{{ route('login') }}" class="space-y-4">
            @csrf
            @if ($errors->any())
                <p class="rounded-md border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-800" role="alert">
                    {{ $errors->first() }}
                </p>
            @endif

            <div>
                <label class="label" for="email">Email</label>
                <input type="email" id="email" name="email" value="{{ old('email') }}" required autofocus autocomplete="username" class="field">
            </div>
            <div>
                <label class="label" for="password">Password</label>
                <div class="relative">
                    <input type="password" id="password" name="password" required autocomplete="current-password" class="field pr-10">
                    <button type="button" data-reveal="password" aria-pressed="false" aria-label="Tampilkan password"
                            class="absolute inset-y-0 right-0 grid w-10 place-items-center rounded-r-md text-ink-400 hover:text-ink-800 cursor-pointer">
                        <x-icon name="eye"/>
                        <x-icon name="eye-off" class="size-4 hidden"/>
                    </button>
                </div>
            </div>
            <label class="flex items-center gap-2 text-sm text-ink-600">
                <input type="checkbox" name="remember" value="1" class="rounded border-ink-300 accent-brand-600">
                Ingat saya
            </label>
            <button class="btn-primary w-full py-2.5">Masuk</button>
        </form>
    </div>
</body>
</html>
