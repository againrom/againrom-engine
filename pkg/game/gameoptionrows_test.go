package game

import (
	"strings"
	"testing"

	"againrom/pkg/ui"
)

// TestGameOptionRowsHoldEveryOption is the game side's one table: every
// ui.GameOption has exactly one row, and each row states each behaviour or
// its absence. The expected column is the whole table; a new option is one
// line here beside its two table rows.
func TestGameOptionRowsHoldEveryOption(t *testing.T) {
	if len(gameOptionRows) != len(ui.GameOptions()) {
		t.Fatalf("%d game rows for %d options", len(gameOptionRows), len(ui.GameOptions()))
	}
	want := map[string]string{
		"ShowTimeFlow":     "initial adopt read recorded apply caption",
		"ShowAllHitPoints": "initial adopt read recorded apply caption",
		"ShowFlyingHP":     "initial adopt read recorded apply caption",
		"FormationMode":    "initial adopt read recorded apply queue caption",
		"WimpyMode":        "initial session remember adopt recorded apply enter queue caption",
		"ShowPathfinding":  "initial session remember adopt read apply caption",
		"Smoothing":        "initial graphics caption",
		"Shadows":          "initial graphics caption",
		"Lighting":         "initial graphics caption",
		"Animation":        "initial graphics caption",
		"AutoCasting":      "initial adopt read check recorded apply open queue caption",
		"LootDots":         "initial graphics caption",
		"AutoCastMode":     "initial adopt read recorded apply queue caption",
	}
	for _, o := range ui.GameOptions() {
		r := gameOptionRows[o]
		var got []string
		for _, f := range []struct {
			name string
			set  bool
		}{
			{"initial", r.initial != nil}, {"graphics", r.graphics != nil}, {"session", r.session != nil},
			{"remember", r.remember != nil}, {"adopt", r.adopt != nil}, {"read", r.read != nil}, {"check", r.check != nil},
			{"recorded", r.recorded}, {"apply", r.apply != nil}, {"enter", r.enter != nil}, {"open", r.open != nil},
			{"queue", r.queue != nil},
			{"caption", r.captions.label != "" || len(r.captions.lines) != 0 || r.captions.perGame != nil || r.captions.patch != 0},
		} {
			if f.set {
				got = append(got, f.name)
			}
		}
		if g := strings.Join(got, " "); g != want[o.Key()] {
			t.Errorf("%s row holds %q, want %q", o.Key(), g, want[o.Key()])
		}
		// A graphics row is decided by its group, never by a session copy, a
		// live read or an apply of its own.
		if r.graphics != nil && (r.session != nil || r.remember != nil || r.read != nil || r.apply != nil || r.adopt != nil || r.queue != nil) {
			t.Errorf("%s is a graphics row with a behaviour of its own", o.Key())
		}
		// The session copy is read and written together.
		if (r.session == nil) != (r.remember == nil) {
			t.Errorf("%s reads or writes a session copy alone", o.Key())
		}
		// The choices a row names fit its value count.
		if len(r.captions.choices) > o.Choices() || len(r.captions.lines) > o.Choices()+1 {
			t.Errorf("%s names more captions than values", o.Key())
		}
	}
	// One queued parameter is one option.
	seen := map[int32]string{}
	for _, o := range ui.GameOptions() {
		if q := gameOptionRows[o].queue; q != nil {
			if other, ok := seen[int32(q.parameter)]; ok {
				t.Errorf("%s and %s queue the same parameter", other, o.Key())
			}
			seen[int32(q.parameter)] = o.Key()
		}
	}
}
