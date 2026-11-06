import { useEffect, useState } from 'react'
import { BUILDS, detectOS, downloadUrl, OS_NAME, RELEASES } from './desktop'
import { useT } from './i18n'
import { Button } from './ui'

/** Chrome's install prompt, kept from the moment it fires so a button can show it. */
type InstallEvent = Event & { prompt: () => Promise<void> }
let deferred: InstallEvent | null = null
const waiting = new Set<() => void>()
if (typeof window !== 'undefined')
  window.addEventListener('beforeinstallprompt', (e) => {
    e.preventDefault()
    deferred = e as InstallEvent
    waiting.forEach((f) => f())
  })

const icons = {
  windows: (
    <path d="M2 4l7-1v6.5H2zM10 2.8 18 2v7.5h-8zM2 10.5h7V17l-7-1zM10 10.5h8V18l-8-1.2z" fill="currentColor" />
  ),
  mac: (
    <path
      d="M13.6 10.6c0-2 1.6-2.9 1.7-3-1-1.4-2.4-1.6-2.9-1.6-1.2-.1-2.4.7-3 .7s-1.6-.7-2.6-.7c-1.3 0-2.6.8-3.3 2-1.4 2.4-.4 6 1 8 .7.9 1.4 2 2.4 1.9 1-.1 1.3-.6 2.5-.6s1.5.6 2.5.6c1 0 1.7-1 2.3-1.9.7-1.1 1-2.1 1-2.2 0 0-2-.8-1.6-3.2zM11.7 4.6c.5-.7.9-1.6.8-2.6-.8 0-1.8.5-2.4 1.2-.5.6-1 1.6-.8 2.5.9.1 1.8-.4 2.4-1.1z"
      fill="currentColor"
    />
  ),
  linux: (
    <path
      d="M10 2c-2 0-3 1.6-3 3.6 0 1.2.2 2-.6 3.2C5.5 10.2 4 12 4 14.2c0 .6.2 1 .5 1.3-.5.4-.7.9-.4 1.4.4.7 1.6.6 2.6 1 .9.3 1.6.4 2-.2.2.1.8.1 1.6.1s1.4 0 1.6-.1c.4.6 1.1.5 2-.2 1-.4 2.2-.3 2.6-1 .3-.5.1-1-.4-1.4.3-.3.5-.7.5-1.3 0-2.2-1.5-4-2.4-5.4-.8-1.2-.6-2-.6-3.2C13 3.6 12 2 10 2z"
      fill="currentColor"
    />
  ),
}

export function DownloadIcon({ size = 18 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 20 20" aria-hidden>
      <path
        d="M10 3v10m0 0-4-4m4 4 4-4M4 16h12"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
}

function OsIcon({ os }: { os: 'windows' | 'mac' | 'linux' }) {
  return (
    <svg width="18" height="18" viewBox="0 0 20 20" aria-hidden>
      {icons[os]}
    </svg>
  )
}

/** /download: the desktop app for this computer, every build, and how to install on a phone. */
export function DownloadPage() {
  const t = useT()
  const os = detectOS()
  const phone = os === 'phone'
  const main = BUILDS.find((b) => b.os === os)
  const [canInstall, setCanInstall] = useState(!!deferred)
  useEffect(() => {
    const f = () => setCanInstall(true)
    waiting.add(f)
    return () => void waiting.delete(f)
  }, [])

  const list = (
    <div className="dl-list">
      {BUILDS.map((b) => (
        <a key={b.file} href={downloadUrl(b.file)} className="dl-row">
          <span className="dl-os">
            <OsIcon os={b.os} />
            {OS_NAME[b.os]}
          </span>
          <span className="dl-kind">
            <span className="max-sm:hidden">{t.desktop.kinds[b.kind]} </span>
            <span className="font-mono text-[13px] text-accent">{b.ext}</span>
          </span>
          <span className="dl-get">
            <DownloadIcon size={16} />
            <span className="max-sm:hidden">{t.desktop.download}</span>
          </span>
        </a>
      ))}
    </div>
  )

  if (phone)
    return (
      <main className="dl">
        <section className="dl-hero">
          <h1>{t.desktop.phoneTitle}</h1>
          <p className="dl-lead">{t.desktop.phoneHero}</p>
          {canInstall && (
            <Button variant="solid" className="dl-big w-full" onClick={() => deferred?.prompt()}>
              {t.desktop.addToHome}
            </Button>
          )}
          <p className="text-sm text-accent">{t.desktop.iphoneHint}</p>
        </section>
        <section className="dl-cell">
          <h2>{t.desktop.onComputer}</h2>
          {list}
        </section>
      </main>
    )

  return (
    <main className="dl">
      <section className="dl-hero">
        <h1>{t.desktop.title}</h1>
        <p className="dl-lead">{t.desktop.about}</p>
        {main && (
          <div>
            <a href={downloadUrl(main.file)} className="dl-big">
              <DownloadIcon size={22} />
              {t.desktop.downloadFor(OS_NAME[main.os])}
            </a>
            <p className="mt-3 font-mono text-[13px] text-accent">
              {t.desktop.kinds[main.kind]} {main.ext}
            </p>
          </div>
        )}
        <p className="dl-note">
          <b>{t.desktop.firstTitle}</b> {t.desktop.firstText}
        </p>
      </section>
      <div className="dl-two">
        <section className="dl-cell">
          <h2>{t.desktop.all}</h2>
          {list}
          <a href={RELEASES} target="_blank" rel="noopener" className="mt-3 block text-sm text-accent hover:text-fg">
            {t.desktop.older}
          </a>
        </section>
        <section className="dl-cell">
          <h2>{t.desktop.phone}</h2>
          <p className="text-accent">{t.desktop.phoneAbout}</p>
          <h2 className="mt-9">{t.desktop.different}</h2>
          <p className="text-accent">{t.desktop.differentAbout}</p>
        </section>
      </div>
    </main>
  )
}
