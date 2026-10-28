import { createElement, Fragment, useSyncExternalStore, type ReactNode } from 'react'
import { en } from './en'
import { hr } from './hr'

/** The shape every language must fill in, taken from English. */
export type Dict = typeof en

export type Lang = 'en' | 'hr'

/** The languages on offer, with their names in themselves. Add a dictionary and a row here for a new one. */
export const LANGS: { code: Lang; name: string; short: string }[] = [
  { code: 'en', name: 'English', short: 'EN' },
  { code: 'hr', name: 'Hrvatski', short: 'HR' },
]

const dicts: Record<Lang, Dict> = { en, hr }

function isLang(s: unknown): s is Lang {
  return LANGS.some((l) => l.code === s)
}

/** The saved choice, or the first of the browser's languages we have, or English. */
function initial(): Lang {
  try {
    const saved = localStorage.getItem('lang')
    if (isLang(saved)) return saved
  } catch {}
  for (const tag of navigator.languages ?? [navigator.language]) {
    const code = tag.slice(0, 2).toLowerCase()
    if (isLang(code)) return code
  }
  return 'en'
}

let lang: Lang = initial()
const listeners = new Set<() => void>()

function apply() {
  document.documentElement.lang = lang
}
apply()

export function currentLang() {
  return lang
}

/** The current dictionary, for code outside React. Components use useT(), which re-renders on a change. */
export function dict(): Dict {
  return dicts[lang]
}

export function setLang(next: Lang) {
  if (next === lang) return
  lang = next
  try {
    localStorage.setItem('lang', next)
  } catch {}
  apply()
  listeners.forEach((l) => l())
}

const subscribe = (cb: () => void) => {
  listeners.add(cb)
  return () => {
    listeners.delete(cb)
  }
}

export function useLang(): Lang {
  return useSyncExternalStore(subscribe, () => lang)
}

/** The strings for the current language. */
export function useT(): Dict {
  return useSyncExternalStore(subscribe, () => dicts[lang])
}

/** Renders **bold** spans in a string as <strong>. */
export function rich(text: string): ReactNode {
  const parts = text.split('**')
  if (parts.length === 1) return text
  return createElement(Fragment, null, ...parts.map((p, i) => (i % 2 ? createElement('strong', { key: i }, p) : p)))
}

/** Fills {name} placeholders in a template with React nodes, such as a bold username. */
export function fill(template: string, parts: Record<string, ReactNode>): ReactNode {
  const out: ReactNode[] = []
  const re = /\{(\w+)\}/g
  let last = 0
  let m: RegExpExecArray | null
  while ((m = re.exec(template))) {
    if (m.index > last) out.push(template.slice(last, m.index))
    out.push(createElement(Fragment, { key: m.index }, parts[m[1]] ?? m[0]))
    last = m.index + m[0].length
  }
  if (last < template.length) out.push(template.slice(last))
  return createElement(Fragment, null, ...out)
}

/** Croatian-style plural pick: 1 → one, 2–4 → few, the rest → many (with the teens as many). */
export function slavic(n: number, one: string, few: string, many: string) {
  const mod10 = n % 10
  const mod100 = n % 100
  if (mod10 === 1 && mod100 !== 11) return one
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)) return few
  return many
}

/** The message for a server error body ({error, code, name?}) in the current language, falling back to the server's English. */
export function errorText(body: unknown): string {
  const b = (body ?? {}) as { error?: string; code?: string; name?: string }
  const d = dict()
  const entry = b.code ? (d.errors as Record<string, string | ((name: string) => string)>)[b.code] : undefined
  if (typeof entry === 'function') return entry(b.name ?? '')
  if (typeof entry === 'string') return entry
  return b.error || d.common.somethingWrong
}
