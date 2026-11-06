import { useEffect, useSyncExternalStore } from 'react'

// Titles, earned by the highest rating reached in either Eat the King or War
// Mode and kept for life. Each comes with a set (THEME_LOCK in sets.ts).
// Must match titles in server/titles.go.

/** The titles from lowest, with the rating that earns each and the tag shown beside a name ("" for none). */
export const TITLES = [
  { id: 'club', at: 1150, tag: '' },
  { id: 'expert', at: 1300, tag: 'EX' },
  { id: 'candidate', at: 1450, tag: 'CM' },
  { id: 'fog', at: 1600, tag: 'FM' },
  { id: 'im', at: 1800, tag: 'IM' },
  { id: 'gm', at: 2000, tag: 'GM' },
  { id: 'king', at: 2500, tag: 'K' },
] as const

export type TitleId = (typeof TITLES)[number]['id']

/** The title that comes with a rating mark, as in a "rating:<n>" unlock. */
export const titleAt = (rating: number) => TITLES.find((x) => x.at === rating)

/** The tag for a title id, or "" for none. */
export const tagOf = (id: string | undefined) => TITLES.find((x) => x.id === id)?.tag ?? ''

// Players' titles, looked up by name in batches and kept for the visit, the
// same way as avatars. "" means none.
const cache = new Map<string, string>()
const wanted = new Set<string>()
const listeners = new Set<() => void>()
let timer: ReturnType<typeof setTimeout> | null = null
let version = 0
const key = (name: string) => name.toLowerCase()

function emit() {
  version++
  listeners.forEach((l) => l())
}

/** Forgets a player's title, as when a rated game of theirs has just ended. */
export function forgetTitle(name: string) {
  cache.delete(key(name))
  emit()
}

function flush() {
  timer = null
  const names = [...wanted].slice(0, 100)
  names.forEach((n) => wanted.delete(n))
  if (wanted.size) timer = setTimeout(flush, 0)
  for (const n of names) if (!cache.has(n)) cache.set(n, '')
  fetch(`/api/titles?u=${encodeURIComponent(names.join(','))}`)
    .then((res) => (res.ok ? res.json() : {}))
    .then((found: Record<string, string>) => {
      for (const [n, id] of Object.entries(found)) cache.set(key(n), id)
      emit()
    })
    .catch(() => {})
}

/** A player's title id, or "" while unknown and for none. */
export function useTitle(name: string | undefined) {
  useSyncExternalStore(
    (cb) => {
      listeners.add(cb)
      return () => listeners.delete(cb)
    },
    () => version,
  )
  const k = name ? key(name) : ''
  useEffect(() => {
    if (!k || cache.has(k) || wanted.has(k)) return
    wanted.add(k)
    timer ??= setTimeout(flush, 30)
  }, [k])
  return k ? (cache.get(k) ?? '') : ''
}
