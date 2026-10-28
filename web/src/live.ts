import { useEffect, useRef, useSyncExternalStore } from 'react'

/** Something changed for the signed-in player; reload what it's about. */
export type LiveEvent = {
  type: 'notification' | 'message' | 'friends' | 'presence' | 'offer' | 'read'
  /** Who it's about: the other side of a chat, a friend coming online, who answered an offer. */
  user?: string
  matchId?: string
  online?: boolean
}

const handlers = new Set<(e: LiveEvent) => void>()
let source: EventSource | null = null

/** Opens the event stream for a signed-in player, or closes it after logging out. */
export function connectLive(signedIn: boolean) {
  source?.close()
  source = null
  if (!signedIn) {
    setCounts({ unread: 0, unreadChats: 0 })
    return
  }
  source = new EventSource('/api/live')
  // Also on every reconnect, to pick up anything missed while the stream was down.
  source.onopen = () => refreshCounts()
  source.onmessage = (m) => {
    let e: LiveEvent
    try {
      e = JSON.parse(m.data)
    } catch {
      return
    }
    if (e.type === 'notification' || e.type === 'message' || e.type === 'read') refreshCounts()
    handlers.forEach((h) => h(e))
  }
}

/** Calls handler for every live event while the component is mounted. */
export function useLive(handler: (e: LiveEvent) => void) {
  const ref = useRef(handler)
  ref.current = handler
  useEffect(() => {
    const h = (e: LiveEvent) => ref.current(e)
    handlers.add(h)
    return () => {
      handlers.delete(h)
    }
  }, [])
}

/** Unread notifications for the bell, and chats with unread messages. */
export type Counts = { unread: number; unreadChats: number }

let counts: Counts = { unread: 0, unreadChats: 0 }
const countListeners = new Set<() => void>()

function setCounts(next: Counts) {
  counts = next
  countListeners.forEach((l) => l())
}

export async function refreshCounts() {
  const res = await fetch('/api/notifications').catch(() => null)
  if (!res?.ok) return
  const body = await res.json()
  setCounts({ unread: body.unread, unreadChats: body.unreadChats })
}

export function useCounts() {
  return useSyncExternalStore(
    (cb) => {
      countListeners.add(cb)
      return () => countListeners.delete(cb)
    },
    () => counts,
  )
}
