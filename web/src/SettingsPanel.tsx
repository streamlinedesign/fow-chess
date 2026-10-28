import * as RadioGroup from '@radix-ui/react-radio-group'
import * as Switch from '@radix-ui/react-switch'
import { useId, useState } from 'react'
import type { ReactNode } from 'react'
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
  return (
    <fieldset className={`setup flex min-w-0 flex-col overflow-hidden rounded-xl ${className}`}>
      <legend className="sr-only">Match settings</legend>
      <div className="flex items-center justify-between gap-3 border-b border-border bg-muted px-4 py-3 text-[13px] sm:px-5">
        <span className="shrink-0 text-sm font-semibold whitespace-nowrap">Match settings</span>
        <span className="truncate text-accent">
          {[
            value.fog ? 'Fog' : 'Full board',
            value.kingCapture ? 'Eat the King' : 'Checkmate',
            ...twists(value),
            describeTime(value.time),
          ].join(' · ')}
        </span>
      </div>

      <div className="flex flex-1 flex-col justify-between gap-6 p-4 sm:p-5">
        <Toggle
          big
          label="Fog of war"
          hint={value.fog ? 'You only see what your pieces see.' : 'The whole board, as in normal chess.'}
          checked={value.fog}
          onChange={(fog) => onChange({ ...value, fog })}
        />

        <Group id={`${id}-win`} label="Win by">
          <RadioGroup.Root
            aria-labelledby={`${id}-win`}
            value={value.kingCapture ? 'capture' : 'mate'}
            onValueChange={(v) => onChange({ ...value, kingCapture: v === 'capture' })}
            className="grid grid-cols-2 gap-2.5"
          >
            {[
              ['capture', 'Eat the King', 'No check. Take the king to win.'],
              ['mate', 'Checkmate', 'Normal chess rules with check.'],
            ].map(([v, label, hint]) => (
              <Choice key={v} value={v} label={label} hint={hint} tall />
            ))}
          </RadioGroup.Root>
          <Toggle
            label="King of the Hill"
            hint="Also win by getting your king to one of the four centre squares."
            checked={!!value.hill}
            onChange={(hill) => onChange({ ...value, hill })}
          />
        </Group>

        <Group id={`${id}-setup`} label="Pieces">
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
              <Choice key={m.value} value={m.value || 'standard'} label={m.label} hint={m.hint} />
            ))}
          </RadioGroup.Root>
        </Group>

        <Group id={`${id}-twists`} label="Twists">
          <Toggle
            label="Jump King"
            hint="Your king can also jump like a knight."
            checked={!!value.jumpKing}
            onChange={(jumpKing) => onChange({ ...value, jumpKing })}
          />
          <Toggle
            label="Backward pawns"
            hint="Pawns can also step one square back, and see behind them."
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
