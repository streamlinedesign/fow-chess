// The piece sets a player can pick for their own pieces and their opponent's.
// Every one-colour set is also a row of avatars. Must match pieceSets in server/prefs.go.
import type { Look } from './pieces'

type Color = 'w' | 'b'

const classic = import.meta.glob<string>('./pieces/*.svg', { eager: true, import: 'default' })
const coloured = import.meta.glob<string>('./pieces/four/*.svg', { eager: true, import: 'default' })
const inkLines = import.meta.glob<string>('./pieces/ink/*.svg', { eager: true, import: 'default' })

/** The pieces a set has: the usual six, the jester pawn (J) and the prince (X). */
export const KINDS = ['K', 'Q', 'R', 'B', 'N', 'P', 'J', 'X'] as const

export type PieceSet = {
  id: string
  /** The image for one piece, by uppercase kind. */
  img: (kind: string) => string
  /** A one-colour set can be an avatar and either side's set; "classic" is the only two-colour one. */
}

const from = (files: Record<string, string>, dir: string, prefix: string) => (kind: string) =>
  files[`./pieces/${dir}${prefix}${kind}.svg`]

/**
 * The one-colour sets, in picker order. Names are in the dictionary (pieces.sets).
 * To add a set: its SVGs in pieces/<name>/, a line here, its name in both
 * dictionaries, and its id in pieceSets in server/prefs.go.
 */
export const SETS: PieceSet[] = [
  { id: 'cream', img: from(classic, '', 'w') },
  { id: 'ink', img: from(classic, '', 'b') },
  { id: 'red', img: from(coloured, 'four/', 'r') },
  { id: 'blue', img: from(coloured, 'four/', 'b') },
  { id: 'green', img: from(coloured, 'four/', 'g') },
  { id: 'violet', img: from(coloured, 'four/', 'v') },
  { id: 'red-ink', img: from(inkLines, 'ink/', 'r') },
  { id: 'blue-ink', img: from(inkLines, 'ink/', 'b') },
  { id: 'green-ink', img: from(inkLines, 'ink/', 'g') },
  { id: 'violet-ink', img: from(inkLines, 'ink/', 'v') },
]

/** Eat the King's own: cream for White, ink for Black. The default for both sides. */
export const CLASSIC = 'classic'

/** Every set a player can pick for a side: classic first. */
export const SET_IDS = [CLASSIC, ...SETS.map((s) => s.id)]

const byId = new Map(SETS.map((s) => [s.id, s]))

/** The one-colour set a choice comes to for one side: classic is cream for White, ink for Black. */
export const concrete = (id: string, color: Color) => (byId.has(id) ? id : color === 'w' ? 'cream' : 'ink')

/**
 * Which set each colour is drawn in, for a board whose bottom is `bottom`:
 * your set for the pieces at the bottom, theirs for the top. If the two come
 * to the same set, the top side falls back to classic, so the sides never look alike.
 */
export function setsFor(primary: string, secondary: string, bottom: Color): Record<Color, string> {
  const top: Color = bottom === 'w' ? 'b' : 'w'
  const mine = concrete(primary, bottom)
  let theirs = concrete(secondary, top)
  if (theirs === mine) theirs = concrete(CLASSIC, top)
  if (theirs === mine) theirs = mine === 'cream' ? 'ink' : 'cream'
  return { [bottom]: mine, [top]: theirs } as Record<Color, string>
}

/** The kind a piece is drawn as, for the game's mods. */
export function lookKind(kind: string, look?: Look) {
  if (look?.jesterPawns && kind === 'P') return 'J'
  if (look?.jumpKing && kind === 'K') return 'X'
  return kind
}

/** The image for one piece of a one-colour set; unknown sets draw in classic. */
export function setImg(id: string, kind: string, color: Color = 'w') {
  return (byId.get(id) ?? byId.get(concrete(CLASSIC, color))!).img(kind.toUpperCase())
}

/** An avatar is "" for the default pawn, or "<set>/<kind>" like "red/N". Must match validAvatar in server/prefs.go. */
export function avatarImg(code: string | undefined) {
  if (!code) return null
  const [set, kind] = code.split('/')
  if (!byId.has(set) || !KINDS.includes(kind as (typeof KINDS)[number])) return null
  return setImg(set, kind)
}
