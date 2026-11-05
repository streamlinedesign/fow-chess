import { useCallback, useEffect, useRef, useState } from 'react'
import { Clock } from './Clock'
import { aroundFrom, parseMove, pieceImg4, SEAT_COLORS, SEATS, type Seat } from './four'
import { FourBoard } from './FourBoard'
import { useT, type Dict } from './i18n'
import { how } from './MatchPage'
import { navigate } from './router'
import { Button, Dialog, Spinner } from './ui'
import { useFourMatch, type FourState } from './useFourMatch'

const seatName = (t: Dict, s: Seat) => t.four.seats[s]

/** A four-player match: the cross board, the four seats, and the table chat. */
export function FourMatchPage({ id }: { id: string }) {
  const { state, conn, send, chat } = useFourMatch(id)
  const t = useT()
  const [pending, setPending] = useState<{ uci: string; sentWith: FourState } | null>(null)
  const [promotion, setPromotion] = useState<{ from: string; to: string } | null>(null)
  const [side, setSide] = useState<Seat>('r')
  const [confirmResign, setConfirmResign] = useState(false)
  const [filling, setFilling] = useState(false)

  const predicting = !!state && !!pending && pending.sentWith === state
  useEffect(() => {
    if (state && pending && pending.sentWith !== state) setPending(null)
  }, [state, pending])

  const you = state?.you || null
  const seat: Seat = you ?? side
  const playing = state?.status === 'playing'
  const legal = state && !predicting && conn === 'open' ? state.legal : []
  const toMove = playing && conn === 'open' && !predicting ? state!.turn : null
  const yourMove = !!you && toMove === you
  const youOut = !!you && !!state?.players[you].out

  // Show our own move at once; the server's next state replaces it.
  const board = (() => {
    if (!state) return []
    if (!predicting) return state.board
    const m = parseMove(pending!.uci)!
    const b = [...state.board]
    const idx = (sq: string) => (Number(sq.slice(1)) - 1) * 14 + sq.charCodeAt(0) - 97
    b[idx(m.to)] = m.promo ? b[idx(m.from)][0] + m.promo.toUpperCase() : b[idx(m.from)]
    b[idx(m.from)] = '.'
    return b
  })()
  const last = predicting ? parseMove(pending!.uci) : state?.last

  const sendMove = useCallback(
    (uci: string) => {
      if (state && send({ type: 'move', uci })) setPending({ uci, sentWith: state })
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

  const watching = state?.watching
  useEffect(() => {
    if (watching && watching !== side) send({ type: 'watch', side })
  }, [watching, side, send])

  useEffect(() => {
    if (!you || !playing) return
    document.title = yourMove ? t.match.titleYourMove : t.match.titleTheirMove
    return () => {
      document.title = t.common.appName
    }
  }, [you, playing, yourMove, t])

  const ply = state?.ply ?? 0
  const lastPly = useRef(ply)
  useEffect(() => {
    if (ply > lastPly.current) navigator.vibrate?.(12)
    lastPly.current = ply
  }, [ply])

  if (conn === 'missing' || conn === 'outdated') {
    return (
      <main className="flex flex-1 flex-col items-start justify-center gap-6 px-4 sm:px-8">
        <h2 className="text-[clamp(42px,8vw,64px)] leading-none font-bold tracking-tight">
          {conn === 'missing' ? t.four.notFound : t.four.outdated}
        </h2>
        <p className="max-w-xl text-accent">{conn === 'missing' ? t.four.notFoundDetail(id) : t.four.outdatedDetail}</p>
        <Button variant="solid" onClick={() => (conn === 'missing' ? navigate('/mods') : location.reload())}>
          {conn === 'missing' ? t.four.newGame : t.common.reload}
        </Button>
      </main>
    )
  }

  const seated = state ? SEATS.filter((s) => state.players[s].taken).length : 0
  const status = (() => {
    if (!state) return conn === 'failed' ? t.match.cantReach : t.match.connecting
    if (conn === 'failed') return t.match.cantReach
    if (conn !== 'open') return t.match.reconnecting
    if (state.status === 'waiting') return t.four.waitingTitle
    if (state.status === 'over') {
      if (state.result === 'draw') return t.four.draw
      const w = state.result as Seat
      return you === w ? t.four.youWin : t.four.wins(seatName(t, w))
    }
    if (youOut) return state.players[you!].out === 'capturing the king' ? t.four.kingEaten : t.four.youAreOut
    if (you) return toMove === you ? t.common.yourMove : t.four.toMove(seatName(t, state.turn))
    return t.four.toMove(seatName(t, state.turn))
  })()
  const overDetail = state?.status === 'over' && state.result !== 'draw' ? how(state.reason, t) : ''
  const danger = !!you && playing && !predicting && !youOut && !!state?.danger

  return (
    <main className="flex flex-1 justify-center px-4 pb-6 sm:px-8">
      <div className="flex w-full max-w-[1240px] flex-col gap-5 lg:flex-row lg:items-start lg:justify-center">
        <section className="four-wrap mx-auto flex flex-col gap-3 lg:mx-0">
          <div className="board-turn relative rounded-md" data-your-move={yourMove}>
            <FourBoard
              board={board.length ? board : emptyBoard}
              seat={seat}
              legal={legal}
              lastMove={last}
              danger={predicting ? undefined : state?.danger}
              hill={state?.settings.hill}
              center={state?.settings.centerVision ? { live: !predicting && !!state.fullVision } : undefined}
              look={state?.settings}
              onMove={onMove}
            />
            <SeatLabels state={state} seat={seat} />
            {state?.status === 'waiting' && (
              <WaitingCard
                id={id}
                seated={seated}
                canFill={!!you}
                filling={filling}
                onFill={() => {
                  setFilling(true)
                  send({ type: 'fill' })
                }}
              />
            )}
            {!state && (
              <div className="absolute inset-0 z-20 grid place-items-center rounded-md bg-bg/60 text-lg font-semibold">
                <Spinner label={t.match.joining} />
              </div>
            )}
          </div>
        </section>

        <aside className="flex w-full flex-col gap-4 lg:w-[340px] lg:pt-1">
          <div>
            <p className="flex items-center gap-2.5 text-2xl font-bold tracking-tight" aria-live="polite">
              {conn !== 'open' ? (
                <span className="spinner text-[0.75em]" aria-hidden />
              ) : (
                yourMove && <span className="turn-dot" aria-hidden />
              )}
              {status}
            </p>
            {overDetail && <p className="mt-1 text-sm text-accent">{overDetail}</p>}
          </div>
          {danger && (
            <p className="-mt-2 flex items-center gap-2 text-sm font-semibold text-danger" role="status">
              <span className="danger-dot" aria-hidden />
              {state?.dangerHidden ? t.match.kingThreatHidden : t.match.kingThreat}
            </p>
          )}

          <Seats state={state} you={you} toMove={toMove} />

          {!you && state && (
            <div className="flex flex-col gap-2">
              <div className="flex items-center gap-3 text-sm">
                <span className="text-accent">{t.four.seeAs}</span>
                <div role="radiogroup" aria-label={t.four.seeAs} className="flex gap-0.5 rounded-lg bg-muted p-[3px]">
                  {SEATS.map((s) => (
                    <button
                      key={s}
                      role="radio"
                      aria-checked={side === s}
                      onClick={() => setSide(s)}
                      className="flex h-8 items-center gap-1.5 rounded-[5px] px-2.5 text-[13px] font-medium transition-colors hover:bg-fg/10 aria-checked:bg-fg aria-checked:text-bg"
                    >
                      <span className="size-2.5 rounded-full" style={{ background: SEAT_COLORS[s] }} aria-hidden />
                      {seatName(t, s)}
                    </button>
                  ))}
                </div>
              </div>
              {!!state.behind && <p className="text-sm text-accent">{t.four.behind(state.behind)}</p>}
            </div>
          )}

          <div className="flex flex-wrap gap-2">
            {you && playing && !youOut && <Button onClick={() => setConfirmResign(true)}>{t.match.resign}</Button>}
            {(state?.status === 'over' || youOut || !you) && (
              <Button variant={state?.status === 'over' ? 'solid' : 'outline'} onClick={() => navigate('/mods')}>
                {t.four.newGame}
              </Button>
            )}
          </div>

          {state && <Moves state={state} />}
          {you && <TableChat lines={chat} you={you} onSend={(text) => send({ type: 'chat', text })} />}
        </aside>
      </div>

      <Dialog open={!!promotion} onOpenChange={(o) => !o && setPromotion(null)} title={t.match.promoteTo}>
        <div className="grid grid-cols-4 gap-2">
          {(['q', 'r', 'b', 'n'] as const).map((p) => (
            <button
              key={p}
              onClick={() => {
                sendMove(promotion!.from + promotion!.to + p)
                setPromotion(null)
              }}
              className="aspect-square rounded-lg bg-sq-light p-1 shadow-[inset_0_0_0_1.5px_var(--color-fg)] transition-transform hover:scale-105 active:scale-95"
              aria-label={p}
            >
              <img src={pieceImg4((you ?? 'r') + p.toUpperCase())} alt="" draggable={false} className="size-full" />
            </button>
          ))}
        </div>
      </Dialog>

      <Dialog
        open={confirmResign}
        onOpenChange={setConfirmResign}
        title={t.match.resignTitle}
        description={t.four.resignDesc}
      >
        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={() => setConfirmResign(false)}>
            {t.common.cancel}
          </Button>
          <Button
            variant="solid"
            onClick={() => {
              send({ type: 'resign' })
              setConfirmResign(false)
            }}
          >
            {t.match.resign}
          </Button>
        </div>
      </Dialog>
    </main>
  )
}

const emptyBoard: string[] = Array.from({ length: 196 }, (_, i) => {
  const x = i % 14
  const y = Math.floor(i / 14)
  return (x < 3 || x > 10) && (y < 3 || y > 10) ? ' ' : '.'
})

/** Each player's name in the empty corner nearest their pieces. */
function SeatLabels({ state, seat }: { state: FourState | null; seat: Seat }) {
  const t = useT()
  if (!state) return null
  // Bottom, left, top, right; each label sits in the corner just clockwise of its side.
  const corners = ['bottom-0 left-0', 'top-0 left-0', 'top-0 right-0', 'bottom-0 right-0']
  return (
    <>
      {aroundFrom(seat).map((s, i) => {
        const p = state.players[s]
        const active = state.status === 'playing' && state.turn === s
        return (
          <div
            key={s}
            className={`seat-corner absolute ${corners[i]} flex flex-col justify-center gap-0.5 p-1.5 text-[clamp(10px,calc(var(--board4)/48),14px)] leading-tight`}
            data-active={active}
            data-out={!!p.out}
          >
            <span className="flex min-w-0 items-center gap-1.5 font-semibold">
              <span className="size-2.5 shrink-0 rounded-full" style={{ background: SEAT_COLORS[s] }} aria-hidden />
              <span className="truncate">{p.taken ? (p.name ?? t.common.guest) : t.four.openSeat}</span>
            </span>
            {state.clock && (
              <Clock
                left={state.clock.left[s]}
                running={state.clock.running === s}
                since={state.receivedAt}
                daily={!!state.settings.time.perMove}
              />
            )}
            {p.out && <span className="text-accent">{t.four.outBy[p.out] ?? t.four.out}</span>}
          </div>
        )
      })}
    </>
  )
}

function Seats({ state, you, toMove }: { state: FourState | null; you: Seat | null; toMove: Seat | null }) {
  const t = useT()
  if (!state) return null
  return (
    <ul className="flex flex-col gap-1.5">
      {SEATS.map((s) => {
        const p = state.players[s]
        return (
          <li
            key={s}
            className="player-bar flex h-11 items-center gap-2 rounded-lg pr-1.5 pl-3 text-[15px]"
            data-active={toMove === s}
            data-out={!!p.out}
          >
            <span
              className="size-3.5 shrink-0 rounded-full shadow-[0_0_0_1.5px_var(--color-fg)]"
              style={{ background: SEAT_COLORS[s] }}
            />
            <span className="min-w-0 truncate font-semibold">
              {p.taken ? (p.name ?? t.common.guest) : t.four.openSeat}
            </span>
            {s === you && <span className="text-accent">{t.match.youTag}</span>}
            {p.bot && <span className="text-accent">{t.four.bot}</span>}
            <span className="ml-auto flex min-w-0 items-center gap-2">
              {p.out ? (
                <span className="text-sm text-accent">{t.four.outBy[p.out] ?? t.four.out}</span>
              ) : (
                <>
                  {p.taken && !p.online && <span className="text-sm text-danger">{t.match.offline}</span>}
                  {toMove === s && !state.clock && <span className="turn-tag">{t.common.toMove}</span>}
                  {state.clock && (
                    <Clock
                      left={state.clock.left[s]}
                      running={state.clock.running === s}
                      since={state.receivedAt}
                      daily={!!state.settings.time.perMove}
                    />
                  )}
                </>
              )}
            </span>
          </li>
        )
      })}
    </ul>
  )
}

function Moves({ state }: { state: FourState }) {
  const t = useT()
  const list = useRef<HTMLOListElement>(null)
  useEffect(() => {
    list.current?.scrollTo({ top: list.current.scrollHeight })
  }, [state.history.length])
  return (
    <section className="flex flex-col gap-2">
      <h2 className="font-semibold">{t.four.moves}</h2>
      {state.history.length === 0 ? (
        <p className="text-sm text-accent">{t.four.noMoves}</p>
      ) : (
        <ol
          ref={list}
          className="grid max-h-[30dvh] grid-cols-4 gap-x-2 gap-y-1 overflow-y-auto font-mono text-[13px] tabular-nums lg:max-h-[34dvh]"
        >
          {state.history.map((e, i) => (
            <li key={i} className="flex min-w-0 items-center gap-1.5" title={t.four.seats[e.by]}>
              <span className="size-2 shrink-0 rounded-full" style={{ background: SEAT_COLORS[e.by] }} aria-hidden />
              <span className="truncate">
                {e.out ? (e.out === 'resignation' ? t.four.resigned : t.four.outOfTime) : e.san}
              </span>
            </li>
          ))}
        </ol>
      )}
    </section>
  )
}

function TableChat({
  lines,
  you,
  onSend,
}: {
  lines: { id: number; from: Seat; text: string }[]
  you: Seat
  onSend: (text: string) => boolean
}) {
  const t = useT()
  const [text, setText] = useState('')
  const list = useRef<HTMLOListElement>(null)
  useEffect(() => {
    list.current?.scrollTo({ top: list.current.scrollHeight })
  }, [lines.length])
  return (
    <section className="flex flex-col gap-2">
      <h2 className="font-semibold">{t.four.chat}</h2>
      {lines.length > 0 && (
        <ol ref={list} className="flex max-h-48 flex-col gap-1 overflow-y-auto text-[15px]">
          {lines.map((l) => (
            <li key={l.id} className="chat-line flex gap-2">
              <span
                className="mt-[7px] size-2 shrink-0 rounded-full"
                style={{ background: SEAT_COLORS[l.from] }}
                aria-hidden
              />
              <span className={l.from === you ? 'font-medium' : ''}>
                <span className="sr-only">{t.four.seats[l.from]}: </span>
                {l.text}
              </span>
            </li>
          ))}
        </ol>
      )}
      <form
        className="flex gap-2"
        onSubmit={(e) => {
          e.preventDefault()
          if (text.trim() && onSend(text)) setText('')
        }}
      >
        <input
          value={text}
          onChange={(e) => setText(e.target.value)}
          maxLength={500}
          placeholder={t.four.chatPlaceholder}
          aria-label={t.four.chatPlaceholder}
          className="h-11 min-w-0 flex-1 rounded-md bg-muted px-3 text-[15px] outline-none focus-visible:shadow-[inset_0_0_0_1.5px_var(--color-fg)]"
        />
        <Button type="submit" disabled={!text.trim()}>
          {t.four.send}
        </Button>
      </form>
    </section>
  )
}

function WaitingCard({
  id,
  seated,
  canFill,
  filling,
  onFill,
}: {
  id: string
  seated: number
  canFill: boolean
  filling: boolean
  onFill: () => void
}) {
  const t = useT()
  const [copied, setCopied] = useState(false)
  const link = `${location.origin}/f/${id}`
  const share = async () => {
    try {
      if (navigator.share && matchMedia('(pointer: coarse)').matches) {
        await navigator.share({ title: t.match.shareTitle, url: link })
        return
      }
      await navigator.clipboard.writeText(link)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {}
  }
  return (
    <div className="absolute inset-0 z-20 grid place-items-center bg-bg/70 p-4 backdrop-blur-[3px]">
      <div className="flex w-full max-w-[340px] flex-col items-center gap-4 rounded-xl bg-bg p-6 text-center shadow-[0_0_0_1.5px_var(--color-fg)]">
        <p className="text-accent">{t.four.sendCode}</p>
        <p className="font-mono text-4xl font-bold tracking-[0.2em] select-all">{id}</p>
        <p className="text-sm text-accent">{t.four.waiting(seated)}</p>
        <Button variant="solid" className="w-full" onClick={share}>
          {copied ? t.match.copied : t.match.shareLink}
        </Button>
        {canFill && (
          <Button className="w-full" onClick={onFill} loading={filling}>
            {t.four.fillBots}
          </Button>
        )}
      </div>
    </div>
  )
}
