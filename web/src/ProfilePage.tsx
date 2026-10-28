import * as Menu from '@radix-ui/react-dropdown-menu'
import { useEffect, useState } from 'react'
import { useMe } from './auth'
import { Avatar } from './Avatar'
import { Icon } from './icons'
import { useLive } from './live'
import { pieceImg } from './position'
import { navigate } from './router'
import { describeSettings, describeTime, NO_CLOCK, type Settings } from './settings'
import { api, openOffer, presence, type FriendState } from './social'
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

type Record = { wins: number; draws: number; losses: number; ateKing: number; kingGotAte: number }

type Profile = {
  username: string
  rating: number
  joined: number
  online: boolean
  lastSeen?: number
  /** Only on your own profile. */
  hideOnline?: boolean
  stats: Record
  rank?: number
  current: GameSummary[]
  finished: GameSummary[]
  // For a signed-in visitor.
  friend?: FriendState
  canOffer?: boolean
  canAdd?: boolean
  scoreWith?: Record
  formWith?: ('W' | 'L' | 'D')[]
}

const dateFmt = new Intl.DateTimeFormat(undefined, { day: 'numeric', month: 'short', year: 'numeric' })
const monthFmt = new Intl.DateTimeFormat(undefined, { month: 'long', year: 'numeric' })

export function ago(ms: number) {
  const s = (Date.now() - ms) / 1000
  if (s < 60) return 'just now'
  if (s < 3600) return `${Math.floor(s / 60)} min ago`
  if (s < 86400) return `${Math.floor(s / 3600)} h ago`
  if (s < 86400 * 7) return `${Math.floor(s / 86400)} d ago`
  return dateFmt.format(ms)
}

export const plural = (n: number, word: string) => `${n} ${word}${n === 1 ? '' : 's'}`

export function go(e: React.MouseEvent, href: string) {
  e.preventDefault()
  navigate(href)
}

type Filter = 'all' | 'win' | 'loss' | 'draw'

function result(g: GameSummary): Exclude<Filter, 'all'> {
  if (g.result === '1/2-1/2') return 'draw'
  return (g.result === '1-0') === (g.color === 'w') ? 'win' : 'loss'
}

export function ProfilePage({ name }: { name: string }) {
  const me = useMe()
  const [profile, setProfile] = useState<Profile | null>(null)
  const [error, setError] = useState('')
  const [filter, setFilter] = useState<Filter>('all')

  const load = () =>
    fetch(`/api/users/${encodeURIComponent(name)}`)
      .then(async (res) => {
        const body = await res.json()
        if (res.ok) setProfile(body)
        else setError(body.error || "Couldn't load the profile.")
      })
      .catch(() => setError("Couldn't reach the server."))

  useEffect(() => {
    setProfile(null)
    setError('')
    load()
  }, [name, me?.username])
  useLive((e) => {
    if (e.type === 'friends' || (e.type === 'presence' && e.user?.toLowerCase() === name.toLowerCase())) load()
  })

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
  const { stats } = profile
  const played = stats.wins + stats.draws + stats.losses
  const yourMove = (g: GameSummary) => own && !g.waiting && g.turn === g.color
  const current = [...profile.current].sort((a, b) => Number(yourMove(b)) - Number(yourMove(a)))
  const finished = profile.finished.filter((g) => filter === 'all' || result(g) === filter)
  const status = own && profile.hideOnline ? 'Hidden from others' : presence(profile)

  return (
    <main className="mx-auto flex w-full max-w-[1184px] flex-col gap-12 px-4 pt-6 pb-16 sm:px-8">
      <section className="prof-head">
        <Avatar size={112} online={profile.online} className="max-[720px]:!size-[72px]" />
        <div className="min-w-0">
          <h1 className="text-[clamp(32px,6vw,48px)] leading-none font-semibold tracking-tight break-all">
            {profile.username}
          </h1>
          <div className="mt-2 flex flex-wrap items-baseline gap-2.5">
            <span className="font-mono text-2xl tabular-nums">{profile.rating}</span>
            <span className="text-[13px] text-accent">
              rating{profile.rank ? ` · #${profile.rank} on the board` : ''}
            </span>
          </div>
          <div className="mt-2.5 flex flex-wrap gap-4 text-[13.5px] text-accent">
            {status && (
              <span className={`status ${profile.online && !(own && profile.hideOnline) ? '' : 'off'}`}>{status}</span>
            )}
            <span>Joined {monthFmt.format(profile.joined)}</span>
          </div>
        </div>
        <div className="prof-acts">
          {own ? (
            <Button onClick={() => navigate('/settings')}>
              <Icon name="gear" />
              Settings
            </Button>
          ) : me ? (
            <VisitorActions profile={profile} onChange={load} />
          ) : null}
        </div>
      </section>

      {profile.scoreWith && (
        <section className="tick score">
          <div>
            <div className="label">You vs {profile.username}</div>
            <div className="score-big mt-2">
              <b>{profile.scoreWith.wins}</b>
              <i>–</i>
              <b>{profile.scoreWith.losses}</b>
            </div>
          </div>
          <div>
            <div className="text-[13px] text-accent">
              {plural(profile.scoreWith.wins + profile.scoreWith.draws + profile.scoreWith.losses, 'game')} ·{' '}
              {profile.scoreWith.draws === 1 ? '1 draw' : `${profile.scoreWith.draws} draws`}
              {profile.formWith?.length ? ` · last ${profile.formWith.length}` : ''}
            </div>
            {profile.formWith && (
              <div className="form-dots mt-2" aria-label="Your last results, newest first">
                {profile.formWith.map((f, i) => (
                  <span key={i} className={`fd ${f === 'W' ? 'w' : f === 'D' ? 'd' : ''}`}>
                    {f}
                  </span>
                ))}
              </div>
            )}
          </div>
          <Button className="h-9 px-3.5 text-sm" onClick={() => navigate(`/messages/${profile.username}`)}>
            Your chat
          </Button>
        </section>
      )}

      <section className="stats">
        <div className="stat">
          <b>{played}</b>
          <span>Played</span>
        </div>
        <div className="stat">
          <b>{stats.wins}</b>
          <span>Wins{stats.ateKing ? ` · ${stats.ateKing} ate the king` : ''}</span>
        </div>
        <div className="stat">
          <b>{stats.draws}</b>
          <span>Draws</span>
        </div>
        <div className="stat">
          <b>{stats.losses}</b>
          <span>Losses{stats.kingGotAte ? ` · ${stats.kingGotAte} king got ate` : ''}</span>
        </div>
      </section>

      <section>
        <h2 className="mb-3.5 text-[22px] font-semibold tracking-tight">Current games</h2>
        {current.length === 0 ? (
          <p className="text-accent">No games in progress.</p>
        ) : (
          <div className="cur-list">
            {current.map((g) => {
              const href = `/m/${g.matchId}`
              return (
                <a key={`${g.matchId}-${g.round}`} href={href} onClick={(e) => go(e, href)} className="cur">
                  <span
                    className={`size-3.5 shrink-0 rounded-full shadow-[0_0_0_1.5px_var(--color-fg)] ${g.color === 'w' ? 'bg-[#fff]' : 'bg-[#111]'}`}
                    aria-label={g.color === 'w' ? 'White' : 'Black'}
                  />
                  <div className="min-w-0">
                    <div className="truncate font-semibold">
                      {g.waiting ? 'Waiting for an opponent' : `vs ${g.opponent || 'Guest'}`}
                    </div>
                    <div className="sub truncate">
                      {describeSettings(g.settings).join(' · ')} · move {Math.floor(g.plies / 2) + 1}
                    </div>
                  </div>
                  {yourMove(g) ? (
                    <span className="turn-tag">Your move</span>
                  ) : (
                    <span className="text-sm text-accent">{ago(g.updatedAt)}</span>
                  )}
                </a>
              )
            })}
          </div>
        )}
      </section>

      <section>
        <div className="mb-3.5 flex flex-wrap items-baseline justify-between gap-3">
          <h2 className="text-[22px] font-semibold tracking-tight">Previous matches</h2>
          <div className="filters" role="radiogroup" aria-label="Show">
            {(
              [
                ['all', 'All'],
                ['win', 'Wins'],
                ['loss', 'Losses'],
                ['draw', 'Draws'],
              ] as [Filter, string][]
            ).map(([f, label]) => (
              <button
                key={f}
                role="radio"
                aria-checked={filter === f}
                aria-pressed={filter === f}
                onClick={() => setFilter(f)}
              >
                {label}
              </button>
            ))}
          </div>
        </div>
        {finished.length === 0 ? (
          <p className="text-accent">{profile.finished.length ? 'Nothing here.' : 'No finished games yet.'}</p>
        ) : (
          <div className="matches">
            {finished.map((g) => (
              <MatchCard key={`${g.matchId}-${g.round}`} g={g} owner={profile.username} />
            ))}
          </div>
        )}
      </section>
    </main>
  )
}

/** Add friend, Offer a match and Message on someone else's profile. */
function VisitorActions({ profile, onChange }: { profile: Profile; onChange: () => void }) {
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const name = profile.username
  const friend = async (method: 'POST' | 'DELETE') => {
    setBusy(true)
    setError('')
    try {
      await api(method, `/api/friends/${name}`)
    } catch (e) {
      setError((e as Error).message)
    }
    setBusy(false)
    onChange()
  }

  return (
    <>
      {profile.canOffer && (
        <Button variant="solid" onClick={() => openOffer(name)}>
          <Icon name="swords" />
          Offer a match
        </Button>
      )}
      {profile.friend === 'friends' && (
        <Button onClick={() => navigate(`/messages/${name}`)}>
          <Icon name="chat" />
          Message
        </Button>
      )}
      {profile.friend === 'friends' ? (
        <Menu.Root modal={false}>
          <Menu.Trigger asChild>
            <Button variant="ghost" aria-label="Friend options">
              <Icon name="check" />
              <span className="max-[720px]:hidden">Friends</span>
            </Button>
          </Menu.Trigger>
          <Menu.Portal>
            <Menu.Content align="end" sideOffset={6} className="menu">
              <Menu.Item className="menu-item text-danger" onSelect={() => friend('DELETE')}>
                <Icon name="out" />
                Remove friend
              </Menu.Item>
            </Menu.Content>
          </Menu.Portal>
        </Menu.Root>
      ) : profile.friend === 'sent' ? (
        <Button variant="ghost" loading={busy} onClick={() => friend('DELETE')} title="Cancel the request">
          Request sent
        </Button>
      ) : profile.friend === 'received' ? (
        <Button variant="solid" loading={busy} onClick={() => friend('POST')}>
          Accept friend request
        </Button>
      ) : profile.canAdd ? (
        <Button loading={busy} onClick={() => friend('POST')}>
          <Icon name="plus" />
          Add friend
        </Button>
      ) : null}
      {error && <p className="w-full text-right text-sm text-danger">{error}</p>}
    </>
  )
}

const HOW: { [reason: string]: string } = { time: 'on time', 'capturing the king': 'by eating the king' }

function MatchCard({ g, owner }: { g: GameSummary; owner: string }) {
  const r = result(g)
  const ate = g.reason === 'capturing the king'
  const title =
    r === 'draw' ? 'Draw' : ate ? (r === 'win' ? 'Ate the king' : 'King got ate') : r === 'win' ? 'Won' : 'Lost'
  const opponent = g.opponent || 'Guest'
  const winner = r === 'draw' ? 'No winner' : `${r === 'win' ? owner : opponent} won`
  const how = HOW[g.reason ?? ''] ?? `by ${g.reason}`
  // A win shows a light king on a filled circle; a loss, your own king toppled.
  const king = r === 'win' ? 'K' : g.color === 'w' ? 'K' : 'k'
  const href = `/g/${g.matchId}/${g.round}`
  const diff = g.ratingDiff
  return (
    <a href={href} onClick={(e) => go(e, href)} className="tick mcard">
      <div className="type">
        <b>
          {g.settings.kingCapture ? 'Eat the King' : 'Checkmate'}
          {g.settings.fog ? ' · Fog' : ''}
        </b>
        <span>{describeTime(g.settings.time ?? NO_CLOCK)}</span>
      </div>
      <div className="res">
        <span className={`king ${r === 'win' ? 'won' : r === 'loss' ? 'toppled' : 'draw'}`}>
          <img src={pieceImg(king)} alt="" />
        </span>
        <div>
          <h3>{title}</h3>
          <p className="how-won">
            {winner} {r === 'draw' ? `by ${g.reason}` : how}
          </p>
        </div>
      </div>
      <div className="opp">
        <span className="flex min-w-0 items-center gap-2.5">
          <Avatar size={30} />
          <span className="truncate font-semibold">{opponent}</span>
        </span>
        <div className="text-right">
          {diff != null ? (
            <span className={`delta ${diff < 0 ? 'neg' : ''}`}>
              {diff > 0 ? '+' : diff < 0 ? '−' : '±'}
              {Math.abs(diff)}
            </span>
          ) : (
            <span className="chip">Unrated</span>
          )}
          <div className="mt-0.5 text-xs text-accent">{ago(g.updatedAt)}</div>
        </div>
      </div>
    </a>
  )
}
