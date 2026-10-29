import * as RadioGroup from '@radix-ui/react-radio-group'
import * as Switch from '@radix-ui/react-switch'
import { useId } from 'react'
import type { ReactNode } from 'react'
import { useT } from './i18n'
import {
  clocked,
  clocksFor,
  describeTime,
  modeName,
  sameTime,
  SETUPS,
  setupMode,
  timeGroup,
  timeLabel,
  type Settings,
} from './settings'

/**
 * The mode and clock for a new match, boxed like a board of its own. Fog of
 * war and Eat the King are the game, so they aren't settings here; on the
 * mods page every dial is exposed.
 */
export function SettingsPanel({
  value,
  onChange,
  signedIn,
  className = '',
}: {
  value: Settings
  onChange: (s: Settings) => void
  signedIn: boolean
  className?: string
}) {
  // The panel can be on screen twice (home page and Offer a match), so ids are per panel.
  const id = useId()
  const t = useT()
  const mods = !!value.mods
  const clocks = clocksFor(mods)
  return (
    <fieldset className={`setup flex min-w-0 flex-col overflow-hidden rounded-xl ${className}`}>
      <legend className="sr-only">{t.panel.title}</legend>
      <div className="flex items-center justify-between gap-3 border-b border-border bg-muted px-4 py-3 text-[13px] sm:px-5">
        <span className="shrink-0 text-sm font-semibold whitespace-nowrap">{mods ? t.common.mods : t.panel.title}</span>
        <span className="truncate text-accent">{[modeName(value, t), describeTime(value.time, t)].join(' · ')}</span>
      </div>

      <div className="flex flex-1 flex-col justify-between gap-6 p-4 sm:p-5">
        {mods && (
          <>
            <Toggle
              big
              label={t.panel.fog}
              hint={value.fog ? t.panel.fogOnHint : t.panel.fogOffHint}
              checked={value.fog}
              onChange={(fog) => onChange({ ...value, fog })}
            />
            <Group id={`${id}-win`} label={t.panel.winBy}>
              <RadioGroup.Root
                aria-labelledby={`${id}-win`}
                value={value.kingCapture ? 'capture' : 'mate'}
                onValueChange={(v) => onChange({ ...value, kingCapture: v === 'capture' })}
                className="grid grid-cols-2 gap-2.5"
              >
                <Choice value="capture" label={t.panel.eat} hint={t.panel.eatHint} />
                <Choice value="mate" label={t.panel.mate} hint={t.panel.mateHint} />
              </RadioGroup.Root>
              <Toggle
                label={t.legacy.hill}
                hint={t.panel.hillHint}
                checked={!!value.hill}
                onChange={(hill) => onChange({ ...value, hill })}
              />
            </Group>
          </>
        )}

        <Group id={`${id}-mode`} label={t.panel.mode}>
          <RadioGroup.Root
            aria-labelledby={`${id}-mode`}
            value={setupMode(value).value || 'standard'}
            onValueChange={(v) => onChange({ ...value, setup: v === 'standard' ? '' : (v as Settings['setup']) })}
            className="grid gap-2.5 sm:grid-cols-3"
          >
            {SETUPS.map((m) => (
              <Choice
                key={m.value}
                value={m.value || 'standard'}
                label={t.setups[m.value].label}
                hint={t.setups[m.value].hint}
              />
            ))}
          </RadioGroup.Root>
        </Group>

        {mods && (
          <Group id={`${id}-twists`} label={t.panel.twists}>
            <Toggle
              label={t.legacy.jump}
              hint={t.panel.jumpHint}
              checked={!!value.jumpKing}
              onChange={(jumpKing) => onChange({ ...value, jumpKing })}
            />
            <Toggle
              label={t.legacy.backPawns}
              hint={t.panel.backPawnsHint}
              checked={!!value.backPawns}
              onChange={(backPawns) => onChange({ ...value, backPawns })}
            />
          </Group>
        )}

        <Group id={`${id}-time`} label={t.panel.timeControl}>
          <RadioGroup.Root
            aria-labelledby={`${id}-time`}
            value={String(clocks.findIndex((p) => sameTime(p, value.time)))}
            onValueChange={(i) => onChange({ ...value, time: clocks[Number(i)] })}
            className="grid grid-cols-2 gap-2 sm:grid-cols-3"
          >
            {clocks.map((p, i) => (
              <RadioGroup.Item
                key={i}
                value={String(i)}
                className="flex flex-col items-start gap-0.5 rounded-md border border-border px-3 py-2 text-left transition-colors hover:border-fg data-[state=checked]:border-fg data-[state=checked]:bg-fg data-[state=checked]:text-bg"
              >
                <span className="text-[15px] leading-tight font-semibold">
                  {clocked(p) ? t.time.groups[timeGroup(p)] : t.time.noClock}
                </span>
                <span className="text-[13px] leading-tight opacity-70">{clocked(p) ? timeLabel(p, t) : '∞'}</span>
              </RadioGroup.Item>
            ))}
          </RadioGroup.Root>
        </Group>

        {mods ? (
          <p className="text-sm text-accent">{t.panel.modsNote}</p>
        ) : signedIn ? (
          <Toggle
            label={t.common.rated}
            hint={value.unrated ? t.panel.unratedOn : t.panel.ratedOn}
            checked={!value.unrated}
            onChange={(rated) => onChange({ ...value, unrated: !rated })}
          />
        ) : (
          <Toggle label={t.common.rated} hint={t.panel.ratedGuest} checked={false} disabled onChange={() => {}} />
        )}
      </div>
    </fieldset>
  )
}

function Group({ id, label, children }: { id: string; label: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-3">
      <span className="font-semibold" id={id}>
        {label}
      </span>
      {children}
    </div>
  )
}

/** One card in a radio group of modes. */
function Choice({ value, label, hint }: { value: string; label: string; hint: string }) {
  return (
    <RadioGroup.Item
      value={value}
      className="flex min-h-[88px] flex-col items-start gap-1.5 rounded-lg border border-border p-3.5 text-left transition-colors hover:border-fg data-[state=checked]:border-fg data-[state=checked]:bg-fg data-[state=checked]:text-bg sm:p-4"
    >
      <span className="text-lg leading-tight font-semibold tracking-tight">{label}</span>
      <span className="text-[13px] leading-snug opacity-70">{hint}</span>
    </RadioGroup.Item>
  )
}

function Toggle({
  label,
  hint,
  checked,
  disabled,
  big,
  onChange,
}: {
  label: ReactNode
  hint: ReactNode
  checked: boolean
  disabled?: boolean
  /** The headline setting, set in larger type. */
  big?: boolean
  onChange: (on: boolean) => void
}) {
  return (
    <label className={`flex items-center justify-between gap-4 ${disabled ? 'opacity-60' : 'cursor-pointer'}`}>
      <span>
        <span className={`block font-semibold ${big ? 'text-[22px] leading-tight tracking-tight' : ''}`}>{label}</span>
        <span className={`block text-accent ${big ? 'mt-0.5 text-[15px]' : 'text-sm'}`}>{hint}</span>
      </span>
      <Switch.Root
        checked={checked}
        disabled={disabled}
        onCheckedChange={onChange}
        className={`relative shrink-0 rounded-full bg-fg/10 shadow-[inset_0_0_0_1px_var(--color-border)] transition-colors data-[state=checked]:bg-fg ${
          big ? 'h-8 w-[52px]' : 'h-6 w-10'
        }`}
      >
        <Switch.Thumb
          className={`block rounded-full bg-fg transition-transform data-[state=checked]:bg-bg ${
            big
              ? 'size-6 translate-x-1 data-[state=checked]:translate-x-6'
              : 'size-[18px] translate-x-[3px] data-[state=checked]:translate-x-[19px]'
          }`}
        />
      </Switch.Root>
    </label>
  )
}
