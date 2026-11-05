package main

import "testing"

func TestPrefsAndAvatars(t *testing.T) {
	site := newTestSite(t)
	ann := site.signup("ann")
	site.signup("ben")

	// Everyone starts with the classic set and the pawn.
	_, me := ann.do("GET", "/api/me", "")
	if me["primary"] != "classic" || me["secondary"] != "classic" || me["side"] != "" || me["avatar"] != "" {
		t.Fatalf("defaults: %v", me)
	}

	for _, bad := range []string{
		`{"primary":"gold"}`,
		`{"side":"red"}`,
		`{"avatar":"classic/K"}`,
		`{"avatar":"red/Z"}`,
		`{"avatar":"red"}`,
		`{"avatar":"galaxy/K"}`,
		`{"lightTiles":"marble"}`,
	} {
		if code, _ := ann.do("PUT", "/api/prefs", bad); code != 400 {
			t.Errorf("%s: %d", bad, code)
		}
	}

	if code, _ := ann.do("PUT", "/api/prefs", `{"primary":"red-ink","secondary":"blue","side":"b","avatar":"violet-ink/N"}`); code != 200 {
		t.Fatalf("save: %d", code)
	}
	_, me = ann.do("GET", "/api/me", "")
	if me["primary"] != "red-ink" || me["secondary"] != "blue" || me["side"] != "b" || me["avatar"] != "violet-ink/N" {
		t.Errorf("saved: %v", me)
	}

	// Themes: a two-sided set, tiles per square colour, and no skins at all.
	if code, _ := ann.do("PUT", "/api/prefs", `{"primary":"holy","secondary":"galaxy","lightTiles":"holy","darkTiles":"galaxy","plain":true,"avatar":"ancient-dark/N"}`); code != 200 {
		t.Fatalf("themes: %d", code)
	}
	_, me = ann.do("GET", "/api/me", "")
	if me["primary"] != "holy" || me["lightTiles"] != "holy" || me["darkTiles"] != "galaxy" || me["plain"] != true || me["avatar"] != "ancient-dark/N" {
		t.Errorf("themes saved: %v", me)
	}
	ann.do("PUT", "/api/prefs", `{"primary":"red-ink","secondary":"blue","side":"b","avatar":"violet-ink/N"}`)

	// Anyone can see avatars; players on the pawn are left out.
	_, got := ann.do("GET", "/api/avatars?u=ANN,ben,nobody", "")
	if len(got) != 1 || got["ann"] != "violet-ink/N" {
		t.Errorf("avatars: %v", got)
	}
}
