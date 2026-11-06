import { DESKTOP } from './desktop'
import type { Settings, TimeControl } from './settings'

/** The site's address for links: the server the desktop app opens, else this page's. */
export function siteURL(path: string) {
  const server = DESKTOP ? window.__ETK_DESKTOP__?.server : ''
  return (server || location.origin).replace(/\/$/, '') + path
}

/**
 * Shares a link: the system share sheet on phones, else the clipboard.
 * Says which happened, or 'failed' (nothing to tell the player when they
 * close the sheet themselves).
 */
export async function shareLink(s: { title: string; text?: string; url: string }): Promise<'shared' | 'copied' | 'failed'> {
  try {
    if (navigator.share && matchMedia('(pointer: coarse)').matches) {
      await navigator.share(s)
      return 'shared'
    }
    if (navigator.clipboard && isSecureContext) await navigator.clipboard.writeText(s.url)
    else copyFallback(s.url) // the clipboard API needs HTTPS, which a phone on the LAN won't have
    return 'copied'
  } catch {
    return 'failed'
  }
}

function copyFallback(text: string) {
  const el = document.createElement('textarea')
  el.value = text
  el.style.position = 'fixed'
  el.style.opacity = '0'
  document.body.append(el)
  el.select()
  document.execCommand('copy')
  el.remove()
}

/*
 * A mods link: /mods with the switches in the query, which the server reads
 * for the link preview too (parseModsLink in server/share.go):
 *
 *   /mods?mod=prince&mod=opensetup&players=4&clock=10-5
 *
 * mod may repeat; clock is minutes-increment, 1d, 7d or none.
 */

const DAY = 24 * 60 * 60

function clockKey(t: TimeControl) {
  if (t.perMove) return t.perMove >= 7 * DAY ? '7d' : '1d'
  if (!t.base && !t.inc) return 'none'
  return `${t.base / 60}-${t.inc}`
}

function parseClock(s: string | null): TimeControl | null {
  if (s === 'none') return { base: 0, inc: 0, perMove: 0 }
  if (s === '1d') return { base: 0, inc: 0, perMove: DAY }
  if (s === '7d') return { base: 0, inc: 0, perMove: 7 * DAY }
  const m = /^(\d{1,3})-(\d{1,3})$/.exec(s ?? '')
  return m ? { base: Number(m[1]) * 60, inc: Number(m[2]), perMove: 0 } : null
}

/** The link that opens the Mods page with these switches on. */
export function modsLink(s: Settings, four: boolean) {
  const mods: string[] = []
  if (four && s.fourBoard === 'extended') mods.push('extended')
  if (s.sync) mods.push('sync')
  if (s.jumpKing) mods.push('prince')
  if (s.jesterPawns) mods.push('jester')
  if (s.setup === 'battleship') mods.push('opensetup')
  if (s.setup === 'backRow') mods.push('backrow')
  if (s.hill && !(four && s.fourBoard === 'extended')) mods.push('hill')
  if (!s.fog) mods.push('nofog')
  if (!s.kingCapture && !s.sync) mods.push('mate')
  if (!s.centerVision) mods.push('nocenter')
  const q = mods.map((m) => `mod=${m}`)
  if (four && s.fourBoard !== 'extended') q.push('players=4')
  q.push(`clock=${clockKey(s.time)}`)
  return siteURL(`/mods?${q.join('&')}`)
}

/**
 * The settings in a mods link, on top of the defaults: every switch named is
 * on, and the rest are the game's own. Null when the query names none of
 * them. The old single ?mod=jump links from the home page read the same way.
 */
export function modsFromLink(
  base: Settings,
  search: string,
  clocks: TimeControl[],
): { settings: Settings; four: boolean } | null {
  const q = new URLSearchParams(search)
  const mods = q.getAll('mod').map((m) => m.toLowerCase())
  const clock = parseClock(q.get('clock'))
  if (!mods.length && !q.has('players') && !clock) return null
  const s: Settings = {
    ...base,
    fog: true,
    kingCapture: true,
    centerVision: 'any',
    setup: '',
    jumpKing: false,
    jesterPawns: false,
    hill: false,
    sync: false,
    fourBoard: '',
  }
  let four = q.get('players') === '4'
  for (const m of mods) {
    if (m === 'extended') ((s.fourBoard = 'extended'), (four = true))
    if (m === 'sync') s.sync = true
    if (m === 'prince' || m === 'jump') s.jumpKing = true
    if (m === 'jester') s.jesterPawns = true
    // Open Setup was called Battleship, then Ambush.
    if (m === 'opensetup' || m === 'ambush' || m === 'battleship') s.setup = 'battleship'
    if (m === 'backrow') s.setup = 'backRow'
    if (m === 'hill') s.hill = true
    if (m === 'nofog') s.fog = false
    if (m === 'mate') s.kingCapture = false
    if (m === 'nocenter') s.centerVision = ''
  }
  if (s.sync) s.kingCapture = true
  // Only a clock the Mods page offers; any other keeps the one picked last.
  if (clock && clocks.some((c) => c.base === clock.base && c.inc === clock.inc && c.perMove === clock.perMove))
    s.time = clock
  return { settings: s, four }
}
