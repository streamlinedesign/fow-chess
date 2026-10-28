package engine

type delta struct{ df, dr int }

var (
	knightDeltas = []delta{{1, 2}, {2, 1}, {2, -1}, {1, -2}, {-1, -2}, {-2, -1}, {-2, 1}, {-1, 2}}
	kingDeltas   = []delta{{1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}, {-1, -1}, {0, -1}, {1, -1}}
	rookDirs     = []delta{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	bishopDirs   = []delta{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}}
	promoKinds   = []Kind{Queen, Rook, Bishop, Knight}
)

func pawnDir(c Color) int {
	if c == White {
		return 1
	}
	return -1
}

// Attacked reports whether any piece of color by attacks sq.
func (p *Position) Attacked(sq Square, by Color) bool {
	// Pawns attack diagonally forward, so look diagonally backward from sq.
	for _, df := range []int{-1, 1} {
		if s, ok := sq.Offset(df, -pawnDir(by)); ok && p.Board[s] == NewPiece(by, Pawn) {
			return true
		}
	}
	for _, d := range knightDeltas {
		if s, ok := sq.Offset(d.df, d.dr); ok && p.Board[s] == NewPiece(by, Knight) {
			return true
		}
	}
	for _, d := range kingDeltas {
		if s, ok := sq.Offset(d.df, d.dr); ok && p.Board[s] == NewPiece(by, King) {
			return true
		}
	}
	if p.JumpKing {
		for _, d := range knightDeltas {
			if s, ok := sq.Offset(d.df, d.dr); ok && p.Board[s] == NewPiece(by, King) {
				return true
			}
		}
	}
	if p.slidingAttack(sq, by, rookDirs, Rook) || p.slidingAttack(sq, by, bishopDirs, Bishop) {
		return true
	}
	return false
}

func (p *Position) slidingAttack(sq Square, by Color, dirs []delta, kind Kind) bool {
	for _, d := range dirs {
		for s, ok := sq.Offset(d.df, d.dr); ok; s, ok = s.Offset(d.df, d.dr) {
			pc := p.Board[s]
			if pc.IsEmpty() {
				continue
			}
			if pc.Color() == by && (pc.Kind() == kind || pc.Kind() == Queen) {
				return true
			}
			break
		}
	}
	return false
}

// PseudoMoves lists every move for the side to move that obeys how pieces move,
// without checking whether it leaves the mover's own king in check. Castling
// already respects the "not out of, through, or into check" rule.
func (p *Position) PseudoMoves() []Move {
	moves := make([]Move, 0, 48)
	for i, pc := range p.Board {
		if pc.Is(p.Turn) {
			moves = p.appendPieceMoves(moves, Square(i), pc)
		}
	}
	return moves
}

// PieceMoves lists the pseudo-legal moves of the piece on sq, whatever side is to move.
func (p *Position) PieceMoves(sq Square) []Move {
	pc := p.Board[sq]
	if pc.IsEmpty() {
		return nil
	}
	return p.appendPieceMoves(nil, sq, pc)
}

func (p *Position) appendPieceMoves(moves []Move, from Square, pc Piece) []Move {
	us := pc.Color()
	switch pc.Kind() {
	case Pawn:
		return p.appendPawnMoves(moves, from, us)
	case Knight:
		return p.appendSteps(moves, from, us, knightDeltas)
	case Bishop:
		return p.appendSlides(moves, from, us, bishopDirs)
	case Rook:
		return p.appendSlides(moves, from, us, rookDirs)
	case Queen:
		moves = p.appendSlides(moves, from, us, rookDirs)
		return p.appendSlides(moves, from, us, bishopDirs)
	case King:
		moves = p.appendSteps(moves, from, us, kingDeltas)
		if p.JumpKing {
			moves = p.appendSteps(moves, from, us, knightDeltas)
		}
		return p.appendCastles(moves, from, us)
	}
	return moves
}

func (p *Position) appendSteps(moves []Move, from Square, us Color, deltas []delta) []Move {
	for _, d := range deltas {
		to, ok := from.Offset(d.df, d.dr)
		if !ok {
			continue
		}
		switch target := p.Board[to]; {
		case target.IsEmpty():
			moves = append(moves, Move{From: from, To: to})
		case target.Color() != us:
			moves = append(moves, Move{From: from, To: to, Flags: FlagCapture})
		}
	}
	return moves
}

func (p *Position) appendSlides(moves []Move, from Square, us Color, dirs []delta) []Move {
	for _, d := range dirs {
		for to, ok := from.Offset(d.df, d.dr); ok; to, ok = to.Offset(d.df, d.dr) {
			target := p.Board[to]
			if target.IsEmpty() {
				moves = append(moves, Move{From: from, To: to})
				continue
			}
			if target.Color() != us {
				moves = append(moves, Move{From: from, To: to, Flags: FlagCapture})
			}
			break
		}
	}
	return moves
}

func (p *Position) appendPawnMoves(moves []Move, from Square, us Color) []Move {
	dir := pawnDir(us)
	lastRank, startRank := 7, 1
	if us == Black {
		lastRank, startRank = 0, 6
	}

	add := func(m Move) {
		if m.To.Rank() == lastRank {
			for _, k := range promoKinds {
				m.Promo = k
				moves = append(moves, m)
			}
			return
		}
		moves = append(moves, m)
	}

	if one, ok := from.Offset(0, dir); ok && p.Board[one].IsEmpty() {
		add(Move{From: from, To: one})
		if from.Rank() == startRank {
			if two, _ := one.Offset(0, dir); p.Board[two].IsEmpty() {
				moves = append(moves, Move{From: from, To: two, Flags: FlagDoublePush})
			}
		}
	}
	// A backward step never captures and never promotes.
	if p.BackPawns {
		if back, ok := from.Offset(0, -dir); ok && p.Board[back].IsEmpty() {
			moves = append(moves, Move{From: from, To: back})
		}
	}

	for _, df := range []int{-1, 1} {
		to, ok := from.Offset(df, dir)
		if !ok {
			continue
		}
		if target := p.Board[to]; !target.IsEmpty() && target.Color() != us {
			add(Move{From: from, To: to, Flags: FlagCapture})
		} else if to == p.EP && us == p.Turn && target.IsEmpty() {
			moves = append(moves, Move{From: from, To: to, Flags: FlagCapture | FlagEnPassant})
		}
	}
	return moves
}

func (p *Position) appendCastles(moves []Move, from Square, us Color) []Move {
	rank, kingSide, queenSide := 0, CastleWK, CastleWQ
	if us == Black {
		rank, kingSide, queenSide = 7, CastleBK, CastleBQ
	}
	if from != SquareAt(4, rank) || p.Castling&(kingSide|queenSide) == 0 {
		return moves
	}
	them := us.Other()
	// Without check, the king may castle out of, through, or into attack.
	attacked := func(sq Square) bool { return !p.KingCapture && p.Attacked(sq, them) }
	if attacked(from) {
		return moves
	}
	empty := func(files ...int) bool {
		for _, f := range files {
			if !p.Board[SquareAt(f, rank)].IsEmpty() {
				return false
			}
		}
		return true
	}
	rook := NewPiece(us, Rook)
	// The destination square is checked by the legality filter like any king move.
	if p.Castling&kingSide != 0 && p.Board[SquareAt(7, rank)] == rook && empty(5, 6) && !attacked(SquareAt(5, rank)) {
		moves = append(moves, Move{From: from, To: SquareAt(6, rank), Flags: FlagCastle})
	}
	if p.Castling&queenSide != 0 && p.Board[SquareAt(0, rank)] == rook && empty(1, 2, 3) && !attacked(SquareAt(3, rank)) {
		moves = append(moves, Move{From: from, To: SquareAt(2, rank), Flags: FlagCastle})
	}
	return moves
}

// LegalMoves lists the moves that don't leave the mover's king in check.
func (p *Position) LegalMoves() []Move {
	pseudo := p.PseudoMoves()
	legal := pseudo[:0]
	for _, m := range pseudo {
		if p.isLegal(m) {
			legal = append(legal, m)
		}
	}
	return legal
}

func (p *Position) isLegal(m Move) bool {
	if p.KingCapture {
		return true
	}
	next := p.Apply(m)
	k := next.KingSquare(p.Turn)
	return k == NoSquare || !next.Attacked(k, next.Turn)
}

func (p *Position) HasLegalMove() bool {
	for _, m := range p.PseudoMoves() {
		if p.isLegal(m) {
			return true
		}
	}
	return false
}

// FindMove matches a UCI string such as "e7e8q" against the legal moves.
func (p *Position) FindMove(uci string) (Move, bool) {
	if len(uci) < 4 || len(uci) > 5 {
		return Move{}, false
	}
	from, err1 := ParseSquare(uci[:2])
	to, err2 := ParseSquare(uci[2:4])
	if err1 != nil || err2 != nil {
		return Move{}, false
	}
	promo := NoKind
	if len(uci) == 5 {
		promo = kindFromLetter(uci[4])
		if promo == NoKind {
			return Move{}, false
		}
	}
	for _, m := range p.LegalMoves() {
		if m.From == from && m.To == to && m.Promo == promo {
			return m, true
		}
	}
	return Move{}, false
}
