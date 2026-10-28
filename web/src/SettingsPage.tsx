import * as Switch from '@radix-ui/react-switch'
import { useEffect, useState } from 'react'
import { logout, refreshMe, useMe } from './auth'
import { SignInFirst } from './FriendsPage'
import { LANGS, setLang, useLang, useT } from './i18n'
import { navigate } from './router'
import { api } from './social'
import { setThemeMode, themeMode, type ThemeMode } from './theme'
import { Button, Dialog } from './ui'

type Privacy = { hideOnline: boolean; offersFrom: 'anyone' | 'friends'; requestsFrom: 'anyone' | 'nobody' }

const SECTIONS = ['look', 'account', 'privacy'] as const

/** Your look, account and privacy. Changes save as you make them. */
export function SettingsPage() {
  const me = useMe()
  const t = useT()
  const lang = useLang()
  const [privacy, setPrivacy] = useState<Privacy | null>(null)
  const [mode, setMode] = useState(themeMode)
  const [saved, setSaved] = useState('')

  useEffect(() => {
    if (me)
      api<Privacy>('GET', '/api/settings')
        .then(setPrivacy)
        .catch(() => {})
  }, [me?.username])

  if (me === null) return <SignInFirst what={t.friends.whatSettings} />
  if (!me) return null

  const savePrivacy = async (p: Privacy) => {
    setPrivacy(p)
    try {
      setPrivacy(await api<Privacy>('PUT', '/api/settings', p))
      setSaved(t.account.saved)
    } catch (e) {
      setSaved((e as Error).message)
    }
  }

  return (
    <main className="mx-auto w-full max-w-[1184px] flex-1 px-4 pt-6 pb-16 sm:px-8">
      <div className="page-head">
        <div>
          <h1>{t.common.settings}</h1>
          <p>{t.account.changesSave}</p>
        </div>
        <span className="text-[13px] text-accent" aria-live="polite">
          {saved}
        </span>
      </div>
      <div className="set-grid">
        <nav className="set-nav" aria-label={t.account.sections}>
          {SECTIONS.map((id) => (
            <a
              key={id}
              href={`#s-${id}`}
              onClick={(e) => {
                e.preventDefault()
                document.getElementById(`s-${id}`)?.scrollIntoView({ behavior: 'smooth' })
              }}
            >
              {t.account[id]}
            </a>
          ))}
        </nav>
        <div>
          <section className="set-sec" id="s-look">
            <h2>{t.account.look}</h2>
            <div className="set-row">
              <div className="row-l">
                <b>{t.header.theme}</b>
                <span>{t.account.themeHint}</span>
              </div>
              <Seg
                label={t.header.theme}
                value={mode}
                options={[
                  ['light', t.header.light],
                  ['dark', t.header.dark],
                  ['system', t.header.system],
                ]}
                onChange={(m, e) => {
                  setThemeMode(m as ThemeMode, e)
                  setMode(m as ThemeMode)
                }}
              />
            </div>
            <div className="set-row">
              <div className="row-l">
                <b>{t.header.language}</b>
                <span>{t.account.languageHint}</span>
              </div>
              <Seg
                label={t.header.language}
                value={lang}
                options={LANGS.map((l) => [l.code, l.name])}
                onChange={(l) => setLang(l as (typeof LANGS)[number]['code'])}
              />
            </div>
          </section>

          <section className="set-sec" id="s-account">
            <h2>{t.account.account}</h2>
            <div className="set-row">
              <div className="row-l">
                <b>{t.auth.username}</b>
                <span>{t.account.usernameFixed}</span>
              </div>
              <span className="font-mono text-[15px]">{me.username}</span>
            </div>
            <PasswordForm />
            <div className="set-row">
              <div className="row-l">
                <b>{t.account.logOutEverywhere}</b>
                <span>{t.account.logOutEverywhereHint}</span>
              </div>
              <Button
                className="h-9 px-3.5 text-sm"
                onClick={async () => {
                  await api('POST', '/api/logout-everywhere').catch(() => {})
                  await refreshMe()
                  navigate('/login')
                }}
              >
                {t.account.logOutEverywhere}
              </Button>
            </div>
            <DeleteAccount />
          </section>

          <section className="set-sec" id="s-privacy">
            <h2>{t.account.privacy}</h2>
            {!privacy ? null : (
              <>
                <label className="set-row cursor-pointer">
                  <div className="row-l">
                    <b>{t.account.showOnline}</b>
                    <span>{privacy.hideOnline ? t.account.hiddenHint : t.account.shownHint}</span>
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
                    <b>{t.account.whoOffers}</b>
                    <span>{t.account.whoOffersHint}</span>
                  </div>
                  <Seg
                    label={t.account.whoOffers}
                    value={privacy.offersFrom}
                    options={[
                      ['friends', t.common.friends],
                      ['anyone', t.account.anyone],
                    ]}
                    onChange={(v) => savePrivacy({ ...privacy, offersFrom: v as Privacy['offersFrom'] })}
                  />
                </div>
                <div className="set-row">
                  <div className="row-l">
                    <b>{t.account.whoRequests}</b>
                    <span>{t.account.whoRequestsHint}</span>
                  </div>
                  <Seg
                    label={t.account.whoRequests}
                    value={privacy.requestsFrom}
                    options={[
                      ['anyone', t.account.anyone],
                      ['nobody', t.account.nobody],
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
  const t = useT()
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [confirm, setConfirm] = useState('')
  const [busy, setBusy] = useState(false)
  const [note, setNote] = useState<{ ok: boolean; text: string } | null>(null)

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (next !== confirm) return setNote({ ok: false, text: t.account.newMismatch })
    setBusy(true)
    setNote(null)
    try {
      await api('POST', '/api/password', { current, new: next })
      setCurrent('')
      setNext('')
      setConfirm('')
      setNote({ ok: true, text: t.account.passwordChanged })
    } catch (err) {
      setNote({ ok: false, text: (err as Error).message })
    }
    setBusy(false)
  }

  return (
    <form className="pw" onSubmit={submit}>
      <div className="row-l">
        <b>{t.account.changePassword}</b>
        <span>{t.account.changePasswordHint}</span>
      </div>
      <label className="field">
        <span>{t.account.currentPassword}</span>
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
          <span>{t.account.newPassword}</span>
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
          <span>{t.account.confirmNewPassword}</span>
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
          {t.account.changePassword}
        </Button>
      </div>
    </form>
  )
}

function DeleteAccount() {
  const t = useT()
  const [open, setOpen] = useState(false)
  const [password, setPassword] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  return (
    <div className="set-row">
      <div className="row-l">
        <b>{t.account.deleteAccount}</b>
        <span>{t.account.deleteHint}</span>
      </div>
      <Button
        className="h-9 px-3.5 text-sm text-danger shadow-[inset_0_0_0_1.5px_var(--color-danger)] hover:bg-danger hover:text-bg"
        onClick={() => setOpen(true)}
      >
        {t.account.deleteAccount}
      </Button>
      <Dialog open={open} onOpenChange={setOpen} title={t.account.deleteTitle} description={t.account.deleteDesc}>
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
            aria-label={t.auth.password}
            placeholder={t.auth.password}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
          {error && <p className="text-sm text-danger">{error}</p>}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="ghost" onClick={() => setOpen(false)}>
              {t.account.keepIt}
            </Button>
            <Button type="submit" variant="solid" className="bg-danger" loading={busy}>
              {t.account.delete}
            </Button>
          </div>
        </form>
      </Dialog>
    </div>
  )
}
