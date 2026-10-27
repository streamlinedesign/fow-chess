import * as Tooltip from '@radix-ui/react-tooltip'
import { ErrorBoundary } from './ErrorBoundary'
import { Header } from './Header'
import { Landing } from './Landing'
import { MatchPage } from './MatchPage'
import { usePath } from './router'

export function App() {
  const path = usePath()
  const match = /^\/m\/([A-Za-z0-9]{4,12})\/?$/.exec(path)

  return (
    <Tooltip.Provider delayDuration={300}>
      <div className="flex min-h-dvh flex-col">
        <Header />
        <ErrorBoundary key={path}>
          {match ? <MatchPage key={match[1].toUpperCase()} id={match[1].toUpperCase()} /> : <Landing />}
        </ErrorBoundary>
      </div>
    </Tooltip.Provider>
  )
}
