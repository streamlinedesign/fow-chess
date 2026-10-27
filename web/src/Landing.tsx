import { useState } from 'react'
import { playerId } from './player'
import { navigate } from './router'
import { loadSettings, saveSettings, type Settings } from './settings'
import { SettingsPanel } from './SettingsPanel'
import { Showcase } from './Showcase'
import { Button } from './ui'

export function Landing() {
  const [code, setCode] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [settings, setSettings] = useState<Settings>(loadSettings)

  const create = async () => {
    setBusy(true)
    setError('')
    try {
      saveSettings(settings)
      const res = await fetch(`/api/matches?player=${playerId}`, { method: 'POST', body: JSON.stringify(settings) })
      if (!res.ok) throw new Error()
      const { id } = await res.json()
      navigate(`/m/${id}`)
    } catch {
      setError("Couldn't reach the server.")
      setBusy(false)
    }
  }

  const join = async (e: React.FormEvent) => {
    e.preventDefault()
    const id = code.trim().toUpperCase()
    if (!id) return
    setBusy(true)
    setError('')
    const res = await fetch(`/api/matches/${id}`).catch(() => null)
    setBusy(false)
    if (res?.ok) navigate(`/m/${id}`)
    else setError(res ? `No match with ID ${id}.` : "Couldn't reach the server.")
  }

  return (
    <main className="flex flex-1 flex-col">
      <section className="mx-auto flex w-full max-w-[520px] flex-col gap-8 px-4 pt-4 pb-10 sm:pt-8">
        <Showcase />

        <div className="flex flex-col gap-4">
          <SettingsPanel value={settings} onChange={setSettings} />

          <Button variant="solid" className="h-14 text-lg" onClick={create} disabled={busy}>
            New match
          </Button>

          <form onSubmit={join} className="flex gap-2">
            <input
              value={code}
              onChange={(e) => setCode(e.target.value.toUpperCase().replace(/[^A-Z0-9]/g, ''))}
              placeholder="Match ID"
              aria-label="Match ID"
              maxLength={12}
              autoCapitalize="characters"
              autoComplete="off"
              spellCheck={false}
              className="h-14 min-w-0 flex-1 rounded-full bg-transparent px-6 font-mono text-lg tracking-[0.2em] uppercase shadow-[inset_0_0_0_2px_var(--color-fg)] outline-none placeholder:font-sans placeholder:tracking-normal placeholder:normal-case placeholder:text-accent focus-visible:shadow-[inset_0_0_0_3px_var(--color-fg)]"
            />
            <Button type="submit" className="h-14 text-lg" disabled={busy || !code}>
              Join
            </Button>
          </form>

          <p role="alert" className="min-h-6 text-danger">
            {error}
          </p>
        </div>
      </section>
    </main>
  )
}
