import type { ReactNode } from 'react'
import { rich, useT, type Dict } from './i18n'
import { pieceImg, type Look } from './position'
import { SEAT_COLORS } from './four'
import { HILL } from './settings'

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
  look,
}: {
  fen: string
  seen?: string[]
  last?: string[]
  dots?: string[]
  hill?: boolean
  danger?: string
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

type CardKey = keyof Dict['rules']['cards']

/** One rule card: its art here, its words in the dictionary. */
function Card({ id, art, children }: { id: CardKey; art: ReactNode; children?: ReactNode }) {
  const t = useT()
  const card = t.rules.cards[id]
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
        {children}
      </div>
    </article>
  )
}

const rows = (n: number) => [...Array(n * 8)].map((_, i) => FILES[i % 8] + (Math.floor(i / 8) + 1))

/** Eat the King, the original game. Keep in step with server/engine and server/view.go. */
export function EatRules() {
  const t = useT()
  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
      <Card
        id="fog"
        art={<MiniBoard fen="8/8/8/8/3N4/8/8/8" seen={['d4', 'b3', 'b5', 'c2', 'c6', 'e2', 'e6', 'f3', 'f5']} />}
      />

      <Card id="eat" art={<KingTaken />} />

      <Card
        id="danger"
        art={<MiniBoard fen="8/8/8/8/8/2n5/8/4K3" seen={['e1', 'd1', 'd2', 'e2', 'f1', 'f2', 'c3']} danger="e1" />}
      />

      <Card
        id="start"
        art={
          <div className="flex flex-col gap-1.5">
            <div className="flex gap-1">
              {'K7Q2XD'.split('').map((c, i) => (
                <span
                  key={i}
                  className="grid h-11 w-8 place-items-center rounded-md border-[1.5px] border-fg font-mono text-xl"
                >
                  {c}
                </span>
              ))}
            </div>
            <span className="font-mono text-xs text-accent">{t.rules.yourMatchCode}</span>
          </div>
        }
      />

      <Card id="clocks" art={<ClockArt time="5:00" inc="+0" />}>
        <span className="font-mono text-[15px]">
          1500 → <strong className="text-fg">1512</strong>
        </span>
      </Card>

      <Card id="draws" art={<span className="text-[60px] leading-none font-light tracking-tight">½ – ½</span>} />
    </div>
  )
}

/** War Mode, the ranked four-player game. Keep in step with server/four and server/four_queue.go. */
export function WarRules() {
  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
      <Card id="warBoard" art={<FourSeats />} />
      <Card id="warSetup" art={<MiniBoard fen="8/8/8/8/1P2N2P/P1R2Q1P/1PB1PPB1/P1RK1N2" seen={rows(4)} />} />
      <Card
        id="warSync"
        art={
          <div className="grid grid-cols-2 gap-1.5" aria-hidden>
            {Object.values(SEAT_COLORS).map((c) => (
              <span key={c} className="grid size-12 place-items-center rounded-md text-xl text-white" style={{ background: c }}>
                ✓
              </span>
            ))}
          </div>
        }
      />
      <Card
        id="warJester"
        art={
          <MiniBoard
            fen="8/8/8/3n4/4P3/8/8/8"
            look={{ jesterPawns: true }}
            dots={['d3', 'e3', 'f3', 'd4', 'f4', 'd5', 'e5', 'f5']}
          />
        }
      />
      <Card id="warWin" art={<KingTaken />} />
      <Card id="warRating" art={<ClockArt time="30:00" inc="+10" />} />
    </div>
  )
}

function KingTaken() {
  return (
    <div className="flex items-center gap-2.5" aria-hidden>
      <img src={pieceImg('Q')} alt="" className="size-[72px]" />
      <span className="text-3xl text-accent">→</span>
      <img src={pieceImg('k')} alt="" className="size-[72px] -rotate-90 opacity-55" />
    </div>
  )
}

function ClockArt({ time, inc }: { time: string; inc: string }) {
  return (
    <span className="flex items-baseline gap-2 rounded-md bg-fg px-3.5 py-2 font-mono text-[34px] leading-none text-bg">
      {time}
      <span className="text-base opacity-70">{inc}</span>
    </span>
  )
}

/** The four seats round the extended board, with its open corners. */
function FourSeats() {
  const side = (c: string, cls: string) => <span className={`absolute rounded-sm ${cls}`} style={{ background: c }} />
  return (
    <div className="relative size-32 rounded-[3px] border-[1.5px] border-fg" aria-hidden>
      {side(SEAT_COLORS.r, 'inset-x-7 bottom-1 h-6')}
      {side(SEAT_COLORS.b, 'inset-y-7 left-1 w-6')}
      {side(SEAT_COLORS.g, 'inset-x-7 top-1 h-6')}
      {side(SEAT_COLORS.v, 'inset-y-7 right-1 w-6')}
    </div>
  )
}
