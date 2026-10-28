import * as RadioGroup from '@radix-ui/react-radio-group'
import * as Switch from '@radix-ui/react-switch'
import { useId, useState } from 'react'
import type { ReactNode } from 'react'
import { clocked, describeTime, SETUPS, setupMode, TIME_GROUPS, timeGroup, timeLabel, type Settings } from './settings'

/** The rules and clock for a new match, boxed like a board of its own. */
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
  return (
    <fieldset className={`setup flex min-w-0 flex-col overflow-hidden rounded-xl ${className}`}>
      <legend className="sr-only">Match settings</legend>
      <div className="flex items-center justify-between gap-3 border-b border-border bg-muted px-4 py-3 text-[13px] sm:px-5">
        <span className="text-sm font-semibold">Match settings</span>
        <span className="truncate text-accent">
          {[
            value.fog ? 'Fog' : 'Full board',
            value.kingCapture ? 'Eat the King' : 'Checkmate',
            setupMode(value).value && setupMode(value).label,
            value.jumpKing && 'Jump King',
            describeTime(value.time),
          ]
            .filter(Boolean)
            .join(' · ')}
        </span>
      </div>

      <div className="flex flex-1 flex-col justify-between gap-6 p-4 sm:p-5">
        <Toggle
          label="Fog of war"
          hint="Hide what your pieces can't see."
          checked={value.fog}
          onChange={(fog) => onChange({ ...value, fog })}
        />

        <div className="flex flex-col gap-2.5">
          <span className="font-semibold" id={`${id}-win-by`}>
            Win by
          </span>
          <RadioGroup.Root
            aria-labelledby={`${id}-win-by`}
            value={value.kingCapture ? 'capture' : 'mate'}
            onValueChange={(v) => onChange({ ...value, kingCapture: v === 'capture' })}
            className="grid grid-cols-2 gap-2.5"
          >
            {[
              ['capture', 'Eat the King', 'No check. Take the king to win.'],
              ['mate', 'Checkmate', 'Normal chess rules with check.'],
            ].map(([v, label, hint]) => (
              <RadioGroup.Item
                key={v}
                value={v}
                className="flex min-h-[104px] flex-col items-start gap-1.5 rounded-lg border border-border p-3.5 text-left transition-colors hover:border-fg data-[state=checked]:border-fg data-[state=checked]:bg-fg data-[state=checked]:text-bg sm:min-h-[112px] sm:p-4"
              >
                <span className="text-lg leading-tight font-semibold tracking-tight sm:text-xl">{label}</span>
                <span className="text-[13px] leading-snug opacity-70">{hint}</span>
              </RadioGroup.Item>
            ))}
          </RadioGroup.Root>
        </div>

        <div className="flex flex-col gap-2.5">
          <span className="font-semibold" id={`${id}-setup`}>
            Pieces
          </span>
          <RadioGroup.Root
            aria-labelledby={`${id}-setup`}
            value={setupMode(value).value || 'standard'}
            onValueChange={(v) => onChange({ ...value, setup: v === 'standard' ? '' : (v as Settings['setup']) })}
            className="grid grid-cols-3 gap-2"
          >
            {SETUPS.map((m) => (
              <RadioGroup.Item
                key={m.value}
                value={m.value || 'standard'}
                className="flex flex-col items-start gap-1 rounded-lg border border-border p-3 text-left transition-colors hover:border-fg data-[state=checked]:border-fg data-[state=checked]:bg-fg data-[state=checked]:text-bg"
              >
                <span className="leading-tight font-semibold tracking-tight sm:text-[17px]">{m.label}</span>
                <span className="text-[12.5px] leading-snug opacity-70">{m.hint}</span>
              </RadioGroup.Item>
            ))}
          </RadioGroup.Root>
        </div>

        <Toggle
          label="Jump King"
          hint="Your king can also jump like a knight."
          checked={!!value.jumpKing}
          onChange={(jumpKing) => onChange({ ...value, jumpKing })}
        />

        <TimePicker value={value} onChange={onChange} signedIn={signedIn} />
      </div>
    </fieldset>
  )
}

function Toggle({
  label,
  hint,
  checked,
  disabled,
  onChange,
}: {
  label: ReactNode
  hint: ReactNode
  checked: boolean
  disabled?: boolean
  onChange: (on: boolean) => void
}) {
  return (
    <label className={`flex items-center justify-between gap-4 ${disabled ? 'opacity-60' : 'cursor-pointer'}`}>
      <span>
        <span className="block font-semibold">{label}</span>
        <span className="block text-sm text-accent">{hint}</span>
      </span>
      <Switch.Root
        checked={checked}
        disabled={disabled}
        onCheckedChange={onChange}
        className="relative h-6 w-10 shrink-0 rounded-full bg-fg/10 shadow-[inset_0_0_0_1px_var(--color-border)] transition-colors data-[state=checked]:bg-fg"
      >
        <Switch.Thumb className="block size-[18px] translate-x-[3px] rounded-full bg-fg transition-transform data-[state=checked]:translate-x-[19px] data-[state=checked]:bg-bg" />
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
        Time control
      </span>
      <div role="radiogroup" aria-label="Time control group" className="tc-tabs">
        {TIME_GROUPS.map(([g]) => (
          <button key={g} type="button" role="radio" aria-checked={group === g} onClick={() => pickGroup(g)}>
            {g}
          </button>
        ))}
      </div>
      <RadioGroup.Root
        aria-labelledby={`${id}-time`}
        value={timeLabel(value.time)}
        onValueChange={(label) => onChange({ ...value, time: presets.find((p) => timeLabel(p) === label)! })}
        className="grid grid-cols-4 gap-2 sm:flex sm:flex-wrap"
      >
        {presets.map((p) => (
          <RadioGroup.Item
            key={timeLabel(p)}
            value={timeLabel(p)}
            className="h-11 rounded-md border border-border px-3.5 font-mono text-[15px] tabular-nums transition-colors hover:border-fg data-[state=checked]:border-fg data-[state=checked]:bg-fg data-[state=checked]:text-bg max-sm:px-0 sm:min-w-[72px] [&:only-child]:col-span-4"
          >
            {p.perMove ? '24 h per move' : clocked(p) ? timeLabel(p) : 'Unlimited'}
          </RadioGroup.Item>
        ))}
      </RadioGroup.Root>
      <div className="mt-2">
        {!signedIn || !clocked(value.time) ? (
          <Toggle
            label="Rated"
            hint={
              !clocked(value.time)
                ? 'No clock games are never rated.'
                : "Guest games aren't rated. Log in to play for rating."
            }
            checked={false}
            disabled
            onChange={() => {}}
          />
        ) : (
          <Toggle
            label="Rated"
            hint={
              value.unrated
                ? 'Just for fun: no rating changes.'
                : 'Counts for both ratings when your opponent is signed in too.'
            }
            checked={!value.unrated}
            onChange={(rated) => onChange({ ...value, unrated: !rated })}
          />
        )}
      </div>
    </div>
  )
}
