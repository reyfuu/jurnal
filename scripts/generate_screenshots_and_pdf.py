import os
import sys
import time
import subprocess
from fpdf import FPDF

SCREENSHOTS_DIR = os.path.join(os.getcwd(), "docs", "screenshots")
os.makedirs(SCREENSHOTS_DIR, exist_ok=True)

def capture_screenshots():
    from playwright.sync_api import sync_playwright
    print("=== [1/3] Menjalankan server lokal di port 8088 ===")
    env = os.environ.copy()
    env["PORT"] = "8088"
    proc = subprocess.Popen(["./bin/laporan-bisnis"], env=env)
    time.sleep(1.5)

    try:
        print("=== [2/3] Mengambil tangkapan layar via Playwright ===")
        with sync_playwright() as p:
            browser = p.chromium.launch(headless=True)
            context = browser.new_context(viewport={"width": 1440, "height": 920})
            page = context.new_page()

            # 1. Login Page
            page.goto("http://localhost:8088")
            page.wait_for_selector("input[type='email']")
            time.sleep(0.5)
            login_path = os.path.join(SCREENSHOTS_DIR, "01_login.png")
            page.screenshot(path=login_path)
            print("✓ Disimpan:", login_path)

            # Login as Owner
            page.fill("input[type='email']", "owner@kedaikopisenja.id")
            page.fill("input[type='password']", "senja2026")
            page.click("button[type='submit']")

            # 2. Dashboard
            page.wait_for_selector(".app-shell")
            time.sleep(1.5)
            dash_path = os.path.join(SCREENSHOTS_DIR, "02_dashboard.png")
            page.screenshot(path=dash_path)
            print("✓ Disimpan:", dash_path)

            # 3. Laporan Keuangan
            page.click("button:has-text('Laporan Keuangan')")
            time.sleep(1.0)
            rep_path = os.path.join(SCREENSHOTS_DIR, "03_laporan_keuangan.png")
            page.screenshot(path=rep_path)
            print("✓ Disimpan:", rep_path)

            # 4. Simulasi & Harga
            page.click("button:has-text('Simulasi & Harga')")
            time.sleep(1.0)
            sim_path = os.path.join(SCREENSHOTS_DIR, "04_simulasi_harga.png")
            page.screenshot(path=sim_path)
            print("✓ Disimpan:", sim_path)

            # 5. Pajak UMKM
            page.click("button:has-text('Pajak UMKM')")
            time.sleep(1.0)
            tax_path = os.path.join(SCREENSHOTS_DIR, "05_pajak_umkm.png")
            page.screenshot(path=tax_path)
            print("✓ Disimpan:", tax_path)

            # 6. AI Agent Bubble
            # Buka kembali dashboard lalu klik bubble
            page.click("button:has-text('Dashboard')")
            time.sleep(0.8)
            page.click("button:has-text('Tanya AI Pajak')")
            time.sleep(0.8)
            # Klik tombol ringkas laporan
            page.click("button:has-text('Ringkas laporan bulan ini')")
            time.sleep(2.0)
            ai_path = os.path.join(SCREENSHOTS_DIR, "06_ai_bubble.png")
            page.screenshot(path=ai_path)
            print("✓ Disimpan:", ai_path)

            browser.close()
    finally:
        proc.terminate()
        proc.wait()
        print("Server lokal port 8088 dimatikan.")

class PDFGuide(FPDF):
    def header(self):
        if self.page_no() > 1:
            self.set_font("Helvetica", "B", 8)
            self.set_text_color(120, 140, 160)
            self.cell(0, 8, "PANDUAN APLIKASI LAPORAN BISNIS - BY MASTER PAJAK", align="L")
            self.cell(0, 8, f"Hal. {self.page_no()}", align="R", new_x="LMARGIN", new_y="NEXT")
            self.set_draw_color(30, 41, 59)
            self.set_line_width(0.3)
            self.line(10, 16, 200, 16)
            self.ln(5)

    def footer(self):
        if self.page_no() > 1:
            self.set_y(-12)
            self.set_font("Helvetica", "", 8)
            self.set_text_color(100, 116, 139)
            self.cell(0, 6, "Sistem Keuangan & Pajak UMKM Terpadu - https://pajak.aipreneur.co.id", align="C")

def build_pdf():
    print("=== [3/3] Menyusun Dokumen PDF Panduan Lengkap ===")
    pdf = PDFGuide(orientation="P", unit="mm", format="A4")
    pdf.set_auto_page_break(auto=True, margin=15)

    # --- COVER PAGE ---
    pdf.add_page()
    pdf.set_fill_color(10, 15, 23) # Dark Obsidian
    pdf.rect(0, 0, 210, 297, "F")

    # Brand Title
    pdf.set_y(55)
    pdf.set_font("Helvetica", "B", 26)
    pdf.set_text_color(16, 185, 129) # Emerald Green
    pdf.cell(0, 12, "LAPORAN BISNIS", align="C", new_x="LMARGIN", new_y="NEXT")
    
    pdf.set_font("Helvetica", "", 13)
    pdf.set_text_color(148, 163, 184) # Slate Muted
    pdf.cell(0, 8, "by Master Pajak", align="C", new_x="LMARGIN", new_y="NEXT")

    pdf.ln(12)
    pdf.set_draw_color(16, 185, 129)
    pdf.set_line_width(0.8)
    pdf.line(75, pdf.get_y(), 135, pdf.get_y())
    pdf.ln(14)

    # Document Title
    pdf.set_font("Helvetica", "B", 18)
    pdf.set_text_color(248, 250, 252) # White Slate
    pdf.cell(0, 10, "Buku Panduan Penggunaan Aplikasi", align="C", new_x="LMARGIN", new_y="NEXT")
    
    pdf.set_font("Helvetica", "", 11)
    pdf.set_text_color(203, 213, 225)
    pdf.cell(0, 6, "Sistem Akuntansi, Arus Kas, Simulasi Harga & Kepatuhan Pajak PP 55/2022", align="C", new_x="LMARGIN", new_y="NEXT")
    pdf.cell(0, 6, "Dilengkapi Fitur AI Agent Konsultan Bisnis & Pajak", align="C", new_x="LMARGIN", new_y="NEXT")

    # Cover Metadata Box
    pdf.set_y(175)
    pdf.set_fill_color(19, 27, 39)
    pdf.set_draw_color(51, 65, 85)
    pdf.rect(35, 175, 140, 52, "FD")

    pdf.set_xy(40, 182)
    pdf.set_font("Helvetica", "B", 9)
    pdf.set_text_color(16, 185, 129)
    pdf.cell(50, 6, "VERSI APLIKASI:", align="L")
    pdf.set_font("Helvetica", "", 9)
    pdf.set_text_color(241, 245, 249)
    pdf.cell(70, 6, "v1.2.0 (Fase 3 + AI Agent)", align="L", new_x="LMARGIN", new_y="NEXT")

    pdf.set_x(40)
    pdf.set_font("Helvetica", "B", 9)
    pdf.set_text_color(16, 185, 129)
    pdf.cell(50, 6, "URL PRODUKSI:", align="L")
    pdf.set_font("Helvetica", "", 9)
    pdf.set_text_color(56, 189, 248) # Sky blue
    pdf.cell(70, 6, "https://pajak.aipreneur.co.id", align="L", new_x="LMARGIN", new_y="NEXT")

    pdf.set_x(40)
    pdf.set_font("Helvetica", "B", 9)
    pdf.set_text_color(16, 185, 129)
    pdf.cell(50, 6, "TARGET AUDIENCE:", align="L")
    pdf.set_font("Helvetica", "", 9)
    pdf.set_text_color(241, 245, 249)
    pdf.cell(70, 6, "100 UMKM Batch Pertama (FnB, Retail, Jasa)", align="L", new_x="LMARGIN", new_y="NEXT")

    pdf.set_x(40)
    pdf.set_font("Helvetica", "B", 9)
    pdf.set_text_color(16, 185, 129)
    pdf.cell(50, 6, "TANGGAL TERBIT:", align="L")
    pdf.set_font("Helvetica", "", 9)
    pdf.set_text_color(241, 245, 249)
    pdf.cell(70, 6, "Oktober 2026", align="L", new_x="LMARGIN", new_y="NEXT")

    # --- PAGE 2: DAFTAR ISI & PENDAHULUAN ---
    pdf.add_page()
    pdf.set_text_color(15, 23, 42)
    pdf.set_font("Helvetica", "B", 18)
    pdf.cell(0, 10, "1. Pendahuluan & Filosofi Sistem", new_x="LMARGIN", new_y="NEXT")
    pdf.set_font("Helvetica", "", 10)
    pdf.set_text_color(51, 65, 85)
    pdf.multi_cell(0, 5.5, 
        "Aplikasi Laporan Bisnis by Master Pajak dirancang khusus untuk memecahkan tiga masalah terbesar pelaku UMKM Indonesia:\n"
        "1. Uang tunai sering habis di akhir bulan meskipun omzet terlihat besar (masalah rekonsiliasi kas vs laba).\n"
        "2. Kebingungan saat menetapkan harga jual di kanal ojek online (GoFood/Grab/Shopee) yang menyebabkan margin tergerus komisi 20%.\n"
        "3. Ketakutan terhadap sanksi dan denda pajak akibat minimnya pemahaman aturan PPh Final UMKM PP 55 Tahun 2022.\n\n"
        "Sistem ini menyajikan analitik bergaya Dark Obsidian Fintech Bento Grid yang bersih, presisi hingga satuan Rupiah, dan dilengkapi asisten cerdas AI Agent Konsultan Pajak & Bisnis."
    )
    pdf.ln(4)

    pdf.set_font("Helvetica", "B", 14)
    pdf.set_text_color(15, 23, 42)
    pdf.cell(0, 8, "Hak Akses & Role Pengguna (RBAC)", new_x="LMARGIN", new_y="NEXT")
    pdf.set_font("Helvetica", "", 9.5)
    pdf.set_text_color(51, 65, 85)
    pdf.multi_cell(0, 5,
        "Untuk keamanan data bisnis Anda, sistem menyediakan 3 role pengguna:\n"
        "- Owner (Pemilik): Akses penuh ke seluruh menu finansial, laporan neraca, simulasi harga, serta pengelolaan pengguna & staf kasir.\n"
        "- Admin Keuangan: Akses ke seluruh laporan operasional, arus kas, piutang, dan pajak (tanpa izin mengelola akun pengguna lain).\n"
        "- Kasir: Akses terbatas hanya untuk memantau ringkasan harian dan menginput transaksi kasir."
    )
    pdf.ln(4)

    # Akun Demo Table
    pdf.set_fill_color(241, 245, 249)
    pdf.set_draw_color(203, 213, 225)
    pdf.set_font("Helvetica", "B", 9)
    pdf.cell(40, 7, "Role", 1, 0, "C", True)
    pdf.cell(75, 7, "Email Akun Demo", 1, 0, "C", True)
    pdf.cell(40, 7, "Password", 1, 0, "C", True)
    pdf.cell(35, 7, "Hak Akses", 1, 1, "C", True)

    pdf.set_font("Helvetica", "", 8.5)
    pdf.cell(40, 6, "Owner (Pemilik)", 1, 0, "L")
    pdf.cell(75, 6, "owner@kedaikopisenja.id", 1, 0, "L")
    pdf.cell(40, 6, "senja2026", 1, 0, "C")
    pdf.cell(35, 6, "Akses Penuh", 1, 1, "C")

    pdf.cell(40, 6, "Admin Keuangan", 1, 0, "L")
    pdf.cell(75, 6, "admin@kedaikopisenja.id", 1, 0, "L")
    pdf.cell(40, 6, "senja2026", 1, 0, "C")
    pdf.cell(35, 6, "Finansial & Pajak", 1, 1, "C")

    pdf.cell(40, 6, "Kasir", 1, 0, "L")
    pdf.cell(75, 6, "kasir@kedaikopisenja.id", 1, 0, "L")
    pdf.cell(40, 6, "senja2026", 1, 0, "C")
    pdf.cell(35, 6, "Kasir & Transaksi", 1, 1, "C")

    # --- PAGE 3: PANDUAN LOGIN & DASHBOARD ---
    pdf.add_page()
    pdf.set_font("Helvetica", "B", 16)
    pdf.cell(0, 9, "2. Panduan Masuk (Login) & Navigasi", new_x="LMARGIN", new_y="NEXT")
    pdf.set_font("Helvetica", "", 9.5)
    pdf.multi_cell(0, 5,
        "Akses alamat https://pajak.aipreneur.co.id melalui browser. Masukkan email dan password akun Anda, lalu klik tombol 'Masuk ke Sistem'."
    )
    pdf.ln(2)

    login_img = os.path.join(SCREENSHOTS_DIR, "01_login.png")
    if os.path.exists(login_img):
        pdf.image(login_img, x=20, y=pdf.get_y(), w=170)
        pdf.set_y(pdf.get_y() + 105)

    pdf.set_font("Helvetica", "B", 16)
    pdf.cell(0, 9, "3. Anatomi Dashboard Utama (Bento Grid)", new_x="LMARGIN", new_y="NEXT")
    pdf.set_font("Helvetica", "", 9.5)
    pdf.multi_cell(0, 5,
        "Dashboard menyajikan 6 KPI Finansial Utama secara real-time:\n"
        "1. Omzet Penjualan (Gross Sales) & tren pertumbuhan bulan berjalan.\n"
        "2. Laba Bersih (Net Profit) beserta Net Margin Bps.\n"
        "3. Margin Laba Bersih terhadap seluruh biaya operasional.\n"
        "4. Saldo Kas Riil (Liquid Cash) yang siap digunakan.\n"
        "5. Pencapaian Target Omzet Bulanan.\n"
        "6. Kontrol Piutang & Utang Berjalan (termasuk deteksi piutang >30 hari)."
    )

    # --- PAGE 4: DASHBOARD SCREENSHOT ---
    pdf.add_page()
    dash_img = os.path.join(SCREENSHOTS_DIR, "02_dashboard.png")
    if os.path.exists(dash_img):
        pdf.image(dash_img, x=15, y=pdf.get_y(), w=180)
        pdf.set_y(pdf.get_y() + 115)

    pdf.set_font("Helvetica", "B", 13)
    pdf.cell(0, 8, "Grafik Analisis Tengah & Transaksi Kasir", new_x="LMARGIN", new_y="NEXT")
    pdf.set_font("Helvetica", "", 9.5)
    pdf.multi_cell(0, 5,
        "- Grafik Tren Omzet vs Biaya: Membandingkan pendapatan terhadap biaya secara visual.\n"
        "- Breakdown Komisi Aplikasi Online: Melihat proporsi potongan fee GoFood, GrabFood, dan ShopeeFood.\n"
        "- Arus Kas Bulanan: Memantau likuiditas kas operasional masuk dan keluar.\n"
        "- Peringatan Resep BOM & Stok: Peringatan dini bahan baku yang menipis (contoh: Fresh Milk)."
    )

    # --- PAGE 5: LAPORAN KEUANGAN & NERACA ---
    pdf.add_page()
    pdf.set_font("Helvetica", "B", 16)
    pdf.cell(0, 9, "4. Laporan Keuangan Terpadu (Laba Rugi & Neraca)", new_x="LMARGIN", new_y="NEXT")
    pdf.set_font("Helvetica", "", 9.5)
    pdf.multi_cell(0, 5,
        "Menu Laporan Keuangan menyajikan Laba Rugi berdampingan dengan Neraca (Balance Sheet). Sistem secara otomatis menguji validitas persamaan akuntansi:\n"
        "ASET (Kas + Piutang + Persediaan) = KEWAJIBAN (Utang) + MODAL (Modal Awal + Laba Berjalan).\n"
        "Jika kedua sisi cocok, sistem memberikan badge hijau 'Aktiva = Pasiva (Seimbang)'."
    )
    pdf.ln(2)

    rep_img = os.path.join(SCREENSHOTS_DIR, "03_laporan_keuangan.png")
    if os.path.exists(rep_img):
        pdf.image(rep_img, x=15, y=pdf.get_y(), w=180)
        pdf.set_y(pdf.get_y() + 115)

    pdf.set_font("Helvetica", "B", 13)
    pdf.cell(0, 8, "Prinsip Rekonsiliasi Kas (Invarian Kas)", new_x="LMARGIN", new_y="NEXT")
    pdf.set_font("Helvetica", "", 9.5)
    pdf.multi_cell(0, 5,
        "Formula Kunci Master Pajak: Kenaikan Kas = Laba Bersih - Piutang Baru + Utang Baru.\n"
        "Dengan formula ini, pemilik bisnis dapat segera mengetahui kemana laba mereka pergi jika uang kas di bank tidak bertambah sesuai laba bersih."
    )

    # --- PAGE 6: SIMULASI HARGA & PAJAK PP 55 ---
    pdf.add_page()
    pdf.set_font("Helvetica", "B", 16)
    pdf.cell(0, 9, "5. Kalkulator Simulasi Harga Komisi Aplikasi", new_x="LMARGIN", new_y="NEXT")
    pdf.set_font("Helvetica", "", 9.5)
    pdf.multi_cell(0, 5,
        "Gunakan kalkulator ini untuk menentukan harga jual produk di GoFood/GrabFood agar komisi 20% tidak memakan profit margin Anda. Sistem menggunakan rumus Gross-up: Harga = (HPP + Laba) / (1 - %Komisi)."
    )
    pdf.ln(2)

    sim_img = os.path.join(SCREENSHOTS_DIR, "04_simulasi_harga.png")
    if os.path.exists(sim_img):
        pdf.image(sim_img, x=15, y=pdf.get_y(), w=180)
        pdf.set_y(pdf.get_y() + 115)

    pdf.set_font("Helvetica", "B", 16)
    pdf.cell(0, 9, "6. Kepatuhan Pajak UMKM (PP 55 Tahun 2022)", new_x="LMARGIN", new_y="NEXT")
    pdf.set_font("Helvetica", "", 9.5)
    pdf.multi_cell(0, 5,
        "Sistem secara otomatis melacak akumulasi omzet tahunan dan menghitung hak bebas pajak Rp500.000.000 untuk Wajib Pajak Orang Pribadi. Anda tidak perlu membayar PPh Final 0,5% sebelum omzet menembus batas tersebut."
    )

    # --- PAGE 7: PAJAK UMKM SCREENSHOT & AI AGENT ---
    pdf.add_page()
    tax_img = os.path.join(SCREENSHOTS_DIR, "05_pajak_umkm.png")
    if os.path.exists(tax_img):
        pdf.image(tax_img, x=15, y=pdf.get_y(), w=180)
        pdf.set_y(pdf.get_y() + 115)

    pdf.set_font("Helvetica", "B", 16)
    pdf.cell(0, 9, "7. AI Agent Konsultan Bisnis & Pajak (Floating Bubble)", new_x="LMARGIN", new_y="NEXT")
    pdf.set_font("Helvetica", "", 9.5)
    pdf.multi_cell(0, 5,
        "Tombol melayang 'Tanya AI Pajak' di pojok kanan bawah dapat dibuka kapan saja. AI Agent terintegrasi langsung dengan database pembukuan dan engine regulasi perpajakan Master Pajak."
    )

    # --- PAGE 8: AI AGENT BUBBLE SCREENSHOT & DETAIL ---
    pdf.add_page()
    ai_img = os.path.join(SCREENSHOTS_DIR, "06_ai_bubble.png")
    if os.path.exists(ai_img):
        pdf.image(ai_img, x=25, y=pdf.get_y(), w=160)
        pdf.set_y(pdf.get_y() + 120)

    pdf.set_font("Helvetica", "B", 13)
    pdf.cell(0, 8, "Kemampuan Utama AI Agent Konsultan:", new_x="LMARGIN", new_y="NEXT")
    pdf.set_font("Helvetica", "", 9.5)
    pdf.multi_cell(0, 5,
        "1. Meringkas Laporan Eksekutif: Memberikan rangkuman omzet, laba kotor, HPP, OPEX, laba bersih, dan saldo kas dalam 1 klik.\n"
        "2. Menjawab Regulasi Pajak PP 55/2022: Menjelaskan ambang batas Rp500 juta, batas waktu fasilitas tarif 0,5% (7 th untuk OP, 4 th CV, 3 th PT), serta kewajiban pelaporan SPT tahunan.\n"
        "3. Peringatan Dini Piutang: Mengingatkan piutang saluran online yang tertahan lebih dari 30 hari (contoh: ShopeePay Rp11 jt).\n"
        "4. Saran Aksi Cepat: Menyediakan tombol rekomendasi otomatis di akhir setiap jawaban."
    )

    pdf_out_path = os.path.join(os.getcwd(), "docs", "Panduan_Laporan_Bisnis_Master_Pajak.pdf")
    pdf.output(pdf_out_path)
    print("==================================================")
    print("✓ DOKUMEN PDF BERHASIL DIBUAT:")
    print("  Path:", pdf_out_path)
    print("==================================================")

if __name__ == "__main__":
    required = ["01_login.png", "02_dashboard.png", "03_laporan_keuangan.png", "04_simulasi_harga.png", "05_pajak_umkm.png", "06_ai_bubble.png"]
    if not all(os.path.exists(os.path.join(SCREENSHOTS_DIR, f)) for f in required):
        capture_screenshots()
    build_pdf()
