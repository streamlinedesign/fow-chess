import { useMemo, useState } from 'react'
import { onBoard, SIZE, startBoard } from './four'
import { FourBoard } from './FourBoard'
import { useT } from './i18n'
import { Page } from './Page'
import { playerId } from './player'
import { navigate } from './router'
import { describeTime, loadSettings, modeName, saveSettings, type Settings } from './settings'
import { SettingsBoard, setupNotes } from './SettingsBoard'
import { SettingsPanel } from './SettingsPanel'
import { Button } from './ui'

/** The game with every dial exposed, laid out like the home page: the board at the left, the dials at the right. Mods games are never ranked. */
export function ModsPage() {
  const t = useT()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [settings, setSettings] = useState<Settings>(() => loadSettings(true))
  const [four, setFour] = useState(() => {
    try {
      return localStorage.getItem('modsPlayers') === '4'
    } catch {
      return false
    }
  })
  const pickFour = (on: boolean) => {
    setFour(on)
    try {
      localStorage.setItem('modsPlayers', on ? '4' : '2')
    } catch {}
  }

  const create = async () => {
    setBusy(true)
    setError('')
    try {
      saveSettings(settings)
      if (four) {
        const { fog, time, jumpKing, backPawns, jesterPawns, hill, centerVision } = settings
        const body = {
          fog,
          time,
          jumpKing,
          backPawns,
          jesterPawns,
          hill,
          centerVision,
        }
        const res = await fetch(`/api/four?player=${playerId}`, {
          method: 'POST',
          body: JSON.stringify(body),
        })
        if (!res.ok) throw new Error()
        const { id } = await res.json()
        navigate(`/f/${id}`)
        return
      }
      const body = { ...settings, mods: true, unrated: true }
      const res = await fetch(`/api/matches?player=${playerId}`, {
        method: 'POST',
        body: JSON.stringify(body),
      })
      if (!res.ok) throw new Error()
      const { id } = await res.json()
      navigate(`/m/${id}`)
    } catch {
      setError(t.common.cantReach)
      setBusy(false)
    }
  }

  return (
    <Page>
      <div className="home-row home-hero mods-hero">
        <div className="home-cell cross">
          <div className="demo mods-board">
            <div className="demo-main">
              {four ? (
                <FourPreview settings={settings} />
              ) : (
                <SettingsBoard settings={settings} boardClassName="w-full" />
              )}
            </div>
          </div>
        </div>
        <div className="home-cell home-menu">
          <section>
            <h1 className="home-title">{t.mods.title}</h1>
            <p className="mt-2.5 text-[15px] text-accent">{t.mods.intro}</p>
          </section>
          <section>
            <CurrentSetup settings={settings} four={four} />
          </section>
          <section>
            <SettingsPanel value={settings} onChange={setSettings} className="bare" four={four} onFour={pickFour} />
          </section>
          <section className="flex flex-col gap-2.5">
            <p className="home-summary">
              <b>{four ? t.four.four : modeName(settings, t)}</b> · {describeTime(settings.time, t)}
              <span className="block">{t.panel.modsNote}</span>
            </p>
            <Button variant="solid" className="home-play" onClick={create} loading={busy} disabled={busy}>
              {t.common.newMatch}
            </Button>
            {error && (
              <p role="alert" className="text-danger">
                {error}
              </p>
            )}
          </section>
        </div>
      </div>
    </Page>
  )
}

/** What the board beside it shows, in words: the game picked and what each setting means at the start. */
function CurrentSetup({ settings, four }: { settings: Settings; four: boolean }) {
  const t = useT()
  const notes = four ? [t.four.preview, ...(settings.fog ? [t.four.previewFog] : [])] : setupNotes(settings, t)
  return (
    <div className="setup-card">
      <h2 id="current-setup" className="text-lg font-semibold tracking-tight">
        {t.mods.currentSetup}
      </h2>
      <p className="mt-0.5 text-sm font-medium">
        {four ? t.four.four : modeName(settings, t)} · {describeTime(settings.time, t)}
      </p>
      <ul className="setup-notes">
        {notes.map((n) => (
          <li key={n}>{n}</li>
        ))}
      </ul>
    </div>
  )
}

/** The four-player board from Red's seat at the start, with the settings' fog. */
function FourPreview({ settings }: { settings: Settings }) {
  const t = useT()
  const board = useMemo(() => {
    const b = startBoard()
    if (!settings.fog) return b
    // At the start Red's pieces see their own two rows and the two ahead.
    return b.map((v, i) => {
      const x = i % SIZE
      const y = Math.floor(i / SIZE)
      return onBoard(x, y) && !(y <= 3 && x >= 3 && x <= 10) ? '?' : v
    })
  }, [settings.fog])
  return (
    <div className="four-preview" role="img" aria-label={t.four.preview}>
      <FourBoard
        board={board}
        seat="r"
        legal={[]}
        hill={settings.hill}
        center={settings.centerVision ? {} : undefined}
        look={settings}
        still
      />
    </div>
  )
}
