import * as Switch from '@radix-ui/react-switch'
import { memo, useEffect, useId, useMemo, useRef, useState } from 'react'
import { AvatarImg } from './Avatar'
import { Board } from './Board'
import { useT } from './i18n'
import type { Look } from './pieces'
import { FOG_STYLES, setPrefs, skinOf, usePrefs, useUnlocked, type FogStyle } from './prefs'
import { AVATAR_GROUPS, concrete, familyOf, KINDS, SET_IDS, setImg, THEME_LOCK, TILE_IDS, tileImg } from './sets'
import { Button } from './ui'
import { BOTS } from './settings'

// The pickers are plain radio buttons, not Radix: a page of them stays quick
// to open on a phone, and the arrow keys move through each group for free.
// Each picker is memoised, so a pick redraws only the picker that changed.

type Opt = { value: string; label: string; face: React.ReactNode }

/** Why an option can't be picked yet ("Beat Solomon"), or undefined when it can. */
type LockOf = (value: string) => string | undefined

const Radios = memo(function Radios({
  label,
  value,
  options,
  onChange,
  lockOf,
  className = 'set-pick',
  optClass = 'set-opt',
}: {
  label: string
  value: string
  options: Opt[]
  onChange: (v: string) => void
  lockOf?: LockOf
  className?: string
  optClass?: string
}) {
  const name = useId()
  return (
    <div role="radiogroup" aria-label={label} className={className}>
      {options.map((o) => {
        const lock = lockOf?.(o.value)
        return (
          <label
            key={o.value}
            className={optClass}
            data-state={o.value === value ? 'checked' : 'unchecked'}
            data-locked={lock ? true : undefined}
            title={lock}
          >
            <input
              type="radio"
              name={name}
              value={o.value}
              checked={o.value === value}
              onChange={() => onChange(o.value)}
              disabled={!!lock}
              className="sr-only"
              aria-label={lock ? `${o.label}, ${lock}` : o.label}
            />
            {o.face}
            {lock && optClass === 'set-opt' && (
              <span className="set-lock" aria-hidden>
                <LockIcon />
                {lock}
              </span>
            )}
            {lock && optClass !== 'set-opt' && (
              <span className="ava-lock" aria-hidden>
                <LockIcon />
              </span>
            )}
          </label>
        )
      })}
    </div>
  )
})

function LockIcon() {
  return (
    <svg viewBox="0 0 16 16" width="12" height="12" aria-hidden>
      <rect x="3" y="7" width="10" height="7" rx="1.5" fill="currentColor" />
      <path d="M5.5 7V5a2.5 2.5 0 0 1 5 0v2" fill="none" stroke="currentColor" strokeWidth="1.6" />
    </svg>
  )
}

/** Why each locked theme can't be worn yet, for this viewer; plain colours and unlocked themes have none. */
function useLockOf(): LockOf {
  const t = useT()
  const unlocked = useUnlocked()
  return useMemo(
    () => (value: string) => {
      const family = familyOf(value.split('/')[0])
      const reason = THEME_LOCK[family]
      if (!reason || !unlocked || unlocked.has(family)) return undefined
      if (reason === 'beta') return t.pieces.lockedBeta
      const bot = BOTS.find((b) => 'bot:' + b.level === reason)
      return bot ? t.pieces.lockedBy(bot.name.split(' ')[0]) : t.pieces.lockedBeta
    },
    [t, unlocked],
  )
}

/** A family's face in a picker: its knight in the half that side plays. */
function setOptions(t: ReturnType<typeof useT>, color: 'w' | 'b'): Opt[] {
  return SET_IDS.map((id) => ({
    value: id,
    label: t.pieces.sets[id] ?? id,
    face: (
      <>
        <span className="set-face" data-side={color} aria-hidden>
          <img src={setImg(concrete(id, color), 'N')} alt="" decoding="async" />
        </span>
        <span className="set-name">{t.pieces.sets[id] ?? id}</span>
      </>
    ),
  }))
}

function tileOptions(t: ReturnType<typeof useT>, light: boolean): Opt[] {
  return TILE_IDS.map((id) => ({
    value: id || 'plain',
    label: id ? t.pieces.sets[id] : t.pieces.plainTiles,
    face: (
      <>
        <span
          className="set-face tile"
          data-light={light}
          style={id ? { backgroundImage: `url("${tileImg(id, light)}")` } : undefined}
          aria-hidden
        />
        <span className="set-name">{id ? t.pieces.sets[id] : t.pieces.plainTiles}</span>
      </>
    ),
  }))
}

// Options depend only on the language, so they're built once per language.
const cache = new Map<unknown, Record<string, Opt[]>>()
function useOptions() {
  const t = useT()
  let o = cache.get(t)
  if (!o) {
    o = {
      w: setOptions(t, 'w'),
      b: setOptions(t, 'b'),
      lightTiles: tileOptions(t, true),
      darkTiles: tileOptions(t, false),
    }
    cache.set(t, o)
  }
  return o
}

const onWhite = (white: string) => setPrefs({ white })
const onBlack = (black: string) => setPrefs({ black })
const onLight = (v: string) => setPrefs({ lightTiles: v === 'plain' ? '' : v })
const onDark = (v: string) => setPrefs({ darkTiles: v === 'plain' ? '' : v })

const START = 'RNBQKBNRPPPPPPPP' + '.'.repeat(32) + 'pppppppprnbqkbnr'
// With the fog shown, the start as one side sees it: their own four ranks.
const FOGGED = { w: START.slice(0, 32) + '?'.repeat(32), b: '?'.repeat(32) + START.slice(32) }
const noMoves: string[] = []
const noop = () => {}

/**
 * The live preview: a full board at the start, both sides in your sets and on
 * your tiles, so every pick shows at once. Swap sides turns it round; the fog
 * button hides the far half in your fog style, vision squares ringed.
 */
export const LivePreview = memo(function LivePreview({ className = '' }: { className?: string }) {
  const t = useT()
  const p = usePrefs()
  const [side, setSide] = useState<'w' | 'b'>(p.side || 'w')
  const [fog, setFog] = useState(false)
  // Picking a fog style shows it.
  const style = useRef(p.fog)
  useEffect(() => {
    if (style.current !== p.fog) setFog(true)
    style.current = p.fog
  }, [p.fog])
  // Your skin on both halves, so the board shows your White set and your Black set together.
  const skin = skinOf(p)
  const look = useMemo<Look>(
    () => ({ skins: { w: skin, b: skin } }),
    [skin.white, skin.black, skin.lightTiles, skin.darkTiles],
  )
  const center = useMemo(() => (fog ? { you: side } : undefined), [fog, side])
  return (
    <div className={`live-preview ${className}`}>
      <div className="live-head">
        <b>{t.pieces.livePreview}</b>
        <div className="flex gap-1.5">
          <Button
            variant="ghost"
            className="h-9 px-3 text-sm"
            aria-pressed={fog}
            data-on={fog || undefined}
            onClick={() => setFog((f) => !f)}
          >
            {t.pieces.showFog}
          </Button>
          <Button variant="ghost" className="h-9 px-3 text-sm" onClick={() => setSide((s) => (s === 'w' ? 'b' : 'w'))}>
            <svg
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              aria-hidden
            >
              <path d="M7 4v16M7 20l-3-3M7 20l3-3M17 20V4M17 4l-3 3M17 4l3 3" />
            </svg>
            {t.pieces.swapSides}
          </Button>
        </div>
      </div>
      <div className="live-board" role="img" aria-label={t.pieces.preview}>
        <Board
          board={fog ? FOGGED[side] : START}
          orientation={side}
          you={side}
          legal={noMoves}
          look={look}
          center={center}
          onMove={noop}
        />
      </div>
      <p className="text-[13px] text-accent">{side === 'w' ? t.pieces.previewWhite : t.pieces.previewBlack}</p>
    </div>
  )
})

/** The fog style picker. */
const FogPicker = memo(function FogPicker({ value }: { value: FogStyle }) {
  const t = useT()
  return (
    <div role="radiogroup" aria-label={t.pieces.fog} className="seg">
      {FOG_STYLES.map((f) => (
        <button key={f} type="button" role="radio" aria-checked={value === f} onClick={() => setPrefs({ fog: f })}>
          {t.pieces.fogStyles[f]}
        </button>
      ))}
    </div>
  )
})

/** The pawn everyone starts with, as an avatar. */
export function PawnFace() {
  return (
    <svg viewBox="0 0 40 40" aria-hidden className="size-full opacity-80">
      <circle cx="20" cy="15.2" r="5.6" fill="currentColor" />
      <path
        d="M15.4 21.6h9.2l-1.3 3.2c3.4 1.6 5.9 4.4 7.4 8.8.6 1.8.9 4 .9 6.4H8.4c0-2.4.3-4.6.9-6.4 1.5-4.4 4-7.2 7.4-8.8l-1.3-3.2Z"
        fill="currentColor"
      />
      <rect x="13.6" y="20.6" width="12.8" height="2.4" rx="1.2" fill="currentColor" />
    </svg>
  )
}

/**
 * The avatar: first a set (or the pawn), then one of its sixteen pieces, its
 * White half and its Black half. Sixteen choices at a time instead of every
 * piece of every set.
 */
const AvatarPicker = memo(function AvatarPicker({ avatar, lockOf }: { avatar: string; lockOf: LockOf }) {
  const t = useT()
  const current = avatar ? familyOf(avatar.split('/')[0]) : 'pawn'
  const [family, setFamily] = useState(current)
  const group = AVATAR_GROUPS.find((g) => g.id === family)
  const families: Opt[] = [
    { value: 'pawn', label: t.pieces.pawn, face: <PawnFace /> },
    ...AVATAR_GROUPS.map((g) => ({
      value: g.id,
      label: t.pieces.sets[g.id] ?? g.id,
      face: <AvatarImg code={`${g.sets[0]}/N`} />,
    })),
  ]
  return (
    <div className="ava-pick">
      <Radios
        label={t.pieces.avatarSet}
        value={family}
        options={families}
        onChange={(f) => {
          setFamily(f)
          if (f === 'pawn') setPrefs({ avatar: '' })
        }}
        lockOf={lockOf}
        className="ava-line"
        optClass="ava-opt"
      />
      {group && !lockOf(group.id) && (
        <Radios
          label={t.pieces.avatarPiece}
          value={avatar}
          options={group.sets.flatMap((s) =>
            KINDS.map((k) => ({
              value: `${s}/${k}`,
              label: `${t.pieces.sets[s] ?? s} ${kindName(t, k)}`,
              face: <AvatarImg code={`${s}/${k}`} />,
            })),
          )}
          onChange={(avatar) => setPrefs({ avatar })}
          className="ava-line ava-pieces"
          optClass="ava-opt"
        />
      )}
    </div>
  )
})

/** The Pieces and avatar settings. Guests have them too, kept in this browser. */
export function PiecesSettings({ signedIn }: { signedIn: boolean }) {
  const t = useT()
  const p = usePrefs()
  const o = useOptions()
  const lockOf = useLockOf()
  return (
    <>
      <div className="set-row stack">
        <LivePreview className="live-inline" />
        <p className="text-sm text-accent">{signedIn ? t.pieces.unlockHint : t.pieces.unlockGuest}</p>
      </div>
      <label className="set-row cursor-pointer">
        <div className="row-l">
          <b>{t.pieces.skins}</b>
          <span>{p.plain ? t.pieces.skinsOff : t.pieces.skinsOn}</span>
        </div>
        <Switch.Root checked={!p.plain} onCheckedChange={(on) => setPrefs({ plain: !on })} className="sl-switch">
          <span className="sl-switch-track" />
        </Switch.Root>
      </label>
      <div className="set-row">
        <div className="row-l">
          <b>{t.pieces.fog}</b>
          <span>{t.pieces.fogHint[p.fog]}</span>
        </div>
        <FogPicker value={p.fog} />
      </div>
      <div className="set-row stack">
        <div className="row-l">
          <b>{t.pieces.whiteSet}</b>
          <span>{t.pieces.whiteSetHint}</span>
        </div>
        <Radios label={t.pieces.whiteSet} value={p.white} options={o.w} onChange={onWhite} lockOf={lockOf} />
      </div>
      <div className="set-row stack">
        <div className="row-l">
          <b>{t.pieces.blackSet}</b>
          <span>{t.pieces.blackSetHint}</span>
        </div>
        <Radios label={t.pieces.blackSet} value={p.black} options={o.b} onChange={onBlack} lockOf={lockOf} />
      </div>
      <div className="set-row stack">
        <div className="row-l">
          <b>{t.pieces.lightTiles}</b>
          <span>{t.pieces.tilesHint}</span>
        </div>
        <Radios
          label={t.pieces.lightTiles}
          value={p.lightTiles || 'plain'}
          options={o.lightTiles}
          onChange={onLight}
          lockOf={lockOf}
        />
      </div>
      <div className="set-row stack">
        <div className="row-l">
          <b>{t.pieces.darkTiles}</b>
          <span>{t.pieces.tilesHint}</span>
        </div>
        <Radios
          label={t.pieces.darkTiles}
          value={p.darkTiles || 'plain'}
          options={o.darkTiles}
          onChange={onDark}
          lockOf={lockOf}
        />
      </div>
      <div className="set-row">
        <div className="row-l">
          <b>{t.pieces.side}</b>
          <span>{t.pieces.sideHint}</span>
        </div>
        <div role="radiogroup" aria-label={t.pieces.side} className="seg">
          {(
            [
              ['w', t.common.white],
              ['b', t.common.black],
              ['', t.pieces.either],
            ] as const
          ).map(([v, text]) => (
            <button
              key={v}
              type="button"
              role="radio"
              aria-checked={p.side === v}
              onClick={() => setPrefs({ side: v })}
            >
              {text}
            </button>
          ))}
        </div>
      </div>
      <div className="set-row stack">
        <div className="row-l">
          <b>{t.pieces.avatar}</b>
          <span>{signedIn ? t.pieces.avatarHint : t.pieces.avatarGuest}</span>
        </div>
        <AvatarPicker avatar={p.avatar} lockOf={lockOf} />
      </div>
    </>
  )
}

function kindName(t: ReturnType<typeof useT>, k: string) {
  const names: Record<string, string> = {
    K: t.match.king,
    Q: t.match.queen,
    R: t.match.rook,
    B: t.match.bishop,
    N: t.match.knight,
    P: t.match.pawn,
    J: t.pieces.jester,
    X: t.pieces.prince,
  }
  return names[k]
}
