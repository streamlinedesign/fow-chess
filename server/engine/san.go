package engine

import "strings"

// SAN returns m in standard algebraic notation (e.g. "Nbd7", "exd6", "e8=Q+", "O-O").
// m must be legal in p.
func (p *Position) SAN(m Move) string {
	var b strings.Builder
	pc := p.Board[m.From]

	switch {
	case m.Is(FlagCastle):
		if m.To.File() == 6 {
			b.WriteString("O-O")
		} else {
			b.WriteString("O-O-O")
		}
	case pc.Kind() == Pawn:
		if m.Is(FlagCapture) {
			b.WriteByte('a' + byte(m.From.File()))
			b.WriteByte('x')
		}
		b.WriteString(m.To.String())
		if m.Promo != NoKind {
			b.WriteByte('=')
			b.WriteByte(m.Promo.Letter())
		}
	default:
		b.WriteByte(pc.Kind().Letter())
		b.WriteString(p.disambiguation(m, pc))
		if m.Is(FlagCapture) {
			b.WriteByte('x')
		}
		b.WriteString(m.To.String())
	}

	next := p.Apply(m)
	if next.InCheck() {
		if next.HasLegalMove() {
			b.WriteByte('+')
		} else {
			b.WriteByte('#')
		}
	}
	return b.String()
}

// disambiguation adds the origin file, rank, or both when another piece of the
// same kind could also legally move to the same square.
func (p *Position) disambiguation(m Move, pc Piece) string {
	sameFile, sameRank, other := false, false, false
	for _, o := range p.LegalMoves() {
		if o.To != m.To || o.From == m.From || p.Board[o.From] != pc {
			continue
		}
		other = true
		if o.From.File() == m.From.File() {
			sameFile = true
		}
		if o.From.Rank() == m.From.Rank() {
			sameRank = true
		}
	}
	switch {
	case !other:
		return ""
	case !sameFile:
		return string('a' + byte(m.From.File()))
	case !sameRank:
		return string('1' + byte(m.From.Rank()))
	default:
		return m.From.String()
	}
}
