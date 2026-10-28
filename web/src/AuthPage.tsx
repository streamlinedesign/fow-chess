import { useState } from 'react'
import { authenticate } from './auth'
import { navigate } from './router'
import { Button } from './ui'

const inputClass =
  'h-13 w-full rounded-md bg-transparent px-4 text-base shadow-[inset_0_0_0_2px_color-mix(in_srgb,var(--color-fg)_35%,transparent)] outline-none placeholder:text-accent focus-visible:shadow-[inset_0_0_0_2px_var(--color-fg)]'

function Field({ label, hint, ...props }: { label: string; hint?: string } & React.ComponentProps<'input'>) {
  return (
    <label className="flex flex-col gap-1.5">
      <span className="text-sm font-semibold">{label}</span>
      <input className={inputClass} {...props} />
      {hint && <span className="text-xs text-accent">{hint}</span>}
    </label>
  )
}

/** Log in or sign up with a username and password. No email, by design. */
export function AuthPage({ mode }: { mode: 'login' | 'signup' }) {
  const signup = mode === 'signup'
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const next = new URLSearchParams(location.search).get('next')
  const back = next?.startsWith('/') && !next.startsWith('//') ? next : '/'

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    if (signup && password !== confirm) return setError("The passwords don't match.")
    setBusy(true)
    const err = await authenticate(mode, username.trim(), password)
    setBusy(false)
    if (err) setError(err)
    else navigate(back)
  }

  const other = signup ? '/login' : '/signup'
  const otherHref = next ? `${other}?next=${encodeURIComponent(next)}` : other

  return (
    <main className="flex flex-1 justify-center px-4 pt-6 pb-16 sm:pt-16">
      <form onSubmit={submit} className="flex w-full max-w-[380px] flex-col gap-5">
        <h1 className="text-[40px] leading-none font-bold tracking-tight">{signup ? 'Create account' : 'Log in'}</h1>
        <p className="text-accent">
          {signup
            ? 'Just a username and password. Signed-in games are saved to your profile and count toward your rating.'
            : 'Welcome back.'}
        </p>

        <Field
          label="Username"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          autoComplete="username"
          autoCapitalize="none"
          spellCheck={false}
          required
          minLength={3}
          maxLength={20}
          pattern="[A-Za-z0-9_\-]+"
          hint={signup ? '3 to 20 letters, numbers, - or _.' : undefined}
          autoFocus
        />
        <Field
          label="Password"
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          autoComplete={signup ? 'new-password' : 'current-password'}
          required
          minLength={signup ? 8 : undefined}
          maxLength={72}
          hint={signup ? 'At least 8 characters.' : undefined}
        />
        {signup && (
          <Field
            label="Confirm password"
            type="password"
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
            autoComplete="new-password"
            required
          />
        )}

        <p role="alert" className="min-h-6 text-sm text-danger empty:min-h-0">
          {error}
        </p>

        <Button type="submit" variant="solid" className="h-13 text-base" loading={busy}>
          {signup ? 'Create account' : 'Log in'}
        </Button>

        <p className="text-center text-sm text-accent">
          {signup ? 'Already have an account? ' : 'New here? '}
          <a
            href={otherHref}
            onClick={(e) => {
              e.preventDefault()
              navigate(otherHref)
            }}
            className="font-semibold text-fg underline underline-offset-4"
          >
            {signup ? 'Log in' : 'Create an account'}
          </a>
        </p>
      </form>
    </main>
  )
}
