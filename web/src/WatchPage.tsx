import { useEffect, useState } from 'react'
import { Avatar } from './Avatar'
import { useT } from './i18n'
import { Icon } from './icons'
import { pieceImg } from './position'
import { navigate } from './router'
import { describeTime, modeName, NO_CLOCK, type Settings } from './settings'
import { api } from './social'
import { Button, Spinner } from './ui'
import type { Color } from './useMatch'

type LiveGame = {
  id: string
  settings: Settings
  players: Record<Color, { name?: string; rating?: number }>
  /** 64 chars a1..h8, as a spectator seeing it as White sees it. */
  board: string
  ply: number
  turn: Color
  spectators: number
  rated: boolean
  /** One of the players is your friend. */
  friend: boolean
}

/** How often the list refreshes while the page is open. */
const REFRESH_MS = 10_000

/** Games being played right now, friends' games first. */
export function WatchPage() {
  const t = useT()
  const [games, setGames] = useState<LiveGame[] | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    let stopped = false
    const load = () => {
      if (document.visibilityState !== 'visible') return
      api<{ games: LiveGame[] }>('GET', '/api/watch')
        .then((b) => {
          if (stopped) return
          setGames(b.games)
          setError('')
        })
        .catch((e) => !stopped && setError(e.message))
    }
    load()
    const timer = setInterval(load, REFRESH_MS)
    document.addEventListener('visibilitychange', load)
    return () => {
      stopped = true
      clearInterval(timer)
      document.removeEventListener('visibilitychange', load)
    }
  }, [])

  const friends = games?.filter((g) => g.friend) ?? []
  const others = games?.filter((g) => !g.friend) ?? []

  return (
    <main className="mx-auto w-full max-w-[1184px] flex-1 px-4 pt-6 pb-16 sm:px-8">
      <div className="page-head">
        <div>
          <h1>
            {t.common.watch}
            {games && games.length > 0 && <span className="count">{games.length}</span>}
          </h1>
          <p>{t.watch.intro}</p>
        </div>
      </div>

      {error && !games ? (
        <p className="text-danger">{error}</p>
      ) : !games ? (
        <Spinner label={t.common.loading} />
      ) : games.length === 0 ? (
        <div className="flex flex-col items-start gap-4">
          <p className="text-accent">{t.watch.nobodyPlaying}</p>
          <Button variant="solid" onClick={() => navigate('/')}>
            {t.common.newMatch}
          </Button>
        </div>
      ) : (
        <div className="flex flex-col gap-10">
          {friends.length > 0 && <GameList title={t.watch.friendsPlaying} games={friends} />}
          {others.length > 0 && <GameList title={friends.length ? t.watch.everyoneElse : ''} games={others} />}
        </div>
      )}
    </main>
  )
}

function GameList({ title, games }: { title: string; games: LiveGame[] }) {
  return (
    <section>
      {title && <h2 className="mb-3.5 text-[22px] font-semibold tracking-tight">{title}</h2>}
      <div className="wgrid">
        {games.map((g) => (
          <GameCard key={g.id} g={g} />
        ))}
      </div>
    </section>
  )
}

function GameCard({ g }: { g: LiveGame }) {
  const t = useT()
  const href = `/m/${g.id}`
  return (
    <a
      href={href}
      onClick={(e) => {
        e.preventDefault()
        navigate(href)
      }}
      className="tick wcard"
      aria-label={t.watch.watchAria(g.players.w.name || t.common.guest, g.players.b.name || t.common.guest)}
    >
      <div className="type">
        <b>{modeName(g.settings, t)}</b>
        <span>{describeTime(g.settings.time ?? NO_CLOCK, t)}</span>
      </div>
      <MiniBoard board={g.board} />
      <div className="flex flex-col gap-1.5">
        <Seat color="b" g={g} />
        <Seat color="w" g={g} />
      </div>
      <div className="meta">
        <span>{t.common.Move(Math.floor(g.ply / 2) + 1)}</span>
        {g.spectators > 0 && (
          <span className="flex items-center gap-1">
            <Icon name="eye" size={14} />
            {t.match.watching(g.spectators)}
          </span>
        )}
        <span className="ml-auto flex gap-1.5">
          {g.friend && <span className="chip">{t.watch.friend}</span>}
          {g.rated && <span className="chip solid">{t.common.rated}</span>}
        </span>
      </div>
    </a>
  )
}

function Seat({ color, g }: { color: Color; g: LiveGame }) {
  const t = useT()
  const p = g.players[color]
  return (
    <div className="flex min-w-0 items-center gap-2 text-[14.5px]">
      <Avatar size={24} />
      <span className={`truncate ${p.name ? 'font-semibold' : 'text-accent'}`}>{p.name || t.common.guest}</span>
      {p.rating ? <span className="font-mono text-[13px] text-accent tabular-nums">{p.rating}</span> : null}
      {g.turn === color && <span className="turn-tag ml-auto">{t.common.toMove}</span>}
    </div>
  )
}

/** A small board from White's side, with fog where that view has it. */
export function MiniBoard({ board }: { board: string }) {
  const squares = []
  for (let row = 7; row >= 0; row--) {
    for (let col = 0; col < 8; col++) {
      const c = board[row * 8 + col]
      squares.push(
        <span key={row * 8 + col} className="sq" data-light={(row + col) % 2 === 1} data-fog={c === '?'}>
          {c !== '.' && c !== '?' && <img className="piece" src={pieceImg(c)} alt="" draggable={false} />}
        </span>,
      )
    }
  }
  return (
    <div className="mini-board" aria-hidden>
      {squares}
    </div>
  )
}
