import { useCallback, useEffect, useRef, useState } from 'react'
import { refreshMe } from './auth'
import { Board } from './Board'
import { Captured } from './Captured'
import { Chat } from './Chat'
import { Clock } from './Clock'
import { useT, type Dict } from './i18n'
import { Icon } from './icons'
import { applyLocal, pieceImg } from './position'
import { navigate } from './router'
import { describeSettings } from './settings'
import { SetupBoard } from './SetupBoard'
import { Button, Dialog, Spinner, Tip } from './ui'
import { useMatch, type Color, type MatchState } from './useMatch'

const other = (c: Color): Color => (c === 'w' ? 'b' : 'w')
const sideName = (t: Dict, c: Color) => (c === 'w' ? t.common.white : t.common.black)

function resultText(s: MatchState, t: Dict) {
  const h = how(s.reason, t)
  if (s.result === '1/2-1/2') return { title: t.match.draw, detail: t.match.drawDetail(h) }
  const winner: Color = s.result === '1-0' ? 'w' : 'b'
  if (s.sniper) {
    const title = s.you && s.you !== winner ? t.match.sniped : t.match.sniperWin
    const detail = t.match.sniperDetail(sideName(t, winner), h)
    return { title, detail: s.rated ? `${detail} ${t.match.sniperRating}` : detail }
  }
  const title = s.you ? (s.you === winner ? t.match.youWin : t.match.youLose) : t.match.sideWins(sideName(t, winner))
  return { title, detail: t.match.winsDetail(sideName(t, winner), h) }
}

/** "on time", or "by checkmate" and the like, in the user's language. */
export function how(reason: string | undefined, t: Dict) {
  return (reason && t.by[reason]) || `by ${reason}`
}

export function MatchPage({ id }: { id: string }) {
  const { state, conn, send, rtt, chat } = useMatch(id)
  const t = useT()
  const [pending, setPending] = useState<{ uci: string; sentWith: MatchState; sentAt: number } | null>(null)
  const [promotion, setPromotion] = useState<{ from: string; to: string } | null>(null)
  const [flipped, setFlipped] = useState(false)
  // Which side a spectator sees the game as.
  const [side, setSide] = useState<Color>('w')
  const [confirmResign, setConfirmResign] = useState(false)
  const [dismissedOver, setDismissedOver] = useState('')
  const [moveMs, setMoveMs] = useState<number | null>(null)

  // Show our own move immediately. Any newer state from the server ends the
  // prediction, whether it confirms the move or rejects it.
  const predicting = !!state && !!pending && pending.sentWith === state
  const board = state ? (predicting ? applyLocal(state.board, pending!.uci) : state.board) : '.'.repeat(64)
  const legal = state && !predicting && conn === 'open' ? state.legal : []
  const last = predicting ? { from: pending!.uci.slice(0, 2), to: pending!.uci.slice(2, 4) } : state?.last

  // Once the game is rated, the rating in the top bar is out of date.
  const rated = !!state?.ratingDiff
  useEffect(() => {
    if (rated) refreshMe()
  }, [rated])

  // How long the server took to confirm our move, for tracking sync lag.
  useEffect(() => {
    if (state && pending && pending.sentWith !== state) {
      if (state.ply > pending.sentWith.ply) setMoveMs(Math.round(performance.now() - pending.sentAt))
      setPending(null)
    }
  }, [state, pending])

  const you = state?.you || null
  const them = you ? other(you) : null
  const orientation: Color = you ? ((you === 'b') !== flipped ? 'b' : 'w') : side
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

  // The server starts every spectator as White, including after a reconnect.
  const watching = state?.watching
  useEffect(() => {
    if (watching && watching !== side) send({ type: 'watch', side })
  }, [watching, side, send])

  // Say whose move it is in the tab title, so a backgrounded tab shows when it's your turn.
  useEffect(() => {
    if (!you || !playing) return
    document.title = yourMove ? t.match.titleYourMove : t.match.titleTheirMove
    return () => {
      document.title = t.common.appName
    }
  }, [you, playing, yourMove, t])

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
          {conn === 'missing' ? t.match.notFound : t.match.outdated}
        </h2>
        <p className="max-w-xl text-accent">
          {conn === 'missing' ? t.match.notFoundDetail(id) : t.match.outdatedDetail}
        </p>
        <Button variant="solid" onClick={() => navigate('/')}>
          {t.common.backHome}
        </Button>
      </main>
    )
  }

  const overKey = state ? `${state.round}:${state.ply}` : ''
  const over = state?.status === 'over' ? resultText(state, t) : null
  // The fog lifts and the last move shows for a moment before the dialog covers the board,
  // so a player sees what took their king.
  const [showOver, setShowOver] = useState('')
  useEffect(() => {
    if (!over) return
    const id = setTimeout(() => setShowOver(overKey), state!.last && state!.ply > 0 ? 1400 : 0)
    return () => clearTimeout(id)
  }, [over, overKey, state])
  const danger = !!you && playing && !predicting && !!state?.danger

  const status = (() => {
    if (!state) return conn === 'failed' ? t.match.cantReach : t.match.connecting
    if (conn === 'failed') return t.match.cantReach
    if (conn !== 'open') return t.match.reconnecting
    if (state.status === 'waiting') return t.match.waitingOpponent
    if (state.status === 'setup') {
      if (!you || !them) return t.match.playersSettingUp
      return state.setup?.ready[you] ? t.match.waitingTheirSetup : t.match.setUpYourPieces
    }
    if (over)
      return state.result === '1/2-1/2'
        ? t.match.drawStatus(how(state.reason, t))
        : you && you !== (state.result === '1-0' ? 'w' : 'b') && state.reason === 'capturing the king'
          ? t.match.kingEaten
          : `${over.title} ${how(state.reason, t)}`
    const check = state.check && !predicting && !state.behind ? t.match.check : ''
    if (!you) return `${check}${t.match.sideToMove(sideName(t, turn!))}`
    return `${check}${turn === you ? t.common.yourMove : t.common.theirMove}`
  })()

  const bottom = orientation
  const top = other(orientation)

  return (
    <main className="flex flex-1 justify-center px-4 pb-6 sm:px-8">
      <div className="flex w-full max-w-[1100px] flex-col gap-6 lg:flex-row lg:items-start lg:justify-center">
        <section
          className="board-wrap mx-auto flex flex-col gap-3 lg:mx-0"
          data-setup={state?.status === 'setup' && !!you}
        >
          <PlayerBar color={top} state={state} you={you} active={toMove === top} />
          {state?.status === 'setup' && you ? (
            <SetupBoard
              key={`${state.round}:${you}`}
              settings={state.settings}
              you={you}
              ready={!!state.setup?.ready[you]}
              mine={state.setup?.mine}
              theirs={state.setup?.theirs}
              onReady={(board) => send({ type: 'setup', board })}
              onChange={() => send({ type: 'setup', board: '' })}
              below={<PlayerBar color={you} state={state} you={you} active={false} />}
            />
          ) : (
            <>
              <div className="board-turn relative rounded-md" data-your-move={yourMove}>
                <Board
                  board={board}
                  orientation={orientation}
                  legal={legal}
                  lastMove={last}
                  check={predicting ? undefined : state?.check}
                  danger={predicting ? undefined : state?.danger}
                  unsafe={state?.unsafe}
                  hill={state?.settings.hill}
                  onMove={onMove}
                />
                {state?.status === 'waiting' && <WaitingCard id={id} link={link} />}
                {state?.status === 'setup' && <SetupWait ready={state.setup?.ready} />}
                {!state && (
                  <div className="absolute inset-0 z-20 grid place-items-center rounded-md bg-bg/60 text-lg font-semibold">
                    <Spinner label={t.match.joining} />
                  </div>
                )}
              </div>
              <PlayerBar color={bottom} state={state} you={you} active={toMove === bottom} />
            </>
          )}
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
          {danger && (
            <p className="-mt-2 flex items-center gap-2 text-sm font-semibold text-danger" role="status">
              <span className="danger-dot" aria-hidden />
              {t.match.kingThreat}
            </p>
          )}

          {state && (
            <ul className="flex flex-wrap gap-1.5" aria-label={t.match.matchRules}>
              {describeSettings(state.settings, t).map((r) => (
                <li key={r} className="tag">
                  {r}
                </li>
              ))}
              <li className={`tag ${state.rated ? 'tag-solid' : ''}`}>
                {state.rated ? t.common.rated : t.common.unrated}
              </li>
              {state.spectators > 0 && (
                <li className="tag flex items-center gap-1.5">
                  <Icon name="eye" size={14} />
                  {t.match.watching(state.spectators)}
                </li>
              )}
            </ul>
          )}

          {!you && state && state.status !== 'waiting' && <Spectating state={state} side={side} setSide={setSide} />}

          {state && you && them && state.draw[them] && playing && (
            <div className="flex items-center justify-between gap-3 rounded-lg p-3 shadow-[inset_0_0_0_1.5px_var(--color-fg)]">
              <span>{t.match.theyOfferDraw}</span>
              <Button variant="solid" className="h-9 px-4" onClick={() => send({ type: 'draw' })}>
                {t.common.accept}
              </Button>
            </div>
          )}

          {state?.status === 'setup' && you && them && (
            <p className="text-sm text-accent">
              {state.setup?.ready[them] ? `${t.match.opponentReady} ` : ''}
              {setupHint(state, t)}
            </p>
          )}

          {state && them && state.rematch[them] && state.status === 'over' && !state.rematch[you!] && (
            <p className="font-semibold">{t.match.theyWantRematch}</p>
          )}

          <div className="flex flex-wrap gap-2">
            {state?.status === 'over' && you ? (
              <Button variant={state.rematch[you] ? 'outline' : 'solid'} onClick={() => send({ type: 'rematch' })}>
                {state.rematch[you] ? t.match.cancelRematch : t.match.rematch}
              </Button>
            ) : (
              you &&
              playing && (
                <>
                  <Button onClick={() => send({ type: 'draw' })}>
                    {state!.draw[you] ? t.match.cancelDraw : t.match.offerDraw}
                  </Button>
                  <Button onClick={() => setConfirmResign(true)} disabled={ply === 0}>
                    {t.match.resign}
                  </Button>
                </>
              )
            )}
            <Tip label={t.match.flipBoard}>
              <Button
                variant="ghost"
                icon
                aria-label={t.match.flipBoard}
                onClick={() => (you ? setFlipped((f) => !f) : setSide(other(side)))}
              >
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

          {rtt !== null && (
            <p className="font-mono text-xs text-accent tabular-nums">
              {t.match.ping(rtt)}
              {moveMs !== null && ` · ${t.match.lastMoveConfirmed(moveMs)}`}
            </p>
          )}
        </aside>
      </div>

      <Dialog open={!!promotion} onOpenChange={(o) => !o && setPromotion(null)} title={t.match.promoteTo}>
        <div className="grid grid-cols-4 gap-2">
          {(['q', 'r', 'b', 'n'] as const).map((p) => (
            <button
              key={p}
              autoFocus={p === 'q'}
              aria-label={{ q: t.match.queen, r: t.match.rook, b: t.match.bishop, n: t.match.knight }[p]}
              className="aspect-square rounded-lg bg-sq-light p-1 shadow-[inset_0_0_0_1.5px_var(--color-fg)] transition-transform hover:scale-105 active:scale-95"
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
        title={t.match.resignTitle}
        description={t.match.resignDesc}
      >
        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={() => setConfirmResign(false)}>
            {t.match.keepPlaying}
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

      <Dialog
        open={!!over && showOver === overKey && dismissedOver !== overKey}
        onOpenChange={(o) => !o && setDismissedOver(overKey)}
        title={over?.title ?? ''}
        description={
          <>
            {over?.detail}
            {state?.rated && you && (
              <span className="mt-3 block font-mono text-fg tabular-nums">
                {state.ratingDiff ? t.match.rating(signed(state.ratingDiff[you])) : t.match.updatingRating}
              </span>
            )}
          </>
        }
      >
        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={() => setDismissedOver(overKey)}>
            {t.match.viewBoard}
          </Button>
          {you && (
            <Button
              variant="solid"
              onClick={() => {
                if (!state!.rematch[you]) send({ type: 'rematch' })
                setDismissedOver(overKey)
              }}
            >
              {t.match.rematch}
            </Button>
          )}
        </div>
      </Dialog>
    </main>
  )
}

type PlayerBarProps = { color: Color; state: MatchState | null; you: Color | null; active: boolean }

function PlayerBar({ color, state, you, active }: PlayerBarProps) {
  const t = useT()
  const seat = state?.players[color]
  const name = seat?.name
  const label = !seat?.taken
    ? t.match.waitingSeat
    : name || (color === you ? t.common.you : you ? t.common.opponent : t.common.guest)
  return (
    <div className="player-bar flex h-12 items-center gap-2 rounded-lg pr-1.5 pl-3 text-[15px]" data-active={active}>
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
      {name && color === you && <span className="text-accent">{t.match.youTag}</span>}
      <span className="text-accent max-sm:hidden">{sideName(t, color)}</span>
      {active && !state?.clock && <span className="turn-tag">{t.common.toMove}</span>}
      <span className="ml-auto flex min-w-0 items-center gap-2">
        {seat?.taken && !seat.online && <span className="text-sm text-danger">{t.match.offline}</span>}
        <Captured pieces={state?.captured[color] ?? ''} className="text-[clamp(16px,calc(var(--board)/30),22px)]" />
        {state?.clock && (
          <Clock
            left={state.clock.left[color]}
            running={state.clock.running === color}
            since={state.receivedAt}
            daily={state.settings.time.perMove > 0}
          />
        )}
      </span>
    </div>
  )
}

/** A spectator picks whose side to see the game from. In a fog game that's
 * what that player saw a few moves ago, so nobody watching can help them. */
function Spectating({ state, side, setSide }: { state: MatchState; side: Color; setSide: (c: Color) => void }) {
  const t = useT()
  const behind = state.behind ?? 0
  return (
    <div className="flex flex-col gap-2.5">
      <div className="flex items-center gap-3 text-sm">
        <span>{t.match.seeItAs}</span>
        <div role="radiogroup" aria-label={t.match.seeAsAria} className="flex gap-0.5 rounded-lg bg-muted p-[3px]">
          {(['w', 'b'] as const).map((c) => (
            <button
              key={c}
              role="radio"
              aria-checked={side === c}
              onClick={() => setSide(c)}
              className="h-8 rounded-[5px] px-3 text-[13px] font-medium transition-colors hover:bg-fg/10 aria-checked:bg-fg aria-checked:text-bg"
            >
              {sideName(t, c)}
            </button>
          ))}
        </div>
      </div>
      <p className="text-sm text-accent">
        {behind > 0 ? t.match.behindNote(sideName(t, side), behind) : t.match.watchingNote}
      </p>
    </div>
  )
}

/** What the setup rules are, for the player arranging their pieces. */
function setupHint(s: MatchState, t: Dict) {
  const rows = s.settings.setup === 'backRow' ? t.match.rowsBack : t.match.rowsFour
  const hill = s.settings.hill ? ` ${t.match.hillNoStart}` : ''
  const hidden = s.settings.fog ? t.match.fogHidden : t.match.noFogShown
  return `${t.match.setupHint(rows)}${hill} ${hidden} ${t.match.clockStarts}`
}

/** Covers the board for a spectator while the players set up. */
function SetupWait({ ready }: { ready?: Record<Color, boolean> }) {
  const t = useT()
  return (
    <div className="absolute inset-0 z-20 grid place-items-center bg-bg/70 p-4 backdrop-blur-[3px]">
      <div className="flex w-full max-w-[320px] flex-col items-center gap-3 rounded-xl bg-bg p-6 text-center shadow-[0_0_0_1.5px_var(--color-fg)]">
        <p className="text-lg font-semibold">{t.match.settingUpTitle}</p>
        <ul className="flex gap-4 text-sm text-accent">
          {(['w', 'b'] as const).map((c) => (
            <li key={c}>
              {sideName(t, c)}: {ready?.[c] ? t.match.ready : t.match.placing}
            </li>
          ))}
        </ul>
      </div>
    </div>
  )
}

/** "+12", "−8" or "±0". */
function signed(n: number) {
  return n > 0 ? `+${n}` : n < 0 ? `−${-n}` : '±0'
}

function WaitingCard({ id, link }: { id: string; link: string }) {
  const t = useT()
  return (
    <div className="absolute inset-0 z-20 grid place-items-center bg-bg/70 p-4 backdrop-blur-[3px]">
      <div className="flex w-full max-w-[320px] flex-col items-center gap-4 rounded-xl bg-bg p-6 text-center shadow-[0_0_0_1.5px_var(--color-fg)]">
        <p className="text-accent">{t.match.sendCode}</p>
        <p className="font-mono text-4xl font-bold tracking-[0.2em] select-all">{id}</p>
        <CopyLink link={link} wide />
      </div>
    </div>
  )
}

function CopyLink({ link, wide }: { link: string; wide?: boolean }) {
  const t = useT()
  const [copied, setCopied] = useState(false)
  const share = async () => {
    try {
      if (wide && navigator.share && matchMedia('(pointer: coarse)').matches) {
        await navigator.share({ title: t.match.shareTitle, url: link })
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
        {copied ? t.match.copied : t.match.shareLink}
      </Button>
    )
  }
  return (
    <Tip label={copied ? t.match.copied : t.match.copyLink}>
      <Button variant="ghost" icon aria-label={t.match.copyLink} onClick={share}>
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
