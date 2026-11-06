package main

import (
	"path/filepath"
	"testing"
)

func TestTitleFor(t *testing.T) {
	for peak, want := range map[int]string{
		999: "", 1000: "", 1149: "", 1150: "club", 1300: "expert", 1449: "expert",
		1450: "candidate", 1600: "fog", 1800: "im", 1999: "im", 2000: "gm", 2499: "gm", 2500: "king", 3100: "king",
	} {
		if got := titleFor(peak); got != want {
			t.Errorf("titleFor(%d) = %q, want %q", peak, got, want)
		}
	}
	// Every title comes with a set, and every rating mark is a title.
	if len(titles) != len(ratingUnlocks) {
		t.Fatalf("%d titles for %d rating marks", len(titles), len(ratingUnlocks))
	}
	for i, ti := range titles {
		if ti.at != ratingUnlocks[i] {
			t.Errorf("title %s at %d, rating mark %d", ti.id, ti.at, ratingUnlocks[i])
		}
		found := false
		for _, reason := range themeLocks {
			found = found || reason == lockRating(ti.at)
		}
		if !found {
			t.Errorf("no set comes with %s", ti.id)
		}
	}
}

// A title comes from the higher of the two ratings, and stays when the
// rating falls once the mark is recorded.
func TestTitles(t *testing.T) {
	store := openTestStore(t, filepath.Join(t.TempDir(), "x"))
	for _, name := range []string{"Ann", "ben", "cy", "dee"} {
		if _, err := store.CreateUser(name, []byte("x")); err != nil {
			t.Fatal(err)
		}
	}
	store.db.Exec(`UPDATE users SET rating = 1310, four_rating = 1000 WHERE username = 'Ann'`)
	store.db.Exec(`UPDATE users SET rating = 1000, four_rating = 2004 WHERE username = 'ben'`)
	store.db.Exec(`UPDATE users SET rating = 1100 WHERE username = 'cy'`)
	cy, _ := store.UserByName("cy")
	store.Unlock(cy.ID, lockRating(1600))
	got, err := store.Titles([]string{"ann", "BEN", "cy", "dee", "nobody"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"Ann": "expert", "ben": "gm", "cy": "fog"}
	if len(got) != len(want) {
		t.Errorf("titles: %v", got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: %q, want %q", k, got[k], v)
		}
	}
}
