import { Component, type ReactNode } from 'react'
import { Button } from './ui'

// Shows what went wrong instead of an empty screen if rendering ever throws.
export class ErrorBoundary extends Component<{ children: ReactNode }, { error: Error | null }> {
  state = { error: null as Error | null }

  static getDerivedStateFromError(error: Error) {
    return { error }
  }

  render() {
    const { error } = this.state
    if (!error) return this.props.children
    return (
      <main className="flex flex-1 flex-col items-start justify-center gap-6 px-4 py-10 sm:px-8">
        <h2 className="text-[clamp(42px,8vw,64px)] leading-none font-bold tracking-tight">Something broke</h2>
        <p className="max-w-xl text-accent">Reloading usually fixes it. If it keeps happening, send this message along:</p>
        <pre className="max-w-full overflow-x-auto rounded-xl p-4 font-mono text-sm whitespace-pre-wrap shadow-[inset_0_0_0_2px_var(--color-fg)]">
          {error.message}
        </pre>
        <Button variant="solid" onClick={() => location.reload()}>
          Reload
        </Button>
      </main>
    )
  }
}
