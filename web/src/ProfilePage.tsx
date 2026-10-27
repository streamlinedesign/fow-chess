import { useEffect, useState } from 'react'
import { logout, useMe } from './auth'
import { navigate } from './router'
import { describeSettings, type Settings } from './settings'
import { Button, Spinner } from './ui'
import type { Color } from './useMatch'

export type GameSummary = {
  matchId: string
  round: number
  color: Color
  opponent: string
  waiting: boolean
  settings: Settings
  plies: number
  turn: Color
  result?: string
  reason?: string
  updatedAt: number
  rated?: boolean
  /** The owner's rating change, once the game is rated. */
  ratingDiff?: number
}

type Profile = {
  username: string
  rating: number
  joined: number
  current: GameSummary[]
  finished: GameSummary[]
}

const dateFmt = new Intl.DateTimeFormat(undefined, { day: 'numeric', month: 'short', year: 'numeric' })

export function ago(ms: number) {
  const s = (Date.now() - ms) / 1000
  if (s < 60) return 'just now'
  if (s < 3600) return `${Math.floor(s / 60)} min ago`
  if (s < 86400) return `${Math.floor(s / 3600)} h ago`
  if (s < 86400 * 7) return `${Math.floor(s / 86400)} d ago`
  return dateFmt.format(ms)
}

function outcome(g: GameSummary) {
  if (g.result === '1/2-1/2') return 'Draw'
  const won = (g.result === '1-0') === (g.color === 'w')
  return won ? 'Won' : 'Lost'
}

export function go(e: React.MouseEvent, href: string) {
  e.preventDefault()
  navigate(href)
}

export function ProfilePage({ name }: { name: string }) {
  const me = useMe()
  const [profile, setProfile] = useState<Profile | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    let live = true
    setProfile(null)
    setError('')
    fetch(`/api/users/${encodeURIComponent(name)}`)
      .then(async (res) => {
        const body = await res.json()
        if (!live) return
        if (res.ok) setProfile(body)
        else setError(body.error || "Couldn't load the profile.")
      })
      .catch(() => live && setError("Couldn't reach the server."))
    return () => {
      live = false
    }
  }, [name])

  if (error) {
    return (
      <main className="flex flex-1 flex-col items-start gap-6 px-4 pt-10 sm:px-8">
        <h1 className="text-[clamp(36px,8vw,56px)] leading-none font-bold tracking-tight">{error}</h1>
        <Button variant="solid" onClick={() => navigate('/')}>
          Back home
        </Button>
      </main>
    )
  }

  if (!profile) {
    return (
      <main className="grid flex-1 place-items-center">
        <Spinner label="Loading profile" />
      </main>
    )
  }

  const own = me?.username.toLowerCase() === profile.username.toLowerCase()

  return (
    <main className="mx-auto flex w-full max-w-[860px] flex-col gap-10 px-4 pt-6 pb-16 sm:px-8">
      <header className="flex flex-wrap items-end justify-between gap-4">
        <div className="flex flex-col gap-2">
          <h1 className="text-[clamp(40px,9vw,64px)] leading-none font-bold tracking-tight break-all">
            {profile.username}
          </h1>
          <p className="text-accent">Joined {dateFmt.format(profile.joined)}</p>
        </div>
        <div className="flex items-center gap-3">
          <div className="flex flex-col items-end">
            <span className="font-mono text-3xl font-bold tabular-nums">{profile.rating}</span>
            <span className="text-xs font-semibold tracking-wide text-accent uppercase">Rating</span>
          </div>
          {own && (
            <Button
              variant="ghost"
              onClick={async () => {
                await logout()
                navigate('/')
              }}
            >
              Log out
            </Button>
          )}
        </div>
      </header>

      <section className="flex flex-col gap-3">
        <h2 className="text-2xl font-bold tracking-tight">Current games</h2>
        {profile.current.length === 0 ? (
          <p className="text-accent">No games in progress.</p>
        ) : (
          <ul className="flex flex-col gap-2">
            {profile.current.map((g) => {
              const href = `/m/${g.matchId}`
              const yourTurn = own && !g.waiting && g.turn === g.color
              return (
                <li key={`${g.matchId}-${g.round}`}>
                  <a href={href} onClick={(e) => go(e, href)} className="game-row" data-active={yourTurn}>
                    <ColorDot color={g.color} />
                    <span className="min-w-0 flex-1">
                      <span className="block truncate font-semibold">
                        {g.waiting ? 'Waiting for an opponent' : `vs ${g.opponent || 'Guest'}`}
                      </span>
                      <span className="block truncate text-sm text-accent">
                        {describeSettings(g.settings).join(' · ')} · move {Math.floor(g.plies / 2) + 1}
                      </span>
                    </span>
                    {yourTurn ? (
                      <span className="turn-tag">Your move</span>
                    ) : (
                      <span className="text-sm text-accent">{ago(g.updatedAt)}</span>
                    )}
                  </a>
                </li>
              )
            })}
          </ul>
        )}
      </section>

      <section className="flex flex-col gap-3">
        <h2 className="text-2xl font-bold tracking-tight">Finished games</h2>
        {profile.finished.length === 0 ? (
          <p className="text-accent">No finished games yet.</p>
        ) : (
          <ul className="flex flex-col gap-2">
            {profile.finished.map((g) => {
              const href = `/g/${g.matchId}/${g.round}`
              const result = outcome(g)
              return (
                <li key={`${g.matchId}-${g.round}`}>
                  <a href={href} onClick={(e) => go(e, href)} className="game-row">
                    <ColorDot color={g.color} />
                    <span className="min-w-0 flex-1">
                      <span className="block truncate font-semibold">vs {g.opponent || 'Guest'}</span>
                      <span className="block truncate text-sm text-accent">
                        {result} by {g.reason} · {describeSettings(g.settings).join(' · ')}
                      </span>
                    </span>
                    <span className="flex flex-col items-end text-sm text-accent">
                      {g.ratingDiff != null && (
                        <span className="font-mono font-semibold text-fg tabular-nums">
                          {g.ratingDiff > 0 ? '+' : g.ratingDiff < 0 ? '−' : '±'}
                          {Math.abs(g.ratingDiff)}
                        </span>
                      )}
                      {ago(g.updatedAt)}
                    </span>
                  </a>
                </li>
              )
            })}
          </ul>
        )}
      </section>
    </main>
  )
}

export function ColorDot({ color }: { color: Color }) {
  return (
    <span
      className={`size-3.5 shrink-0 rounded-full shadow-[0_0_0_1.5px_var(--color-fg)] ${color === 'w' ? 'bg-[#fff]' : 'bg-[#111]'}`}
      aria-label={color === 'w' ? 'White' : 'Black'}
    />
  )
}
