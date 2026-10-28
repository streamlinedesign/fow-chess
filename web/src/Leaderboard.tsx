import { useEffect, useState } from 'react'
import { Avatar } from './Avatar'
import { errorText, useT } from './i18n'
import { navigate } from './router'
import { Button, Spinner } from './ui'

type Leader = { rank: number; username: string; rating: number; games: number }

const SHORT = 10

/** The Top 100 by rating, with your own row pinned underneath when you're further down. */
export function Leaderboard({ me }: { me?: string }) {
  const t = useT()
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
        else setError(errorText(body))
      })
      .catch(() => live && setError(t.common.cantReach))
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
        <h2 className="text-[22px] font-semibold tracking-tight">{t.leaderboard.title}</h2>
        <span className="text-[13px] text-accent">{t.leaderboard.updated}</span>
      </div>
      {error ? (
        <p className="text-accent">{error}</p>
      ) : !data ? (
        <Spinner label={t.leaderboard.loading} />
      ) : data.top.length === 0 ? (
        <p className="text-accent">{t.leaderboard.nobodyRated}</p>
      ) : (
        <>
          <table className="lb">
            <thead>
              <tr>
                <th>#</th>
                <th>{t.leaderboard.player}</th>
                <th className="r max-sm:hidden">{t.leaderboard.games}</th>
                <th className="r">{t.leaderboard.rating}</th>
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
              {all ? t.leaderboard.showLess : t.leaderboard.showAll(data.top.length)}
            </Button>
          )}
        </>
      )}
    </section>
  )
}

function Row({ leader, you }: { leader: Leader; you: boolean }) {
  const t = useT()
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
            {you && <span className="text-[13px] text-accent">{t.common.you}</span>}
          </span>
        </a>
      </td>
      <td className="r font-mono text-accent tabular-nums max-sm:hidden">{leader.games}</td>
      <td className="r rt">{leader.rating}</td>
    </tr>
  )
}
