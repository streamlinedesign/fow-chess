import * as RadioGroup from '@radix-ui/react-radio-group'
import { useEffect, useState } from 'react'
import { Avatar, Who } from './Avatar'
import { useMe } from './auth'
import { DemoBoard, savedDemoMode, type DemoMode } from './DemoBoard'
import { DOWNLOADS, TOURNAMENTS, releaseDate } from './features'
import { BUILDS, detectOS, downloadUrl, OS_NAME } from './desktop'
import { DownloadIcon } from './DownloadPage'
import { useT } from './i18n'
import { Icon } from './icons'
import { Leaderboard } from './Leaderboard'
import { useLive } from './live'
import { playerId } from './player'
import { getPrefs, useUnlocked } from './prefs'
import { ago, go, type GameSummary } from './ProfilePage'
import { navigate } from './router'
import {
  BOT_FACE,
  BOT_SET,
  BOTS,
  describeTime,
  loadSettings,
  modeName,
  NO_CLOCK,
  sameTime,
  saveSettings,
  TIME_GROUPS,
  timeGroup,
  timeLabel,
  type Bot,
  type Settings,
  type TimeControl,
} from './settings'
import { api, openOffer, type UserCard } from './social'
import { countdown, type Tournament } from './TournamentsPage'
import { Button } from './ui'

/** Clocks split by how long a game takes: Short is Blitz and Rapid, Long the rest. */
const SHORT_GROUPS = ['Blitz', 'Rapid']
export const presets = (long: boolean) =>
  TIME_GROUPS.filter(([g]) => SHORT_GROUPS.includes(g) !== long).flatMap(([, p]) => p)
const isLong = (t: TimeControl) => !SHORT_GROUPS.includes(timeGroup(t))

/**
 * The home page is one big board, full width, its cells split by hairlines:
 * a game you can play at once against a bot (Solomon by default) next to the
 * play menu, what the game is, then what's going on around you. Every game started here is ranked when both
 * players are signed in; Mods have their own page and are never ranked.
 */
export function Landing() {
  const me = useMe()
  const t = useT()
  const [settings, setSettings] = useState<Settings>(loadSettings)
  const [busy, setBusy] = useState(false)
  const [demo, setDemo] = useState<DemoMode>(savedDemoMode)

  return (
    <main className="home">
      <div className="home-row home-hero">
        <div className="home-cell cross">
          <DemoBoard me={me?.username} mode={demo} onMode={setDemo} />
        </div>
        <div className="home-cell home-menu">
          <section>
            <h1 className="home-title">{t.landing.tagline}</h1>
            <p className="home-release">
              {t.landing.release(releaseDate(t.locale))}
            </p>
          </section>
          <PlayMenu settings={settings} onChange={setSettings} signedIn={!!me} busy={busy} rating={me?.rating} />
          <FindFour
            signedIn={!!me}
            rating={me?.fourRating}
            onTry={() => {
              setDemo('war')
              document.querySelector('.home-hero')?.scrollIntoView({ behavior: 'smooth' })
            }}
          />
          <CodeGame settings={settings} busy={busy} setBusy={setBusy} />
          <section>
            <a href="/mods" onClick={(e) => go(e, '/mods')} className="home-mods">
              <span className="home-mods-title">{t.common.mods}</span>
              <span className="block text-[13px] text-accent">{t.home.modsSub}</span>
            </a>
          </section>
        </div>
      </div>

      <About />

      <Plans />

      <div className={`home-row home-r1 ${TOURNAMENTS ? '' : 'two'}`}>
        <div className="home-cell cross">{me ? <YourGames username={me.username} /> : <GuestPitch />}</div>
        {TOURNAMENTS && (
          <div className="home-cell cross">
            <NextArena />
          </div>
        )}
        <div className="home-cell">{me ? <FriendsOnline /> : <WatchTeaser />}</div>
      </div>

      {DOWNLOADS && <AppStrip />}

      <Bots settings={settings} />

      <div className="home-row home-r3">
        <div className="home-cell">
          <Leaderboard me={me?.username} />
        </div>
      </div>
    </main>
  )
}

/** Eat the King matchmaking: game length and clock, then one big Play. Always the usual start; layouts are mods. */
function PlayMenu({
  settings,
  onChange,
  signedIn,
  busy,
  rating,
}: {
  settings: Settings
  onChange: (s: Settings) => void
  signedIn: boolean
  busy: boolean
  rating?: number
}) {
  const t = useT()
  const long = isLong(settings.time)
  // Each length remembers the clock picked in it while the page is open.
  const [last, setLast] = useState<Record<'short' | 'long', TimeControl>>(() => ({
    short: long ? presets(false)[1] : settings.time,
    long: long ? settings.time : presets(true)[1],
  }))
  const set = (s: Settings) => {
    saveSettings(s)
    onChange(s)
  }
  const pickLength = (l: string) => {
    const toLong = l === 'long'
    if (toLong !== long) set({ ...settings, time: last[toLong ? 'long' : 'short'] })
  }
  const pickTime = (time: TimeControl) => {
    setLast({ ...last, [long ? 'long' : 'short']: time })
    set({ ...settings, time })
  }
  const clocks = presets(long)
  const summary = [describeTime(settings.time, t), t.home.etkAbout, signedIn ? t.home.ranked : t.home.guestNote]

  return (
    <section className="home-official">
      <OfficialHead title={t.home.etkTitle} rating={rating} />
      <div className="mt-4 flex flex-col gap-3.5">
        <RadioGroup.Root
          aria-label={t.home.length}
          value={long ? 'long' : 'short'}
          onValueChange={pickLength}
          className="seg seg-lg"
        >
          <RadioGroup.Item value="short">
            {t.home.short}
            <small>{t.home.shortSub}</small>
          </RadioGroup.Item>
          <RadioGroup.Item value="long">
            {t.home.long}
            <small>{t.home.longSub}</small>
          </RadioGroup.Item>
        </RadioGroup.Root>

        <RadioGroup.Root
          aria-label={t.panel.timeControl}
          value={String(clocks.findIndex((p) => sameTime(p, settings.time)))}
          onValueChange={(i) => pickTime(clocks[Number(i)])}
          className="home-times"
        >
          {clocks.map((p, i) => (
            <RadioGroup.Item key={i} value={String(i)} className="home-time">
              <b>{p.perMove ? timeLabel(p, t).split(' ').slice(0, 2).join(' ') : timeLabel(p, t)}</b>
              <span>{p.perMove ? t.time.groups[timeGroup(p)] : t.time.groups[timeGroup(p)]}</span>
            </RadioGroup.Item>
          ))}
        </RadioGroup.Root>

        <FindMatch settings={settings} signedIn={signedIn} busy={busy} summary={summary} />
      </div>
    </section>
  )
}

/** A custom game: make a code to share, or join one. Quieter than matchmaking. */
export function CodeGame({
  settings,
  busy,
  setBusy,
}: {
  settings: Settings
  busy: boolean
  setBusy: (b: boolean) => void
}) {
  const t = useT()
  const [code, setCode] = useState('')
  const [doing, setDoing] = useState<'create' | 'join' | null>(null)
  const [error, setError] = useState('')
  const begin = (what: 'create' | 'join') => {
    setDoing(what)
    setBusy(true)
    setError('')
  }
  const fail = (msg: string) => {
    setError(msg)
    setDoing(null)
    setBusy(false)
  }

  const create = async () => {
    begin('create')
    try {
      saveSettings(settings)
      const res = await fetch(`/api/matches?player=${playerId}`, {
        method: 'POST',
        body: JSON.stringify(settings),
      })
      if (!res.ok) throw new Error()
      const { id } = await res.json()
      navigate(`/m/${id}`)
    } catch {
      fail(t.common.cantReach)
    }
  }

  const join = async (e: React.FormEvent) => {
    e.preventDefault()
    const id = code.trim().toUpperCase()
    if (!id) return
    begin('join')
    const res = await fetch(`/api/matches/${id}`).catch(() => null)
    if (res?.ok) return navigate(`/m/${id}`)
    fail(res ? t.landing.noMatch(id) : t.common.cantReach)
  }

  return (
    <section className="home-code">
      <h2 className="text-lg font-semibold tracking-tight">{t.home.codeTitle}</h2>
      <p className="mt-1 text-sm text-accent">{t.home.codeSub}</p>
      <div className="home-code-acts">
        <Button className="h-11 px-4" onClick={create} loading={doing === 'create'} disabled={busy}>
          {t.home.createCode}
        </Button>
        <form onSubmit={join} className="flex min-w-0 gap-1.5">
          <input
            value={code}
            onChange={(e) => setCode(e.target.value.toUpperCase().replace(/[^A-Z0-9]/g, ''))}
            placeholder={t.landing.codePlaceholder}
            aria-label={t.landing.codeLabel}
            maxLength={12}
            autoCapitalize="characters"
            autoComplete="off"
            spellCheck={false}
            className="code-input"
          />
          <Button type="submit" className="h-11 px-4" loading={doing === 'join'} disabled={busy || !code}>
            {t.landing.join}
          </Button>
        </form>
      </div>
      {error && (
        <p role="alert" className="mt-2 text-sm text-danger">
          {error}
        </p>
      )}
    </section>
  )
}

/** How often the page asks whether a match was found; the server drops players who stop asking. */
const QUEUE_POLL_MS = 3000
/** How long Find match waits before suggesting a bot. */
const NOBODY_AFTER_S = 20

/** Starts a game against a bot with the current mode and clock, and goes there. Never ranked. */
async function playBot(settings: Settings, bot: Bot) {
  const body = { ...settings, bot, side: getPrefs().side }
  const res = await fetch(`/api/matches?player=${playerId}`, {
    method: 'POST',
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error()
  const { id } = await res.json()
  navigate(`/m/${id}`)
}

/** The computer opponents, a row of cells. They play with the mode and clock set above. */
function Bots({ settings }: { settings: Settings }) {
  const t = useT()
  const unlocked = useUnlocked()
  const [starting, setStarting] = useState<Bot | null>(null)
  const [error, setError] = useState('')
  const start = (bot: Bot) => {
    setStarting(bot)
    setError('')
    playBot(settings, bot).catch(() => {
      setError(t.common.cantReach)
      setStarting(null)
    })
  }
  return (
    <div className="home-row home-bots">
      {BOTS.map((b, i) => (
        <div key={b.level} className={`home-cell home-bot ${i < 3 ? 'cross' : ''}`}>
          <span className="home-bot-top">
            <span className="home-bot-face" aria-hidden>
              <img src={BOT_FACE[b.level]} alt="" />
            </span>
            <span className="strength" aria-label={`${i + 1}/4`}>
              {BOTS.map((_, j) => (
                <i key={j} data-on={j <= i} />
              ))}
            </span>
          </span>
          <h3>{b.name}</h3>
          <p className="text-sm text-accent">{t.landing.botHints[b.level]}</p>
          <p className="home-bot-skin text-sm">
            {(() => {
              const set = BOT_SET[b.level]
              if (!set) return t.landing.noSkin
              return unlocked?.has(set) ? t.landing.skinWon(t.pieces.sets[set]) : t.landing.skinToWin(t.pieces.sets[set])
            })()}
          </p>
          <Button
            variant={b.level === 'solomon' ? 'solid' : 'outline'}
            className="home-bot-play h-10 px-4"
            onClick={() => start(b.level)}
            loading={starting === b.level}
            disabled={starting !== null}
          >
            {t.home.playBot(b.name.split(' ')[0])}
          </Button>
          {error && starting === null && i === 0 && (
            <p role="alert" className="text-sm text-danger">
              {error}
            </p>
          )}
        </div>
      ))}
    </div>
  )
}

/**
 * Find match: wait in the server's queue for the player nearest your rating
 * who wants the same game. Guests can't, since they have no rating.
 */
export function FindMatch({
  settings,
  signedIn,
  busy,
  summary,
  bare,
}: {
  settings: Settings
  signedIn: boolean
  busy: boolean
  summary: string[]
  /** Just the button: the desktop app's home says what the game is itself. */
  bare?: boolean
}) {
  const [botBusy, setBotBusy] = useState(false)
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

  const about = bare ? null : (
    <p className="home-summary">
      <b>{summary[0]}</b> · {summary[1]}
      <span className="block">{summary[2]}</span>
    </p>
  )
  if (!signedIn)
    return (
      <div className="flex flex-col gap-2.5">
        {about}
        <Button variant="solid" className="home-play" onClick={() => navigate('/login')}>
          {t.home.logInToPlay}
        </Button>
      </div>
    )
  if (waiting) {
    const secs = Math.max(0, Math.round(((now || since) - since) / 1000))
    return (
      <div className="tick flex flex-col gap-3 p-3.5" role="status">
        <div className="flex items-center gap-3">
          <span className="spinner" aria-hidden />
          <span className="flex min-w-0 flex-1 flex-col">
            <span className="font-semibold">{t.landing.finding}</span>
            <span className="text-sm text-accent">{t.landing.findingSince(secs)}</span>
          </span>
          <Button className="h-9 px-3.5 text-sm" onClick={stop}>
            {t.common.cancel}
          </Button>
        </div>
        {secs >= NOBODY_AFTER_S && (
          <div className="flex flex-wrap items-center gap-2 text-sm">
            <span className="text-accent">{t.landing.nobodyAround}</span>
            {BOTS.slice(1, 3).map((b) => (
              <Button
                key={b.level}
                className="h-8 px-3 text-sm"
                disabled={botBusy}
                onClick={() => {
                  setBotBusy(true)
                  stop()
                  playBot(settings, b.level).catch(() => setBotBusy(false))
                }}
              >
                {b.name}
              </Button>
            ))}
          </div>
        )}
      </div>
    )
  }
  return (
    <div className="flex flex-col gap-2.5">
      {about}
      <Button variant="solid" className="home-play" onClick={start} disabled={busy}>
        {t.home.play}
      </Button>
      {error && (
        <p role="alert" className="text-sm text-danger">
          {error}
        </p>
      )}
    </div>
  )
}

/**
 * War Mode, the ranked four-player game: one fixed game (the cross,
 * Battleship, jester pawns, fog of war, taking turns), found in its own queue of four.
 */
export function FindFour({
  signedIn,
  bare,
  onTry,
  rating,
}: {
  signedIn: boolean
  bare?: boolean
  onTry?: () => void
  rating?: number
}) {
  const t = useT()
  const [since, setSince] = useState<number | null>(null)
  const [now, setNow] = useState(0)
  const [size, setSize] = useState(1)
  const [error, setError] = useState('')
  const waiting = since !== null

  const found = (matchId: string) => {
    setSince(null)
    navigate(`/f/${matchId}`)
  }
  type QueueReply = { matchId?: string; waiting?: boolean; size?: number }

  const start = async () => {
    setError('')
    try {
      const b = await api<QueueReply>('POST', '/api/four/queue')
      if (b.matchId) return found(b.matchId)
      setSize(b.size ?? 1)
      setSince(Date.now())
    } catch (e) {
      setError((e as Error).message)
    }
  }
  const stop = () => {
    setSince(null)
    api('DELETE', '/api/four/queue').catch(() => {})
  }

  useEffect(() => {
    if (!waiting) return
    let live = true
    const poll = () =>
      api<QueueReply>('GET', '/api/four/queue')
        .then((b) => {
          if (!live) return
          if (b.matchId) found(b.matchId)
          else if (!b.waiting) setSince(null)
          else setSize(b.size ?? 1)
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
    if (waiting && e.type === 'four' && e.matchId) found(e.matchId)
  })
  useEffect(() => () => void (since !== null && api('DELETE', '/api/four/queue').catch(() => {})), [since])

  return (
    <section className={`flex flex-col gap-2.5 ${bare ? '' : 'home-official'}`}>
      {!bare && (
        <>
          <OfficialHead title={t.home.fourTitle} rating={rating} />
          <p className="home-summary">
            <b>{t.home.fourAbout}</b>
            <span className="block">{signedIn ? t.home.fourRanked : t.home.fourGuest}</span>
          </p>
        </>
      )}
      {!signedIn ? (
        <Button variant="solid" className="home-play" onClick={() => navigate('/login')}>
          {t.home.logInToPlay}
        </Button>
      ) : waiting ? (
        <div className="tick flex items-center gap-3 p-3.5" role="status">
          <span className="spinner" aria-hidden />
          <span className="flex min-w-0 flex-1 flex-col">
            <span className="font-semibold">{t.home.fourFinding}</span>
            <span className="text-sm text-accent">
              {t.home.fourWaiting(
                Math.min(4, Math.max(1, size)),
                Math.max(0, Math.round(((now || since) - since) / 1000)),
              )}
            </span>
          </span>
          <Button className="h-9 px-3.5 text-sm" onClick={stop}>
            {t.common.cancel}
          </Button>
        </div>
      ) : (
        <Button variant="solid" className="home-play" onClick={start}>
          {t.home.play}
        </Button>
      )}
      {onTry && (
        <Button
          variant="ghost"
          className="self-start !px-0 underline-offset-4 hover:bg-transparent hover:underline"
          onClick={onTry}
        >
          {t.home.tryWar}
        </Button>
      )}
      {error && (
        <p role="alert" className="text-sm text-danger">
          {error}
        </p>
      )}
    </section>
  )
}

/** The title of one of the two official, ranked games: an Official tag beside
 * it, and your rating in it on the right when you're signed in. */
function OfficialHead({ title, rating }: { title: string; rating?: number }) {
  const t = useT()
  return (
    <div className="official-head">
      <h2>
        {title}
        <span className="official-tag">{t.home.official}</span>
      </h2>
      {rating !== undefined && (
        <span className="official-rating" aria-label={t.home.yourRatingIn(title, rating)}>
          <b>{rating}</b>
          <span>{t.home.yourRating}</span>
        </span>
      )}
    </div>
  )
}

function Heading({ title, href, link }: { title: string; href: string; link: string }) {
  return (
    <div className="mb-[18px] flex items-baseline justify-between gap-3">
      <h2 className="text-[22px] font-semibold tracking-tight">{title}</h2>
      <a href={href} onClick={(e) => go(e, href)} className="text-sm font-medium text-accent hover:text-fg">
        {link}
      </a>
    </div>
  )
}

/** Your games in progress, the ones waiting on you first. */
export function YourGames({ username }: { username: string }) {
  const t = useT()
  const [games, setGames] = useState<GameSummary[] | null>(null)

  useEffect(() => {
    let live = true
    fetch(`/api/users/${encodeURIComponent(username)}`)
      .then((res) => (res.ok ? res.json() : null))
      .then((p) => live && setGames(p?.current ?? []))
      .catch(() => live && setGames([]))
    return () => {
      live = false
    }
  }, [username])

  const yours = (g: GameSummary) => !g.waiting && g.turn === g.color
  const sorted = [...(games ?? [])].sort((a, b) => Number(yours(b)) - Number(yours(a)))

  return (
    <section>
      <Heading title={t.landing.yourGames} href={`/u/${encodeURIComponent(username)}`} link={t.landing.allOnProfile} />
      {games && games.length === 0 ? (
        <p className="text-accent">{t.home.noGames}</p>
      ) : (
        <div className="home-list">
          {sorted.slice(0, 4).map((g) => {
            const href = `/m/${g.matchId}`
            return (
              <a key={`${g.matchId}-${g.round}`} href={href} onClick={(e) => go(e, href)} className="home-li">
                <Avatar size={36} name={g.waiting ? undefined : g.opponent || undefined} />
                <span className="min-w-0 flex-1">
                  <span className="block truncate font-semibold">
                    {g.waiting ? t.match.waitingOpponent : g.opponent || t.common.guest}
                  </span>
                  <span className="block truncate text-[13px] text-accent">
                    {describeTime(g.settings.time ?? NO_CLOCK, t)} · {modeName(g.settings, t)} ·{' '}
                    {t.common.Move(Math.floor(g.plies / 2) + 1)}
                  </span>
                </span>
                {yours(g) ? (
                  <span className="turn-tag">{t.common.yourMove}</span>
                ) : (
                  <span className="shrink-0 text-[13px] text-accent">
                    {g.waiting ? t.landing.waiting : `${t.common.theirMove} · ${ago(g.updatedAt, t)}`}
                  </span>
                )}
              </a>
            )
          })}
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
    <section>
      <Heading title={t.landing.friendsOnline} href="/friends" link={t.landing.allFriends} />
      {friends && friends.length === 0 ? (
        <p className="text-accent">{t.landing.noFriendsOnline}</p>
      ) : (
        <div className="f-list">
          {(friends ?? []).slice(0, 4).map((f) => (
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

/** The next arena, with the way in. */
function NextArena() {
  const t = useT()
  const [list, setList] = useState<Tournament[] | null>(null)
  const [, setTick] = useState(0)
  useEffect(() => {
    let live = true
    api<{ upcoming: Tournament[] }>('GET', '/api/tournaments')
      .then((b) => live && setList(b.upcoming))
      .catch(() => live && setList([]))
    const timer = setInterval(() => setTick((n) => n + 1), 30_000)
    return () => {
      live = false
      clearInterval(timer)
    }
  }, [])
  const next = list?.find((x) => x.status === 'running') ?? list?.[0]
  const running = next?.status === 'running'
  const at = next && new Date(next.startsAt)
  return (
    <section>
      <Heading title={t.common.arena} href="/tournaments" link={t.tournaments.title} />
      {list && !next ? (
        <p className="text-accent">{t.home.noArena}</p>
      ) : next && at ? (
        <a href="/tournaments" onClick={(e) => go(e, '/tournaments')} className="home-arena">
          <span className="flex items-center gap-2 text-sm text-accent">
            {t.tournaments.kinds[next.kind]}
            {running ? <span className="turn-tag">{t.tournaments.running}</span> : null}
          </span>
          <span className="home-arena-time">{at.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</span>
          <span className="text-sm text-accent">
            {running ? t.tournaments.endsIn(countdown(next.endsAt)) : t.tournaments.startsIn(countdown(next.startsAt))}
            {' · '}
            {t.tournaments.players(next.players)}
          </span>
          <span className="home-arena-go">{next.joined ? t.tournaments.joined.split('.')[0] : t.tournaments.join}</span>
        </a>
      ) : null}
    </section>
  )
}

/** What the game is: Eat the King is the original, everything else a mod, and where the ideas came from. */
function About() {
  const t = useT()
  return (
    <section className="home-row home-about" aria-labelledby="about-title">
      <div className="home-cell cross">
        <h2 id="about-title" className="text-[22px] font-semibold tracking-tight">
          {t.about.title}
        </h2>
        <div className="home-about-text">
          {t.about.paras.map((p) => (
            <p key={p}>{p}</p>
          ))}
        </div>
        <div className="mt-5 flex flex-wrap gap-2">
          <Button className="h-10 px-4" onClick={() => navigate('/rules')}>
            {t.landing.howToPlay}
          </Button>
          <Button className="h-10 px-4" onClick={() => navigate('/mods')}>
            {t.common.mods}
          </Button>
        </div>
      </div>
      <div className="home-cell">
        <h3 className="text-[15px] font-semibold">{t.about.inspired}</h3>
        <dl className="home-inspo">
          {t.about.sources.map(([name, what]) => (
            <div key={name}>
              <dt>{name}</dt>
              <dd>{what}</dd>
            </div>
          ))}
        </dl>
      </div>
    </section>
  )
}

/** What's coming with 1.0: one short line each. */
function Plans() {
  const t = useT()
  return (
    <section className="home-row" aria-labelledby="plans-title">
      <div className="home-cell">
        <h2 id="plans-title" className="text-[22px] font-semibold tracking-tight">
          {t.about.plansHeading}
        </h2>
        <p className="mt-1 text-[13px] text-accent">{t.about.plansSub(releaseDate(t.locale))}</p>
        <ol className="plans">
          {t.about.plans.map((p, i) => (
            <li key={p}>
              <span className="plans-n">{String(i + 1).padStart(2, '0')}</span>
              {p}
            </li>
          ))}
        </ol>
      </div>
    </section>
  )
}

/** Why a guest might sign in. */
export function GuestPitch() {
  const t = useT()
  return (
    <section className="flex flex-col gap-4">
      <h2 className="text-[22px] font-semibold tracking-tight">{t.home.keepGames}</h2>
      <p className="max-w-[46ch] text-base text-accent">{t.landing.guestPitch}</p>
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

/** For guests, where the friends would be: a way to watch. */
function WatchTeaser() {
  const t = useT()
  return (
    <section>
      <Heading title={t.common.watch} href="/watch" link={t.landing.allGames} />
      <p className="text-accent">{t.watch.intro}</p>
    </section>
  )
}

/** On a computer: the desktop app, one click for this system. Hidden on phones by CSS. */
function AppStrip() {
  const t = useT()
  const os = detectOS()
  const build = BUILDS.find((b) => b.os === os)
  if (!build) return null
  return (
    <div className="home-row home-app">
      <div className="home-cell">
        <div className="flex items-baseline justify-between gap-3">
          <h2 className="text-[22px] font-semibold tracking-tight">{t.desktop.strip}</h2>
          <a
            href="/download"
            onClick={(e) => go(e, '/download')}
            className="text-sm font-medium text-accent hover:text-fg"
          >
            {t.desktop.otherSystems}
          </a>
        </div>
        <p className="mt-2 mb-5 text-accent">{t.desktop.stripAbout}</p>
        <a href={downloadUrl(build.file)} className="home-app-get">
          <DownloadIcon />
          {t.desktop.downloadFor(OS_NAME[build.os])}
        </a>
      </div>
    </div>
  )
}
