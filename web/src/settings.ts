import type { Dict } from './i18n'
import type { Look, Skin } from './pieces'
import { setImg } from './sets'

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
  /** Pawns can also step and take one square in any direction, like a king. */
  jesterPawns?: boolean
  /** King of the Hill: a king on one of the four centre squares wins. */
  hill?: boolean
  /** Centre vision: a piece on the centre of the opponent's half shows you the whole board. Every game has 'any'; mods games may turn it off; 'pawn' is from early demo games. Must match centerVisions in server/view.go. */
  centerVision?: CenterVision
  /** Sync Move: everyone moves at once each round. Always Eat the King. Mods only. */
  sync?: boolean
  /** Four players only: the extended board, 16 by 16 with neutral corners. Sent as board. */
  fourBoard?: '' | 'extended'
  /** Never ranked: set by the server for bot and mods games. Players can't ask for it. */
  unrated?: boolean
  /** From the mods page: any option, any clock, never ranked. */
  mods?: boolean
  /** A computer opponent's level. Never ranked. Must match Levels in server/bots. */
  bot?: Bot
  /** The arena this game is part of. */
  tournament?: number
}

export type Bot = 'rex' | 'vercingetorix' | 'ramesses' | 'solomon'

/**
 * The themed set each bot plays with, on its own tiles: the light half as
 * White, the dark half as Black (see concrete in sets.ts). Beating a bot
 * unlocks its theme (THEME_LOCK in sets.ts). Rex inutilis has none: he plays
 * the classic pieces and unlocks nothing.
 */
export const BOT_SET: Partial<Record<Bot, string>> = {
  vercingetorix: 'survey',
  ramesses: 'ancient',
  solomon: 'galaxy',
}

/** Each bot's face: a piece of its strength from its own set. J is the jester pawn, X the prince. */
export const BOT_FACE: Record<Bot, string> = {
  rex: setImg('ink', 'J'), // Rex inutilis: the classic jester pawn
  vercingetorix: setImg('survey-dark', 'N'),
  ramesses: setImg('ancient-dark', 'X'),
  solomon: setImg('galaxy-dark', 'K'),
}

/** The bot that unlocks a theme, if a bot does. */
export function botFor(theme: string): Bot | undefined {
  return (Object.keys(BOT_SET) as Bot[]).find((b) => BOT_SET[b] === theme)
}

/**
 * How a game's pieces are drawn: its mods, a bot's pieces in the bot's set,
 * and each player's own skin as the server passes it on.
 */
export function gameLook(
  settings: (Look & { bot?: Bot; demo?: boolean }) | undefined,
  botSide?: 'w' | 'b' | null,
  players?: Partial<Record<'w' | 'b', { skin?: Skin }>>,
): Look | undefined {
  if (!settings) return settings
  const look: Look = { ...settings }
  // The home page board's Solomon plays the default pieces; bots wear their set only in a game started from their card.
  const botSet = settings.bot && !settings.demo && BOT_SET[settings.bot]
  if (botSet && botSide) {
    look.sets = { [botSide]: botSet }
    look.tiles = { [botSide]: botSet }
  }
  const w = players?.w?.skin
  const b = players?.b?.skin
  if (w || b) look.skins = { w, b }
  return look
}

/** The computer opponents, weakest first. Their names are the same in every language. */
export const BOTS: { level: Bot; name: string }[] = [
  { level: 'rex', name: 'Rex inutilis' },
  { level: 'vercingetorix', name: 'Vercingetorix' },
  { level: 'ramesses', name: 'Ramesses II' },
  { level: 'solomon', name: 'Solomon' },
]

export type CenterVision = '' | 'pawn' | 'any'

/** The squares that give centre vision: the centre of the opponent's half. Must match engine.Outposts. */
export const OUTPOSTS: Record<'w' | 'b', string[]> = { w: ['d5', 'e5'], b: ['d4', 'e4'] }

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

/**
 * War Mode's clocks: 30 minutes plus 10 seconds a move, 2 hours plus 30
 * seconds a move, and a day for each move. Must match warTimes in
 * server/four_match.go.
 */
export const WAR_TIMES: TimeControl[] = [tc(1800, 10), tc(7200, 30), tc(0, 0, DAY)]

/** The War Mode clock last picked, for the queue and for games with friends. */
export function loadWarTime(): TimeControl {
  try {
    const t = JSON.parse(localStorage.getItem('warTime') ?? 'null')
    return WAR_TIMES.find((p) => t && sameTime(p, t)) ?? WAR_TIMES[0]
  } catch {
    return WAR_TIMES[0]
  }
}

export function saveWarTime(t: TimeControl) {
  try {
    localStorage.setItem('warTime', JSON.stringify(t))
  } catch {}
}

/**
 * The extra clock a mods game can have: none. The server's modTimeControls
 * (server/view.go) still accepts the longer list the mods page used to offer.
 */
export const MOD_TIMES: TimeControl[] = [tc(0)]

export const NO_CLOCK = tc(0)
const DEFAULTS: Settings = {
  fog: true,
  kingCapture: true,
  time: tc(300),
  setup: '',
  centerVision: 'any',
}

export const sameTime = (a: TimeControl, b: TimeControl) =>
  a.base === b.base && a.inc === b.inc && a.perMove === b.perMove

export const clocked = (t: TimeControl) => !sameTime(t, NO_CLOCK)

/** The group a time control belongs to, like "Blitz". */
export function timeGroup(t: TimeControl) {
  const known = TIME_GROUPS.find(([, presets]) => presets.some((p) => sameTime(p, t)))?.[0]
  if (known) return known
  if (t.perMove) return 'Daily'
  if (!clocked(t)) return 'No clock'
  // The extra mods clocks, by the usual names.
  return t.base < 180 ? 'Bullet' : t.base < 600 ? 'Blitz' : t.base < 1800 ? 'Rapid' : 'Classical'
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
  if (t.perMove || !clocked(t)) return timeLabel(t, d)
  return `${d.time.groups[timeGroup(t)]} ${timeLabel(t, d)}`
}

/** The clocks a game may have: the presets, plus the extra ones for a mods game. */
export const clocksFor = (mods: boolean) => [
  ...TIME_GROUPS.flatMap(([, presets]) => presets),
  ...(mods ? MOD_TIMES : []),
]

/** The saved settings for the home page, or with mods for the mods page. Each remembers its own. */
export function loadSettings(mods = false): Settings {
  try {
    const saved = { ...DEFAULTS, ...JSON.parse(localStorage.getItem(mods ? 'mods' : 'settings') ?? '{}') }
    // Settings saved before clocks existed, or a preset that's since gone, fall back to the default.
    const known = clocksFor(mods).some((p) => saved.time && sameTime(p, saved.time))
    if (!SETUPS.some((m) => m.value === saved.setup)) saved.setup = ''
    // Backward pawns were a mod once; settings saved with them drop them.
    delete saved.backPawns
    if (mods) {
      saved.mods = true
      saved.unrated = true
    } else {
      // The site plays one game; twists saved earlier are dropped.
      delete saved.jumpKing
      delete saved.jesterPawns
      delete saved.hill
      delete saved.sync
      delete saved.fourBoard
      saved.setup = ''
      saved.centerVision = 'any'
      delete saved.mods
      delete saved.bot
      delete saved.unrated
      saved.fog = true
      saved.kingCapture = true
    }
    return known ? saved : { ...saved, time: DEFAULTS.time }
  } catch {
    return mods ? { ...DEFAULTS, mods: true, unrated: true } : DEFAULTS
  }
}

export function saveSettings(s: Settings) {
  try {
    localStorage.setItem(s.mods ? 'mods' : 'settings', JSON.stringify(s))
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
    ...(s.jesterPawns ? [d.legacy.jesterPawns] : []),
    ...(s.sync ? [d.four.sync] : []),
    // Centre vision for any piece is the game; mods games can turn it off, and early demo games counted only pawns.
    ...(s.centerVision === 'pawn' ? [d.legacy.centerPawn] : s.mods && !s.centerVision ? [d.legacy.noCenter] : []),
  ]
}

/** A game's kind in a few words for a card: "Battleship", or "Standard" for the plain game. */
export function modeName(s: Settings, d: Dict) {
  const t = twists(s, d)
  return t.length ? t.join(' · ') : d.setups[''].label
}

/** The tags on a match page: what's special about it, then the clock. Games saved before clocks existed had none. */
export function describeSettings(s: Settings, d: Dict) {
  return [
    ...(s.tournament ? [d.common.arena] : []),
    ...(s.mods ? [d.common.mods] : []),
    ...twists(s, d),
    describeTime(s.time ?? NO_CLOCK, d),
  ]
}
