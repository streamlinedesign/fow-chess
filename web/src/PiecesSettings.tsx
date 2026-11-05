import * as Switch from '@radix-ui/react-switch'
import { memo, useId, useState } from 'react'
import { AvatarImg } from './Avatar'
import { useT } from './i18n'
import { setPrefs, usePrefs, type Prefs } from './prefs'
import { AVATAR_GROUPS, CLASSIC, concrete, familyOf, KINDS, SET_IDS, setImg, TILE_IDS, tileImg } from './sets'

// The pickers are plain radio buttons, not Radix: a page of them stays quick
// to open on a phone, and the arrow keys move through each group for free.
// Each picker is memoised, so a pick redraws only the picker that changed.

type Opt = { value: string; label: string; face: React.ReactNode }

const Radios = memo(function Radios({
  label,
  value,
  options,
  onChange,
  className = 'set-pick',
  optClass = 'set-opt',
}: {
  label: string
  value: string
  options: Opt[]
  onChange: (v: string) => void
  className?: string
  optClass?: string
}) {
  const name = useId()
  return (
    <div role="radiogroup" aria-label={label} className={className}>
      {options.map((o) => (
        <label key={o.value} className={optClass} data-state={o.value === value ? 'checked' : 'unchecked'}>
          <input
            type="radio"
            name={name}
            value={o.value}
            checked={o.value === value}
            onChange={() => onChange(o.value)}
            className="sr-only"
            aria-label={o.label}
          />
          {o.face}
        </label>
      ))}
    </div>
  )
})

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

/** Your Black set's back row and pawns at the top, your White set's at the bottom, all on your tiles. */
const Preview = memo(function Preview({
  white,
  black,
  lightTiles,
  darkTiles,
  plain,
}: Pick<Prefs, 'white' | 'black' | 'lightTiles' | 'darkTiles' | 'plain'>) {
  const t = useT()
  const row = 'RNBQKBNR'
  const lines = [row.toLowerCase(), 'pppppppp', 'PPPPPPPP', row]
  const img = (p: string) => {
    const c = p === p.toUpperCase() ? 'w' : 'b'
    return setImg(concrete(plain ? CLASSIC : c === 'w' ? white : black, c), p, c)
  }
  return (
    <div className="set-preview" role="img" aria-label={t.pieces.preview}>
      {lines.map((line, r) =>
        [...line].map((p, c) => {
          const light = (r + c) % 2 === 0
          const src = plain ? undefined : tileImg(light ? lightTiles : darkTiles, light)
          return (
            <span
              key={`${r}${c}`}
              className="sq"
              data-light={light}
              style={src ? { background: `url("${src}") center / cover` } : undefined}
            >
              <img className="piece" src={img(p)} alt="" draggable={false} decoding="async" />
            </span>
          )
        }),
      )}
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
const AvatarPicker = memo(function AvatarPicker({ avatar }: { avatar: string }) {
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
        className="ava-line"
        optClass="ava-opt"
      />
      {group && (
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
  return (
    <>
      <div className="set-row stack">
        <Preview white={p.white} black={p.black} lightTiles={p.lightTiles} darkTiles={p.darkTiles} plain={p.plain} />
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
      <div className="set-row stack">
        <div className="row-l">
          <b>{t.pieces.whiteSet}</b>
          <span>{t.pieces.whiteSetHint}</span>
        </div>
        <Radios label={t.pieces.whiteSet} value={p.white} options={o.w} onChange={onWhite} />
      </div>
      <div className="set-row stack">
        <div className="row-l">
          <b>{t.pieces.blackSet}</b>
          <span>{t.pieces.blackSetHint}</span>
        </div>
        <Radios label={t.pieces.blackSet} value={p.black} options={o.b} onChange={onBlack} />
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
        />
      </div>
      <div className="set-row stack">
        <div className="row-l">
          <b>{t.pieces.darkTiles}</b>
          <span>{t.pieces.tilesHint}</span>
        </div>
        <Radios label={t.pieces.darkTiles} value={p.darkTiles || 'plain'} options={o.darkTiles} onChange={onDark} />
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
        <AvatarPicker avatar={p.avatar} />
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
