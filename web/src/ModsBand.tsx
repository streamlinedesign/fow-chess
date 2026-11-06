import { useEffect, useMemo, useState } from 'react'
import { useT } from './i18n'
import { go } from './ProfilePage'
import { KINDS, setImg } from './sets'

/**
 * The Mods band on the home page: one full-width night band with pieces
 * flying through it, an extended four-player board playing itself in the fog,
 * every mod's name running past, and a cell per mod that opens the Mods page
 * with that switch on (see modFromQuery in ModsPage). It is dark in both
 * themes, since the fog only reads on dark.
 */
export function ModsBand() {
  const t = useT()
  const b = t.modsBand
  const cells: [string, string][] = [
    ['extended', setImg('cream', 'R')],
    ['sync', setImg('ink', 'N')],
    ['jump', setImg('cream', 'X')],
    ['jester', setImg('ink', 'J')],
    ['battleship', setImg('ink', 'Q')],
    ['hill', setImg('cream', 'K')],
    ['nofog', ''],
    ['mate', setImg('ink', 'K')],
  ]
  return (
    <section className="mods-band" aria-labelledby="mods-band-title">
      <Sky />
      <div className="mods-band-top">
        <div>
          <h2 className="mods-band-title" id="mods-band-title">
            {b.title.map((w, i) => (
              <span key={w} className={i === 1 ? 'o' : undefined}>
                {w}
              </span>
            ))}
          </h2>
          <p className="mods-band-lede">{b.lede}</p>
          <p className="mods-band-sub">{b.sub}</p>
          <div className="mods-band-ctas">
            <a className="mods-btn solid" href="/mods" onClick={(e) => go(e, '/mods')}>
              {b.open}
            </a>
            <a className="mods-btn" href="/mods?mod=extended" onClick={(e) => go(e, '/mods?mod=extended')}>
              {b.tryExtended}
            </a>
          </div>
        </div>
        <FogDemo />
      </div>

      <div className="mods-ticker" aria-hidden>
        <div className="mods-ticker-run">
          {[0, 1].flatMap((r) =>
            b.ticker.flatMap((n) => [
              <span key={`${r}${n}`}>{n}</span>,
              <span key={`${r}${n}.`} className="dot">
                ·
              </span>,
            ]),
          )}
        </div>
      </div>

      <nav className="mods-cells" aria-label={t.common.mods}>
        {cells.map(([id, img]) => {
          const [name, hint] = b.cells[id]
          const href = `/mods?mod=${id}`
          return (
            <a key={id} className="mods-cell" href={href} onClick={(e) => go(e, href)}>
              {img ? (
                <img className="ic" src={img} alt="" />
              ) : (
                <span className="ic sw" aria-hidden>
                  <i />
                  <i />
                  <i />
                  <i />
                </span>
              )}
              <h3>{name}</h3>
              <p>{hint}</p>
            </a>
          )
        })}
      </nav>
      <div className="mods-band-foot">
        <span>{b.footLeft}</span>
        <span>{b.footRight}</span>
      </div>
    </section>
  )
}

/** Black and white pieces drifting up through the band, near ones bigger, faster and sharper. */
function Sky() {
  const pieces = useMemo(() => {
    let seed = 7
    const rnd = () => (seed = (seed * 16807) % 2147483647) / 2147483647
    return Array.from({ length: 30 }, (_, i) => {
      const depth = rnd()
      return {
        src: setImg(i % 2 ? 'ink' : 'cream', KINDS[Math.floor(rnd() * KINDS.length)]),
        // Phones show every other one.
        far: i % 2 === 1,
        style: {
          '--x': `${(rnd() * 110 - 5).toFixed(1)}%`,
          '--s': `${Math.round(28 + depth * 110)}px`,
          '--o': (0.25 + depth * 0.6).toFixed(2),
          '--b': `${((1 - depth) * 3).toFixed(1)}px`,
          '--d': `${Math.round(26 - depth * 14)}s`,
          '--delay': `-${(rnd() * 26).toFixed(1)}s`,
          '--dx': `${Math.round(rnd() * 360 - 120)}px`,
          '--r0': `${Math.round(rnd() * 360)}deg`,
          '--r1': `${Math.round(rnd() * 720 - 360)}deg`,
          '--y0': `${Math.round(rnd() * 100)}%`,
        } as React.CSSProperties,
      }
    })
  }, [])
  return (
    <div className="mods-sky" aria-hidden>
      {pieces.map((p, i) => (
        <img key={i} src={p.src} alt="" className={p.far ? 'far' : undefined} style={p.style} />
      ))}
    </div>
  )
}

const S = 16
type P = { seat: string; kind: string; x: number; y: number; dead?: boolean; id: number }
type Move = [number, number, number, number]
const SEAT_SET: Record<string, string> = { r: 'red', b: 'blue', g: 'green', v: 'violet' }
const BACK = ['R', 'N', 'B', 'Q', 'K', 'B', 'N', 'R']
const corner = (x: number, y: number) => (x < 4 || x > 11) && (y < 4 || y > 11)
const onBoard = (x: number, y: number) => x >= 0 && y >= 0 && x < S && y < S

/** Red's moves in the loop, ending with the knight eating a blue pawn from the open corner. */
const SCRIPT: Move[] = [
  [7, 14, 7, 12],
  [5, 15, 4, 13],
  [6, 15, 9, 12],
  [4, 13, 2, 12],
  [7, 15, 7, 13],
  [2, 12, 1, 10],
]

function apply(ps: P[], [x, y, nx, ny]: Move): P[] {
  return ps.map((p) =>
    p.dead
      ? p
      : p.x === nx && p.y === ny
        ? { ...p, dead: true }
        : p.x === x && p.y === y
          ? { ...p, x: nx, y: ny }
          : p,
  )
}

/** The extended board at the start, with a few moves the others have made, hidden in the fog. */
function startPosition(): P[] {
  let id = 0
  let ps: P[] = []
  for (let i = 0; i < 8; i++) {
    ps.push(
      { seat: 'r', kind: BACK[i], x: 4 + i, y: 15, id: id++ },
      { seat: 'r', kind: 'P', x: 4 + i, y: 14, id: id++ },
      { seat: 'g', kind: BACK[7 - i], x: 4 + i, y: 0, id: id++ },
      { seat: 'g', kind: 'P', x: 4 + i, y: 1, id: id++ },
      { seat: 'b', kind: BACK[7 - i], x: 0, y: 4 + i, id: id++ },
      { seat: 'b', kind: 'P', x: 1, y: 4 + i, id: id++ },
      { seat: 'v', kind: BACK[i], x: 15, y: 4 + i, id: id++ },
      { seat: 'v', kind: 'P', x: 14, y: 4 + i, id: id++ },
    )
  }
  for (const m of [
    [8, 1, 8, 3],
    [1, 7, 3, 7],
    [14, 9, 12, 9],
    [0, 9, 2, 10],
  ] as Move[])
    ps = apply(ps, m)
  return ps
}

/** The squares Red's pieces see: where they stand, and every square they could move to or take on. */
function redSight(ps: P[]) {
  const seen = new Set<number>()
  const at = (x: number, y: number) => ps.some((p) => !p.dead && p.x === x && p.y === y)
  const add = (x: number, y: number) => onBoard(x, y) && seen.add(y * S + x)
  const ray = (p: P, dirs: number[][]) =>
    dirs.forEach(([dx, dy]) => {
      for (let x = p.x + dx, y = p.y + dy; onBoard(x, y); x += dx, y += dy) {
        add(x, y)
        if (at(x, y)) break
      }
    })
  const R = [
    [1, 0],
    [-1, 0],
    [0, 1],
    [0, -1],
  ]
  const B = [
    [1, 1],
    [1, -1],
    [-1, 1],
    [-1, -1],
  ]
  for (const p of ps) {
    if (p.dead || p.seat !== 'r') continue
    add(p.x, p.y)
    if (p.kind === 'P') {
      add(p.x, p.y - 1)
      if (p.y === 14 && !at(p.x, p.y - 1)) add(p.x, p.y - 2)
      add(p.x - 1, p.y - 1)
      add(p.x + 1, p.y - 1)
    } else if (p.kind === 'N') {
      for (const [dx, dy] of [
        [1, 2],
        [2, 1],
        [-1, 2],
        [-2, 1],
        [1, -2],
        [2, -1],
        [-1, -2],
        [-2, -1],
      ])
        add(p.x + dx, p.y + dy)
    } else if (p.kind === 'K') {
      for (const [dx, dy] of [...R, ...B]) add(p.x + dx, p.y + dy)
    } else ray(p, p.kind === 'R' ? R : p.kind === 'B' ? B : [...R, ...B])
  }
  return seen
}

const STATES = (() => {
  const out = [startPosition()]
  for (const m of SCRIPT) out.push(apply(out[out.length - 1], m))
  return out
})()

/** The extended board in the fog, from Red's seat, playing a short loop. Tilted until hovered; still with reduced motion. */
function FogDemo() {
  const t = useT()
  const [step, setStep] = useState(0)
  useEffect(() => {
    if (window.matchMedia?.('(prefers-reduced-motion: reduce)').matches) return
    const id = setInterval(() => setStep((s) => (s + 1) % STATES.length), 1700)
    return () => clearInterval(id)
  }, [])
  const ps = STATES[step]
  const seen = useMemo(() => redSight(ps), [ps])
  const last = step > 0 ? SCRIPT[step - 1] : null
  const squares = []
  for (let y = 0; y < S; y++)
    for (let x = 0; x < S; x++) {
      const i = y * S + x
      squares.push(
        <div
          key={i}
          className="sq"
          data-light={(x + y) % 2 === 0 || undefined}
          data-neutral={corner(x, y) || undefined}
          data-fog={!seen.has(i) || undefined}
          data-mark={last && last[0] === x && last[1] === y ? 'from' : last && last[2] === x && last[3] === y ? 'to' : undefined}
        />,
      )
    }
  return (
    <div className="mods-stage">
      <div className="mods-demo" role="img" aria-label={t.modsBand.boardLabel}>
        {squares}
        {ps.map((p) => (
          <div
            key={p.id}
            className="pc"
            data-hidden={(!p.dead && !seen.has(p.y * S + p.x)) || undefined}
            data-eaten={p.dead || undefined}
            style={{ '--x': p.x, '--y': p.y } as React.CSSProperties}
          >
            <img src={setImg(SEAT_SET[p.seat], p.kind)} alt="" />
          </div>
        ))}
      </div>
      <p className="mods-demo-cap">
        <span>{t.modsBand.boardCap}</span>
        <span aria-live="off">{step === 0 ? t.modsBand.start : t.modsBand.steps[step - 1]}</span>
      </p>
    </div>
  )
}
