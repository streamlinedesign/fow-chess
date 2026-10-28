import { useState } from 'react'
import { logout, useMe } from './auth'
import { Avatar } from './Avatar'
import { Bell } from './Bell'
import { LANGS, setLang, useLang, useT } from './i18n'
import { Icon } from './icons'
import { useCounts } from './live'
import { navigate } from './router'
import { DropMenu, MenuItem, MenuSeparator } from './Sheet'
import { setThemeMode, themeMode, toggleTheme, type ThemeMode } from './theme'

function go(e: React.MouseEvent, href: string) {
  e.preventDefault()
  navigate(href)
}

/** The same bar on every screen size: the name, then your rating and avatar menu, or Log in. */
export function Header() {
  const me = useMe()
  const t = useT()
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
        {t.common.appName}
      </a>
      <div className="flex items-center gap-1.5">
        {me === undefined ? null : me ? (
          <>
            <a
              href={`/u/${me.username}`}
              onClick={(e) => go(e, `/u/${me.username}`)}
              aria-label={t.header.yourRating(me.rating)}
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
            <Bell />
            <UserMenu username={me.username} rating={me.rating} />
          </>
        ) : (
          <>
            <a
              href="/watch"
              onClick={(e) => go(e, '/watch')}
              className="flex h-9 items-center rounded-md px-3 text-sm font-semibold transition-colors hover:bg-muted"
            >
              {t.common.watch}
            </a>
            <LangMenu />
            <button
              onClick={toggleTheme}
              aria-label={t.header.toggleDark}
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
              {t.common.logIn}
            </a>
          </>
        )}
      </div>
    </header>
  )
}

const MODES: ThemeMode[] = ['light', 'dark', 'system']

/** A small menu of languages for guests; signed-in players have the row in their menu. */
function LangMenu() {
  const t = useT()
  const lang = useLang()
  return (
    <DropMenu
      title={t.header.language}
      trigger={
        <button
          aria-label={t.header.language}
          className="flex h-9 items-center rounded-md px-2.5 font-mono text-sm transition-colors hover:bg-muted"
        >
          {LANGS.find((l) => l.code === lang)?.short}
        </button>
      }
    >
      {LANGS.map((l) => (
        <MenuItem key={l.code} onSelect={() => setLang(l.code)}>
          <span className="w-7 font-mono text-[13px] text-accent">{l.short}</span>
          {l.name}
          {l.code === lang && <Icon name="check" className="ml-auto" />}
        </MenuItem>
      ))}
    </DropMenu>
  )
}

/** A row of choices in the menu: the theme or the language. */
function Seg<T extends string>({
  label,
  value,
  options,
  onChange,
}: {
  label: string
  value: T
  options: [T, string][]
  onChange: (v: T, e: React.MouseEvent) => void
}) {
  return (
    <div className="flex items-center justify-between gap-2 px-2.5 py-1.5 text-sm">
      <span>{label}</span>
      <div role="radiogroup" aria-label={label} className="flex gap-0.5 rounded-lg bg-muted p-[3px]">
        {options.map(([v, text]) => (
          <button
            key={v}
            role="radio"
            aria-checked={value === v}
            onClick={(e) => onChange(v, e)}
            className="h-7 rounded-[5px] px-2.5 text-[13px] transition-colors hover:bg-fg/10 aria-checked:bg-fg aria-checked:text-bg"
          >
            {text}
          </button>
        ))}
      </div>
    </div>
  )
}

function UserMenu({ username, rating }: { username: string; rating: number }) {
  const t = useT()
  const lang = useLang()
  const [mode, setMode] = useState(themeMode)
  const { unreadChats } = useCounts()
  return (
    <DropMenu
      title={t.header.yourMenu}
      trigger={
        <button
          aria-label={t.header.yourMenu}
          className="rounded-full p-0.5 transition-shadow outline-none hover:shadow-[0_0_0_2px_var(--color-fg)] focus-visible:shadow-[0_0_0_2px_var(--color-fg)] data-[state=open]:shadow-[0_0_0_2px_var(--color-fg)]"
        >
          <span className="relative block">
            <Avatar size={34} />
            {unreadChats > 0 && (
              <span className="absolute -top-0.5 -right-0.5 size-2.5 rounded-full bg-fg shadow-[0_0_0_2px_var(--color-bg)]" />
            )}
          </span>
        </button>
      }
    >
      <div className="flex items-center gap-3 px-2.5 pt-3 pb-3.5">
        <Avatar size={44} />
        <div className="flex min-w-0 flex-col">
          <span className="truncate text-[17px] font-semibold">{username}</span>
          <span className="font-mono text-sm text-accent tabular-nums">{rating}</span>
        </div>
      </div>
      <MenuSeparator />
      <MenuItem onSelect={() => navigate(`/u/${username}`)}>
        <Icon name="user" />
        {t.common.profile}
      </MenuItem>
      <MenuItem onSelect={() => navigate('/friends')}>
        <Icon name="people" />
        {t.common.friends}
      </MenuItem>
      <MenuItem onSelect={() => navigate('/messages')}>
        <Icon name="chat" />
        {t.common.messages}
        {unreadChats > 0 && (
          <span className="ml-auto grid h-5 min-w-5 place-items-center rounded-full bg-fg px-1.5 text-[11px] font-bold text-bg">
            {unreadChats}
          </span>
        )}
      </MenuItem>
      <MenuItem onSelect={() => navigate('/watch')}>
        <Icon name="eye" />
        {t.common.watch}
      </MenuItem>
      <MenuItem onSelect={() => navigate('/settings')}>
        <Icon name="gear" />
        {t.common.settings}
      </MenuItem>
      <MenuSeparator />
      <Seg
        label={t.header.theme}
        value={mode}
        options={MODES.map((m) => [m, t.header[m]])}
        onChange={(m, e) => {
          setThemeMode(m, e)
          setMode(m)
        }}
      />
      <Seg
        label={t.header.language}
        value={lang}
        options={LANGS.map((l) => [l.code, l.short])}
        onChange={(l) => setLang(l)}
      />
      <MenuSeparator />
      <MenuItem
        onSelect={async () => {
          await logout()
          navigate('/')
        }}
      >
        <Icon name="out" />
        {t.common.logOut}
      </MenuItem>
    </DropMenu>
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
