// The piece sets: the families a player picks for White and for Black, and the
// one-colour sets they are drawn in, each of which is also a row of avatars.
// Must match setFamilies and pieceSets in server/prefs.go.
import type { Look } from './pieces'

type Color = 'w' | 'b'

const classic = import.meta.glob<string>('./pieces/*.svg', { eager: true, import: 'default' })
const coloured = import.meta.glob<string>('./pieces/four/*.svg', { eager: true, import: 'default' })
const lightLines = import.meta.glob<string>('./pieces/light/*.svg', { eager: true, import: 'default' })
const inkLines = import.meta.glob<string>('./pieces/ink/*.svg', { eager: true, import: 'default' })
const themed = import.meta.glob<string>(
  './pieces/{mono,galaxy,blueprint,holy,ancient,cyberpunk,survey,bronze,silver,gold,grandmaster,king}/*.svg',
  { eager: true, import: 'default' },
)
const still = import.meta.glob<string>('./pieces/king-still/*.svg', { eager: true, import: 'default' })
/** Whether this player asked their device for less motion: animated sets are drawn still. */
const lessMotion = typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches
const tiles = import.meta.glob<string>('./tiles/*.svg', { eager: true, import: 'default' })

/** The plain colours, and the letter their files go by. */
export const COLOURS = ['red', 'blue', 'green', 'violet'] as const
const LETTER: Record<string, string> = { red: 'r', blue: 'b', green: 'g', violet: 'v' }

/** The themed sets: each has a light side for White and a dark side for Black, and tiles to match. */
export const THEMES = [
  'mono',
  'galaxy',
  'blueprint',
  'holy',
  'ancient',
  'cyberpunk',
  'survey',
  'bronze',
  'silver',
  'gold',
  'grandmaster',
  'king',
] as const

/**
 * What unlocks each themed set and its tiles and avatars: beating the bot that
 * wears it ("bot:<level>"), the title for reaching a rating in either mode
 * ("rating:<n>", see titles.ts), or "beta" for everyone during the beta. Plain colours are
 * never locked. Must match themeLocks in server/unlocks.go.
 */
export const THEME_LOCK: Record<string, string> = {
  king: 'rating:2500',
  grandmaster: 'rating:2000',
  holy: 'rating:1800',
  gold: 'rating:1600',
  silver: 'rating:1450',
  bronze: 'rating:1300',
  blueprint: 'rating:1150',
  survey: 'bot:vercingetorix',
  ancient: 'bot:ramesses',
  galaxy: 'bot:solomon',
  cyberpunk: 'beta',
}

/** The pieces a set has: the usual six, the jester pawn (J) and the prince (X). */
export const KINDS = ['K', 'Q', 'R', 'B', 'N', 'P', 'J', 'X'] as const

export type PieceSet = {
  id: string
  /** The image for one piece, by uppercase kind. */
  img: (kind: string) => string
}

const from = (files: Record<string, string>, dir: string, prefix: string) => (kind: string) =>
  files[`./pieces/${dir}${prefix}${kind}.svg`]

/**
 * The one-colour sets. Names are in the dictionary (pieces.sets). A colour has
 * a light-line half for White ("red-light") and a dark-line half for Black
 * ("red-ink"); the plain ids ("red") are the four-player seat colours, whose
 * lines are mixed, kept for those games and for avatars picked before.
 */
export const SETS: PieceSet[] = [
  { id: 'cream', img: from(classic, '', 'w') },
  { id: 'ink', img: from(classic, '', 'b') },
  ...COLOURS.flatMap((c) => [
    { id: c, img: from(coloured, 'four/', LETTER[c]) },
    { id: `${c}-light`, img: from(lightLines, 'light/', LETTER[c]) },
    { id: `${c}-ink`, img: from(inkLines, 'ink/', LETTER[c]) },
  ]),
  ...THEMES.flatMap((t) => {
    const [files, dir] = t === 'king' && lessMotion ? [still, 'king-still/'] : [themed, `${t}/`]
    return [
      { id: `${t}-light`, img: from(files, dir, 'w') },
      { id: `${t}-dark`, img: from(files, dir, 'b') },
    ]
  }),
]

/** Eat the King's own: cream for White, ink for Black. The default for both sides. */
export const CLASSIC = 'classic'

/** The sets a player picks from for White and for Black, in picker order. */
export const SET_IDS: string[] = [CLASSIC, ...COLOURS, ...THEMES]

const byId = new Map(SETS.map((s) => [s.id, s]))

/**
 * The one-colour set a family is drawn in for one side: classic is cream for
 * White and ink for Black, a colour cream drawn in it or ink lined in it, a theme its light
 * or dark half. So the sides never look alike, even in the same family.
 */
export function concrete(id: string, color: Color) {
  // A colour's own id is also its four-player set, so families come first.
  if (COLOURS.includes(id as (typeof COLOURS)[number])) return `${id}-${color === 'w' ? 'light' : 'ink'}`
  if (byId.has(id)) return id
  if (THEMES.includes(id as (typeof THEMES)[number])) return `${id}-${color === 'w' ? 'light' : 'dark'}`
  return color === 'w' ? 'cream' : 'ink'
}

/** The family a one-colour set belongs to, as old choices are carried over. */
export function familyOf(id: string) {
  return SET_IDS.find((f) => f !== CLASSIC && (id === f || id.startsWith(f + '-'))) ?? CLASSIC
}

/** The avatar rows, by family: White's half, then Black's. */
export const AVATAR_GROUPS = SET_IDS.map((f) => ({ id: f, sets: [concrete(f, 'w'), concrete(f, 'b')] }))

/** The tile skins: "" is the plain board. Must match tileSkins in server/prefs.go. */
export const TILE_IDS = ['', ...THEMES]

/** A tile skin's image for a light or dark square; undefined for the plain board. */
export const tileImg = (id: string, light: boolean) =>
  id ? tiles[`./tiles/${id}-${light ? 'l' : 'd'}.svg`] : undefined

/** The kind a piece is drawn as, for the game's mods. */
export function lookKind(kind: string, look?: Look) {
  if (look?.jesterPawns && kind === 'P') return 'J'
  if (look?.jumpKing && kind === 'K') return 'X'
  return kind
}

/** The image for one piece of a one-colour set (see concrete for a family); unknown sets draw in classic. */
export function setImg(id: string, kind: string, color: Color = 'w') {
  return (byId.get(id) ?? byId.get(concrete(id, color))!).img(kind.toUpperCase())
}

/** An avatar is "" for the default pawn, or "<set>/<kind>" like "red/N". Must match validAvatar in server/prefs.go. */
export function avatarImg(code: string | undefined) {
  if (!code) return null
  const [set, kind] = code.split('/')
  if (!byId.has(set) || !KINDS.includes(kind as (typeof KINDS)[number])) return null
  return setImg(set, kind)
}

/**
 * The colour of the fog over each tile skin's squares: a deep shade of the
 * skin, so hidden squares still read as that player's half. "" is the plain board.
 */
export const FOG_TINT: Record<string, string> = {
  '': '#34342c',
  mono: '#2a2a2a',
  galaxy: '#231b52',
  blueprint: '#112a58',
  holy: '#4a3c1c',
  ancient: '#3a2214',
  cyberpunk: '#3a1250',
  survey: '#2a3020',
  bronze: '#3a2416',
  silver: '#2a2f37',
  gold: '#3a2c0c',
  grandmaster: '#140b30',
  king: '#2a2210',
}

/** Tile skins whose board has a light passing over it (see .board-shine in index.css). */
export const SHINING = new Set<string>(['king'])

const skinOfTile = new Map<string, string>()
for (const id of THEMES) for (const light of [true, false]) skinOfTile.set(tileImg(id, light)!, id)

/** Whether a square drawn with this tile image has a light passing over it. */
export const shines = (tile: string | undefined) => !!tile && SHINING.has(skinOfTile.get(tile) ?? '') && !lessMotion

/** The fog colour over a square drawn with this tile image (undefined for a plain square). */
export const fogTint = (tile: string | undefined) => FOG_TINT[(tile && skinOfTile.get(tile)) || '']
