import * as Switch from '@radix-ui/react-switch'
import { useEffect, useState } from 'react'
import { logout, refreshMe, useMe } from './auth'
import { SignInFirst } from './FriendsPage'
import { navigate } from './router'
import { api } from './social'
import { setThemeMode, themeMode, type ThemeMode } from './theme'
import { Button, Dialog } from './ui'

type Privacy = { hideOnline: boolean; offersFrom: 'anyone' | 'friends'; requestsFrom: 'anyone' | 'nobody' }

const SECTIONS = [
  ['s-look', 'Look'],
  ['s-account', 'Account'],
  ['s-privacy', 'Privacy'],
]

/** Your look, account and privacy. Changes save as you make them. */
export function SettingsPage() {
  const me = useMe()
  const [privacy, setPrivacy] = useState<Privacy | null>(null)
  const [mode, setMode] = useState(themeMode)
  const [saved, setSaved] = useState('')

  useEffect(() => {
    if (me)
      api<Privacy>('GET', '/api/settings')
        .then(setPrivacy)
        .catch(() => {})
  }, [me?.username])

  if (me === null) return <SignInFirst what="change your settings" />
  if (!me) return null

  const savePrivacy = async (p: Privacy) => {
    setPrivacy(p)
    try {
      setPrivacy(await api<Privacy>('PUT', '/api/settings', p))
      setSaved('Saved')
    } catch (e) {
      setSaved((e as Error).message)
    }
  }

  return (
    <main className="mx-auto w-full max-w-[1184px] flex-1 px-4 pt-6 pb-16 sm:px-8">
      <div className="page-head">
        <div>
          <h1>Settings</h1>
          <p>Changes save as you make them.</p>
        </div>
        <span className="text-[13px] text-accent" aria-live="polite">
          {saved}
        </span>
      </div>
      <div className="set-grid">
        <nav className="set-nav" aria-label="Settings sections">
          {SECTIONS.map(([id, label]) => (
            <a
              key={id}
              href={`#${id}`}
              onClick={(e) => {
                e.preventDefault()
                document.getElementById(id)?.scrollIntoView({ behavior: 'smooth' })
              }}
            >
              {label}
            </a>
          ))}
        </nav>
        <div>
          <section className="set-sec" id="s-look">
            <h2>Look</h2>
            <div className="set-row">
              <div className="row-l">
                <b>Theme</b>
                <span>System follows your phone or computer.</span>
              </div>
              <Seg
                label="Theme"
                value={mode}
                options={[
                  ['light', 'Light'],
                  ['dark', 'Dark'],
                  ['system', 'System'],
                ]}
                onChange={(m, e) => {
                  setThemeMode(m as ThemeMode, e)
                  setMode(m as ThemeMode)
                }}
              />
            </div>
          </section>

          <section className="set-sec" id="s-account">
            <h2>Account</h2>
            <div className="set-row">
              <div className="row-l">
                <b>Username</b>
                <span>Usernames can't be changed.</span>
              </div>
              <span className="font-mono text-[15px]">{me.username}</span>
            </div>
            <PasswordForm />
            <div className="set-row">
              <div className="row-l">
                <b>Log out everywhere</b>
                <span>Signs out every phone and computer, including this one.</span>
              </div>
              <Button
                className="h-9 px-3.5 text-sm"
                onClick={async () => {
                  await api('POST', '/api/logout-everywhere').catch(() => {})
                  await refreshMe()
                  navigate('/login')
                }}
              >
                Log out everywhere
              </Button>
            </div>
            <DeleteAccount />
          </section>

          <section className="set-sec" id="s-privacy">
            <h2>Privacy</h2>
            {!privacy ? null : (
              <>
                <label className="set-row cursor-pointer">
                  <div className="row-l">
                    <b>Show when I'm online</b>
                    <span>
                      {privacy.hideOnline
                        ? "Nobody sees when you're online or when you were last here."
                        : 'Friends and visitors see the green dot and "last seen".'}
                    </span>
                  </div>
                  <Switch.Root
                    checked={!privacy.hideOnline}
                    onCheckedChange={(on) => savePrivacy({ ...privacy, hideOnline: !on })}
                    className="relative h-6 w-10 shrink-0 rounded-full bg-fg/10 shadow-[inset_0_0_0_1px_var(--color-border)] transition-colors data-[state=checked]:bg-fg"
                  >
                    <Switch.Thumb className="block size-[18px] translate-x-[3px] rounded-full bg-fg transition-transform data-[state=checked]:translate-x-[19px] data-[state=checked]:bg-bg" />
                  </Switch.Root>
                </label>
                <div className="set-row">
                  <div className="row-l">
                    <b>Who can offer me matches</b>
                    <span>Anyone can still join a game you share by code.</span>
                  </div>
                  <Seg
                    label="Who can offer me matches"
                    value={privacy.offersFrom}
                    options={[
                      ['friends', 'Friends'],
                      ['anyone', 'Anyone'],
                    ]}
                    onChange={(v) => savePrivacy({ ...privacy, offersFrom: v as Privacy['offersFrom'] })}
                  />
                </div>
                <div className="set-row">
                  <div className="row-l">
                    <b>Who can send me friend requests</b>
                    <span>Messages are always friends only.</span>
                  </div>
                  <Seg
                    label="Who can send me friend requests"
                    value={privacy.requestsFrom}
                    options={[
                      ['anyone', 'Anyone'],
                      ['nobody', 'Nobody'],
                    ]}
                    onChange={(v) => savePrivacy({ ...privacy, requestsFrom: v as Privacy['requestsFrom'] })}
                  />
                </div>
              </>
            )}
          </section>
        </div>
      </div>
    </main>
  )
}

function Seg({
  label,
  value,
  options,
  onChange,
}: {
  label: string
  value: string
  options: [string, string][]
  onChange: (v: string, e: React.MouseEvent) => void
}) {
  return (
    <div role="radiogroup" aria-label={label} className="seg">
      {options.map(([v, text]) => (
        <button key={v} type="button" role="radio" aria-checked={value === v} onClick={(e) => onChange(v, e)}>
          {text}
        </button>
      ))}
    </div>
  )
}

function PasswordForm() {
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [confirm, setConfirm] = useState('')
  const [busy, setBusy] = useState(false)
  const [note, setNote] = useState<{ ok: boolean; text: string } | null>(null)

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (next !== confirm) return setNote({ ok: false, text: "The new passwords don't match." })
    setBusy(true)
    setNote(null)
    try {
      await api('POST', '/api/password', { current, new: next })
      setCurrent('')
      setNext('')
      setConfirm('')
      setNote({ ok: true, text: 'Password changed. Your other devices were signed out.' })
    } catch (err) {
      setNote({ ok: false, text: (err as Error).message })
    }
    setBusy(false)
  }

  return (
    <form className="pw" onSubmit={submit}>
      <div className="row-l">
        <b>Change password</b>
        <span>8 to 72 characters. Your other devices get signed out.</span>
      </div>
      <label className="field">
        <span>Current password</span>
        <input
          className="input"
          type="password"
          autoComplete="current-password"
          value={current}
          onChange={(e) => setCurrent(e.target.value)}
          required
        />
      </label>
      <div className="pw-grid">
        <label className="field">
          <span>New password</span>
          <input
            className="input"
            type="password"
            autoComplete="new-password"
            minLength={8}
            maxLength={72}
            value={next}
            onChange={(e) => setNext(e.target.value)}
            required
          />
        </label>
        <label className="field">
          <span>Confirm new password</span>
          <input
            className="input"
            type="password"
            autoComplete="new-password"
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
            required
          />
        </label>
      </div>
      {note && (
        <p role="status" className={`text-sm ${note.ok ? 'text-accent' : 'text-danger'}`}>
          {note.text}
        </p>
      )}
      <div>
        <Button variant="solid" type="submit" loading={busy}>
          Change password
        </Button>
      </div>
    </form>
  )
}

function DeleteAccount() {
  const [open, setOpen] = useState(false)
  const [password, setPassword] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  return (
    <div className="set-row">
      <div className="row-l">
        <b>Delete account</b>
        <span>
          Removes your profile, rating, friends and chats. Finished games stay in your opponents' history, played by a
          guest.
        </span>
      </div>
      <Button
        className="h-9 px-3.5 text-sm text-danger shadow-[inset_0_0_0_1.5px_var(--color-danger)] hover:bg-danger hover:text-bg"
        onClick={() => setOpen(true)}
      >
        Delete account
      </Button>
      <Dialog
        open={open}
        onOpenChange={setOpen}
        title="Delete your account?"
        description="This can't be undone. Type your password to confirm."
      >
        <form
          className="flex flex-col gap-4"
          onSubmit={async (e) => {
            e.preventDefault()
            setBusy(true)
            setError('')
            try {
              await api('POST', '/api/account/delete', { password })
              await logout()
              navigate('/')
            } catch (err) {
              setError((err as Error).message)
              setBusy(false)
            }
          }}
        >
          <input
            className="input"
            type="password"
            autoComplete="current-password"
            aria-label="Password"
            placeholder="Password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
          {error && <p className="text-sm text-danger">{error}</p>}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="ghost" onClick={() => setOpen(false)}>
              Keep it
            </Button>
            <Button type="submit" variant="solid" className="bg-danger" loading={busy}>
              Delete
            </Button>
          </div>
        </form>
      </Dialog>
    </div>
  )
}
