import { useEffect, useState } from 'react'
import { Board } from './Board'
import { Captured } from './Captured'
import type { Color } from './useMatch'

type Side = { board: string; last: { from: string; to: string } | null; check?: string }
type Frame = { turn: Color; w: Side; b: Side; captured: Record<Color, string> }

const STEP_MS = 1300
const END_HOLD_MS = 5000
const noMoves: string[] = []
const noop = () => {}

/**
 * Replays Morphy's Opera Game as each player saw it under fog, side by side.
 * The frames come from the real server rules (server/showcase_test.go), so the
 * replay never drifts from what players actually see.
 */
export function Showcase() {
  const [frames, setFrames] = useState<Frame[] | null>(null)
  const [i, setI] = useState(0)

  useEffect(() => {
    import('./showcase.json').then((m) => setFrames(m.default as Frame[]))
  }, [])

  useEffect(() => {
    if (!frames) return
    const atEnd = i === frames.length - 1
    const t = setTimeout(() => setI(atEnd ? 0 : i + 1), atEnd ? END_HOLD_MS : i === 0 ? STEP_MS * 2 : STEP_MS)
    return () => clearTimeout(t)
  }, [frames, i])

  const f = frames?.[i]
  const over = !!frames && i === frames.length - 1

  return (
    <figure className="flex w-full flex-col gap-3" aria-label="A fog of war game seen from both sides">
      <div className="showcase grid grid-cols-2 gap-3 sm:gap-5">
        {(['w', 'b'] as const).map((c) => (
          <div key={c} className="flex min-w-0 flex-col gap-2">
            <div className="flex h-6 items-center gap-2 text-sm">
              <span
                className={`size-3 shrink-0 rounded-full shadow-[0_0_0_1.5px_var(--color-fg)] ${c === 'w' ? 'bg-[#fff]' : 'bg-[#111]'}`}
                aria-hidden
              />
              <span className="font-semibold">{c === 'w' ? 'White' : 'Black'} sees</span>
              {f && !over && f.turn === c && <span className="turn-tag max-sm:hidden">to move</span>}
              <Captured pieces={f?.captured[c] ?? ''} className="ml-auto text-[clamp(12px,2.6vw,18px)]" />
            </div>
            <div className="pointer-events-none" aria-hidden>
              <Board
                board={f ? f[c].board : '?'.repeat(64)}
                orientation={c}
                legal={noMoves}
                lastMove={f?.[c].last}
                check={f?.[c].check}
                onMove={noop}
              />
            </div>
          </div>
        ))}
      </div>
      <figcaption className="text-sm text-accent">
        {over
          ? 'Checkmate. The fog lifts when the game ends.'
          : "Same game, two boards. Each side sees only what its pieces see, and never the opponent's moves."}
      </figcaption>
    </figure>
  )
}
