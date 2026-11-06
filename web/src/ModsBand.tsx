import { useMemo } from 'react'
import { useT } from './i18n'
import { go } from './ProfilePage'
import { KINDS, setImg } from './sets'

/**
 * The Mods band on the home page: one full-width night band with pieces
 * flying through it, the headline, and a card per mod that opens the Mods page
 * with that switch on (see modFromLink in ModsPage). It is dark in both themes.
 */
export function ModsBand() {
  const t = useT()
  const b = t.modsBand
  const cells: [string, string][] = [
    ['extended', setImg('cream', 'R')],
    ['sync', setImg('ink', 'N')],
    ['jump', setImg('cream', 'X')],
    ['jester', setImg('ink', 'J')],
    ['battleship', setImg('ink', 'Q')],
    ['hill', setImg('cream', 'K')],
    ['nofog', ''],
    ['mate', setImg('ink', 'K')],
  ]
  return (
    <section className="mods-band" aria-labelledby="mods-band-title">
      <Sky />
      <div className="mods-band-top">
        <div>
          <h2 className="mods-band-title" id="mods-band-title">
            {b.title.map((w, i) => (
              <span key={w} className={i === 1 ? 'o' : undefined}>
                {w}
              </span>
            ))}
          </h2>
          <p className="mods-band-lede">{b.lede}</p>
          <p className="mods-band-sub">{b.sub}</p>
          <div className="mods-band-ctas">
            <a className="mods-btn solid" href="/mods" onClick={(e) => go(e, '/mods')}>
              {b.open}
            </a>
            <a className="mods-btn" href="/mods?mod=extended" onClick={(e) => go(e, '/mods?mod=extended')}>
              {b.tryExtended}
            </a>
          </div>
        </div>
        <nav className="mods-cells" aria-label={t.common.mods}>
          {cells.map(([id, img]) => {
            const [name, hint] = b.cells[id]
            const href = `/mods?mod=${id}`
            return (
              <a key={id} className="mods-cell" href={href} onClick={(e) => go(e, href)}>
                {img ? (
                  <img className="ic" src={img} alt="" />
                ) : (
                  <span className="ic sw" aria-hidden>
                    <i />
                    <i />
                    <i />
                    <i />
                  </span>
                )}
                <h3>{name}</h3>
                <p>{hint}</p>
              </a>
            )
          })}
        </nav>
      </div>
      <div className="mods-band-foot">
        <span>{b.footLeft}</span>
        <span>{b.footRight}</span>
      </div>
    </section>
  )
}

/** Black and white pieces drifting up through the band, near ones bigger, faster and sharper. */
function Sky() {
  const pieces = useMemo(() => {
    let seed = 7
    const rnd = () => (seed = (seed * 16807) % 2147483647) / 2147483647
    return Array.from({ length: 30 }, (_, i) => {
      const depth = rnd()
      return {
        src: setImg(i % 2 ? 'ink' : 'cream', KINDS[Math.floor(rnd() * KINDS.length)]),
        // Phones show every other one.
        far: i % 2 === 1,
        style: {
          '--x': `${(rnd() * 110 - 5).toFixed(1)}%`,
          '--s': `${Math.round(28 + depth * 110)}px`,
          '--o': (0.25 + depth * 0.6).toFixed(2),
          '--b': `${((1 - depth) * 3).toFixed(1)}px`,
          '--d': `${Math.round(26 - depth * 14)}s`,
          '--delay': `-${(rnd() * 26).toFixed(1)}s`,
          '--dx': `${Math.round(rnd() * 360 - 120)}px`,
          '--r0': `${Math.round(rnd() * 360)}deg`,
          '--r1': `${Math.round(rnd() * 720 - 360)}deg`,
          '--y0': `${Math.round(rnd() * 100)}%`,
        } as React.CSSProperties,
      }
    })
  }, [])
  return (
    <div className="mods-sky" aria-hidden>
      {pieces.map((p, i) => (
        <img key={i} src={p.src} alt="" className={p.far ? 'far' : undefined} style={p.style} />
      ))}
    </div>
  )
}
