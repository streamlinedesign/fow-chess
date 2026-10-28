import { navigate } from './router'
import { dismissToast, useToasts } from './social'

/** Short notes at the bottom of the screen. */
export function Toaster() {
  const toasts = useToasts()
  return (
    <div
      aria-live="polite"
      className="pointer-events-none fixed inset-x-0 bottom-0 z-50 flex flex-col items-center gap-2 p-4 pb-[max(16px,env(safe-area-inset-bottom))]"
    >
      {toasts.map((t) => (
        <div
          key={t.id}
          role="status"
          className="toast pointer-events-auto flex w-full max-w-[420px] items-center gap-3 rounded-lg bg-fg py-2.5 pr-2.5 pl-4 text-[15px] text-bg shadow-[0_12px_32px_-8px_rgb(5_5_5/0.4)]"
        >
          <span className="min-w-0 flex-1">{t.text}</span>
          {t.action && (
            <button
              onClick={() => {
                dismissToast(t.id)
                navigate(t.action!.href)
              }}
              className="h-8 rounded-md bg-bg px-3 text-sm font-semibold text-fg"
            >
              {t.action.label}
            </button>
          )}
          <button
            onClick={() => dismissToast(t.id)}
            aria-label="Dismiss"
            className="grid size-8 place-items-center rounded-md opacity-70 hover:opacity-100"
          >
            ✕
          </button>
        </div>
      ))}
    </div>
  )
}
