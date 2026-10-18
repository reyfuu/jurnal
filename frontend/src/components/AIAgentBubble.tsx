import React, { useState, useRef, useEffect } from 'react'
import { Bot, Sparkles, X, Send, RefreshCw, MessageSquareQuote } from 'lucide-react'
import { askAIChat, type AIChatResponse } from '../lib/api'

interface Message {
  id: string
  sender: 'user' | 'assistant'
  text: string
  suggestions?: string[]
}

const INITIAL_SUGGESTIONS = [
  '📊 Ringkas laporan bulan ini',
  '⚖️ Bagaimana aturan pajak UMKM PP 55/2022?',
  '💰 Bagaimana kondisi arus kas dan piutang saya?',
  '💡 Bagaimana cara hitung simulasi harga komisi GoFood?',
]

export const AIAgentBubble: React.FC<{ period?: string }> = ({ period = '2026-08' }) => {
  const [isOpen, setIsOpen] = useState(false)
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [messages, setMessages] = useState<Message[]>([
    {
      id: 'welcome',
      sender: 'assistant',
      text: 'Halo! Saya **AI Konsultan Bisnis & Pajak** dari *Akademi Pajak*.\n\nSaya siap membantu Anda **meringkas laporan keuangan**, menganalisis arus kas & piutang, serta menjelaskan **aturan perpajakan UMKM (PP 55/2022)** secara akurat.',
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

  // Helper untuk merender formatting teks sederhana (bold, list, headings)
  const renderFormattedText = (raw: string) => {
    const lines = raw.split('\n')
    return (
      <div className="space-y-1 text-sm leading-relaxed text-slate-200">
        {lines.map((line, idx) => {
          if (line.startsWith('### ')) {
            return (
              <h4 key={idx} className="font-semibold text-emerald-400 mt-2 mb-1 text-base">
                {line.replace('### ', '')}
              </h4>
            )
          }
          if (line.startsWith('- ')) {
            const content = line.replace('- ', '')
            return (
              <div key={idx} className="flex items-start gap-1.5 pl-1">
                <span className="text-emerald-400 mt-0.5">•</span>
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
      .replace(/\*\*(.*?)\*\*/g, '<strong class="font-semibold text-white">$1</strong>')
      .replace(/\*(.*?)\*/g, '<em class="text-slate-300">$1</em>')
      .replace(/`([^`]+)`/g, '<code class="px-1 py-0.5 rounded bg-black/40 font-mono text-xs text-amber-300">$1</code>')
  }

  return (
    <div className="fixed bottom-6 right-6 z-50 flex flex-col items-end">
      {/* Panel Chat */}
      {isOpen && (
        <div className="mb-4 w-[430px] max-w-[94vw] h-[580px] max-h-[82vh] rounded-2xl border border-slate-700/80 bg-[#0d131a]/95 backdrop-blur-xl shadow-2xl flex flex-col overflow-hidden animate-in fade-in slide-in-from-bottom-5 duration-200">
          {/* Header */}
          <div className="px-4 py-3.5 border-b border-slate-800 bg-gradient-to-r from-emerald-950/40 via-slate-900/60 to-teal-950/40 flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="relative w-9 h-9 rounded-xl bg-gradient-to-tr from-emerald-600 to-teal-500 flex items-center justify-center shadow-lg shadow-emerald-500/20">
                <Bot className="w-5 h-5 text-white" />
                <span className="absolute -bottom-0.5 -right-0.5 w-2.5 h-2.5 rounded-full bg-emerald-400 border-2 border-[#0d131a]" />
              </div>
              <div>
                <div className="flex items-center gap-1.5">
                  <h3 className="font-semibold text-sm text-white">AI Konsultan Pajak & Bisnis</h3>
                  <span className="text-[10px] font-mono px-1.5 py-0.2 rounded bg-emerald-500/20 text-emerald-400 border border-emerald-500/30">
                    Live
                  </span>
                </div>
                <p className="text-[11px] text-slate-400 font-mono">by Akademi Pajak</p>
              </div>
            </div>
            <div className="flex items-center gap-1">
              <button
                onClick={() => setMessages([{
                  id: 'reset',
                  sender: 'assistant',
                  text: 'Riwayat chat telah dibersihkan. Ada yang bisa saya bantu terkait laporan bisnis atau aturan pajak UMKM Anda?',
                  suggestions: INITIAL_SUGGESTIONS,
                }])}
                title="Reset Percakapan"
                className="p-1.5 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800/80 transition"
              >
                <RefreshCw className="w-4 h-4" />
              </button>
              <button
                onClick={() => setIsOpen(false)}
                title="Tutup Panel"
                className="p-1.5 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800/80 transition"
              >
                <X className="w-4 h-4" />
              </button>
            </div>
          </div>

          {/* Messages list */}
          <div className="flex-1 overflow-y-auto p-4 space-y-4 text-sm scrollbar-thin scrollbar-thumb-slate-700">
            {messages.map((m) => (
              <div
                key={m.id}
                className={`flex flex-col ${m.sender === 'user' ? 'items-end' : 'items-start'}`}
              >
                <div
                  className={`max-w-[88%] rounded-2xl px-4 py-3 shadow-sm ${
                    m.sender === 'user'
                      ? 'bg-emerald-600 text-white rounded-br-none'
                      : 'bg-slate-900/90 border border-slate-800 text-slate-200 rounded-bl-none'
                  }`}
                >
                  {m.sender === 'user' ? (
                    <p className="whitespace-pre-wrap">{m.text}</p>
                  ) : (
                    renderFormattedText(m.text)
                  )}
                </div>

                {/* Quick suggestions button */}
                {m.suggestions && m.suggestions.length > 0 && (
                  <div className="mt-2.5 flex flex-wrap gap-1.5 max-w-[92%]">
                    {m.suggestions.map((sug, i) => (
                      <button
                        key={i}
                        onClick={() => handleSend(sug)}
                        disabled={loading}
                        className="text-xs px-2.5 py-1.5 rounded-lg bg-slate-800/80 hover:bg-emerald-950/60 hover:text-emerald-300 hover:border-emerald-500/40 border border-slate-700/60 text-slate-300 transition text-left flex items-center gap-1.5"
                      >
                        <MessageSquareQuote className="w-3 h-3 text-emerald-400 shrink-0" />
                        <span>{sug}</span>
                      </button>
                    ))}
                  </div>
                )}
              </div>
            ))}

            {loading && (
              <div className="flex items-center gap-2 text-xs text-slate-400 bg-slate-900/60 px-3 py-2 rounded-xl border border-slate-800/80 w-fit">
                <Sparkles className="w-4 h-4 text-emerald-400 animate-spin" />
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
            className="p-3 border-t border-slate-800 bg-slate-950/60 flex items-center gap-2"
          >
            <input
              type="text"
              value={input}
              onChange={(e) => setInput(e.target.value)}
              placeholder="Tanya laporan bisnis atau aturan pajak..."
              className="flex-1 bg-slate-900 border border-slate-700/80 rounded-xl px-3.5 py-2.5 text-sm text-white placeholder-slate-500 focus:outline-none focus:border-emerald-500 transition"
              disabled={loading}
            />
            <button
              type="submit"
              disabled={loading || !input.trim()}
              className="p-2.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 disabled:opacity-40 disabled:hover:bg-emerald-600 text-white transition shadow-lg shadow-emerald-600/20"
            >
              <Send className="w-4 h-4" />
            </button>
          </form>
        </div>
      )}

      {/* Floating Trigger Button */}
      <button
        onClick={() => setIsOpen((prev) => !prev)}
        className="group relative flex items-center gap-2.5 px-4 py-3 rounded-full bg-gradient-to-r from-emerald-600 via-teal-600 to-emerald-700 text-white font-medium text-sm shadow-xl shadow-emerald-500/25 hover:shadow-emerald-500/40 hover:scale-[1.02] active:scale-[0.98] transition border border-emerald-400/30"
      >
        <div className="relative">
          <Bot className="w-5 h-5 text-white" />
          <span className="absolute -top-1 -right-1 flex h-2 w-2">
            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-300 opacity-75"></span>
            <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-400"></span>
          </span>
        </div>
        <span className="font-semibold tracking-wide">Tanya AI Pajak</span>
        <Sparkles className="w-4 h-4 text-emerald-200 group-hover:rotate-12 transition-transform" />
      </button>
    </div>
  )
}
