import * as Menu from '@radix-ui/react-dropdown-menu'
import { useState } from 'react'
import { logout, useMe } from './auth'
import { Avatar } from './Avatar'
import { navigate } from './router'
import { setThemeMode, themeMode, toggleTheme, type ThemeMode } from './theme'

function go(e: React.MouseEvent, href: string) {
  e.preventDefault()
  navigate(href)
}

/** The same bar on every screen size: the name, then your rating and avatar menu, or Log in. */
export function Header() {
  const me = useMe()
  const here = location.pathname + location.search
  const loginHref =
    here === '/' || here.startsWith('/login') || here.startsWith('/signup')
      ? '/login'
      : `/login?next=${encodeURIComponent(here)}`
  return (
    <header className="flex h-16 items-center justify-between gap-3 px-4 sm:h-20 sm:px-8">
      <a
        href="/"
        onClick={(e) => go(e, '/')}
        className="text-[22px] font-light tracking-tight whitespace-nowrap underline decoration-1 underline-offset-[5px] sm:text-[28px]"
      >
        Eat the King
      </a>
      <div className="flex items-center gap-1.5">
        {me === undefined ? null : me ? (
          <>
            <a
              href={`/u/${me.username}`}
              onClick={(e) => go(e, `/u/${me.username}`)}
              aria-label={`Your rating, ${me.rating}`}
              className="flex h-8 items-center gap-1.5 rounded-md bg-muted px-2.5 font-mono text-sm tabular-nums transition-colors hover:bg-fg/10"
            >
              <svg
                width="14"
                height="14"
                viewBox="0 0 15 15"
                fill="none"
                aria-hidden
                className="text-accent max-sm:hidden"
              >
                <path
                  d="M1.5 11 5.5 7l2.5 2.5L13.5 4M10 4h3.5v3.5"
                  stroke="currentColor"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
              </svg>
              {me.rating}
            </a>
            <UserMenu username={me.username} rating={me.rating} />
          </>
        ) : (
          <>
            <button
              onClick={toggleTheme}
              aria-label="Toggle dark mode"
              className="grid size-10 place-items-center rounded-md transition-colors hover:bg-muted"
            >
              <SunIcon className="sun" />
              <MoonIcon className="moon" />
            </button>
            <a
              href={loginHref}
              onClick={(e) => go(e, loginHref)}
              className="flex h-9 items-center rounded-md px-3.5 text-sm font-semibold shadow-[inset_0_0_0_1.5px_var(--color-fg)] transition-colors hover:bg-fg hover:text-bg"
            >
              Log in
            </a>
          </>
        )}
      </div>
    </header>
  )
}

const MODES: [ThemeMode, string][] = [
  ['light', 'Light'],
  ['dark', 'Dark'],
  ['system', 'System'],
]

function UserMenu({ username, rating }: { username: string; rating: number }) {
  const [mode, setMode] = useState(themeMode)
  return (
    <Menu.Root modal={false}>
      <Menu.Trigger
        aria-label="Your menu"
        className="rounded-full p-0.5 transition-shadow outline-none hover:shadow-[0_0_0_2px_var(--color-fg)] focus-visible:shadow-[0_0_0_2px_var(--color-fg)] data-[state=open]:shadow-[0_0_0_2px_var(--color-fg)]"
      >
        <Avatar size={34} />
      </Menu.Trigger>
      <Menu.Portal>
        <Menu.Content align="end" sideOffset={8} className="menu">
          <div className="flex items-center gap-3 px-2.5 pt-3 pb-3.5">
            <Avatar size={44} />
            <div className="flex min-w-0 flex-col">
              <span className="truncate text-[17px] font-semibold">{username}</span>
              <span className="font-mono text-sm text-accent tabular-nums">{rating}</span>
            </div>
          </div>
          <Menu.Separator className="mx-1 my-1.5 h-px bg-border" />
          <Menu.Item className="menu-item" onSelect={() => navigate(`/u/${username}`)}>
            <svg width="16" height="16" viewBox="0 0 15 15" fill="none" aria-hidden>
              <circle cx="7.5" cy="5" r="2.5" stroke="currentColor" />
              <path d="M2.5 13c.6-2.5 2.6-4 5-4s4.4 1.5 5 4" stroke="currentColor" strokeLinecap="round" />
            </svg>
            Profile
          </Menu.Item>
          <Menu.Separator className="mx-1 my-1.5 h-px bg-border" />
          <div className="flex items-center justify-between gap-2 px-2.5 py-1.5 text-sm">
            <span>Theme</span>
            <div role="radiogroup" aria-label="Theme" className="flex gap-0.5 rounded-lg bg-muted p-[3px]">
              {MODES.map(([m, label]) => (
                <button
                  key={m}
                  role="radio"
                  aria-checked={mode === m}
                  onClick={(e) => {
                    setThemeMode(m, e)
                    setMode(m)
                  }}
                  className="h-7 rounded-[5px] px-2.5 text-[13px] transition-colors hover:bg-fg/10 aria-checked:bg-fg aria-checked:text-bg"
                >
                  {label}
                </button>
              ))}
            </div>
          </div>
          <Menu.Separator className="mx-1 my-1.5 h-px bg-border" />
          <Menu.Item
            className="menu-item"
            onSelect={async () => {
              await logout()
              navigate('/')
            }}
          >
            <svg width="16" height="16" viewBox="0 0 15 15" fill="none" aria-hidden>
              <path
                d="M6 2H3.5A1.5 1.5 0 0 0 2 3.5v8A1.5 1.5 0 0 0 3.5 13H6M10 10.5l3-3-3-3M13 7.5H5.5"
                stroke="currentColor"
                strokeLinecap="round"
                strokeLinejoin="round"
              />
            </svg>
            Log out
          </Menu.Item>
        </Menu.Content>
      </Menu.Portal>
    </Menu.Root>
  )
}

function SunIcon({ className }: { className: string }) {
  return (
    <svg
      className={className}
      width="20"
      height="20"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
    >
      <circle cx="12" cy="12" r="4.5" />
      <path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4" />
    </svg>
  )
}

function MoonIcon({ className }: { className: string }) {
  return (
    <svg
      className={className}
      width="20"
      height="20"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinejoin="round"
    >
      <path d="M20.5 14.5A8.5 8.5 0 0 1 9.5 3.5a8.5 8.5 0 1 0 11 11Z" />
    </svg>
  )
}
