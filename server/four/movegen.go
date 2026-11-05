package four

type delta struct{ dx, dy int }

var (
	knightDeltas = []delta{{1, 2}, {2, 1}, {2, -1}, {1, -2}, {-1, -2}, {-2, -1}, {-2, 1}, {-1, 2}}
	kingDeltas   = []delta{{1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}, {-1, -1}, {0, -1}, {1, -1}}
	rookDirs     = []delta{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	bishopDirs   = []delta{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}}
	promoKinds   = []Kind{Queen, Rook, Bishop, Knight}
)

// LegalMoves lists the moves of the player to move. Without check, every move a
// piece can make is legal, kings walking into danger included.
func (p *Position) LegalMoves() []Move {
	moves := make([]Move, 0, 64)
	for i, pc := range p.Board {
		if pc.Is(p.Turn) {
			moves = append(moves, p.PieceMoves(Square(i))...)
		}
	}
	return moves
}

// FindMove looks up a move given as UCI among the legal moves.
func (p *Position) FindMove(uci string) (Move, bool) {
	from, to, promo, err := ParseUCI(uci)
	if err != nil || !p.Board[from].Is(p.Turn) {
		return Move{}, false
	}
	for _, m := range p.PieceMoves(from) {
		if m.To == to && m.Promo == promo {
			return m, true
		}
	}
	return Move{}, false
}

// PieceMoves lists the moves the piece on sq could make if it were its
// player's turn.
func (p *Position) PieceMoves(sq Square) []Move {
	pc := p.Board[sq]
	if pc.IsEmpty() {
		return nil
	}
	var moves []Move
	us := pc.Color()
	// add adds a move to an empty square or a capture of another player's piece,
	// and reports whether a slider can carry on past to.
	add := func(to Square) bool {
		t := p.Board[to]
		switch {
		case t.IsEmpty():
			moves = append(moves, Move{From: sq, To: to})
			return true
		case t.Color() != us:
			moves = append(moves, Move{From: sq, To: to, Flags: FlagCapture})
		}
		return false
	}
	steps := func(ds []delta) {
		for _, d := range ds {
			if to, ok := sq.Offset(d.dx, d.dy); ok {
				add(to)
			}
		}
	}
	slides := func(ds []delta) {
		for _, d := range ds {
			for to, ok := sq.Offset(d.dx, d.dy); ok && add(to); to, ok = to.Offset(d.dx, d.dy) {
			}
		}
	}
	switch pc.Kind() {
	case Pawn:
		return p.pawnMoves(sq, us)
	case Knight:
		steps(knightDeltas)
	case Bishop:
		slides(bishopDirs)
	case Rook:
		slides(rookDirs)
	case Queen:
		slides(rookDirs)
		slides(bishopDirs)
	case King:
		steps(kingDeltas)
		if p.JumpKing {
			steps(knightDeltas)
		}
		moves = append(moves, p.castles(sq, us)...)
	}
	return moves
}

func (p *Position) castles(sq Square, us Color) []Move {
	king, kr, qr := castleSquares(us)
	if sq != king {
		return nil
	}
	rook := NewPiece(us, Rook)
	var moves []Move
	empty := func(files ...int) bool {
		for _, f := range files {
			if !p.Board[cell(us, f, 0)].IsEmpty() {
				return false
			}
		}
		return true
	}
	if p.Castling&(1<<(2*us)) != 0 && p.Board[kr] == rook && empty(5, 6) {
		moves = append(moves, Move{From: sq, To: cell(us, 6, 0), Flags: FlagCastle})
	}
	if p.Castling&(2<<(2*us)) != 0 && p.Board[qr] == rook && empty(1, 2, 3) {
		moves = append(moves, Move{From: sq, To: cell(us, 2, 0), Flags: FlagCastle})
	}
	return moves
}

func (p *Position) pawnMoves(sq Square, us Color) []Move {
	var moves []Move
	fx, fy := forward[us][0], forward[us][1]
	add := func(to Square, flags MoveFlag) {
		if rankOf(us, to) >= PromotionRank {
			for _, k := range promoKinds {
				moves = append(moves, Move{From: sq, To: to, Promo: k, Flags: flags})
			}
			return
		}
		moves = append(moves, Move{From: sq, To: to, Flags: flags})
	}
	enemy := func(to Square) bool { t := p.Board[to]; return !t.IsEmpty() && t.Color() != us }

	if one, ok := sq.Offset(fx, fy); ok && p.Board[one].IsEmpty() {
		if !p.JesterPawns {
			add(one, 0)
		}
		if two, ok := one.Offset(fx, fy); ok && rankOf(us, sq) == 1 && p.Board[two].IsEmpty() {
			moves = append(moves, Move{From: sq, To: two, Flags: FlagDoublePush})
		}
	}
	// The diagonals ahead: sideways is the axis across the pawn's way.
	for _, side := range []int{-1, 1} {
		to, ok := sq.Offset(fx+side*fy*fy, fy+side*fx*fx)
		if !ok {
			continue
		}
		if !p.JesterPawns && enemy(to) {
			add(to, FlagCapture)
		}
		if to == p.EP && p.EPPawn != NoSquare && enemy(p.EPPawn) && p.Board[p.EPPawn].Kind() == Pawn {
			moves = append(moves, Move{From: sq, To: to, Flags: FlagCapture | FlagEnPassant})
		}
	}
	if p.JesterPawns {
		for _, d := range kingDeltas {
			to, ok := sq.Offset(d.dx, d.dy)
			switch {
			case !ok:
			case p.Board[to].IsEmpty():
				add(to, 0)
			case enemy(to):
				add(to, FlagCapture)
			}
		}
	}
	return moves
}

// Attacked reports whether any piece of a player other than us could take on sq.
func (p *Position) Attacked(sq Square, us Color, seen *[NumSquares]bool) bool {
	for i, pc := range p.Board {
		if pc.IsEmpty() || pc.Color() == us || (seen != nil && !seen[i]) {
			continue
		}
		for _, m := range p.PieceMoves(Square(i)) {
			if m.To == sq && m.Is(FlagCapture) {
				return true
			}
		}
	}
	return false
}
