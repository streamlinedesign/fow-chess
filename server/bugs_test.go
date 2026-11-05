package main

import (
	"bytes"
	"net/http"
	"strings"
	"testing"
)

func TestBugReports(t *testing.T) {
	site := newTestSite(t)
	carol := site.signup("carol")
	guest := &testUser{site, &http.Client{}, "guest"}

	for _, c := range []struct {
		who  *testUser
		body string
		want int
	}{
		{carol, `{"category":"rules","match":" ab12cd ","text":"My knight vanished.","page":"/m/AB12CD"}`, 200},
		{guest, `{"category":"ui","text":"Board too small\non my phone","player":"browser-9"}`, 200},
		{carol, `{"category":"nonsense","text":"x"}`, 400},
		{carol, `{"category":"ui","text":"   "}`, 400},
		{carol, `{"category":"ui","match":"no way!","text":"x"}`, 400},
		{carol, `{"category":"ui","text":"` + strings.Repeat("a", maxBugBody+1) + `"}`, 400},
	} {
		if code, body := c.who.do("POST", "/api/bugs", c.body); code != c.want {
			t.Errorf("%s: got %d %v, want %d", c.body[:min(len(c.body), 60)], code, body, c.want)
		}
	}

	reports, err := site.store.BugReports(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 2 {
		t.Fatalf("got %d reports, want 2", len(reports))
	}
	g, c := reports[0], reports[1] // newest first
	if c.Username != "carol" || c.MatchID != "AB12CD" || c.Category != "rules" || c.Page != "/m/AB12CD" {
		t.Errorf("carol's report: %+v", c)
	}
	if g.Username != "" || g.Player != "browser-9" || g.MatchID != "" {
		t.Errorf("guest's report: %+v", g)
	}

	var out bytes.Buffer
	if err := printBugReports(&out, site.store, 10); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"rules  by carol  match AB12CD", "> My knight vanished.", "> on my phone", "by guest"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("printout lacks %q:\n%s", want, out.String())
		}
	}

	// A browser gets a handful of reports, then has to wait.
	limited := 0
	for range maxFails + 2 {
		if code, _ := guest.do("POST", "/api/bugs", `{"category":"other","text":"again","player":"spammer"}`); code == http.StatusTooManyRequests {
			limited++
		}
	}
	if limited != 2 {
		t.Errorf("%d reports refused, want 2", limited)
	}
}
