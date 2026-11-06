import { useT } from './i18n'
import { tagOf, useTitle } from './titles'

/** The tag before a player's name (GM, IM…), titled with its name; nothing for players without one. */
export function TitleTag({ name, className = '' }: { name: string | undefined; className?: string }) {
  const t = useT()
  const id = useTitle(name)
  const tag = tagOf(id)
  if (!tag) return null
  return (
    <abbr className={`title-tag ${className}`} data-title={id} title={t.titles.names[id]}>
      {tag}
    </abbr>
  )
}
