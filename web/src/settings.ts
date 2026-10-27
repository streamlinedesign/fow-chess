export type Settings = {
  /** Hide every square your pieces can't see. */
  fog: boolean
  /** No check: win by capturing the king. */
  kingCapture: boolean
}

const DEFAULTS: Settings = { fog: true, kingCapture: true }

export function loadSettings(): Settings {
  try {
    return { ...DEFAULTS, ...JSON.parse(localStorage.getItem('settings') ?? '{}') }
  } catch {
    return DEFAULTS
  }
}

export function saveSettings(s: Settings) {
  try {
    localStorage.setItem('settings', JSON.stringify(s))
  } catch {}
}

export function describeSettings(s: Settings) {
  return [s.fog ? 'Fog of war' : 'Full board', s.kingCapture ? 'Capture the king' : 'Checkmate']
}
