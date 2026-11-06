import * as RadioGroup from '@radix-ui/react-radio-group'
import { useCallback, useEffect, useState } from 'react'
import { Board } from './Board'
import { useT } from './i18n'
import { applyLocal } from './position'
import { playerId } from './player'
import { getPrefs, usePieces, usePrefs, useUnlocked } from './prefs'
import { navigate } from './router'
import { avatarImg } from './sets'
import { PromotionDialog } from './Promotion'
import { AvatarImg } from './Avatar'
import { BOT_FACE, BOT_SET, BOTS, gameLook, type Bot } from './settings'
import { botSide, useMatch, type Color, type MatchState } from './useMatch'
import { WarDemo } from './WarDemo'

/** Your view of the start under fog: your four rows, the rest hidden. Drawn until the game connects. */
const startView = (side: Color) => {
  const start = 'RNBQKBNRPPPPPPPP' + '.'.repeat(32) + 'pppppppprnbqkbnr'
  return side === 'w' ? start.slice(0, 32) + '?'.repeat(32) : '?'.repeat(32) + start.slice(32)
}

/** The home page board's opponent, always: the other bots have their own cards further down. */
export const DEMO_BOT: Bot = 'solomon'

/** Which game the home page board plays: Eat the King, or War Mode against three bots. */
export type DemoMode = 'etk' | 'war'

export function savedDemoMode(): DemoMode {
  try {
    return localStorage.getItem('demoMode') === 'war' ? 'war' : 'etk'
  } catch {
    return 'etk'
  }
}

/**
 * The home page's board: a game against Solomon that's playable the moment
 * the page loads, or in War Mode you against three copies of him, starting with your Battleship setup. The
 * server keeps it in memory only, with no clock, and never saves, lists or
 * ranks it.
 */
export function DemoBoard({ me, mode, onMode }: { me?: string; mode: DemoMode; onMode: (m: DemoMode) => void }) {
  const t = useT()
  const [side, setSide] = useState<Color>(() => getPrefs().side || 'w')
  const bot = DEMO_BOT
  const unlocked = useUnlocked()
  const [id, setId] = useState<string | null>(null)
  const [failed, setFailed] = useState(false)

  const start = useCallback((s: Color, b: Bot) => {
    setSide(s)
    setId(null)
    setFailed(false)
    fetch(`/api/matches?player=${playerId}`, {
      method: 'POST',
      body: JSON.stringify({ demo: true, side: s, bot: b }),
    })
      .then((res) => (res.ok ? res.json() : Promise.reject()))
      .then((b) => setId(b.id))
      .catch(() => setFailed(true))
  }, [])

  const [warId, setWarId] = useState<string | null>(null)
  const startWar = useCallback((b: Bot) => {
    setWarId(null)
    setFailed(false)
    fetch(`/api/four?player=${playerId}`, { method: 'POST', body: JSON.stringify({ demo: true, bot: b }) })
      .then((res) => (res.ok ? res.json() : Promise.reject()))
      .then((b) => setWarId(b.id))
      .catch(() => setFailed(true))
  }, [])

  // You start on the side you prefer (White unless you picked Black in settings).
  // War Mode starts a fresh table each time you switch to it.
  useEffect(() => {
    try {
      localStorage.setItem('demoMode', mode)
    } catch {}
    if (mode === 'war') startWar(DEMO_BOT)
    else start(getPrefs().side || 'w', DEMO_BOT)
  }, [mode, start, startWar])

  const other: Color = side === 'w' ? 'b' : 'w'
  const actions = (
    <div className="flex flex-wrap gap-2">
      <button type="button" className="demo-btn" onClick={() => start(side, bot)}>
        <span aria-hidden>↻</span> {t.home.newGame}
      </button>
      <button type="button" className="demo-btn" onClick={() => start(other, bot)}>
        {t.home.playAs(other === 'w' ? t.common.white : t.common.black)}
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
  )
  const war = mode === 'war'
  const botName = BOTS.find((b) => b.level === bot)!.name
  const skin = BOT_SET[bot]
  const picker = (
    <>
      <div className="demo-opp">
        <h3 id="demo-mode">{t.home.boardMode}</h3>
        <RadioGroup.Root
          aria-labelledby="demo-mode"
          value={mode}
          onValueChange={(m) => onMode(m as DemoMode)}
          className="demo-modes"
        >
          <RadioGroup.Item value="etk" className="demo-mode">
            {t.home.etkTitle}
          </RadioGroup.Item>
          <RadioGroup.Item value="war" className="demo-mode">
            {t.home.fourTitle}
          </RadioGroup.Item>
        </RadioGroup.Root>
      </div>
      <div className="demo-opp">
        <h3>{war ? t.home.opponents : t.home.opponent}</h3>
        <div className="demo-bot">
          <span className="demo-bot-face" aria-hidden>
            <img src={BOT_FACE[bot]} alt="" />
          </span>
          <span className="min-w-0 flex-1">
            <span className="block font-semibold">{war ? t.home.threeOf(botName) : botName}</span>
            <span className="block text-[13px] text-accent">{t.landing.botHints[bot]}</span>
          </span>
        </div>
        {!war && skin && <p className="mt-2 text-[13px] text-accent">{skinLine(skin, !!me, unlocked, t)}</p>}
      </div>
    </>
  )
  const extra = { bot, name: botName, picker }

  if (war)
    return (
      <WarDemo
        key={warId ?? 'none'}
        id={warId}
        failed={failed}
        me={me}
        bot={bot}
        name={botName}
        picker={picker}
        onNew={() => startWar(bot)}
      />
    )

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
  const img = usePieces(side)
  const { state, send } = useMatch(id)
  // Our own move shows at once; the server's next state replaces it.
  const [pending, setPending] = useState<{ uci: string; ply: number } | null>(null)
  const shown = pending && state && state.ply === pending.ply ? pending : null
  const [promotion, setPromotion] = useState<{ from: string; to: string } | null>(null)

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
  const play = (uci: string) => {
    if (state.legal.includes(uci) && send({ type: 'move', uci })) setPending({ uci, ply: state.ply })
  }
  // A pawn reaching the last rank asks what it becomes, as in a real game.
  const move = (from: string, to: string) => {
    if (state.legal.includes(from + to + 'q')) setPromotion({ from, to })
    else play(from + to)
  }
  return (
    <>
      <Frame
        {...extra}
        me={me}
        you={you}
        board={shown ? applyLocal(state.board, shown.uci) : state.board}
        legal={shown ? [] : state.legal}
        state={state}
        predicting={!!shown}
        onMove={move}
        history={state.history}
        status={over(state, you, t, extra.name, !!me)}
        thinking={!!shown || (state.status === 'playing' && state.turn !== you)}
        actions={actions}
      />
      <PromotionDialog
        open={!!promotion}
        img={(p) => img(you === 'b' ? p : p.toUpperCase(), gameLook(state.settings, botSide(state)))}
        onPick={(p) => promotion && play(promotion.from + promotion.to + p)}
        onClose={() => setPromotion(null)}
      />
    </>
  )
}

function over(state: MatchState, you: Color, t: ReturnType<typeof useT>, bot: string, signedIn: boolean) {
  if (state.status !== 'over') return null
  if (state.result === '1/2-1/2') return t.home.drawn
  const won = (state.result === '1-0') === (you === 'w')
  if (won && state.unlocked) return `${t.home.youWon} ${t.home.unlockedSkin(t.pieces.sets[state.unlocked] ?? state.unlocked)}`
  // A guest's win counts for nothing: say so, and how to make the next one count.
  if (won && !signedIn) return `${t.home.youWon} ${t.home.guestWon}`
  return won ? t.home.youWon : t.home.youLost(bot.split(' ')[0])
}

/** What beating a bot that wears a set gets you: already yours, yours to win, or nothing until you log in. */
export function skinLine(
  set: string,
  signedIn: boolean,
  unlocked: ReadonlySet<string> | null,
  t: ReturnType<typeof useT>,
) {
  const name = t.pieces.sets[set]
  if (!signedIn) return t.landing.skinGuest(name)
  return unlocked?.has(set) ? t.landing.skinWon(name) : t.landing.skinToWin(name)
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
  predicting = false,
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
  /** Our own move is drawn before the server confirms it, so the state's marks are out of date. */
  predicting?: boolean
  onMove: (from: string, to: string) => void
  history: string[]
  status: string | null
  thinking: boolean
  actions: React.ReactNode
} & Extra) {
  const t = useT()
  const avatar = usePrefs().avatar
  const yourTurn = !status && legal.length > 0
  const rows: [string, string?][] = []
  // A game where you're Black starts with White's hidden move.
  for (let i = 0; i < history.length; i += 2) rows.push([history[i], history[i + 1]])
  return (
    <div className="demo">
      <div className="demo-main">
        <div className="strip">
          <span className="strip-ava bot" aria-hidden>
            <img src={BOT_FACE[bot]} alt="" />
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
            you={you}
            legal={legal}
            lastMove={state?.last}
            check={predicting ? undefined : state?.check}
            danger={predicting ? undefined : state?.danger}
            unsafe={state?.unsafe}
            reach={predicting ? undefined : state?.reach}
            hill={state?.settings.hill}
            look={gameLook(state?.settings ?? { bot }, you === 'w' ? 'b' : 'w')}
            // The same outpost rings as a real game, from the game's own rules.
            center={state?.settings.centerVision ? { you, live: !predicting && !!state.fullVision } : undefined}
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
            {avatarImg(avatar) ? <AvatarImg code={avatar} /> : me ? me[0].toUpperCase() : '?'}
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
