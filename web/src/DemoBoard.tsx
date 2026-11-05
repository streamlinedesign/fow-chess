import * as RadioGroup from '@radix-ui/react-radio-group'
import { useCallback, useEffect, useState } from 'react'
import { Board } from './Board'
import { useT } from './i18n'
import { applyLocal, pieceImg } from './position'
import { playerId } from './player'
import { BOT_PIECE, BOTS, type Bot } from './settings'
import { useMatch, type Color, type MatchState } from './useMatch'

/** Your view of the start under fog: your four rows, the rest hidden. Drawn until the game connects. */
const startView = (side: Color) => {
  const start = 'RNBQKBNRPPPPPPPP' + '.'.repeat(32) + 'pppppppprnbqkbnr'
  return side === 'w' ? start.slice(0, 32) + '?'.repeat(32) : '?'.repeat(32) + start.slice(32)
}

/** The demo board's opponent unless the player picked another. */
export const DEMO_BOT: Bot = 'solomon'

function savedBot(): Bot {
  try {
    const b = localStorage.getItem('demoBot')
    if (BOTS.some((x) => x.level === b)) return b as Bot
  } catch {}
  return DEMO_BOT
}

/**
 * The home page's board: a game against a bot (Solomon unless you pick
 * another) that's playable the moment the page loads. The server keeps it in
 * memory only, with no clock, and never saves, lists or ranks it.
 */
export function DemoBoard({ me }: { me?: string }) {
  const t = useT()
  const [side, setSide] = useState<Color>('w')
  const [bot, setBot] = useState<Bot>(savedBot)
  const [id, setId] = useState<string | null>(null)
  const [failed, setFailed] = useState(false)

  const start = useCallback((s: Color, b: Bot) => {
    setSide(s)
    setBot(b)
    setId(null)
    setFailed(false)
    try {
      localStorage.setItem('demoBot', b)
    } catch {}
    fetch(`/api/matches?player=${playerId}`, {
      method: 'POST',
      body: JSON.stringify({ demo: true, side: s, bot: b }),
    })
      .then((res) => (res.ok ? res.json() : Promise.reject()))
      .then((b) => setId(b.id))
      .catch(() => setFailed(true))
  }, [])

  useEffect(() => start('w', savedBot()), [start])

  const other: Color = side === 'w' ? 'b' : 'w'
  const actions = (
    <div className="flex flex-wrap gap-2">
      <button type="button" className="demo-btn" onClick={() => start(side, bot)}>
        <span aria-hidden>↻</span> {t.home.newGame}
      </button>
      <button type="button" className="demo-btn" onClick={() => start(other, bot)}>
        {t.home.playAs(other === 'w' ? t.common.white : t.common.black)}
      </button>
    </div>
  )
  const picker = (
    <div className="demo-opp">
      <h3 id="demo-opp">{t.home.opponent}</h3>
      <RadioGroup.Root
        aria-labelledby="demo-opp"
        value={bot}
        onValueChange={(b) => start(side, b as Bot)}
        className="demo-bots"
      >
        {BOTS.map((b, i) => (
          <RadioGroup.Item key={b.level} value={b.level} className="demo-bot">
            <span className="demo-bot-face" aria-hidden>
              <img src={pieceImg(BOT_PIECE[b.level])} alt="" />
            </span>
            <span className="min-w-0 flex-1">
              <span className="block font-semibold">{b.name}</span>
              <span className="block truncate text-[13px] text-accent">{t.landing.botHints[b.level]}</span>
            </span>
            <span className="strength" aria-label={`${i + 1}/4`}>
              {BOTS.map((_, j) => (
                <i key={j} data-on={j <= i} />
              ))}
            </span>
          </RadioGroup.Item>
        ))}
      </RadioGroup.Root>
    </div>
  )
  const name = BOTS.find((b) => b.level === bot)!.name
  const extra = { bot, name, picker }

  return id ? (
    <Live key={id} id={id} side={side} me={me} actions={actions} {...extra} />
  ) : (
    <Frame
      {...extra}
      me={me}
      you={side}
      board={startView(side)}
      legal={[]}
      onMove={() => {}}
      history={[]}
      status={failed ? t.home.cantStart : null}
      thinking={false}
      actions={actions}
    />
  )
}

type Extra = { bot: Bot; name: string; picker: React.ReactNode }

function Live({
  id,
  side,
  me,
  actions,
  ...extra
}: { id: string; side: Color; me?: string; actions: React.ReactNode } & Extra) {
  const t = useT()
  const { state, send } = useMatch(id)
  // Our own move shows at once; the server's next state replaces it.
  const [pending, setPending] = useState<{ uci: string; ply: number } | null>(null)
  const shown = pending && state && state.ply === pending.ply ? pending : null

  if (!state)
    return (
      <Frame
        {...extra}
        me={me}
        you={side}
        board={startView(side)}
        legal={[]}
        onMove={() => {}}
        history={[]}
        status={null}
        thinking={false}
        actions={actions}
      />
    )
  const you = (state.you || 'w') as Color
  const move = (from: string, to: string) => {
    const legal = state.legal
    const uci = legal.includes(from + to) ? from + to : legal.includes(from + to + 'q') ? from + to + 'q' : null
    if (uci && send({ type: 'move', uci })) setPending({ uci, ply: state.ply })
  }
  return (
    <Frame
      {...extra}
      me={me}
      you={you}
      board={shown ? applyLocal(state.board, shown.uci) : state.board}
      legal={shown ? [] : state.legal}
      state={state}
      onMove={move}
      history={state.history}
      status={over(state, you, t, extra.name)}
      thinking={!!shown || (state.status === 'playing' && state.turn !== you)}
      actions={actions}
    />
  )
}

function over(state: MatchState, you: Color, t: ReturnType<typeof useT>, bot: string) {
  if (state.status !== 'over') return null
  if (state.result === '1/2-1/2') return t.home.drawn
  const won = (state.result === '1-0') === (you === 'w')
  return won ? t.home.youWon : t.home.youLost(bot.split(' ')[0])
}

function Frame({
  bot,
  name,
  picker,
  me,
  you,
  board,
  legal,
  state,
  onMove,
  history,
  status,
  thinking,
  actions,
}: {
  me?: string
  you: Color
  board: string
  legal: string[]
  state?: MatchState
  onMove: (from: string, to: string) => void
  history: string[]
  status: string | null
  thinking: boolean
  actions: React.ReactNode
} & Extra) {
  const t = useT()
  const yourTurn = !status && legal.length > 0
  const rows: [string, string?][] = []
  // A game where you're Black starts with White's hidden move.
  for (let i = 0; i < history.length; i += 2) rows.push([history[i], history[i + 1]])
  return (
    <div className="demo">
      <div className="demo-main">
        <div className="strip">
          <span className="strip-ava bot" aria-hidden>
            <img src={pieceImg(BOT_PIECE[bot])} alt="" />
          </span>
          <span className="strip-who">
            <span className="strip-name">{name}</span>
            <span className="strip-meta">
              {t.home.demoBot}
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
          <Board
            board={board}
            orientation={you}
            legal={legal}
            lastMove={state?.last}
            danger={state?.danger}
            unsafe={state?.unsafe}
            onMove={onMove}
          />
          {status && (
            <div className="demo-over" role="status">
              {status}
            </div>
          )}
        </div>
        <div className="strip">
          <span className="strip-ava you" aria-hidden>
            {me ? me[0].toUpperCase() : '?'}
          </span>
          <span className="strip-who">
            <span className="strip-name">{me ?? t.common.you}</span>
            <span className="strip-meta">
              {you === 'w' ? t.common.white : t.common.black}
              {yourTurn && ` · ${t.home.yourMove}`}
            </span>
          </span>
        </div>
      </div>
      <aside className="demo-side">
        {picker}
        <div className="demo-moves">
          <h3>{t.home.moves}</h3>
          {rows.length === 0 ? (
            <p className="text-sm text-accent">{t.home.noMovesYet}</p>
          ) : (
            <ol className="moves">
              {rows.slice(-8).map(([w, b], i) => {
                const n = Math.max(0, rows.length - 8) + i + 1
                return (
                  <li key={n}>
                    <span className="n">{n}.</span>
                    <span data-hidden={w === '?'}>{w}</span>
                    <span data-hidden={b === '?'}>{b ?? ''}</span>
                  </li>
                )
              })}
            </ol>
          )}
        </div>
        {actions}
        <p className="text-[13.5px] text-accent">{t.home.demoTip}</p>
      </aside>
    </div>
  )
}
