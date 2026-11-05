import { useSyncExternalStore } from 'react'
import { setCachedAvatar } from './avatars'
import type { Look, Skin } from './pieces'
import { avatarImg, CLASSIC, concrete, familyOf, lookKind, SET_IDS, setImg, TILE_IDS, tileImg } from './sets'

/**
 * How a player likes the game to look: their set as White and as Black and
 * the tiles on their half of the board, which their opponents and spectators
 * see too; the side they like to play; their avatar. Guests keep them in this
 * browser; signed-in players on the server too, so they follow them to every
 * device. Must match prefs in server/prefs.go.
 */
export type Prefs = Skin & {
  /** The side you start as on the home board and get against bots; "" for either. */
  side: '' | 'w' | 'b'
  /** "" for the pawn, else "<set>/<kind>". */
  avatar: string
  /**
   * No skins on your screen: classic pieces and the plain board for everyone,
   * yours and your opponents' alike. Bots and four-player colours stay. Your picks are kept.
   */
  plain: boolean
}

export const DEFAULT_PREFS: Prefs = {
  white: CLASSIC,
  black: CLASSIC,
  side: '',
  avatar: '',
  lightTiles: '',
  darkTiles: '',
  plain: false,
}

function clean(p: (Partial<Prefs> & { primary?: string }) | null | undefined): Prefs {
  const out = { ...DEFAULT_PREFS, ...(p ?? {}) }
  // Before White and Black sets there was one set for your own pieces: it becomes both.
  if (p?.primary && !p.white && !p.black) out.white = out.black = familyOf(p.primary)
  if (!SET_IDS.includes(out.white)) out.white = CLASSIC
  if (!SET_IDS.includes(out.black)) out.black = CLASSIC
  if (out.side !== 'w' && out.side !== 'b') out.side = ''
  if (out.avatar && !avatarImg(out.avatar)) out.avatar = ''
  if (!TILE_IDS.includes(out.lightTiles)) out.lightTiles = ''
  if (!TILE_IDS.includes(out.darkTiles)) out.darkTiles = ''
  const { white, black, side, avatar, lightTiles, darkTiles } = out
  return { white, black, side, avatar, lightTiles, darkTiles, plain: out.plain === true }
}

/** The part of your preferences everyone in your games sees. */
export const skinOf = ({ white, black, lightTiles, darkTiles }: Skin): Skin => ({ white, black, lightTiles, darkTiles })

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

// Picks often come several in a row; the account gets the last of them.
let pushTimer: ReturnType<typeof setTimeout> | undefined
function push(p: Prefs) {
  clearTimeout(pushTimer)
  pushTimer = setTimeout(() => send(p), 400)
}

function send(p: Prefs) {
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
  if (isDefault(server) && !isDefault(prefs)) send(prefs)
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

/** The family a side is drawn in: a bot's set first, then nothing on a no-skins screen, then the player's own skin. */
function familyFor(p: Prefs, color: 'w' | 'b', mine: 'w' | 'b' | undefined, look?: Look) {
  const bot = look?.sets?.[color]
  if (bot) return bot
  if (p.plain) return CLASSIC
  const skin = color === mine ? p : look?.skins?.[color]
  return (color === 'w' ? skin?.white : skin?.black) ?? CLASSIC
}

/**
 * Draws pieces as a viewer with preferences `p` sees them: each side in its
 * player's own set for that colour (yours from `p` when you play `mine`, so a
 * change shows at once), a bot's side in the bot's set. No skins draws classic
 * for every player.
 */
export function piecesFor(p: Prefs, mine?: 'w' | 'b'): PieceImg {
  return (letter, look) => {
    const color = letter === letter.toUpperCase() ? 'w' : 'b'
    const kind = lookKind(letter.toUpperCase(), look)
    return setImg(concrete(familyFor(p, color, mine, look), color), kind, color)
  }
}

/** The image function for a board where you play `mine` (none for a spectator). */
export function usePieces(mine?: 'w' | 'b'): PieceImg {
  return piecesFor(usePrefs(), mine)
}

/**
 * The tile image for a square, or undefined for a plain one. A player's tiles
 * cover their own half: ranks 1 to 4 for White, 5 to 8 for Black. Rank counts from 0.
 */
export function tilesFor(p: Prefs, mine?: 'w' | 'b', look?: Look) {
  return (rank: number, light: boolean) => {
    const color = rank <= 3 ? 'w' : 'b'
    if (p.plain || look?.sets?.[color]) return undefined
    const skin = color === mine ? p : look?.skins?.[color]
    return skin ? tileImg(light ? skin.lightTiles : skin.darkTiles, light) : undefined
  }
}

/** Tiles for a board where you play `mine`. */
export function useTiles(mine?: 'w' | 'b', look?: Look) {
  return tilesFor(usePrefs(), mine, look)
}
