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
    settings.fog ? 'Fog covers everything your pieces can’t see.' : 'No fog: the whole board shows.',
    settings.kingCapture ? 'Win by eating the king.' : 'Win by checkmate.',
    settings.hill && 'Or get your king to a ringed centre square.',
    setup.value === 'backRow' && 'Arrange your back row first.',
    setup.value === 'battleship' && 'Place every piece anywhere in the ringed rows first.',
    settings.jumpKing && 'Your king can also jump, to the dotted squares here.',
    settings.backPawns && 'Pawns can step back, and see behind them.',
  ].filter(Boolean)

  return (
    <>
      <div
        className={`board setup-board grid aspect-square w-full grid-cols-8 grid-rows-8 overflow-hidden rounded-md shadow-[0_0_0_2px_var(--color-fg)] ${boardClassName}`}
        role="img"
        aria-label={`The board at the start with these settings. ${notes.join(' ')}`}
      >
        {squares}
      </div>
      <div className={`flex flex-col gap-1.5 ${captionClassName}`}>
        <p className="text-[15px] font-semibold">Your view at the start</p>
        <p className="text-sm text-accent">{notes.join(' ')}</p>
      </div>
    </>
  )
}
