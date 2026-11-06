import { useT } from './i18n'
import { Dialog } from './ui'

export type Promo = 'q' | 'r' | 'b' | 'n'

/**
 * Asks which piece a pawn becomes. img draws each choice in the player's
 * colour; pick gets the letter, and the dialog closes either way.
 */
export function PromotionDialog({
  open,
  img,
  onPick,
  onClose,
}: {
  open: boolean
  img: (p: Promo) => string
  onPick: (p: Promo) => void
  onClose: () => void
}) {
  const t = useT()
  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()} title={t.match.promoteTo}>
      <div className="grid grid-cols-4 gap-2">
        {(['q', 'r', 'b', 'n'] as const).map((p) => (
          <button
            key={p}
            type="button"
            autoFocus={p === 'q'}
            aria-label={
              {
                q: t.match.queen,
                r: t.match.rook,
                b: t.match.bishop,
                n: t.match.knight,
              }[p]
            }
            className="aspect-square rounded-lg bg-sq-light p-1 shadow-[inset_0_0_0_1.5px_var(--color-fg)] transition-transform hover:scale-105 active:scale-95"
            onClick={() => {
              onPick(p)
              onClose()
            }}
          >
            <img src={img(p)} alt="" draggable={false} className="size-full" />
          </button>
        ))}
      </div>
    </Dialog>
  )
}
