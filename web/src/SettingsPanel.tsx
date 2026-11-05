import * as RadioGroup from '@radix-ui/react-radio-group'
import * as Switch from '@radix-ui/react-switch'
import { useId, useState } from 'react'
import type { ReactNode } from 'react'
import { useT, type Dict } from './i18n'
import {
  clocked,
  describeTime,
  modeName,
  NO_CLOCK,
  sameTime,
  SETUPS,
  setupMode,
  TIME_GROUPS,
  timeGroup,
  timeLabel,
  type Settings,
  type TimeControl,
} from './settings'

/**
 * The mode and clock for a new match, boxed like a board of its own. Fog of
 * war and Eat the King are the game, so they aren't settings here; on the
 * mods page every dial is exposed. There's no ranked switch: a game is ranked
 * whenever both players are signed in, and a mods game never is.
 */
export function SettingsPanel({
  value,
  onChange,
  className = '',
  four,
  onFour,
}: {
  value: Settings
  onChange: (s: Settings) => void
  className?: string
  /** A four-player game: always Eat the King, and Sync Move is on offer. */
  four?: boolean
  /** Offers the choice of two or four players (the mods page). */
  onFour?: (four: boolean) => void
}) {
  // The panel can be on screen twice (home page and Offer a match), so ids are per panel.
  const id = useId()
  const t = useT()
  const mods = !!value.mods
  return (
    <fieldset className={`setup flex min-w-0 flex-col overflow-hidden rounded-xl ${className}`}>
      <legend className="sr-only">{t.panel.title}</legend>
      <div className="flex items-center justify-between gap-3 border-b border-border bg-muted px-4 py-3 text-[13px] sm:px-5">
        <span className="shrink-0 text-sm font-semibold whitespace-nowrap">{mods ? t.common.mods : t.panel.title}</span>
        <span className="truncate text-accent">
          {[four ? fourName(value, t) : modeName(value, t), describeTime(value.time, t)].join(' · ')}
        </span>
      </div>

      <div className="setup-body flex flex-1 flex-col justify-between gap-6 p-4 sm:p-5">
        <div className="@container flex min-w-0 flex-col gap-6">
          {onFour && (
            <Group id={`${id}-players`} label={t.four.players}>
              <RadioGroup.Root
                aria-labelledby={`${id}-players`}
                value={four ? '4' : '2'}
                onValueChange={(v) => onFour(v === '4')}
                className="grid grid-cols-2 gap-2.5"
              >
                <Choice value="2" label={t.four.two} hint={t.four.twoHint} />
                <Choice value="4" label={t.four.four} hint={t.four.fourHint} />
              </RadioGroup.Root>
            </Group>
          )}

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
                {four || value.sync ? (
                  <p className="text-sm text-accent">{four ? t.four.eatAlways : t.panel.syncEat}</p>
                ) : (
                  <RadioGroup.Root
                    aria-labelledby={`${id}-win`}
                    value={value.kingCapture ? 'capture' : 'mate'}
                    onValueChange={(v) => onChange({ ...value, kingCapture: v === 'capture' })}
                    className="grid grid-cols-2 gap-2.5"
                  >
                    <Choice value="capture" label={t.panel.eat} hint={t.panel.eatHint} />
                    <Choice value="mate" label={t.panel.mate} hint={t.panel.mateHint} />
                  </RadioGroup.Root>
                )}
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
              onValueChange={(v) =>
                onChange({
                  ...value,
                  setup: v === 'standard' ? '' : (v as Settings['setup']),
                })
              }
              className="grid gap-2.5 @md:grid-cols-3"
            >
              {SETUPS.map((m) => (
                <Choice
                  key={m.value}
                  value={m.value || 'standard'}
                  label={t.setups[m.value].label}
                  hint={four && m.value === 'battleship' ? t.four.battleshipHint : t.setups[m.value].hint}
                />
              ))}
            </RadioGroup.Root>
          </Group>
        </div>

        <div className="@container flex min-w-0 flex-col gap-6">
          {mods && (
            <Group id={`${id}-twists`} label={t.panel.twists}>
              <Toggle
                label={t.four.sync}
                hint={t.four.syncHint}
                checked={!!value.sync}
                onChange={(sync) => onChange({ ...value, sync })}
              />
              <Toggle
                label={t.legacy.jump}
                hint={t.panel.jumpHint}
                checked={!!value.jumpKing}
                onChange={(jumpKing) => onChange({ ...value, jumpKing })}
              />
              <Toggle
                label={t.legacy.jesterPawns}
                hint={t.panel.jesterPawnsHint}
                checked={!!value.jesterPawns}
                onChange={(jesterPawns) => onChange({ ...value, jesterPawns })}
              />
              <Toggle
                label={t.panel.center}
                hint={four ? t.four.centerHint : t.panel.centerHint}
                checked={!!value.centerVision}
                onChange={(on) => onChange({ ...value, centerVision: on ? 'any' : '' })}
              />
            </Group>
          )}

          <Group id={`${id}-time`} label={t.panel.timeControl}>
            <ClockPicker
              labelledBy={`${id}-time`}
              value={value.time}
              noClock={mods}
              onChange={(time) => onChange({ ...value, time })}
            />
          </Group>
        </div>
      </div>
    </fieldset>
  )
}

/** A four-player game's name: Four players, and its setup and Sync Move when they're on. */
export function fourName(s: Settings, t: Dict) {
  const setup = setupMode(s).value
  return [t.four.four, ...(setup ? [t.setups[setup].label] : []), ...(s.sync ? [t.four.sync] : [])].join(' · ')
}

/** Clocks split by how long a game takes, the same way as on the home page: Short is Blitz and Rapid, Long the rest. */
const SHORT_GROUPS = ['Blitz', 'Rapid']
type Length = 'short' | 'long' | 'none'
const lengthOf = (t: TimeControl): Length =>
  !clocked(t) ? 'none' : SHORT_GROUPS.includes(timeGroup(t)) ? 'short' : 'long'
const presetsFor = (l: Length) =>
  l === 'none' ? [] : TIME_GROUPS.filter(([g]) => SHORT_GROUPS.includes(g) === (l === 'short')).flatMap(([, p]) => p)

/** The clock: game length first (and No clock on the mods page), then a few presets of that length. */
function ClockPicker({
  labelledBy,
  value,
  noClock,
  onChange,
}: {
  labelledBy: string
  value: TimeControl
  noClock: boolean
  onChange: (t: TimeControl) => void
}) {
  const t = useT()
  const length = lengthOf(value)
  // Each length remembers the clock picked in it while the panel is open.
  const [last, setLast] = useState<Record<'short' | 'long', TimeControl>>(() => ({
    short: length === 'short' ? value : presetsFor('short')[0],
    long: length === 'long' ? value : presetsFor('long')[0],
  }))
  const pickLength = (l: string) => {
    if (l === length) return
    onChange(l === 'none' ? NO_CLOCK : last[l as 'short' | 'long'])
  }
  const pick = (time: TimeControl) => {
    if (length !== 'none') setLast({ ...last, [length]: time })
    onChange(time)
  }
  const presets = presetsFor(length)
  return (
    <div className="flex flex-col gap-2.5">
      <RadioGroup.Root aria-labelledby={labelledBy} value={length} onValueChange={pickLength} className="seg seg-lg">
        <RadioGroup.Item value="short">
          {t.home.short}
          <small>{t.home.shortSub}</small>
        </RadioGroup.Item>
        <RadioGroup.Item value="long">
          {t.home.long}
          <small>{t.home.longSub}</small>
        </RadioGroup.Item>
        {noClock && (
          <RadioGroup.Item value="none">
            {t.time.noClock}
            <small>∞</small>
          </RadioGroup.Item>
        )}
      </RadioGroup.Root>
      {presets.length > 0 && (
        <RadioGroup.Root
          aria-labelledby={labelledBy}
          value={String(presets.findIndex((p) => sameTime(p, value)))}
          onValueChange={(i) => pick(presets[Number(i)])}
          className="home-times"
        >
          {presets.map((p, i) => (
            <RadioGroup.Item key={i} value={String(i)} className="home-time">
              <b>{p.perMove ? timeLabel(p, t).split(' ').slice(0, 2).join(' ') : timeLabel(p, t)}</b>
              <span>{t.time.groups[timeGroup(p)]}</span>
            </RadioGroup.Item>
          ))}
        </RadioGroup.Root>
      )}
    </div>
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

/** A setting that's on or off, with the design system's switch: a lever, not a pill. */
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
        className={`sl-switch ${big ? 'sl-switch-lg' : ''}`}
      >
        <span className="sl-switch-track" />
      </Switch.Root>
    </label>
  )
}
