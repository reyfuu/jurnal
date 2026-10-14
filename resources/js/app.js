// Interaksi kecil tanpa framework — cukup untuk form jurnal.
// Semua perhitungan uang yang mengikat tetap di PHP (JournalPoster).

/** Format angka ke gaya Indonesia untuk tampilan total. */
const rupiah = (n) =>
    new Intl.NumberFormat('id-ID', { minimumFractionDigits: 0, maximumFractionDigits: 2 }).format(n || 0);

/**
 * Hitung ulang total debit/kredit satu blok jurnal dan beri tahu kalau belum
 * balance — supaya user tahu sebelum submit, bukan setelah ditolak server.
 */
function wireBalance(root) {
    const inputs = root.querySelectorAll('[data-money]');
    const out = {
        debit: root.querySelector('[data-total="debit"]'),
        credit: root.querySelector('[data-total="credit"]'),
        state: root.querySelector('[data-balance-state]'),
    };
    if (!out.debit || !out.credit) return;

    const recalc = () => {
        let debit = 0;
        let credit = 0;
        inputs.forEach((el) => {
            const value = parseFloat(el.value) || 0;
            if (el.dataset.money === 'debit') debit += value;
            else credit += value;
        });

        out.debit.textContent = rupiah(debit);
        out.credit.textContent = rupiah(credit);

        if (!out.state) return;
        const diff = Math.round((debit - credit) * 100) / 100;
        const balanced = Math.abs(diff) < 0.005;
        out.state.textContent = balanced ? 'Balance' : `Selisih Rp${rupiah(Math.abs(diff))}`;
        out.state.className = balanced ? 'badge-ok' : 'badge-bad';
    };

    inputs.forEach((el) => el.addEventListener('input', recalc));
    recalc();
}

/** Satu baris hanya boleh debit ATAU kredit — kosongkan sisi lain otomatis. */
function wireExclusiveSides(root) {
    root.querySelectorAll('[data-money]').forEach((el) => {
        el.addEventListener('input', () => {
            if (!el.value) return;
            const row = el.closest('[data-line]');
            const other = row?.querySelector(
                `[data-money="${el.dataset.money === 'debit' ? 'credit' : 'debit'}"]`,
            );
            if (other && parseFloat(el.value)) other.value = '';
        });
    });
}

/**
 * Tukar sisi debit dan kredit seluruh baris dalam satu jurnal.
 *
 * Arah transaksi adalah kesalahan paling sering saat AI membaca screenshot
 * mutasi: bukti transfer tidak selalu menunjukkan siapa pengirimnya. Daripada
 * mengetik ulang semua nominal, satu klik membalik seluruh blok. Jurnal tetap
 * balance karena total kedua sisi hanya bertukar tempat.
 */
function wireReverse() {
    document.querySelectorAll('[data-reverse]').forEach((btn) => {
        btn.addEventListener('click', () => {
            const root = btn.closest('[data-journal]');
            if (!root) return;

            root.querySelectorAll('[data-line]').forEach((row) => {
                const debit = row.querySelector('[data-money="debit"]');
                const credit = row.querySelector('[data-money="credit"]');
                if (!debit || !credit) return;
                [debit.value, credit.value] = [credit.value, debit.value];
            });

            // Hitung ulang total & status balance setelah nilai bertukar.
            root.querySelectorAll('[data-money]').forEach((el) =>
                el.dispatchEvent(new Event('input', { bubbles: true })),
            );
        });
    });
}

/** Tambah baris kosong di form jurnal manual. */
function wireAddLine() {
    document.querySelectorAll('[data-add-line]').forEach((btn) => {
        btn.addEventListener('click', () => {
            const body = document.querySelector(btn.dataset.addLine);
            const template = body?.querySelector('[data-line]');
            if (!body || !template) return;

            const clone = template.cloneNode(true);
            const index = body.querySelectorAll('[data-line]').length;
            clone.querySelectorAll('[name]').forEach((el) => {
                el.name = el.name.replace(/\[\d+\]/, `[${index}]`);
                if (el.tagName === 'SELECT') el.selectedIndex = 0;
                else el.value = '';
            });
            body.appendChild(clone);

            const form = body.closest('[data-journal]');
            if (form) {
                wireBalance(form);
                wireExclusiveSides(form);
            }
        });
    });
}


/** Sidebar mobile: buka/tutup drawer. Desktop sidebar selalu tampil lewat CSS. */
function wireNav() {
    const shell = document.querySelector('[data-shell]');
    if (!shell) return;
    document.querySelectorAll('[data-nav-open]').forEach((el) =>
        el.addEventListener('click', () => shell.setAttribute('data-open', '')),
    );
    document.querySelectorAll('[data-nav-close]').forEach((el) =>
        el.addEventListener('click', () => shell.removeAttribute('data-open')),
    );
    document.addEventListener('keydown', (e) => e.key === 'Escape' && shell.removeAttribute('data-open'));
}

/**
 * Penanda "AI" berubah jadi "dikoreksi" begitu manusia mengubah baris itu —
 * supaya jelas angka mana yang belum pernah dilihat orang.
 */
function wireAiMarkers() {
    document.querySelectorAll('[data-line]').forEach((row) => {
        const marker = row.querySelector('[data-ai-marker]');
        if (!marker) return;
        const mark = () => {
            marker.textContent = 'dikoreksi';
            marker.className = 'badge-mute';
            marker.nextElementSibling?.remove();
        };
        row.querySelectorAll('input, select').forEach((el) => el.addEventListener('change', mark, { once: true }));
    });
}

/**
 * Ledger Seam: total semua blok yang dicentang "Posting". Tombol posting
 * dimatikan selama ada blok yang tidak balance. Hanya UX — JournalPoster
 * tetap menolak di server.
 */
function wireSeam() {
    const seam = document.querySelector('[data-seam]');
    const form = seam?.closest('form');
    if (!form) return;
    const out = {
        debit: seam.querySelector('[data-seam-debit]'),
        credit: seam.querySelector('[data-seam-credit]'),
        state: seam.querySelector('[data-seam-state]'),
        submit: seam.querySelector('[data-seam-submit]'),
    };

    const recalc = () => {
        let debit = 0;
        let credit = 0;
        let bad = 0;
        let picked = 0;
        form.querySelectorAll('[data-journal]').forEach((block) => {
            if (!block.querySelector('input[name$="[post]"]')?.checked) return;
            picked++;
            let d = 0;
            let c = 0;
            block.querySelectorAll('[data-money]').forEach((el) => {
                const v = parseFloat(el.value) || 0;
                if (el.dataset.money === 'debit') d += v;
                else c += v;
            });
            if (Math.abs(Math.round((d - c) * 100)) > 0) bad++;
            debit += d;
            credit += c;
        });

        out.debit.textContent = rupiah(debit);
        out.credit.textContent = rupiah(credit);
        if (!picked) {
            out.state.textContent = 'Tidak ada blok dicentang';
            out.state.className = 'badge-mute';
        } else if (bad) {
            out.state.textContent = `${bad} blok tidak balance · selisih Rp${rupiah(Math.abs(debit - credit))}`;
            out.state.className = 'badge-bad';
        } else {
            out.state.textContent = `Balance · ${picked} jurnal`;
            out.state.className = 'badge-ok';
        }
        out.submit.disabled = !picked || bad > 0;
    };

    form.addEventListener('input', recalc);
    form.addEventListener('change', recalc);
    recalc();
}

/** Tombol mata di field password (login). */
function wireReveal() {
    document.querySelectorAll('[data-reveal]').forEach((btn) => {
        const input = document.getElementById(btn.dataset.reveal);
        btn.addEventListener('click', () => {
            const show = input.type === 'password';
            input.type = show ? 'text' : 'password';
            btn.setAttribute('aria-pressed', String(show));
            btn.setAttribute('aria-label', show ? 'Sembunyikan password' : 'Tampilkan password');
            btn.querySelectorAll('svg').forEach((svg) => svg.classList.toggle('hidden'));
        });
    });
}

/**
 * Konfirmasi lewat <dialog> untuk form/tombol ber-data-confirm. Tanpa JS,
 * form tetap terkirim (server tetap menjaga: void bukan hapus, dst.).
 */
function wireConfirm() {
    const dialog = document.getElementById('confirm-dialog');
    if (!dialog) return;
    document.addEventListener('submit', (e) => {
        const form = e.target;
        const source = e.submitter?.dataset.confirm ? e.submitter : form;
        if (!source.dataset?.confirm || form.dataset.confirmed) return;
        e.preventDefault();
        dialog.querySelector('[data-confirm-message]').textContent = source.dataset.confirm;
        dialog.returnValue = '';
        dialog.showModal();
        dialog.addEventListener('close', () => {
            if (dialog.returnValue !== 'ok') return;
            form.dataset.confirmed = '1';
            form.requestSubmit(e.submitter ?? undefined);
        }, { once: true });
    });
}

function wireToast() {
    document.querySelectorAll('[data-toast]').forEach((toast) => {
        const hide = () => { toast.style.opacity = '0'; setTimeout(() => toast.remove(), 300); };
        toast.querySelector('[data-toast-close]')?.addEventListener('click', hide);
        setTimeout(hide, 6000);
    });
}

/** Palet ⌘K / Ctrl+K: saring daftar tautan, Enter = buka yang pertama/terpilih. */
function wirePalette() {
    const dialog = document.getElementById('palette');
    if (!dialog) return;
    const input = dialog.querySelector('[data-palette-input]');
    const items = [...dialog.querySelectorAll('[data-palette-item]')];
    let active = 0;
    const visible = () => items.filter((a) => !a.parentElement.hidden);
    const mark = () => visible().forEach((a, i) => a.setAttribute('aria-selected', String(i === active)));
    const filter = () => {
        const q = input.value.toLowerCase().trim();
        items.forEach((a) => { a.parentElement.hidden = q !== '' && !a.textContent.toLowerCase().includes(q); });
        active = 0;
        mark();
    };
    const open = () => { input.value = ''; filter(); dialog.showModal(); input.focus(); };

    document.addEventListener('keydown', (e) => {
        if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') { e.preventDefault(); open(); }
    });
    document.querySelectorAll('[data-palette-open]').forEach((b) => b.addEventListener('click', open));
    input.addEventListener('input', filter);
    input.addEventListener('keydown', (e) => {
        const list = visible();
        if (e.key === 'ArrowDown') { e.preventDefault(); active = Math.min(active + 1, list.length - 1); mark(); list[active]?.scrollIntoView({ block: 'nearest' }); }
        if (e.key === 'ArrowUp') { e.preventDefault(); active = Math.max(active - 1, 0); mark(); list[active]?.scrollIntoView({ block: 'nearest' }); }
        if (e.key === 'Enter' && list[active]) { e.preventDefault(); window.location = list[active].href; }
    });
    dialog.addEventListener('click', (e) => e.target === dialog && dialog.close());
}

/** Dropzone + paste screenshot ke input file upload dokumen. */
function wireDropzone() {
    const zone = document.querySelector('[data-dropzone]');
    const input = zone?.querySelector('input[type=file]');
    if (!zone || !input) return;
    const label = zone.querySelector('[data-dropzone-label]');
    const take = (files) => {
        if (!files?.length) return;
        const dt = new DataTransfer();
        dt.items.add(files[0]);
        input.files = dt.files;
        label.textContent = `Siap diupload: ${files[0].name || 'gambar dari clipboard'}`;
    };
    input.addEventListener('change', () => input.files.length && (label.textContent = `Siap diupload: ${input.files[0].name}`));
    zone.addEventListener('dragover', (e) => { e.preventDefault(); zone.setAttribute('data-over', ''); });
    zone.addEventListener('dragleave', () => zone.removeAttribute('data-over'));
    zone.addEventListener('drop', (e) => { e.preventDefault(); zone.removeAttribute('data-over'); take(e.dataTransfer.files); });
    document.addEventListener('paste', (e) => {
        const files = [...(e.clipboardData?.files ?? [])].filter((f) => f.type.startsWith('image/') || f.type === 'application/pdf');
        if (files.length && e.target.tagName !== 'TEXTAREA') { e.preventDefault(); take(files); }
    });
}

/**
 * Form pengguna: hitung perusahaan terpilih vs batas peran, kunci centang
 * sisanya saat batas tercapai. Penjaga sebenarnya tetap validasi di server.
 */
function wireClientLimit() {
    const picker = document.querySelector('[data-client-picker]');
    if (!picker) return;
    const limits = JSON.parse(picker.dataset.limits || '{}');
    const boxes = [...picker.querySelectorAll('input[type=checkbox]')];
    const counter = picker.querySelector('[data-client-count]');
    const update = () => {
        const role = document.querySelector('[data-role-radio]:checked')?.value;
        const limit = limits[role];
        const picked = boxes.filter((b) => b.checked).length;
        picker.disabled = limit === undefined; // admin: semua klien, tidak perlu memilih
        boxes.forEach((b) => { b.disabled = limit !== undefined && !b.checked && picked >= limit; });
        if (counter) {
            counter.textContent = limit === undefined ? '' : `${picked}/${limit} dipilih`;
            counter.classList.toggle('text-rose-700', limit !== undefined && picked > limit);
        }
    };
    picker.addEventListener('change', update);
    document.querySelectorAll('[data-role-radio]').forEach((r) => r.addEventListener('change', update));
    update();
}

document.addEventListener('DOMContentLoaded', () => {
    document.querySelectorAll('[data-journal]').forEach((root) => {
        wireBalance(root);
        wireExclusiveSides(root);
    });
    wireAddLine();
    wireReverse();
    wireNav();
    wireAiMarkers();
    wireSeam();
    wireReveal();
    wireConfirm();
    wireToast();
    wirePalette();
    wireDropzone();
    wireClientLimit();
});
