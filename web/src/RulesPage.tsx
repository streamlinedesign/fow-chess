import { useT } from './i18n'
import { go } from './ProfilePage'
import { Cell, Page, PageHead } from './Page'
import { EatRules, WarRules } from './Rules'
import { releaseDate } from './features'

/** How to play: the rules of the two official games, Eat the King and War Mode,
 * and what's planned for 1.0. Mods explain themselves on /mods. */
export function RulesPage() {
  const t = useT()
  return (
    <Page>
      <PageHead title={t.rules.heading} sub={t.rules.intro} />
      <div className="home-row">
        <Cell
          title={t.rules.etkHeading}
          right={<span className="text-[13px] text-accent">{t.rules.etkSub}</span>}
        >
          <EatRules />
        </Cell>
      </div>
      <div className="home-row">
        <Cell
          title={t.rules.warHeading}
          right={<span className="text-[13px] text-accent">{t.rules.warSub}</span>}
        >
          <WarRules />
          <p className="mt-6 text-[15px] text-accent">
            <a href="/mods" onClick={(e) => go(e, '/mods')} className="underline underline-offset-4 hover:text-fg">
              {t.rules.modsNote}
            </a>
          </p>
        </Cell>
      </div>
      <div className="home-row">
        <Cell
          title={t.rules.plansHeading}
          right={<span className="text-[13px] text-accent">{t.rules.plansSub(releaseDate(t.locale))}</span>}
        >
          <ol className="plans">
            {t.rules.plans.map(([title, text], i) => (
              <li key={title}>
                <span className="plans-n">{String(i + 1).padStart(2, '0')}</span>
                <span>
                  <b>{title}</b>
                  <span className="block text-accent">{text}</span>
                </span>
              </li>
            ))}
          </ol>
        </Cell>
      </div>
    </Page>
  )
}
