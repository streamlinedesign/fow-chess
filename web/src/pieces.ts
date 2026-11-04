import wK from './pieces/wK.svg'
import wQ from './pieces/wQ.svg'
import wR from './pieces/wR.svg'
import wB from './pieces/wB.svg'
import wN from './pieces/wN.svg'
import wP from './pieces/wP.svg'
import bK from './pieces/bK.svg'
import bQ from './pieces/bQ.svg'
import bR from './pieces/bR.svg'
import bB from './pieces/bB.svg'
import bN from './pieces/bN.svg'
import bP from './pieces/bP.svg'
import wJ from './pieces/wJ.svg'
import bJ from './pieces/bJ.svg'

// Eat the King's own set: cream pieces with black lines, black pieces with a cream rim so they
// still show on the dark theme's background. Small enough that Vite inlines them.
// J is the jester pawn, drawn for pawns in a jester pawns game.
export const pieceSrc: Record<string, string> = { wK, wQ, wR, wB, wN, wP, wJ, bK, bQ, bR, bB, bN, bP, bJ }

/** The image for a FEN piece letter (uppercase is white). With jester, pawns are drawn as jesters. */
export function pieceImg(letter: string, jester = false) {
  const white = letter === letter.toUpperCase()
  const kind = letter.toUpperCase()
  return pieceSrc[(white ? 'w' : 'b') + (jester && kind === 'P' ? 'J' : kind)]
}

// Decode the images up front so the first render of the board doesn't flash.
for (const src of Object.values(pieceSrc)) {
  const img = new Image()
  img.src = src
}
