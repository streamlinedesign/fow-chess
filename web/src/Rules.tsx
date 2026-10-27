import type { ReactNode } from 'react'

function Card({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="flex flex-col gap-3 rounded-2xl p-5 shadow-[inset_0_0_0_2px_var(--color-fg)] sm:p-6">
      <h3 className="text-xl font-bold tracking-tight">{title}</h3>
      <div className="flex flex-col gap-2.5 text-[15px] leading-relaxed text-accent [&_strong]:font-semibold [&_strong]:text-fg">
        {children}
      </div>
    </section>
  )
}

/** How the game works, for the landing page. Keep in step with server/engine and server/view.go. */
export function Rules() {
  return (
    <section id="rules" className="mx-auto flex w-full max-w-[1180px] flex-col gap-6 px-4 pt-6 pb-16 sm:px-8">
      <h2 className="text-[clamp(32px,6vw,48px)] leading-none font-bold tracking-tight">How it plays</h2>

      <div className="grid gap-4 md:grid-cols-2">
        <Card title="Starting a match">
          <p>
            Pick the rules and press <strong>New match</strong>. You get a 6-character match ID and a link to share. The
            creator plays White, and whoever opens the link next plays Black.
          </p>
          <p>
            Refreshing or losing signal puts you back in your seat. Anyone else who opens the link{' '}
            <strong>watches</strong> instead of playing.
          </p>
          <p>
            You can offer a draw or resign at any time. After the game, a <strong>rematch</strong> starts a new one with
            colors swapped.
          </p>
        </Card>

        <Card title="Fog of war">
          <p>
            You only see the squares your pieces can see. Each piece sees <strong>its own square</strong> and{' '}
            <strong>every square it could move to</strong>, including captures. Everything else is covered.
          </p>
          <p>
            A <strong>pawn</strong> always sees the square straight ahead, even when it's blocked. It sees its diagonals
            only when there's something to capture there.
          </p>
          <p>
            You get <strong>no hints</strong> about your opponent: no highlight of their last move, a <code>?</code> for
            their moves in the move list, and no check mark on your own moves. When the game ends, the fog lifts for
            everyone.
          </p>
        </Card>

        <Card title="Win by: Eat the King">
          <p>
            There is <strong>no check</strong>. Kings may walk into danger, stay in danger, and castle out of, through or
            into an attack.
          </p>
          <p>
            You win by <strong>capturing the enemy king</strong>. With fog on, this is the natural way to play, since
            neither side can always see the threats.
          </p>
        </Card>

        <Card title="Win by: Checkmate">
          <p>
            Normal chess rules. You can't make a move that leaves your king in check, and you win by{' '}
            <strong>checkmate</strong>.
          </p>
          <p>
            With fog on, you're told when <strong>you</strong> are in check, because it limits your moves. You're never
            told when you've put your opponent in check.
          </p>
        </Card>

        <Card title="Draws">
          <p>
            A game is drawn by <strong>stalemate</strong>, <strong>threefold repetition</strong>, the{' '}
            <strong>fifty-move rule</strong>, <strong>insufficient material</strong>, or when both players agree.
            Repetition and the fifty-move rule end the game automatically.
          </p>
        </Card>

        <Card title="Everything else">
          <p>
            All the usual moves work: <strong>castling</strong>, <strong>en passant</strong> and{' '}
            <strong>promotion</strong>, where you pick the piece. The server checks every move, so the board only offers
            moves that are legal.
          </p>
          <p>Captured pieces show next to each player, and the bar of the side to move lights up.</p>
        </Card>
      </div>
    </section>
  )
}
