import { navigate } from './router'
import { toggleTheme } from './theme'

export function Header() {
  return (
    <header className="flex items-center justify-between px-4 pt-4 pb-2 sm:px-8 sm:pt-6">
      <a
        href="/"
        onClick={(e) => {
          e.preventDefault()
          navigate('/')
        }}
        className="text-2xl font-light tracking-tight sm:text-[32px]"
      >
        chess
      </a>
      <button
        onClick={toggleTheme}
        aria-label="Toggle dark mode"
        className="grid size-10 place-items-center rounded-full transition-opacity hover:opacity-70"
      >
        <svg className="sun" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
          <circle cx="12" cy="12" r="4.5" />
          <path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4" />
        </svg>
        <svg className="moon" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinejoin="round">
          <path d="M20.5 14.5A8.5 8.5 0 0 1 9.5 3.5a8.5 8.5 0 1 0 11 11Z" />
        </svg>
      </button>
    </header>
  )
}
