// Panen metadata Primbon -> manifest JSON.
//
// JALANKAN DI KONSOL TAB primbonpajak.id YANG SUDAH LOGIN.
// Halaman daftar butuh sesi, jadi langkah ini tidak bisa lewat curl.
//
// Yang diambil hanya metadata faktual + slug JDIH. Ringkasan/Catatan/Bookmark
// TIDAK diambil (berhak cipta CV Abadi Tax Indonesia).
//
// Pemakaian:
//   await panen()                      // semua kategori
//   await panen(['FAKTUR','PPS'])      // kategori tertentu
// Hasil: unduh manifest-primbon.json, pindahkan ke scripts/peraturan/.

const KATEGORI = [
  'PPN','PPnBM','PPh','KUP','PBB','BEA_CUKAI','PAJAK_DAERAH','UMKM','FAKTUR',
  'NPWP','PPS','CARBON_TAX','TRANSFER_PRICING','TINDAK_PIDANA','SENGKETA_PAJAK',
  'KURS','KEUNEG','PENG_SP_DJP','BEA_MATERAI',
];

const tidur = (ms) => new Promise((r) => setTimeout(r, ms));

// Satu kartu daftar -> {uuid, kode, judul, jenis, status, tahun}
function parseKartu(html) {
  const out = [];
  const re = /<a class="text-lg font-semibold[^"]*" href="\/regulations\/([0-9a-f-]{36})"><span class="font-mono[^"]*">([^<]*)<\/span><span>—\s*(?:<!--\s*-->)?([^<]*)<\/span><\/a>/g;
  let m;
  while ((m = re.exec(html))) {
    const sebelum = html.slice(Math.max(0, m.index - 1200), m.index);
    const badges = [...sebelum.matchAll(/text-\[10px\]">([^<]+)</g)].map((b) => b[1].trim());
    const thn = sebelum.match(/·\s*(?:<!--\s*-->)?(\d{4})\s*<\/span>\s*<\/div>\s*$/);
    out.push({
      uuid: m[1],
      kode: m[2].trim(),
      judul: m[3].replace(/\s+/g, ' ').trim(),
      jenis: badges[0] || '',
      status: badges[1] || '',
      tahun: thn ? thn[1] : '',
    });
  }
  return out;
}

async function ambilHalaman(kat, page) {
  const r = await fetch(`/regulations?kategori=${kat}&sort=terbaru&page=${page}`);
  if (!r.ok) throw new Error(`${kat} p${page}: HTTP ${r.status}`);
  const t = await r.text();
  if (t.includes('/login?next=')) throw new Error('sesi login habis, login ulang');
  const pg = t.match(/dari <!-- -->(\d+)/);
  return { kartu: parseKartu(t), totalHalaman: pg ? +pg[1] : 1 };
}

// slug JDIH dari halaman detail (URL PDF-nya diambil nanti dari JDIH, tanpa login)
async function ambilSlug(uuid) {
  const t = await (await fetch(`/regulations/${uuid}`)).text();
  // Payload RSC Next.js bisa memotong string di batas chunk ("pmk-148-t"),
  // jadi ambil kecocokan terpanjang, bukan yang pertama.
  const m = [...t.matchAll(/jdih\.kemenkeu\.go\.id\/dok\/([a-z0-9-]+)/g)].map((x) => x[1]);
  return m.length ? m.reduce((a, b) => (b.length > a.length ? b : a)) : null;
}

async function panen(kategori = KATEGORI, jedaMs = 400) {
  const dok = new Map(); // uuid -> record, digabung antar kategori

  for (const kat of kategori) {
    let page = 1, total = 1;
    do {
      const { kartu, totalHalaman } = await ambilHalaman(kat, page);
      total = totalHalaman;
      for (const k of kartu) {
        if (dok.has(k.uuid)) {
          const d = dok.get(k.uuid);
          if (!d.kategori.includes(kat)) d.kategori.push(kat);
        } else {
          dok.set(k.uuid, { ...k, kategori: [kat], jdih_slug: null });
        }
      }
      console.log(`${kat} hal ${page}/${total} — ${kartu.length} kartu, unik total ${dok.size}`);
      page++;
      await tidur(jedaMs);
    } while (page <= total);
  }

  // slug JDIH, hanya untuk yang belum punya
  const perlu = [...dok.values()].filter((d) => !d.jdih_slug);
  for (let i = 0; i < perlu.length; i++) {
    try {
      perlu[i].jdih_slug = await ambilSlug(perlu[i].uuid);
    } catch (e) {
      perlu[i].jdih_slug = null;
    }
    if (i % 25 === 0) console.log(`slug ${i + 1}/${perlu.length}`);
    await tidur(jedaMs);
  }

  const hasil = {
    dipanen: new Date().toISOString(),
    totalUnik: dok.size,
    tanpaSlugJdih: [...dok.values()].filter((d) => !d.jdih_slug).length,
    dokumen: [...dok.values()],
  };

  const blob = new Blob([JSON.stringify(hasil, null, 1)], { type: 'application/json' });
  const a = document.createElement('a');
  a.href = URL.createObjectURL(blob);
  a.download = 'manifest-primbon.json';
  a.click();
  console.log(`selesai: ${hasil.totalUnik} unik, ${hasil.tanpaSlugJdih} tanpa slug JDIH`);
  return hasil;
}
