import { formatRp } from '@/lib/format'
import { angka } from '@/lib/angka'
import type { Langkah } from '@/lib/api'

// Potongan tampilan bersama kalkulator pajak: dua kolom (formulir | hasil), input Rupiah, hasil + langkah.

export function Dua({ kiri, kanan }: { kiri: React.ReactNode; kanan: React.ReactNode }) {
  return (
    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(min(330px, 100%), 1fr))', gap: 14 }}>
      <div className="card" style={{ padding: 20, justifyContent: 'flex-start' }}>{kiri}</div>
      <div className="card" style={{ padding: 20, justifyContent: 'flex-start' }}>
        {kanan ?? <p className="workspace-subtitle" style={{ margin: 0 }}>Isi formulir di sebelah kiri lalu tekan Hitung.</p>}
      </div>
    </div>
  )
}

export function Uang({ label, v, set, hint }: { label: string; v: number; set: (n: number) => void; hint?: string }) {
  return (
    <label style={{ display: 'block', marginBottom: 12 }}>
      <span className="kpi-label" style={{ display: 'block', marginBottom: 4 }}>{label}</span>
      <input className="input-field tabular" {...angka(v, set)} />
      <span style={{ display: 'block', fontSize: 11.5, color: 'var(--text-muted)', marginTop: 4 }}>
        {formatRp(v)}{hint ? ` · ${hint}` : ''}
      </span>
    </label>
  )
}

export function Hasil({ utama, langkah }: { utama: { label: string; v: number; warna?: string; persen?: boolean }[]; langkah: Langkah[] }) {
  return (
    <>
      <div className="chart-title">Hasil</div>
      <div style={{ marginTop: 12, marginBottom: 16 }}>
        {utama.map((u) => (
          <div key={u.label} style={{ display: 'flex', justifyContent: 'space-between', gap: 16, padding: '8px 0' }}>
            <span style={{ color: 'var(--text-muted)', fontSize: 13 }}>{u.label}</span>
            <span className="tabular" style={{ fontWeight: 700, fontSize: 16, color: u.warna }}>
              {u.persen ? `${u.v}`.replace('.', ',') + '%' : formatRp(u.v)}
            </span>
          </div>
        ))}
      </div>
      {langkah.length > 0 && <div className="chart-subtitle" style={{ marginBottom: 6 }}>Langkah perhitungan</div>}
      <div className="list-rows tabular">
        {langkah.map((l, i) => (
          <div key={i} style={{ display: 'flex', justifyContent: 'space-between', gap: 16, padding: '6px 0', fontSize: 13 }}>
            <span>
              {l.label}
              {l.keterangan && <span style={{ display: 'block', fontSize: 11, color: 'var(--text-muted)' }}>{l.keterangan}</span>}
            </span>
            <span style={{ whiteSpace: 'nowrap' }}>{formatRp(l.nilai)}</span>
          </div>
        ))}
      </div>
    </>
  )
}

export function Pilih({ label, v, set, opsi, hint }: { label: string; v: string; set: (s: string) => void; opsi: [string, string][]; hint?: string }) {
  return (
    <label style={{ display: 'block', marginBottom: 12 }}>
      <span className="kpi-label" style={{ display: 'block', marginBottom: 4 }}>{label}</span>
      <select className="input-field" value={v} onChange={(e) => set(e.target.value)}>
        {opsi.map(([val, lbl]) => <option key={val} value={val}>{lbl}</option>)}
      </select>
      {hint && <span style={{ display: 'block', fontSize: 11.5, color: 'var(--text-muted)', marginTop: 4 }}>{hint}</span>}
    </label>
  )
}

export function Persen({ label, v, set, hint }: { label: string; v: string; set: (s: string) => void; hint?: string }) {
  return (
    <label style={{ display: 'block', marginBottom: 12 }}>
      <span className="kpi-label" style={{ display: 'block', marginBottom: 4 }}>{label}</span>
      <input className="input-field tabular" inputMode="decimal" value={v} onChange={(e) => set(e.target.value.replace(/[^\d.,-]/g, ''))} />
      {hint && <span style={{ display: 'block', fontSize: 11.5, color: 'var(--text-muted)', marginTop: 4 }}>{hint}</span>}
    </label>
  )
}

export function Centang({ v, set, children }: { v: boolean; set: (b: boolean) => void; children: React.ReactNode }) {
  return (
    <label style={{ display: 'flex', gap: 9, alignItems: 'center', fontSize: 13, marginBottom: 14 }}>
      <input type="checkbox" checked={v} onChange={(e) => set(e.target.checked)} />
      <span>{children}</span>
    </label>
  )
}

/** salah: kesalahan isian yang terlihat sebelum dihitung (mis. laba > omzet); tombol dimatikan. */
export function TombolHitung({ busy, error, salah }: { busy: boolean; error: string; salah?: string }) {
  return (
    <>
      {salah && <p className="pill-status pill-red" role="alert" style={{ marginBottom: 10, whiteSpace: 'normal', borderRadius: 10, lineHeight: 1.45 }}>{salah}</p>}
      <button className="btn-cta" style={{ padding: '9px 22px', fontSize: 13 }} disabled={busy || !!salah}>{busy ? 'Menghitung…' : 'Hitung'}</button>
      {error && <p className="pill-status pill-red" role="alert" style={{ marginTop: 12, whiteSpace: 'normal', borderRadius: 10, lineHeight: 1.45 }}>{error}</p>}
    </>
  )
}
