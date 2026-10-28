import { useEffect, useState, type ReactNode } from 'react'
import { pieceImg, squareIndex } from './position'
import { setupMode, type Settings } from './settings'
import { Button } from './ui'
import type { Color } from './useMatch'

const FILES = 'abcdefgh'
/** Every player's pieces, as uppercase FEN letters. Must match army in server/setup.go. */
const ARMY = 'KQRRBBNNPPPPPPPP'
const KINDS = ['K', 'Q', 'R', 'B', 'N', 'P']
const STANDARD = 'RNBQKBNRPPPPPPPP' + '.'.repeat(32) + 'pppppppprnbqkbnr'

/** How far square i is from color's back rank: 0 for the back rank itself. */
const ownRow = (color: Color, i: number) => (color === 'w' ? i >> 3 : 7 - (i >> 3))

/** color's pieces from the usual starting position, every other square empty. */
function ownStandard(color: Color) {
  return [...STANDARD].map((c, i) => (ownRow(color, i) < 2 ? c : '.')).join('')
}

/** Turns a setup into White's side of the board, and back: setups are remembered from White's side. */
function mirror(board: string) {
  const out = []
  for (let i = 0; i < 64; i++) {
    const c = board[(7 - (i >> 3)) * 8 + (i & 7)]
    out.push(c === '.' ? c : c === c.toUpperCase() ? c.toLowerCase() : c.toUpperCase())
  }
  return out.join('')
}

const storageKey = (setup: string) => `setup:${setup}`

/** The setup you used last time in this mode, or the usual position. */
function remembered(setup: string, color: Color) {
  try {
    const saved = localStorage.getItem(storageKey(setup))
    if (saved && saved.length === 64) return color === 'w' ? saved : mirror(saved)
  } catch {}
  return ownStandard(color)
}

function remember(setup: string, color: Color, board: string) {
  try {
    localStorage.setItem(storageKey(setup), color === 'w' ? board : mirror(board))
  } catch {}
}

type Pick = { tray: string } | { square: number } | null

/**
 * Where a player arranges their pieces before a setup game starts. Tap a piece
 * in the tray, then a square in your rows; tap a placed piece to move it, or
 * tap the tray to take it back off.
 */
export function SetupBoard({
  settings,
  you,
  ready,
  mine,
  theirs,
  onReady,
  onChange,
  below,
}: {
  settings: Settings
  you: Color
  ready: boolean
  /** What you sent, so a reload picks it up. */
  mine?: string
  /** The opponent's setup, shown once they're ready in a game without fog. */
  theirs?: string
  onReady: (board: string) => void
  onChange: () => void
  /** Shown right under the board, such as your player bar. */
  below?: ReactNode
}) {
  const mode = setupMode(settings)
  const [start] = useState(() => mine || remembered(mode.value, you))
  const [board, setBoard] = useState(start)
  const [pick, setPick] = useState<Pick>(null)
  useEffect(() => {
    if (mine) setBoard(mine)
  }, [mine])

  const fixed = (c: string) => mode.pawnsFixed && c.toUpperCase() === 'P'
  const allowed = (i: number) => ownRow(you, i) < mode.rows
  const own = (c: string) => c !== '.' && (c === c.toUpperCase()) === (you === 'w')
  const letter = (k: string) => (you === 'w' ? k : k.toLowerCase())

  // What's still in the tray: the army minus what's on the board.
  const left: Record<string, number> = {}
  for (const k of ARMY) left[k] = (left[k] ?? 0) + 1
  for (const c of board) if (own(c)) left[c.toUpperCase()]--
  const placedAll = KINDS.every((k) => left[k] === 0)

  const set = (b: string[]) => {
    setBoard(b.join(''))
    setPick(null)
  }

  const tapSquare = (i: number) => {
    if (ready) return
    const b = [...board]
    const here = b[i]
    if (pick && 'tray' in pick) {
      if (!allowed(i) || fixed(here)) return
      // Dropping onto one of your pieces sends that piece back to the tray.
      b[i] = letter(pick.tray)
      set(b)
      return
    }
    if (pick && 'square' in pick) {
      if (pick.square === i) return setPick(null)
      if (!allowed(i) || fixed(here)) return
      ;[b[i], b[pick.square]] = [b[pick.square], here]
      set(b)
      return
    }
    if (own(here) && !fixed(here) && allowed(i)) setPick({ square: i })
  }

  const tapTray = (k: string) => {
    if (ready) return
    if (pick && 'square' in pick) {
      const b = [...board]
      b[pick.square] = '.'
      return set(b)
    }
    if (left[k] > 0) setPick(pick && 'tray' in pick && pick.tray === k ? null : { tray: k })
  }

  const clear = () => set([...board].map((c) => (own(c) && !fixed(c) ? '.' : c)))

  // Your rows at the bottom, as in the game.
  const squares = []
  for (let row = 0; row < 8; row++) {
    for (let col = 0; col < 8; col++) {
      const file = you === 'w' ? col : 7 - col
      const rank = you === 'w' ? 7 - row : row
      const sq = FILES[file] + (rank + 1)
      const i = squareIndex(sq)
      const theirsHere = theirs?.[i] && theirs[i] !== '.' ? theirs[i] : null
      const piece = own(board[i]) ? board[i] : theirsHere
      const mineSide = ownRow(you, i) < 4
      // Under fog, or before they're ready, the other side stays covered.
      const covered = !mineSide && !theirs && settings.fog
      const selected = pick && 'square' in pick && pick.square === i
      squares.push(
        <button
          key={sq}
          type="button"
          className="sq"
          data-square={sq}
          data-light={((i & 7) + (i >> 3)) % 2 === 1}
          data-fog={covered}
          data-zone={!ready && allowed(i) && !fixed(board[i])}
          data-selected={!!selected}
          aria-label={`${sq}${piece ? `, ${piece}` : ''}`}
          onClick={() => tapSquare(i)}
          disabled={ready}
        >
          {col === 0 && <span className="coord top-[4%] left-[5%]">{sq[1]}</span>}
          {row === 7 && <span className="coord right-[5%] bottom-[4%]">{sq[0]}</span>}
          {piece && <img className="piece" src={pieceImg(piece)} alt="" draggable={false} />}
        </button>,
      )
    }
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="board setup-board grid aspect-square w-full grid-cols-8 grid-rows-8 overflow-hidden rounded-md shadow-[0_0_0_2px_var(--color-fg)]">
        {squares}
      </div>
      {below}

      <div
        className="tray"
        role="group"
        aria-label={pick && 'square' in pick ? 'Tap to take the piece off the board' : 'Your pieces to place'}
        data-dropping={!!pick && 'square' in pick}
        onClick={(e) => {
          // Tapping the tray itself, not a piece, takes the picked piece off the board.
          if (e.target === e.currentTarget && pick && 'square' in pick) tapTray('')
        }}
      >
        {KINDS.filter((k) => !(mode.pawnsFixed && k === 'P')).map((k) => (
          <button
            key={k}
            type="button"
            className="tray-piece"
            aria-pressed={!!pick && 'tray' in pick && pick.tray === k}
            aria-label={`${{ K: 'King', Q: 'Queen', R: 'Rook', B: 'Bishop', N: 'Knight', P: 'Pawn' }[k]}, ${left[k]} left`}
            disabled={ready || (left[k] === 0 && !(pick && 'square' in pick))}
            onClick={() => tapTray(k)}
          >
            <img src={pieceImg(letter(k))} alt="" draggable={false} />
            <span>{left[k]}</span>
          </button>
        ))}
      </div>

      <div className="grid grid-cols-4 gap-1.5">
        {ready ? (
          <Button className="col-span-2" onClick={onChange}>
            Change setup
          </Button>
        ) : (
          <>
            <Button
              variant="solid"
              className="px-2"
              disabled={!placedAll}
              onClick={() => {
                remember(mode.value, you, board)
                onReady(board)
              }}
            >
              Ready
            </Button>
            <Button variant="ghost" onClick={() => set([...start])}>
              Reset
            </Button>
            <Button variant="ghost" onClick={() => set([...ownStandard(you)])}>
              Default
            </Button>
            <Button variant="ghost" className="px-2" onClick={clear}>
              Clear
            </Button>
          </>
        )}
      </div>
    </div>
  )
}
