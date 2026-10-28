import { useT } from './i18n'
import { Rules } from './Rules'
import { Showcase } from './Showcase'

/** How to play, with the Opera Game replayed under fog to show it. */
export function RulesPage() {
  const t = useT()
  return (
    <main className="mx-auto flex w-full max-w-[1184px] flex-1 flex-col gap-12 px-4 pt-6 pb-16 sm:px-8 sm:pt-8 lg:gap-16">
      <Rules />

      <section className="flex flex-col gap-5" aria-labelledby="replay-heading">
        <div className="border-b border-fg pb-4">
          <h2 id="replay-heading" className="text-[clamp(30px,5vw,44px)] leading-none font-semibold tracking-tight">
            {t.landing.replayHeading}
          </h2>
          <p className="mt-2 text-accent">{t.landing.replaySub}</p>
        </div>
        <div className="replay-grid grid items-start gap-5 lg:grid-cols-[minmax(0,520px)_minmax(0,1fr)] lg:gap-10">
          <Showcase boardClassName="w-full" captionClassName="w-full lg:pt-2" />
        </div>
      </section>
    </main>
  )
}
