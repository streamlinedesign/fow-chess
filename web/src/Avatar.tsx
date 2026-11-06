import { TitleTag } from './TitleTag'
import { useAvatar } from './avatars'
import { useT } from './i18n'
import { avatarImg } from './sets'

/**
 * A piece as an avatar, zoomed on its head: the top of the piece fills the
 * circle (see .ava-head in index.css, which places each kind).
 */
export function AvatarImg({ code }: { code: string }) {
  const src = avatarImg(code)
  if (!src) return null
  return (
    <span className="ava-head" aria-hidden>
      <img src={src} alt="" draggable={false} data-kind={code.split('/')[1]} decoding="async" />
    </span>
  )
}

/**
 * A player's avatar: the piece they picked (see sets.ts), or the pawn
 * everyone starts with, cut off at the shoulders. Give the player's name to
 * look theirs up, or code to draw one directly.
 */
export function Avatar({
  size = 36,
  online,
  className = '',
  name,
  code,
}: {
  size?: number
  online?: boolean
  className?: string
  name?: string
  code?: string
}) {
  const t = useT()
  const looked = useAvatar(code === undefined ? name : undefined)
  const avatar = code ?? looked
  const src = avatarImg(avatar)
  const dot = Math.min(Math.max(Math.round(size * 0.28), 9), 20)
  return (
    <span
      className={`relative inline-block shrink-0 rounded-full bg-fg/10 align-middle ${className}`}
      style={{ width: size, height: size }}
    >
      {src ? (
        <AvatarImg code={avatar!} />
      ) : (
        <svg viewBox="0 0 40 40" aria-hidden className="block size-full rounded-full opacity-80">
          <circle cx="20" cy="15.2" r="5.6" fill="currentColor" />
          <path
            d="M15.4 21.6h9.2l-1.3 3.2c3.4 1.6 5.9 4.4 7.4 8.8.6 1.8.9 4 .9 6.4H8.4c0-2.4.3-4.6.9-6.4 1.5-4.4 4-7.2 7.4-8.8l-1.3-3.2Z"
            fill="currentColor"
          />
          <rect x="13.6" y="20.6" width="12.8" height="2.4" rx="1.2" fill="currentColor" />
        </svg>
      )}
      {online && (
        <span
          className="absolute -right-px -bottom-px rounded-full bg-online shadow-[0_0_0_2px_var(--color-bg)]"
          style={{ width: dot, height: dot }}
          aria-label={t.common.online}
        />
      )}
    </span>
  )
}

/** An avatar with a name and a line under it, usually the rating. */
export function Who({
  name,
  sub,
  size = 36,
  online,
}: {
  name: string
  sub?: React.ReactNode
  size?: number
  online?: boolean
}) {
  return (
    <span className="flex min-w-0 items-center gap-2.5">
      <Avatar size={size} online={online} name={name} />
      <span className="flex min-w-0 flex-col leading-tight">
        <span className="truncate font-semibold">
          <TitleTag name={name} />
          {name}
        </span>
        {sub !== undefined && sub !== '' && <span className="truncate text-[13px] text-accent">{sub}</span>}
      </span>
    </span>
  )
}
