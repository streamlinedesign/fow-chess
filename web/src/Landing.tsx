import { useEffect, useState } from 'react'
import { Avatar } from './Avatar'
import { useMe } from './auth'
import { Leaderboard } from './Leaderboard'
import { playerId } from './player'
import { ago, go, type GameSummary } from './ProfilePage'
import { navigate } from './router'
import { describeTime, loadSettings, NO_CLOCK, saveSettings, type Settings } from './settings'
import { SettingsPanel } from './SettingsPanel'
import { Rules } from './Rules'
import { Showcase } from './Showcase'
import { Button } from './ui'

export function Landing() {
  const me = useMe()
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
      setError("Couldn't reach the server.")
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
    setError(res ? `No match with code ${id}.` : "Couldn't reach the server.")
  }

  return (
    <main className="mx-auto flex w-full max-w-[1184px] flex-1 flex-col gap-12 px-4 pt-6 pb-16 sm:px-8 sm:pt-8 lg:gap-16">
      <section className="hero">
        <div className="hero-head">
          <h1 className="text-[clamp(34px,5vw,52px)] leading-none font-semibold tracking-tight">Play</h1>
          <p className="mt-3 max-w-[44ch] text-accent">Chess where you only see what your pieces see.</p>
        </div>

        <SettingsPanel value={settings} onChange={setSettings} signedIn={!!me} className="hero-setup" />

        <div className="hero-acts flex flex-col gap-3">
          <Button
            variant="solid"
            className="h-13 text-[17px]"
            onClick={create}
            loading={busy === 'create'}
            disabled={!!busy}
          >
            New match
          </Button>
          <form onSubmit={join} className="flex gap-2">
            <input
              value={code}
              onChange={(e) => setCode(e.target.value.toUpperCase().replace(/[^A-Z0-9]/g, ''))}
              placeholder="Have a code? Paste it here"
              aria-label="Match code"
              maxLength={12}
              autoCapitalize="characters"
              autoComplete="off"
              spellCheck={false}
              className="h-12 min-w-0 flex-1 rounded-md bg-transparent px-4 font-mono text-[17px] tracking-[0.12em] uppercase shadow-[inset_0_0_0_1.5px_var(--color-border)] outline-none placeholder:font-sans placeholder:text-[15px] placeholder:tracking-normal placeholder:normal-case placeholder:text-accent hover:shadow-[inset_0_0_0_1.5px_var(--color-fg)] focus-visible:shadow-[inset_0_0_0_2px_var(--color-fg)]"
            />
            <Button type="submit" className="h-12 px-6" loading={busy === 'join'} disabled={!!busy || !code}>
              Join
            </Button>
          </form>
          {error && (
            <p role="alert" className="text-danger">
              {error}
            </p>
          )}
        </div>

        <Showcase boardClassName="hero-board" captionClassName="hero-cap" />
      </section>

      {me && <YourGames username={me.username} />}

      <div className="max-w-[720px]">
        <Leaderboard me={me?.username} />
      </div>

      <Rules />
    </main>
  )
}

/** Your games in progress, the ones waiting on you first. Hidden when there are none. */
function YourGames({ username }: { username: string }) {
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
  const profile = `/u/${encodeURIComponent(username)}`

  return (
    <section className="flex flex-col gap-3.5">
      <div className="flex items-baseline justify-between gap-3">
        <h2 className="text-[22px] font-semibold tracking-tight">Your games</h2>
        <a href={profile} onClick={(e) => go(e, profile)} className="text-sm font-medium text-accent hover:text-fg">
          All on your profile
        </a>
      </div>
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
                  <span className="turn-tag">Your move</span>
                ) : (
                  <span>{g.waiting ? 'Waiting' : 'Their move'}</span>
                )}
                <span>{ago(g.updatedAt)}</span>
              </span>
              <span className="flex min-w-0 items-center gap-2.5">
                <Avatar size={36} />
                <span className="truncate font-semibold">
                  {g.waiting ? 'Waiting for an opponent' : g.opponent || 'Guest'}
                </span>
              </span>
              <span className="flex justify-between gap-2 text-[13px] text-accent">
                <span className="truncate">
                  {describeTime(g.settings.time ?? NO_CLOCK)} · {g.settings.kingCapture ? 'Eat the King' : 'Checkmate'}
                </span>
                <span className="shrink-0">Move {Math.floor(g.plies / 2) + 1}</span>
              </span>
            </a>
          )
        })}
      </div>
    </section>
  )
}
