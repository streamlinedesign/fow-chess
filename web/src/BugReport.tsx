import * as RadioGroup from '@radix-ui/react-radio-group'
import { useEffect, useId, useState, type FormEvent } from 'react'
import './bug.css'
import { errorText, useT } from './i18n'
import { Icon } from './icons'
import { playerId } from './player'
import { Button, Dialog, Tip } from './ui'

const CATEGORIES = ['ui', 'rules', 'connection', 'clock', 'bots', 'account', 'other'] as const
type Category = (typeof CATEGORIES)[number]

const MAX = 2000

// The last match this browser had open, so a report sent from the footer
// after leaving a game still knows which game it was.
function lastMatch() {
  try {
    return sessionStorage.getItem('lastMatch') ?? ''
  } catch {
    return ''
  }
}

function rememberMatch(id: string) {
  try {
    sessionStorage.setItem('lastMatch', id)
  } catch {}
}

/** The report form: what it's about, the match code, and what happened. */
export function BugDialog({
  open,
  onOpenChange,
  match = '',
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Prefills the match code. */
  match?: string
}) {
  const t = useT()
  const ids = useId()
  const [category, setCategory] = useState<Category | ''>('')
  const [code, setCode] = useState(match)
  const [text, setText] = useState('')
  const [error, setError] = useState('')
  const [sending, setSending] = useState(false)
  const [sent, setSent] = useState<number | null>(null)

  // Each opening starts fresh, with the current match filled in.
  useEffect(() => {
    if (!open) return
    setCode(match || lastMatch())
    setError('')
    setSent(null)
  }, [open, match])

  async function submit(e: FormEvent) {
    e.preventDefault()
    if (!category) return setError(t.errors.bug_category)
    if (!text.trim()) return setError(t.errors.bug_empty)
    setSending(true)
    setError('')
    const res = await fetch('/api/bugs', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ category, match: code.trim(), text, page: location.pathname, player: playerId }),
    }).catch(() => null)
    setSending(false)
    if (!res) return setError(t.common.cantReach)
    const body = await res.json().catch(() => ({}))
    if (!res.ok) return setError(errorText(body))
    setSent(body.id)
    setCategory('')
    setText('')
  }

  if (sent !== null)
    return (
      <Dialog open={open} onOpenChange={onOpenChange} title={t.bug.sentTitle} description={t.bug.sent(sent)}>
        <div className="flex justify-end">
          <Button variant="solid" onClick={() => onOpenChange(false)}>
            {t.bug.close}
          </Button>
        </div>
      </Dialog>
    )

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={t.bug.report}
      description={t.bug.desc}
      className="w-[min(92vw,480px)]"
    >
      <form className="bug-form" onSubmit={submit} noValidate>
        <div className="field">
          <span id={`${ids}-cat`}>{t.bug.category}</span>
          <RadioGroup.Root
            aria-labelledby={`${ids}-cat`}
            aria-describedby={category ? `${ids}-hint` : undefined}
            value={category}
            onValueChange={(v) => setCategory(v as Category)}
            className="bug-cats"
          >
            {CATEGORIES.map((c) => (
              <RadioGroup.Item key={c} value={c}>
                {t.bug.categories[c]}
              </RadioGroup.Item>
            ))}
          </RadioGroup.Root>
          <small id={`${ids}-hint`} className="bug-hint">
            {category ? t.bug.hints[category] : ' '}
          </small>
        </div>

        <label className="field">
          <span>{t.bug.match}</span>
          <input
            className="input bug-code"
            value={code}
            onChange={(e) => setCode(e.target.value.toUpperCase())}
            maxLength={12}
            autoComplete="off"
            autoCapitalize="characters"
            spellCheck={false}
            aria-describedby={`${ids}-code`}
          />
          <small id={`${ids}-code`} className="bug-hint">
            {t.bug.matchHint}
          </small>
        </label>

        <label className="field">
          <span className="flex justify-between gap-3">
            {t.bug.text}
            {text.length > MAX * 0.8 && (
              <span className={`tabular-nums ${text.length > MAX ? 'text-danger' : 'text-accent'}`}>
                {text.length}/{MAX}
              </span>
            )}
          </span>
          <textarea
            className="input bug-text"
            value={text}
            onChange={(e) => setText(e.target.value)}
            placeholder={t.bug.placeholder}
            rows={5}
            required
          />
        </label>

        {error && (
          <p role="alert" className="text-sm text-danger">
            {error}
          </p>
        )}

        <div className="flex justify-end gap-2">
          <Button type="button" variant="ghost" onClick={() => onOpenChange(false)}>
            {t.common.cancel}
          </Button>
          <Button type="submit" variant="solid" loading={sending}>
            {t.bug.send}
          </Button>
        </div>
      </form>
    </Dialog>
  )
}

/** A bug icon button for a game page; the report it opens names the match. */
export function BugButton({ match }: { match?: string }) {
  const t = useT()
  const [open, setOpen] = useState(false)
  useEffect(() => {
    if (match) rememberMatch(match)
  }, [match])
  return (
    <>
      <Tip label={t.bug.report}>
        <Button variant="ghost" icon aria-label={t.bug.report} onClick={() => setOpen(true)}>
          <Icon name="bug" size={20} />
        </Button>
      </Tip>
      <BugDialog open={open} onOpenChange={setOpen} match={match} />
    </>
  )
}

/** The footer's line: the game is in beta, and a way to report what breaks. */
export function BugNote() {
  const t = useT()
  const [open, setOpen] = useState(false)
  return (
    <span className="bug-note">
      <span>{t.bug.note}</span>
      <button type="button" className="bug-link" onClick={() => setOpen(true)}>
        <Icon name="bug" size={15} />
        {t.bug.report}
      </button>
      <BugDialog open={open} onOpenChange={setOpen} />
    </span>
  )
}
