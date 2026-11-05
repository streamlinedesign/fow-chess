import type { ReactNode } from 'react'
import { BugNote } from './BugReport'
import { useT } from './i18n'

/**
 * The pages share the home page's shape: one big board, full width like the
 * header, rows of cells split by hairlines. A page starts with a head cell
 * (title, a line under it, actions at the right), then whatever rows it needs.
 * Cells take `home-cell`; a row of them takes `home-row` and a grid template.
 */
export function Page({ children, className = '' }: { children: ReactNode; className?: string }) {
  return <main className={`home ${className}`}>{children}</main>
}

export function PageHead({
  title,
  count,
  sub,
  right,
}: {
  title: ReactNode
  /** A number beside the title, like how many games are on. */
  count?: number
  sub?: ReactNode
  right?: ReactNode
}) {
  return (
    <div className="home-row">
      <div className="home-cell page-top">
        <div className="min-w-0">
          <h1 className="home-title">
            {title}
            {count !== undefined && count > 0 && <span className="count">{count}</span>}
          </h1>
          {sub && <p className="mt-2 max-w-[60ch] text-accent">{sub}</p>}
        </div>
        {right}
      </div>
    </div>
  )
}

/** A cell with a section heading, and a link or note at the right of it. */
export function Cell({
  title,
  right,
  children,
  className = '',
}: {
  title?: ReactNode
  right?: ReactNode
  children?: ReactNode
  className?: string
}) {
  return (
    <div className={`home-cell ${className}`}>
      {title && (
        <div className="mb-[18px] flex items-baseline justify-between gap-3">
          <h2 className="text-[22px] font-semibold tracking-tight">{title}</h2>
          {right}
        </div>
      )}
      {children}
    </div>
  )
}

/** The site's footer: who made it, and a way to report bugs. */
export function Footer() {
  const t = useT()
  return (
    <footer className="site-foot">
      <span>
        © 2026{' '}
        <a href="https://streamlinedesign.github.io/homepage" target="_blank" rel="noopener">
          Streamline
        </a>
      </span>
      <BugNote />
      <span className="text-accent">{t.common.appName}</span>
    </footer>
  )
}
