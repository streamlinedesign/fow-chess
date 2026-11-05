import { memo, useMemo, useRef, useState } from 'react'
import { CENTRE, cellFor, onBoard, parseMove, pieceImg4, SIZE, squareName, type Seat } from './four'
import type { Look } from './pieces'

type Props = {
  /** 196 squares from the bottom left, as the server sends them. */
  board: string[]
  /** Whose seat is drawn at the bottom. */
  seat: Seat
  /** Moves you may play now; empty when it isn't your turn. */
  legal: string[]
  lastMove?: { from: string; to: string } | null
  /** Your king's square while something attacks it. */
  danger?: string
  /** King of the Hill: ring the four centre squares. */
  hill?: boolean
  /** Centre vision: ring the four centre squares, lit while you hold one. */
  center?: { live?: boolean }
  look?: Look
  onMove?: (from: string, to: string) => void
  /** A still board, for previews. */
  still?: boolean
}

type Drag = { from: string; el: HTMLElement; x: number; y: number; moved: boolean; wasSelected: boolean }

/** The cross-shaped board, turned so the viewer's pieces are at the bottom. */
export const FourBoard = memo(function FourBoard({
  board,
  seat,
  legal,
  lastMove,
  danger,
  hill,
  center,
  look,
  onMove,
  still,
}: Props) {
  const boardRef = useRef<HTMLDivElement>(null)
  const drag = useRef<Drag | null>(null)
  const [selected, setSelected] = useState<string | null>(null)

  // The square drawn in each cell, for this seat.
  const grid = useMemo(() => {
    const g: (string | null)[][] = Array.from({ length: SIZE }, () => Array(SIZE).fill(null))
    for (let y = 0; y < SIZE; y++)
      for (let x = 0; x < SIZE; x++) {
        if (!onBoard(x, y)) continue
        const { col, row } = cellFor(x, y, seat)
        g[row][col] = squareName(x, y)
      }
    return g
  }, [seat])

  const moves = useMemo(() => legal.map(parseMove).filter((m) => m !== null), [legal])
  const canMoveFrom = (sq: string) => moves.some((m) => m.from === sq)
  const sel = selected && canMoveFrom(selected) ? selected : null
  const targets = new Set(sel ? moves.filter((m) => m.from === sel).map((m) => m.to) : [])
  const targetsOf = (from: string) => new Set(moves.filter((m) => m.from === from).map((m) => m.to))

  const squareFromPoint = (px: number, py: number) => {
    const rect = boardRef.current!.getBoundingClientRect()
    const col = Math.floor(((px - rect.left) / rect.width) * SIZE)
    const row = Math.floor(((py - rect.top) / rect.height) * SIZE)
    return grid[row]?.[col] ?? null
  }

  const at = (sq: string) => {
    const [x, y] = [sq.charCodeAt(0) - 97, Number(sq.slice(1)) - 1]
    return board[y * SIZE + x]
  }

  const onPointerDown = (e: React.PointerEvent) => {
    if (still || e.button !== 0) return
    const sq = squareFromPoint(e.clientX, e.clientY)
    if (!sq) return setSelected(null)
    if (sel && sel !== sq && targets.has(sq)) {
      setSelected(null)
      onMove?.(sel, sq)
      return
    }
    if (!canMoveFrom(sq)) return setSelected(null)
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
    const r = d.el.parentElement!.getBoundingClientRect()
    d.el.dataset.dragging = 'true'
    d.el.style.transform = `translate(${e.clientX - (r.left + r.width / 2)}px, ${e.clientY - (r.top + r.height / 2)}px) scale(1.2)`
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
      if (to && to !== d.from && targetsOf(d.from).has(to)) {
        setSelected(null)
        onMove?.(d.from, to)
      }
    } else if (d.wasSelected) {
      setSelected(null)
    }
  }

  const cells = []
  for (let row = 0; row < SIZE; row++) {
    for (let col = 0; col < SIZE; col++) {
      const sq = grid[row][col]
      if (!sq) {
        cells.push(<div key={`${row}-${col}`} className="off" aria-hidden />)
        continue
      }
      const v = at(sq)
      const fog = v === '?'
      const piece = v && v.length === 2 ? v : null
      const target = targets.has(sq)
      // A pawn moving onto an empty square sideways is taking en passant.
      const capture = target && (!!piece || (at(sel!)?.[1] === 'P' && isDiagonal(sel!, sq)))
      const [x, y] = [sq.charCodeAt(0) - 97, Number(sq.slice(1)) - 1]
      cells.push(
        <div
          key={sq}
          className="sq"
          data-square={sq}
          data-light={(x + y) % 2 === 1}
          data-last={!fog && (lastMove?.from === sq || lastMove?.to === sq)}
          data-selected={sel === sq}
          data-danger={danger === sq}
          data-target={target ? (capture ? 'capture' : 'move') : undefined}
          data-fog={fog}
          data-hill={hill && CENTRE.includes(sq)}
          data-outpost={center && CENTRE.includes(sq) ? (center.live ? 'live' : 'both') : undefined}
        >
          {piece && <img className="piece" src={pieceImg4(piece, look)} alt={piece} draggable={false} />}
        </div>,
      )
    }
  }

  return (
    <div
      ref={boardRef}
      className="board4 grid aspect-square w-full"
      data-still={still}
      onPointerDown={onPointerDown}
      onPointerMove={onPointerMove}
      onPointerUp={(e) => endDrag(e)}
      onPointerCancel={(e) => endDrag(e, true)}
      onContextMenu={(e) => e.preventDefault()}
    >
      {cells}
    </div>
  )
})

function isDiagonal(a: string, b: string) {
  const dx = Math.abs(a.charCodeAt(0) - b.charCodeAt(0))
  const dy = Math.abs(Number(a.slice(1)) - Number(b.slice(1)))
  return dx === 1 && dy === 1
}
