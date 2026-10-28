import * as Popover from '@radix-ui/react-popover'
import { useEffect, useState, type ReactNode } from 'react'
import { Avatar } from './Avatar'
import { fill, useT, type Dict } from './i18n'
import { Icon } from './icons'
import { refreshCounts, useCounts, useLive } from './live'
import { pieceImg } from './position'
import { ago } from './ProfilePage'
import { how } from './MatchPage'
import { navigate } from './router'
import { BottomDrawer, usePhone } from './Sheet'
import { clocked, describeSettings } from './settings'
import { api, type Notification, type Offer } from './social'
import { Button, Spinner } from './ui'

/** The bell in the top bar: friend requests, match offers, results and slow moves. */
export function Bell() {
  const t = useT()
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
  const phone = usePhone()

  const trigger = (
    <button className="icon-btn" aria-label={unread ? t.bell.titleNew(unread) : t.bell.title}>
      <Icon name="bell" size={20} />
      {unread > 0 && <span className="badge">{unread > 9 ? '9+' : unread}</span>}
    </button>
  )
  const list = (
    <div className="bell-list">
      {!items ? (
        <div className="grid place-items-center p-8">
          <Spinner label={t.common.loading} />
        </div>
      ) : items.length === 0 ? (
        <p className="px-3 pt-2 pb-6 text-sm text-accent">{t.bell.nothingYet}</p>
      ) : (
        items.map((n) => <Note key={n.id} n={n} onChange={load} onGo={close} />)
      )}
    </div>
  )

  // On phones the bell opens as a drawer from the bottom.
  if (phone)
    return (
      <BottomDrawer open={open} onOpenChange={setOpen} trigger={trigger} title={t.bell.title} className="bell-drawer">
        {list}
      </BottomDrawer>
    )
  return (
    <Popover.Root open={open} onOpenChange={setOpen}>
      <Popover.Trigger asChild>{trigger}</Popover.Trigger>
      <Popover.Portal>
        <Popover.Content align="end" sideOffset={8} className="menu bell" aria-label={t.bell.title}>
          <div className="bell-head">
            <h3>{t.bell.title}</h3>
          </div>
          {list}
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  )
}

function Name({ name }: { name?: string }) {
  const t = useT()
  return <b className="font-semibold">{name || t.common.aGuest}</b>
}

/** What a finished game was, from your side: "You ate dino_r's king". */
function resultLine(n: Notification, t: Dict) {
  const parts = { who: <Name name={n.actor} />, how: how(n.data.reason, t) }
  const { result, reason } = n.data
  const line = (key: keyof Dict['bell']) => fill(t.bell[key] as string, parts)
  if (result === 'draw') return line('drawn')
  if (n.data.sniper) return line(result === 'won' ? 'sniperWon' : 'sniperLost')
  if (reason === 'capturing the king') return line(result === 'won' ? 'ateKing' : 'ateYourKing')
  if (reason === 'reaching the hill') return line(result === 'won' ? 'tookHill' : 'theyTookHill')
  return line(result === 'won' ? 'beat' : 'lostTo')
}

function Note({ n, onChange, onGo }: { n: Notification; onChange: () => void; onGo: () => void }) {
  const t = useT()
  const who = { who: <Name name={n.actor} /> }
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
      text = fill(t.bell.wantsFriends, who)
      if (n.pending)
        actions = (
          <>
            <Button
              variant="solid"
              className="h-8 px-3 text-[13px]"
              loading={busy === 'accept'}
              onClick={() => act('accept', () => api('POST', `/api/friends/${n.actor}`))}
            >
              {t.common.accept}
            </Button>
            <Button
              variant="ghost"
              className="h-8 px-3 text-[13px]"
              loading={busy === 'decline'}
              onClick={() => act('decline', () => api('DELETE', `/api/friends/${n.actor}`))}
            >
              {t.common.decline}
            </Button>
          </>
        )
      break
    case 'friend_accepted':
      text = fill(t.bell.acceptedFriend, who)
      break
    case 'offer':
      text = fill(t.bell.offeredMatch, who)
      if (n.offer) actions = <OfferActions offer={n.offer} busy={busy} act={act} go={go} />
      break
    case 'offer_accepted':
      text = fill(t.bell.acceptedOffer, who)
      actions = (
        <Button variant="solid" className="h-8 px-3 text-[13px]" onClick={() => go(game)}>
          {t.bell.goToGame}
        </Button>
      )
      break
    case 'offer_declined':
      text = fill(t.bell.declinedOffer, who)
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
      text = resultLine(n, t)
      break
    case 'move':
      icon = (
        <span className="note-icon">
          <img src={pieceImg('n')} alt="" />
        </span>
      )
      text = fill(t.bell.moved, who)
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
        <div className="meta">{ago(n.at, t)}</div>
        {actions && <div className="acts">{actions}</div>}
        {error && <p className="mt-2 text-[13px] text-danger">{error}</p>}
      </div>
    </div>
  )
}

/** The rules and clock of an offer as small tags. */
export function OfferTags({ offer }: { offer: Offer }) {
  const t = useT()
  return (
    <div className="chips">
      {describeSettings(offer.settings, t).map((tag) => (
        <span key={tag} className="chip">
          {tag}
        </span>
      ))}
      {clocked(offer.settings.time ?? { base: 0, inc: 0, perMove: 0 }) && (
        <span className="chip solid">{t.common.rated}</span>
      )}
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
  const t = useT()
  if (offer.status === 'accepted' && offer.matchId)
    return (
      <Button variant="solid" className="h-8 px-3 text-[13px]" onClick={() => go(`/m/${offer.matchId}`)}>
        {t.bell.goToGame}
      </Button>
    )
  if (offer.status !== 'open') return <span className="text-[13px] text-accent">{statusText(offer.status, t)}</span>
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
        {t.common.accept}
      </Button>
      <Button
        variant="ghost"
        className="h-8 px-3 text-[13px]"
        loading={busy === 'decline'}
        onClick={() => act('decline', () => api('POST', `/api/offers/${offer.id}/decline`))}
      >
        {t.common.decline}
      </Button>
    </>
  )
}

export function statusText(status: Offer['status'], t: Dict) {
  return t.bell.status[status]
}
