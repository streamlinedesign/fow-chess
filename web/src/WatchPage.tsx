import { useEffect, useState } from 'react'
import { Avatar } from './Avatar'
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
          <h1>Watch{games && games.length > 0 && <span className="count">{games.length}</span>}</h1>
          <p>Games being played right now. In fog games you see one side's view, a few moves behind the players.</p>
        </div>
      </div>

      {error && !games ? (
        <p className="text-danger">{error}</p>
      ) : !games ? (
        <Spinner label="Loading" />
      ) : games.length === 0 ? (
        <div className="flex flex-col items-start gap-4">
          <p className="text-accent">Nobody is playing right now. Start a match and someone might watch yours.</p>
          <Button variant="solid" onClick={() => navigate('/')}>
            New match
          </Button>
        </div>
      ) : (
        <div className="flex flex-col gap-10">
          {friends.length > 0 && <GameList title="Friends playing" games={friends} />}
          {others.length > 0 && <GameList title={friends.length ? 'Everyone else' : ''} games={others} />}
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
  const href = `/m/${g.id}`
  return (
    <a
      href={href}
      onClick={(e) => {
        e.preventDefault()
        navigate(href)
      }}
      className="tick wcard"
      aria-label={`Watch ${g.players.w.name || 'Guest'} against ${g.players.b.name || 'Guest'}`}
    >
      <div className="type">
        <b>{modeName(g.settings)}</b>
        <span>{describeTime(g.settings.time ?? NO_CLOCK)}</span>
      </div>
      <MiniBoard board={g.board} />
      <div className="flex flex-col gap-1.5">
        <Seat color="b" g={g} />
        <Seat color="w" g={g} />
      </div>
      <div className="meta">
        <span>Move {Math.floor(g.ply / 2) + 1}</span>
        {g.spectators > 0 && (
          <span className="flex items-center gap-1">
            <Icon name="eye" size={14} />
            {g.spectators} watching
          </span>
        )}
        <span className="ml-auto flex gap-1.5">
          {g.friend && <span className="chip">Friend</span>}
          {g.rated && <span className="chip solid">Rated</span>}
        </span>
      </div>
    </a>
  )
}

function Seat({ color, g }: { color: Color; g: LiveGame }) {
  const p = g.players[color]
  return (
    <div className="flex min-w-0 items-center gap-2 text-[14.5px]">
      <Avatar size={24} />
      <span className={`truncate ${p.name ? 'font-semibold' : 'text-accent'}`}>{p.name || 'Guest'}</span>
      {p.rating ? <span className="font-mono text-[13px] text-accent tabular-nums">{p.rating}</span> : null}
      {g.turn === color && <span className="turn-tag ml-auto">to move</span>}
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
