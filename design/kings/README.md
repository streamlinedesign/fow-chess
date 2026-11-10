# Proposed kings

Four new kings for the mods, drawn in every piece set. Nothing here is in the game yet:
the art lives outside `web/src` so the app doesn't carry pieces it can't use, and each
king moves into `web/src/pieces/` with its rule. `python3 scripts/kings.py` redraws them
all from the classic silhouettes; open `index.html` here for the contact sheet.

| File | King | Drawn as |
|---|---|---|
| `E` | The Eye | a lidless eye with a slit pupil under a crown of flame |
| `S` | Swap king | rook battlements for a crown, swap arrows on the cup |
| `H` | Pawn king | a pawn with a small cross on its head |
| `I` | Infinity king | an infinity sign where the cross was |

`E` and `I` are free letters next to `J` (jester pawn) and `X` (prince); `H` is for hidden.

## Rules

Each is a mod like the prince, and works with fog of war and Eat the King as they are.

**The Eye.** Sees like a queen, along all eight lines until a piece blocks the view,
but still moves one square like a king. Its gaze gives it away: the other player always
sees the square the Eye stands on, fog or not.

**Swap king.** As its move, the king can swap places with any of its own rooks, wherever
that rook is on the board. A swap takes nothing and costs the whole turn. It can't swap
two turns running, so it can't dodge forever. Castling stays as it is.

**Pawn king.** Starts on e2 (e7 for Black) in place of that pawn, with e1 left empty, and
moves like a king. The other player sees it drawn as a plain pawn, so they have to work
out which of eight pawns to eat. It gives itself away by moving in a way no pawn can.
No castling.

**Infinity king.** Nothing can take it from a distance: only a piece moving one square
(a king, a pawn, or a queen, rook or bishop standing next to it) can capture it. A
knight can never take it. It moves like a king.

## Open

- Four-player sets: the art is drawn for the four seat colours; the rules above are for
  two players first.
- The pawn king needs the server to send the king as a pawn in the other player's view
  (`server/view.go`); the Eye needs the other player's view to show its square.
- Bots (`server/bots`, `chess-bots`) need to know each rule before it can be played
  against them.
