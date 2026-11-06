/** The desktop app (desktop/ in the repo) and the downloads for it. */

declare global {
  interface Window {
    /** Set by the desktop app on every page it loads. */
    __ETK_DESKTOP__?: { version: string; server: string }
  }
}

/** The app's version when the site runs inside the desktop app, else null. */
export const DESKTOP: { version: string } | null = (typeof window !== 'undefined' && window.__ETK_DESKTOP__) || null

export type OS = 'windows' | 'mac' | 'linux' | 'phone' | 'other'

/** A best guess at the visitor's system, for the big download button. */
export function detectOS(): OS {
  const ua = navigator.userAgent
  if (/Android|iPhone|iPad|iPod/i.test(ua) || (/Macintosh/.test(ua) && navigator.maxTouchPoints > 1)) return 'phone'
  if (/Windows/i.test(ua)) return 'windows'
  if (/Macintosh|Mac OS X/i.test(ua)) return 'mac'
  if (/Linux|X11|CrOS/i.test(ua)) return 'linux'
  return 'other'
}

export const OS_NAME: Record<'windows' | 'mac' | 'linux', string> = { windows: 'Windows', mac: 'Mac', linux: 'Linux' }

export const RELEASES = 'https://github.com/streamlinedesign/fow-chess/releases'

/**
 * The builds the release workflow (.github/workflows/desktop.yml) attaches to
 * every release, under these fixed names so the links always point at the
 * latest one.
 */
export const BUILDS = [
  { os: 'windows', kind: 'windows', file: 'EatTheKing-windows-x64.msi', ext: '.msi' },
  { os: 'mac', kind: 'mac', file: 'EatTheKing-mac-universal.dmg', ext: '.dmg' },
  { os: 'linux', kind: 'appimage', file: 'EatTheKing-linux-x86_64.AppImage', ext: '.AppImage' },
  { os: 'linux', kind: 'deb', file: 'EatTheKing-linux-amd64.deb', ext: '.deb' },
] as const

export const downloadUrl = (file: string) => `${RELEASES}/latest/download/${file}`
