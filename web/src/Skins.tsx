import { useMe } from './auth'
import { DEMO_BOT } from './DemoBoard'
import { useT } from './i18n'
import { useUnlocked } from './prefs'
import { go } from './ProfilePage'
import { setImg, THEME_LOCK, tileImg } from './sets'
import { BOTS } from './settings'
import { titleAt } from './titles'

/** The order of the showcase: the bots' sets from strongest, then the title sets from the first, then the beta set. */
const SHOWCASE = [
  'galaxy',
  'ancient',
  'survey',
  'blueprint',
  'bronze',
  'silver',
  'gold',
  'holy',
  'grandmaster',
  'king',
  'cyberpunk',
]
/** Sets new since the last release, tagged in the showcase. */
const NEW = new Set(['bronze', 'silver', 'gold', 'grandmaster', 'king'])

/**
 * A small showcase of the themed sets on the home page: each on its own tiles,
 * with what unlocks it and whether this player has. Locked sets sit under fog.
 */
export function Skins() {
  const t = useT()
  const me = useMe()
  const unlocked = useUnlocked()
  const peak = Math.round(Math.max(me?.rating ?? 0, me?.fourRating ?? 0))
  return (
    <section className="home-row skins-row" aria-labelledby="skins-title">
      <div className="home-cell">
        <div className="skins-head">
          <h2 id="skins-title" className="text-[22px] font-semibold tracking-tight">
            {t.skins.title}
          </h2>
          <a href="/settings#s-pieces" onClick={(e) => go(e, '/settings#s-pieces')}>
            {t.skins.all}
          </a>
        </div>
        <p className="mt-1 text-[15px] text-accent">{t.skins.sub}</p>
        <div className="skins-grid">
          {SHOWCASE.map((theme) => {
            const reason = THEME_LOCK[theme] ?? ''
            const has = !!unlocked?.has(theme)
            const bot = BOTS.find((b) => 'bot:' + b.level === reason)
            const rating = reason.startsWith('rating:') ? Number(reason.slice(7)) : 0
            const how = bot
              ? t.skins.beat(bot.name)
              : rating
                ? t.skins.reach(rating, t.titles.names[titleAt(rating)?.id ?? ''] ?? '')
                : t.skins.beta
            return (
              <div key={theme} className="skin" data-locked={!has || undefined}>
                <span className="skin-sw">
                  <i style={{ backgroundImage: `url("${tileImg(theme, true)}")` }} />
                  <i style={{ backgroundImage: `url("${tileImg(theme, false)}")` }} />
                  <i style={{ backgroundImage: `url("${tileImg(theme, false)}")` }} />
                  <i style={{ backgroundImage: `url("${tileImg(theme, true)}")` }} />
                  <img src={setImg(`${theme}-light`, 'K')} alt="" />
                  <img src={setImg(`${theme}-dark`, 'Q')} alt="" />
                  {!has && (
                    <span className="skin-lock" role="img" aria-label={t.skins.locked}>
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4" aria-hidden>
                        <rect x="5" y="11" width="14" height="10" rx="2" />
                        <path d="M8 11V7a4 4 0 0 1 8 0v4" />
                      </svg>
                    </span>
                  )}
                </span>
                <h3>
                  {t.pieces.sets[theme]}
                  {NEW.has(theme) && <span className="skin-new">{t.skins.isNew}</span>}
                </h3>
                <span className="skin-how">
                  {has ? (
                    <>
                      {how} · <b>{t.skins.unlocked}</b>
                    </>
                  ) : (
                    <>
                      <b>{how}</b>
                      {bot?.level === DEMO_BOT && ` ${t.skins.onThisPage}`}
                      {rating > 0 && me && ` · ${peak}`}
                    </>
                  )}
                </span>
                {!has && rating > 0 && me && (
                  <span className="skin-bar" aria-hidden>
                    <i style={{ width: `${Math.min(100, (peak / rating) * 100)}%` }} />
                  </span>
                )}
                {!has && bot?.level === DEMO_BOT && (
                  <button
                    type="button"
                    className="skin-play"
                    onClick={() => document.querySelector('.home-hero')?.scrollIntoView({ behavior: 'smooth' })}
                  >
                    {t.skins.playHere(bot.name)}
                  </button>
                )}
              </div>
            )
          })}
        </div>
        {!me && <p className="mt-4 text-[13px] text-accent">{t.skins.guest}</p>}
      </div>
    </section>
  )
}
