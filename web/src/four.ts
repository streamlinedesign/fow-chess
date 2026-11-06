// Four-player chess, as the browser sees it. The server holds the rules; this
// file only names squares, colours and pieces for drawing.
import type { Look, Skin } from './pieces'
import { CLASSIC, concrete, setImg, tileImg } from './sets'

const svgs = import.meta.glob<string>('./pieces/four/*.svg', { eager: true, import: 'default' })

/** The seats in turn order: Red at the bottom, then clockwise. Must match four.Colors in server/four. */
export const SEATS = ['r', 'b', 'g', 'v'] as const
export type Seat = (typeof SEATS)[number]

/** The piece colours, for swatches. Must match the fills in pieces/four. */
export const SEAT_COLORS: Record<Seat, string> = { r: '#b65a40', b: '#4e6a9a', g: '#4e7f63', v: '#835482' }

/**
 * The board's shape: the usual cross, 14 by 14 with the corners cut off and
 * three rows for each player, or the extended board, 16 by 16 with four rows
 * each and neutral corners anyone may cross. Must match four.Shape.
 */
export type BoardShape = '' | 'extended'

const FILES = 'abcdefghijklmnop'

export const squareName = (x: number, y: number) => FILES[x] + (y + 1)
export const squareXY = (sq: string) => [FILES.indexOf(sq[0]), Number(sq.slice(1)) - 1] as const

/** A move as the server sends it: "e2e4", "g13g11", "e7e8q". */
export function parseMove(uci: string) {
  const m = /^([a-p]\d{1,2})([a-p]\d{1,2})([qrbn]?)$/.exec(uci)
  return m ? { from: m[1], to: m[2], promo: m[3] } : null
}

/** A board shape's geometry. */
export type Geo = {
  shape: BoardShape
  /** Squares across. */
  size: number
  /** How many rows each player starts in. */
  rows: number
  onBoard: (x: number, y: number) => boolean
  /** A corner square on the extended board, which belongs to nobody. */
  neutral: (x: number, y: number) => boolean
  /** The index of a square in the board array the server sends. */
  squareIdx: (sq: string) => number
  /** Where square (x, y) is drawn for a player at seat: their own pieces at the bottom, as they'd see it from their chair. */
  cellFor: (x: number, y: number, seat: Seat) => { col: number; row: number }
  /** seat's square at file f (0 to 7, from that player's left) and rank r (0 is their back row). Must match Shape.Cell in server/four. */
  ownCell: (seat: Seat, f: number, r: number) => [number, number]
  /** The seat whose own rows hold square (x, y), if any. */
  homeOf: (x: number, y: number) => Seat | null
  /** The middle four squares: the hill on the cross. */
  centre: string[]
  /** The squares that give seat centre vision. Must match Shape.Outposts. */
  outposts: (seat: Seat) => string[]
  /** The start position as a board array, for previews. Must match four.StartPosition. */
  startBoard: () => string[]
}

function makeGeo(shape: BoardShape): Geo {
  const size = shape === 'extended' ? 16 : 14
  const a = shape === 'extended' ? 4 : 3
  const last = size - 1
  const corner = (x: number, y: number) => (x < a || x > last - a) && (y < a || y > last - a)
  const onBoard = (x: number, y: number) =>
    x >= 0 && y >= 0 && x < size && y < size && (shape === 'extended' || !corner(x, y))
  const neutral = (x: number, y: number) => shape === 'extended' && corner(x, y)
  const ownCell = (seat: Seat, f: number, r: number): [number, number] => {
    switch (seat) {
      case 'r':
        return [a + f, r]
      case 'b':
        return [r, a + 7 - f]
      case 'g':
        return [a + 7 - f, last - r]
      case 'v':
        return [last - r, a + f]
    }
  }
  const m = size / 2 - 1
  // Bottom left, bottom right, top left, top right.
  const centre = [squareName(m, m), squareName(m + 1, m), squareName(m, m + 1), squareName(m + 1, m + 1)]
  // On the extended board each seat has the pair just past the middle on the far side.
  const far: Record<Seat, number[]> = { r: [2, 3], b: [1, 3], g: [0, 1], v: [0, 2] }
  return {
    shape,
    size,
    rows: a,
    onBoard,
    neutral,
    squareIdx: (sq) => {
      const [x, y] = squareXY(sq)
      return y * size + x
    },
    cellFor: (x, y, seat) => {
      switch (seat) {
        case 'r':
          return { col: x, row: last - y }
        case 'b':
          return { col: last - y, row: last - x }
        case 'g':
          return { col: last - x, row: y }
        case 'v':
          return { col: y, row: x }
      }
    },
    ownCell,
    homeOf: (x, y) => {
      if (corner(x, y)) return null
      if (y < a) return 'r'
      if (x < a) return 'b'
      if (y > last - a) return 'g'
      if (x > last - a) return 'v'
      return null
    },
    centre,
    outposts: (seat) => (shape === 'extended' ? far[seat].map((i) => centre[i]) : centre),
    startBoard: () => {
      const back = 'RNBQKBNR'
      const b: string[] = []
      for (let y = 0; y < size; y++) for (let x = 0; x < size; x++) b.push(onBoard(x, y) ? '.' : ' ')
      for (const seat of SEATS)
        for (let f = 0; f < 8; f++) {
          const [bx, by] = ownCell(seat, f, 0)
          const [px, py] = ownCell(seat, f, 1)
          b[by * size + bx] = seat + back[f]
          b[py * size + px] = seat + 'P'
        }
      return b
    },
  }
}

const GEOS: Record<BoardShape, Geo> = { '': makeGeo(''), extended: makeGeo('extended') }

/** The geometry for a game's board setting. */
export const geo = (board?: string) => GEOS[board === 'extended' ? 'extended' : '']

/** The geometry of a board array as the server sends it, by its length. */
export const geoOf = (board: string[]) => GEOS[board.length === 256 ? 'extended' : '']

/** The seats in the order they sit round the board for the viewer: bottom, left, top, right. */
export function aroundFrom(seat: Seat): Seat[] {
  const i = SEATS.indexOf(seat)
  return [0, 1, 2, 3].map((k) => SEATS[(i + k) % 4])
}

/**
 * The mods, and how each seat is drawn: a player with a skin plays in one of
 * their own sets, the rest in their seat colour.
 */
export type FourLook = Look & { seats?: Partial<Record<Seat, SeatSkin>> }

/** A seat's pieces as a one-colour set (see sets.ts), and the tiles on its own rows (three, or four on the extended board). */
export type SeatSkin = { set?: string; lightTiles: string; darkTiles: string }

/** The image for a piece code like "rK", drawn for the game's mods and the seat's skin. */
export function pieceImg4(code: string, look?: FourLook) {
  let kind = code[1]
  if (look?.jesterPawns && kind === 'P') kind = 'J'
  if (look?.jumpKing && kind === 'K') kind = 'X'
  const set = look?.seats?.[code[0] as Seat]?.set
  return set ? setImg(set, kind) : svgs[`./pieces/four/${code[0]}${kind}.svg`]
}

/**
 * A seat's skin: the one of the player's White and Black sets the server
 * picked for this game (half), or the other if that one is classic. A player
 * with neither, a bot, or any player on a no-skins screen plays in the seat colour.
 */
export function seatSkin(skin: Skin | undefined, half: 'w' | 'b' | undefined): SeatSkin | undefined {
  if (!skin) return undefined
  const order: ['w' | 'b', string][] = [
    ['w', skin.white],
    ['b', skin.black],
  ]
  if (half === 'b') order.reverse()
  const pick = order.find(([, f]) => f !== CLASSIC)
  return { set: pick && concrete(pick[1], pick[0]), lightTiles: skin.lightTiles, darkTiles: skin.darkTiles }
}

/** The tile image for square (x, y), or undefined for a plain one. */
export function tile4(look: FourLook | undefined, g: Geo, x: number, y: number) {
  const home = g.homeOf(x, y)
  const s = home && look?.seats?.[home]
  if (!s) return undefined
  const light = (x + y) % 2 === 1
  return tileImg(light ? s.lightTiles : s.darkTiles, light)
}

// Decode the images up front so the first board doesn't flash.
for (const src of Object.values(svgs)) {
  const img = new Image()
  img.src = src
}
