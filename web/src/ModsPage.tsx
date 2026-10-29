import { useState } from 'react'
import { useMe } from './auth'
import { useT } from './i18n'
import { playerId } from './player'
import { navigate } from './router'
import { loadSettings, saveSettings, type Settings } from './settings'
import { SettingsBoard } from './SettingsBoard'
import { SettingsPanel } from './SettingsPanel'
import { Button } from './ui'

/** The game with every dial exposed. Mods games are never ranked. */
export function ModsPage() {
  const me = useMe()
  const t = useT()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [settings, setSettings] = useState<Settings>(() => loadSettings(true))

  const create = async () => {
    setBusy(true)
    setError('')
    try {
      saveSettings(settings)
      const body = { ...settings, mods: true, unrated: true }
      const res = await fetch(`/api/matches?player=${playerId}`, { method: 'POST', body: JSON.stringify(body) })
      if (!res.ok) throw new Error()
      const { id } = await res.json()
      navigate(`/m/${id}`)
    } catch {
      setError(t.common.cantReach)
      setBusy(false)
    }
  }

  return (
    <main className="mx-auto flex w-full max-w-[1184px] flex-1 flex-col gap-12 px-4 pt-6 pb-16 sm:px-8 sm:pt-8">
      <section className="hero">
        <div className="hero-head">
          <h1 className="text-[clamp(34px,5vw,52px)] leading-none font-semibold tracking-tight">{t.mods.title}</h1>
          <p className="mt-3 max-w-[60ch] text-accent">{t.mods.intro}</p>
        </div>

        <SettingsPanel value={settings} onChange={setSettings} signedIn={!!me} className="hero-setup" />

        <div className="hero-acts flex flex-col gap-3">
          <Button variant="solid" className="h-13 text-[17px]" onClick={create} loading={busy} disabled={busy}>
            {t.common.newMatch}
          </Button>
          {error && (
            <p role="alert" className="text-danger">
              {error}
            </p>
          )}
        </div>

        <SettingsBoard settings={settings} boardClassName="hero-board" captionClassName="hero-cap" />
      </section>
    </main>
  )
}
