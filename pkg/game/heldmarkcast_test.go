package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/rules"
	"againrom/pkg/sim"
)

// heldMarkRecords casts the spell whose cast picture is picture from a unit
// caster with game's presentation set and answers how many cast records the
// World keeps after the release.
func heldMarkRecords(t *testing.T, game rules.SpellGame, picture int32) int {
	t.Helper()
	const at = 5
	w, err := sim.NewSpelledWorld(1, sim.Bounds{Width: 32, Height: 32}, sim.ModeCanonical, nil,
		[]sim.Entity{{ID: 1, X: at, Y: at, HP: 10, MaxHP: 10, Class: 7}}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	w.SetFlightInputs(buildFlightInputs(data.SpellPresentationOf(game), sim.FlightProfile{}, flightClassRule,
		[]flightPictureRow{{ID: picture, Present: true, Drawn: true, Phases: 4}}, nil))
	spell := uint16((picture - 8) / 2)
	w.ReleaseCasts([]sim.CastEvent{{Caster: 1, Target: 1, AtCell: true, Spell: spell, FromX: at, FromY: at, ToX: at, ToY: at}})
	return len(w.SavedProjectiles().Items)
}

// A unit caster's release of every identifier outside 10, 12, 18, 28, 30,
// 40, 54, 56 and 60 has life 0 (R2-ENGINE-691); Poison Cloud's 20 and Curse's
// 64 leave only the held mark (R2-ENGINE-1520). Drain Life's 60 lives one
// update, and the first game's 20 keeps its one-update record.
func TestSecondUnitCasterOfAHeldMarkSpellEndsItsCastRecordAtOnce(t *testing.T) {
	for _, id := range []int32{16, 20, 24, 34, 46, 48, 62, 64} {
		if n := heldMarkRecords(t, rules.SpellGameSecond, id); n != 0 {
			t.Errorf("identifier %d: %d cast records survive the release, want 0", id, n)
		}
	}
	if n := heldMarkRecords(t, rules.SpellGameSecond, 60); n == 0 {
		t.Error("identifier 60 lost its one-update cast record")
	}
	if n := heldMarkRecords(t, rules.SpellGameFirst, 20); n == 0 {
		t.Error("the first game's identifier 20 lost its cast record")
	}
}
