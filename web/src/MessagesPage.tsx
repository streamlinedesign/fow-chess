import { Fragment, useEffect, useLayoutEffect, useRef, useState } from 'react'
import { useMe } from './auth'
import { Avatar, Who } from './Avatar'
import { OfferTags, statusText } from './Bell'
import { SignInFirst } from './FriendsPage'
import { useT, type Dict } from './i18n'
import { Icon } from './icons'
import { refreshCounts, useLive } from './live'
import { pieceImg } from './position'
import { type GameSummary } from './ProfilePage'
import { how } from './MatchPage'
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

const timeFmt = (t: Dict) => new Intl.DateTimeFormat(t.locale, { hour: '2-digit', minute: '2-digit' })

/** "14:02" today, "Mon" this week, "20 Oct" before that. */
function shortTime(ms: number, t: Dict) {
  const days = (Date.now() - ms) / 86400000
  if (new Date(ms).toDateString() === new Date().toDateString()) return timeFmt(t).format(ms)
  return days < 6
    ? new Intl.DateTimeFormat(t.locale, { weekday: 'short' }).format(ms)
    : new Intl.DateTimeFormat(t.locale, { day: 'numeric', month: 'short' }).format(ms)
}

function dayLabel(ms: number, t: Dict) {
  const d = new Date(ms).toDateString()
  if (d === new Date().toDateString()) return t.messages.today
  if (d === new Date(Date.now() - 86400000).toDateString()) return t.messages.yesterday
  return new Intl.DateTimeFormat(t.locale, { weekday: 'long', day: 'numeric', month: 'long' }).format(ms)
}

/** Chats with friends: the list, and the open one beside it (or on its own on phones). */
export function MessagesPage({ name }: { name?: string }) {
  const me = useMe()
  const t = useT()
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

  if (me === null) return <SignInFirst what={t.friends.whatMessages} />
  if (!me) return null

  return (
    <main className="msgs-page mx-auto w-full max-w-[1184px] flex-1 px-4 pb-6 sm:px-8">
      <div className="msgs" data-open={name ? '' : undefined}>
        <aside className="convs">
          <div className="convs-head">
            <h1>{t.common.messages}</h1>
            <Button variant="ghost" className="h-8 px-3 text-[13px]" onClick={() => navigate('/friends')}>
              <Icon name="plus" />
              {t.messages.new}
            </Button>
          </div>
          <div className="conv-list">
            {!convs ? (
              <div className="p-4">
                <Spinner />
              </div>
            ) : convs.length === 0 ? (
              <p className="p-3 text-sm text-accent">{t.messages.noChats}</p>
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
                      {c.last.from === me.username && t.chat.youPrefix}
                      {c.last.offer ? t.messages.matchOffer : c.last.text}
                    </div>
                  </div>
                  <span className="t">
                    {shortTime(c.last.at, t)}
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
            <div className="empty-chat">{t.messages.pickChat}</div>
          )}
        </section>
      </div>
    </main>
  )
}

function ChatView({ name, me }: { name: string; me: string }) {
  const t = useT()
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
    const line = text.trim()
    if (!line) return
    setSending(true)
    try {
      await api('POST', `/api/messages/${encodeURIComponent(name)}`, { text: line })
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
        {error ? <span className="text-danger">{error}</span> : <Spinner label={t.common.loading} />}
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
          aria-label={t.messages.backToChats}
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
          <Who name={them} sub={presence(chat.with, t) || chat.with.rating} size={40} online={chat.with.online} />
        </a>
        {wins + draws + losses > 0 && (
          <span className="chat-score" title={t.messages.scoreTitle}>
            <span className="lbl">{t.messages.score}</span>
            <b>
              {wins}–{losses}
            </b>
            {draws > 0 && <span>· {t.common.draws(draws)}</span>}
          </span>
        )}
        {friends && (
          <Button
            variant="solid"
            icon
            className="size-9"
            aria-label={t.friends.offerXMatch(them)}
            onClick={() => openOffer(them)}
          >
            <Icon name="swords" />
          </Button>
        )}
        <DropMenu
          title={t.messages.chatOptions}
          trigger={
            <Button variant="ghost" icon className="size-9" aria-label={t.messages.chatOptions}>
              <Icon name="dots" />
            </Button>
          }
        >
          <MenuItem onSelect={() => navigate(`/u/${them}`)}>
            <Icon name="user" />
            {t.common.profile}
          </MenuItem>
          <MenuItem onSelect={() => askDelete(!chat.deletion.mine)}>
            <Icon name="out" />
            {chat.deletion.mine ? t.messages.keepChat : t.messages.askDelete}
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
            {t.messages.loadOlder}
          </button>
        )}
        <Timeline chat={chat} me={me} />
        {chat.deletion.mine && (
          <div className="sys">
            {t.messages.youAskedDelete(them)}
            <Button variant="ghost" className="h-8 px-3 text-[13px]" onClick={() => askDelete(false)}>
              {t.messages.takeBack}
            </Button>
          </div>
        )}
        {chat.deletion.theirs && !chat.deletion.mine && (
          <div className="sys">
            {t.messages.theyAskedDelete(them)}
            <span className="flex gap-2">
              <Button variant="solid" className="h-8 px-3 text-[13px]" onClick={() => askDelete(true)}>
                {t.messages.deleteForBoth}
              </Button>
            </span>
          </div>
        )}
      </div>

      {friends ? (
        <form className="composer" onSubmit={send}>
          <Button type="button" icon aria-label={t.friends.offerXMatch(them)} onClick={() => openOffer(them)}>
            <Icon name="swords" />
          </Button>
          <input
            className="input"
            value={text}
            onChange={(e) => setText(e.target.value)}
            placeholder={t.friends.messageX(them)}
            aria-label={t.common.message}
            maxLength={1000}
            autoComplete="off"
          />
          <Button
            variant="solid"
            icon
            type="submit"
            aria-label={t.common.send}
            loading={sending}
            disabled={!text.trim()}
          >
            <Icon name="send" />
          </Button>
        </form>
      ) : (
        <p className="composer text-sm text-accent">{t.messages.notFriends(them)}</p>
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
  const t = useT()
  const oldest = chat.more ? (chat.messages[0]?.at ?? 0) : 0
  const items: Item[] = [
    ...chat.messages.map((m) => ({ at: m.at, key: `m${m.id}`, message: m })),
    // Games older than the loaded messages wait until those load.
    ...chat.games
      .filter((g) => g.updatedAt >= oldest)
      .map((g) => ({ at: g.updatedAt, key: `g${g.matchId}${g.round}`, game: g })),
  ].sort((a, b) => a.at - b.at)

  if (items.length === 0) return <div className="sys">{t.chat.sayHi(chat.with.username)}</div>

  let lastDay = ''
  return (
    <>
      {items.map((it) => {
        const day = new Date(it.at).toDateString()
        const showDay = day !== lastDay
        lastDay = day
        return (
          <Fragment key={it.key}>
            {showDay && <div className="day">{dayLabel(it.at, t)}</div>}
            {it.game ? (
              <ResultCard g={it.game} them={chat.with.username} />
            ) : it.message!.offer ? (
              <OfferCard offer={it.message!.offer} me={me} />
            ) : (
              <div className={`bub ${it.message!.from === me ? 'me' : ''}`}>
                {it.message!.text}
                <time>{timeFmt(t).format(it.at)}</time>
              </div>
            )}
          </Fragment>
        )
      })}
    </>
  )
}

function OfferCard({ offer, me }: { offer: Offer; me: string }) {
  const t = useT()
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
    offer.color === 'random'
      ? t.messages.random
      : (offer.color === 'w') === mine
        ? t.messages.youPlayWhite
        : t.messages.youPlayBlack
  const go = (e: React.MouseEvent) => {
    e.preventDefault()
    navigate(`/m/${offer.matchId}`)
  }

  return (
    <div className={`offer ${mine ? 'me' : ''}`}>
      <div className="label">{mine ? t.messages.yourOffer : t.messages.offerFrom(offer.from)}</div>
      <OfferTags offer={offer} />
      <dl>
        <dt>{t.messages.time}</dt>
        <dd>{describeTime(offer.settings.time ?? NO_CLOCK, t)}</dd>
        <dt>{t.messages.colour}</dt>
        <dd>{yours}</dd>
      </dl>
      {offer.status === 'open' ? (
        <div className="acts">
          {mine ? (
            <Button variant="ghost" className="h-9 text-sm" loading={busy === 'cancel'} onClick={() => act('cancel')}>
              {t.messages.cancelOffer}
            </Button>
          ) : (
            <>
              <Button variant="solid" className="h-9 text-sm" loading={busy === 'accept'} onClick={() => act('accept')}>
                {t.common.accept}
              </Button>
              <Button
                variant="ghost"
                className="h-9 text-sm"
                loading={busy === 'decline'}
                onClick={() => act('decline')}
              >
                {t.common.decline}
              </Button>
            </>
          )}
        </div>
      ) : (
        <div className="flex items-center gap-2 text-[13px] text-accent">
          <span className="chip">{statusText(offer.status, t)}</span>
          {offer.status === 'accepted' && offer.matchId && (
            <a href={`/m/${offer.matchId}`} onClick={go} className="font-semibold text-fg underline underline-offset-2">
              {t.messages.goToGame}
            </a>
          )}
        </div>
      )}
      {error && <p className="text-[13px] text-danger">{error}</p>}
    </div>
  )
}

function ResultCard({ g, them }: { g: GameSummary; them: string }) {
  const t = useT()
  const draw = g.result === '1/2-1/2'
  const won = !draw && (g.result === '1-0') === (g.color === 'w')
  const ate = g.reason === 'capturing the king'
  const title = draw
    ? t.match.draw
    : ate
      ? won
        ? t.messages.youAteKing(them)
        : t.messages.ateYourKing(them)
      : won
        ? t.messages.youBeat(them)
        : t.messages.xWon(them)
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
          {describeTime(g.settings.time ?? NO_CLOCK, t)} · {t.common.moves(Math.ceil(g.plies / 2))}
          {!draw && !ate && ` · ${how(g.reason, t)}`}
          {diff != null && ` · ${diff > 0 ? '+' : diff < 0 ? '−' : '±'}${Math.abs(diff)}`}
        </p>
      </div>
      <span className="text-[13px] text-accent">{t.common.replay}</span>
    </a>
  )
}
