import { useSyncExternalStore } from 'react'
import { dict, errorText, type Dict } from './i18n'
import type { Settings } from './settings'

/** Another player in a list or a chat. */
export type UserCard = {
  username: string
  rating: number
  online: boolean
  /** Unix ms; missing when they hide it. */
  lastSeen?: number
  /** The match they're playing right now, to watch. */
  playing?: string
}

export type FriendState = 'none' | 'friends' | 'sent' | 'received'

export type Offer = {
  id: number
  from: string
  to: string
  settings: Settings
  /** The offering player's colour. */
  color: 'w' | 'b' | 'random'
  status: 'open' | 'accepted' | 'declined' | 'cancelled'
  matchId?: string
  at: number
}

export type Notification = {
  id: number
  kind: 'friend_request' | 'friend_accepted' | 'offer' | 'offer_accepted' | 'offer_declined' | 'game_over' | 'move'
  actor?: string
  offer?: Offer
  matchId?: string
  round?: number
  data: { result?: 'won' | 'lost' | 'draw'; reason?: string; kingCapture?: boolean; sniper?: boolean }
  read: boolean
  at: number
  pending?: boolean
}

export type Message = { id: number; from: string; text?: string; offer?: Offer; at: number }

/**
 * Calls the API. Resolves to the JSON body, or throws an Error with the
 * server's message so callers can show it.
 */
export async function api<T = unknown>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(path, {
    method,
    headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body),
  }).catch(() => null)
  if (!res) throw new Error(dict().common.cantReach)
  const data = res.status === 204 ? null : await res.json().catch(() => null)
  if (!res.ok) throw new Error(errorText(data))
  return data as T
}

/** "Online", "Last seen 40 min ago", or nothing when hidden. */
export function presence(u: Pick<UserCard, 'online' | 'lastSeen' | 'playing'>, d: Dict) {
  if (u.playing) return d.presence.inGame
  if (u.online) return d.common.online
  if (!u.lastSeen) return ''
  const s = (Date.now() - u.lastSeen) / 1000
  if (s < 60) return d.presence.lastSeenJustNow
  if (s < 3600) return d.presence.lastSeenMin(Math.floor(s / 60))
  if (s < 86400) return d.presence.lastSeenH(Math.floor(s / 3600))
  if (s < 86400 * 7) return d.presence.lastSeenD(Math.floor(s / 86400))
  return d.presence.lastSeen(new Intl.DateTimeFormat(d.locale, { day: 'numeric', month: 'short' }).format(u.lastSeen))
}

// The Offer a match dialog, opened from anywhere.
let offerTo: string | null = null
const offerListeners = new Set<() => void>()

export function openOffer(username: string | null) {
  offerTo = username
  offerListeners.forEach((l) => l())
}

export function useOfferTo() {
  return useSyncExternalStore(
    (cb) => {
      offerListeners.add(cb)
      return () => offerListeners.delete(cb)
    },
    () => offerTo,
  )
}

// Short notes at the bottom of the screen, like "ana accepted your offer".
export type Toast = { id: number; text: string; action?: { label: string; href: string } }
let toasts: Toast[] = []
let toastID = 0
const toastListeners = new Set<() => void>()

function setToasts(next: Toast[]) {
  toasts = next
  toastListeners.forEach((l) => l())
}

export function showToast(t: Omit<Toast, 'id'>, ms = 8000) {
  const id = ++toastID
  setToasts([...toasts, { ...t, id }].slice(-3))
  setTimeout(() => dismissToast(id), ms)
}

export function dismissToast(id: number) {
  setToasts(toasts.filter((t) => t.id !== id))
}

export function useToasts() {
  return useSyncExternalStore(
    (cb) => {
      toastListeners.add(cb)
      return () => toastListeners.delete(cb)
    },
    () => toasts,
  )
}
