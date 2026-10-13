<?php

use App\Http\Controllers\AccountController;
use App\Http\Controllers\AuthController;
use App\Http\Controllers\ClientController;
use App\Http\Controllers\DashboardController;
use App\Http\Controllers\DocumentController;
use App\Http\Controllers\JournalController;
use App\Http\Controllers\ReportController;
use App\Http\Controllers\SettingController;
use App\Http\Controllers\TaxController;
use App\Http\Controllers\UserController;
use Illuminate\Support\Facades\Route;

Route::get('/login', [AuthController::class, 'showLogin'])->name('login');
Route::post('/login', [AuthController::class, 'login'])->middleware('throttle:10,1');

/*
 * Hak akses: kemampuan di App\Models\User::ABILITIES (middleware `ability:`).
 * `client.access` memastikan user hanya membuka klien yang ditugaskan padanya.
 */
Route::middleware(['auth', 'active', 'client.access'])->group(function () {
    Route::post('/logout', [AuthController::class, 'logout'])->name('logout');

    Route::get('/', [DashboardController::class, 'index'])->name('dashboard');

    // ── Dokumen: semua peran boleh upload & melihat ─────────────────────────
    Route::middleware('ability:documents.upload')->group(function () {
        Route::get('/klien/{client}/dokumen', [DocumentController::class, 'index'])->name('documents.index');
        Route::get('/klien/{client}/dokumen/baru', [DocumentController::class, 'create'])->name('documents.create');
        Route::post('/klien/{client}/dokumen', [DocumentController::class, 'store'])->name('documents.store');
        Route::get('/klien/{client}/dokumen/{document}', [DocumentController::class, 'show'])->name('documents.show');
        Route::get('/klien/{client}/dokumen/{document}/berkas', [DocumentController::class, 'file'])->name('documents.file');
    });

    // ── Melihat buku: klien, COA, jurnal, laporan, pajak ────────────────────
    Route::middleware('ability:books.view')->group(function () {
        Route::get('/klien', [ClientController::class, 'index'])->name('clients.index');
        Route::get('/klien/{client}', [ClientController::class, 'show'])->name('clients.show');
        Route::get('/klien/{client}/coa', [AccountController::class, 'index'])->name('accounts.index');
        Route::get('/klien/{client}/jurnal', [JournalController::class, 'index'])->name('journals.index');

        Route::get('/klien/{client}/laporan/neraca-saldo', [ReportController::class, 'trialBalance'])->name('reports.trial-balance');
        Route::get('/klien/{client}/laporan/laba-rugi', [ReportController::class, 'incomeStatement'])->name('reports.income-statement');
        Route::get('/klien/{client}/laporan/neraca', [ReportController::class, 'balanceSheet'])->name('reports.balance-sheet');
        Route::get('/klien/{client}/laporan/arus-kas', [ReportController::class, 'cashFlow'])->name('reports.cash-flow');
        Route::get('/klien/{client}/laporan/ekspor/{jenis}', [ReportController::class, 'export'])->name('reports.export');

        Route::get('/klien/{client}/pajak', [TaxController::class, 'index'])->name('tax.index');
        Route::get('/klien/{client}/pajak/ekspor/{jenis}', [TaxController::class, 'export'])->name('tax.export');
        Route::get('/klien/{client}/pajak/profil', [TaxController::class, 'profile'])->name('tax.profile');
    });

    // ── Mengubah buku: review & posting, jurnal manual, void, status pajak ──
    Route::middleware('ability:books.post')->group(function () {
        Route::post('/klien/{client}/dokumen/{document}/baca-ulang', [DocumentController::class, 'reextract'])->name('documents.reextract');
        Route::post('/klien/{client}/dokumen/{document}/posting', [DocumentController::class, 'post'])->name('documents.post');
        Route::get('/klien/{client}/jurnal/baru', [JournalController::class, 'create'])->name('journals.create');
        Route::post('/klien/{client}/jurnal', [JournalController::class, 'store'])->name('journals.store');
        Route::post('/klien/{client}/jurnal/{journal}/void', [JournalController::class, 'void'])->name('journals.void');
        Route::post('/klien/{client}/pajak/status', [TaxController::class, 'storeFiling'])->name('tax.filing');
        Route::get('/klien/{client}/pajak/draft/{jenis}', [TaxController::class, 'draft'])->name('tax.draft');
    });

    // ── Struktur buku: klien, COA, profil pajak, audit ──────────────────────
    Route::middleware('ability:clients.manage')->group(function () {
        Route::get('/klien-baru', [ClientController::class, 'create'])->name('clients.create');
        Route::post('/klien', [ClientController::class, 'store'])->name('clients.store');
        Route::get('/klien/{client}/ubah', [ClientController::class, 'edit'])->name('clients.edit');
        Route::put('/klien/{client}', [ClientController::class, 'update'])->name('clients.update');
        Route::delete('/klien/{client}', [ClientController::class, 'destroy'])->name('clients.destroy');
        Route::post('/klien/{client}/coa/sinkron', [ClientController::class, 'syncCoa'])->name('clients.coa.sync');
        Route::post('/klien/{client}/coa', [AccountController::class, 'store'])->name('accounts.store');
        Route::put('/klien/{client}/coa/{account}', [AccountController::class, 'update'])->name('accounts.update');
        Route::delete('/klien/{client}/coa/{account}', [AccountController::class, 'destroy'])->name('accounts.destroy');
        Route::put('/klien/{client}/pajak/profil', [TaxController::class, 'updateProfile'])->name('tax.profile.update');
        Route::get('/audit', [TaxController::class, 'audit'])->name('audit');
    });

    // ── Hapus permanen (selain itu: void / nonaktifkan) ─────────────────────
    Route::middleware('ability:records.delete')->group(function () {
        Route::delete('/klien/{client}/dokumen/{document}', [DocumentController::class, 'destroy'])->name('documents.destroy');
        Route::delete('/klien/{client}/jurnal/{journal}', [JournalController::class, 'destroy'])->name('journals.destroy');
    });

    // ── Tarif pajak & pengguna ──────────────────────────────────────────────
    Route::middleware('ability:tax.rates')->group(function () {
        Route::get('/pajak/tarif', [TaxController::class, 'rates'])->name('tax.rates');
        Route::post('/pajak/tarif', [TaxController::class, 'storeRate'])->name('tax.rates.store');
        Route::post('/pajak/tarif/{rate}/verifikasi', [TaxController::class, 'verifyRate'])->name('tax.rates.verify');
    });

    Route::middleware('ability:settings.manage')->group(function () {
        Route::get('/pengaturan', [SettingController::class, 'edit'])->name('settings.edit');
        Route::put('/pengaturan', [SettingController::class, 'update'])->name('settings.update');
        Route::post('/pengaturan/tes', [SettingController::class, 'test'])->middleware('throttle:6,1')->name('settings.test');
    });

    Route::middleware('ability:users.manage')->group(function () {
        Route::get('/pengguna', [UserController::class, 'index'])->name('users.index');
        Route::get('/pengguna/baru', [UserController::class, 'create'])->name('users.create');
        Route::post('/pengguna', [UserController::class, 'store'])->name('users.store');
        Route::get('/pengguna/{user}/ubah', [UserController::class, 'edit'])->name('users.edit');
        Route::put('/pengguna/{user}', [UserController::class, 'update'])->name('users.update');
        Route::post('/pengguna/{user}/status', [UserController::class, 'toggle'])->name('users.toggle');
    });
});
