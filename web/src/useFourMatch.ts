import { useCallback, useEffect, useRef, useState } from 'react'
import type { Seat } from './four'
import type { Skin } from './pieces'
import { playerId } from './player'
import { skinOf, usePrefs } from './prefs'
import type { TimeControl } from './settings'

/** Must match fourSettings in server/four_match.go. */
export type FourSettings = {
  fog: boolean
  time: TimeControl
  jumpKing?: boolean
  jesterPawns?: boolean
  hill?: boolean
  centerVision?: '' | 'pawn' | 'any'
  /** Each player arranges their own pieces first. */
  setup?: '' | 'backRow' | 'battleship'
  /** Sync Move: everyone moves at once each round. */
  sync?: boolean
}

export type FourSeatState = {
  taken: boolean
  online: boolean
  name?: string
  bot?: boolean
  /** How the player left the game: "capturing the king", "resignation" or "time". */
  out?: string
  /** Has sent a move for this Sync Move round, or placed their pieces. */
  ready?: boolean
  /** The player's own skin, and which of its two sets they play in here. */
  skin?: Skin
  half?: 'w' | 'b'
}

/** Must match fourState in server/four_match.go. */
export type FourState = {
  id: string
  you: Seat | ''
  settings: FourSettings
  /** 196 squares from the bottom left, row by row: " " off the board, "." empty, "?" hidden, or a piece like "rK". */
  board: string[]
  turn: Seat
  ply: number
  legal: string[]
  last: { from: string; to: string } | null
  /** Every move of the last Sync Move round, when you can see them all. */
  lastRound?: { from: string; to: string }[]
  danger?: string
  dangerHidden?: boolean
  fullVision?: boolean
  /** Every move and leaving; san is "?" for a move you can't know. */
  history: { by: Seat; san?: string; out?: string; round?: number }[]
  /** The pieces each player has taken that you know of, as codes like "gQ". */
  captured: Record<Seat, string[]>
  status: 'waiting' | 'setup' | 'playing' | 'over'
  /** The winner's seat, or "draw". */
  result?: Seat | 'draw'
  reason?: string
  players: Record<Seat, FourSeatState>
  clock?: { left: Record<Seat, number>; running?: Seat }
  watching?: Seat
  behind?: number
  spectators: number
  /** Whose clocks run in a Sync Move round. */
  ticking?: Seat[]
  /** The move you've sent for this Sync Move round. */
  planned?: string
  /** Your arrangement, once placed, while players arrange their pieces. */
  arranged?: string
  receivedAt: number
}

export type FourChatLine = { id: number; from: Seat; text: string; at: number }

export type FourOutgoing =
  | { type: 'move'; uci: string }
  | { type: 'setup'; board: string }
  | { type: 'resign' }
  | { type: 'fill' }
  | { type: 'chat'; text: string }
  | { type: 'watch'; side: Seat }

export type FourConnection = 'connecting' | 'open' | 'closed' | 'failed' | 'missing' | 'outdated'

/** Must match fourProtocol in server/four_match.go. */
const FOUR_PROTOCOL = 3

/** The live connection to a four-player match, reconnecting as useMatch does. */
export function useFourMatch(id: string) {
  const [state, setState] = useState<FourState | null>(null)
  const [conn, setConn] = useState<FourConnection>('connecting')
  const [chat, setChat] = useState<FourChatLine[]>([])
  const ws = useRef<WebSocket | null>(null)

  useEffect(() => {
    let stopped = false
    let retry = 0
    let timer: ReturnType<typeof setTimeout>

    const connect = () => {
      const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
      const sock = new WebSocket(`${proto}//${location.host}/ws/four/${id}?player=${playerId}`)
      ws.current = sock
      if (retry < 3) setConn('connecting')
      sock.onopen = () => {
        retry = 0
        setConn('open')
        sendSkin()
      }
      sock.onmessage = (e) => {
        const msg = JSON.parse(e.data)
        if (msg.type === 'state') {
          if (msg.protocol !== FOUR_PROTOCOL) {
            stopped = true
            sock.close()
            return setConn('outdated')
          }
          setState({ ...msg, receivedAt: performance.now() })
        } else if (msg.type === 'chat') {
          setChat((c) => (msg.reset ? msg.lines : [...c, ...msg.lines]))
        }
      }
      sock.onclose = async () => {
        if (stopped || ws.current !== sock) return
        ws.current = null
        const res = await fetch(`/api/four/${id}`).catch(() => null)
        if (stopped || ws.current) return
        if (res && res.status === 404) return setConn('missing')
        setConn(retry >= 3 ? 'failed' : 'closed')
        timer = setTimeout(connect, Math.min(500 * 2 ** retry++, 8000))
      }
    }

    const onVisible = () => {
      if (document.visibilityState === 'visible' && !ws.current) {
        clearTimeout(timer)
        retry = 0
        connect()
      }
    }
    // Keep the socket warm through proxies that drop quiet connections.
    const pinger = setInterval(() => {
      if (ws.current?.readyState === WebSocket.OPEN) ws.current.send(JSON.stringify({ type: 'ping', t: 0 }))
    }, 20000)

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

  useEffect(() => setChat([]), [id])

  // Your skin goes to the match on connecting and whenever you change it.
  const prefs = usePrefs()
  const skin = JSON.stringify(skinOf(prefs))
  const skinRef = useRef(skin)
  skinRef.current = skin
  const sendSkin = () => {
    if (ws.current?.readyState === WebSocket.OPEN) ws.current.send(`{"type":"skin","skin":${skinRef.current}}`)
  }
  useEffect(sendSkin, [skin])

  const send = useCallback((msg: FourOutgoing) => {
    if (ws.current?.readyState !== WebSocket.OPEN) return false
    ws.current.send(JSON.stringify(msg))
    return true
  }, [])

  return { state, conn, send, chat }
}
