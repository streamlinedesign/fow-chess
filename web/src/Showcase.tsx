import { useEffect, useState } from 'react'
import { Board } from './Board'
import { Captured } from './Captured'
import { Spinner } from './ui'
import type { Color } from './useMatch'

type Side = { board: string; last: { from: string; to: string } | null; check?: string }
type Frame = { turn: Color; w: Side; b: Side; captured: Record<Color, string> }
type View = 'turns' | Color

// In Auto, each move first shows on the board of the side that played it, then
// the fog shifts to the side about to move. Pinned to one side, it's one step per move.
const MOVE_MS = 1300
const FOG_MS = 1300
const STEP_MS = 2400
const END_HOLD_MS = 6000
const noMoves: string[] = []
const noop = () => {}
const NAMES: Record<Color, string> = { w: 'White', b: 'Black' }

const VIEWS: [View, string][] = [
  ['turns', 'Auto'],
  ['w', 'White'],
  ['b', 'Black'],
]

/**
 * Replays Morphy's Opera Game as the players saw it under fog. By default each
 * move shows from the mover's side, then the fog shifts to whoever moves next.
 * The frames come from the real server rules (server/showcase_test.go), so the
 * replay never drifts from what players see. It renders as two blocks, the
 * board and its caption, so the page can line the board up with the settings.
 */
export function Showcase({
  boardClassName = '',
  captionClassName = '',
}: {
  boardClassName?: string
  captionClassName?: string
}) {
  const [frames, setFrames] = useState<Frame[] | null>(null)
  const [i, setI] = useState(0)
  const [phase, setPhase] = useState<'move' | 'fog'>('move')
  const [view, setView] = useState<View>('turns')

  useEffect(() => {
    import('./showcase.json').then((m) => setFrames(m.default as Frame[]))
  }, [])

  useEffect(() => {
    if (!frames) return
    const atEnd = i === frames.length - 1
    const next = () => {
      setI(atEnd ? 0 : i + 1)
      setPhase('move')
    }
    let t: ReturnType<typeof setTimeout>
    if (atEnd) t = setTimeout(next, END_HOLD_MS)
    else if (view !== 'turns') t = setTimeout(next, i === 0 ? STEP_MS * 1.5 : STEP_MS)
    else if (phase === 'move' && i > 0) t = setTimeout(() => setPhase('fog'), MOVE_MS)
    else t = setTimeout(next, i === 0 ? FOG_MS * 2 : FOG_MS)
    return () => clearTimeout(t)
  }, [frames, i, phase, view])

  const f = frames?.[i]
  const over = !!frames && i === frames.length - 1
  // Ply i was played by white when i is odd (the first frame is the start position).
  const mover: Color = i % 2 === 1 ? 'w' : 'b'
  const autoSide: Color = i > 0 && phase === 'move' ? mover : (f?.turn ?? 'w')
  const side: Color = view === 'turns' ? autoSide : view
  const moveNo = Math.ceil(i / 2)

  return (
    <>
      <div
        className={`pointer-events-none relative ${boardClassName}`}
        role="img"
        aria-label={`A fog of war game replayed from each side. ${over ? 'Fog lifted' : `${NAMES[side]}'s view`}, move ${moveNo}.`}
      >
        {!frames && (
          <div className="absolute inset-0 z-20 grid place-items-center text-[#fffabd]">
            <Spinner label="Loading replay" />
          </div>
        )}
        <Board
          board={f ? f[side].board : '?'.repeat(64)}
          orientation="w"
          legal={noMoves}
          lastMove={f?.[side].last}
          check={f?.[side].check}
          onMove={noop}
        />
      </div>

      <div className={`flex flex-col gap-2.5 ${captionClassName}`}>
        <div className="flex items-center justify-between gap-3">
          <div className="flex min-w-0 items-center gap-2 text-[15px] whitespace-nowrap">
            <span
              className={`size-3.5 shrink-0 rounded-full shadow-[0_0_0_1.5px_var(--color-fg)] transition-colors ${side === 'w' ? 'bg-[#fff]' : 'bg-[#111]'}`}
              aria-hidden
            />
            <span className="font-semibold">{over ? 'Fog lifted' : `${NAMES[side]}'s view`}</span>
            {moveNo > 0 && (
              <span className="font-mono text-sm text-accent tabular-nums max-[359px]:hidden">move {moveNo}</span>
            )}
          </div>
          <div role="radiogroup" aria-label="Whose view" className="flex shrink-0 gap-0.5 rounded-md bg-muted p-[3px]">
            {VIEWS.map(([v, label]) => (
              <button
                key={v}
                role="radio"
                aria-checked={view === v}
                onClick={() => setView(v)}
                className="rounded px-2.5 py-1 text-xs font-semibold whitespace-nowrap transition-colors aria-checked:bg-fg aria-checked:text-bg"
              >
                {label}
              </button>
            ))}
          </div>
        </div>
        <div className="flex h-6 items-center justify-between gap-3 text-[clamp(14px,4vw,18px)]">
          <Captured pieces={f?.captured.w ?? ''} />
          <Captured pieces={f?.captured.b ?? ''} className="ml-auto" />
        </div>
        <p className="text-sm text-accent">
          <span className="font-semibold text-fg">The Opera Game</span>, Morphy vs. the Duke of Brunswick and Count
          Isouard, Paris 1858, replayed with fog of war.{' '}
          {over ? 'At checkmate the fog lifts for both sides.' : 'Each side sees only what its own pieces can reach.'}
        </p>
      </div>
    </>
  )
}
