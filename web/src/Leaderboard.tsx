import { useEffect, useState } from 'react'
import { Avatar } from './Avatar'
import { navigate } from './router'
import { Button, Spinner } from './ui'

type Leader = { rank: number; username: string; rating: number; games: number }

const SHORT = 10

/** The Top 100 by rating, with your own row pinned underneath when you're further down. */
export function Leaderboard({ me }: { me?: string }) {
  const [data, setData] = useState<{ top: Leader[]; me: Leader | null } | null>(null)
  const [error, setError] = useState('')
  const [all, setAll] = useState(false)

  useEffect(() => {
    let live = true
    fetch('/api/leaderboard')
      .then(async (res) => {
        const body = await res.json()
        if (!live) return
        if (res.ok) setData({ top: body.top ?? [], me: body.me })
        else setError(body.error || "Couldn't load the leaderboard.")
      })
      .catch(() => live && setError("Couldn't reach the server."))
    return () => {
      live = false
    }
    // Refetch when you sign in or out, so your own row follows.
  }, [me])

  const rows = data ? (all ? data.top : data.top.slice(0, SHORT)) : []
  const pinned = data?.me && !rows.some((r) => r.username === data.me!.username) ? data.me : null

  return (
    <section className="flex flex-col gap-3.5">
      <div className="flex items-baseline justify-between gap-3">
        <h2 className="text-[22px] font-semibold tracking-tight">Top 100</h2>
        <span className="text-[13px] text-accent">Updated after every rated game</span>
      </div>
      {error ? (
        <p className="text-accent">{error}</p>
      ) : !data ? (
        <Spinner label="Loading the leaderboard" />
      ) : data.top.length === 0 ? (
        <p className="text-accent">
          Nobody's rated yet. Play a game with a clock against someone signed in, and you'll both show up here.
        </p>
      ) : (
        <>
          <table className="lb">
            <thead>
              <tr>
                <th>#</th>
                <th>Player</th>
                <th className="r max-sm:hidden">Games</th>
                <th className="r">Rating</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <Row key={r.username} leader={r} you={r.username === me} />
              ))}
              {pinned && (
                <>
                  <tr className="gap" aria-hidden>
                    <td colSpan={4}>···</td>
                  </tr>
                  <Row leader={pinned} you />
                </>
              )}
            </tbody>
          </table>
          {data.top.length > SHORT && (
            <Button className="h-9 self-start px-4 text-sm" onClick={() => setAll(!all)}>
              {all ? 'Show less' : `Show all ${data.top.length}`}
            </Button>
          )}
        </>
      )}
    </section>
  )
}

function Row({ leader, you }: { leader: Leader; you: boolean }) {
  const href = `/u/${encodeURIComponent(leader.username)}`
  return (
    // The whole row opens the profile. The name is the real link, for keyboards and screen readers.
    <tr
      data-podium={leader.rank <= 3 || undefined}
      data-you={you || undefined}
      onClick={(e) => {
        if (!(e.target as Element).closest('a')) navigate(href)
      }}
      className="cursor-pointer"
    >
      <td className="rank">{leader.rank}</td>
      <td>
        <a
          href={href}
          onClick={(e) => {
            e.preventDefault()
            navigate(href)
          }}
          className="flex min-w-0 items-center gap-2.5"
        >
          <Avatar size={30} />
          <span className="flex min-w-0 flex-col leading-tight">
            <span className="truncate font-semibold">{leader.username}</span>
            {you && <span className="text-[13px] text-accent">You</span>}
          </span>
        </a>
      </td>
      <td className="r font-mono text-accent tabular-nums max-sm:hidden">{leader.games}</td>
      <td className="r rt">{leader.rating}</td>
    </tr>
  )
}
