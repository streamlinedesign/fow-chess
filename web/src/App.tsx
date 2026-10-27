import * as Tooltip from '@radix-ui/react-tooltip'
import { AuthPage } from './AuthPage'
import { ErrorBoundary } from './ErrorBoundary'
import { Header } from './Header'
import { Landing } from './Landing'
import { MatchPage } from './MatchPage'
import { ProfilePage } from './ProfilePage'
import { ReplayPage } from './ReplayPage'
import { usePath } from './router'

export function App() {
  const path = usePath()
  const match = /^\/m\/([A-Za-z0-9]{4,12})\/?$/.exec(path)
  const user = /^\/u\/([A-Za-z0-9_-]{1,20})\/?$/.exec(path)
  const game = /^\/g\/([A-Za-z0-9]{4,12})\/(\d+)\/?$/.exec(path)

  const page = match ? (
    <MatchPage key={match[1].toUpperCase()} id={match[1].toUpperCase()} />
  ) : user ? (
    <ProfilePage name={user[1]} />
  ) : game ? (
    <ReplayPage id={game[1].toUpperCase()} round={Number(game[2])} />
  ) : path === '/login' || path === '/signup' ? (
    <AuthPage key={path} mode={path === '/login' ? 'login' : 'signup'} />
  ) : (
    <Landing />
  )

  return (
    <Tooltip.Provider delayDuration={300}>
      <div className="flex min-h-dvh flex-col">
        <Header />
        <ErrorBoundary key={path}>{page}</ErrorBoundary>
      </div>
    </Tooltip.Provider>
  )
}
