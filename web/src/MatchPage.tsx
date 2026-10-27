import { useCallback, useEffect, useRef, useState } from 'react'
import { Board } from './Board'
import { Captured } from './Captured'
import { Chat } from './Chat'
import { applyLocal, pieceImg } from './position'
import { navigate } from './router'
import { describeSettings } from './settings'
import { Button, Dialog, Spinner, Tip } from './ui'
import { useMatch, type Color, type MatchState } from './useMatch'

const NAMES: Record<Color, string> = { w: 'White', b: 'Black' }
const other = (c: Color): Color => (c === 'w' ? 'b' : 'w')

function resultText(s: MatchState) {
  if (s.result === '1/2-1/2') return { title: 'Draw', detail: `By ${s.reason}.` }
  const winner: Color = s.result === '1-0' ? 'w' : 'b'
  const title = s.you ? (s.you === winner ? 'You win' : 'You lose') : `${NAMES[winner]} wins`
  return { title, detail: `${NAMES[winner]} wins by ${s.reason}.` }
}

export function MatchPage({ id }: { id: string }) {
  const { state, conn, send, rtt, chat } = useMatch(id)
  const [pending, setPending] = useState<{ uci: string; sentWith: MatchState; sentAt: number } | null>(null)
  const [promotion, setPromotion] = useState<{ from: string; to: string } | null>(null)
  const [flipped, setFlipped] = useState(false)
  const [confirmResign, setConfirmResign] = useState(false)
  const [dismissedOver, setDismissedOver] = useState('')
  const [moveMs, setMoveMs] = useState<number | null>(null)

  // Show our own move immediately. Any newer state from the server ends the
  // prediction, whether it confirms the move or rejects it.
  const predicting = !!state && !!pending && pending.sentWith === state
  const board = state ? (predicting ? applyLocal(state.board, pending!.uci) : state.board) : '.'.repeat(64)
  const legal = state && !predicting && conn === 'open' ? state.legal : []
  const last = predicting ? { from: pending!.uci.slice(0, 2), to: pending!.uci.slice(2, 4) } : state?.last

  // How long the server took to confirm our move, for tracking sync lag.
  useEffect(() => {
    if (state && pending && pending.sentWith !== state) {
      if (state.ply > pending.sentWith.ply) setMoveMs(Math.round(performance.now() - pending.sentAt))
      setPending(null)
    }
  }, [state, pending])

  const you = state?.you || null
  const them = you ? other(you) : null
  const orientation: Color = (you === 'b') !== flipped ? 'b' : 'w'
  const playing = state?.status === 'playing'
  const ply = state?.ply ?? 0
  const turn = state && (predicting ? other(state.turn) : state.turn)
  const toMove = playing && conn === 'open' ? turn : null
  const yourMove = !!you && toMove === you

  const sendMove = useCallback(
    (uci: string) => {
      if (state && send({ type: 'move', uci })) setPending({ uci, sentWith: state, sentAt: performance.now() })
    },
    [send, state],
  )

  const onMove = useCallback(
    (from: string, to: string) => {
      if (legal.includes(from + to + 'q')) setPromotion({ from, to })
      else sendMove(from + to)
    },
    [legal, sendMove],
  )

  // Say whose move it is in the tab title, so a backgrounded tab shows when it's your turn.
  useEffect(() => {
    if (!you || !playing) return
    document.title = yourMove ? '● Your move · Chess' : 'Their move · Chess'
    return () => {
      document.title = 'Chess'
    }
  }, [you, playing, yourMove])

  // Haptic tick when a move lands, on phones that support it.
  const lastPly = useRef(ply)
  useEffect(() => {
    if (ply > lastPly.current) navigator.vibrate?.(12)
    lastPly.current = ply
  }, [ply])

  const link = `${location.origin}/m/${id}`

  if (conn === 'missing' || conn === 'outdated') {
    return (
      <main className="flex flex-1 flex-col items-start justify-center gap-6 px-4 sm:px-8">
        <h2 className="text-[clamp(42px,8vw,64px)] leading-none font-bold tracking-tight">
          {conn === 'missing' ? 'Match not found' : 'Server out of date'}
        </h2>
        <p className="max-w-xl text-accent">
          {conn === 'missing'
            ? `The ID ${id} doesn't exist, or the server restarted.`
            : 'The game server is a different version from this page. Stop every running copy of the server and start it again.'}
        </p>
        <Button variant="solid" onClick={() => navigate('/')}>
          Back home
        </Button>
      </main>
    )
  }

  const overKey = state ? `${state.round}:${state.ply}` : ''
  const over = state?.status === 'over' ? resultText(state) : null

  const status = (() => {
    if (!state) return conn === 'failed' ? "Can't reach the game server. Retrying…" : 'Connecting…'
    if (conn === 'failed') return "Can't reach the game server. Retrying…"
    if (conn !== 'open') return 'Reconnecting…'
    if (state.status === 'waiting') return 'Waiting for an opponent'
    if (over) return `${over.title} by ${state.reason}`
    const check = state.check && !predicting ? 'Check. ' : ''
    if (!you) return `${check}${NAMES[turn!]} to move`
    return `${check}${turn === you ? 'Your move' : 'Their move'}`
  })()

  const bottom = orientation
  const top = other(orientation)

  return (
    <main className="flex flex-1 justify-center px-4 pb-6 sm:px-8">
      <div className="flex w-full max-w-[1100px] flex-col gap-6 lg:flex-row lg:items-start lg:justify-center">
        <section className="board-wrap mx-auto flex flex-col gap-3 lg:mx-0">
          <PlayerBar color={top} state={state} you={you} active={toMove === top} />
          <div className="board-turn relative rounded-md" data-your-move={yourMove}>
            <Board
              board={board}
              orientation={orientation}
              legal={legal}
              lastMove={last}
              check={predicting ? undefined : state?.check}
              onMove={onMove}
            />
            {state?.status === 'waiting' && <WaitingCard id={id} link={link} />}
            {!state && (
              <div className="absolute inset-0 z-20 grid place-items-center rounded-md bg-bg/60 text-lg font-semibold">
                <Spinner label="Joining match" />
              </div>
            )}
          </div>
          <PlayerBar color={bottom} state={state} you={you} active={toMove === bottom} />
        </section>

        <aside className="flex w-full flex-col gap-4 lg:w-[320px] lg:pt-9">
          <p className="flex items-center gap-2.5 text-2xl font-bold tracking-tight" aria-live="polite">
            {conn !== 'open' ? (
              <span className="spinner text-[0.75em]" aria-hidden />
            ) : (
              yourMove && <span className="turn-dot" aria-hidden />
            )}
            {status}
          </p>

          {state && (
            <ul className="flex flex-wrap gap-1.5" aria-label="Match rules">
              {describeSettings(state.settings).map((r) => (
                <li
                  key={r}
                  className="rounded-full px-2.5 py-1 text-xs font-semibold shadow-[inset_0_0_0_1.5px_var(--color-fg)]"
                >
                  {r}
                </li>
              ))}
            </ul>
          )}

          {state && you && them && state.draw[them] && playing && (
            <div className="flex items-center justify-between gap-3 rounded-xl p-3 shadow-[inset_0_0_0_2px_var(--color-fg)]">
              <span>They offer a draw.</span>
              <Button variant="solid" className="h-9 px-4" onClick={() => send({ type: 'draw' })}>
                Accept
              </Button>
            </div>
          )}

          {state && them && state.rematch[them] && state.status === 'over' && !state.rematch[you!] && (
            <p className="font-semibold">They want a rematch.</p>
          )}

          <div className="flex flex-wrap gap-2">
            {state?.status === 'over' && you ? (
              <Button variant={state.rematch[you] ? 'outline' : 'solid'} onClick={() => send({ type: 'rematch' })}>
                {state.rematch[you] ? 'Cancel rematch' : 'Rematch'}
              </Button>
            ) : (
              you &&
              playing && (
                <>
                  <Button onClick={() => send({ type: 'draw' })}>
                    {state!.draw[you] ? 'Cancel draw offer' : 'Offer draw'}
                  </Button>
                  <Button onClick={() => setConfirmResign(true)} disabled={ply === 0}>
                    Resign
                  </Button>
                </>
              )
            )}
            <Tip label="Flip board">
              <Button variant="ghost" icon aria-label="Flip board" onClick={() => setFlipped((f) => !f)}>
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
                  <path d="M7 4v16M7 20l-3-3M7 20l3-3M17 20V4M17 4l-3 3M17 4l3 3" />
                </svg>
              </Button>
            </Tip>
            <CopyLink link={link} />
          </div>

          {state && you && them && state.status !== 'waiting' && (
            <Chat
              lines={chat}
              you={you}
              themName={state.players[them].name}
              disabled={conn !== 'open'}
              onSend={(text) => send({ type: 'chat', text })}
            />
          )}

          <MoveList history={state?.history ?? []} />

          {!you && state && state.status !== 'waiting' && (
            <p className="text-sm text-accent">
              You're watching this match.
              {state.settings.fog && state.status === 'playing' && " With fog on, you see only each side's last move."}
            </p>
          )}

          {rtt !== null && (
            <p className="font-mono text-xs text-accent tabular-nums">
              ping {rtt} ms{moveMs !== null && ` · last move confirmed in ${moveMs} ms`}
            </p>
          )}
        </aside>
      </div>

      <Dialog open={!!promotion} onOpenChange={(o) => !o && setPromotion(null)} title="Promote to">
        <div className="grid grid-cols-4 gap-2">
          {(['q', 'r', 'b', 'n'] as const).map((p) => (
            <button
              key={p}
              autoFocus={p === 'q'}
              aria-label={{ q: 'Queen', r: 'Rook', b: 'Bishop', n: 'Knight' }[p]}
              className="aspect-square rounded-xl bg-sq-light p-1 shadow-[inset_0_0_0_2px_var(--color-fg)] transition-transform hover:scale-105 active:scale-95"
              onClick={() => {
                sendMove(promotion!.from + promotion!.to + p)
                setPromotion(null)
              }}
            >
              <img src={pieceImg(you === 'b' ? p : p.toUpperCase())} alt="" draggable={false} className="size-full" />
            </button>
          ))}
        </div>
      </Dialog>

      <Dialog
        open={confirmResign}
        onOpenChange={setConfirmResign}
        title="Resign?"
        description="This ends the game and gives your opponent the win."
      >
        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={() => setConfirmResign(false)}>
            Keep playing
          </Button>
          <Button
            variant="solid"
            onClick={() => {
              send({ type: 'resign' })
              setConfirmResign(false)
            }}
          >
            Resign
          </Button>
        </div>
      </Dialog>

      <Dialog
        open={!!over && dismissedOver !== overKey}
        onOpenChange={(o) => !o && setDismissedOver(overKey)}
        title={over?.title ?? ''}
        description={over?.detail}
      >
        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={() => setDismissedOver(overKey)}>
            View board
          </Button>
          {you && (
            <Button
              variant="solid"
              onClick={() => {
                if (!state!.rematch[you]) send({ type: 'rematch' })
                setDismissedOver(overKey)
              }}
            >
              Rematch
            </Button>
          )}
        </div>
      </Dialog>
    </main>
  )
}

type PlayerBarProps = { color: Color; state: MatchState | null; you: Color | null; active: boolean }

function PlayerBar({ color, state, you, active }: PlayerBarProps) {
  const seat = state?.players[color]
  const name = seat?.name
  const label = !seat?.taken ? 'Waiting…' : name || (color === you ? 'You' : you ? 'Opponent' : 'Guest')
  return (
    <div className="player-bar flex h-10 items-center gap-2 rounded-full px-3 text-[15px]" data-active={active}>
      <span
        className={`size-3.5 shrink-0 rounded-full shadow-[0_0_0_1.5px_var(--color-fg)] ${color === 'w' ? 'bg-[#fff]' : 'bg-[#111]'}`}
        aria-hidden
      />
      {name ? (
        <a
          href={`/u/${name}`}
          onClick={(e) => {
            e.preventDefault()
            navigate(`/u/${name}`)
          }}
          className="min-w-0 truncate font-semibold hover:underline"
        >
          {label}
        </a>
      ) : (
        <span className="font-semibold">{label}</span>
      )}
      {name && color === you && <span className="text-accent">(you)</span>}
      <span className="text-accent max-sm:hidden">{NAMES[color]}</span>
      {active && <span className="turn-tag">to move</span>}
      <span className="ml-auto flex min-w-0 items-center gap-2">
        {seat?.taken && !seat.online && <span className="text-sm text-danger">offline</span>}
        <Captured pieces={state?.captured[color] ?? ''} className="text-[clamp(16px,calc(var(--board)/30),22px)]" />
      </span>
    </div>
  )
}

function WaitingCard({ id, link }: { id: string; link: string }) {
  return (
    <div className="absolute inset-0 z-20 grid place-items-center bg-bg/70 p-4 backdrop-blur-[3px]">
      <div className="flex w-full max-w-[320px] flex-col items-center gap-4 rounded-2xl bg-bg p-6 text-center shadow-[0_0_0_2px_var(--color-fg)]">
        <p className="text-accent">Send this ID to your opponent</p>
        <p className="font-mono text-4xl font-bold tracking-[0.2em] select-all">{id}</p>
        <CopyLink link={link} wide />
      </div>
    </div>
  )
}

function CopyLink({ link, wide }: { link: string; wide?: boolean }) {
  const [copied, setCopied] = useState(false)
  const share = async () => {
    try {
      if (wide && navigator.share && matchMedia('(pointer: coarse)').matches) {
        await navigator.share({ title: 'Play chess with me', url: link })
        return
      }
      if (navigator.clipboard) await navigator.clipboard.writeText(link)
      else copyFallback(link) // the clipboard API needs HTTPS, which a phone on the LAN won't have
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {}
  }
  if (wide) {
    return (
      <Button variant="solid" className="w-full" onClick={share}>
        {copied ? 'Copied' : 'Share link'}
      </Button>
    )
  }
  return (
    <Tip label={copied ? 'Copied' : 'Copy match link'}>
      <Button variant="ghost" icon aria-label="Copy match link" onClick={share}>
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
          <path d="M10 14a4 4 0 0 0 5.7 0l3-3a4 4 0 0 0-5.7-5.7l-1 1M14 10a4 4 0 0 0-5.7 0l-3 3a4 4 0 0 0 5.7 5.7l1-1" />
        </svg>
      </Button>
    </Tip>
  )
}

function MoveList({ history }: { history: string[] }) {
  const ref = useRef<HTMLOListElement>(null)
  useEffect(() => {
    ref.current?.scrollTo({ top: ref.current.scrollHeight })
  }, [history.length])

  if (!history.length) return null
  const rows = []
  for (let i = 0; i < history.length; i += 2) rows.push([history[i], history[i + 1]])

  return (
    <ol
      ref={ref}
      className="grid max-h-[40dvh] max-w-[300px] grid-cols-[2.5rem_1fr_1fr] gap-y-1 overflow-y-auto font-mono text-[15px] tabular-nums lg:max-h-[50dvh]"
    >
      {rows.map(([w, b], i) => (
        <li key={i} className="contents">
          <span className="text-accent">{i + 1}.</span>
          <span>{w}</span>
          <span>{b ?? ''}</span>
        </li>
      ))}
    </ol>
  )
}

function copyFallback(text: string) {
  const el = document.createElement('textarea')
  el.value = text
  el.style.position = 'fixed'
  el.style.opacity = '0'
  document.body.append(el)
  el.select()
  document.execCommand('copy')
  el.remove()
}
