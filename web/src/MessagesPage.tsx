import { Fragment, useEffect, useLayoutEffect, useRef, useState } from 'react'
import { useMe } from './auth'
import { Avatar, Who } from './Avatar'
import { OfferTags, statusText } from './Bell'
import { SignInFirst } from './FriendsPage'
import { Icon } from './icons'
import { refreshCounts, useLive } from './live'
import { pieceImg } from './position'
import { plural, type GameSummary } from './ProfilePage'
import { navigate } from './router'
import { DropMenu, MenuItem } from './Sheet'
import { describeTime, NO_CLOCK } from './settings'
import { api, openOffer, presence, type FriendState, type Message, type Offer, type UserCard } from './social'
import { Button, Spinner } from './ui'

type Conversation = { with: UserCard; last: Message; unread: number }

type Chat = {
  messages: Message[]
  more: boolean
  with: UserCard
  friend: FriendState
  score: { wins: number; draws: number; losses: number }
  games: GameSummary[]
  deletion: { mine: boolean; theirs: boolean }
}

const timeFmt = new Intl.DateTimeFormat(undefined, { hour: '2-digit', minute: '2-digit' })
const weekdayFmt = new Intl.DateTimeFormat(undefined, { weekday: 'short' })
const dateFmt = new Intl.DateTimeFormat(undefined, { day: 'numeric', month: 'short' })
const dayFmt = new Intl.DateTimeFormat(undefined, { weekday: 'long', day: 'numeric', month: 'long' })

/** "14:02" today, "Mon" this week, "20 Oct" before that. */
function shortTime(ms: number) {
  const days = (Date.now() - ms) / 86400000
  if (new Date(ms).toDateString() === new Date().toDateString()) return timeFmt.format(ms)
  return days < 6 ? weekdayFmt.format(ms) : dateFmt.format(ms)
}

function dayLabel(ms: number) {
  const d = new Date(ms).toDateString()
  if (d === new Date().toDateString()) return 'Today'
  if (d === new Date(Date.now() - 86400000).toDateString()) return 'Yesterday'
  return dayFmt.format(ms)
}

/** Chats with friends: the list, and the open one beside it (or on its own on phones). */
export function MessagesPage({ name }: { name?: string }) {
  const me = useMe()
  const [convs, setConvs] = useState<Conversation[] | null>(null)

  const load = () =>
    api<Conversation[]>('GET', '/api/messages')
      .then(setConvs)
      .catch(() => {})

  useEffect(() => {
    if (me) load()
  }, [me?.username])
  useLive((e) => {
    if (e.type === 'message' || e.type === 'presence' || e.type === 'read') load()
  })

  if (me === null) return <SignInFirst what="message your friends" />
  if (!me) return null

  return (
    <main className="msgs-page mx-auto w-full max-w-[1184px] flex-1 px-4 pb-6 sm:px-8">
      <div className="msgs" data-open={name ? '' : undefined}>
        <aside className="convs">
          <div className="convs-head">
            <h1>Messages</h1>
            <Button variant="ghost" className="h-8 px-3 text-[13px]" onClick={() => navigate('/friends')}>
              <Icon name="plus" />
              New
            </Button>
          </div>
          <div className="conv-list">
            {!convs ? (
              <div className="p-4">
                <Spinner />
              </div>
            ) : convs.length === 0 ? (
              <p className="p-3 text-sm text-accent">
                No chats yet. Message a friend from their profile or your friends list.
              </p>
            ) : (
              convs.map((c) => (
                <button
                  key={c.with.username}
                  className="conv"
                  aria-current={c.with.username.toLowerCase() === name?.toLowerCase()}
                  data-unread={c.unread > 0 || undefined}
                  onClick={() => navigate(`/messages/${c.with.username}`)}
                >
                  <Avatar size={44} online={c.with.online} />
                  <div className="min-w-0">
                    <div className="font-semibold">{c.with.username}</div>
                    <div className="prev">
                      {c.last.from === me.username && 'You: '}
                      {c.last.offer ? 'Match offer' : c.last.text}
                    </div>
                  </div>
                  <span className="t">
                    {shortTime(c.last.at)}
                    {c.unread > 0 && <i />}
                  </span>
                </button>
              ))
            )}
          </div>
        </aside>
        <section className="chat">
          {name ? (
            <ChatView key={name.toLowerCase()} name={name} me={me.username} />
          ) : (
            <div className="empty-chat">Pick a chat, or start one from your friends list.</div>
          )}
        </section>
      </div>
    </main>
  )
}

function ChatView({ name, me }: { name: string; me: string }) {
  const [chat, setChat] = useState<Chat | null>(null)
  const [error, setError] = useState('')
  const [text, setText] = useState('')
  const [sending, setSending] = useState(false)
  const body = useRef<HTMLDivElement>(null)
  const stick = useRef(true) // keep the newest message in view

  const load = async () => {
    try {
      const c = await api<Chat>('GET', `/api/messages/${encodeURIComponent(name)}`)
      setChat((old) => ({ ...c, messages: mergeOlder(old, c) }))
      setError('')
    } catch (e) {
      setError((e as Error).message)
    }
  }

  useEffect(() => {
    load()
  }, [name])
  useLive((e) => {
    if ((e.type === 'message' || e.type === 'offer') && (!e.user || e.user.toLowerCase() === name.toLowerCase())) load()
    if (e.type === 'presence' && e.user?.toLowerCase() === name.toLowerCase()) load()
  })

  // Mark what's on screen as read.
  const lastFromThem = chat?.messages.filter((m) => m.from !== me).at(-1)?.id
  useEffect(() => {
    if (!lastFromThem) return
    api('POST', `/api/messages/${encodeURIComponent(name)}/read`, { upTo: lastFromThem })
      .then(refreshCounts)
      .catch(() => {})
  }, [lastFromThem])

  useLayoutEffect(() => {
    const el = body.current
    if (el && stick.current) el.scrollTop = el.scrollHeight
  }, [chat])

  const send = async (e: React.FormEvent) => {
    e.preventDefault()
    const t = text.trim()
    if (!t) return
    setSending(true)
    try {
      await api('POST', `/api/messages/${encodeURIComponent(name)}`, { text: t })
      setText('')
      stick.current = true
      await load()
    } catch (err) {
      setError((err as Error).message)
    }
    setSending(false)
  }

  const older = async () => {
    if (!chat) return
    const first = chat.messages[0]?.id
    const page = await api<{ messages: Message[]; more: boolean }>(
      'GET',
      `/api/messages/${encodeURIComponent(name)}?before=${first}`,
    ).catch(() => null)
    if (page) {
      stick.current = false
      setChat({ ...chat, messages: [...page.messages, ...chat.messages], more: page.more })
    }
  }

  const askDelete = async (want: boolean) => {
    await api('POST', `/api/messages/${encodeURIComponent(name)}/delete`, { want }).catch(() => {})
    load()
  }

  if (!chat)
    return (
      <div className="empty-chat">
        {error ? <span className="text-danger">{error}</span> : <Spinner label="Loading" />}
      </div>
    )

  const them = chat.with.username
  const { wins, draws, losses } = chat.score
  const friends = chat.friend === 'friends'

  return (
    <>
      <div className="chat-head">
        <Button
          variant="ghost"
          icon
          className="back-btn size-10"
          aria-label="Back to chats"
          onClick={() => navigate('/messages')}
        >
          <Icon name="back" size={20} />
        </Button>
        <a
          href={`/u/${them}`}
          onClick={(e) => {
            e.preventDefault()
            navigate(`/u/${them}`)
          }}
          className="min-w-0 flex-1"
        >
          <Who name={them} sub={presence(chat.with) || chat.with.rating} size={40} online={chat.with.online} />
        </a>
        {wins + draws + losses > 0 && (
          <span className="chat-score" title="Your score against them">
            <span className="lbl">Score</span>
            <b>
              {wins}–{losses}
            </b>
            {draws > 0 && <span>· {draws === 1 ? '1 draw' : `${draws} draws`}</span>}
          </span>
        )}
        {friends && (
          <Button
            variant="solid"
            icon
            className="size-9"
            aria-label={`Offer ${them} a match`}
            onClick={() => openOffer(them)}
          >
            <Icon name="swords" />
          </Button>
        )}
        <DropMenu
          title="Chat options"
          trigger={
            <Button variant="ghost" icon className="size-9" aria-label="Chat options">
              <Icon name="dots" />
            </Button>
          }
        >
          <MenuItem onSelect={() => navigate(`/u/${them}`)}>
            <Icon name="user" />
            Profile
          </MenuItem>
          <MenuItem onSelect={() => askDelete(!chat.deletion.mine)}>
            <Icon name="out" />
            {chat.deletion.mine ? 'Keep this chat after all' : 'Ask to delete this chat'}
          </MenuItem>
        </DropMenu>
      </div>

      <div
        className="chat-body"
        ref={body}
        onScroll={(e) => {
          const el = e.currentTarget
          stick.current = el.scrollHeight - el.scrollTop - el.clientHeight < 40
        }}
      >
        {chat.more && (
          <button className="sys underline" onClick={older}>
            Load older messages
          </button>
        )}
        <Timeline chat={chat} me={me} />
        {chat.deletion.mine && (
          <div className="sys">
            You asked {them} to delete this chat. It goes away for both of you only when they agree.
            <Button variant="ghost" className="h-8 px-3 text-[13px]" onClick={() => askDelete(false)}>
              Take it back
            </Button>
          </div>
        )}
        {chat.deletion.theirs && !chat.deletion.mine && (
          <div className="sys">
            {them} asked to delete this chat. If you agree, it's gone for both of you.
            <span className="flex gap-2">
              <Button variant="solid" className="h-8 px-3 text-[13px]" onClick={() => askDelete(true)}>
                Delete for both
              </Button>
            </span>
          </div>
        )}
      </div>

      {friends ? (
        <form className="composer" onSubmit={send}>
          <Button type="button" icon aria-label={`Offer ${them} a match`} onClick={() => openOffer(them)}>
            <Icon name="swords" />
          </Button>
          <input
            className="input"
            value={text}
            onChange={(e) => setText(e.target.value)}
            placeholder={`Message ${them}`}
            aria-label="Message"
            maxLength={1000}
            autoComplete="off"
          />
          <Button variant="solid" icon type="submit" aria-label="Send" loading={sending} disabled={!text.trim()}>
            <Icon name="send" />
          </Button>
        </form>
      ) : (
        <p className="composer text-sm text-accent">
          You're not friends with {them}, so you can't send messages. The chat stays here.
        </p>
      )}
      {error && <p className="px-4 pb-2 text-sm text-danger">{error}</p>}
    </>
  )
}

/** Keeps older pages that were loaded before a refresh of the newest page. */
function mergeOlder(old: Chat | null, fresh: Chat) {
  if (!old || fresh.messages.length === 0) return fresh.messages
  const first = fresh.messages[0].id
  return [...old.messages.filter((m) => m.id < first), ...fresh.messages]
}

type Item = { at: number; key: string; message?: Message; game?: GameSummary }

/** Messages, offers and finished games between you, in order, with a line for each new day. */
function Timeline({ chat, me }: { chat: Chat; me: string }) {
  const oldest = chat.more ? (chat.messages[0]?.at ?? 0) : 0
  const items: Item[] = [
    ...chat.messages.map((m) => ({ at: m.at, key: `m${m.id}`, message: m })),
    // Games older than the loaded messages wait until those load.
    ...chat.games
      .filter((g) => g.updatedAt >= oldest)
      .map((g) => ({ at: g.updatedAt, key: `g${g.matchId}${g.round}`, game: g })),
  ].sort((a, b) => a.at - b.at)

  if (items.length === 0) return <div className="sys">Say hi to {chat.with.username}.</div>

  let lastDay = ''
  return (
    <>
      {items.map((it) => {
        const day = new Date(it.at).toDateString()
        const showDay = day !== lastDay
        lastDay = day
        return (
          <Fragment key={it.key}>
            {showDay && <div className="day">{dayLabel(it.at)}</div>}
            {it.game ? (
              <ResultCard g={it.game} them={chat.with.username} />
            ) : it.message!.offer ? (
              <OfferCard offer={it.message!.offer} me={me} />
            ) : (
              <div className={`bub ${it.message!.from === me ? 'me' : ''}`}>
                {it.message!.text}
                <time>{timeFmt.format(it.at)}</time>
              </div>
            )}
          </Fragment>
        )
      })}
    </>
  )
}

function OfferCard({ offer, me }: { offer: Offer; me: string }) {
  const [busy, setBusy] = useState('')
  const [error, setError] = useState('')
  const mine = offer.from === me
  const act = async (action: 'accept' | 'decline' | 'cancel') => {
    setBusy(action)
    setError('')
    try {
      const res = await api<{ matchId?: string }>('POST', `/api/offers/${offer.id}/${action}`)
      if (res.matchId) navigate(`/m/${res.matchId}`)
    } catch (e) {
      setError((e as Error).message)
    }
    setBusy('')
  }
  // The colour from the reader's side.
  const yours =
    offer.color === 'random' ? 'Random' : (offer.color === 'w') === mine ? 'You play White' : 'You play Black'
  const go = (e: React.MouseEvent) => {
    e.preventDefault()
    navigate(`/m/${offer.matchId}`)
  }

  return (
    <div className={`offer ${mine ? 'me' : ''}`}>
      <div className="label">{mine ? 'Your match offer' : `Match offer from ${offer.from}`}</div>
      <OfferTags offer={offer} />
      <dl>
        <dt>Time</dt>
        <dd>{describeTime(offer.settings.time ?? NO_CLOCK)}</dd>
        <dt>Colour</dt>
        <dd>{yours}</dd>
      </dl>
      {offer.status === 'open' ? (
        <div className="acts">
          {mine ? (
            <Button variant="ghost" className="h-9 text-sm" loading={busy === 'cancel'} onClick={() => act('cancel')}>
              Cancel offer
            </Button>
          ) : (
            <>
              <Button variant="solid" className="h-9 text-sm" loading={busy === 'accept'} onClick={() => act('accept')}>
                Accept
              </Button>
              <Button
                variant="ghost"
                className="h-9 text-sm"
                loading={busy === 'decline'}
                onClick={() => act('decline')}
              >
                Decline
              </Button>
            </>
          )}
        </div>
      ) : (
        <div className="flex items-center gap-2 text-[13px] text-accent">
          <span className="chip">{statusText(offer.status)}</span>
          {offer.status === 'accepted' && offer.matchId && (
            <a href={`/m/${offer.matchId}`} onClick={go} className="font-semibold text-fg underline underline-offset-2">
              Go to the game
            </a>
          )}
        </div>
      )}
      {error && <p className="text-[13px] text-danger">{error}</p>}
    </div>
  )
}

function ResultCard({ g, them }: { g: GameSummary; them: string }) {
  const draw = g.result === '1/2-1/2'
  const won = !draw && (g.result === '1-0') === (g.color === 'w')
  const ate = g.reason === 'capturing the king'
  const title = draw
    ? 'Draw'
    : ate
      ? won
        ? `You ate ${them}'s king`
        : `${them} ate your king`
      : won
        ? `You beat ${them}`
        : `${them} won`
  const href = `/g/${g.matchId}/${g.round}`
  const diff = g.ratingDiff
  return (
    <a
      href={href}
      onClick={(e) => {
        e.preventDefault()
        navigate(href)
      }}
      className="tick result-card"
    >
      <span className={`king ${draw ? 'draw' : won ? 'won' : 'toppled'}`}>
        <img src={pieceImg(won ? 'K' : g.color === 'w' ? 'K' : 'k')} alt="" />
      </span>
      <div className="min-w-0 text-left">
        <h4>{title}</h4>
        <p>
          {describeTime(g.settings.time ?? NO_CLOCK)} · {plural(Math.ceil(g.plies / 2), 'move')}
          {!draw && !ate && ` · by ${g.reason === 'time' ? 'time' : g.reason}`}
          {diff != null && ` · ${diff > 0 ? '+' : diff < 0 ? '−' : '±'}${Math.abs(diff)}`}
        </p>
      </div>
      <span className="text-[13px] text-accent">Replay</span>
    </a>
  )
}
