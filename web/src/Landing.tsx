import { useEffect, useState } from 'react'
import { Avatar, Who } from './Avatar'
import { useT } from './i18n'
import { Icon } from './icons'
import { useMe } from './auth'
import { Leaderboard } from './Leaderboard'
import { useLive } from './live'
import { playerId } from './player'
import { ago, go, type GameSummary } from './ProfilePage'
import { navigate } from './router'
import { describeTime, loadSettings, modeName, NO_CLOCK, saveSettings, type Settings } from './settings'
import { SettingsBoard } from './SettingsBoard'
import { SettingsPanel } from './SettingsPanel'
import { api, openOffer, type UserCard } from './social'
import { Button } from './ui'
import { GameCard, type LiveGame } from './WatchPage'

/**
 * The home page is the game's hub: start or join a match, then what's going on
 * around you. The rules and the replay live on their own page.
 */
export function Landing() {
  const me = useMe()
  const t = useT()
  const [code, setCode] = useState('')
  const [busy, setBusy] = useState<'create' | 'join' | null>(null)
  const [error, setError] = useState('')
  const [settings, setSettings] = useState<Settings>(loadSettings)

  const create = async () => {
    setBusy('create')
    setError('')
    try {
      saveSettings(settings)
      const res = await fetch(`/api/matches?player=${playerId}`, { method: 'POST', body: JSON.stringify(settings) })
      if (!res.ok) throw new Error()
      const { id } = await res.json()
      navigate(`/m/${id}`)
    } catch {
      setError(t.common.cantReach)
      setBusy(null)
    }
  }

  const join = async (e: React.FormEvent) => {
    e.preventDefault()
    const id = code.trim().toUpperCase()
    if (!id) return
    setBusy('join')
    setError('')
    const res = await fetch(`/api/matches/${id}`).catch(() => null)
    if (res?.ok) return navigate(`/m/${id}`)
    setBusy(null)
    setError(res ? t.landing.noMatch(id) : t.common.cantReach)
  }

  return (
    <main className="mx-auto flex w-full max-w-[1184px] flex-1 flex-col gap-12 px-4 pt-6 pb-16 sm:px-8 sm:pt-8 lg:gap-14">
      <section className="hero">
        <div className="hero-head">
          <p className="text-[clamp(26px,4vw,40px)] leading-tight font-semibold tracking-tight">{t.landing.tagline}</p>
          <p className="mt-3 flex flex-wrap gap-x-5 gap-y-1 text-accent">
            <a
              href="/rules"
              onClick={(e) => go(e, '/rules')}
              className="font-medium text-fg underline underline-offset-4"
            >
              {t.landing.howToPlay}
            </a>
            <a
              href="/mods"
              onClick={(e) => go(e, '/mods')}
              className="font-medium text-fg underline underline-offset-4"
            >
              {t.common.mods}
            </a>
          </p>
        </div>

        <SettingsPanel value={settings} onChange={setSettings} signedIn={!!me} className="hero-setup" />

        <div className="hero-acts flex flex-col gap-3">
          <FindMatch settings={settings} signedIn={!!me} busy={!!busy} />
          <Button className="h-12 text-[16px]" onClick={create} loading={busy === 'create'} disabled={!!busy}>
            {t.common.newMatch}
          </Button>
          <form onSubmit={join} className="flex gap-2">
            <input
              value={code}
              onChange={(e) => setCode(e.target.value.toUpperCase().replace(/[^A-Z0-9]/g, ''))}
              placeholder={t.landing.codePlaceholder}
              aria-label={t.landing.codeLabel}
              maxLength={12}
              autoCapitalize="characters"
              autoComplete="off"
              spellCheck={false}
              className="h-12 min-w-0 flex-1 rounded-md bg-transparent px-4 font-mono text-[17px] tracking-[0.12em] uppercase shadow-[inset_0_0_0_1.5px_var(--color-border)] outline-none placeholder:font-sans placeholder:text-[15px] placeholder:tracking-normal placeholder:normal-case placeholder:text-accent hover:shadow-[inset_0_0_0_1.5px_var(--color-fg)] focus-visible:shadow-[inset_0_0_0_2px_var(--color-fg)]"
            />
            <Button type="submit" className="h-12 px-6" loading={busy === 'join'} disabled={!!busy || !code}>
              {t.landing.join}
            </Button>
          </form>
          {error && (
            <p role="alert" className="text-danger">
              {error}
            </p>
          )}
        </div>

        <SettingsBoard settings={settings} boardClassName="hero-board" captionClassName="hero-cap" />
      </section>

      <div className="hub">
        <div className="hub-main">
          {me && <YourGames username={me.username} />}
          <LiveNow />
        </div>
        <div className="hub-side">
          {me ? <FriendsOnline /> : <GuestPitch />}
          <Leaderboard me={me?.username} />
        </div>
      </div>
    </main>
  )
}

/** How often the page asks whether a match was found; the server drops players who stop asking. */
const QUEUE_POLL_MS = 3000

/**
 * Find match: wait in the server's queue for the player nearest your rating
 * who wants the same game. Guests can't, since they have no rating.
 */
function FindMatch({ settings, signedIn, busy }: { settings: Settings; signedIn: boolean; busy: boolean }) {
  const t = useT()
  const [since, setSince] = useState<number | null>(null)
  const [now, setNow] = useState(0)
  const [error, setError] = useState('')
  const waiting = since !== null

  const found = (matchId: string) => {
    setSince(null)
    navigate(`/m/${matchId}`)
  }

  const start = async () => {
    setError('')
    saveSettings(settings)
    try {
      const b = await api<{ matchId?: string; waiting?: boolean }>('POST', '/api/queue', settings)
      if (b.matchId) return found(b.matchId)
      setSince(Date.now())
    } catch (e) {
      setError((e as Error).message)
    }
  }

  const stop = () => {
    setSince(null)
    api('DELETE', '/api/queue').catch(() => {})
  }

  // While waiting: keep the seat in the queue, pick up the match, and count the seconds.
  useEffect(() => {
    if (!waiting) return
    let live = true
    const poll = () =>
      api<{ matchId?: string; waiting?: boolean }>('GET', '/api/queue')
        .then((b) => {
          if (!live) return
          if (b.matchId) found(b.matchId)
          else if (!b.waiting) setSince(null)
        })
        .catch(() => {})
    const timer = setInterval(poll, QUEUE_POLL_MS)
    const tick = setInterval(() => setNow(Date.now()), 1000)
    return () => {
      live = false
      clearInterval(timer)
      clearInterval(tick)
    }
  }, [waiting])
  useLive((e) => {
    if (waiting && e.type === 'match' && e.matchId) found(e.matchId)
  })
  // Leaving the page leaves the queue.
  useEffect(() => () => void (since !== null && api('DELETE', '/api/queue').catch(() => {})), [since])

  if (!signedIn)
    return (
      <div className="flex flex-col gap-1.5">
        <Button variant="solid" className="h-13 text-[17px]" disabled>
          {t.landing.findMatch}
        </Button>
        <p className="text-sm text-accent">{t.landing.findGuest}</p>
      </div>
    )
  if (waiting)
    return (
      <div className="tick flex items-center gap-3 p-3.5" role="status">
        <span className="spinner" aria-hidden />
        <span className="flex min-w-0 flex-1 flex-col">
          <span className="font-semibold">{t.landing.finding}</span>
          <span className="text-sm text-accent">
            {t.landing.findingSince(Math.max(0, Math.round(((now || since) - since) / 1000)))}
          </span>
        </span>
        <Button className="h-9 px-3.5 text-sm" onClick={stop}>
          {t.common.cancel}
        </Button>
      </div>
    )
  return (
    <div className="flex flex-col gap-1.5">
      <Button variant="solid" className="h-13 text-[17px]" onClick={start} disabled={busy}>
        {t.landing.findMatch}
      </Button>
      {error && (
        <p role="alert" className="text-sm text-danger">
          {error}
        </p>
      )}
    </div>
  )
}

function Heading({ title, href, link }: { title: string; href: string; link: string }) {
  return (
    <div className="flex items-baseline justify-between gap-3">
      <h2 className="text-[22px] font-semibold tracking-tight">{title}</h2>
      <a href={href} onClick={(e) => go(e, href)} className="text-sm font-medium text-accent hover:text-fg">
        {link}
      </a>
    </div>
  )
}

/** Your games in progress, the ones waiting on you first. Hidden when there are none. */
function YourGames({ username }: { username: string }) {
  const t = useT()
  const [games, setGames] = useState<GameSummary[]>([])

  useEffect(() => {
    let live = true
    fetch(`/api/users/${encodeURIComponent(username)}`)
      .then((res) => (res.ok ? res.json() : null))
      .then((p) => live && p && setGames(p.current ?? []))
      .catch(() => {})
    return () => {
      live = false
    }
  }, [username])

  if (games.length === 0) return null
  const yours = (g: GameSummary) => !g.waiting && g.turn === g.color
  const sorted = [...games].sort((a, b) => Number(yours(b)) - Number(yours(a)))

  return (
    <section className="flex flex-col gap-3.5">
      <Heading title={t.landing.yourGames} href={`/u/${encodeURIComponent(username)}`} link={t.landing.allOnProfile} />
      <div className="grid grid-cols-[repeat(auto-fill,minmax(250px,1fr))] gap-4">
        {sorted.slice(0, 6).map((g) => {
          const href = `/m/${g.matchId}`
          return (
            <a
              key={`${g.matchId}-${g.round}`}
              href={href}
              onClick={(e) => go(e, href)}
              className="tick flex flex-col gap-3.5 p-[18px]"
            >
              <span className="flex items-center justify-between gap-2 text-[13px] text-accent">
                {yours(g) ? (
                  <span className="turn-tag">{t.common.yourMove}</span>
                ) : (
                  <span>{g.waiting ? t.landing.waiting : t.common.theirMove}</span>
                )}
                <span>{ago(g.updatedAt, t)}</span>
              </span>
              <span className="flex min-w-0 items-center gap-2.5">
                <Avatar size={36} />
                <span className="truncate font-semibold">
                  {g.waiting ? t.match.waitingOpponent : g.opponent || t.common.guest}
                </span>
              </span>
              <span className="flex justify-between gap-2 text-[13px] text-accent">
                <span className="truncate">
                  {modeName(g.settings, t)} · {describeTime(g.settings.time ?? NO_CLOCK, t)}
                </span>
                <span className="shrink-0">{t.common.Move(Math.floor(g.plies / 2) + 1)}</span>
              </span>
            </a>
          )
        })}
      </div>
    </section>
  )
}

/** How often the live games refresh while the page is open. */
const REFRESH_MS = 15_000

/** A few games being played right now, friends' games first. */
function LiveNow() {
  const t = useT()
  const [games, setGames] = useState<LiveGame[] | null>(null)

  useEffect(() => {
    let stopped = false
    const load = () => {
      if (document.visibilityState !== 'visible') return
      api<{ games: LiveGame[] }>('GET', '/api/watch')
        .then((b) => !stopped && setGames(b.games))
        .catch(() => !stopped && setGames((g) => g ?? []))
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

  return (
    <section className="flex flex-col gap-3.5">
      <Heading title={t.landing.liveNow} href="/watch" link={t.landing.allGames} />
      {games && games.length === 0 ? (
        <p className="text-accent">{t.landing.nobodyLive}</p>
      ) : (
        <div className="wgrid">
          {(games ?? []).slice(0, 3).map((g) => (
            <GameCard key={g.id} g={g} />
          ))}
        </div>
      )}
    </section>
  )
}

/** Friends who are online right now, with a way to challenge or watch them. */
function FriendsOnline() {
  const t = useT()
  const [friends, setFriends] = useState<UserCard[] | null>(null)

  const load = () =>
    api<{ friends: UserCard[] }>('GET', '/api/friends')
      .then((b) => setFriends(b.friends.filter((f) => f.online)))
      .catch(() => setFriends((f) => f ?? []))
  useEffect(() => {
    load()
  }, [])
  // Friends coming and going show up without a reload.
  useLive((e) => {
    if (e.type === 'friends' || e.type === 'presence') load()
  })

  return (
    <section className="flex flex-col gap-3.5">
      <Heading title={t.landing.friendsOnline} href="/friends" link={t.landing.allFriends} />
      {friends && friends.length === 0 ? (
        <p className="text-accent">{t.landing.noFriendsOnline}</p>
      ) : (
        <div className="f-list">
          {(friends ?? []).slice(0, 6).map((f) => (
            <div key={f.username} className="f-row">
              <a href={`/u/${f.username}`} onClick={(e) => go(e, `/u/${f.username}`)} className="min-w-0">
                <Who name={f.username} sub={f.rating} size={40} online />
              </a>
              <div className="f-acts">
                {f.playing ? (
                  <Button
                    className="squash h-9 px-3.5 text-sm"
                    onClick={() => navigate(`/m/${f.playing}`)}
                    aria-label={t.friends.watchX(f.username)}
                  >
                    <Icon name="eye" />
                    <span className="txt">{t.common.watch}</span>
                  </Button>
                ) : (
                  <Button className="squash h-9 px-3.5 text-sm" onClick={() => openOffer(f.username)}>
                    <Icon name="swords" />
                    <span className="txt">{t.profile.offerMatch}</span>
                  </Button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </section>
  )
}

/** Why a guest might sign in. */
function GuestPitch() {
  const t = useT()
  return (
    <section className="tick flex flex-col gap-4 p-5">
      <p className="text-[15px] leading-relaxed">{t.landing.guestPitch}</p>
      <div className="flex gap-2">
        <Button variant="solid" className="h-10 px-4" onClick={() => navigate('/signup')}>
          {t.auth.createAccount}
        </Button>
        <Button className="h-10 px-4" onClick={() => navigate('/login')}>
          {t.common.logIn}
        </Button>
      </div>
    </section>
  )
}
