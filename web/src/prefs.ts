import { useSyncExternalStore } from 'react'
import { setCachedAvatar } from './avatars'
import type { Look, Skin } from './pieces'
import { avatarImg, CLASSIC, concrete, familyOf, lookKind, SET_IDS, setImg, THEME_LOCK, TILE_IDS, tileImg } from './sets'

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
  /** How hidden squares look on your screen, and nobody else's: see FOG_STYLES. */
  fog: FogStyle
}

/**
 * Smoke draws one soft cloud over what you can't see; dusk leaves the squares
 * under a dark veil, so you still see their colour. Must match fogStyles in server/prefs.go.
 */
export const FOG_STYLES = ['smoke', 'dusk'] as const
export type FogStyle = (typeof FOG_STYLES)[number]

export const DEFAULT_PREFS: Prefs = {
  white: CLASSIC,
  black: CLASSIC,
  side: '',
  avatar: '',
  lightTiles: '',
  darkTiles: '',
  plain: false,
  fog: 'smoke',
}

/**
 * The locked themes this viewer has unlocked, from the server; null until it
 * has answered, and meanwhile nothing counts as locked.
 */
let unlocked: ReadonlySet<string> | null = null

/** Whether a set family or tile skin may be worn: a plain colour, or a theme this viewer has unlocked. */
export const isUnlocked = (id: string) => !THEME_LOCK[id] || !unlocked || unlocked.has(id)

/** Asks the server again what this viewer has unlocked: on signing in or out, and after beating a bot. */
export function refreshUnlocks() {
  fetch('/api/unlocks')
    .then((r) => (r.ok ? r.json() : null))
    .then((u: { themes: string[] } | null) => {
      if (!u) return
      unlocked = new Set(u.themes)
      // What's still locked goes back to the default here too; the account already has it so.
      set(clean(prefs))
    })
    .catch(() => {})
}

/** The unlocked themes, for the pickers. */
export function useUnlocked() {
  return useSyncExternalStore(
    (cb) => {
      listeners.add(cb)
      return () => listeners.delete(cb)
    },
    () => unlocked,
  )
}

function clean(p: (Partial<Prefs> & { primary?: string }) | null | undefined): Prefs {
  const out = { ...DEFAULT_PREFS, ...(p ?? {}) }
  // Before White and Black sets there was one set for your own pieces: it becomes both.
  if (p?.primary && !p.white && !p.black) out.white = out.black = familyOf(p.primary)
  if (!SET_IDS.includes(out.white) || !isUnlocked(out.white)) out.white = CLASSIC
  if (!SET_IDS.includes(out.black) || !isUnlocked(out.black)) out.black = CLASSIC
  if (out.side !== 'w' && out.side !== 'b') out.side = ''
  if (out.avatar && (!avatarImg(out.avatar) || !isUnlocked(familyOf(out.avatar.split('/')[0])))) out.avatar = ''
  if (!TILE_IDS.includes(out.lightTiles) || !isUnlocked(out.lightTiles)) out.lightTiles = ''
  if (!TILE_IDS.includes(out.darkTiles) || !isUnlocked(out.darkTiles)) out.darkTiles = ''
  const { white, black, side, avatar, lightTiles, darkTiles } = out
  const fog = FOG_STYLES.includes(out.fog) ? out.fog : 'smoke'
  return { white, black, side, avatar, lightTiles, darkTiles, plain: out.plain === true, fog }
}

/** The part of your preferences everyone in your games sees. */
export const skinOf = ({ white, black, lightTiles, darkTiles }: Skin): Skin => ({ white, black, lightTiles, darkTiles })

// A guest's picks and the signed-in account's are kept apart, so signing out
// goes back to what this browser picked as a guest (the defaults if nothing)
// rather than keeping the account's sets. The account's copy is only a cache
// for the first paint before /api/me answers, and is dropped on sign-out.
const GUEST_KEY = 'prefs.guest'
const ACCOUNT_KEY = 'prefs.account'

function load(key: string): Prefs | null {
  try {
    const raw = localStorage.getItem(key)
    return raw ? clean(JSON.parse(raw)) : null
  } catch {
    return null
  }
}

function forget(key: string) {
  try {
    localStorage.removeItem(key)
  } catch {}
}

// 'prefs' held whichever was last, guest or account, so it can't be trusted as
// a guest's own picks.
forget('prefs')

let prefs = load(ACCOUNT_KEY) ?? load(GUEST_KEY) ?? DEFAULT_PREFS
showFog(prefs)
let username: string | null = null
const listeners = new Set<() => void>()

const isDefault = (p: Prefs) => (Object.keys(DEFAULT_PREFS) as (keyof Prefs)[]).every((k) => p[k] === DEFAULT_PREFS[k])

/** The fog style is drawn by CSS, from the page's root (see index.css). */
function showFog(p: Prefs) {
  if (typeof document !== 'undefined') document.documentElement.dataset.fogStyle = p.fog
}

function set(next: Prefs) {
  prefs = next
  showFog(next)
  try {
    localStorage.setItem(username ? ACCOUNT_KEY : GUEST_KEY, JSON.stringify(next))
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
 * account that has none yet takes what this browser picked as a guest. A guest
 * gets this browser's guest picks back.
 */
export function syncPrefs(me: (Partial<Prefs> & { username: string }) | null) {
  username = me?.username ?? null
  refreshUnlocks()
  if (!me) {
    forget(ACCOUNT_KEY)
    set(load(GUEST_KEY) ?? DEFAULT_PREFS)
    return
  }
  const guest = load(GUEST_KEY) ?? DEFAULT_PREFS
  const server = clean(me)
  if (isDefault(server) && !isDefault(guest)) {
    send(guest)
    set(guest)
  } else set(server)
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
    // A bot's half is in its own tiles, like its pieces, skins or not.
    const bot = look?.tiles?.[color]
    if (bot !== undefined) return tileImg(bot, light)
    if (p.plain || look?.sets?.[color]) return undefined
    const skin = color === mine ? p : look?.skins?.[color]
    return skin ? tileImg(light ? skin.lightTiles : skin.darkTiles, light) : undefined
  }
}

/** Tiles for a board where you play `mine`. */
export function useTiles(mine?: 'w' | 'b', look?: Look) {
  return tilesFor(usePrefs(), mine, look)
}
