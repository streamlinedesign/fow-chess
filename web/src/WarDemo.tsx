import { useEffect, useState } from 'react'
import { AvatarImg } from './Avatar'
import { geo, parseMove, pieceImg4, SEAT_COLORS, type Seat } from './four'
import { FourBoard } from './FourBoard'
import { fourLook, Moves } from './FourMatchPage'
import { FourSetup } from './FourSetup'
import { useT } from './i18n'
import { usePrefs } from './prefs'
import { PromotionDialog } from './Promotion'
import { navigate } from './router'
import { avatarImg } from './sets'
import { BOT_FACE, type Bot } from './settings'
import { useFourMatch, type FourState } from './useFourMatch'

type Props = {
  id: string | null
  failed: boolean
  me?: string
  bot: Bot
  name: string
  picker: React.ReactNode
  onNew: () => void
}

/**
 * War Mode on the home page board: you as Red against three copies of the
 * bot you pick, on the extended board with Battleship and jester pawns,
 * taking turns. It opens on your setup; the bots have placed theirs already.
 */
export function WarDemo(props: Props) {
  return props.id ? <Live {...props} id={props.id} /> : <Table {...props} state={null} />
}

function Live(props: Props & { id: string }) {
  const { state, conn, send } = useFourMatch(props.id)
  // The move you sent this round, until the server's next state says so itself.
  const [sent, setSent] = useState<{ uci: string; with: FourState } | null>(null)
  useEffect(() => {
    if (sent && state !== sent.with) setSent(null)
  }, [state, sent])
  return (
    <Table
      {...props}
      state={conn === 'missing' ? null : state}
      failed={props.failed || conn === 'missing' || conn === 'failed'}
      sent={sent?.uci}
      onMove={(uci) => state && send({ type: 'move', uci }) && setSent({ uci, with: state })}
      onSetup={(board) => send({ type: 'setup', board })}
    />
  )
}

function Table({
  state,
  failed,
  me,
  bot,
  name,
  picker,
  onNew,
  sent,
  onMove,
  onSetup,
}: Props & {
  state: FourState | null
  sent?: string
  onMove?: (uci: string) => void
  onSetup?: (board: string) => void
}) {
  const t = useT()
  const prefs = usePrefs()
  const [promotion, setPromotion] = useState<{ from: string; to: string } | null>(null)
  const you: Seat = state?.you || 'r'
  const look = fourLook(state, you, prefs)
  const playing = state?.status === 'playing'
  const out = !!state?.players[you]?.out
  const planned = sent || state?.planned || ''
  const legal = state && playing && !planned ? state.legal : []
  const yourMove = playing && !out && legal.length > 0
  const thinking = playing && !!planned
  const blank = geo('extended')
    .startBoard()
    .map((v) => (v === ' ' ? ' ' : '.'))

  const status = (() => {
    if (failed) return t.home.cantStart
    if (!state) return null
    if (state.status === 'over') {
      if (state.result === 'draw') return t.four.draw
      return state.result === you ? t.four.youWin : t.four.wins(t.four.seats[state.result as Seat])
    }
    if (out) return t.four.kingEaten
    return null
  })()
  const line = !state
    ? null
    : state.status === 'setup'
      ? t.home.placing
      : playing && !out
        ? planned
          ? t.four.sent
          : t.four.pickMove
        : null

  const move = (from: string, to: string) => {
    if (legal.includes(from + to + 'q')) setPromotion({ from, to })
    else onMove?.(from + to)
  }

  return (
    <div className="demo">
      <div className="demo-main">
        <div className="strip">
          <span className="strip-ava bot" aria-hidden>
            <img src={BOT_FACE[bot]} alt="" />
          </span>
          <span className="strip-who">
            <span className="strip-name">{t.home.threeBots(name)}</span>
            <span className="strip-meta">
              {t.home.demoBots}
              {thinking && (
                <>
                  {' · '}
                  {t.home.thinking}
                  <span className="thinking" aria-hidden>
                    <b />
                    <b />
                    <b />
                  </span>
                </>
              )}
            </span>
          </span>
        </div>
        <div className="relative">
          {state?.status === 'setup' && onSetup ? (
            <div className="war-setup">
              <FourSetup
                seat={you}
                layout="battleship"
                board="extended"
                look={look}
                mine={state.arranged}
                onReady={onSetup}
                onChange={() => onSetup('')}
              />
            </div>
          ) : (
            <div className="board-turn relative rounded-md" data-your-move={yourMove}>
              <FourBoard
                board={state?.board.length ? state.board : blank}
                seat={you}
                legal={legal}
                lastMove={state?.last}
                lastMoves={state?.lastRound}
                planned={planned ? parseMove(planned) : null}
                danger={state?.danger}
                reach={state?.reach}
                center={state?.settings.centerVision ? { live: !!state.fullVision } : undefined}
                look={look ?? { jesterPawns: true }}
                onMove={move}
              />
            </div>
          )}
          {status && (
            <div className="demo-over" role="status">
              {status}
            </div>
          )}
        </div>
        <div className="strip">
          <span className="strip-ava you" aria-hidden>
            {avatarImg(prefs.avatar) ? <AvatarImg code={prefs.avatar} /> : me ? me[0].toUpperCase() : '?'}
          </span>
          <span className="strip-who">
            <span className="strip-name">{me ?? t.common.you}</span>
            <span className="strip-meta">
              <span
                className="mr-1.5 inline-block size-2.5 rounded-full align-[-1px]"
                style={{ background: SEAT_COLORS[you] }}
                aria-hidden
              />
              {t.four.seats[you]}
              {line && ` · ${line}`}
            </span>
          </span>
        </div>
      </div>
      <aside className="demo-side">
        {picker}
        {state && state.status !== 'setup' && <Moves state={state} />}
        <div className="flex flex-wrap gap-2">
          <button type="button" className="demo-btn" onClick={onNew}>
            <span aria-hidden>↻</span> {t.home.newGame}
          </button>
          <a
            href="/settings#s-pieces"
            className="demo-btn"
            onClick={(e) => {
              e.preventDefault()
              navigate('/settings#s-pieces')
            }}
          >
            {t.pieces.link}
          </a>
        </div>
        <p className="text-[13.5px] text-accent">{t.home.warTip}</p>
      </aside>
      <PromotionDialog
        open={!!promotion}
        img={(p) => pieceImg4(you + p.toUpperCase(), look)}
        onPick={(p) => promotion && onMove?.(promotion.from + promotion.to + p)}
        onClose={() => setPromotion(null)}
      />
    </div>
  )
}
