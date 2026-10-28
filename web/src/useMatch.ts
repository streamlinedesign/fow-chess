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
  /** Your king's square when a piece you can see attacks it. */
  danger?: string
  /** Squares your king could step to that a piece you can see attacks. */
  unsafe?: string[]
  /** Moves so far in algebraic notation. */
  history: string[]
  /** Pieces each side has taken, as FEN letters, most valuable first. */
  captured: Record<Color, string>
  /** setup: both players are arranging their pieces, in a game with a setup mode. */
  status: 'waiting' | 'setup' | 'playing' | 'over'
  result?: '1-0' | '0-1' | '1/2-1/2'
  reason?: string
  /** A game won on the winner's very first move. It moves ratings a quarter as far as a whole game. */
  sniper?: boolean
  /** During setup: who is ready, what you placed, and the opponent's setup once they're ready in a game without fog. */
  setup?: { ready: Record<Color, boolean>; mine?: string; theirs?: string }
  /** name is the username of a signed-in player, absent for a guest. */
  players: Record<Color, { taken: boolean; online: boolean; name?: string }>
  rematch: Record<Color, boolean>
  draw: Record<Color, boolean>
  /** Time left as of when this state was sent, and whose clock is ticking. Absent for No clock games. */
  clock?: { left: Record<Color, number>; running?: Color }
  /** Whether the game counts toward rating: two signed-in players and a clock. */
  rated: boolean
  /** Each side's rating change, once a rated game is over and saved. */
  ratingDiff?: Record<Color, number>
  /** The side a spectator sees the game as. Absent for players. */
  watching?: Color
  /** How many moves a spectator of a fog game is kept behind the players. */
  behind?: number
  /** People watching, not counting the players. */
  spectators: number
  /** When this state arrived, by performance.now(), for counting the clock down. Set here, not by the server. */
  receivedAt: number
}

/** A chat message between the two players. The server keeps them only while the game lasts. */
export type ChatLine = { id: number; from: Color; text: string; at: number }

export type Outgoing =
  | { type: 'move'; uci: string }
  | { type: 'resign' }
  | { type: 'draw' }
  | { type: 'rematch' }
  | { type: 'chat'; text: string }
  | { type: 'watch'; side: Color }
  /** Your pieces, 64 squares from a1 to h8. Empty takes back a setup you sent. */
  | { type: 'setup'; board: string }

export type Connection = 'connecting' | 'open' | 'closed' | 'failed' | 'missing' | 'outdated'

/** Must match protocolVersion in server/match.go. */
const PROTOCOL = 10

export function useMatch(id: string) {
  const [state, setState] = useState<MatchState | null>(null)
  const [conn, setConn] = useState<Connection>('connecting')
  const [rtt, setRtt] = useState<number | null>(null)
  const [chat, setChat] = useState<ChatLine[]>([])
  const ws = useRef<WebSocket | null>(null)
  const lastPong = useRef(0)

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
          setState({ ...msg, receivedAt: performance.now() })
        } else if (msg.type === 'chat') {
          // Players get the whole chat on (re)connecting and at a rematch, then each new line.
          setChat((c) => (msg.reset ? msg.lines : [...c, ...msg.lines]))
        } else if (msg.type === 'pong') {
          lastPong.current = performance.now()
          setRtt(Math.round(performance.now() - msg.t))
        }
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

    // Reconnect right away when a phone comes back to the tab. A socket the
    // phone slept on may be dead without the browser knowing yet, so ask the
    // server for a pong and drop the socket if none comes.
    let health: ReturnType<typeof setTimeout>
    const onVisible = () => {
      if (document.visibilityState !== 'visible') return
      const sock = ws.current
      if (!sock) {
        clearTimeout(timer)
        retry = 0
        connect()
      } else if (sock.readyState === WebSocket.OPEN) {
        const before = lastPong.current
        ping()
        clearTimeout(health)
        health = setTimeout(() => {
          if (ws.current === sock && lastPong.current === before) {
            retry = 0
            sock.close()
          }
        }, 3000)
      }
    }

    // Round-trip time to the server, shown on the match page for play testing.
    const ping = () => {
      if (ws.current?.readyState === WebSocket.OPEN)
        ws.current.send(JSON.stringify({ type: 'ping', t: performance.now() }))
    }
    const pinger = setInterval(ping, 5000)

    connect()
    document.addEventListener('visibilitychange', onVisible)
    return () => {
      stopped = true
      clearTimeout(timer)
      clearTimeout(health)
      clearInterval(pinger)
      document.removeEventListener('visibilitychange', onVisible)
      ws.current?.close()
    }
  }, [id])

  useEffect(() => setChat([]), [id])

  const send = useCallback((msg: Outgoing) => {
    if (ws.current?.readyState !== WebSocket.OPEN) return false
    ws.current.send(JSON.stringify(msg))
    return true
  }, [])

  return { state, conn, send, rtt, chat }
}
