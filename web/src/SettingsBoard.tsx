import { useT } from './i18n'
import { pieceImg, squareIndex } from './position'
import { HILL, setupMode, type Settings } from './settings'

const FILES = 'abcdefgh'
const START = 'RNBQKBNRPPPPPPPP' + '.'.repeat(32) + 'pppppppprnbqkbnr'
/** Where the king on e1 could jump to at the start. */
const JUMPS = ['d3', 'f3']

/**
 * A still board next to the match settings, showing what they mean from
 * White's seat at the start: the fog, the rows you'd arrange, the hill, and
 * where a jump king could go.
 */
export function SettingsBoard({
  settings,
  boardClassName = '',
  captionClassName = '',
}: {
  settings: Settings
  boardClassName?: string
  captionClassName?: string
}) {
  const t = useT()
  const setup = setupMode(settings)
  const squares = []
  for (let row = 0; row < 8; row++) {
    for (let col = 0; col < 8; col++) {
      const sq = FILES[col] + (8 - row)
      const i = squareIndex(sq)
      const rank = i >> 3
      // At the start your pieces see your own four ranks: the pawns reach rank 4.
      const fog = settings.fog && rank >= 4
      const piece = fog ? null : START[i] === '.' ? null : START[i]
      squares.push(
        <div
          key={sq}
          className="sq"
          data-square={sq}
          data-light={((i & 7) + rank) % 2 === 1}
          data-fog={fog}
          data-zone={setup.rows > 0 && rank < setup.rows && !(setup.pawnsFixed && rank === 1)}
          data-hill={settings.hill && HILL.includes(sq)}
          data-target={settings.jumpKing && JUMPS.includes(sq) ? 'move' : undefined}
        >
          {col === 0 && <span className="coord top-[4%] left-[5%]">{sq[1]}</span>}
          {row === 7 && <span className="coord right-[5%] bottom-[4%]">{sq[0]}</span>}
          {piece && <img className="piece" src={pieceImg(piece)} alt="" draggable={false} />}
        </div>,
      )
    }
  }

  const notes = [
    settings.fog ? t.settingsBoard.fogOn : t.settingsBoard.fogOff,
    settings.kingCapture ? t.settingsBoard.eat : t.settingsBoard.mate,
    settings.hill && t.settingsBoard.hill,
    setup.value === 'backRow' && t.settingsBoard.backRow,
    setup.value === 'battleship' && t.settingsBoard.battleship,
    settings.jumpKing && t.settingsBoard.jump,
    settings.backPawns && t.settingsBoard.backPawns,
  ].filter(Boolean)

  return (
    <>
      <div
        className={`board setup-board grid aspect-square w-full grid-cols-8 grid-rows-8 overflow-hidden rounded-md shadow-[0_0_0_2px_var(--color-fg)] ${boardClassName}`}
        role="img"
        aria-label={`${t.settingsBoard.aria} ${notes.join(' ')}`}
      >
        {squares}
      </div>
      <div className={`flex flex-col gap-1.5 ${captionClassName}`}>
        <p className="text-[15px] font-semibold">{t.settingsBoard.yourView}</p>
        <p className="text-sm text-accent">{notes.join(' ')}</p>
      </div>
    </>
  )
}
