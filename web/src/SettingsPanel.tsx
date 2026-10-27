import * as RadioGroup from '@radix-ui/react-radio-group'
import * as Switch from '@radix-ui/react-switch'
import type { Settings } from './settings'

export function SettingsPanel({ value, onChange }: { value: Settings; onChange: (s: Settings) => void }) {
  return (
    <fieldset className="flex flex-col gap-4 rounded-2xl p-5 shadow-[inset_0_0_0_2px_var(--color-fg)]">
      <legend className="sr-only">Match settings</legend>

      <label className="flex cursor-pointer items-center justify-between gap-4">
        <span>
          <span className="block text-lg font-semibold">Fog of war</span>
          <span className="block text-sm text-accent">You only see what your pieces can reach.</span>
        </span>
        <Switch.Root
          checked={value.fog}
          onCheckedChange={(fog) => onChange({ ...value, fog })}
          className="relative h-7 w-12 shrink-0 rounded-full shadow-[inset_0_0_0_2px_var(--color-fg)] transition-colors data-[state=checked]:bg-fg"
        >
          <Switch.Thumb className="block size-5 translate-x-1 rounded-full bg-fg transition-transform data-[state=checked]:translate-x-6 data-[state=checked]:bg-bg" />
        </Switch.Root>
      </label>

      <div className="flex flex-col gap-2">
        <span className="text-lg font-semibold" id="win-by">
          Win by
        </span>
        <RadioGroup.Root
          aria-labelledby="win-by"
          value={value.kingCapture ? 'capture' : 'mate'}
          onValueChange={(v) => onChange({ ...value, kingCapture: v === 'capture' })}
          className="grid grid-cols-2 gap-2"
        >
          {[
            ['capture', 'Eat the King', 'No check. Kings can walk into danger.'],
            ['mate', 'Checkmate', 'Normal check rules.'],
          ].map(([v, label, hint]) => (
            <RadioGroup.Item
              key={v}
              value={v}
              className="flex flex-col items-start gap-0.5 rounded-xl p-3 text-left shadow-[inset_0_0_0_2px_color-mix(in_srgb,var(--color-fg)_25%,transparent)] transition-colors data-[state=checked]:bg-fg data-[state=checked]:text-bg data-[state=checked]:shadow-none"
            >
              <span className="font-semibold">{label}</span>
              <span className="text-xs opacity-70">{hint}</span>
            </RadioGroup.Item>
          ))}
        </RadioGroup.Root>
      </div>
    </fieldset>
  )
}
