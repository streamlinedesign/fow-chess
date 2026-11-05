import { useSyncExternalStore } from 'react'
import { setCachedAvatar } from './avatars'
import { pieceImg, type Look, type PieceSet } from './pieces'
import { avatarImg, CLASSIC, lookKind, SET_IDS, setImg, setsFor, TILE_IDS, tileImg } from './sets'

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
  /** The skin for the light and the dark squares on your half of the board; "" for plain. */
  lightTiles: string
  darkTiles: string
  /** No skins at all: classic pieces for both sides (bots too) and the plain board. Your picks are kept. */
  plain: boolean
}

export const DEFAULT_PREFS: Prefs = {
  primary: CLASSIC,
  secondary: CLASSIC,
  side: '',
  avatar: '',
  lightTiles: '',
  darkTiles: '',
  plain: false,
}

function clean(p: Partial<Prefs> | null | undefined): Prefs {
  const out = { ...DEFAULT_PREFS, ...(p ?? {}) }
  if (!SET_IDS.includes(out.primary)) out.primary = CLASSIC
  if (!SET_IDS.includes(out.secondary)) out.secondary = CLASSIC
  if (out.side !== 'w' && out.side !== 'b') out.side = ''
  if (out.avatar && !avatarImg(out.avatar)) out.avatar = ''
  if (!TILE_IDS.includes(out.lightTiles)) out.lightTiles = ''
  if (!TILE_IDS.includes(out.darkTiles)) out.darkTiles = ''
  const { primary, secondary, side, avatar, lightTiles, darkTiles } = out
  return { primary, secondary, side, avatar, lightTiles, darkTiles, plain: out.plain === true }
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

const isDefault = (p: Prefs) => (Object.keys(DEFAULT_PREFS) as (keyof Prefs)[]).every((k) => p[k] === DEFAULT_PREFS[k])

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

/** The game's own set letters (pieces.ts) as sets here. */
const BOT_SETS: Record<PieceSet, string> = { w: 'cream', b: 'ink', r: 'red', g: 'green', v: 'violet' }

export type PieceImg = (letter: string, look?: Look) => string

/** Draws pieces in a set pair for a board with `bottom` at the bottom. */
export function piecesFor(p: Pick<Prefs, 'primary' | 'secondary' | 'plain'>, bottom: 'w' | 'b'): PieceImg {
  if (p.plain) return (letter, look) => pieceImg(letter, look && { ...look, sets: undefined })
  if (p.primary === CLASSIC && p.secondary === CLASSIC) return pieceImg
  const sets = setsFor(p.primary, p.secondary, bottom)
  const top = bottom === 'w' ? 'b' : 'w'
  return (letter, look) => {
    const color = letter === letter.toUpperCase() ? 'w' : 'b'
    const kind = lookKind(letter.toUpperCase(), look)
    // A bot plays in its own colour (see gameLook): that wins over your opponent's set,
    // and if your own set comes to the same, yours falls back to what the game picked.
    const forced = look?.sets
    if (forced?.[top]) {
      const theirs = BOT_SETS[forced[top]]
      if (color === top) return setImg(theirs, kind, color)
      const mine = sets[bottom] === theirs ? BOT_SETS[forced[bottom] ?? bottom] : sets[bottom]
      return setImg(mine, kind, color)
    }
    return setImg(sets[color], kind, color)
  }
}

/** The image function for a board seen from `bottom`'s side, in your sets: yours at the bottom. */
export function usePieces(bottom: 'w' | 'b' = 'w'): PieceImg {
  const p = usePrefs()
  return piecesFor(p, bottom)
}

/**
 * The tile image for a square, or undefined for a plain one. Skins cover only
 * your own half: ranks 1 to 4 as White, 5 to 8 as Black.
 */
export function tilesFor(p: Pick<Prefs, 'lightTiles' | 'darkTiles' | 'plain'>, bottom: 'w' | 'b') {
  if (p.plain || (!p.lightTiles && !p.darkTiles)) return () => undefined
  return (rank: number, light: boolean) => {
    if (bottom === 'w' ? rank >= 4 : rank < 4) return undefined
    return tileImg(light ? p.lightTiles : p.darkTiles, light)
  }
}

/** Tiles for a board seen from `bottom`'s side; rank counts from 0. */
export function useTiles(bottom: 'w' | 'b' = 'w') {
  return tilesFor(usePrefs(), bottom)
}
