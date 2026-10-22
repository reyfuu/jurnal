import React, { useState, useRef, useEffect } from 'react'
import { ArrowRight, Bot, Sparkles, X, Send, RefreshCw, MessageSquareQuote } from 'lucide-react'
import { askAIChat, type AIChatResponse, type AIIsiForm, type View } from '../lib/api'
import { Monogram } from '@/components/Logo'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { JUDUL_FORM, pratinjau, rencana } from '@/lib/aiForm'

interface Message {
  id: string
  sender: 'user' | 'assistant'
  text: string
  suggestions?: string[]
  tautan?: AIChatResponse['tautan']
  /** Usulan isian form; status: menunggu Jalankan (mode rencana), sudah diisi, diurungkan, batal. */
  aksi?: AIIsiForm
  status?: 'rencana' | 'diisi' | 'diurungkan' | 'batal'
  isiId?: number
}

// Mode rencana: AI menampilkan langkah & pratinjau lalu menunggu Jalankan. Disimpan per browser.
const KUNCI_MODE = 'ai-mode-rencana'
const bacaMode = () => { try { return localStorage.getItem(KUNCI_MODE) !== '0' } catch { return true } }

const INITIAL_SUGGESTIONS = [
  'Di mana saya bisa mengisi laporan keuangan?',
  'Ringkas laporan bulan ini',
  'Catat penjualan kopi Rp2.000.000 tunai hari ini',
  'Bagaimana aturan pajak UMKM PP 55/2022?',
  'Bagaimana kondisi arus kas dan piutang saya?',
  'Bagaimana cara hitung simulasi harga komisi GoFood?',
]

/** onIsiForm: buka & isi form tujuan, mengembalikan id isian; onUrungkan: kembalikan form seperti sebelum diisi. */
export const AIAgentBubble: React.FC<{
  period?: string
  onBuka?: (v: View, tab?: string) => void
  onIsiForm?: (a: AIIsiForm) => number
  onUrungkan?: (id: number, a: AIIsiForm) => void
}> = ({ period = '2026-08', onBuka, onIsiForm, onUrungkan }) => {
  const [isOpen, setIsOpen] = useState(false)
  const [modeRencana, setModeRencana] = useState(bacaMode)
  const gantiMode = (v: boolean) => {
    setModeRencana(v)
    try { localStorage.setItem(KUNCI_MODE, v ? '1' : '0') } catch { /* mode tetap berlaku sampai halaman dimuat ulang */ }
  }
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [messages, setMessages] = useState<Message[]>([
    {
      id: 'welcome',
      sender: 'assistant',
      text: 'Halo! Saya **AI Konsultan Bisnis & Pajak** dari *Master Pajak*.\n\nSaya siap membantu Anda **meringkas laporan keuangan**, menganalisis arus kas & piutang, serta menjelaskan **aturan perpajakan UMKM (PP 55/2022)** secara akurat.\n\nBingung menu? Tanyakan saja, mis. *di mana saya mengisi laporan keuangan?* — saya beri tombol langsung ke halamannya.\n\nSaya juga bisa **mengisi form** (transaksi, invoice, barang, pemasok), mis. *catat penjualan kopi 2 juta tunai hari ini*. Anda melihat pratinjaunya dulu, dan tetap Anda yang menekan Simpan.',
      suggestions: INITIAL_SUGGESTIONS,
    },
  ])

  const scrollRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (isOpen) {
      scrollRef.current?.scrollIntoView({ behavior: 'smooth' })
    }
  }, [messages, isOpen])

  const handleSend = async (queryText?: string) => {
    const textToSend = (queryText ?? input).trim()
    if (!textToSend || loading) return

    const userMsg: Message = {
      id: Date.now().toString(),
      sender: 'user',
      text: textToSend,
    }

    setMessages((prev) => [...prev, userMsg])
    setInput('')
    setLoading(true)

    try {
      const res: AIChatResponse = await askAIChat(textToSend, period)
      const assistantMsg: Message = {
        id: (Date.now() + 1).toString(),
        sender: 'assistant',
        text: res.answer,
        suggestions: res.action_suggestions,
        tautan: res.tautan,
      }
      if (res.aksi && onIsiForm) {
        // Mode rencana menunggu Jalankan; mode langsung mengisi sekarang, pratinjau tetap tampil.
        assistantMsg.aksi = res.aksi
        if (modeRencana) assistantMsg.status = 'rencana'
        else Object.assign(assistantMsg, { status: 'diisi', isiId: onIsiForm(res.aksi) })
      }
      setMessages((prev) => [...prev, assistantMsg])
    } catch (err: any) {
      const errorMsg: Message = {
        id: (Date.now() + 1).toString(),
        sender: 'assistant',
        text: `Mohon maaf, terjadi kendala saat memproses: ${err?.message || 'Server error'}. Silakan coba lagi.`,
      }
      setMessages((prev) => [...prev, errorMsg])
    } finally {
      setLoading(false)
    }
  }

  const ubahPesan = (id: string, patch: Partial<Message>) => setMessages((prev) => prev.map((m) => (m.id === id ? { ...m, ...patch } : m)))
  const jalankan = (m: Message) => m.aksi && onIsiForm && ubahPesan(m.id, { status: 'diisi', isiId: onIsiForm(m.aksi) })
  const urungkan = (m: Message) => {
    if (!m.aksi || m.isiId === undefined) return
    onUrungkan?.(m.isiId, m.aksi)
    ubahPesan(m.id, { status: 'diurungkan' })
  }

  const renderAksi = (m: Message) => {
    const a = m.aksi!
    return (
      <div className="mt-2 w-[92%] rounded-xl border border-[var(--border-subtle)] bg-card p-3 text-xs">
        <div className="flex items-center justify-between gap-2 mb-2">
          <span className="font-semibold text-foreground text-[13px]">Isi form {JUDUL_FORM[a.form]}</span>
          <span className="pill-status pill-brand">Saran AI</span>
        </div>
        <table className="w-full">
          <tbody>
            {pratinjau(a).map((r) => (
              <tr key={r.label}>
                <td className="text-muted-foreground pr-3 py-0.5 align-top whitespace-nowrap">{r.label}</td>
                <td className="text-foreground font-medium py-0.5">{r.nilai}</td>
              </tr>
            ))}
          </tbody>
        </table>
        {!!a.catatan?.length && (
          <ul className="mt-2 space-y-0.5 text-muted-foreground">
            {a.catatan.map((c) => <li key={c}>• {c}</li>)}
          </ul>
        )}
        {m.status === 'rencana' && (
          <>
            <p className="mt-2.5 font-semibold text-foreground">Rencana</p>
            <ol className="mt-1 list-decimal pl-4 space-y-0.5 text-foreground">
              {rencana(a).map((l) => <li key={l}>{l}</li>)}
            </ol>
            <div className="mt-2.5 flex gap-1.5">
              <Button size="sm" onClick={() => jalankan(m)}>Jalankan</Button>
              <Button size="sm" variant="ghost" onClick={() => ubahPesan(m.id, { status: 'batal' })}>Batal</Button>
            </div>
          </>
        )}
        {m.status === 'diisi' && (
          <div className="mt-2.5 flex items-center justify-between gap-2">
            <span className="text-muted-foreground">Form sudah diisi. Periksa lalu klik Simpan.</span>
            <Button size="sm" variant="outline" onClick={() => urungkan(m)}>Urungkan</Button>
          </div>
        )}
        {m.status === 'diurungkan' && <p className="mt-2.5 text-muted-foreground">Isian diurungkan; form kembali seperti sebelumnya.</p>}
        {m.status === 'batal' && <p className="mt-2.5 text-muted-foreground">Dibatalkan, form tidak diisi.</p>}
      </div>
    )
  }

  // Helper untuk merender formatting teks sederhana (bold, list, headings)
  const renderFormattedText = (raw: string) => {
    const lines = raw.split('\n')
    return (
      <div className="space-y-1 text-sm leading-relaxed text-foreground">
        {lines.map((line, idx) => {
          if (line.startsWith('### ')) {
            return (
              <h4 key={idx} className="font-heading font-bold text-[var(--ink)] mt-2 mb-1 text-base">
                {line.replace('### ', '')}
              </h4>
            )
          }
          if (line.startsWith('- ')) {
            const content = line.replace('- ', '')
            return (
              <div key={idx} className="flex items-start gap-1.5 pl-1">
                <span className="text-[var(--link)] mt-0.5">•</span>
                <span dangerouslySetInnerHTML={{ __html: formatInline(content) }} />
              </div>
            )
          }
          if (line.trim() === '') {
            return <div key={idx} className="h-1" />
          }
          return <p key={idx} dangerouslySetInnerHTML={{ __html: formatInline(line) }} />
        })}
      </div>
    )
  }

  const formatInline = (text: string) => {
    return text
      .replace(/\*\*(.*?)\*\*/g, '<strong class="font-semibold text-foreground">$1</strong>')
      .replace(/\*(.*?)\*/g, '<em class="text-muted-foreground">$1</em>')
      .replace(/`([^`]+)`/g, '<code class="px-1 py-0.5 rounded bg-[var(--bg-raised)] font-mono text-xs">$1</code>')
  }

  return (
    <div className="fixed bottom-6 right-6 z-50 flex flex-col items-end">
      {/* Panel Chat */}
      {isOpen && (
        <div className="mb-4 w-[430px] max-w-[94vw] h-[580px] max-h-[82vh] rounded-2xl border border-[var(--border-subtle)] bg-card shadow-[var(--shadow-3)] flex flex-col overflow-hidden animate-in fade-in slide-in-from-bottom-5 duration-200">
          {/* Header */}
          <div className="dark px-4 py-3.5 bg-[var(--navy-700)] flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="w-9 h-9 rounded-[10px] bg-[var(--navy-900)] flex items-center justify-center">
                <Monogram size={16} />
              </div>
              <div>
                <h3 className="font-heading font-bold text-sm">AI Konsultan Pajak & Bisnis</h3>
                <p className="text-[11px] text-muted-foreground">Master Pajak</p>
              </div>
            </div>
            <div className="flex items-center gap-1">
              {onIsiForm && (
                <label className="flex items-center gap-1.5 mr-1.5 text-[11px] text-muted-foreground cursor-pointer whitespace-nowrap" title="Saat mengisi form, AI menampilkan rencana dan menunggu Anda klik Jalankan">
                  <Switch size="sm" checked={modeRencana} onCheckedChange={gantiMode} aria-label="Mode rencana" />
                  Mode rencana
                </label>
              )}
              <button
                onClick={() => setMessages([{
                  id: 'reset',
                  sender: 'assistant',
                  text: 'Riwayat chat telah dibersihkan. Ada yang bisa saya bantu terkait laporan bisnis atau aturan pajak UMKM Anda?',
                  suggestions: INITIAL_SUGGESTIONS,
                }])}
                title="Reset Percakapan"
                className="p-1.5 rounded-lg text-muted-foreground hover:text-foreground hover:bg-white/10 transition"
              >
                <RefreshCw className="w-4 h-4" />
              </button>
              <button
                onClick={() => setIsOpen(false)}
                title="Tutup Panel"
                className="p-1.5 rounded-lg text-muted-foreground hover:text-foreground hover:bg-white/10 transition"
              >
                <X className="w-4 h-4" />
              </button>
            </div>
          </div>

          {/* Messages list */}
          <div className="flex-1 overflow-y-auto p-4 space-y-4 text-sm scrollbar-thin">
            {messages.map((m) => (
              <div
                key={m.id}
                className={`flex flex-col ${m.sender === 'user' ? 'items-end' : 'items-start'}`}
              >
                <div
                  className={`max-w-[88%] rounded-2xl px-4 py-3 shadow-sm ${
                    m.sender === 'user'
                      ? 'bg-[var(--navy-600)] text-white rounded-br-none'
                      : 'bg-[var(--bg-card-hover)] text-foreground rounded-bl-none'
                  }`}
                >
                  {m.sender === 'user' ? (
                    <p className="whitespace-pre-wrap">{m.text}</p>
                  ) : (
                    renderFormattedText(m.text)
                  )}
                </div>

                {m.aksi && renderAksi(m)}

                {/* Tautan ke halaman aplikasi (panduan menu) */}
                {m.tautan && m.tautan.length > 0 && onBuka && (
                  <div className="mt-2 flex flex-wrap gap-1.5 max-w-[92%]">
                    {m.tautan.map((l) => (
                      <button
                        key={l.view + (l.tab ?? '') + l.label}
                        onClick={() => onBuka(l.view, l.tab)}
                        className="text-xs font-semibold px-3 py-1.5 rounded-lg bg-primary text-primary-foreground hover:bg-[var(--primary-hover)] transition flex items-center gap-1.5"
                      >
                        <span>{l.label}</span>
                        <ArrowRight className="w-3.5 h-3.5" />
                      </button>
                    ))}
                  </div>
                )}

                {/* Quick suggestions button */}
                {m.suggestions && m.suggestions.length > 0 && (
                  <div className="mt-2.5 flex flex-wrap gap-1.5 max-w-[92%]">
                    {m.suggestions.map((sug, i) => (
                      <button
                        key={i}
                        onClick={() => handleSend(sug)}
                        disabled={loading}
                        className="text-xs px-2.5 py-1.5 rounded-lg bg-card hover:bg-[var(--bg-card-hover)] hover:text-foreground border border-[var(--border-shell)] text-muted-foreground transition text-left flex items-center gap-1.5"
                      >
                        <MessageSquareQuote className="w-3 h-3 text-[var(--link)] shrink-0" />
                        <span>{sug}</span>
                      </button>
                    ))}
                  </div>
                )}
              </div>
            ))}

            {loading && (
              <div className="flex items-center gap-2 text-xs text-muted-foreground bg-[var(--bg-raised)] px-3 py-2 rounded-xl w-fit">
                <Sparkles className="w-4 h-4 text-[var(--link)] animate-spin" />
                <span>AI sedang menganalisis laporan & aturan pajak...</span>
              </div>
            )}
            <div ref={scrollRef} />
          </div>

          {/* Chat Input */}
          <form
            onSubmit={(e) => {
              e.preventDefault()
              handleSend()
            }}
            className="p-3 border-t border-[var(--border-subtle)] flex items-center gap-2"
          >
            <input
              type="text"
              value={input}
              onChange={(e) => setInput(e.target.value)}
              placeholder="Tanya laporan bisnis atau aturan pajak..."
              className="flex-1 h-10 bg-[var(--bg-input)] border border-input rounded-lg px-3 text-sm text-foreground placeholder:text-muted-foreground outline-none focus-visible:border-primary focus-visible:ring-3 focus-visible:ring-primary/25 transition"
              disabled={loading}
            />
            <button
              type="submit"
              disabled={loading || !input.trim()}
              aria-label="Kirim"
              className="size-10 grid place-items-center rounded-lg bg-primary text-primary-foreground hover:bg-[var(--primary-hover)] disabled:opacity-40 transition"
            >
              <Send className="w-4 h-4" />
            </button>
          </form>
        </div>
      )}

      {/* Floating Trigger Button */}
      <button
        onClick={() => setIsOpen((prev) => !prev)}
        className="group relative flex items-center gap-2.5 px-4 py-3 rounded-full bg-[var(--navy-700)] text-white text-sm shadow-[var(--shadow-2)] hover:bg-[var(--navy-900)] active:scale-[0.98] transition border border-white/10 outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
      >
        <div className="relative">
          <Bot className="w-5 h-5 text-white" />
          <span className="absolute -top-1 -right-1 flex h-2 w-2">
            <span className="motion-safe:animate-ping absolute inline-flex h-full w-full rounded-full bg-[var(--yellow-300)] opacity-75"></span>
            <span className="relative inline-flex rounded-full h-2 w-2 bg-[var(--yellow-500)]"></span>
          </span>
        </div>
        <span className="font-semibold tracking-wide">Tanya AI Pajak</span>
        <Sparkles className="w-4 h-4 text-[var(--navy-200)]" />
      </button>
    </div>
  )
}
