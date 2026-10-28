import * as Tooltip from '@radix-ui/react-tooltip'
import { useEffect } from 'react'
import { useMe } from './auth'
import { AuthPage } from './AuthPage'
import { ErrorBoundary } from './ErrorBoundary'
import { Header } from './Header'
import { FriendsPage } from './FriendsPage'
import { Landing } from './Landing'
import { connectLive, useLive } from './live'
import { MatchPage } from './MatchPage'
import { MessagesPage } from './MessagesPage'
import { NotFound } from './NotFound'
import { OfferDialog } from './OfferDialog'
import { ProfilePage } from './ProfilePage'
import { ReplayPage } from './ReplayPage'
import { usePath } from './router'
import { SettingsPage } from './SettingsPage'
import { showToast } from './social'
import { Toaster } from './Toaster'

export function App() {
  const path = usePath()
  const match = /^\/m\/([A-Za-z0-9]{4,12})\/?$/.exec(path)
  const user = /^\/u\/([A-Za-z0-9_-]{1,20})\/?$/.exec(path)
  const game = /^\/g\/([A-Za-z0-9]{4,12})\/(\d+)\/?$/.exec(path)
  const chat = /^\/messages(?:\/([A-Za-z0-9_-]{1,20}))?\/?$/.exec(path)
  const me = useMe()

  // Live updates follow whoever is signed in.
  useEffect(() => {
    if (me !== undefined) connectLive(!!me)
  }, [me?.username])

  // Someone accepting your offer starts a game you'll want to get to.
  useLive((e) => {
    if (e.type === 'offer' && e.matchId && e.user)
      showToast(
        { text: `${e.user} accepted your match offer.`, action: { label: 'Play', href: `/m/${e.matchId}` } },
        20000,
      )
  })

  const page = match ? (
    <MatchPage key={match[1].toUpperCase()} id={match[1].toUpperCase()} />
  ) : user ? (
    <ProfilePage name={user[1]} />
  ) : game ? (
    <ReplayPage id={game[1].toUpperCase()} round={Number(game[2])} />
  ) : path === '/login' || path === '/signup' ? (
    <AuthPage key={path} mode={path === '/login' ? 'login' : 'signup'} />
  ) : chat ? (
    <MessagesPage name={chat[1]} />
  ) : path === '/friends' ? (
    <FriendsPage />
  ) : path === '/settings' ? (
    <SettingsPage />
  ) : path === '/' ? (
    <Landing />
  ) : (
    <NotFound />
  )

  return (
    <Tooltip.Provider delayDuration={300}>
      <div className="flex min-h-dvh flex-col">
        <Header />
        <ErrorBoundary key={path}>{page}</ErrorBoundary>
        <OfferDialog />
        <Toaster />
      </div>
    </Tooltip.Provider>
  )
}
