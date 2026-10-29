// Command simulate plays Eat the King between bots, as many games as you
// like, and writes one line of JSON per game. It needs no server or database.
//
//	go run ./cmd/simulate -games 10000
//	go run ./cmd/simulate -games 0 -out overnight.jsonl   # until Ctrl-C
//	go run ./cmd/simulate -summary overnight.jsonl
//
// Every combination of -bots (as White and as Black), -modes and -fog is
// played in turn, so a run is a balanced mix. Stopping and starting again
// with the same -out adds to the file, and the summary covers all of it.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"chess/engine"
	"chess/sim"
)

func main() {
	games := flag.Int("games", 1000, "games to play; 0 plays until Ctrl-C")
	out := flag.String("out", "games.jsonl", "file to add games to")
	bots := flag.String("bots", "hunter", "bots to pair up, comma-separated: "+botList())
	modes := flag.String("modes", "standard,backRow,battleship", "start modes, comma-separated")
	guarded := flag.Bool("guarded", false, "Battleship setups keep the king walled in, as a careful player would, instead of anywhere")
	fog := flag.String("fog", "both", "on, off or both")
	mods := flag.String("mods", "", "extra rules, comma-separated: checkmate, jumpKing, backPawns, hill")
	workers := flag.Int("workers", runtime.NumCPU(), "games played at once")
	seed := flag.Uint64("seed", uint64(time.Now().UnixNano()), "first game's seed; game i uses seed+i")
	moves := flag.Bool("moves", true, "keep every game's moves so it can be replayed")
	summary := flag.String("summary", "", "summarise this file instead of playing")
	flag.Parse()

	if *summary != "" {
		stats, n, err := readStats(*summary)
		if err != nil {
			fail(err)
		}
		fmt.Printf("%d games in %s\n", n, *summary)
		stats.Print(os.Stdout)
		return
	}

	rules := engine.Rules{KingCapture: true}
	for _, m := range split(*mods) {
		switch m {
		case "checkmate":
			rules.KingCapture = false
		case "jumpKing":
			rules.JumpKing = true
		case "backPawns":
			rules.BackPawns = true
		case "hill":
			rules.Hill = true
		default:
			fail(fmt.Errorf("unknown mod %q", m))
		}
	}
	var fogs []bool
	switch *fog {
	case "on":
		fogs = []bool{true}
	case "off":
		fogs = []bool{false}
	case "both":
		fogs = []bool{true, false}
	default:
		fail(fmt.Errorf("-fog is on, off or both"))
	}
	var configs []sim.Config
	for _, w := range split(*bots) {
		for _, b := range split(*bots) {
			for _, m := range split(*modes) {
				for _, f := range fogs {
					if sim.Bots[w] == nil || sim.Bots[b] == nil {
						fail(fmt.Errorf("unknown bot; the bots are %s", botList()))
					}
					configs = append(configs, sim.Config{White: w, Black: b, Mode: m, Guarded: *guarded, Fog: f, Rules: rules, Moves: *moves})
				}
			}
		}
	}

	file, err := os.OpenFile(*out, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		fail(err)
	}
	defer file.Close()
	w := bufio.NewWriter(file)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	var stopping atomic.Bool
	go func() {
		<-stop
		fmt.Fprintln(os.Stderr, "\nStopping after the games being played...")
		stopping.Store(true)
	}()

	var next atomic.Int64
	results := make(chan sim.Record, 256)
	var wg sync.WaitGroup
	for range *workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for !stopping.Load() {
				i := next.Add(1) - 1
				if *games > 0 && i >= int64(*games) {
					return
				}
				c := configs[i%int64(len(configs))]
				c.Seed = *seed + uint64(i)
				rec, err := sim.Play(c)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					continue
				}
				results <- rec
			}
		}()
	}
	go func() { wg.Wait(); close(results) }()

	stats := sim.NewStats()
	start, lastReport := time.Now(), time.Now()
	n := 0
	enc := json.NewEncoder(w)
	for rec := range results {
		if err := enc.Encode(&rec); err != nil {
			fail(err)
		}
		stats.Add(&rec)
		n++
		if time.Since(lastReport) > 5*time.Second {
			w.Flush()
			lastReport = time.Now()
			rate := float64(n) / time.Since(start).Seconds()
			fmt.Fprintf(os.Stderr, "%d games, %.0f a second\n", n, rate)
		}
	}
	if err := w.Flush(); err != nil {
		fail(err)
	}
	fmt.Printf("Played %d games in %s (%.0f a second), added to %s\n", n, time.Since(start).Round(time.Second),
		float64(n)/time.Since(start).Seconds(), *out)
	stats.Print(os.Stdout)
}

func readStats(path string) (*sim.Stats, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()
	stats := sim.NewStats()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	n := 0
	for sc.Scan() {
		var r sim.Record
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			continue // a line cut short when a run was killed
		}
		stats.Add(&r)
		n++
	}
	return stats, n, sc.Err()
}

func split(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func botList() string {
	var names []string
	for name := range sim.Bots {
		names = append(names, name)
	}
	slices.Sort(names)
	return strings.Join(names, ", ")
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
