import { useT } from './i18n'
import { go } from './ProfilePage'
import { Cell, Page, PageHead } from './Page'
import { EatRules, WarRules } from './Rules'
import { Showcase } from './Showcase'

/** How to play: the rules of the two official games, Eat the King and War Mode,
 * and the Opera Game replayed under fog. Mods explain themselves on /mods. */
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
          title={t.landing.replayHeading}
          right={<span className="text-[13px] text-accent">{t.landing.replaySub}</span>}
        >
          <div className="replay-grid grid items-start gap-5 lg:grid-cols-[minmax(0,520px)_minmax(0,1fr)] lg:gap-10">
            <Showcase boardClassName="w-full" captionClassName="w-full lg:pt-2" />
          </div>
        </Cell>
      </div>
    </Page>
  )
}
