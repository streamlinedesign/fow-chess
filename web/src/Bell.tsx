import * as Popover from '@radix-ui/react-popover'
import { useEffect, useState, type ReactNode } from 'react'
import { Avatar } from './Avatar'
import { Icon } from './icons'
import { refreshCounts, useCounts, useLive } from './live'
import { pieceImg } from './position'
import { ago } from './ProfilePage'
import { navigate } from './router'
import { clocked, describeSettings } from './settings'
import { api, type Notification, type Offer } from './social'
import { Button, Spinner } from './ui'

/** The bell in the top bar: friend requests, match offers, results and slow moves. */
export function Bell() {
  const { unread } = useCounts()
  const [open, setOpen] = useState(false)
  const [items, setItems] = useState<Notification[] | null>(null)

  const load = async () => {
    const body = await api<{ items: Notification[] }>('GET', '/api/notifications').catch(() => null)
    if (body) setItems(body.items)
  }

  useEffect(() => {
    if (!open) return
    load()
    // Opening the bell reads everything in it; the dots stay until it closes.
    api('POST', '/api/notifications/read')
      .then(refreshCounts)
      .catch(() => {})
  }, [open])

  useLive((e) => {
    if (open && (e.type === 'notification' || e.type === 'offer' || e.type === 'friends')) load()
  })

  const close = () => setOpen(false)

  return (
    <Popover.Root open={open} onOpenChange={setOpen}>
      <Popover.Trigger className="icon-btn" aria-label={unread ? `Notifications, ${unread} new` : 'Notifications'}>
        <Icon name="bell" size={20} />
        {unread > 0 && <span className="badge">{unread > 9 ? '9+' : unread}</span>}
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Content align="end" sideOffset={8} className="menu bell" aria-label="Notifications">
          <div className="bell-head">
            <h3>Notifications</h3>
          </div>
          <div className="bell-list">
            {!items ? (
              <div className="grid place-items-center p-8">
                <Spinner label="Loading" />
              </div>
            ) : items.length === 0 ? (
              <p className="px-3 pt-2 pb-6 text-sm text-accent">
                Nothing yet. Friend requests, match offers and your results show up here.
              </p>
            ) : (
              items.map((n) => <Note key={n.id} n={n} onChange={load} onGo={close} />)
            )}
          </div>
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  )
}

function Name({ name }: { name?: string }) {
  return <b className="font-semibold">{name || 'A guest'}</b>
}

/** What a finished game was, from your side: "You ate dino_r's king". */
function resultLine(n: Notification) {
  const who = <Name name={n.actor} />
  const { result, reason } = n.data
  if (result === 'draw')
    return (
      <>
        Your game with {who} was drawn by {reason}.
      </>
    )
  if (reason === 'capturing the king') return result === 'won' ? <>You ate {who}'s king.</> : <>{who} ate your king.</>
  const how = reason === 'time' ? 'on time' : `by ${reason}`
  return result === 'won' ? (
    <>
      You beat {who} {how}.
    </>
  ) : (
    <>
      You lost to {who} {how}.
    </>
  )
}

function Note({ n, onChange, onGo }: { n: Notification; onChange: () => void; onGo: () => void }) {
  const [busy, setBusy] = useState('')
  const [error, setError] = useState('')
  const act = async (label: string, run: () => Promise<void>) => {
    setBusy(label)
    setError('')
    try {
      await run()
    } catch (e) {
      setError((e as Error).message)
    }
    setBusy('')
    onChange()
  }
  const go = (href: string) => {
    onGo()
    navigate(href)
  }
  const game = n.matchId ? `/m/${n.matchId}` : ''

  let icon: ReactNode = <Avatar size={36} />
  let text: ReactNode
  let actions: ReactNode = null
  switch (n.kind) {
    case 'friend_request':
      text = (
        <>
          <Name name={n.actor} /> wants to be friends.
        </>
      )
      if (n.pending)
        actions = (
          <>
            <Button
              variant="solid"
              className="h-8 px-3 text-[13px]"
              loading={busy === 'accept'}
              onClick={() => act('accept', () => api('POST', `/api/friends/${n.actor}`))}
            >
              Accept
            </Button>
            <Button
              variant="ghost"
              className="h-8 px-3 text-[13px]"
              loading={busy === 'decline'}
              onClick={() => act('decline', () => api('DELETE', `/api/friends/${n.actor}`))}
            >
              Decline
            </Button>
          </>
        )
      break
    case 'friend_accepted':
      text = (
        <>
          <Name name={n.actor} /> accepted your friend request.
        </>
      )
      break
    case 'offer':
      text = (
        <>
          <Name name={n.actor} /> offered you a match.
        </>
      )
      if (n.offer) actions = <OfferActions offer={n.offer} busy={busy} act={act} go={go} />
      break
    case 'offer_accepted':
      text = (
        <>
          <Name name={n.actor} /> accepted your match offer.
        </>
      )
      actions = (
        <Button variant="solid" className="h-8 px-3 text-[13px]" onClick={() => go(game)}>
          Go to the game
        </Button>
      )
      break
    case 'offer_declined':
      text = (
        <>
          <Name name={n.actor} /> declined your match offer.
        </>
      )
      break
    case 'game_over':
      icon = (
        <span className="note-icon">
          <img
            src={pieceImg(n.data.result === 'won' ? 'k' : 'K')}
            alt=""
            className={n.data.result === 'draw' ? '' : '-rotate-90'}
          />
        </span>
      )
      text = resultLine(n)
      break
    case 'move':
      icon = (
        <span className="note-icon">
          <img src={pieceImg('n')} alt="" />
        </span>
      )
      text = (
        <>
          <Name name={n.actor} /> moved in your game. It's your move.
        </>
      )
      break
  }
  const link = (n.kind === 'game_over' || n.kind === 'move') && game

  return (
    <div className="note" data-unread={!n.read || undefined}>
      {icon}
      <div className="min-w-0">
        {link ? (
          <a
            href={link}
            onClick={(e) => {
              e.preventDefault()
              go(link)
            }}
            className="block hover:underline"
          >
            <p>{text}</p>
          </a>
        ) : (
          <p>{text}</p>
        )}
        {n.kind === 'offer' && n.offer && <OfferTags offer={n.offer} />}
        <div className="meta">{ago(n.at)}</div>
        {actions && <div className="acts">{actions}</div>}
        {error && <p className="mt-2 text-[13px] text-danger">{error}</p>}
      </div>
    </div>
  )
}

/** The rules and clock of an offer as small tags. */
export function OfferTags({ offer }: { offer: Offer }) {
  return (
    <div className="chips">
      {describeSettings(offer.settings).map((t) => (
        <span key={t} className="chip">
          {t}
        </span>
      ))}
      {clocked(offer.settings.time ?? { base: 0, inc: 0, perMove: 0 }) && <span className="chip solid">Rated</span>}
    </div>
  )
}

function OfferActions({
  offer,
  busy,
  act,
  go,
}: {
  offer: Offer
  busy: string
  act: (label: string, run: () => Promise<void>) => void
  go: (href: string) => void
}) {
  if (offer.status === 'accepted' && offer.matchId)
    return (
      <Button variant="solid" className="h-8 px-3 text-[13px]" onClick={() => go(`/m/${offer.matchId}`)}>
        Go to the game
      </Button>
    )
  if (offer.status !== 'open') return <span className="text-[13px] text-accent">{statusText(offer.status)}</span>
  return (
    <>
      <Button
        variant="solid"
        className="h-8 px-3 text-[13px]"
        loading={busy === 'accept'}
        onClick={() =>
          act('accept', async () => {
            const { matchId } = await api<{ matchId: string }>('POST', `/api/offers/${offer.id}/accept`)
            go(`/m/${matchId}`)
          })
        }
      >
        Accept
      </Button>
      <Button
        variant="ghost"
        className="h-8 px-3 text-[13px]"
        loading={busy === 'decline'}
        onClick={() => act('decline', () => api('POST', `/api/offers/${offer.id}/decline`))}
      >
        Decline
      </Button>
    </>
  )
}

export function statusText(status: Offer['status']) {
  return { open: 'Waiting for an answer', accepted: 'Accepted', declined: 'Declined', cancelled: 'Cancelled' }[status]
}
