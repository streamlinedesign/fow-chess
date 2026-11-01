import { useEffect, useState } from 'react'
import { Avatar } from './Avatar'
import { useMe } from './auth'
import { rich, useT } from './i18n'
import { useLive } from './live'
import { Cell, Page, PageHead } from './Page'
import { go } from './ProfilePage'
import { navigate } from './router'
import { describeTime, type TimeControl } from './settings'
import { api } from './social'
import { Button, Spinner } from './ui'

export type Standing = {
  username: string
  rating: number
  score: number
  games: number
  wins: number
  paused?: boolean
}

export type Tournament = {
  id: number
  kind: string
  startsAt: string
  endsAt: string
  status: 'scheduled' | 'running' | 'finished' | 'cancelled'
  players: number
  time: TimeControl
  standings?: Standing[]
  joined?: boolean
  paused?: boolean
}

type Listing = {
  upcoming: Tournament[]
  recent: Tournament[]
  zone: string
  minPlayers: number
}

/** "1 h 05 min", "12 min" or "40 s" until an instant. */
export function countdown(iso: string) {
  const s = Math.max(0, Math.round((new Date(iso).getTime() - Date.now()) / 1000))
  if (s < 60) return `${s} s`
  const min = Math.floor(s / 60) % 60
  const h = Math.floor(s / 3600)
  if (h === 0) return `${min} min`
  return `${h} h ${String(min).padStart(2, '0')} min`
}

const REFRESH_MS = 5000

/** The arenas: what's coming, the one running with its table, and recent results. */
export function TournamentsPage() {
  const me = useMe()
  const t = useT()
  const [data, setData] = useState<Listing | null>(null)
  const [error, setError] = useState('')
  const [, setTick] = useState(0)

  const load = () =>
    api<Listing>('GET', '/api/tournaments')
      .then((b) => {
        setData(b)
        setError('')
      })
      .catch((e) => setError(e.message))
  useEffect(() => {
    load()
    const timer = setInterval(() => {
      if (document.visibilityState === 'visible') load()
      setTick((n) => n + 1)
    }, REFRESH_MS)
    return () => clearInterval(timer)
  }, [me?.username])
  useLive((e) => {
    if (e.type === 'notification') load()
  })

  const act = async (x: Tournament, join: boolean) => {
    try {
      await api(join ? 'POST' : 'DELETE', `/api/tournaments/${x.id}/join`)
      load()
    } catch (e) {
      setError((e as Error).message)
    }
  }

  const running = data?.upcoming.filter((x) => x.status === 'running') ?? []
  const scheduled = data?.upcoming.filter((x) => x.status === 'scheduled') ?? []

  return (
    <Page>
      <PageHead title={t.tournaments.title} sub={t.tournaments.intro} />
      {error && !data ? (
        <Cell>
          <p className="text-danger">{error}</p>
        </Cell>
      ) : !data ? (
        <Cell>
          <Spinner label={t.common.loading} />
        </Cell>
      ) : (
        <>
          {error && (
            <Cell>
              <p className="text-danger">{error}</p>
            </Cell>
          )}
          {running.map((x) => (
            <div key={x.id} className="home-row">
              <Arena x={x} me={me?.username} act={act} />
            </div>
          ))}

          <div className="home-row home-r3">
            <Cell title={t.tournaments.upcoming} className="cross">
              <div className="grid gap-4">
                {scheduled.slice(0, 4).map((x) => (
                  <article key={x.id} className="tick flex flex-col gap-3 p-5">
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <h3 className="text-lg font-semibold tracking-tight">{t.tournaments.kinds[x.kind]}</h3>
                        <p className="text-sm text-accent">
                          {describeTime(x.time, t)} ·{' '}
                          {t.tournaments.length(Math.round((+new Date(x.endsAt) - +new Date(x.startsAt)) / 60000))}
                        </p>
                      </div>
                      <span className="turn-tag whitespace-nowrap">
                        {t.tournaments.startsIn(countdown(x.startsAt))}
                      </span>
                    </div>
                    <p className="text-sm">
                      {t.tournaments.startsAt(
                        new Intl.DateTimeFormat(t.locale, {
                          weekday: 'short',
                          hour: 'numeric',
                          minute: '2-digit',
                        }).format(new Date(x.startsAt)),
                      )}
                      {' · '}
                      {t.tournaments.players(x.players)}
                      {' · '}
                      <span className="text-accent">{t.tournaments.needed(data.minPlayers)}</span>
                    </p>
                    {me ? (
                      x.joined ? (
                        <div className="flex items-center justify-between gap-3">
                          <span className="text-sm font-medium">{t.tournaments.joined}</span>
                          <Button className="h-9 px-3.5 text-sm" onClick={() => act(x, false)}>
                            {t.tournaments.leave}
                          </Button>
                        </div>
                      ) : (
                        <Button variant="solid" className="h-10 self-start px-5" onClick={() => act(x, true)}>
                          {t.tournaments.join}
                        </Button>
                      )
                    ) : (
                      <p className="text-sm">
                        <a
                          href="/login"
                          onClick={(e) => go(e, '/login')}
                          className="font-medium underline underline-offset-4"
                        >
                          {t.tournaments.joinGuest}
                        </a>
                      </p>
                    )}
                  </article>
                ))}
              </div>
            </Cell>

            <Cell title={t.tournaments.recent}>
              {data.recent.length === 0 ? (
                <p className="text-accent">{t.tournaments.noneRecent}</p>
              ) : (
                <div className="grid gap-4">
                  {data.recent.map((x) => (
                    <article key={x.id} className="tick flex flex-col gap-2 p-5">
                      <h3 className="text-lg font-semibold tracking-tight">{t.tournaments.kinds[x.kind]}</h3>
                      <p className="text-sm text-accent">
                        {new Intl.DateTimeFormat(t.locale, {
                          day: 'numeric',
                          month: 'short',
                          hour: 'numeric',
                          minute: '2-digit',
                        }).format(new Date(x.startsAt))}
                        {' · '}
                        {t.tournaments.players(x.players)}
                      </p>
                      <ol className="flex flex-col gap-1 text-[15px]">
                        {(x.standings ?? []).slice(0, 3).map((p, i) => (
                          <li key={p.username} className="flex items-center gap-2">
                            <span className="w-5 font-mono text-sm text-accent">{i + 1}</span>
                            <a
                              href={`/u/${p.username}`}
                              onClick={(e) => go(e, `/u/${p.username}`)}
                              className="font-semibold hover:underline"
                            >
                              {p.username}
                            </a>
                            <span className="ml-auto font-mono text-sm tabular-nums">{p.score}</span>
                          </li>
                        ))}
                      </ol>
                    </article>
                  ))}
                </div>
              )}
            </Cell>
          </div>
        </>
      )}

      <div className="home-row">
        <Cell title={t.tournaments.rules.heading}>
          <div className="flex max-w-[720px] flex-col gap-2.5 text-[15px] leading-relaxed text-accent [&_strong]:font-semibold [&_strong]:text-fg">
            {t.tournaments.rules.p.map((p, i) => (
              <p key={i}>{rich(p)}</p>
            ))}
          </div>
        </Cell>
      </div>
    </Page>
  )
}

/** A running arena: the clock, your state, and the table. */
function Arena({ x, me, act }: { x: Tournament; me?: string; act: (x: Tournament, join: boolean) => void }) {
  const t = useT()
  const mine = x.standings?.find((p) => p.username === me)
  return (
    <section className="home-cell flex flex-col gap-4" aria-live="polite">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 className="text-[clamp(26px,4vw,36px)] leading-none font-semibold tracking-tight">
            {t.tournaments.kinds[x.kind]}
          </h2>
          <p className="mt-2 text-sm text-accent">{t.tournaments.running}</p>
        </div>
        <span className="turn-tag">{t.tournaments.endsIn(countdown(x.endsAt))}</span>
      </div>
      {me ? (
        x.joined ? (
          <div className="tick flex flex-wrap items-center justify-between gap-3 p-4">
            <span className="flex items-center gap-3">
              {!x.paused && <span className="spinner" aria-hidden />}
              <span className="text-[15px] font-medium">
                {x.paused ? t.tournaments.sittingOut : t.tournaments.waitingPairing}
              </span>
            </span>
            {x.paused ? (
              <Button variant="solid" className="h-9 px-4 text-sm" onClick={() => act(x, true)}>
                {t.tournaments.resume}
              </Button>
            ) : (
              <Button className="h-9 px-4 text-sm" onClick={() => act(x, false)}>
                {t.tournaments.sitOut}
              </Button>
            )}
          </div>
        ) : (
          <Button variant="solid" className="h-11 self-start px-6 text-[16px]" onClick={() => act(x, true)}>
            {t.tournaments.join}
          </Button>
        )
      ) : (
        <p className="text-sm">
          <a href="/login" onClick={(e) => go(e, '/login')} className="font-medium underline underline-offset-4">
            {t.tournaments.joinGuest}
          </a>
        </p>
      )}
      <table className="lb">
        <thead>
          <tr>
            <th>{t.tournaments.place}</th>
            <th>{t.tournaments.player}</th>
            <th className="r max-sm:hidden">{t.tournaments.games}</th>
            <th className="r">{t.tournaments.score}</th>
          </tr>
        </thead>
        <tbody>
          {(x.standings ?? []).map((p, i) => (
            <tr
              key={p.username}
              data-podium={i < 3 || undefined}
              data-you={p.username === me || undefined}
              className="cursor-pointer"
              onClick={(e) => {
                if (!(e.target as Element).closest('a')) navigate(`/u/${p.username}`)
              }}
            >
              <td className="rank">{i + 1}</td>
              <td>
                <a
                  href={`/u/${p.username}`}
                  onClick={(e) => go(e, `/u/${p.username}`)}
                  className="flex min-w-0 items-center gap-2.5"
                >
                  <Avatar size={30} />
                  <span className="flex min-w-0 flex-col leading-tight">
                    <span className="truncate font-semibold">{p.username}</span>
                    <span className="text-xs text-accent">
                      {p.rating}
                      {p.paused ? ` · ${t.tournaments.sittingOut.split('.')[0]}` : ''}
                    </span>
                  </span>
                </a>
              </td>
              <td className="r max-sm:hidden">{p.games}</td>
              <td className="r rt">{p.score}</td>
            </tr>
          ))}
        </tbody>
      </table>
      {mine && !x.paused && <p className="text-sm text-accent">{t.tournaments.joined}</p>}
    </section>
  )
}
