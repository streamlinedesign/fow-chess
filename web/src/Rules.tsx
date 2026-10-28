import type { ReactNode } from 'react'
import { pieceImg } from './position'

/** A FEN board as 64 characters from a1 to h8, the way the server sends boards. */
function fenBoard(fen: string) {
  const ranks = fen.split('/').map((r) => r.replace(/\d/g, (n) => '.'.repeat(Number(n))))
  return ranks.reverse().join('')
}

const FILES = 'abcdefgh'

/** A small still board for the rules: no coordinates, no input. */
function MiniBoard({ fen, seen, last }: { fen: string; seen?: string[]; last?: string[] }) {
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

function Card({ art, title, children }: { art: ReactNode; title: string; children: ReactNode }) {
  return (
    <article className="tick flex flex-col gap-3 p-5 sm:p-6">
      <div className="mb-1.5 flex h-32 items-center">{art}</div>
      <h3 className="text-xl font-semibold tracking-tight">{title}</h3>
      <div className="flex flex-col gap-3 text-[15px] leading-relaxed text-accent [&_strong]:font-semibold [&_strong]:text-fg">
        {children}
      </div>
    </article>
  )
}

/** How the game works, for the landing page. Keep in step with server/engine and server/view.go. */
export function Rules() {
  return (
    <section id="rules" className="flex flex-col gap-6">
      <div className="border-b border-fg pb-4">
        <h2 className="text-[clamp(30px,5vw,44px)] leading-none font-semibold tracking-tight">How it plays</h2>
        <p className="mt-2 text-accent">
          Normal chess pieces and moves. What changes is what you can see, and how you win.
        </p>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        <Card
          title="Starting a match"
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
              <span className="font-mono text-xs text-accent">your match code</span>
            </div>
          }
        >
          <p>
            Pick the rules and a time control, then press <strong>New match</strong>. You get a 6-character{' '}
            <strong>code</strong> and a link to share. You play White, and whoever opens the link next plays Black.
          </p>
          <p>
            Refreshing or losing signal puts you back in your seat. You can offer a draw or resign at any time, and a{' '}
            <strong>rematch</strong> swaps colours.
          </p>
        </Card>

        <Card
          title="Fog of war"
          art={<MiniBoard fen="8/8/8/8/3N4/8/8/8" seen={['d4', 'b3', 'b5', 'c2', 'c6', 'e2', 'e6', 'f3', 'f5']} />}
        >
          <p>
            You only see what your pieces see: <strong>their own squares</strong> and{' '}
            <strong>every square they could move to</strong>, captures included. The rest is covered.
          </p>
          <p>
            A <strong>pawn</strong> always sees the square ahead, even when blocked, and its diagonals only when there's
            something to take. You get no hints about the other side. The fog lifts when the game ends.
          </p>
        </Card>

        <Card
          title="Win by Eat the King"
          art={
            <div className="flex items-center gap-2.5" aria-hidden>
              <img src={pieceImg('Q')} alt="" className="size-[72px]" />
              <span className="text-3xl text-accent">→</span>
              <img src={pieceImg('k')} alt="" className="size-[72px] -rotate-90 opacity-55" />
            </div>
          }
        >
          <p>
            There is <strong>no check</strong>. Kings may walk into danger, stay there, and castle through an attack.
            You win by <strong>capturing the king</strong>.
          </p>
          <p>With fog on, it's the natural way to play, since nobody can always see the threats.</p>
        </Card>

        <Card title="Win by Checkmate" art={<MiniBoard fen="R5k1/5ppp/8/8/8/8/5PPP/6K1" last={['a1', 'a8']} />}>
          <p>
            Normal chess rules. You can't leave your king in check, and you win by <strong>checkmate</strong>.
          </p>
          <p>
            With fog on, you're told when <strong>you</strong> are in check, because it limits your moves. You're never
            told when you've put your opponent in check.
          </p>
        </Card>

        <Card
          title="Clocks and rating"
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
        >
          <p>
            Clocks run on the server, so closing the tab doesn't stop them. Run out and you lose, unless your opponent
            can't win with what's left. <strong>Daily</strong> gives you 24 hours per move. <strong>No clock</strong>{' '}
            waits forever.
          </p>
          <p>
            Games between two signed-in players are <strong>rated</strong>, on one rating for every mode, starting at
            1500. Guest and No clock games aren't.
          </p>
        </Card>

        <Card
          title="Draws and the rest"
          art={<span className="text-[60px] leading-none font-light tracking-tight">½ – ½</span>}
        >
          <p>
            A game is drawn by <strong>stalemate</strong>, <strong>threefold repetition</strong>, the{' '}
            <strong>fifty-move rule</strong>, <strong>insufficient material</strong>, or when you both agree.
          </p>
          <p>
            <strong>Castling</strong>, <strong>en passant</strong> and <strong>promotion</strong> all work. The server
            checks every move, so the board only offers legal ones.
          </p>
        </Card>
      </div>
    </section>
  )
}
