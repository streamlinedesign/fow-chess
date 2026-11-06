import type { ReactNode } from 'react'
import { rich, useT } from './i18n'
import { pieceImg, type Look } from './position'
import { pieceImg4, SEATS } from './four'
import { HILL, OUTPOSTS } from './settings'

/** A FEN board as 64 characters from a1 to h8, the way the server sends boards. */
function fenBoard(fen: string) {
  const ranks = fen.split('/').map((r) => r.replace(/\d/g, (n) => '.'.repeat(Number(n))))
  return ranks.reverse().join('')
}

const FILES = 'abcdefgh'

/** A small still board for the rules: no coordinates, no input. */
function MiniBoard({
  fen,
  seen,
  last,
  dots,
  hill,
  danger,
  outposts,
  look,
}: {
  fen: string
  seen?: string[]
  last?: string[]
  dots?: string[]
  hill?: boolean
  danger?: string
  /** Ring the centre vision squares, White's solid and Black's dashed. */
  outposts?: boolean
  look?: Look
}) {
  const board = fenBoard(fen)
  const squares = []
  for (let row = 0; row < 8; row++) {
    for (let col = 0; col < 8; col++) {
      const sq = FILES[col] + (8 - row)
      const piece = board[(7 - row) * 8 + col]
      const fog = seen && !seen.includes(sq)
      const light = (col + row) % 2 === 0
      squares.push(
        <div
          key={sq}
          className="sq relative"
          data-light={light}
          data-fog={fog || undefined}
          data-last={(!fog && last?.includes(sq)) || undefined}
          data-target={dots?.includes(sq) ? (piece === '.' ? 'move' : 'capture') : undefined}
          data-hill={hill && HILL.includes(sq)}
          data-danger={danger === sq}
          data-outpost={
            !outposts ? undefined : OUTPOSTS.w.includes(sq) ? 'yours' : OUTPOSTS.b.includes(sq) ? 'theirs' : undefined
          }
        >
          {piece !== '.' && !fog && <img className="piece" src={pieceImg(piece, look)} alt="" draggable={false} />}
        </div>,
      )
    }
  }
  return (
    <div className="board grid aspect-square w-32 grid-cols-8 grid-rows-8 overflow-hidden rounded-[3px]" aria-hidden>
      {squares}
    </div>
  )
}

type CardText = { kind: string; title: string; p: string[] }

/** One rule card: its art here, its words in the dictionary. */
function Card({ card, art }: { card: CardText; art: ReactNode }) {
  return (
    <article className="tick flex flex-col gap-3 p-5 sm:p-6">
      <div className="mb-1.5 flex h-32 items-center">{art}</div>
      <div>
        <h3 className="text-xl font-semibold tracking-tight">{card.title}</h3>
        <p className="text-[13px] text-accent">{card.kind}</p>
      </div>
      <div className="flex flex-col gap-3 text-[15px] leading-relaxed text-accent [&_strong]:font-semibold [&_strong]:text-fg">
        {card.p.map((p, i) => (
          <p key={i}>{rich(p)}</p>
        ))}
      </div>
    </article>
  )
}

const rows = (n: number) => [...Array(n * 8)].map((_, i) => FILES[i % 8] + (Math.floor(i / 8) + 1))

/** Every square a piece on sq reaches with these steps. */
const steps = (sq: string, d: [number, number][]) =>
  d
    .map(([x, y]) => [FILES.indexOf(sq[0]) + x, Number(sq[1]) + y] as const)
    .filter(([x, y]) => x >= 0 && x < 8 && y >= 1 && y <= 8)
    .map(([x, y]) => FILES[x] + y)
const KING: [number, number][] = [-1, 0, 1].flatMap((x) => [-1, 0, 1].map((y) => [x, y] as [number, number])).filter(([x, y]) => x || y)
const KNIGHT: [number, number][] = [
  [1, 2], [2, 1], [2, -1], [1, -2], [-1, -2], [-2, -1], [-2, 1], [-1, 2],
]

const Grid = ({ children }: { children: ReactNode }) => (
  <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">{children}</div>
)

/** The game itself: Eat the King. Keep in step with server/engine and normalize in server/view.go. */
export function CoreRules() {
  const t = useT()
  const c = t.rules.cards
  return (
    <Grid>
      <Card
        card={c.fog}
        art={<MiniBoard fen="8/8/8/8/3N4/8/8/8" seen={['d4', 'b3', 'b5', 'c2', 'c6', 'e2', 'e6', 'f3', 'f5']} />}
      />
      <Card
        card={c.eat}
        art={
          <div className="flex items-center gap-2.5" aria-hidden>
            <img src={pieceImg('Q')} alt="" className="size-[72px]" />
            <span className="text-3xl text-accent">→</span>
            <img src={pieceImg('k')} alt="" className="size-[72px] -rotate-90 opacity-55" />
          </div>
        }
      />
      {/* The knight on c3 attacks the king, and nothing on the board says so. */}
      <Card card={c.blind} art={<MiniBoard fen="8/8/8/8/8/2n5/8/4K3" seen={['e1', 'd1', 'd2', 'e2', 'f1', 'f2']} />} />
    </Grid>
  )
}

/** Where the pieces start: the usual position, or one you arrange. Still Eat the King. */
export function LayoutRules() {
  const c = useT().rules.cards
  return (
    <Grid>
      <Card card={c.backRow} art={<MiniBoard fen="8/8/8/8/8/8/PPPPPPPP/BNRKQRNB" seen={rows(2)} last={rows(1)} />} />
      <Card card={c.battleship} art={<MiniBoard fen="8/8/8/8/1P2N2P/P1R2Q1P/1PB1PPB1/P1RK1N2" seen={rows(4)} />} />
      <Card card={c.sniper} art={<MiniBoard fen="4k3/8/8/8/8/4R3/8/4K3" last={['e3', 'e8']} />} />
    </Grid>
  )
}

/** Starting, clocks, rating, draws. */
export function PlayingRules() {
  const t = useT()
  const c = t.rules.cards
  return (
    <Grid>
      <Card
        card={c.start}
        art={
          <div className="flex flex-col gap-1.5">
            <div className="flex gap-1">
              {'K7Q2XD'.split('').map((ch, i) => (
                <span
                  key={i}
                  className="grid h-11 w-8 place-items-center rounded-md border-[1.5px] border-fg font-mono text-xl"
                >
                  {ch}
                </span>
              ))}
            </div>
            <span className="font-mono text-xs text-accent">{t.rules.yourMatchCode}</span>
          </div>
        }
      />
      <Card
        card={c.clocks}
        art={
          <div className="flex flex-col gap-2">
            <span className="flex items-baseline gap-2 rounded-md bg-fg px-3.5 py-2 font-mono text-[34px] leading-none text-bg">
              5:00<span className="text-base opacity-70">+0</span>
            </span>
            <span className="font-mono text-[15px]">
              1500 → <strong className="text-fg">1512</strong>
            </span>
          </div>
        }
      />
      <Card card={c.draws} art={<span className="text-[60px] leading-none font-light tracking-tight">½ – ½</span>} />
    </Grid>
  )
}

/** The mods: one rule changed each. Keep in step with the mods page's switches. */
export function ModRules() {
  const c = useT().rules.modCards
  return (
    <Grid>
      <Card card={c.fullBoard} art={<MiniBoard fen="rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR" />} />
      <Card card={c.checkmate} art={<MiniBoard fen="4R1k1/5ppp/8/8/8/8/8/6K1" last={['e1', 'e8']} danger="g8" />} />
      <Card
        card={c.warning}
        art={<MiniBoard fen="8/8/8/8/8/2n5/8/4K3" seen={['e1', 'd1', 'd2', 'e2', 'f1', 'f2']} danger="e1" />}
      />
      <Card
        card={c.center}
        art={<MiniBoard fen="r3k3/1p4p1/8/3N4/8/8/8/4K3" outposts last={['d5']} />}
      />
      <Card card={c.hill} art={<MiniBoard fen="8/8/8/8/4K3/8/8/8" hill last={['e3', 'e4']} />} />
      <Card
        card={c.jump}
        art={<MiniBoard fen="8/8/8/8/3K4/8/8/8" look={{ jumpKing: true }} dots={[...steps('d4', KING), ...steps('d4', KNIGHT)]} />}
      />
      <Card
        card={c.jester}
        art={
          <MiniBoard
            fen="8/8/8/3n4/4P3/8/8/8"
            look={{ jesterPawns: true }}
            dots={steps('e4', KING)}
          />
        }
      />
      <Card
        card={c.sync}
        art={
          <div className="flex flex-col gap-1.5 font-mono text-[22px] leading-none" aria-hidden>
            <span className="rounded-md bg-fg px-3 py-2 text-bg">e2–e4</span>
            <span className="rounded-md border-[1.5px] border-fg px-3 py-2">e7–e5</span>
          </div>
        }
      />
      <Card
        card={c.four}
        art={
          <div className="grid grid-cols-2 gap-1" aria-hidden>
            {SEATS.map((s) => (
              <img key={s} src={pieceImg4(s + 'K')} alt="" className="size-14" />
            ))}
          </div>
        }
      />
    </Grid>
  )
}
