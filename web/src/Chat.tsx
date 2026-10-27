import { useEffect, useRef, useState } from 'react'
import { Button } from './ui'
import type { ChatLine, Color } from './useMatch'

/** Must match chatMaxLen in server/match.go. */
const MAX_LEN = 500

type ChatProps = {
  lines: ChatLine[]
  you: Color
  /** The opponent's username, if they're signed in. */
  themName?: string
  disabled: boolean
  onSend: (text: string) => boolean
}

/** Chat between the two players. It lasts only while the game does. */
export function Chat({ lines, you, themName, disabled, onSend }: ChatProps) {
  const [text, setText] = useState('')
  const [inView, setInView] = useState(true)
  const [seen, setSeen] = useState(0)
  const panel = useRef<HTMLElement>(null)
  const list = useRef<HTMLOListElement>(null)

  useEffect(() => {
    list.current?.scrollTo({ top: list.current.scrollHeight })
  }, [lines.length])

  // On a phone the chat sits below the board, so say when a message lands out of sight.
  useEffect(() => {
    const el = panel.current
    if (!el) return
    const io = new IntersectionObserver(([e]) => setInView(e.isIntersecting), { threshold: 0.4 })
    io.observe(el)
    return () => io.disconnect()
  }, [])
  const lastId = lines.at(-1)?.id ?? 0
  useEffect(() => {
    if (inView || lastId < seen) setSeen(lastId)
  }, [inView, lastId, seen])
  const unread = lines.filter((l) => l.id > seen && l.from !== you).length

  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    const t = text.trim()
    if (t && onSend(t)) setText('')
  }

  return (
    <section ref={panel} aria-label="Chat" className="flex flex-col gap-2 rounded-xl p-3 shadow-[inset_0_0_0_2px_var(--color-fg)]">
      <h3 className="flex items-baseline justify-between gap-2 font-semibold">
        Chat
        <span className="text-xs font-medium text-accent">Cleared when the game ends</span>
      </h3>

      <ol
        ref={list}
        aria-live="polite"
        className="flex max-h-[30dvh] min-h-16 flex-col gap-1.5 overflow-y-auto overscroll-contain text-[15px] lg:max-h-[240px]"
      >
        {lines.length === 0 && <li className="m-auto text-sm text-accent">Say hi to {themName ?? 'your opponent'}.</li>}
        {lines.map((l) => {
          const mine = l.from === you
          return (
            <li
              key={l.id}
              className={`chat-line max-w-[85%] rounded-2xl px-3 py-1.5 break-words whitespace-pre-wrap ${
                mine
                  ? 'self-end rounded-br-md bg-fg text-bg'
                  : 'self-start rounded-bl-md shadow-[inset_0_0_0_1.5px_var(--color-fg)]'
              }`}
              title={new Date(l.at).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' })}
            >
              <span className="sr-only">{mine ? 'You: ' : `${themName ?? 'Opponent'}: `}</span>
              {l.text}
            </li>
          )
        })}
      </ol>

      <form onSubmit={submit} className="flex gap-2">
        <input
          value={text}
          onChange={(e) => setText(e.target.value)}
          maxLength={MAX_LEN}
          placeholder={disabled ? 'Reconnecting…' : 'Message'}
          aria-label="Chat message"
          enterKeyHint="send"
          autoComplete="off"
          disabled={disabled}
          // 16px keeps iOS from zooming in on focus.
          className="h-11 min-w-0 flex-1 rounded-full bg-transparent px-4 text-base shadow-[inset_0_0_0_2px_color-mix(in_srgb,var(--color-fg)_35%,transparent)] outline-none placeholder:text-accent focus-visible:shadow-[inset_0_0_0_2px_var(--color-fg)] disabled:opacity-40"
        />
        <Button type="submit" variant="solid" icon aria-label="Send" disabled={disabled || !text.trim()}>
          <svg
            width="20"
            height="20"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <path d="M12 19V5M5 12l7-7 7 7" />
          </svg>
        </Button>
      </form>

      {unread > 0 && !inView && (
        <button
          className="chat-pill fixed bottom-[max(16px,env(safe-area-inset-bottom))] left-1/2 z-30 h-10 -translate-x-1/2 rounded-full bg-fg px-4 text-sm font-semibold text-bg shadow-[0_8px_24px_rgb(0_0_0/0.25)]"
          onClick={() => panel.current?.scrollIntoView({ behavior: 'smooth', block: 'center' })}
        >
          {unread === 1 ? '1 new message' : `${unread} new messages`}
        </button>
      )}
    </section>
  )
}
