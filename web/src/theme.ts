export function toggleTheme(e: React.MouseEvent) {
  const root = document.documentElement
  const next = root.dataset.theme === 'dark' ? 'light' : 'dark'
  const apply = () => {
    root.dataset.theme = next
    document.querySelector('meta[name=theme-color]')?.setAttribute('content', next === 'dark' ? '#050505' : '#fffabd')
    try {
      localStorage.setItem('theme', next)
    } catch {}
  }

  const reduce = matchMedia('(prefers-reduced-motion: reduce)').matches
  if (!document.startViewTransition || reduce) return apply()

  // Circle reveal from the toggle button.
  const { clientX: x, clientY: y } = e
  const r = Math.hypot(Math.max(x, innerWidth - x), Math.max(y, innerHeight - y))
  document.startViewTransition(apply).ready.then(() => {
    root.animate(
      { clipPath: [`circle(0px at ${x}px ${y}px)`, `circle(${r}px at ${x}px ${y}px)`] },
      { duration: 450, easing: 'cubic-bezier(0.4, 0, 0.2, 1)', pseudoElement: '::view-transition-new(root)' },
    )
  })
}
