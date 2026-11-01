import { useCallback, useEffect, useState } from 'react'
import { Board } from './Board'
import { useT } from './i18n'
import { applyLocal, pieceImg } from './position'
import { playerId } from './player'
import { useMatch, type Color, type MatchState } from './useMatch'

/** Your view of the start under fog: your four rows, the rest hidden. Drawn until the game connects. */
const startView = (side: Color) => {
  const start = 'RNBQKBNRPPPPPPPP' + '.'.repeat(32) + 'pppppppprnbqkbnr'
  return side === 'w' ? start.slice(0, 32) + '?'.repeat(32) : '?'.repeat(32) + start.slice(32)
}

/**
 * The home page's board: a game against Ramesses II that's playable the moment
 * the page loads. The server keeps it in memory only, with no clock, and never
 * saves, lists or ranks it.
 */
export function DemoBoard({ me }: { me?: string }) {
  const t = useT()
  const [side, setSide] = useState<Color>('w')
  const [id, setId] = useState<string | null>(null)
  const [failed, setFailed] = useState(false)

  const start = useCallback((s: Color) => {
    setSide(s)
    setId(null)
    setFailed(false)
    fetch(`/api/matches?player=${playerId}`, {
      method: 'POST',
      body: JSON.stringify({ demo: true, side: s }),
    })
      .then((res) => (res.ok ? res.json() : Promise.reject()))
      .then((b) => setId(b.id))
      .catch(() => setFailed(true))
  }, [])

  useEffect(() => start('w'), [start])

  const other: Color = side === 'w' ? 'b' : 'w'
  const actions = (
    <div className="flex flex-wrap gap-2">
      <button type="button" className="demo-btn" onClick={() => start(side)}>
        <span aria-hidden>↻</span> {t.home.newGame}
      </button>
      <button type="button" className="demo-btn" onClick={() => start(other)}>
        {t.home.playAs(other === 'w' ? t.common.white : t.common.black)}
      </button>
    </div>
  )

  return id ? (
    <Live key={id} id={id} side={side} me={me} actions={actions} />
  ) : (
    <Frame
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

function Live({ id, side, me, actions }: { id: string; side: Color; me?: string; actions: React.ReactNode }) {
  const t = useT()
  const { state, send } = useMatch(id)
  // Our own move shows at once; the server's next state replaces it.
  const [pending, setPending] = useState<{ uci: string; ply: number } | null>(null)
  const shown = pending && state && state.ply === pending.ply ? pending : null

  if (!state)
    return (
      <Frame
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
      me={me}
      you={you}
      board={shown ? applyLocal(state.board, shown.uci) : state.board}
      legal={shown ? [] : state.legal}
      state={state}
      onMove={move}
      history={state.history}
      status={over(state, you, t)}
      thinking={!!shown || (state.status === 'playing' && state.turn !== you)}
      actions={actions}
    />
  )
}

function over(state: MatchState, you: Color, t: ReturnType<typeof useT>) {
  if (state.status !== 'over') return null
  if (state.result === '1/2-1/2') return t.home.drawn
  const won = (state.result === '1-0') === (you === 'w')
  return won ? t.home.youWon : t.home.youLost
}

function Frame({
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
}) {
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
            <img src={pieceImg('K')} alt="" />
          </span>
          <span className="strip-who">
            <span className="strip-name">Ramesses II</span>
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
        <div className="demo-how">
          <h3>{t.home.howItPlays}</h3>
          <ul>
            {t.home.howItPlaysList.map((line) => (
              <li key={line}>{line}</li>
            ))}
          </ul>
        </div>
        {actions}
        <p className="text-[13.5px] text-accent">{t.home.demoTip}</p>
      </aside>
    </div>
  )
}
