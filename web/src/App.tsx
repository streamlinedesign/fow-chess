import * as Tooltip from '@radix-ui/react-tooltip'
import { useEffect } from 'react'
import { useMe } from './auth'
import { AuthPage } from './AuthPage'
import { ErrorBoundary } from './ErrorBoundary'
import { Header } from './Header'
import { dict } from './i18n'
import { FriendsPage } from './FriendsPage'
import { Landing } from './Landing'
import { Footer } from './Page'
import { connectLive, useLive } from './live'
import { FourMatchPage } from './FourMatchPage'
import { MatchPage } from './MatchPage'
import { MessagesPage } from './MessagesPage'
import { NotFound } from './NotFound'
import { OfferDialog } from './OfferDialog'
import { ProfilePage } from './ProfilePage'
import { ReplayPage } from './ReplayPage'
import { RulesPage } from './RulesPage'
import { ModsPage } from './ModsPage'
import { TournamentsPage } from './TournamentsPage'
import { DOWNLOADS, TOURNAMENTS } from './features'
import { AppHome } from './AppHome'
import { DESKTOP } from './desktop'
import { DownloadPage } from './DownloadPage'
import { navigate, usePath } from './router'
import { WatchPage } from './WatchPage'
import { SettingsPage } from './SettingsPage'
import { showToast } from './social'
import { Toaster } from './Toaster'

export function App() {
  const path = usePath()
  const match = /^\/m\/([A-Za-z0-9]{4,12})\/?$/.exec(path)
  const four = /^\/f\/([A-Za-z0-9]{4,12})\/?$/.exec(path)
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
        {
          text: dict().toast.acceptedOffer(e.user),
          action: { label: dict().toast.play, href: `/m/${e.matchId}` },
        },
        20000,
      )
    // Find match or an arena paired you: go to the game, wherever you are.
    if (e.type === 'match' && e.matchId) navigate(`/m/${e.matchId}`)
    if (e.type === 'four' && e.matchId) navigate(`/f/${e.matchId}`)
  })

  const page = match ? (
    <MatchPage key={match[1].toUpperCase()} id={match[1].toUpperCase()} />
  ) : four ? (
    <FourMatchPage key={four[1].toUpperCase()} id={four[1].toUpperCase()} />
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
  ) : path === '/watch' ? (
    <WatchPage />
  ) : path === '/settings' ? (
    <SettingsPage />
  ) : path === '/rules' ? (
    <RulesPage />
  ) : path === '/mods' ? (
    <ModsPage />
  ) : TOURNAMENTS && path === '/tournaments' ? (
    <TournamentsPage />
  ) : DOWNLOADS && path === '/download' ? (
    <DownloadPage />
  ) : path === '/' ? (
    DESKTOP ? (
      <AppHome />
    ) : (
      <Landing />
    )
  ) : (
    <NotFound />
  )

  return (
    <Tooltip.Provider delayDuration={300}>
      <div className="flex min-h-dvh flex-col">
        <Header />
        <ErrorBoundary key={path}>{page}</ErrorBoundary>
        {/* Game pages and chats fill the screen; the rest end with the footer. */}
        {!match && !game && !chat && !DESKTOP && <Footer />}
        <OfferDialog />
        <Toaster />
      </div>
    </Tooltip.Provider>
  )
}
