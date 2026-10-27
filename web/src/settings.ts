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
}

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
const DEFAULTS: Settings = { fog: true, kingCapture: true, time: tc(300) }

export const sameTime = (a: TimeControl, b: TimeControl) =>
  a.base === b.base && a.inc === b.inc && a.perMove === b.perMove

export const clocked = (t: TimeControl) => !sameTime(t, NO_CLOCK)

/** The group a time control belongs to, like "Blitz". */
export function timeGroup(t: TimeControl) {
  return TIME_GROUPS.find(([, presets]) => presets.some((p) => sameTime(p, t)))?.[0] ?? 'No clock'
}

/** "5+0" for a clock, "24 h per move" for Daily, "No clock" otherwise. */
export function timeLabel(t: TimeControl) {
  if (t.perMove) return `${Math.round(t.perMove / 3600)} h per move`
  if (!t.base) return 'No clock'
  return `${t.base / 60}+${t.inc}`
}

/** "Blitz 5+0", "Daily" or "No clock". */
export function describeTime(t: TimeControl) {
  const group = timeGroup(t)
  return group === 'Daily' || group === 'No clock' ? group : `${group} ${timeLabel(t)}`
}

export function loadSettings(): Settings {
  try {
    const saved = { ...DEFAULTS, ...JSON.parse(localStorage.getItem('settings') ?? '{}') }
    // Settings saved before clocks existed, or a preset that's since gone, fall back to the default.
    const known = TIME_GROUPS.some(([, presets]) => presets.some((p) => saved.time && sameTime(p, saved.time)))
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

/** Games saved before clocks existed have no time setting; those had no clock. */
export function describeSettings(s: Settings) {
  return [
    s.fog ? 'Fog of war' : 'Full board',
    s.kingCapture ? 'Eat the King' : 'Checkmate',
    describeTime(s.time ?? NO_CLOCK),
  ]
}
