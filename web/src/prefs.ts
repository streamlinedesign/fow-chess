import { useSyncExternalStore } from 'react'
import { setCachedAvatar } from './avatars'
import { pieceImg, type Look } from './pieces'
import { avatarImg, CLASSIC, lookKind, SET_IDS, setImg, setsFor } from './sets'

/**
 * How a player likes the game to look: a set for their own pieces and one for
 * their opponent's, the side they like to play, and their avatar. Guests keep
 * them in this browser; signed-in players on the server too, so they follow
 * them to every device. Must match prefs in server/prefs.go.
 */
export type Prefs = {
  primary: string
  secondary: string
  /** The side you start as on the home board and get against bots; "" for either. */
  side: '' | 'w' | 'b'
  /** "" for the pawn, else "<set>/<kind>". */
  avatar: string
}

export const DEFAULT_PREFS: Prefs = { primary: CLASSIC, secondary: CLASSIC, side: '', avatar: '' }

function clean(p: Partial<Prefs> | null | undefined): Prefs {
  const out = { ...DEFAULT_PREFS, ...(p ?? {}) }
  if (!SET_IDS.includes(out.primary)) out.primary = CLASSIC
  if (!SET_IDS.includes(out.secondary)) out.secondary = CLASSIC
  if (out.side !== 'w' && out.side !== 'b') out.side = ''
  if (out.avatar && !avatarImg(out.avatar)) out.avatar = ''
  return { primary: out.primary, secondary: out.secondary, side: out.side, avatar: out.avatar }
}

function loadLocal(): Prefs {
  try {
    return clean(JSON.parse(localStorage.getItem('prefs') ?? 'null'))
  } catch {
    return DEFAULT_PREFS
  }
}

let prefs = loadLocal()
let username: string | null = null
const listeners = new Set<() => void>()

const isDefault = (p: Prefs) => p.primary === CLASSIC && p.secondary === CLASSIC && p.side === '' && p.avatar === ''

function set(next: Prefs) {
  prefs = next
  try {
    localStorage.setItem('prefs', JSON.stringify(next))
  } catch {}
  if (username) setCachedAvatar(username, next.avatar)
  listeners.forEach((l) => l())
}

function push(p: Prefs) {
  fetch('/api/prefs', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(p),
  }).catch(() => {})
}

/**
 * Called whenever the signed-in user changes. The account's choices win; an
 * account that has none yet takes what this browser picked as a guest.
 */
export function syncPrefs(me: (Partial<Prefs> & { username: string }) | null) {
  username = me?.username ?? null
  if (!me) return
  const server = clean(me)
  if (isDefault(server) && !isDefault(prefs)) push(prefs)
  else set(server)
  setCachedAvatar(me.username, prefs.avatar)
}

/** Changes some preferences, here and, when signed in, on the account. */
export function setPrefs(patch: Partial<Prefs>) {
  const next = clean({ ...prefs, ...patch })
  set(next)
  if (username) push(next)
}

/** The current preferences, outside React. */
export const getPrefs = () => prefs

export function usePrefs() {
  return useSyncExternalStore(
    (cb) => {
      listeners.add(cb)
      return () => listeners.delete(cb)
    },
    () => prefs,
  )
}

export type PieceImg = (letter: string, look?: Look) => string

/** Draws pieces in a set pair for a board with `bottom` at the bottom. */
export function piecesFor(p: Pick<Prefs, 'primary' | 'secondary'>, bottom: 'w' | 'b'): PieceImg {
  if (p.primary === CLASSIC && p.secondary === CLASSIC) return pieceImg
  const sets = setsFor(p.primary, p.secondary, bottom)
  return (letter, look) => {
    const color = letter === letter.toUpperCase() ? 'w' : 'b'
    return setImg(sets[color], lookKind(letter.toUpperCase(), look), color)
  }
}

/** The image function for a board seen from `bottom`'s side, in your sets: yours at the bottom. */
export function usePieces(bottom: 'w' | 'b' = 'w'): PieceImg {
  const p = usePrefs()
  return piecesFor(p, bottom)
}
