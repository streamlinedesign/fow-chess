package sim

import (
	"fmt"
	"io"
	"math"
	"slices"
	"sort"
	"strings"
)

// Group is the games played under one set of conditions.
type Group struct {
	Key                    string
	Mode, Rules, White, Bk string
	Fog                    bool
	N                      int
	WhiteWins, BlackWins   int
	Draws                  int
	Snipers                int
	KingKills, HiddenKills int
	Plies                  []int
	Methods, Killers       map[string]int
	firstCapture, captured int
	contact, contacted     int
	vision                 float64
}

func groupKey(r *Record) string {
	fog := "fog"
	if !r.Fog {
		fog = "nofog"
	}
	mode := r.Mode
	if r.Guarded {
		mode += "(guarded)"
	}
	return strings.Join([]string{mode, fog, r.Rules, r.White, r.Black}, " ")
}

// Stats gathers records into groups by mode, fog, rules and players.
type Stats struct{ groups map[string]*Group }

func NewStats() *Stats { return &Stats{groups: map[string]*Group{}} }

func (s *Stats) Add(r *Record) {
	k := groupKey(r)
	g := s.groups[k]
	if g == nil {
		mode := r.Mode
		if r.Guarded {
			mode += " (guarded setups)"
		}
		g = &Group{Key: k, Mode: mode, Rules: r.Rules, White: r.White, Bk: r.Black, Fog: r.Fog,
			Methods: map[string]int{}, Killers: map[string]int{}}
		s.groups[k] = g
	}
	g.N++
	switch r.Result {
	case "1-0":
		g.WhiteWins++
	case "0-1":
		g.BlackWins++
	default:
		g.Draws++
	}
	if r.Sniper {
		g.Snipers++
	}
	if r.Killer != "" {
		g.KingKills++
		g.Killers[r.Killer]++
		if r.KillerHidden {
			g.HiddenKills++
		}
	}
	g.Plies = append(g.Plies, r.Plies)
	g.Methods[r.Method]++
	if r.FirstCapture >= 0 {
		g.firstCapture += r.FirstCapture
		g.captured++
	}
	if r.FirstContact >= 0 {
		g.contact += r.FirstContact
		g.contacted++
	}
	g.vision += (r.Vision[0] + r.Vision[1]) / 2
}

// WhiteScore is White's average score (win 1, draw ½) and its 95% margin.
func (g *Group) WhiteScore() (score, margin float64) {
	n := float64(g.N)
	score = (float64(g.WhiteWins) + 0.5*float64(g.Draws)) / n
	sq := (float64(g.WhiteWins) + 0.25*float64(g.Draws)) / n
	variance := sq - score*score
	return score, 1.96 * math.Sqrt(variance/n)
}

func pct(a, n int) string { return fmt.Sprintf("%.1f%%", 100*float64(a)/float64(n)) }

func (s *Stats) sorted() []*Group {
	out := make([]*Group, 0, len(s.groups))
	for _, g := range s.groups {
		out = append(out, g)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// Print writes a summary of every group, then how fog changes each matchup
// that was played both with and without it.
func (s *Stats) Print(w io.Writer) {
	for _, g := range s.sorted() {
		fog := "fog on"
		if !g.Fog {
			fog = "fog off"
		}
		fmt.Fprintf(w, "\n%s vs %s · %s · %s · %s · %d games\n", g.White, g.Bk, g.Mode, fog, g.Rules, g.N)
		score, margin := g.WhiteScore()
		fmt.Fprintf(w, "  White %s  Draw %s  Black %s   White scores %.3f ± %.3f\n",
			pct(g.WhiteWins, g.N), pct(g.Draws, g.N), pct(g.BlackWins, g.N), score, margin)
		p := slices.Clone(g.Plies)
		slices.Sort(p)
		sum := 0
		for _, x := range p {
			sum += x
		}
		fmt.Fprintf(w, "  Length (plies): mean %.0f, median %d, 10%% under %d, 90%% under %d\n",
			float64(sum)/float64(g.N), p[len(p)/2], p[len(p)/10], p[len(p)*9/10])
		fmt.Fprintf(w, "  Ended by: %s\n", counts(g.Methods, g.N))
		if g.KingKills > 0 {
			fmt.Fprintf(w, "  King eaten by: %s\n", counts(g.Killers, g.KingKills))
			if g.Fog {
				fmt.Fprintf(w, "  Eaten by a piece the loser couldn't see: %s of king kills\n", pct(g.HiddenKills, g.KingKills))
			}
		}
		if g.Snipers > 0 {
			fmt.Fprintf(w, "  Sniper wins: %s of games\n", pct(g.Snipers, g.N))
		}
		if g.captured > 0 {
			fmt.Fprintf(w, "  First capture on ply %.1f on average", float64(g.firstCapture)/float64(g.captured))
			if g.contacted > 0 {
				fmt.Fprintf(w, ", first sight of an enemy piece on ply %.1f", float64(g.contact)/float64(g.contacted))
			}
			fmt.Fprintf(w, "\n")
		}
		if g.Fog {
			fmt.Fprintf(w, "  Board seen per turn: %.0f%%\n", 100*g.vision/float64(g.N))
		}
	}

	var lines []string
	for _, on := range s.sorted() {
		if !on.Fog {
			continue
		}
		off := s.groups[strings.Replace(on.Key, " fog ", " nofog ", 1)]
		if off == nil {
			continue
		}
		a, ma := on.WhiteScore()
		b, mb := off.WhiteScore()
		lines = append(lines, fmt.Sprintf("  %s vs %s · %s · %s: White scores %.3f with fog, %.3f without (difference %+.3f ± %.3f); games last %.0f vs %.0f plies; draws %s vs %s",
			on.White, on.Bk, on.Mode, on.Rules, a, b, a-b, math.Sqrt(ma*ma+mb*mb), mean(on.Plies), mean(off.Plies),
			pct(on.Draws, on.N), pct(off.Draws, off.N)))
	}
	if len(lines) > 0 {
		fmt.Fprintf(w, "\nWhat fog changes\n%s\n", strings.Join(lines, "\n"))
	}
}

func mean(x []int) float64 {
	s := 0
	for _, v := range x {
		s += v
	}
	return float64(s) / float64(len(x))
}

func counts(m map[string]int, n int) string {
	type kv struct {
		k string
		v int
	}
	var all []kv
	for k, v := range m {
		all = append(all, kv{k, v})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].v > all[j].v || all[i].v == all[j].v && all[i].k < all[j].k })
	parts := make([]string, len(all))
	for i, e := range all {
		parts[i] = e.k + " " + pct(e.v, n)
	}
	return strings.Join(parts, ", ")
}
