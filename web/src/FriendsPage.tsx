import * as Menu from '@radix-ui/react-dropdown-menu'
import { useEffect, useState } from 'react'
import { useMe } from './auth'
import { Who } from './Avatar'
import { Icon } from './icons'
import { useLive } from './live'
import { navigate } from './router'
import { api, openOffer, presence, type FriendState, type UserCard } from './social'
import { Button, Dialog, Spinner } from './ui'

type Found = UserCard & { friend: FriendState }

function go(e: React.MouseEvent, href: string) {
  e.preventDefault()
  navigate(href)
}

/** Search for players, your friends with ways to play or talk, and the requests you've sent. */
export function FriendsPage() {
  const me = useMe()
  const [data, setData] = useState<{ friends: UserCard[]; sent: UserCard[] } | null>(null)
  const [error, setError] = useState('')
  const [removing, setRemoving] = useState<string | null>(null)

  const load = () =>
    api<{ friends: UserCard[]; sent: UserCard[] }>('GET', '/api/friends')
      .then(setData)
      .catch((e) => setError(e.message))

  useEffect(() => {
    if (me) load()
  }, [me?.username])
  useLive((e) => {
    if (e.type === 'friends' || e.type === 'presence') load()
  })

  if (me === null) return <SignInFirst what="see your friends" />

  const online = data?.friends.filter((f) => f.online).length ?? 0

  return (
    <main className="mx-auto w-full max-w-[1184px] flex-1 px-4 pt-6 pb-16 sm:px-8">
      <div className="page-head">
        <div>
          <h1>Friends{data && <span className="count">{data.friends.length}</span>}</h1>
          <p>Find players by username to add a friend or offer them a match.</p>
        </div>
      </div>

      <div className="flex flex-col gap-10">
        <Search onChange={load} />

        <section>
          <div className="mb-3.5 flex items-baseline justify-between gap-3">
            <h2 className="text-[22px] font-semibold tracking-tight">Your friends</h2>
            {online > 0 && <span className="text-[13px] text-accent">{online} online</span>}
          </div>
          {error ? (
            <p className="text-danger">{error}</p>
          ) : !data ? (
            <Spinner label="Loading" />
          ) : data.friends.length === 0 ? (
            <p className="text-accent">No friends yet. Search for someone above, or add them from their profile.</p>
          ) : (
            <div className="f-list">
              {data.friends.map((f) => (
                <div key={f.username} className="f-row">
                  <a href={`/u/${f.username}`} onClick={(e) => go(e, `/u/${f.username}`)} className="min-w-0">
                    <Who name={f.username} sub={f.rating} size={44} online={f.online} />
                  </a>
                  <div className="f-acts">
                    <span className="hide-sm mr-2 text-[13px] text-accent">
                      {f.online && !f.playing ? '' : presence(f)}
                    </span>
                    {f.playing && (
                      <Button
                        className="squash h-9 px-3.5 text-sm"
                        onClick={() => navigate(`/m/${f.playing}`)}
                        aria-label={`Watch ${f.username}`}
                      >
                        <Icon name="eye" />
                        <span className="txt">Watch</span>
                      </Button>
                    )}
                    <Button className="squash h-9 px-3.5 text-sm" onClick={() => openOffer(f.username)}>
                      <Icon name="swords" />
                      <span className="txt">Offer a match</span>
                    </Button>
                    <Button
                      variant="ghost"
                      icon
                      className="size-9"
                      aria-label={`Message ${f.username}`}
                      onClick={() => navigate(`/messages/${f.username}`)}
                    >
                      <Icon name="chat" />
                    </Button>
                    <Menu.Root modal={false}>
                      <Menu.Trigger asChild>
                        <Button variant="ghost" icon className="size-9" aria-label={`More for ${f.username}`}>
                          <Icon name="dots" />
                        </Button>
                      </Menu.Trigger>
                      <Menu.Portal>
                        <Menu.Content align="end" sideOffset={6} className="menu">
                          <Menu.Item className="menu-item" onSelect={() => navigate(`/u/${f.username}`)}>
                            <Icon name="user" />
                            Profile
                          </Menu.Item>
                          <Menu.Item className="menu-item text-danger" onSelect={() => setRemoving(f.username)}>
                            <Icon name="out" />
                            Remove friend
                          </Menu.Item>
                        </Menu.Content>
                      </Menu.Portal>
                    </Menu.Root>
                  </div>
                </div>
              ))}
            </div>
          )}
        </section>

        {data && data.sent.length > 0 && (
          <section>
            <h2 className="mb-3.5 text-[22px] font-semibold tracking-tight">Sent requests</h2>
            <div className="f-list">
              {data.sent.map((f) => (
                <div key={f.username} className="f-row">
                  <a href={`/u/${f.username}`} onClick={(e) => go(e, `/u/${f.username}`)} className="min-w-0">
                    <Who name={f.username} sub={f.rating} size={44} />
                  </a>
                  <div className="f-acts">
                    <Button
                      variant="ghost"
                      className="h-9 px-3.5 text-sm"
                      onClick={() => api('DELETE', `/api/friends/${f.username}`).then(load)}
                    >
                      Cancel
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          </section>
        )}
        <p className="-mt-6 text-[13px] text-accent">Requests sent to you show up in the bell.</p>
      </div>

      <Dialog
        open={!!removing}
        onOpenChange={(o) => !o && setRemoving(null)}
        title={`Remove ${removing}?`}
        description="You won't be friends any more, and you can't message each other. Your chat history stays."
      >
        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={() => setRemoving(null)}>
            Keep
          </Button>
          <Button
            variant="solid"
            onClick={async () => {
              await api('DELETE', `/api/friends/${removing}`).catch(() => {})
              setRemoving(null)
              load()
            }}
          >
            Remove
          </Button>
        </div>
      </Dialog>
    </main>
  )
}

function Search({ onChange }: { onChange: () => void }) {
  const [q, setQ] = useState('')
  const [results, setResults] = useState<Found[] | null>(null)
  const [error, setError] = useState('')

  const search = (query: string) =>
    api<Found[]>('GET', `/api/users/search?q=${encodeURIComponent(query)}`)
      .then(setResults)
      .catch(() => {})

  useEffect(() => {
    const query = q.trim()
    if (!query) {
      setResults(null)
      return
    }
    const t = setTimeout(() => search(query), 200)
    return () => clearTimeout(t)
  }, [q])
  // Requests answered elsewhere change the buttons here.
  useLive((e) => {
    if (e.type === 'friends' && q.trim()) search(q.trim())
  })

  const add = async (name: string) => {
    setError('')
    try {
      await api('POST', `/api/friends/${name}`)
    } catch (e) {
      setError((e as Error).message)
    }
    search(q.trim())
    onChange()
  }

  return (
    <section>
      <label className="search block">
        <Icon name="search" size={18} />
        <input
          className="input"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder="Search players"
          aria-label="Search players"
          autoComplete="off"
          spellCheck={false}
        />
      </label>
      {error && <p className="mt-2 text-sm text-danger">{error}</p>}
      {results &&
        (results.length === 0 ? (
          <p className="mt-3 text-accent">No players match that name.</p>
        ) : (
          <div className="results">
            {results.map((r) => (
              <div key={r.username} className="res-row">
                <a href={`/u/${r.username}`} onClick={(e) => go(e, `/u/${r.username}`)} className="min-w-0">
                  <Who name={r.username} sub={r.rating} size={32} online={r.online} />
                </a>
                <div className="f-acts">
                  {r.friend === 'friends' ? (
                    <span className="inline-flex items-center gap-1.5 text-[13px] text-accent">
                      <Icon name="check" size={14} />
                      Friends
                    </span>
                  ) : r.friend === 'sent' ? (
                    <span className="text-[13px] text-accent">Request sent</span>
                  ) : r.friend === 'received' ? (
                    <Button variant="solid" className="h-8 px-3 text-[13px]" onClick={() => add(r.username)}>
                      Accept request
                    </Button>
                  ) : (
                    <Button className="h-8 px-3 text-[13px]" onClick={() => add(r.username)}>
                      <Icon name="plus" />
                      Add friend
                    </Button>
                  )}
                  <Button
                    variant="ghost"
                    icon
                    className="size-8"
                    aria-label={`Offer ${r.username} a match`}
                    onClick={() => openOffer(r.username)}
                  >
                    <Icon name="swords" />
                  </Button>
                </div>
              </div>
            ))}
          </div>
        ))}
    </section>
  )
}

/** For pages that need an account. */
export function SignInFirst({ what }: { what: string }) {
  const next = `/login?next=${encodeURIComponent(location.pathname)}`
  return (
    <main className="mx-auto flex w-full max-w-[1184px] flex-1 flex-col items-start gap-5 px-4 pt-10 sm:px-8">
      <h1 className="text-[clamp(32px,6vw,44px)] leading-none font-semibold tracking-tight">Log in to {what}</h1>
      <p className="text-accent">Friends, messages and match offers are for signed-in players.</p>
      <a
        href={next}
        onClick={(e) => go(e, next)}
        className="inline-flex h-11 items-center rounded-md bg-fg px-5 font-semibold text-bg hover:opacity-85"
      >
        Log in
      </a>
    </main>
  )
}
