// Four-player chess, as the browser sees it. The server holds the rules; this
// file only names squares, colours and pieces for drawing.
import type { Look } from './pieces'

const svgs = import.meta.glob<string>('./pieces/four/*.svg', { eager: true, import: 'default' })

/** The seats in turn order: Red at the bottom, then clockwise. Must match four.Colors in server/four. */
export const SEATS = ['r', 'b', 'g', 'v'] as const
export type Seat = (typeof SEATS)[number]

/** The piece colours, for swatches. Must match the fills in pieces/four. */
export const SEAT_COLORS: Record<Seat, string> = { r: '#b65a40', b: '#4e6a9a', g: '#4e7f63', v: '#835482' }

/** Board width and height. The corners, three by three, are cut off. */
export const SIZE = 14
const FILES = 'abcdefghijklmn'

export const onBoard = (x: number, y: number) =>
  x >= 0 && y >= 0 && x < SIZE && y < SIZE && !((x < 3 || x > 10) && (y < 3 || y > 10))

export const squareName = (x: number, y: number) => FILES[x] + (y + 1)
export const squareXY = (sq: string) => [FILES.indexOf(sq[0]), Number(sq.slice(1)) - 1] as const
export const squareIdx = (sq: string) => {
  const [x, y] = squareXY(sq)
  return y * SIZE + x
}

/** A move as the server sends it: "e2e4", "g13g11", "e7e8q". */
export function parseMove(uci: string) {
  const m = /^([a-n]\d{1,2})([a-n]\d{1,2})([qrbn]?)$/.exec(uci)
  return m ? { from: m[1], to: m[2], promo: m[3] } : null
}

/**
 * Where square (x, y) is drawn for a player sitting at seat: their own pieces
 * at the bottom, as they'd see the board from their chair.
 */
export function cellFor(x: number, y: number, seat: Seat) {
  switch (seat) {
    case 'r':
      return { col: x, row: SIZE - 1 - y }
    case 'b':
      return { col: SIZE - 1 - y, row: SIZE - 1 - x }
    case 'g':
      return { col: SIZE - 1 - x, row: y }
    case 'v':
      return { col: y, row: x }
  }
}

/** seat's square at file f (0 to 7, from that player's left) and rank r (0 is their back row). Must match cell in server/four. */
export function ownCell(seat: Seat, f: number, r: number): [number, number] {
  switch (seat) {
    case 'r':
      return [3 + f, r]
    case 'b':
      return [r, 10 - f]
    case 'g':
      return [10 - f, SIZE - 1 - r]
    case 'v':
      return [SIZE - 1 - r, 3 + f]
  }
}

/** The seats in the order they sit round the board for the viewer: bottom, left, top, right. */
export function aroundFrom(seat: Seat): Seat[] {
  const i = SEATS.indexOf(seat)
  return [0, 1, 2, 3].map((k) => SEATS[(i + k) % 4])
}

/** The image for a piece code like "rK", drawn for the game's mods. */
export function pieceImg4(code: string, look?: Look) {
  let kind = code[1]
  if (look?.jesterPawns && kind === 'P') kind = 'J'
  if (look?.jumpKing && kind === 'K') kind = 'X'
  return svgs[`./pieces/four/${code[0]}${kind}.svg`]
}

/** The four centre squares: the hill, and the squares that give centre vision. Must match four.Centre. */
export const CENTRE = ['g7', 'h7', 'g8', 'h8']

/** The start position as a board array, for previews. Must match four.StartPosition. */
export function startBoard(): string[] {
  const back = 'RNBQKBNR'
  const b: string[] = []
  for (let y = 0; y < SIZE; y++) for (let x = 0; x < SIZE; x++) b.push(onBoard(x, y) ? '.' : ' ')
  const put = (x: number, y: number, code: string) => (b[y * SIZE + x] = code)
  for (let f = 0; f < 8; f++) {
    put(3 + f, 0, 'r' + back[f])
    put(3 + f, 1, 'rP')
    put(0, 10 - f, 'b' + back[f])
    put(1, 10 - f, 'bP')
    put(10 - f, 13, 'g' + back[f])
    put(10 - f, 12, 'gP')
    put(13, 3 + f, 'v' + back[f])
    put(12, 3 + f, 'vP')
  }
  return b
}

// Decode the images up front so the first board doesn't flash.
for (const src of Object.values(svgs)) {
  const img = new Image()
  img.src = src
}
