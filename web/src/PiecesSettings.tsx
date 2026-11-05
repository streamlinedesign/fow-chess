import * as RadioGroup from '@radix-ui/react-radio-group'
import { useT } from './i18n'
import { piecesFor, setPrefs, usePrefs } from './prefs'
import { CLASSIC, KINDS, SET_IDS, SETS, setImg } from './sets'

/** A set's face in a picker: its knight, or for classic a cream and an ink one. */
function SetFace({ id }: { id: string }) {
  if (id === CLASSIC)
    return (
      <span className="set-face pair" aria-hidden>
        <img src={setImg('cream', 'N')} alt="" />
        <img src={setImg('ink', 'N')} alt="" />
      </span>
    )
  return (
    <span className="set-face" aria-hidden>
      <img src={setImg(id, 'N')} alt="" />
    </span>
  )
}

function SetPicker({ label, value, onChange }: { label: string; value: string; onChange: (id: string) => void }) {
  const t = useT()
  return (
    <RadioGroup.Root aria-label={label} value={value} onValueChange={onChange} className="set-pick">
      {SET_IDS.map((id) => (
        <RadioGroup.Item key={id} value={id} className="set-opt">
          <SetFace id={id} />
          <span className="set-name">{t.pieces.sets[id] ?? id}</span>
        </RadioGroup.Item>
      ))}
    </RadioGroup.Root>
  )
}

/** Their back row and pawns at the top, yours at the bottom, as on a board you play as `bottom`. */
function Preview({ primary, secondary, bottom }: { primary: string; secondary: string; bottom: 'w' | 'b' }) {
  const t = useT()
  const img = piecesFor({ primary, secondary }, bottom)
  const row = 'RNBQKBNR'
  const top = bottom === 'w' ? row.toLowerCase() : row
  const mine = bottom === 'w' ? row : row.toLowerCase()
  const pawn = (c: 'w' | 'b') => (c === 'w' ? 'P' : 'p')
  const lines = [top, pawn(bottom === 'w' ? 'b' : 'w').repeat(8), pawn(bottom).repeat(8), mine]
  return (
    <div className="set-preview" role="img" aria-label={t.pieces.preview}>
      {lines.map((line, r) =>
        [...line].map((p, c) => (
          <span key={`${r}${c}`} className="sq" data-light={(r + c) % 2 === 0}>
            <img className="piece" src={img(p)} alt="" draggable={false} />
          </span>
        )),
      )}
    </div>
  )
}

/** The Pieces and avatar settings. Guests have them too, kept in this browser. */
export function PiecesSettings({ signedIn }: { signedIn: boolean }) {
  const t = useT()
  const p = usePrefs()
  return (
    <>
      <div className="set-row stack">
        <Preview primary={p.primary} secondary={p.secondary} bottom={p.side || 'w'} />
      </div>
      <div className="set-row stack">
        <div className="row-l">
          <b>{t.pieces.yours}</b>
          <span>{t.pieces.yoursHint}</span>
        </div>
        <SetPicker label={t.pieces.yours} value={p.primary} onChange={(primary) => setPrefs({ primary })} />
      </div>
      <div className="set-row stack">
        <div className="row-l">
          <b>{t.pieces.theirs}</b>
          <span>{t.pieces.theirsHint}</span>
        </div>
        <SetPicker label={t.pieces.theirs} value={p.secondary} onChange={(secondary) => setPrefs({ secondary })} />
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
        <RadioGroup.Root
          aria-label={t.pieces.avatar}
          value={p.avatar || 'pawn'}
          onValueChange={(v) => setPrefs({ avatar: v === 'pawn' ? '' : v })}
          className="ava-pick"
        >
          <div className="ava-line">
            <RadioGroup.Item value="pawn" className="ava-opt" aria-label={t.pieces.pawn}>
              <svg viewBox="0 0 40 40" aria-hidden className="size-full opacity-80">
                <circle cx="20" cy="15.2" r="5.6" fill="currentColor" />
                <path
                  d="M15.4 21.6h9.2l-1.3 3.2c3.4 1.6 5.9 4.4 7.4 8.8.6 1.8.9 4 .9 6.4H8.4c0-2.4.3-4.6.9-6.4 1.5-4.4 4-7.2 7.4-8.8l-1.3-3.2Z"
                  fill="currentColor"
                />
                <rect x="13.6" y="20.6" width="12.8" height="2.4" rx="1.2" fill="currentColor" />
              </svg>
            </RadioGroup.Item>
            <span className="text-[13px] text-accent">{t.pieces.pawn}</span>
          </div>
          {SETS.map((s) => (
            <div key={s.id} className="ava-line">
              {KINDS.map((k) => (
                <RadioGroup.Item
                  key={k}
                  value={`${s.id}/${k}`}
                  className="ava-opt"
                  aria-label={`${t.pieces.sets[s.id] ?? s.id} ${kindName(t, k)}`}
                >
                  <img src={s.img(k)} alt="" draggable={false} />
                </RadioGroup.Item>
              ))}
            </div>
          ))}
        </RadioGroup.Root>
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
