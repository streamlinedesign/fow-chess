import { useEffect, useRef, useState, type ReactNode } from 'react'
import { useT } from './i18n'
import { pieceImg, squareIndex } from './position'
import { HILL, setupMode, type Settings } from './settings'
import { Button } from './ui'
import type { Color } from './useMatch'

const FILES = 'abcdefgh'
/** Every player's pieces, as uppercase FEN letters. Must match army in server/setup.go. */
const ARMY = 'KQRRBBNNPPPPPPPP'
const KINDS = ['K', 'Q', 'R', 'B', 'N', 'P']
const STANDARD = 'RNBQKBNRPPPPPPPP' + '.'.repeat(32) + 'pppppppprnbqkbnr'
const HILL_INDEXES = HILL.map(squareIndex)

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

/** A drag in progress: a piece from the tray, or one already on the board. */
type Drag = {
  x: number
  y: number
  moved: boolean
  /** The tray piece being dragged, or the square it comes from. */
  source: { tray: string } | { square: number; el: HTMLElement }
  ghost?: HTMLImageElement
}

const DRAG_START = 4

/**
 * Where a player arranges their pieces before a setup game starts. Drag a
 * piece from the tray onto a square in your rows, or tap it and then the
 * square. Tap or drag a placed piece to move it, or drop it on the tray to
 * take it back off.
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
  const t = useT()
  const kindNames: Record<string, string> = {
    K: t.match.king,
    Q: t.match.queen,
    R: t.match.rook,
    B: t.match.bishop,
    N: t.match.knight,
    P: t.match.pawn,
  }
  const mode = setupMode(settings)
  const [start] = useState(() => mine || remembered(mode.value, you))
  const [board, setBoard] = useState(start)
  const [pick, setPick] = useState<Pick>(null)
  const boardRef = useRef<HTMLDivElement>(null)
  const trayRef = useRef<HTMLDivElement>(null)
  const drag = useRef<Drag | null>(null)
  // A drag that moved ends with a click on whatever is under the pointer; that click is not a tap.
  const swallowClick = useRef(false)
  useEffect(() => {
    if (mine) setBoard(mine)
  }, [mine])

  const fixed = (c: string) => mode.pawnsFixed && c.toUpperCase() === 'P'
  const inRows = (i: number) => ownRow(you, i) < mode.rows
  const own = (c: string) => c !== '.' && (c === c.toUpperCase()) === (you === 'w')
  const letter = (k: string) => (you === 'w' ? k : k.toLowerCase())
  const kindOf = (c: string) => c.toUpperCase()
  /** Whether a piece of kind k may go on square i. The king can't start on the hill. */
  const allowed = (i: number, k: string) =>
    inRows(i) && !fixed(board[i]) && !(settings.hill && k === 'K' && HILL_INDEXES.includes(i))

  // What's still in the tray: the army minus what's on the board.
  const left: Record<string, number> = {}
  for (const k of ARMY) left[k] = (left[k] ?? 0) + 1
  for (const c of board) if (own(c)) left[kindOf(c)]--
  const placedAll = KINDS.every((k) => left[k] === 0)
  const kingOnHill = !!settings.hill && HILL_INDEXES.some((i) => own(board[i]) && kindOf(board[i]) === 'K')

  const set = (b: string[], next: Pick = null) => {
    setBoard(b.join(''))
    setPick(next)
  }

  /** Puts tray piece k on square i. Any piece there goes back to the tray. */
  const placeFromTray = (k: string, i: number) => {
    if (!allowed(i, k)) return false
    const b = [...board]
    const here = b[i]
    b[i] = letter(k)
    // When you have more of the same piece, keep it picked so the next tap places another.
    const remaining = left[k] - 1 + (own(here) && kindOf(here) === k ? 1 : 0)
    set(b, remaining > 0 ? { tray: k } : null)
    return true
  }

  /** Moves the piece on square from to square to, swapping with whatever is there. */
  const moveOnBoard = (from: number, to: number) => {
    if (from === to || !allowed(to, kindOf(board[from]))) return false
    const b = [...board]
    // The piece coming back the other way must be allowed where the first one was, too.
    if (own(b[to]) && !allowed(from, kindOf(b[to]))) return false
    ;[b[to], b[from]] = [b[from], b[to]]
    set(b)
    return true
  }

  const removeFromBoard = (i: number) => {
    const b = [...board]
    b[i] = '.'
    set(b)
  }

  const tapSquare = (i: number) => {
    if (ready) return
    if (swallowClick.current) {
      swallowClick.current = false
      return
    }
    const here = board[i]
    if (pick && 'tray' in pick) {
      placeFromTray(pick.tray, i)
      return
    }
    if (pick && 'square' in pick) {
      if (pick.square === i) return setPick(null)
      moveOnBoard(pick.square, i)
      return
    }
    if (own(here) && !fixed(here) && inRows(i)) setPick({ square: i })
  }

  const tapTray = (k: string) => {
    if (ready) return
    if (swallowClick.current) {
      swallowClick.current = false
      return
    }
    if (pick && 'square' in pick) return removeFromBoard(pick.square)
    if (left[k] > 0) setPick(pick && 'tray' in pick && pick.tray === k ? null : { tray: k })
  }

  const clear = () => set([...board].map((c) => (own(c) && !fixed(c) ? '.' : c)))

  // Dragging. Taps stay click events on the buttons; a pointer that moves becomes a drag.
  const squareFromPoint = (x: number, y: number) => {
    const rect = boardRef.current!.getBoundingClientRect()
    const col = Math.floor(((x - rect.left) / rect.width) * 8)
    const row = Math.floor(((y - rect.top) / rect.height) * 8)
    if (col < 0 || col > 7 || row < 0 || row > 7) return null
    const file = you === 'w' ? col : 7 - col
    const rank = you === 'w' ? 7 - row : row
    return rank * 8 + file
  }
  const overTray = (x: number, y: number) => {
    const r = trayRef.current?.getBoundingClientRect()
    return !!r && x >= r.left && x <= r.right && y >= r.top && y <= r.bottom
  }

  const startDrag = (e: React.PointerEvent, source: Drag['source']) => {
    if (e.button !== 0 || ready || drag.current) return
    drag.current = { x: e.clientX, y: e.clientY, moved: false, source }
    const onMove = (ev: PointerEvent) => {
      const d = drag.current
      if (!d) return
      if (!d.moved && Math.hypot(ev.clientX - d.x, ev.clientY - d.y) < DRAG_START) return
      d.moved = true
      if ('tray' in d.source) {
        if (!d.ghost) {
          d.ghost = document.createElement('img')
          d.ghost.src = pieceImg(letter(d.source.tray))
          d.ghost.className = 'drag-ghost'
          d.ghost.style.width = `${boardRef.current!.getBoundingClientRect().width / 8}px`
          document.body.append(d.ghost)
        }
        d.ghost.style.transform = `translate(${ev.clientX}px, ${ev.clientY}px) translate(-50%, -50%) scale(1.15)`
      } else {
        const r = d.source.el.parentElement!.getBoundingClientRect()
        d.source.el.dataset.dragging = 'true'
        d.source.el.style.transform = `translate(${ev.clientX - (r.left + r.width / 2)}px, ${ev.clientY - (r.top + r.height / 2)}px) scale(1.15)`
      }
    }
    const end = (ev: PointerEvent, cancelled: boolean) => {
      window.removeEventListener('pointermove', onMove)
      window.removeEventListener('pointerup', onUp)
      window.removeEventListener('pointercancel', onCancel)
      const d = drag.current
      drag.current = null
      if (!d) return
      d.ghost?.remove()
      if ('square' in d.source) {
        d.source.el.style.transform = ''
        delete d.source.el.dataset.dragging
      }
      if (cancelled || !d.moved) return
      swallowClick.current = true
      // The click that follows lands on whatever is under the pointer, or nowhere: either way, forget it soon.
      setTimeout(() => (swallowClick.current = false), 0)
      const to = squareFromPoint(ev.clientX, ev.clientY)
      if ('tray' in d.source) {
        if (to !== null) placeFromTray(d.source.tray, to)
      } else if (overTray(ev.clientX, ev.clientY)) {
        removeFromBoard(d.source.square)
      } else if (to !== null) {
        moveOnBoard(d.source.square, to)
      }
    }
    const onUp = (ev: PointerEvent) => end(ev, false)
    const onCancel = (ev: PointerEvent) => end(ev, true)
    window.addEventListener('pointermove', onMove)
    window.addEventListener('pointerup', onUp)
    window.addEventListener('pointercancel', onCancel)
  }

  const pickedKind = pick && 'tray' in pick ? pick.tray : pick && 'square' in pick ? kindOf(board[pick.square]) : null

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
      // The squares the picked piece may go to, or every square you may use while nothing is picked.
      const zone = !ready && (pickedKind ? allowed(i, pickedKind) : inRows(i) && !fixed(board[i]))
      const draggable = !ready && own(board[i]) && !fixed(board[i]) && inRows(i)
      squares.push(
        <button
          key={sq}
          type="button"
          className="sq"
          data-square={sq}
          data-light={((i & 7) + (i >> 3)) % 2 === 1}
          data-fog={covered}
          data-zone={zone}
          data-selected={!!selected}
          data-hill={settings.hill && HILL_INDEXES.includes(i)}
          aria-label={`${sq}${piece ? `, ${piece}` : ''}`}
          onClick={() => tapSquare(i)}
          onPointerDown={(e) => {
            const el = e.currentTarget.querySelector<HTMLElement>('.piece')
            if (draggable && el) startDrag(e, { square: i, el })
          }}
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
      <div
        ref={boardRef}
        className="board setup-board grid aspect-square w-full grid-cols-8 grid-rows-8 overflow-hidden rounded-md shadow-[0_0_0_2px_var(--color-fg)]"
        onContextMenu={(e) => e.preventDefault()}
      >
        {squares}
      </div>
      {below}

      <div
        ref={trayRef}
        className="tray"
        role="group"
        aria-label={pick && 'square' in pick ? t.setup.tapToRemove : t.setup.yourPieces}
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
            aria-label={t.setup.left(kindNames[k], left[k])}
            disabled={ready || (left[k] === 0 && !(pick && 'square' in pick))}
            onClick={() => tapTray(k)}
            onPointerDown={(e) => {
              if (left[k] > 0) startDrag(e, { tray: k })
            }}
          >
            <img src={pieceImg(letter(k))} alt="" draggable={false} />
            <span>{left[k]}</span>
          </button>
        ))}
      </div>

      <div className="grid grid-cols-4 gap-1.5">
        {ready ? (
          <Button className="col-span-2" onClick={onChange}>
            {t.setup.changeSetup}
          </Button>
        ) : (
          <>
            <Button
              variant="solid"
              className="px-2"
              disabled={!placedAll || kingOnHill}
              onClick={() => {
                remember(mode.value, you, board)
                onReady(board)
              }}
            >
              {t.setup.ready}
            </Button>
            <Button variant="ghost" className="px-2" onClick={() => set([...start])}>
              {t.setup.reset}
            </Button>
            <Button variant="ghost" className="px-2" onClick={() => set([...ownStandard(you)])}>
              {t.setup.default}
            </Button>
            <Button variant="ghost" className="px-2" onClick={clear}>
              {t.setup.clear}
            </Button>
          </>
        )}
      </div>
      {kingOnHill && !ready && (
        <p className="text-sm text-danger" role="alert">
          {t.setup.kingOnHill}
        </p>
      )}
    </div>
  )
}
