import { useSyncExternalStore } from 'react'

const listeners = new Set<() => void>()

function subscribe(cb: () => void) {
  listeners.add(cb)
  window.addEventListener('popstate', cb)
  return () => {
    listeners.delete(cb)
    window.removeEventListener('popstate', cb)
  }
}

export function navigate(path: string) {
  history.pushState(null, '', path)
  listeners.forEach((l) => l())
}

export function usePath() {
  return useSyncExternalStore(subscribe, () => location.pathname)
}
