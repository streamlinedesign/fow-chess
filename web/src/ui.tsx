import * as RadixDialog from '@radix-ui/react-dialog'
import * as Tooltip from '@radix-ui/react-tooltip'
import type { ComponentProps, ReactNode } from 'react'

type ButtonProps = ComponentProps<'button'> & {
  variant?: 'solid' | 'outline' | 'ghost'
  icon?: boolean
  /** Shows a spinner over the label and disables the button, keeping its size. */
  loading?: boolean
}

const variants = {
  solid: 'bg-fg text-bg hover:opacity-85',
  outline: 'shadow-[inset_0_0_0_1.5px_var(--color-fg)] hover:bg-fg hover:text-bg',
  ghost: 'hover:bg-fg/10',
}

export function Button({
  variant = 'outline',
  icon,
  loading,
  className = '',
  children,
  disabled,
  ...props
}: ButtonProps) {
  return (
    <button
      className={`relative inline-flex h-11 items-center justify-center gap-2 rounded-md ${icon ? 'w-11 shrink-0' : 'px-5'} text-[15px] font-semibold whitespace-nowrap transition-[opacity,background-color,color] active:scale-[0.98] disabled:pointer-events-none ${loading ? '' : 'disabled:opacity-40'} ${variants[variant]} ${className}`}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      {...props}
    >
      {loading ? (
        <>
          <span className="invisible contents">{children}</span>
          <Spinner className="absolute" />
        </>
      ) : (
        children
      )}
    </button>
  )
}

/** A small spinning ring in the current text colour. */
export function Spinner({ className = '', label }: { className?: string; label?: string }) {
  return (
    <span role={label ? 'status' : undefined} className={`inline-flex items-center gap-2 ${className}`}>
      <span className="spinner" aria-hidden />
      {label && <span>{label}</span>}
    </span>
  )
}

export function Tip({ label, children }: { label: string; children: ReactNode }) {
  return (
    <Tooltip.Root>
      <Tooltip.Trigger asChild>{children}</Tooltip.Trigger>
      <Tooltip.Portal>
        <Tooltip.Content sideOffset={6} className="z-50 rounded-md bg-fg px-2.5 py-1.5 text-xs font-semibold text-bg">
          {label}
        </Tooltip.Content>
      </Tooltip.Portal>
    </Tooltip.Root>
  )
}

type DialogProps = {
  open: boolean
  onOpenChange?: (open: boolean) => void
  title: string
  description?: ReactNode
  children?: ReactNode
  dismissable?: boolean
}

export function Dialog({ open, onOpenChange, title, description, children, dismissable = true }: DialogProps) {
  return (
    <RadixDialog.Root open={open} onOpenChange={onOpenChange}>
      <RadixDialog.Portal>
        <RadixDialog.Overlay className="overlay fixed inset-0 z-40 bg-black/40 backdrop-blur-[2px]" />
        <RadixDialog.Content
          className="dialog fixed top-1/2 left-1/2 z-50 w-[min(92vw,380px)] -translate-x-1/2 -translate-y-1/2 rounded-xl bg-bg p-6 text-fg shadow-[0_0_0_1.5px_var(--color-fg),0_20px_60px_rgb(0_0_0/0.35)]"
          onPointerDownOutside={(e) => !dismissable && e.preventDefault()}
          onEscapeKeyDown={(e) => !dismissable && e.preventDefault()}
        >
          <RadixDialog.Title className="text-3xl font-bold tracking-tight">{title}</RadixDialog.Title>
          {description ? (
            <RadixDialog.Description className="mt-2 text-accent">{description}</RadixDialog.Description>
          ) : (
            <RadixDialog.Description className="sr-only">{title}</RadixDialog.Description>
          )}
          {children && <div className="mt-6">{children}</div>}
        </RadixDialog.Content>
      </RadixDialog.Portal>
    </RadixDialog.Root>
  )
}
