import * as RadioGroup from '@radix-ui/react-radio-group'
import * as Switch from '@radix-ui/react-switch'
import { useId, useState } from 'react'
import type { ReactNode } from 'react'
import { useT } from './i18n'
import {
  clocked,
  describeTime,
  SETUPS,
  setupMode,
  TIME_GROUPS,
  timeGroup,
  timeLabel,
  twists,
  type Settings,
} from './settings'

/**
 * The rules and clock for a new match, boxed like a board of its own. Fog of
 * war comes first, since it's the point of the game; the modes below it mix
 * freely.
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
  return (
    <fieldset className={`setup flex min-w-0 flex-col overflow-hidden rounded-xl ${className}`}>
      <legend className="sr-only">{t.panel.title}</legend>
      <div className="flex items-center justify-between gap-3 border-b border-border bg-muted px-4 py-3 text-[13px] sm:px-5">
        <span className="shrink-0 text-sm font-semibold whitespace-nowrap">{t.panel.title}</span>
        <span className="truncate text-accent">
          {[
            value.fog ? t.panel.fogShort : t.panel.fullBoard,
            value.kingCapture ? t.panel.eat : t.panel.mate,
            ...twists(value, t),
            describeTime(value.time, t),
          ].join(' · ')}
        </span>
      </div>

      <div className="flex flex-1 flex-col justify-between gap-6 p-4 sm:p-5">
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
            {[
              ['capture', t.panel.eat, t.panel.eatHint],
              ['mate', t.panel.mate, t.panel.mateHint],
            ].map(([v, label, hint]) => (
              <Choice key={v} value={v} label={label} hint={hint} tall />
            ))}
          </RadioGroup.Root>
          <Toggle
            label={t.twists.hill}
            hint={t.panel.hillHint}
            checked={!!value.hill}
            onChange={(hill) => onChange({ ...value, hill })}
          />
        </Group>

        <Group id={`${id}-setup`} label={t.panel.pieces}>
          <RadioGroup.Root
            aria-labelledby={`${id}-setup`}
            value={setupMode(value).value || 'standard'}
            onValueChange={(v) => {
              const setup = v === 'standard' ? '' : (v as Settings['setup'])
              // Battleship is a guessing game: it turns fog on, though you can turn it back off.
              onChange({ ...value, setup, fog: value.fog || setup === 'battleship' })
            }}
            className="grid grid-cols-3 gap-2"
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

        <Group id={`${id}-twists`} label={t.panel.twists}>
          <Toggle
            label={t.twists.jump}
            hint={t.panel.jumpHint}
            checked={!!value.jumpKing}
            onChange={(jumpKing) => onChange({ ...value, jumpKing })}
          />
          <Toggle
            label={t.twists.backPawns}
            hint={t.panel.backPawnsHint}
            checked={!!value.backPawns}
            onChange={(backPawns) => onChange({ ...value, backPawns })}
          />
        </Group>

        <TimePicker value={value} onChange={onChange} signedIn={signedIn} />
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
function Choice({ value, label, hint, tall }: { value: string; label: string; hint: string; tall?: boolean }) {
  return (
    <RadioGroup.Item
      value={value}
      className={`flex flex-col items-start gap-1 rounded-lg border border-border text-left transition-colors hover:border-fg data-[state=checked]:border-fg data-[state=checked]:bg-fg data-[state=checked]:text-bg ${
        tall ? 'min-h-[96px] gap-1.5 p-3.5 sm:p-4' : 'p-3'
      }`}
    >
      <span className={`leading-tight font-semibold tracking-tight ${tall ? 'text-lg sm:text-xl' : 'sm:text-[17px]'}`}>
        {label}
      </span>
      <span className={`leading-snug opacity-70 ${tall ? 'text-[13px]' : 'text-[12.5px]'}`}>{hint}</span>
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

function TimePicker({
  value,
  onChange,
  signedIn,
}: {
  value: Settings
  onChange: (s: Settings) => void
  signedIn: boolean
}) {
  const id = useId()
  const t = useT()
  // The group shown can differ from the chosen preset's group until a preset is picked.
  const [group, setGroup] = useState(() => timeGroup(value.time))
  const presets = TIME_GROUPS.find(([g]) => g === group)![1]
  const pickGroup = (g: string) => {
    setGroup(g)
    const options = TIME_GROUPS.find(([name]) => name === g)![1]
    // Blitz opens on 5+0, the default; other groups on their first preset.
    onChange({ ...value, time: options.find((p) => p.base === 300 && !p.inc) ?? options[0] })
  }

  return (
    <div className="flex flex-col gap-2.5">
      <span className="font-semibold" id={`${id}-time`}>
        {t.panel.timeControl}
      </span>
      <div role="radiogroup" aria-label={t.panel.timeGroup} className="tc-tabs">
        {TIME_GROUPS.map(([g]) => (
          <button key={g} type="button" role="radio" aria-checked={group === g} onClick={() => pickGroup(g)}>
            {t.time.groups[g]}
          </button>
        ))}
      </div>
      <RadioGroup.Root
        aria-labelledby={`${id}-time`}
        value={timeLabel(value.time, t)}
        onValueChange={(label) => onChange({ ...value, time: presets.find((p) => timeLabel(p, t) === label)! })}
        className="grid grid-cols-4 gap-2 sm:flex sm:flex-wrap"
      >
        {presets.map((p) => (
          <RadioGroup.Item
            key={timeLabel(p, t)}
            value={timeLabel(p, t)}
            className="h-11 rounded-md border border-border px-3.5 font-mono text-[15px] tabular-nums transition-colors hover:border-fg data-[state=checked]:border-fg data-[state=checked]:bg-fg data-[state=checked]:text-bg max-sm:px-0 sm:min-w-[72px] [&:only-child]:col-span-4"
          >
            {clocked(p) ? timeLabel(p, t) : t.panel.unlimited}
          </RadioGroup.Item>
        ))}
      </RadioGroup.Root>
      <div className="mt-2">
        {!signedIn || !clocked(value.time) ? (
          <Toggle
            label={t.common.rated}
            hint={!clocked(value.time) ? t.panel.ratedNoClock : t.panel.ratedGuest}
            checked={false}
            disabled
            onChange={() => {}}
          />
        ) : (
          <Toggle
            label={t.common.rated}
            hint={value.unrated ? t.panel.unratedOn : t.panel.ratedOn}
            checked={!value.unrated}
            onChange={(rated) => onChange({ ...value, unrated: !rated })}
          />
        )}
      </div>
    </div>
  )
}
