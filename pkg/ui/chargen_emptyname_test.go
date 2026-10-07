package ui

import (
	"reflect"
	"testing"
	"time"
)

// emptyPreCreateName removes the name field's text with key 8, one byte per
// key, the way a player does.
func emptyPreCreateName(t *testing.T, a *App, c *Chargen, now *time.Time) {
	t.Helper()
	for i := 0; c.NameText() != ""; i++ {
		if i == 16 {
			t.Fatal("Backspace did not empty the name")
		}
		soundKey(a, now, appInput{Backspace: true})
	}
}

// The pre-create page's routes to the detailed page and what each does with
// the name field. TEXT-CHARGEN-028: the OK control emits its continue message
// only for a non-empty name. VIDEO-SFX-058: with an empty name, OK and Enter
// request nothing and send nothing. The page therefore stays open, keeps its
// focus, choices and playing samples, and a name typed afterwards lets the
// same route continue. The hero double-click is the owner-authored second
// route to the same transition (DIV-1507), so it follows OK.
func TestPreCreateContinuesOnlyWithAName(t *testing.T) {
	const hero = chargenChoice3
	routes := []struct {
		name  string
		leave func(t *testing.T, a *App, c *Chargen, now *time.Time)
		// prior is what the route itself requests before its transition: the
		// double-click's first click is a hero press.
		prior []string
	}{
		{"OK press", func(t *testing.T, a *App, c *Chargen, now *time.Time) {
			soundClick(a, now, preControlRect(c, chargenForward).Min)
		}, nil},
		{"Enter on OK", func(t *testing.T, a *App, c *Chargen, now *time.Time) {
			soundFocus(t, a, c, now, 6)
			soundKey(a, now, appInput{Enter: true})
		}, nil},
		{"hero double-click", func(t *testing.T, a *App, c *Chargen, now *time.Time) {
			p := preControlRect(c, hero).Min
			soundPress(a, now, time.Second, p)
			soundRelease(a, now, p)
			soundPress(a, now, 100*time.Millisecond, p)
		}, []string{"chrgen/char.wav"}},
	}
	for _, route := range routes {
		t.Run(route.name+" with an empty name keeps the page", func(t *testing.T) {
			now := time.Unix(1_700_000_000, 0)
			rec := &memberRecorder{}
			a, c := openSoundChargen(t, rec)
			c.SelectDifficulty(2)
			emptyPreCreateName(t, a, c, &now)
			route.leave(t, a, c, &now)
			if c.Stage() != PreCreateStage || a.Screen() != ScreenChargen {
				t.Fatalf("with an empty name the route left stage %v on screen %v", c.Stage(), a.Screen())
			}
			if got := rec.members(); !reflect.DeepEqual(got, append([]string{}, route.prior...)) {
				t.Fatalf("requested %q, want %q: an empty name requests nothing", got, route.prior)
			}
			for _, v := range rec.voices {
				if !v.playing || v.stops != 0 {
					t.Fatalf("%s was stopped, so the page closed: %+v", v.member, *v)
				}
			}
			if c.NameText() != "" || c.Difficulty() != 3 {
				t.Fatalf("the page changed: name %q, difficulty %d", c.NameText(), c.Difficulty())
			}
		})
		t.Run(route.name+" with a name continues", func(t *testing.T) {
			now := time.Unix(1_700_000_000, 0)
			rec := &memberRecorder{}
			a, c := openSoundChargen(t, rec)
			emptyPreCreateName(t, a, c, &now)
			route.leave(t, a, c, &now)
			if c.Stage() != PreCreateStage {
				t.Fatalf("an empty name did not keep the page: stage %v", c.Stage())
			}
			soundFocus(t, a, c, &now, 0)
			soundKey(a, &now, appInput{Typed: "x"})
			if c.NameText() != "x" {
				t.Fatalf("typed name %q, want x", c.NameText())
			}
			route.leave(t, a, c, &now)
			if c.Stage() != DetailedStage {
				t.Fatalf("with the name %q the route left stage %v, want the detailed page", c.NameText(), c.Stage())
			}
		})
	}
}

// A one-byte name is a name: the claims test the length of the field, not
// what the byte is.
func TestPreCreateContinuesWithASpaceName(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	rec := &memberRecorder{}
	a, c := openSoundChargen(t, rec)
	emptyPreCreateName(t, a, c, &now)
	soundKey(a, &now, appInput{Typed: " "})
	if c.NameText() != " " {
		t.Fatalf("typed name %q, want one space", c.NameText())
	}
	soundClick(a, &now, preControlRect(c, chargenForward).Min)
	if c.Stage() != DetailedStage {
		t.Fatalf("a one-byte name left stage %v, want the detailed page", c.Stage())
	}
}
