import * as Menu from '@radix-ui/react-dropdown-menu'
import { createContext, useContext, useState, useSyncExternalStore, type ReactElement, type ReactNode } from 'react'
import { Drawer } from 'vaul'

const PHONE = '(max-width: 639px)'

function subscribe(cb: () => void) {
  const m = matchMedia(PHONE)
  m.addEventListener('change', cb)
  return () => m.removeEventListener('change', cb)
}

/** Whether the screen is phone sized, where menus open as drawers from the bottom. */
export function usePhone() {
  return useSyncExternalStore(subscribe, () => matchMedia(PHONE).matches)
}

type DrawerProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  trigger: ReactElement
  title: ReactNode
  /** Keep the title for screen readers only, when the content has its own heading. */
  hideTitle?: boolean
  className?: string
  children: ReactNode
}

/** A drawer from the bottom of the screen, as in shadcn/ui: drag the handle down or tap outside to close. */
export function BottomDrawer({ open, onOpenChange, trigger, title, hideTitle, className = '', children }: DrawerProps) {
  return (
    <Drawer.Root open={open} onOpenChange={onOpenChange}>
      <Drawer.Trigger asChild>{trigger}</Drawer.Trigger>
      <Drawer.Portal>
        <Drawer.Overlay className="drawer-overlay" />
        <Drawer.Content className={`drawer ${className}`} aria-describedby={undefined}>
          <div className="drawer-handle" aria-hidden />
          <Drawer.Title className={hideTitle ? 'sr-only' : 'drawer-title'}>{title}</Drawer.Title>
          {children}
        </Drawer.Content>
      </Drawer.Portal>
    </Drawer.Root>
  )
}

/** Set inside a drawer menu: closes it. Its absence means the menu is a dropdown. */
const CloseDrawer = createContext<(() => void) | null>(null)

type DropMenuProps = {
  trigger: ReactElement
  /** Names the menu for screen readers. */
  title: string
  children: ReactNode
}

/** A menu that drops down from its button, or opens as a drawer on phones. */
export function DropMenu({ trigger, title, children }: DropMenuProps) {
  const phone = usePhone()
  const [open, setOpen] = useState(false)
  if (phone)
    return (
      <BottomDrawer open={open} onOpenChange={setOpen} trigger={trigger} title={title} hideTitle>
        <CloseDrawer.Provider value={() => setOpen(false)}>
          <div className="drawer-menu">{children}</div>
        </CloseDrawer.Provider>
      </BottomDrawer>
    )
  return (
    <Menu.Root modal={false} open={open} onOpenChange={setOpen}>
      <Menu.Trigger asChild>{trigger}</Menu.Trigger>
      <Menu.Portal>
        <Menu.Content align="end" sideOffset={8} className="menu">
          {children}
        </Menu.Content>
      </Menu.Portal>
    </Menu.Root>
  )
}

export function MenuItem({
  onSelect,
  className = '',
  children,
}: {
  onSelect: () => void
  className?: string
  children: ReactNode
}) {
  const close = useContext(CloseDrawer)
  if (close)
    return (
      <button
        type="button"
        className={`menu-item w-full ${className}`}
        onClick={() => {
          close()
          onSelect()
        }}
      >
        {children}
      </button>
    )
  return (
    <Menu.Item className={`menu-item ${className}`} onSelect={onSelect}>
      {children}
    </Menu.Item>
  )
}

export function MenuSeparator() {
  const close = useContext(CloseDrawer)
  return close ? (
    <div className="mx-1 my-1.5 h-px bg-border" />
  ) : (
    <Menu.Separator className="mx-1 my-1.5 h-px bg-border" />
  )
}
