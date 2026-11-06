import { concrete, setImg } from './sets'
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
import wX from './pieces/wX.svg'
import bX from './pieces/bX.svg'

// Eat the King's own set: cream pieces with black lines, black pieces with a cream rim so they
// still show on the dark theme's background. Small enough that Vite inlines them.
// The mods have their own pieces: J is the jester pawn, X the jump king (a prince).
export const pieceSrc: Record<string, string> = {
  ...{ wK, wQ, wR, wB, wN, wP, wJ, wX },
  ...{ bK, bQ, bR, bB, bN, bP, bJ, bX },
}

/** How a player likes their own pieces and squares drawn; see prefs.ts. Must match skin in server/prefs.go. */
export type Skin = { white: string; black: string; lightTiles: string; darkTiles: string }

/**
 * The mods that change how pieces are drawn, the set a bot's side is drawn in
 * (a family from sets.ts, which every viewer sees, skins or not), and each
 * player's own skin. A game's settings will do.
 */
export type Look = {
  jesterPawns?: boolean
  jumpKing?: boolean
  sets?: Partial<Record<'w' | 'b', string>>
  /** A bot's tile skin on its half ("" for plain), seen by everyone. */
  tiles?: Partial<Record<'w' | 'b', string>>
  skins?: Partial<Record<'w' | 'b', Skin>>
}

/** The image for a FEN piece letter (uppercase is white), drawn for the game's mods and a bot's set. */
export function pieceImg(letter: string, look?: Look) {
  const side = letter === letter.toUpperCase() ? 'w' : 'b'
  let kind = letter.toUpperCase()
  if (look?.jesterPawns && kind === 'P') kind = 'J'
  if (look?.jumpKing && kind === 'K') kind = 'X'
  const set = look?.sets?.[side]
  return set ? setImg(concrete(set, side), kind, side) : pieceSrc[side + kind]
}

// Decode the images up front so the first render of the board doesn't flash.
for (const src of Object.values(pieceSrc)) {
  const img = new Image()
  img.src = src
}
