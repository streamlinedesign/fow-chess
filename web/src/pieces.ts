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

// Pieces by Colin M.L. Burnett (cburnett), CC BY-SA 3.0. Small enough that Vite inlines them.
export const pieceSrc: Record<string, string> = { wK, wQ, wR, wB, wN, wP, bK, bQ, bR, bB, bN, bP }

/** The image for a FEN piece letter (uppercase is white). */
export function pieceImg(letter: string) {
  const white = letter === letter.toUpperCase()
  return pieceSrc[(white ? 'w' : 'b') + letter.toUpperCase()]
}

// Decode the images up front so the first render of the board doesn't flash.
for (const src of Object.values(pieceSrc)) {
  const img = new Image()
  img.src = src
}
