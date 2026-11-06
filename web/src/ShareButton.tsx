import { useState } from 'react'
import { useT } from './i18n'
import { Icon } from './icons'
import { shareLink } from './share'
import { Button, Tip } from './ui'

/**
 * Shares a page: the share sheet on phones, else copies the link and says so.
 * With a label it's a button with words; without, an icon button with a tip.
 */
export function ShareButton({
  title,
  text,
  url,
  label,
  variant = 'outline',
  className = '',
}: {
  title: string
  text?: string
  url: string
  label?: string
  variant?: 'solid' | 'outline' | 'ghost'
  className?: string
}) {
  const t = useT()
  const [copied, setCopied] = useState(false)
  const share = async () => {
    if ((await shareLink({ title, text, url })) === 'copied') {
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    }
  }
  if (label) {
    return (
      <Button variant={variant} className={className} onClick={share}>
        <Icon name="share" />
        <span aria-live="polite">{copied ? t.share.copied : label}</span>
      </Button>
    )
  }
  return (
    <Tip label={copied ? t.share.copied : t.share.share}>
      <Button variant={variant} icon aria-label={t.share.share} className={className} onClick={share}>
        <Icon name="share" size={18} />
      </Button>
    </Tip>
  )
}
