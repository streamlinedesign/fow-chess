import type { MouseEvent } from 'react'
import { useT } from './i18n'
import { Cell, Page, PageHead } from './Page'
import { CoreRules, LayoutRules, ModRules, PlayingRules } from './Rules'
import { navigate } from './router'
import { Showcase } from './Showcase'
import { Button } from './ui'

/**
 * The rules, on two pages: Eat the King itself at /rules (the game, its
 * layouts, playing a match, and the Opera Game replayed under fog), and the
 * mods at /rules/mods, one rule changed each.
 */
export function RulesPage({ mods }: { mods?: boolean }) {
  const t = useT()
  return (
    <Page>
      <PageHead
        title={mods ? t.rules.modsHeading : t.rules.nav.core}
        sub={mods ? t.rules.modsIntro : t.rules.intro}
        right={<RulesNav mods={!!mods} />}
      />
      {mods ? (
        <div className="home-row">
          <Cell>
            <ModRules />
            <div className="mt-6">
              <Button variant="solid" className="h-10 px-4" onClick={() => navigate('/mods')}>
                {t.rules.openMods}
              </Button>
            </div>
          </Cell>
        </div>
      ) : (
        <>
          <div className="home-row">
            <Cell>
              <CoreRules />
            </Cell>
          </div>
          <div className="home-row">
            <Cell title={t.rules.layouts} right={<span className="text-[13px] text-accent">{t.rules.layoutsSub}</span>}>
              <LayoutRules />
            </Cell>
          </div>
          <div className="home-row">
            <Cell title={t.rules.playing}>
              <PlayingRules />
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
        </>
      )}
    </Page>
  )
}

/** Eat the King or Mods: two links that look like a segmented switch. */
function RulesNav({ mods }: { mods: boolean }) {
  const t = useT()
  const go = (e: MouseEvent, path: string) => {
    if (e.metaKey || e.ctrlKey || e.shiftKey || e.button !== 0) return
    e.preventDefault()
    navigate(path)
    window.scrollTo(0, 0)
  }
  return (
    <nav aria-label={t.rules.nav.aria} className="seg shrink-0 self-start">
      <a href="/rules" aria-current={mods ? undefined : 'page'} onClick={(e) => go(e, '/rules')}>
        {t.rules.nav.core}
      </a>
      <a href="/rules/mods" aria-current={mods ? 'page' : undefined} onClick={(e) => go(e, '/rules/mods')}>
        {t.rules.nav.mods}
      </a>
    </nav>
  )
}
