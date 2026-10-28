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

/** The presets players pick from, by group. */
export const TIME_GROUPS: [string, TimeControl[]][] = [
  ['Bullet', [tc(60), tc(120, 1)]],
  ['Blitz', [tc(180), tc(180, 2), tc(300), tc(300, 3)]],
  ['Rapid', [tc(600), tc(600, 5), tc(900, 10)]],
  ['Classical', [tc(1800), tc(1800, 20), tc(5400, 30)]],
  ['Daily', [tc(0, 0, 24 * 60 * 60)]],
  ['No clock', [tc(0)]],
]

export const NO_CLOCK = tc(0)
const DEFAULTS: Settings = {
  fog: true,
  kingCapture: true,
  time: tc(300),
  setup: '',
  jumpKing: false,
  backPawns: false,
  hill: false,
  unrated: false,
}

export const sameTime = (a: TimeControl, b: TimeControl) =>
  a.base === b.base && a.inc === b.inc && a.perMove === b.perMove

export const clocked = (t: TimeControl) => !sameTime(t, NO_CLOCK)

/** The group a time control belongs to, like "Blitz". */
export function timeGroup(t: TimeControl) {
  return TIME_GROUPS.find(([, presets]) => presets.some((p) => sameTime(p, t)))?.[0] ?? 'No clock'
}

/** "5+0" for a clock, "24 h per move" for Daily, "No clock" otherwise. */
export function timeLabel(t: TimeControl, d: Dict) {
  if (t.perMove) return d.time.perMove(Math.round(t.perMove / 3600))
  if (!t.base) return d.time.noClock
  return `${t.base / 60}+${t.inc}`
}

/** "Blitz 5+0", "Daily" or "No clock". */
export function describeTime(t: TimeControl, d: Dict) {
  const group = timeGroup(t)
  return group === 'Daily' || group === 'No clock' ? d.time.groups[group] : `${d.time.groups[group]} ${timeLabel(t, d)}`
}

export function loadSettings(): Settings {
  try {
    const saved = { ...DEFAULTS, ...JSON.parse(localStorage.getItem('settings') ?? '{}') }
    // Settings saved before clocks existed, or a preset that's since gone, fall back to the default.
    const known = TIME_GROUPS.some(([, presets]) => presets.some((p) => saved.time && sameTime(p, saved.time)))
    if (!SETUPS.some((m) => m.value === saved.setup)) saved.setup = ''
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

/** The extra rules a game has on, as short labels. */
export function twists(s: Settings, d: Dict) {
  return [
    ...(s.hill ? [d.twists.hill] : []),
    ...(setupMode(s).value ? [d.setups[setupMode(s).value].label] : []),
    ...(s.jumpKing ? [d.twists.jump] : []),
    ...(s.backPawns ? [d.twists.backPawns] : []),
  ]
}

/** A game's kind in a few words for a card: "Eat the King · Fog · Battleship". */
export function modeName(s: Settings, d: Dict) {
  return [s.kingCapture ? d.panel.eat : d.panel.mate, ...(s.fog ? [d.panel.fogShort] : []), ...twists(s, d)].join(' · ')
}

/** Games saved before clocks existed have no time setting; those had no clock. */
export function describeSettings(s: Settings, d: Dict) {
  return [
    s.fog ? d.panel.fog : d.panel.fullBoard,
    s.kingCapture ? d.panel.eat : d.panel.mate,
    ...twists(s, d),
    describeTime(s.time ?? NO_CLOCK, d),
  ]
}
