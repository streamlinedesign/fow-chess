import { useEffect, useRef, useState } from 'react'
import { geo, pieceImg4, squareName, type FourLook, type Seat } from './four'
import { useT } from './i18n'
import { Button } from './ui'

/** Every player's pieces. Must match army in server/four/setup.go. */
const ARMY = 'KQRRBBNNPPPPPPPP'
const KINDS = ['K', 'Q', 'R', 'B', 'N', 'P']
/** The usual arrangement in a player's own rows (three, or four on the extended board): back row, pawns, empty rows. Must match Shape.StandardSetup. */
const standard = (rows: number) => 'RNBQKBNR' + 'PPPPPPPP' + '.'.repeat(8 * (rows - 2))

type Layout = 'backRow' | 'battleship'

const storageKey = (layout: Layout, rows: number) => `four-setup:${layout}${rows === 4 ? ':extended' : ''}`

/** Whether board places exactly the army, in the rows layout allows. Mirrors four.CheckSetup. */
function valid(layout: Layout, rows: number, board: string) {
  if (board.length !== 8 * rows) return false
  const left: Record<string, number> = {}
  for (const k of ARMY) left[k] = (left[k] ?? 0) + 1
  for (let i = 0; i < board.length; i++) {
    const c = board[i]
    if (c === '.') continue
    if (!KINDS.includes(c)) return false
    const row = i >> 3
    if (layout === 'backRow' && (c === 'P' ? row !== 1 : row !== 0)) return false
    left[c]--
  }
  return KINDS.every((k) => left[k] === 0)
}

/** The arrangement used last time in this layout, or the usual one. */
function remembered(layout: Layout, rows: number) {
  try {
    const saved = localStorage.getItem(storageKey(layout, rows))
    if (saved && valid(layout, rows, saved)) return saved
  } catch {}
  return standard(rows)
}

type Pick = { tray: string } | { square: number } | null

/** A drag in progress: a piece from the tray, or one already on the board. As in SetupBoard. */
type Drag = {
  x: number
  y: number
  moved: boolean
  source: { tray: string } | { square: number; el: HTMLElement }
  ghost?: HTMLImageElement
}

const DRAG_START = 4

/**
 * Where a four-player arranges their pieces before a Back row or Battleship
 * game: their own rows (three, or four on the extended board), back row at the bottom, as they'll sit at the
 * table. Tap a piece in the tray and then a square, or a placed piece and
 * then another square to move or swap it; tap the tray to take a picked
 * piece off.
 */
export function FourSetup({
  seat,
  layout,
  board: shape,
  look,
  mine,
  onReady,
  onChange,
}: {
  seat: Seat
  layout: Layout
  /** The game's board setting: '' for the cross, 'extended'. */
  board?: string
  look?: FourLook
  /** What you sent, so a reload picks it up. */
  mine?: string
  onReady: (board: string) => void
  onChange: () => void
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
  const g = geo(shape)
  const STANDARD = standard(g.rows)
  const ready = !!mine
  const [start] = useState(() => mine || remembered(layout, g.rows))
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

  const rows = layout === 'backRow' ? 1 : g.rows
  const fixed = (c: string) => layout === 'backRow' && c === 'P'
  const inRows = (i: number) => i >> 3 < rows
  const allowed = (i: number) => inRows(i) && !fixed(board[i])

  const left: Record<string, number> = {}
  for (const k of ARMY) left[k] = (left[k] ?? 0) + 1
  for (const c of board) if (c !== '.') left[c]--
  const placedAll = KINDS.every((k) => left[k] === 0)

  const set = (b: string[], next: Pick = null) => {
    setBoard(b.join(''))
    setPick(next)
  }

  /** Puts a k from the tray on square i, sending whatever stood there back to the tray. */
  const placeFromTray = (k: string, i: number) => {
    if (!allowed(i) || left[k] <= 0) return
    const b = [...board]
    const here = b[i]
    b[i] = k
    // When you have more of the same piece, keep it picked so the next tap places another.
    const remaining = left[k] - 1 + (here === k ? 1 : 0)
    set(b, remaining > 0 ? { tray: k } : null)
  }

  /** Moves the piece on from to to, swapping with whatever is there. */
  const moveOnBoard = (from: number, to: number) => {
    if (from === to || !allowed(to)) return setPick(null)
    const b = [...board]
    ;[b[to], b[from]] = [b[from], b[to]]
    set(b)
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
    if (pick && 'tray' in pick) return placeFromTray(pick.tray, i)
    if (pick && 'square' in pick) {
      if (pick.square === i) return setPick(null)
      return moveOnBoard(pick.square, i)
    }
    if (board[i] !== '.' && allowed(i)) setPick({ square: i })
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

  // Dragging. Taps stay click events on the buttons; a pointer that moves becomes a drag.
  const squareFromPoint = (x: number, y: number) => {
    const rect = boardRef.current!.getBoundingClientRect()
    const col = Math.floor(((x - rect.left) / rect.width) * 8)
    const row = Math.floor(((y - rect.top) / rect.height) * g.rows)
    if (col < 0 || col > 7 || row < 0 || row >= g.rows) return null
    return (g.rows - 1 - row) * 8 + col
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
          d.ghost.src = pieceImg4(seat + d.source.tray, look)
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

  const clear = () => set([...board].map((c) => (fixed(c) ? c : '.')))

  const squares = []
  for (let r = g.rows - 1; r >= 0; r--) {
    for (let f = 0; f < 8; f++) {
      const i = r * 8 + f
      const c = board[i]
      // Named and coloured as the square it stands for on the board.
      const [x, y] = g.ownCell(seat, f, r)
      const name = squareName(x, y)
      squares.push(
        <button
          key={i}
          type="button"
          className="sq aspect-square"
          data-light={(x + y) % 2 === 1}
          data-zone={!ready && allowed(i)}
          data-selected={!!pick && 'square' in pick && pick.square === i}
          aria-label={c === '.' ? name : `${name}, ${kindNames[c]}`}
          onClick={() => tapSquare(i)}
          onPointerDown={(e) => {
            const el = e.currentTarget.querySelector<HTMLElement>('.piece')
            if (el && allowed(i)) startDrag(e, { square: i, el })
          }}
          disabled={ready}
        >
          {c !== '.' && <img className="piece" src={pieceImg4(seat + c, look)} alt="" draggable={false} />}
        </button>,
      )
    }
  }

  return (
    <div
      className="flex w-full max-w-[560px] flex-col gap-3"
      style={{ '--board': 'min(var(--board4), 560px)' } as React.CSSProperties}
    >
      <div>
        <h2 className="text-xl font-bold tracking-tight">{t.four.arrange}</h2>
        <p className="text-sm text-accent">{ready ? t.four.arranged : t.four.arrangeHint(g.rows)}</p>
      </div>
      <div
        ref={boardRef}
        className="board setup-board grid w-full grid-cols-8 overflow-hidden board-crisp"
        onContextMenu={(e) => e.preventDefault()}
      >
        {squares}
      </div>

      <div
        ref={trayRef}
        className="tray"
        role="group"
        aria-label={pick && 'square' in pick ? t.setup.tapToRemove : t.setup.yourPieces}
        data-dropping={!!pick && 'square' in pick}
        onClick={(e) => {
          if (e.target === e.currentTarget && pick && 'square' in pick) tapTray('')
        }}
      >
        {KINDS.filter((k) => !(layout === 'backRow' && k === 'P')).map((k) => (
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
            <img src={pieceImg4(seat + k, look)} alt="" draggable={false} />
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
              disabled={!placedAll}
              onClick={() => {
                try {
                  localStorage.setItem(storageKey(layout, g.rows), board)
                } catch {}
                onReady(board)
              }}
            >
              {t.setup.ready}
            </Button>
            <Button variant="ghost" className="px-2" onClick={() => set([...start])}>
              {t.setup.reset}
            </Button>
            <Button variant="ghost" className="px-2" onClick={() => set([...STANDARD])}>
              {t.setup.default}
            </Button>
            <Button variant="ghost" className="px-2" onClick={clear}>
              {t.setup.clear}
            </Button>
          </>
        )}
      </div>
    </div>
  )
}
