import type { ReactNode } from 'react'
import { rich, useT, type Dict } from './i18n'
import { pieceImg } from './position'
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
}: {
  fen: string
  seen?: string[]
  last?: string[]
  dots?: string[]
  hill?: boolean
  danger?: string
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
          data-target={dots?.includes(sq) ? 'move' : undefined}
          data-hill={hill && HILL.includes(sq)}
          data-danger={danger === sq}
        >
          {piece !== '.' && !fog && <img className="piece" src={pieceImg(piece)} alt="" draggable={false} />}
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
        <p className="text-xs font-semibold tracking-wider text-accent uppercase">{card.kind}</p>
        <h3 className="text-xl font-semibold tracking-tight">{card.title}</h3>
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

/** How the game works. Keep in step with server/engine and server/view.go. */
export function Rules() {
  const t = useT()
  return (
    <section id="rules" className="flex flex-col gap-6">
      <div className="border-b border-fg pb-4">
        <h1 className="text-[clamp(30px,5vw,44px)] leading-none font-semibold tracking-tight">{t.rules.heading}</h1>
        <p className="mt-2 text-accent">{t.rules.intro}</p>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        <Card
          id="fog"
          art={<MiniBoard fen="8/8/8/8/3N4/8/8/8" seen={['d4', 'b3', 'b5', 'c2', 'c6', 'e2', 'e6', 'f3', 'f5']} />}
        />

        <Card
          id="eat"
          art={
            <div className="flex items-center gap-2.5" aria-hidden>
              <img src={pieceImg('Q')} alt="" className="size-[72px]" />
              <span className="text-3xl text-accent">→</span>
              <img src={pieceImg('k')} alt="" className="size-[72px] -rotate-90 opacity-55" />
            </div>
          }
        />

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

        <Card id="backRow" art={<MiniBoard fen="8/8/8/8/8/8/PPPPPPPP/BNRKQRNB" seen={rows(2)} last={rows(1)} />} />

        <Card id="battleship" art={<MiniBoard fen="8/8/8/8/1P2N2P/P1R2Q1P/1PB1PPB1/P1RK1N2" seen={rows(4)} />} />

        <Card id="sniper" art={<MiniBoard fen="4k3/8/8/8/8/4R3/8/4K3" last={['e3', 'e8']} />} />

        <Card
          id="clocks"
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

        <Card id="draws" art={<span className="text-[60px] leading-none font-light tracking-tight">½ – ½</span>} />
      </div>
    </section>
  )
}
