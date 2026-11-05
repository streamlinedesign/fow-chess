import { useEffect, useState } from 'react'
import { ownCell, pieceImg4, squareName, type Seat } from './four'
import { useT } from './i18n'
import type { Look } from './pieces'
import { Button } from './ui'

/** Every player's pieces. Must match army in server/four/setup.go. */
const ARMY = 'KQRRBBNNPPPPPPPP'
const KINDS = ['K', 'Q', 'R', 'B', 'N', 'P']
/** The usual arrangement: back row, pawns, an empty third row. Must match four.StandardSetup. */
const STANDARD = 'RNBQKBNR' + 'PPPPPPPP' + '........'

type Layout = 'backRow' | 'battleship'

const storageKey = (layout: Layout) => `four-setup:${layout}`

/** Whether board places exactly the army, in the rows layout allows. Mirrors four.CheckSetup. */
function valid(layout: Layout, board: string) {
  if (board.length !== 24) return false
  const left: Record<string, number> = {}
  for (const k of ARMY) left[k] = (left[k] ?? 0) + 1
  for (let i = 0; i < 24; i++) {
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
function remembered(layout: Layout) {
  try {
    const saved = localStorage.getItem(storageKey(layout))
    if (saved && valid(layout, saved)) return saved
  } catch {}
  return STANDARD
}

type Pick = { tray: string } | { square: number } | null

/**
 * Where a four-player arranges their pieces before a Back row or Battleship
 * game: their three rows, back row at the bottom, as they'll sit at the
 * table. Tap a piece in the tray and then a square, or a placed piece and
 * then another square to move or swap it; tap the tray to take a picked
 * piece off.
 */
export function FourSetup({
  seat,
  layout,
  look,
  mine,
  onReady,
  onChange,
}: {
  seat: Seat
  layout: Layout
  look?: Look
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
  const ready = !!mine
  const [start] = useState(() => mine || remembered(layout))
  const [board, setBoard] = useState(start)
  const [pick, setPick] = useState<Pick>(null)
  useEffect(() => {
    if (mine) setBoard(mine)
  }, [mine])

  const rows = layout === 'backRow' ? 1 : 3
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

  const tapSquare = (i: number) => {
    if (ready) return
    const b = [...board]
    if (pick && 'tray' in pick) {
      if (!allowed(i)) return
      const here = b[i]
      b[i] = pick.tray
      const remaining = left[pick.tray] - 1 + (here === pick.tray ? 1 : 0)
      return set(b, remaining > 0 ? pick : null)
    }
    if (pick && 'square' in pick) {
      if (pick.square === i || !allowed(i)) return setPick(null)
      ;[b[i], b[pick.square]] = [b[pick.square], b[i]]
      return set(b)
    }
    if (board[i] !== '.' && allowed(i)) setPick({ square: i })
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

  const clear = () => set([...board].map((c) => (fixed(c) ? c : '.')))

  const squares = []
  for (let r = 2; r >= 0; r--) {
    for (let f = 0; f < 8; f++) {
      const i = r * 8 + f
      const c = board[i]
      // Named and coloured as the square it stands for on the board.
      const [x, y] = ownCell(seat, f, r)
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
          disabled={ready}
        >
          {c !== '.' && <img className="piece" src={pieceImg4(seat + c, look)} alt="" draggable={false} />}
        </button>,
      )
    }
  }

  return (
    <div className="flex w-full max-w-[560px] flex-col gap-3" style={{ '--board': 'min(var(--board4), 560px)' } as React.CSSProperties}>
      <div>
        <h2 className="text-xl font-bold tracking-tight">{t.four.arrange}</h2>
        <p className="text-sm text-accent">{ready ? t.four.arranged : t.four.arrangeHint}</p>
      </div>
      <div
        className="board setup-board grid w-full grid-cols-8 overflow-hidden rounded-md shadow-[0_0_0_2px_var(--color-fg)]"
        onContextMenu={(e) => e.preventDefault()}
      >
        {squares}
      </div>

      <div
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
                  localStorage.setItem(storageKey(layout), board)
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
