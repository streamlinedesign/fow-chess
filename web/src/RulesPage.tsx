import { useT } from './i18n'
import { Cell, Page, PageHead } from './Page'
import { Rules } from './Rules'
import { Showcase } from './Showcase'

/** How to play, with the Opera Game replayed under fog to show it. */
export function RulesPage() {
  const t = useT()
  return (
    <Page>
      <PageHead title={t.rules.heading} sub={t.rules.intro} />
      <div className="home-row">
        <Cell>
          <Rules />
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
