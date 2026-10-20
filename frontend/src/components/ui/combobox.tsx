import * as React from "react"
import { cn } from "cn"
import { Popover as PopoverPrimitive } from "radix-ui"
import { CheckIcon, ChevronsUpDownIcon } from "lucide-react"
import { filterOptions, MAX_SHOWN } from "@/lib/combobox"

// Combobox: Select yang bisa dicari (Popover radix + listbox). Nilai harus salah satu options.
function Combobox({ id, value, onChange, options, placeholder = "Pilih", searchPlaceholder = "Cari…", empty = "Tidak ada yang cocok.", disabled, className }: {
  id?: string
  value: string
  onChange: (v: string) => void
  options: string[]
  placeholder?: string
  searchPlaceholder?: string
  empty?: string
  disabled?: boolean
  className?: string
}) {
  const [open, setOpen] = React.useState(false)
  const [q, setQ] = React.useState("")
  const [active, setActive] = React.useState(0)
  const listId = React.useId()
  const shown = filterOptions(options, q)
  const selected = options.includes(value) ? value : ""

  function pick(v: string) {
    onChange(v)
    setOpen(false)
    setQ("")
  }

  function onKeyDown(e: React.KeyboardEvent) {
    if (e.key === "ArrowDown" || e.key === "ArrowUp") {
      e.preventDefault()
      const n = shown.length
      if (n) setActive((i) => (i + (e.key === "ArrowDown" ? 1 : n - 1)) % n)
    } else if (e.key === "Enter" && shown[active]) {
      e.preventDefault()
      pick(shown[active])
    }
  }

  React.useEffect(() => {
    document.getElementById(`${listId}-${active}`)?.scrollIntoView({ block: "nearest" })
  }, [active, listId])

  return (
    <PopoverPrimitive.Root open={open} onOpenChange={(o) => {
      setOpen(o)
      setQ("")
      setActive(Math.max(0, filterOptions(options, "").indexOf(selected))) // buka di pilihan saat ini
    }}>
      <PopoverPrimitive.Trigger asChild disabled={disabled}>
        <button
          id={id}
          type="button"
          role="combobox"
          aria-expanded={open}
          aria-controls={listId}
          data-placeholder={selected ? undefined : ""}
          className={cn(
            "flex h-8 w-full items-center justify-between gap-1.5 rounded-lg border border-input bg-transparent py-2 pr-2 pl-2.5 text-left text-sm transition-colors outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50 data-placeholder:text-muted-foreground dark:bg-input/30 dark:hover:bg-input/50",
            className
          )}
        >
          <span className="truncate">{selected || placeholder}</span>
          <ChevronsUpDownIcon className="size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
        </button>
      </PopoverPrimitive.Trigger>
      <PopoverPrimitive.Portal>
        <PopoverPrimitive.Content
          align="start"
          sideOffset={4}
          className="z-50 w-(--radix-popover-trigger-width) min-w-56 rounded-lg bg-popover p-1 text-popover-foreground shadow-md ring-1 ring-foreground/10"
        >
          <input
            autoFocus
            value={q}
            onChange={(e) => { setQ(e.target.value); setActive(0) }}
            onKeyDown={onKeyDown}
            placeholder={searchPlaceholder}
            aria-label={searchPlaceholder}
            aria-controls={listId}
            aria-activedescendant={shown[active] ? `${listId}-${active}` : undefined}
            className="mb-1 h-9 w-full rounded-md border border-input bg-transparent px-2.5 text-sm outline-none focus-visible:border-ring"
          />
          <div id={listId} role="listbox" className="max-h-64 overflow-y-auto">
            {shown.length === 0 && <p className="px-2.5 py-2 text-sm text-muted-foreground">{empty}</p>}
            {shown.map((o, i) => (
              <div
                key={o}
                id={`${listId}-${i}`}
                role="option"
                aria-selected={o === selected}
                data-active={i === active || undefined}
                onMouseEnter={() => setActive(i)}
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => pick(o)}
                className="flex cursor-pointer items-center justify-between gap-2 rounded-md px-2.5 py-1.5 text-sm data-active:bg-accent data-active:text-accent-foreground"
              >
                <span>{o}</span>
                {o === selected && <CheckIcon className="size-4" aria-hidden="true" />}
              </div>
            ))}
            {!q && options.length > MAX_SHOWN && (
              <p className="px-2.5 py-1.5 text-xs text-muted-foreground">Ketik untuk mencari dari {options.length} pilihan…</p>
            )}
          </div>
        </PopoverPrimitive.Content>
      </PopoverPrimitive.Portal>
    </PopoverPrimitive.Root>
  )
}

export { Combobox }
