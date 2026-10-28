import { useEffect, useState } from 'react'
import { useMe } from './auth'
import { Avatar } from './Avatar'
import { errorText, useT, type Dict } from './i18n'
import { Icon } from './icons'
import { useLive } from './live'
import { pieceImg } from './position'
import { navigate } from './router'
import { DropMenu, MenuItem } from './Sheet'
import { describeSettings, describeTime, modeName, NO_CLOCK, type Settings } from './settings'
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
  /** Won on the winner's first move; see the rules. */
  sniper?: boolean
}

type Record = { wins: number; draws: number; losses: number; ateKing: number; kingGotAte: number }

type Profile = {
  username: string
  rating: number
  joined: number
  online: boolean
  lastSeen?: number
  /** The match they're playing right now. */
  playing?: string
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

export function ago(ms: number, t: Dict) {
  const s = (Date.now() - ms) / 1000
  if (s < 60) return t.profile.justNow
  if (s < 3600) return t.profile.minAgo(Math.floor(s / 60))
  if (s < 86400) return t.profile.hAgo(Math.floor(s / 3600))
  if (s < 86400 * 7) return t.profile.dAgo(Math.floor(s / 86400))
  return new Intl.DateTimeFormat(t.locale, { day: 'numeric', month: 'short', year: 'numeric' }).format(ms)
}

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
  const t = useT()
  const [profile, setProfile] = useState<Profile | null>(null)
  const [error, setError] = useState('')
  const [filter, setFilter] = useState<Filter>('all')

  const load = () =>
    fetch(`/api/users/${encodeURIComponent(name)}`)
      .then(async (res) => {
        const body = await res.json()
        if (res.ok) setProfile(body)
        else setError(errorText(body))
      })
      .catch(() => setError(t.common.cantReach))

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
          {t.common.backHome}
        </Button>
      </main>
    )
  }

  if (!profile) {
    return (
      <main className="grid flex-1 place-items-center">
        <Spinner label={t.profile.loading} />
      </main>
    )
  }

  const own = me?.username.toLowerCase() === profile.username.toLowerCase()
  const { stats } = profile
  const played = stats.wins + stats.draws + stats.losses
  const yourMove = (g: GameSummary) => own && !g.waiting && g.turn === g.color
  const current = [...profile.current].sort((a, b) => Number(yourMove(b)) - Number(yourMove(a)))
  const finished = profile.finished.filter((g) => filter === 'all' || result(g) === filter)
  const status = own && profile.hideOnline ? t.profile.hidden : presence(profile, t)

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
              {t.profile.ratingLabel}
              {profile.rank ? ` · ${t.profile.onBoard(profile.rank)}` : ''}
            </span>
          </div>
          <div className="mt-2.5 flex flex-wrap gap-4 text-[13.5px] text-accent">
            {status && (
              <span className={`status ${profile.online && !(own && profile.hideOnline) ? '' : 'off'}`}>{status}</span>
            )}
            <span>
              {t.profile.joined(
                new Intl.DateTimeFormat(t.locale, { month: 'long', year: 'numeric' }).format(profile.joined),
              )}
            </span>
          </div>
        </div>
        <div className="prof-acts">
          {!own && profile.playing && (
            <Button onClick={() => navigate(`/m/${profile.playing}`)}>
              <Icon name="eye" />
              {t.common.watch}
            </Button>
          )}
          {own ? (
            <Button onClick={() => navigate('/settings')}>
              <Icon name="gear" />
              {t.common.settings}
            </Button>
          ) : me ? (
            <VisitorActions profile={profile} onChange={load} />
          ) : null}
        </div>
      </section>

      {profile.scoreWith && (
        <section className="tick score">
          <div>
            <div className="label">{t.profile.youVs(profile.username)}</div>
            <div className="score-big mt-2">
              <b>{profile.scoreWith.wins}</b>
              <i>–</i>
              <b>{profile.scoreWith.losses}</b>
            </div>
          </div>
          <div>
            <div className="text-[13px] text-accent">
              {t.common.games(profile.scoreWith.wins + profile.scoreWith.draws + profile.scoreWith.losses)} ·{' '}
              {t.common.draws(profile.scoreWith.draws)}
              {profile.formWith?.length ? ` · ${t.profile.lastN(profile.formWith.length)}` : ''}
            </div>
            {profile.formWith && (
              <div className="form-dots mt-2" aria-label={t.profile.formAria}>
                {profile.formWith.map((f, i) => (
                  <span key={i} className={`fd ${f === 'W' ? 'w' : f === 'D' ? 'd' : ''}`}>
                    {f}
                  </span>
                ))}
              </div>
            )}
          </div>
          <Button className="h-9 px-3.5 text-sm" onClick={() => navigate(`/messages/${profile.username}`)}>
            {t.profile.yourChat}
          </Button>
        </section>
      )}

      <section className="stats">
        <div className="stat">
          <b>{played}</b>
          <span>{t.profile.played}</span>
        </div>
        <div className="stat">
          <b>{stats.wins}</b>
          <span>
            {t.profile.wins}
            {stats.ateKing ? ` · ${t.profile.ateKing(stats.ateKing)}` : ''}
          </span>
        </div>
        <div className="stat">
          <b>{stats.draws}</b>
          <span>{t.profile.drawsLabel}</span>
        </div>
        <div className="stat">
          <b>{stats.losses}</b>
          <span>
            {t.profile.losses}
            {stats.kingGotAte ? ` · ${t.profile.kingGotAte(stats.kingGotAte)}` : ''}
          </span>
        </div>
      </section>

      <section>
        <h2 className="mb-3.5 text-[22px] font-semibold tracking-tight">{t.profile.currentGames}</h2>
        {current.length === 0 ? (
          <p className="text-accent">{t.profile.noCurrent}</p>
        ) : (
          <div className="cur-list">
            {current.map((g) => {
              const href = `/m/${g.matchId}`
              return (
                <a key={`${g.matchId}-${g.round}`} href={href} onClick={(e) => go(e, href)} className="cur">
                  <span
                    className={`size-3.5 shrink-0 rounded-full shadow-[0_0_0_1.5px_var(--color-fg)] ${g.color === 'w' ? 'bg-[#fff]' : 'bg-[#111]'}`}
                    aria-label={g.color === 'w' ? t.common.white : t.common.black}
                  />
                  <div className="min-w-0">
                    <div className="truncate font-semibold">
                      {g.waiting ? t.match.waitingOpponent : t.profile.vs(g.opponent || t.common.guest)}
                    </div>
                    <div className="sub truncate">
                      {describeSettings(g.settings, t).join(' · ')} · {t.common.move(Math.floor(g.plies / 2) + 1)}
                    </div>
                  </div>
                  {yourMove(g) ? (
                    <span className="turn-tag">{t.common.yourMove}</span>
                  ) : (
                    <span className="text-sm text-accent">{ago(g.updatedAt, t)}</span>
                  )}
                </a>
              )
            })}
          </div>
        )}
      </section>

      <section>
        <div className="mb-3.5 flex flex-wrap items-baseline justify-between gap-3">
          <h2 className="text-[22px] font-semibold tracking-tight">{t.profile.previousMatches}</h2>
          <div className="filters" role="radiogroup" aria-label={t.profile.show}>
            {(
              [
                ['all', t.profile.all],
                ['win', t.profile.wins],
                ['loss', t.profile.losses],
                ['draw', t.profile.drawsLabel],
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
          <p className="text-accent">{profile.finished.length ? t.profile.nothingHere : t.profile.noFinished}</p>
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
  const t = useT()
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
          {t.profile.offerMatch}
        </Button>
      )}
      {profile.friend === 'friends' && (
        <Button onClick={() => navigate(`/messages/${name}`)}>
          <Icon name="chat" />
          {t.common.message}
        </Button>
      )}
      {profile.friend === 'friends' ? (
        <DropMenu
          title={t.profile.friendOptions}
          trigger={
            <Button variant="ghost" aria-label={t.profile.friendOptions}>
              <Icon name="check" />
              <span className="max-[720px]:hidden">{t.common.friends}</span>
            </Button>
          }
        >
          <MenuItem className="text-danger" onSelect={() => friend('DELETE')}>
            <Icon name="out" />
            {t.profile.removeFriend}
          </MenuItem>
        </DropMenu>
      ) : profile.friend === 'sent' ? (
        <Button variant="ghost" loading={busy} onClick={() => friend('DELETE')} title={t.profile.cancelRequest}>
          {t.profile.requestSent}
        </Button>
      ) : profile.friend === 'received' ? (
        <Button variant="solid" loading={busy} onClick={() => friend('POST')}>
          {t.profile.acceptFriendRequest}
        </Button>
      ) : profile.canAdd ? (
        <Button loading={busy} onClick={() => friend('POST')}>
          <Icon name="plus" />
          {t.profile.addFriend}
        </Button>
      ) : null}
      {error && <p className="w-full text-right text-sm text-danger">{error}</p>}
    </>
  )
}

function MatchCard({ g, owner }: { g: GameSummary; owner: string }) {
  const t = useT()
  const r = result(g)
  const ate = g.reason === 'capturing the king'
  const hill = g.reason === 'reaching the hill'
  const title =
    r === 'draw'
      ? t.profile.cardDraw
      : g.sniper
        ? r === 'win'
          ? t.match.sniperWin
          : t.match.sniped
        : ate
          ? r === 'win'
            ? t.profile.cardAteKing
            : t.profile.cardKingGotAte
          : hill
            ? r === 'win'
              ? t.profile.cardTookHill
              : t.profile.cardLostHill
            : r === 'win'
              ? t.profile.cardWon
              : t.profile.cardLost
  const opponent = g.opponent || t.common.guest
  const winner = r === 'draw' ? t.profile.noWinner : t.profile.won(r === 'win' ? owner : opponent)
  const how = ate ? t.profile.howEat : hill ? t.profile.howHill : (g.reason && t.by[g.reason]) || `by ${g.reason}`
  // A win shows a light king on a filled circle; a loss, your own king toppled.
  const king = r === 'win' ? 'K' : g.color === 'w' ? 'K' : 'k'
  const href = `/g/${g.matchId}/${g.round}`
  const diff = g.ratingDiff
  return (
    <a href={href} onClick={(e) => go(e, href)} className="tick mcard">
      <div className="type">
        <b>{modeName(g.settings, t)}</b>
        <span>{describeTime(g.settings.time ?? NO_CLOCK, t)}</span>
      </div>
      <div className="res">
        <span className={`king ${r === 'win' ? 'won' : r === 'loss' ? 'toppled' : 'draw'}`}>
          <img src={pieceImg(king)} alt="" />
        </span>
        <div>
          <h3>{title}</h3>
          <p className="how-won">
            {winner} {how}
            {g.sniper && t.profile.onFirstMove}
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
            <span className="chip">{t.common.unrated}</span>
          )}
          <div className="mt-0.5 text-xs text-accent">{ago(g.updatedAt, t)}</div>
        </div>
      </div>
    </a>
  )
}
