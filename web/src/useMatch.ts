import { useCallback, useEffect, useRef, useState } from 'react'
import { playerId } from './player'
import type { Settings } from './settings'

export type Color = 'w' | 'b'

export type MatchState = {
  id: string
  round: number
  you: Color | ''
  settings: Settings
  /** 64 chars a1..h8; see position.ts */
  board: string
  turn: Color
  /** Half-moves played; bumps on every move so stale local predictions can be spotted. */
  ply: number
  /** UCI moves you may play now. Empty unless it's your turn. */
  legal: string[]
  last: { from: string; to: string } | null
  check?: string
  /** Moves so far in algebraic notation. */
  history: string[]
  status: 'waiting' | 'playing' | 'over'
  result?: '1-0' | '0-1' | '1/2-1/2'
  reason?: string
  players: Record<Color, { taken: boolean; online: boolean }>
  rematch: Record<Color, boolean>
  draw: Record<Color, boolean>
}

export type Outgoing = { type: 'move'; uci: string } | { type: 'resign' } | { type: 'draw' } | { type: 'rematch' }

export type Connection = 'connecting' | 'open' | 'closed' | 'failed' | 'missing' | 'outdated'

/** Must match protocolVersion in server/match.go. */
const PROTOCOL = 3

export function useMatch(id: string) {
  const [state, setState] = useState<MatchState | null>(null)
  const [conn, setConn] = useState<Connection>('connecting')
  const [rtt, setRtt] = useState<number | null>(null)
  const ws = useRef<WebSocket | null>(null)

  useEffect(() => {
    let stopped = false
    let retry = 0
    let timer: ReturnType<typeof setTimeout>

    const connect = () => {
      const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
      const sock = new WebSocket(`${proto}//${location.host}/ws/${id}?player=${playerId}`)
      ws.current = sock
      if (retry < 3) setConn('connecting')
      sock.onopen = () => {
        retry = 0
        setConn('open')
        ping()
      }
      sock.onmessage = (e) => {
        const msg = JSON.parse(e.data)
        if (msg.type === 'state') {
          // A server from another version sends state this app can't draw.
          if (msg.protocol !== PROTOCOL) {
            stopped = true
            sock.close()
            return setConn('outdated')
          }
          setState(msg)
        }
        else if (msg.type === 'pong') setRtt(Math.round(performance.now() - msg.t))
      }
      sock.onclose = async () => {
        // A socket from an earlier effect run (React StrictMode mounts twice in dev)
        // closes after the new one opened; it must not clear the live socket.
        if (stopped || ws.current !== sock) return
        ws.current = null
        // A 404 on the upgrade means the match is gone; don't retry forever.
        const res = await fetch(`/api/matches/${id}`).catch(() => null)
        if (stopped || ws.current) return
        if (res && res.status === 404) return setConn('missing')
        setConn(retry >= 3 ? 'failed' : 'closed')
        timer = setTimeout(connect, Math.min(500 * 2 ** retry++, 8000))
      }
    }

    // Reconnect right away when a phone comes back to the tab.
    const onVisible = () => {
      if (document.visibilityState === 'visible' && !ws.current) {
        clearTimeout(timer)
        retry = 0
        connect()
      }
    }

    // Round-trip time to the server, shown on the match page for play testing.
    const ping = () => {
      if (ws.current?.readyState === WebSocket.OPEN) ws.current.send(JSON.stringify({ type: 'ping', t: performance.now() }))
    }
    const pinger = setInterval(ping, 5000)

    connect()
    document.addEventListener('visibilitychange', onVisible)
    return () => {
      stopped = true
      clearTimeout(timer)
      clearInterval(pinger)
      document.removeEventListener('visibilitychange', onVisible)
      ws.current?.close()
    }
  }, [id])

  const send = useCallback((msg: Outgoing) => {
    if (ws.current?.readyState !== WebSocket.OPEN) return false
    ws.current.send(JSON.stringify(msg))
    return true
  }, [])

  return { state, conn, send, rtt }
}
