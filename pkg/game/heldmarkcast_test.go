package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/rules"
	"againrom/pkg/sim"
)

// heldMarkRecords casts the spell whose cast picture is picture from a unit
// caster and answers how many cast records the World keeps after the release.
func heldMarkRecords(t *testing.T, picture int32) int {
	t.Helper()
	const at = 5
	w, err := sim.NewSpelledWorld(1, sim.Bounds{Width: 32, Height: 32}, sim.ModeCanonical, nil,
		[]sim.Entity{{ID: 1, X: at, Y: at, HP: 10, MaxHP: 10, Class: 7}}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	w.SetFlightInputs(buildFlightInputs(data.SpellPresentationOf(rules.SpellGameSecond), sim.FlightProfile{}, flightClassRule,
		[]flightPictureRow{{ID: picture, Present: true, Drawn: true, Phases: 4}}, nil))
	spell := uint16((picture - 8) / 2)
	w.ReleaseCasts([]sim.CastEvent{{Caster: 1, Target: 1, AtCell: true, Spell: spell, FromX: at, FromY: at, ToX: at, ToY: at}})
	return len(w.SavedProjectiles().Items)
}

func TestSecondUnitCasterOfAHeldMarkSpellEndsItsCastRecordAtOnce(t *testing.T) {
	for _, id := range []int32{16, 24, 34, 46, 48, 62} {
		if n := heldMarkRecords(t, id); n != 0 {
			t.Errorf("identifier %d: %d cast records survive the release, want 0", id, n)
		}
	}
	for _, id := range []int32{20, 60, 64} {
		if n := heldMarkRecords(t, id); n == 0 {
			t.Errorf("identifier %d outside the six lost its cast record", id)
		}
	}
}
