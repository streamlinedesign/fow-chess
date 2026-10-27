import { useEffect, useState } from 'react'

/** Formats time left: 1:30:00, 4:59, 0:19.4 under 20 seconds, or "23 h 40 min" for Daily. */
export function formatClock(ms: number, daily = false) {
  const t = Math.max(0, ms)
  const s = Math.floor(t / 1000)
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  if (daily && h > 0) return `${h} h ${m} min`
  if (h > 0) return `${h}:${String(m).padStart(2, '0')}:${String(s % 60).padStart(2, '0')}`
  if (t < 20_000) return `0:${String(s).padStart(2, '0')}.${Math.floor((t % 1000) / 100)}`
  return `${m}:${String(s % 60).padStart(2, '0')}`
}

/**
 * A player's clock. The server sends the time left when each state goes out; a
 * running clock counts down locally from when that state arrived.
 */
export function Clock({
  left,
  running,
  since,
  daily,
}: {
  left: number
  running: boolean
  since: number
  daily: boolean
}) {
  const [now, setNow] = useState(() => performance.now())
  useEffect(() => {
    if (!running) return
    const id = setInterval(() => setNow(performance.now()), 100)
    return () => clearInterval(id)
  }, [running])
  const ms = running ? left - Math.max(0, now - since) : left
  return (
    <span className="clock" data-running={running} data-low={!daily && ms < 20_000} role="timer">
      {formatClock(ms, daily)}
    </span>
  )
}
