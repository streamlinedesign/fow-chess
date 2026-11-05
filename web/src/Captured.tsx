import { useT } from './i18n'
import type { Look } from './position'
import { usePieces } from './prefs'

/** The pieces one side has taken, as FEN letters (most valuable first), drawn as overlapping icons. */
export function Captured({
  pieces,
  look,
  mine,
  className = '',
}: {
  pieces: string
  /** The side the viewer plays, drawn in their own set as they pick it (see prefs.ts). */
  mine?: 'w' | 'b'
  /** The mods that change how pieces are drawn. */
  look?: Look
  className?: string
}) {
  const t = useT()
  const img = usePieces(mine)
  if (!pieces) return null
  const groups: string[][] = []
  for (const p of pieces) {
    const g = groups[groups.length - 1]
    if (g?.[0] === p) g.push(p)
    else groups.push([p])
  }
  return (
    <span className={`flex items-center gap-1 ${className}`} aria-label={t.captured(pieces.length)}>
      {groups.map((g, i) => (
        <span key={i} className="flex">
          {g.map((p, j) => (
            <img
              key={j}
              src={img(p, look)}
              alt={p}
              draggable={false}
              className="captured-piece size-[1.15em] first:ml-0 -ml-[0.6em]"
            />
          ))}
        </span>
      ))}
    </span>
  )
}
