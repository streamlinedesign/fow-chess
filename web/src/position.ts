// The server sends the board as 64 characters from a1 to h8: FEN letters for
// pieces (uppercase white), '.' for empty, '?' for squares hidden by fog.

export function squareIndex(sq: string) {
  return (sq.charCodeAt(1) - 49) * 8 + (sq.charCodeAt(0) - 97)
}

/** The piece letter on sq, or null for an empty or hidden square. */
export function pieceAt(board: string, sq: string): string | null {
  const c = board[squareIndex(sq)]
  return c === '.' || c === '?' || c === undefined ? null : c
}

export { pieceImg } from './pieces'

/**
 * Plays a move on the board for display only, so our own move shows instantly
 * while the server confirms it. The server's next state always replaces this.
 */
export function applyLocal(board: string, uci: string) {
  const b = board.split('')
  const from = squareIndex(uci.slice(0, 2))
  const to = squareIndex(uci.slice(2, 4))
  let piece = b[from]
  const kind = piece.toLowerCase()
  const white = piece !== kind

  if (kind === 'p' && from % 8 !== to % 8 && b[to] === '.') {
    b[(from & ~7) + (to % 8)] = '.' // en passant: the captured pawn sits beside us
  }
  if (kind === 'k' && Math.abs(to - from) === 2) {
    const [rookFrom, rookTo] = to > from ? [from + 3, from + 1] : [from - 4, from - 1]
    b[rookTo] = b[rookFrom]
    b[rookFrom] = '.'
  }
  if (uci.length === 5) piece = white ? uci[4].toUpperCase() : uci[4]

  b[from] = '.'
  b[to] = piece
  return b.join('')
}
