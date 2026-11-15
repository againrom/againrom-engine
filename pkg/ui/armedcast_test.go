package ui

import (
	"fmt"
	"testing"
)

// armedPair is the armed-cast state a test reads: a book spell armed
// (spellArmed) and a scroll armed (itemCast), each as one bit.
func armedPair(v *Viewer) string {
	return fmt.Sprintf("spell=%t scroll=%t", v.spellArmed, v.itemCast != nil)
}

type armedSetup struct {
	name string
	make func(t *testing.T) *Viewer
}

// armedSetups arms a book spell (chosen in the shown book) and a book scroll
// (pressed in the shown book), each with the book open.
var armedSetups = []armedSetup{
	{"spell", func(t *testing.T) *Viewer {
		a, v := castCursorApp(t)
		pressPanelKey(t, a, "space")
		chooseSpell(t, v, castTarget)
		return v
	}},
	{"scroll", func(t *testing.T) *Viewer {
		v, _ := bookScrollFighter(t)
		x, y := bookCellCentre(t, v, 4)
		sbClick(v, x, y)
		return v
	}},
}

type armedEvent struct {
	name string
	do   func(t *testing.T, v *Viewer)
}

// armedEvents are the events the armed-cast row names, each applied to a fresh
// armed setup.
var armedEvents = []armedEvent{
	{"cast on the caster cell", func(t *testing.T, v *Viewer) {
		fx, fy := cellPoint(v, 3, 3)
		tapAt(v, fx, fy)
	}},
	{"book panel closes", func(t *testing.T, v *Viewer) {
		if v.hudShown(hudPanelBook) {
			v.toggleHudPanel(hudPanelBook)
		}
	}},
	{"scroll leaves the list", func(t *testing.T, v *Viewer) {
		entries := append([]SpellEntry(nil), v.spellbook...)
		for i := range entries {
			entries[i].ItemCast = nil
		}
		v.SetSpellbook(panelCasterID, entries)
	}},
	{"defend", func(t *testing.T, v *Viewer) { v.armDefend() }},
	{"retreat", func(t *testing.T, v *Viewer) { v.issuePlayerRetreat() }},
	{"map cancel", func(t *testing.T, v *Viewer) { v.cancelMapCommand() }},
	{"attack key", func(t *testing.T, v *Viewer) { v.armAttack() }},
	{"move order key", func(t *testing.T, v *Viewer) { v.armCommand(commandMove) }},
	{"spellbook owner changes", func(t *testing.T, v *Viewer) { v.SetSpellbook(panelCasterID+1, v.spellbook) }},
	{"spellbook cleared", func(t *testing.T, v *Viewer) { v.ClearSpellbook() }},
}

// armedCastWant is the armed-cast state on base after each event, per kind.
var armedCastWant = map[string]map[string]string{
	"spell": {
		"cast on the caster cell": "spell=true scroll=false",
		"book panel closes":       "spell=true scroll=false",
		"scroll leaves the list":  "spell=true scroll=false",
		"defend":                  "spell=false scroll=false",
		"retreat":                 "spell=false scroll=false",
		"map cancel":              "spell=true scroll=false",
		"attack key":              "spell=false scroll=false",
		"move order key":          "spell=false scroll=false",
		"spellbook owner changes": "spell=false scroll=false",
		"spellbook cleared":       "spell=false scroll=false",
	},
	"scroll": {
		"cast on the caster cell": "spell=false scroll=true",
		"book panel closes":       "spell=false scroll=true",
		"scroll leaves the list":  "spell=false scroll=false",
		"defend":                  "spell=false scroll=false",
		"retreat":                 "spell=false scroll=false",
		"map cancel":              "spell=false scroll=false",
		"attack key":              "spell=false scroll=false",
		"move order key":          "spell=false scroll=false",
		"spellbook owner changes": "spell=false scroll=false",
		"spellbook cleared":       "spell=false scroll=false",
	},
}

// TestArmedCastEventsPinEachKind pins, for a book spell and for a book scroll,
// armed or not after each event the armed-cast row names. It runs on base and
// after any change to the armed-cast state that keeps each kind's answer.
func TestArmedCastEventsPinEachKind(t *testing.T) {
	for _, s := range armedSetups {
		for _, e := range armedEvents {
			v := s.make(t)
			before := armedPair(v)
			e.do(t, v)
			got := armedPair(v)
			if w := armedCastWant[s.name][e.name]; w != got {
				t.Errorf("kind %s after %q: before [%s] got [%s] want [%s]", s.name, e.name, before, got, w)
			}
		}
	}
}

// TestArmedCastBothKindsCoexistOnBase pins a state base reaches: a scroll
// armed from the book, the book closed, then a spell armed by the quick-slot
// route (quickspells.go, armSpell) leaves both armed at once.
func TestArmedCastBothKindsCoexistOnBase(t *testing.T) {
	v := armedSetups[1].make(t)
	v.toggleHudPanel(hudPanelBook)
	v.armSpell()
	if got, want := armedPair(v), "spell=true scroll=true"; got != want {
		t.Fatalf("both armed: got [%s] want [%s]", got, want)
	}
}
