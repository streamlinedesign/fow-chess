import { useEffect, useState } from 'react'
import { Avatar } from './Avatar'
import { useT } from './i18n'
import { loadSettings, type Settings } from './settings'
import { SettingsPanel } from './SettingsPanel'
import { api, openOffer, showToast, useOfferTo, type UserCard } from './social'
import { Button, Dialog } from './ui'

const COLORS = ['w', 'random', 'b'] as const

/** Offer a match: pick the rules, clock and your colour, then send. Opened with openOffer(name). */
export function OfferDialog() {
  const to = useOfferTo()
  const t = useT()
  const colorNames = { w: t.common.white, random: t.messages.random, b: t.common.black }
  const [settings, setSettings] = useState<Settings>(loadSettings)
  const [color, setColor] = useState<'w' | 'b' | 'random'>('random')
  const [them, setThem] = useState<UserCard | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    setError('')
    setThem(null)
    if (!to) return
    let live = true
    fetch(`/api/users/${encodeURIComponent(to)}`)
      .then((r) => (r.ok ? r.json() : null))
      .then((p) => live && p && setThem(p))
      .catch(() => {})
    return () => {
      live = false
    }
  }, [to])

  const send = async () => {
    setBusy(true)
    setError('')
    try {
      await api('POST', '/api/offers', { to, settings, color })
      showToast({ text: t.toast.offerSent(to!) })
      openOffer(null)
    } catch (e) {
      setError((e as Error).message)
    }
    setBusy(false)
  }

  return (
    <Dialog
      open={!!to}
      onOpenChange={(open) => !open && openOffer(null)}
      title={t.offer.title}
      className="w-[min(94vw,520px)]"
    >
      <div className="flex flex-col gap-5">
        <div className="flex items-center gap-3">
          <Avatar size={40} online={them?.online} />
          <div className="flex min-w-0 flex-col leading-tight">
            <span className="truncate font-semibold">{to}</span>
            <span className="font-mono text-sm text-accent tabular-nums">{them?.rating ?? ''}</span>
          </div>
        </div>
        <SettingsPanel value={settings} onChange={setSettings} signedIn />
        <div className="flex items-center justify-between gap-3">
          <span className="font-semibold" id="offer-color">
            {t.offer.yourColour}
          </span>
          <div role="radiogroup" aria-labelledby="offer-color" className="seg">
            {COLORS.map((c) => (
              <button key={c} type="button" role="radio" aria-checked={color === c} onClick={() => setColor(c)}>
                {colorNames[c]}
              </button>
            ))}
          </div>
        </div>
        {error && (
          <p role="alert" className="text-danger">
            {error}
          </p>
        )}
        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={() => openOffer(null)}>
            {t.common.cancel}
          </Button>
          <Button variant="solid" onClick={send} loading={busy}>
            {t.offer.sendOffer}
          </Button>
        </div>
      </div>
    </Dialog>
  )
}
