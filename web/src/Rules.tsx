import { useState, type ReactNode } from 'react'
import { pieceImg } from './position'

/** A FEN board as 64 characters from a1 to h8, the way the server sends boards. */
function fenBoard(fen: string) {
  const ranks = fen.split('/').map((r) => r.replace(/\d/g, (n) => '.'.repeat(Number(n))))
  return ranks.reverse().join('')
}

const FILES = 'abcdefgh'

/** A small still board for the rules: no coordinates, no input. */
function MiniBoard({ fen, seen, last, dots }: { fen: string; seen?: string[]; last?: string[]; dots?: string[] }) {
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

function Card({ art, kind, title, children }: { art: ReactNode; kind?: string; title: string; children: ReactNode }) {
  return (
    <article className="tick flex flex-col gap-3 p-5 sm:p-6">
      <div className="mb-1.5 flex h-32 items-center">{art}</div>
      <div>
        {kind && <p className="text-xs font-semibold tracking-wider text-accent uppercase">{kind}</p>}
        <h3 className="text-xl font-semibold tracking-tight">{title}</h3>
      </div>
      <div className="flex flex-col gap-3 text-[15px] leading-relaxed text-accent [&_strong]:font-semibold [&_strong]:text-fg">
        {children}
      </div>
    </article>
  )
}

const rows = (n: number) => [...Array(n * 8)].map((_, i) => FILES[i % 8] + (Math.floor(i / 8) + 1))

/** Squares a king on d4 reaches in Jump King: one step, or a knight's jump. */
const JUMPS = ['c3', 'c4', 'c5', 'd3', 'd5', 'e3', 'e4', 'e5', 'b3', 'b5', 'c2', 'c6', 'e2', 'e6', 'f3', 'f5']

/** The Jump King card's art. Tap the king enough times and he reaches the top. */
function JumpKingArt({ onTop }: { onTop: () => void }) {
  const [hops, setHops] = useState(0)
  const [hopping, setHopping] = useState(false)
  return (
    <div className="flex items-center gap-4">
      <MiniBoard fen="8/8/8/8/3K4/8/8/8" dots={JUMPS} />
      <button
        type="button"
        aria-label="Make the king jump"
        className="jump-king"
        data-hopping={hopping}
        onAnimationEnd={() => setHopping(false)}
        onClick={() => {
          if (hopping) return
          setHopping(true)
          setHops(hops + 1)
          if (hops + 1 === 7) onTop()
        }}
      >
        <img src={pieceImg('K')} alt="" draggable={false} className="size-[64px]" />
      </button>
    </div>
  )
}

/** How the game works, for the landing page. Keep in step with server/engine and server/view.go. */
export function Rules() {
  const [top, setTop] = useState(false)
  return (
    <section id="rules" className="flex flex-col gap-6">
      <div className="border-b border-fg pb-4">
        <h2 className="text-[clamp(30px,5vw,44px)] leading-none font-semibold tracking-tight">How it plays</h2>
        <p className="mt-2 text-accent">
          Normal chess pieces and moves. Modes change what you can see, how you win, where your pieces start, and how
          your king moves. Mix them however you like.
        </p>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        <Card
          kind="Getting started"
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
            Pick the modes and a time control, then press <strong>New match</strong>. You get a 6-character{' '}
            <strong>code</strong> and a link to share. You play White, and whoever opens the link next plays Black.
          </p>
          <p>
            Refreshing or losing signal puts you back in your seat. You can offer a draw or resign at any time, and a{' '}
            <strong>rematch</strong> swaps colours.
          </p>
        </Card>

        <Card
          kind="What you see"
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
          kind="How you win"
          title="Eat the King"
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

        <Card
          kind="How you win"
          title="Checkmate"
          art={<MiniBoard fen="R5k1/5ppp/8/8/8/8/5PPP/6K1" last={['a1', 'a8']} />}
        >
          <p>
            Normal chess rules. You can't leave your king in check, and you win by <strong>checkmate</strong>.
          </p>
          <p>
            With fog on, you're told when <strong>you</strong> are in check, because it limits your moves. You're never
            told when you've put your opponent in check.
          </p>
        </Card>

        <Card
          kind="Where pieces start"
          title="Standard"
          art={<MiniBoard fen="rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR" />}
        >
          <p>
            The usual starting position. Neither king can be in danger before the first move, so every game is a whole
            game.
          </p>
        </Card>

        <Card
          kind="Where pieces start"
          title="Back row"
          art={<MiniBoard fen="8/8/8/8/8/8/PPPPPPPP/BNRKQRNB" seen={rows(2)} last={rows(1)} />}
        >
          <p>
            Before the game, you each <strong>arrange your back row</strong> however you like. The pawns stay where they
            always are.
          </p>
          <p>
            You set up at the same time, so nobody gets to react. Castling works only for a king and rook left on their
            usual squares.
          </p>
        </Card>

        <Card
          kind="Where pieces start"
          title="Battleship"
          art={<MiniBoard fen="8/8/8/8/1P2N2P/P1R2Q1P/1PB1PPB1/P1RK1N2" seen={rows(4)} />}
        >
          <p>
            Place <strong>all sixteen pieces anywhere in your four rows</strong>, pawns included. Hide the king, build a
            wall, or aim a rook down an open file.
          </p>
          <p>
            With fog on you never see the other setup, so the first moves are a hunt. Without fog, each side's setup
            shows once that player is ready.
          </p>
        </Card>

        <Card kind="The king" title="Jump King" art={<JumpKingArt onTop={() => setTop(true)} />}>
          <p>
            The king still steps one square, and can also <strong>jump like a knight</strong>. That makes him harder to
            trap and quicker to reach the fight. It also means a king can be attacked by a king a knight's jump away.
          </p>
          <p className="italic">
            {top
              ? 'You made it to the top. Legend says there is a babe up here.'
              : 'Named after the game. Unlike that king, this one never falls all the way back down.'}
          </p>
        </Card>

        <Card
          kind="Any start you set up"
          title="Sniper Win"
          art={<MiniBoard fen="4k3/8/8/8/8/4R3/8/4K3" last={['e3', 'e8']} />}
        >
          <p>
            When you set up your own pieces, a king can start in danger. Nothing stops that: it's part of the game. If
            you win <strong>on your very first move</strong>, that's a <strong>Sniper Win</strong>.
          </p>
          <p>It still counts, but it moves both ratings only a quarter as far as a whole game.</p>
        </Card>

        <Card
          kind="Rated or unrated"
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
            Choose <strong>Rated</strong> or <strong>Unrated</strong> when you create a match. A rated game needs two
            signed-in players and a clock, and every mode shares one rating, starting at 1500. Unrated games are for
            trying things out.
          </p>
        </Card>

        <Card
          kind="Every mode"
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
