import { useSyncExternalStore } from 'react'

export type Me = { username: string; rating: number }

// The signed-in user: undefined while the first check runs, null for a guest.
let me: Me | null | undefined
const listeners = new Set<() => void>()

function set(next: Me | null) {
  me = next
  listeners.forEach((l) => l())
}

export async function refreshMe() {
  const res = await fetch('/api/me').catch(() => null)
  set(res?.ok ? await res.json() : null)
}

refreshMe()

export function useMe() {
  return useSyncExternalStore(
    (cb) => {
      listeners.add(cb)
      return () => listeners.delete(cb)
    },
    () => me,
  )
}

/** Signs up or logs in. Returns an error message to show, or null on success. */
export async function authenticate(kind: 'signup' | 'login', username: string, password: string) {
  const res = await fetch(`/api/${kind}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password }),
  }).catch(() => null)
  if (!res) return "Couldn't reach the server."
  const body = await res.json().catch(() => ({}))
  if (!res.ok) return (body.error as string) || 'Something went wrong.'
  set(body)
  return null
}

export async function logout() {
  await fetch('/api/logout', { method: 'POST' }).catch(() => null)
  set(null)
}
