import { useEffect, useState } from 'react'
import { Board } from './Board'
import { Captured } from './Captured'
import { errorText, useT, type Dict } from './i18n'
import { how } from './MatchPage'
import { navigate } from './router'
import { describeSettings, type Settings } from './settings'
import { Button, Spinner } from './ui'
import type { Color } from './useMatch'

type Frame = {
  board: string
  last: { from: string; to: string } | null
  check?: string
  captured: Record<Color, string>
}

type Replay = {
  matchId: string
  round: number
  settings: Settings
  players: Record<Color, string>
  result: string
  reason: string
  history: string[]
  frames: Frame[]
  ended: number
  sniper?: boolean
}

const noMoves: string[] = []
const noop = () => {}

function resultLine(r: Replay, t: Dict) {
  if (r.result === '1/2-1/2') return t.replay.drawBy(how(r.reason, t))
  const won = t.replay.sideWon(r.result === '1-0' ? t.common.white : t.common.black, how(r.reason, t))
  return r.sniper ? t.replay.sniper(won) : won
}

/** Steps through a finished game with the whole board visible. */
export function ReplayPage({ id, round }: { id: string; round: number }) {
  const t = useT()
  const [replay, setReplay] = useState<Replay | null>(null)
  const [error, setError] = useState('')
  const [ply, setPly] = useState(0)
  const [flipped, setFlipped] = useState(false)

  useEffect(() => {
    fetch(`/api/games/${id}/${round}`)
      .then(async (res) => {
        const body = await res.json()
        if (res.ok) {
          setReplay(body)
          setPly(body.frames.length - 1)
        } else setError(errorText(body))
      })
      .catch(() => setError(t.common.cantReach))
  }, [id, round])

  // Arrow keys step through the moves.
  useEffect(() => {
    if (!replay) return
    const last = replay.frames.length - 1
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'ArrowLeft') setPly((p) => Math.max(0, p - 1))
      else if (e.key === 'ArrowRight') setPly((p) => Math.min(last, p + 1))
      else if (e.key === 'Home') setPly(0)
      else if (e.key === 'End') setPly(last)
    }
    addEventListener('keydown', onKey)
    return () => removeEventListener('keydown', onKey)
  }, [replay])

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
  if (!replay) {
    return (
      <main className="grid flex-1 place-items-center">
        <Spinner label={t.replay.loadingGame} />
      </main>
    )
  }

  const f = replay.frames[ply]
  const last = replay.frames.length - 1
  const orientation: Color = flipped ? 'b' : 'w'
  const rows = []
  for (let i = 0; i < replay.history.length; i += 2) rows.push(i)

  const bar = (c: Color) => (
    <div className="flex h-10 items-center gap-2 px-3 text-[15px]">
      <span
        className={`size-3.5 shrink-0 rounded-full shadow-[0_0_0_1.5px_var(--color-fg)] ${c === 'w' ? 'bg-[#fff]' : 'bg-[#111]'}`}
        aria-hidden
      />
      {replay.players[c] ? (
        <a
          href={`/u/${replay.players[c]}`}
          onClick={(e) => (e.preventDefault(), navigate(`/u/${replay.players[c]}`))}
          className="font-semibold hover:underline"
        >
          {replay.players[c]}
        </a>
      ) : (
        <span className="font-semibold">{t.common.guest}</span>
      )}
      <span className="text-accent">{c === 'w' ? t.common.white : t.common.black}</span>
      <Captured pieces={f.captured[c]} className="ml-auto text-[clamp(16px,calc(var(--board)/30),22px)]" />
    </div>
  )

  return (
    <main className="flex flex-1 justify-center px-4 pb-6 sm:px-8">
      <div className="flex w-full max-w-[1100px] flex-col gap-6 lg:flex-row lg:items-start lg:justify-center">
        <section className="board-wrap mx-auto flex flex-col gap-3 lg:mx-0">
          {bar(orientation === 'w' ? 'b' : 'w')}
          <Board
            board={f.board}
            orientation={orientation}
            legal={noMoves}
            lastMove={f.last}
            check={f.check}
            hill={replay.settings.hill}
            onMove={noop}
          />
          {bar(orientation)}
        </section>

        <aside className="flex w-full flex-col gap-4 lg:w-[320px] lg:pt-9">
          <div className="flex flex-col gap-1">
            <p className="text-2xl font-bold tracking-tight">{resultLine(replay, t)}</p>
            <p className="text-sm text-accent">
              {describeSettings(replay.settings, t).join(' · ')} · {t.replay.fogLifted}
            </p>
          </div>

          <div className="flex gap-2">
            <Button variant="outline" icon aria-label={t.replay.first} onClick={() => setPly(0)} disabled={ply === 0}>
              «
            </Button>
            <Button
              variant="outline"
              icon
              aria-label={t.replay.previous}
              onClick={() => setPly(ply - 1)}
              disabled={ply === 0}
            >
              ‹
            </Button>
            <Button
              variant="outline"
              icon
              aria-label={t.replay.next}
              onClick={() => setPly(ply + 1)}
              disabled={ply === last}
            >
              ›
            </Button>
            <Button
              variant="outline"
              icon
              aria-label={t.replay.last}
              onClick={() => setPly(last)}
              disabled={ply === last}
            >
              »
            </Button>
            <Button variant="ghost" onClick={() => setFlipped((v) => !v)}>
              {t.replay.flip}
            </Button>
          </div>

          <ol className="grid max-h-[40dvh] max-w-[300px] grid-cols-[2.5rem_1fr_1fr] gap-y-1 overflow-y-auto font-mono text-[15px] tabular-nums lg:max-h-[50dvh]">
            {rows.map((i) => (
              <li key={i} className="contents">
                <span className="text-accent">{i / 2 + 1}.</span>
                {[i, i + 1].map((j) =>
                  j < replay.history.length ? (
                    <button
                      key={j}
                      onClick={() => setPly(j + 1)}
                      className={`w-fit rounded px-1 text-left ${ply === j + 1 ? 'bg-fg text-bg' : 'hover:bg-fg/10'}`}
                    >
                      {replay.history[j]}
                    </button>
                  ) : (
                    <span key={j} />
                  ),
                )}
              </li>
            ))}
          </ol>
        </aside>
      </div>
    </main>
  )
}
