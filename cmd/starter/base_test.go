package main

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	basepkg "againrom/pkg/base"
	"againrom/pkg/game"
)

func profileInfo(id string, exact bool) game.InstallInfo {
	p, ok := basepkg.Find(id)
	if !ok {
		panic("unknown profile " + id)
	}
	return game.InstallInfo{Language: p.Language, Base: basepkg.Match{Profile: p, Exact: exact}}
}

// Play passes the detected profile to the game, so the game refuses a root that
// is no longer that basepkg.
func TestPlayNamesTheDetectedBase(t *testing.T) {
	cases := []struct {
		name string
		info game.InstallInfo
		want []string
	}{
		{"exact release", profileInfo(basepkg.ROM1EN, true), []string{"-assets", "ROOT", "-base", "rom1-en"}},
		{"release by language", profileInfo(basepkg.ROM1RU, false), []string{"-assets", "ROOT", "-base", "rom1-ru"}},
		{"second game, exact", profileInfo(basepkg.ROM2RU, true), []string{"-assets", "ROOT", "-base", "rom2-ru"}},
		{"second game, by language", profileInfo(basepkg.ROM2EN, false), []string{"-assets", "ROOT", "-base", "rom2-en"}},
		{"second game, unknown language", game.InstallInfo{Base: basepkg.Match{Profile: unknownSecond()}}, []string{"-assets", "ROOT", "-base", "rom2"}},
		{"demo", profileInfo(basepkg.ROM1Demo, true), []string{"-assets", "ROOT", "-base", "rom1-demo"}},
		{"no profile named", game.InstallInfo{Language: "english"}, []string{"-assets", "ROOT"}},
		{"generic profile", game.InstallInfo{Base: basepkg.Match{Profile: basepkg.Profile{ID: basepkg.ROM1}}}, []string{"-assets", "ROOT"}},
	}
	for _, c := range cases {
		r := newRig(t, "[starter]\nlast-base = en\n[bases]\nen = {root}\n")
		r.fakes.validRoots[r.root] = c.info
		a := r.app()
		do(t, a, action{kind: aPlay})
		want := []string{filepath.Join(r.dir, "againrom.exe")}
		for _, w := range c.want {
			if w == "ROOT" {
				w = r.root
			}
			want = append(want, w)
		}
		if len(r.fakes.started) != 1 || !reflect.DeepEqual(r.fakes.started[0], want) {
			t.Errorf("%s: started %q\nwant %q", c.name, r.fakes.started, want)
		}
		if line, err := a.command(); err != nil || !strings.Contains(line, strings.Join(want[1:], " ")) {
			t.Errorf("%s: command line %q (%v)", c.name, line, err)
		}
	}
}

// The base rows and the status line name the profile and state its limits.
func TestBaseRowsAndNotesStateTheProfile(t *testing.T) {
	r := newRig(t, "[starter]\nlast-base = en\n[bases]\nen = {root}\ndemo = {root}2\n")
	demo := r.root + "2"
	r.fakes.validRoots[r.root] = profileInfo(basepkg.ROM1EN, true)
	r.fakes.validRoots[demo] = profileInfo(basepkg.ROM1Demo, true)
	a := r.app()
	var rows []string
	for _, it := range a.layout() {
		if it.kind == iBaseRow {
			rows = append(rows, it.text)
		}
	}
	if len(rows) != 2 || !strings.HasSuffix(rows[0], "OK rom1-en") || !strings.HasSuffix(rows[1], "OK rom1-demo, has limits") {
		t.Fatalf("rows = %q", rows)
	}
	do(t, a, action{aSelectBase, 1})
	for _, want := range []string{"rom1-demo (Rage of Mages demo 1.01; exact build)", "no character generation, new game opens mission 41", "original saves refused"} {
		if !strings.Contains(a.status, want) || a.statusBad {
			t.Fatalf("status %q lacks %q", a.status, want)
		}
	}
	do(t, a, action{aSelectBase, 0})
	if strings.Contains(a.status, "no character generation") || !strings.Contains(a.status, "rom1-en") {
		t.Fatalf("status %q", a.status)
	}
}

func TestBaseLabelOfAnUnrecognisedBuild(t *testing.T) {
	if got := baseLabel(profileInfo(basepkg.ROM1EN, false)); got != "rom1-en (unrecognised build)" {
		t.Fatalf("label = %q", got)
	}
	if got := baseLabel(game.InstallInfo{Language: "russian"}); got != "russian" {
		t.Fatalf("label = %q", got)
	}
}

func unknownSecond() basepkg.Profile {
	p, ok := basepkg.Find(basepkg.ROM2)
	if !ok {
		panic("no generic second-game profile")
	}
	return p
}

// A second-game root is a valid base the starter selects and launches; its row
// and status line state the limits.
func TestSecondGameRootIsSelectableAndLaunches(t *testing.T) {
	r := newRig(t, "[starter]\nlast-base = en\n[bases]\nen = {root}\nsecond = {root}2\n")
	second := r.root + "2"
	r.fakes.validRoots[r.root] = profileInfo(basepkg.ROM1EN, true)
	r.fakes.validRoots[second] = profileInfo(basepkg.ROM2RU, true)
	a := r.app()
	var rows []string
	for _, it := range a.layout() {
		if it.kind == iBaseRow {
			rows = append(rows, it.text)
		}
	}
	if len(rows) != 2 || !strings.HasSuffix(rows[1], "OK rom2-ru, has limits") {
		t.Fatalf("rows = %q", rows)
	}
	do(t, a, action{aSelectBase, 1})
	for _, want := range []string{"rom2-ru (Rage of Mages II, Russian; exact build)", "new game opens mission 10", "original saves refused"} {
		if !strings.Contains(a.status, want) || a.statusBad {
			t.Fatalf("status %q lacks %q", a.status, want)
		}
	}
	do(t, a, action{kind: aPlay})
	want := []string{filepath.Join(r.dir, "againrom.exe"), "-assets", second, "-base", "rom2-ru"}
	if len(r.fakes.started) != 1 || !reflect.DeepEqual(r.fakes.started[0], want) {
		t.Fatalf("started %q, want %q", r.fakes.started, want)
	}
}
