export type ThemeMode = 'light' | 'dark' | 'system'

const dark = matchMedia('(prefers-color-scheme: dark)')

function paint(theme: 'light' | 'dark') {
  document.documentElement.dataset.theme = theme
  document.querySelector('meta[name=theme-color]')?.setAttribute('content', theme === 'dark' ? '#050505' : '#fffabd')
}

/** The saved choice; "system" when none is saved and the page follows the device. */
export function themeMode(): ThemeMode {
  try {
    const t = localStorage.getItem('theme')
    if (t === 'light' || t === 'dark') return t
  } catch {}
  return 'system'
}

// With no saved choice, follow the device as it switches between light and dark.
dark.addEventListener('change', () => themeMode() === 'system' && paint(dark.matches ? 'dark' : 'light'))

export function setThemeMode(mode: ThemeMode, e?: { clientX: number; clientY: number }) {
  const next = mode === 'system' ? (dark.matches ? 'dark' : 'light') : mode
  const apply = () => {
    paint(next)
    try {
      if (mode === 'system') localStorage.removeItem('theme')
      else localStorage.setItem('theme', mode)
    } catch {}
  }

  const root = document.documentElement
  const reduce = matchMedia('(prefers-reduced-motion: reduce)').matches
  if (!e || !document.startViewTransition || reduce || root.dataset.theme === next) return apply()

  // Circle reveal from where the choice was made.
  const { clientX: x, clientY: y } = e
  const r = Math.hypot(Math.max(x, innerWidth - x), Math.max(y, innerHeight - y))
  document.startViewTransition(apply).ready.then(() => {
    root.animate(
      { clipPath: [`circle(0px at ${x}px ${y}px)`, `circle(${r}px at ${x}px ${y}px)`] },
      { duration: 450, easing: 'cubic-bezier(0.4, 0, 0.2, 1)', pseudoElement: '::view-transition-new(root)' },
    )
  })
}

export function toggleTheme(e: React.MouseEvent) {
  setThemeMode(document.documentElement.dataset.theme === 'dark' ? 'light' : 'dark', e)
}
