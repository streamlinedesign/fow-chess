import { useEffect, useSyncExternalStore } from 'react'

// Other players' avatars, looked up by name in batches and kept for the visit.
// "" means the default pawn.
const cache = new Map<string, string>()
const wanted = new Set<string>()
const listeners = new Set<() => void>()
let timer: ReturnType<typeof setTimeout> | null = null

const key = (name: string) => name.toLowerCase()

function emit() {
  version++
  listeners.forEach((l) => l())
}
let version = 0

/** Sets a player's avatar in the cache, as when you change your own. */
export function setCachedAvatar(name: string, code: string) {
  cache.set(key(name), code)
  emit()
}

function flush() {
  timer = null
  const names = [...wanted].slice(0, 100)
  names.forEach((n) => wanted.delete(n))
  if (wanted.size) timer = setTimeout(flush, 0)
  // Mark them as looked up first, so a failed request doesn't repeat on every render.
  for (const n of names) if (!cache.has(n)) cache.set(n, '')
  fetch(`/api/avatars?u=${encodeURIComponent(names.join(','))}`)
    .then((res) => (res.ok ? res.json() : {}))
    .then((found: Record<string, string>) => {
      for (const [n, code] of Object.entries(found)) cache.set(key(n), code)
      emit()
    })
    .catch(() => {})
}

/** A player's avatar code, or "" while unknown and for the pawn. */
export function useAvatar(name: string | undefined) {
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
