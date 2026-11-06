import { useEffect, useState, type ReactNode } from 'react'
import { useMe } from './auth'
import { SEAT_COLORS } from './four'
import { useT } from './i18n'
import { CodeGame, FindFour, FindMatch, GuestPitch, presets, YourGames } from './Landing'
import { pieceSrc } from './pieces'
import { navigate } from './router'
import {
  describeTime,
  loadSettings,
  sameTime,
  saveSettings,
  type Settings,
  type TimeControl,
} from './settings'
import { DropMenu, MenuItem } from './Sheet'
import { api } from './social'
import { Button } from './ui'

type Standing = { rank: number; rating: number; games: number; of: number } | null

/**
 * The desktop app's home: three ways to play side by side. Eat the King and
 * War Mode (the ranked four-player game) show your rating and place under
 * them; Modded is never ranked. Your games and a friend's code below.
 */
export function AppHome() {
  const me = useMe()
  const t = useT()
  const [settings, setSettings] = useState<Settings>(loadSettings)
  const [busy, setBusy] = useState(false)
  const [ranks, setRanks] = useState<{ one: Standing; four: Standing } | null>(null)

  useEffect(() => {
    if (!me) return setRanks(null)
    let live = true
    api<{ one: Standing; four: Standing }>('GET', '/api/ranks')
      .then((r) => live && setRanks(r))
      .catch(() => {})
    return () => {
      live = false
    }
  }, [me])

  const set = (s: Settings) => {
    saveSettings(s)
    setSettings(s)
  }

  const rank = (s: Standing | undefined) => {
    if (me === null) return <RankLine num="—" text={t.desktop.logInForRating} />
    if (!ranks) return <RankLine num={me ? String(me.rating) : ' '} text=" " />
    if (!s) return <RankLine num="—" text={t.desktop.noRankYet} />
    return <RankLine num={String(s.rating)} text={t.desktop.place(s.rank, s.of)} />
  }

  return (
    <main className="home app-home">
      <div className="app-modes">
        <Mode art={<FogArt />} title={t.desktop.original} about={t.desktop.originalAbout} rank={rank(ranks?.one)}>
          <div className="app-go">
            <div className="min-w-0 flex-1">
              <FindMatch settings={settings} signedIn={!!me} busy={busy} summary={[]} bare />
            </div>
            <ClockChip settings={settings} onChange={set} />
          </div>
        </Mode>
        <Mode art={<WarArt />} title={t.desktop.war} about={t.desktop.warAbout} rank={rank(ranks?.four)}>
          <div className="app-go">
            <div className="min-w-0 flex-1">
              <FindFour signedIn={!!me} bare />
            </div>
            <span className="app-chip" aria-label={t.desktop.clock}>
              {t.desktop.warClock}
            </span>
          </div>
        </Mode>
        <Mode
          art={<ModsArt />}
          title={t.desktop.modded}
          about={t.desktop.moddedAbout}
          rank={<RankLine num="—" text={t.desktop.neverRanked} />}
        >
          <Button className="home-play w-full" onClick={() => navigate('/mods')}>
            {t.desktop.chooseMods}
          </Button>
        </Mode>
      </div>

      <div className="app-below">
        <div className="home-cell">{me ? <YourGames username={me.username} /> : <GuestPitch />}</div>
        <div className="home-cell">
          <CodeGame settings={settings} busy={busy} setBusy={setBusy} />
        </div>
      </div>
    </main>
  )
}

function Mode({
  art,
  title,
  about,
  rank,
  children,
}: {
  art: ReactNode
  title: string
  about: string
  rank: ReactNode
  children: ReactNode
}) {
  return (
    <section className="app-mode">
      <div className="app-art" aria-hidden>
        {art}
      </div>
      <h2>{title}</h2>
      <p className="app-about">{about}</p>
      {rank}
      {children}
    </section>
  )
}

function RankLine({ num, text }: { num: string; text: string }) {
  return (
    <div className="app-rank">
      <span className="num">{num}</span>
      <span>{text}</span>
    </div>
  )
}

/** The clock Play uses, remembered like the website's picker. */
function ClockChip({ settings, onChange }: { settings: Settings; onChange: (s: Settings) => void }) {
  const t = useT()
  const clocks: TimeControl[] = [...presets(false), ...presets(true)]
  return (
    <DropMenu
      title={t.desktop.clock}
      trigger={
        <button type="button" className="app-chip app-chip-btn" aria-label={t.desktop.clock}>
          {describeTime(settings.time, t)}
          <svg width="10" height="10" viewBox="0 0 10 10" aria-hidden>
            <path d="M2 3.5 5 6.5 8 3.5" fill="none" stroke="currentColor" strokeWidth="1.5" />
          </svg>
        </button>
      }
    >
      {clocks.map((c, i) => (
        <MenuItem key={i} onSelect={() => onChange({ ...settings, time: c })}>
          <span className="w-4">{sameTime(c, settings.time) ? '✓' : ''}</span>
          {describeTime(c, t)}
        </MenuItem>
      ))}
    </DropMenu>
  )
}

/** A two-player board in the fog: your pieces after 1. e4, the rest unseen. */
function FogArt() {
  const at: Record<string, string> = { '3,4': 'bP', '4,2': 'wB', '4,4': 'wP', '5,5': 'wN' }
  'RNBQK..R'.split('').forEach((p, c) => p !== '.' && (at[`7,${c}`] = 'w' + p))
  for (let c = 0; c < 8; c++) if (c !== 4) at[`6,${c}`] = 'wP'
  const seen = (r: number, c: number) => r > 3 || (r === 3 && c >= 2 && c <= 5)
  return (
    <div className="app-fog">
      {Array.from({ length: 64 }, (_, i) => {
        const r = Math.floor(i / 8)
        const c = i % 8
        const p = at[`${r},${c}`]
        return (
          <div key={i} className={seen(r, c) ? ((r + c) % 2 ? 'd' : 'l') : 'f'}>
            {p && seen(r, c) && <img src={pieceSrc[p]} alt="" />}
          </div>
        )
      })}
    </div>
  )
}

/** The extended four-player board: four rows each, neutral corners. */
function WarArt() {
  const n = 16
  const seat = (r: number, c: number) => {
    const mid = (x: number) => x >= 4 && x < 12
    if (r >= 14 && mid(c)) return SEAT_COLORS.r
    if (c < 2 && mid(r)) return SEAT_COLORS.b
    if (r < 2 && mid(c)) return SEAT_COLORS.g
    if (c >= 14 && mid(r)) return SEAT_COLORS.v
    return ({ '9,7': SEAT_COLORS.r, '6,9': SEAT_COLORS.g, '7,5': SEAT_COLORS.b } as Record<string, string>)[`${r},${c}`]
  }
  return (
    <div className="app-war">
      {Array.from({ length: n * n }, (_, i) => {
        const r = Math.floor(i / n)
        const c = i % n
        const corner = (r < 4 || r >= 12) && (c < 4 || c >= 12)
        const color = seat(r, c)
        return (
          <div key={i} className={corner ? 'n' : (r + c) % 2 ? 'd' : 'l'}>
            {color && <i style={{ background: color }} />}
          </div>
        )
      })}
    </div>
  )
}

/** A few of the mods page's switches. */
function ModsArt() {
  const t = useT()
  const rows: [string, boolean][] = [
    [t.panel.fog, true],
    [t.legacy.checkmate, false],
    [t.legacy.hill, true],
    [t.legacy.jump, false],
  ]
  return (
    <div className="app-dials">
      {rows.map(([label, on]) => (
        <div key={label}>
          <span>{label}</span>
          <i data-on={on} />
        </div>
      ))}
    </div>
  )
}
