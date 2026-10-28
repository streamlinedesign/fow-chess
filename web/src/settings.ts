import type { Dict } from './i18n'

/** A clock setting. All zero means no clock. Must match timeControls in server/view.go. */
export type TimeControl = {
  /** Seconds per player. */
  base: number
  /** Seconds added after each move. */
  inc: number
  /** Seconds for every move, reset after each one (Daily). */
  perMove: number
}

export type Settings = {
  /** Hide every square your pieces can't see. */
  fog: boolean
  /** No check: win by capturing the king. */
  kingCapture: boolean
  time: TimeControl
  /** How the pieces start: the usual position, or one of the setup modes. Must match setupModes in server/setup.go. */
  setup?: Setup
  /** Kings can also jump like a knight. */
  jumpKing?: boolean
  /** Pawns can also step one square straight back, and see behind them. */
  backPawns?: boolean
  /** King of the Hill: a king on one of the four centre squares wins. */
  hill?: boolean
  /** Play a game that could be rated without rating it. */
  unrated?: boolean
}

export type Setup = '' | 'backRow' | 'battleship'

/** The hill in King of the Hill: the four centre squares. */
export const HILL = ['d4', 'e4', 'd5', 'e5']

/** The setup modes: how many of your own rows you arrange, and whether the pawns stay put. Names are in the dictionary. */
export const SETUPS: { value: Setup; rows: number; pawnsFixed: boolean }[] = [
  { value: '', rows: 0, pawnsFixed: false },
  { value: 'backRow', rows: 1, pawnsFixed: true },
  { value: 'battleship', rows: 4, pawnsFixed: false },
]

export const setupMode = (s: Settings) => SETUPS.find((m) => m.value === (s.setup ?? '')) ?? SETUPS[0]

const tc = (base: number, inc = 0, perMove = 0): TimeControl => ({ base, inc, perMove })

const DAY = 24 * 60 * 60

/** The presets players pick from, by group. Must match timeControls in server/view.go. */
export const TIME_GROUPS: [string, TimeControl[]][] = [
  ['Blitz', [tc(300), tc(180, 2)]],
  ['Rapid', [tc(600)]],
  ['Tournament', [tc(5400, 30)]],
  ['Daily', [tc(0, 0, DAY)]],
  ['Weekly', [tc(0, 0, 7 * DAY)]],
]

export const NO_CLOCK = tc(0)
const DEFAULTS: Settings = {
  fog: true,
  kingCapture: true,
  time: tc(300),
  setup: '',
  unrated: false,
}

export const sameTime = (a: TimeControl, b: TimeControl) =>
  a.base === b.base && a.inc === b.inc && a.perMove === b.perMove

export const clocked = (t: TimeControl) => !sameTime(t, NO_CLOCK)

/** The group a time control belongs to, like "Blitz". Clocks no longer offered fall in a group of their own. */
export function timeGroup(t: TimeControl) {
  const known = TIME_GROUPS.find(([, presets]) => presets.some((p) => sameTime(p, t)))?.[0]
  if (known) return known
  if (t.perMove) return 'Daily'
  return clocked(t) ? 'Other' : 'No clock'
}

/** "5+0" for a clock, "24 h per move" or "7 days per move" for a reset clock, "No clock" otherwise. */
export function timeLabel(t: TimeControl, d: Dict) {
  if (t.perMove >= 2 * DAY && t.perMove % DAY === 0) return d.time.perMoveDays(t.perMove / DAY)
  if (t.perMove) return d.time.perMoveHours(Math.round(t.perMove / 3600))
  if (!t.base) return d.time.noClock
  return `${t.base / 60}+${t.inc}`
}

/** "Blitz 5+0", "24 h per move" or "No clock". */
export function describeTime(t: TimeControl, d: Dict) {
  const group = timeGroup(t)
  if (t.perMove || !clocked(t)) return timeLabel(t, d)
  return group === 'Other' ? timeLabel(t, d) : `${d.time.groups[group]} ${timeLabel(t, d)}`
}

export function loadSettings(): Settings {
  try {
    const saved = { ...DEFAULTS, ...JSON.parse(localStorage.getItem('settings') ?? '{}') }
    // Settings saved before clocks existed, or a preset that's since gone, fall back to the default.
    const known = TIME_GROUPS.some(([, presets]) => presets.some((p) => saved.time && sameTime(p, saved.time)))
    if (!SETUPS.some((m) => m.value === saved.setup)) saved.setup = ''
    // The site plays one game now; twists saved earlier are dropped.
    delete saved.jumpKing
    delete saved.backPawns
    delete saved.hill
    saved.fog = true
    saved.kingCapture = true
    return known ? saved : { ...saved, time: DEFAULTS.time }
  } catch {
    return DEFAULTS
  }
}

export function saveSettings(s: Settings) {
  try {
    localStorage.setItem('settings', JSON.stringify(s))
  } catch {}
}

/**
 * What makes a game different from the plain one, as short labels: the setup,
 * and for games saved before the site became one game, the old modes too.
 */
export function twists(s: Settings, d: Dict) {
  return [
    ...(setupMode(s).value ? [d.setups[setupMode(s).value].label] : []),
    ...(s.fog === false ? [d.legacy.fullBoard] : []),
    ...(s.kingCapture === false ? [d.legacy.checkmate] : []),
    ...(s.hill ? [d.legacy.hill] : []),
    ...(s.jumpKing ? [d.legacy.jump] : []),
    ...(s.backPawns ? [d.legacy.backPawns] : []),
  ]
}

/** A game's kind in a few words for a card: "Battleship", or "Standard" for the plain game. */
export function modeName(s: Settings, d: Dict) {
  const t = twists(s, d)
  return t.length ? t.join(' · ') : d.setups[''].label
}

/** The tags on a match page: what's special about it, then the clock. Games saved before clocks existed had none. */
export function describeSettings(s: Settings, d: Dict) {
  return [...twists(s, d), describeTime(s.time ?? NO_CLOCK, d)]
}
