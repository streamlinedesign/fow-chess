package main

import "testing"

func TestLongClock(t *testing.T) {
	for _, tc := range []struct {
		t    TimeControl
		long bool
	}{
		{TimeControl{Base: 180, Inc: 2}, false},
		{TimeControl{Base: 600}, false},
		{TimeControl{Base: 1800, Inc: 10}, false},
		{TimeControl{Base: 5400, Inc: 30}, true},
		{TimeControl{Base: 7200, Inc: 30}, true},
		{TimeControl{PerMove: 86400}, true},
	} {
		if longClock(tc.t) != tc.long {
			t.Errorf("%+v long %v", tc.t, !tc.long)
		}
	}
}

func TestWinValue(t *testing.T) {
	short, long := TimeControl{Base: 300}, TimeControl{PerMove: 86400}
	for _, tc := range []struct {
		t             TimeControl
		winner, loser float64
		want          int
	}{
		{short, 1000, 1000, 21},
		{short, 1000, 1400, 23},
		{short, 1000, 2000, 23},
		{short, 1400, 1000, 18},
		{short, 1000, 1200, 22},
		{long, 1000, 1000, 26},
		{long, 1000, 1400, 30},
		{long, 1600, 1000, 22},
	} {
		if got := winValue(tc.t, tc.winner, tc.loser); got != tc.want {
			t.Errorf("%+v %v beats %v: %d, want %d", tc.t, tc.winner, tc.loser, got, tc.want)
		}
	}
}

func TestDuelPoints(t *testing.T) {
	short := TimeControl{Base: 300}
	if w, l := duelPoints(short, 1000, 1000, false, false); w != 21 || l != -21 {
		t.Errorf("a loss %d %d", w, l)
	}
	if w, l := duelPoints(short, 1000, 1000, true, false); w != 21 || l != -11 {
		t.Errorf("a first loss of the day %d %d", w, l)
	}
	if w, l := duelPoints(short, 1000, 1000, false, true); w != 5 || l != -5 {
		t.Errorf("a Sniper Win %d %d", w, l)
	}
}

func TestWarPoints(t *testing.T) {
	short, long := TimeControl{Base: 1800, Inc: 10}, TimeControl{Base: 7200, Inc: 30}
	even := [4]float64{1000, 1000, 1000, 1000}
	all := [4]bool{true, true, true, true}
	none := [4]bool{}
	for _, tc := range []struct {
		name      string
		t         TimeControl
		places    []int
		ratings   [4]float64
		firstLoss [4]bool
		want      [4]int
	}{
		{"short", short, []int{1, 4, 3, 2}, even, none, [4]int{21, -11, -11, -21}},
		{"long", long, []int{1, 4, 3, 2}, even, none, [4]int{26, -13, -13, -26}},
		{"first losses", long, []int{1, 4, 3, 2}, even, all, [4]int{26, -7, 0, -13}},
		{"underdog", short, []int{2, 3, 4, 1}, [4]float64{1400, 1400, 1400, 1000}, none, [4]int{-23, -12, -12, 23}},
		{"draw", short, []int{1, 1, 3, 4}, even, none, [4]int{0, 0, -11, -11}},
		{"everyone level", short, []int{1, 1, 1, 1}, even, none, [4]int{}},
	} {
		if got := warPoints(tc.t, tc.places, tc.ratings, all, tc.firstLoss); got != tc.want {
			t.Errorf("%s: %v, want %v", tc.name, got, tc.want)
		}
	}
}
