package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"strings"
	"testing"

	"chess/engine"
)

var update = flag.Bool("update", false, "rewrite the landing page showcase")

// operaGame is Morphy's Opera Game (Paris, 1858), short and full of captures.
const operaGame = "e2e4 e7e5 g1f3 d7d6 d2d4 c8g4 d4e5 g4f3 d1f3 d6e5 f1c4 g8f6 f3b3 d8e7 b1c3 c7c6 c1g5 b7b5 " +
	"c3b5 c6b5 c4b5 b8d7 e1c1 a8d8 d1d7 d8d7 h1d1 e7e6 b5d7 f6d7 b3b8 d7b8 d1d8"

const showcasePath = "../web/src/showcase.json"

type showcaseSide struct {
	Board string    `json:"board"`
	Last  *lastMove `json:"last"`
	Check string    `json:"check,omitempty"`
}

type showcaseFrame struct {
	Turn     string            `json:"turn"`
	W        showcaseSide      `json:"w"`
	B        showcaseSide      `json:"b"`
	Captured map[string]string `json:"captured"`
}

// TestShowcase keeps the landing page's replay in step with the fog rules.
// After changing what players see, run: go test -run Showcase -update
func TestShowcase(t *testing.T) {
	g := engine.NewGame()
	frame := func() showcaseFrame {
		w, b := viewFor(g, white, false, true), viewFor(g, black, false, true)
		return showcaseFrame{
			Turn:     w.Turn,
			W:        showcaseSide{w.Board, w.Last, w.Check},
			B:        showcaseSide{b.Board, b.Last, b.Check},
			Captured: w.Captured,
		}
	}
	frames := []showcaseFrame{frame()}
	for _, uci := range strings.Fields(operaGame) {
		if _, err := g.PlayUCI(uci); err != nil {
			t.Fatal(uci, err)
		}
		frames = append(frames, frame())
	}
	if g.Method() != engine.Checkmate {
		t.Fatalf("showcase game should end in checkmate, got %v", g.Method())
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(frames); err != nil {
		t.Fatal(err)
	}
	if *update {
		if err := os.WriteFile(showcasePath, buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	have, err := os.ReadFile(showcasePath)
	if err != nil || !bytes.Equal(have, buf.Bytes()) {
		t.Errorf("%s is out of date; run: go test -run Showcase -update", showcasePath)
	}
}
