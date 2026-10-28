import { memo, useRef, useState } from 'react'
import { pieceAt, pieceImg, squareIndex } from './position'
import { HILL } from './settings'
import type { Color } from './useMatch'

const FILES = 'abcdefgh'

type Props = {
  /** 64 characters from a1 to h8, as sent by the server (see position.ts). */
  board: string
  orientation: Color
  /** UCI moves the viewer may play right now; empty when it isn't their turn. */
  legal: string[]
  lastMove?: { from: string; to: string } | null
  check?: string
  /** Your king's square while a piece you can see attacks it. */
  danger?: string
  /** Squares a seen piece attacks, marked when the king is picked up. */
  unsafe?: string[]
  /** King of the Hill: mark the four centre squares. */
  hill?: boolean
  onMove: (from: string, to: string) => void
}

type Drag = {
  from: string
  el: HTMLElement
  x: number
  y: number
  moved: boolean
  wasSelected: boolean
}

function squareAt(col: number, row: number, orientation: Color) {
  return orientation === 'w' ? FILES[col] + (8 - row) : FILES[7 - col] + (row + 1)
}

function targetsFrom(legal: string[], from: string) {
  const out = new Set<string>()
  for (const m of legal) if (m.startsWith(from)) out.add(m.slice(2, 4))
  return out
}

export const Board = memo(function Board({
  board,
  orientation,
  legal,
  lastMove,
  check,
  danger,
  unsafe,
  hill,
  onMove,
}: Props) {
  const boardRef = useRef<HTMLDivElement>(null)
  const drag = useRef<Drag | null>(null)
  const [selected, setSelected] = useState<string | null>(null)

  const canMoveFrom = (sq: string) => legal.some((m) => m.startsWith(sq))

  // Drop the selection once that piece can no longer move (it moved, or the turn passed).
  const sel = selected && canMoveFrom(selected) ? selected : null
  const targets = sel ? targetsFrom(legal, sel) : new Set<string>()
  // Only the king has unsafe squares, so they show when the king is the piece picked up.
  const kingPicked = !!sel && pieceAt(board, sel)?.toLowerCase() === 'k'

  const squareFromPoint = (x: number, y: number) => {
    const rect = boardRef.current!.getBoundingClientRect()
    const col = Math.floor(((x - rect.left) / rect.width) * 8)
    const row = Math.floor(((y - rect.top) / rect.height) * 8)
    if (col < 0 || col > 7 || row < 0 || row > 7) return null
    return squareAt(col, row, orientation)
  }

  const onPointerDown = (e: React.PointerEvent) => {
    if (e.button !== 0) return
    const sq = squareFromPoint(e.clientX, e.clientY)
    if (!sq) return

    if (sel && sel !== sq && targets.has(sq)) {
      setSelected(null)
      onMove(sel, sq)
      return
    }

    if (!canMoveFrom(sq)) {
      setSelected(null)
      return
    }

    const el = boardRef.current!.querySelector<HTMLElement>(`[data-square="${sq}"] .piece`)
    if (!el) return
    e.currentTarget.setPointerCapture(e.pointerId)
    drag.current = { from: sq, el, x: e.clientX, y: e.clientY, moved: false, wasSelected: sel === sq }
    setSelected(sq)
  }

  const onPointerMove = (e: React.PointerEvent) => {
    const d = drag.current
    if (!d) return
    if (!d.moved && Math.hypot(e.clientX - d.x, e.clientY - d.y) < 4) return
    d.moved = true
    // Center the piece under the pointer. Direct style writes keep dragging off React's render path.
    const r = d.el.parentElement!.getBoundingClientRect()
    const dx = e.clientX - (r.left + r.width / 2)
    const dy = e.clientY - (r.top + r.height / 2)
    d.el.dataset.dragging = 'true'
    d.el.style.transform = `translate(${dx}px, ${dy}px) scale(1.15)`
  }

  const endDrag = (e: React.PointerEvent, cancelled = false) => {
    const d = drag.current
    if (!d) return
    drag.current = null
    d.el.style.transform = ''
    delete d.el.dataset.dragging

    if (cancelled) return
    if (d.moved) {
      const to = squareFromPoint(e.clientX, e.clientY)
      if (to && to !== d.from && targetsFrom(legal, d.from).has(to)) {
        setSelected(null)
        onMove(d.from, to)
      }
      // Dropped off a target: leave it selected so a tap can finish the move.
    } else if (d.wasSelected) {
      setSelected(null)
    }
  }

  const squares = []
  for (let row = 0; row < 8; row++) {
    for (let col = 0; col < 8; col++) {
      const sq = squareAt(col, row, orientation)
      const i = squareIndex(sq)
      const piece = pieceAt(board, sq)
      const fog = board[i] === '?'
      const target = targets.has(sq)
      // A pawn moving diagonally onto an empty square is capturing en passant.
      const capture = target && (piece !== null || (pieceAt(board, sel!)?.toLowerCase() === 'p' && sel![0] !== sq[0]))
      squares.push(
        <div
          key={sq}
          className="sq"
          data-square={sq}
          data-light={((i & 7) + (i >> 3)) % 2 === 1}
          // Under fog the square your piece left is often out of sight; a highlight there would read as visible.
          data-last={!fog && (lastMove?.from === sq || lastMove?.to === sq)}
          data-selected={sel === sq}
          data-check={check === sq}
          data-danger={danger === sq}
          data-unsafe={kingPicked && target && !!unsafe?.includes(sq)}
          data-target={target ? (capture ? 'capture' : 'move') : undefined}
          data-fog={fog}
          data-hill={hill && HILL.includes(sq)}
        >
          {col === 0 && <span className="coord top-[4%] left-[5%]">{sq[1]}</span>}
          {row === 7 && <span className="coord right-[5%] bottom-[4%]">{sq[0]}</span>}
          {piece && <img className="piece" src={pieceImg(piece)} alt={piece} draggable={false} />}
        </div>,
      )
    }
  }

  return (
    <div
      ref={boardRef}
      className="board grid aspect-square w-full grid-cols-8 grid-rows-8 overflow-hidden rounded-md shadow-[0_0_0_2px_var(--color-fg)]"
      onPointerDown={onPointerDown}
      onPointerMove={onPointerMove}
      onPointerUp={(e) => endDrag(e)}
      onPointerCancel={(e) => endDrag(e, true)}
      onContextMenu={(e) => e.preventDefault()}
    >
      {squares}
    </div>
  )
})
