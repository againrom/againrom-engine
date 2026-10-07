package sim

import "testing"

// bearingReplay transcribes AI-361's replayed direction-helper outputs: rows
// are the target's dy from -4 to 4, columns its dx from -4 to 4, each entry a
// heading byte. The centre entry is the coincident-centre result, which the
// delta-based helper cannot see and which is skipped.
var bearingReplay = [9][9]uint8{
	{224, 224, 224, 0, 0, 0, 32, 32, 32},
	{224, 224, 224, 0, 0, 0, 32, 32, 32},
	{224, 224, 224, 224, 0, 32, 32, 32, 32},
	{192, 192, 224, 224, 0, 32, 32, 64, 64},
	{192, 192, 192, 192, 224, 64, 64, 64, 64},
	{192, 192, 160, 160, 128, 96, 96, 64, 64},
	{160, 160, 160, 160, 128, 96, 96, 96, 96},
	{160, 160, 160, 128, 128, 128, 96, 96, 96},
	{160, 160, 160, 128, 128, 128, 96, 96, 96},
}

func TestBearingFacingMatchesTheReplayedDirectionHelper(t *testing.T) {
	for row := 0; row < 9; row++ {
		for col := 0; col < 9; col++ {
			dx, dy := int32(col-4), int32(row-4)
			if dx == 0 && dy == 0 {
				continue
			}
			got, ok := bearingFacing(dx, dy)
			if !ok || got != bearingReplay[row][col] {
				t.Errorf("bearingFacing(%d,%d) = %d,%v, want %d", dx, dy, got, ok, bearingReplay[row][col])
			}
		}
	}
	if _, ok := bearingFacing(0, 0); ok {
		t.Error("a zero delta has no heading")
	}
}

// A ranged attacker five cells off, one column across, faces the victim's
// nearest heading (north) and not the diagonal the delta's signs name.
func TestARangedAttackerTurnsToTheVictimsBearing(t *testing.T) {
	a := cbFighter(1, 5, 10, 8, 4)
	a.Reach = 6
	a.RotationSpeed = 32
	v := cbEnt(2, 6, 5)
	v.HP, v.MaxHP = 1_000_000, 1_000_000
	w := cbWorld(t, 1, a, v)
	Step(w, []Command{cbOrder(1, 2)})
	for n := 0; n < 10; n++ {
		Step(w, nil)
	}
	got := cbAt(t, w, 1)
	if got.X != 5 || got.Y != 10 {
		t.Fatalf("attacker moved to (%d,%d); the victim is inside its reach", got.X, got.Y)
	}
	if got.Facing != 0 {
		t.Errorf("attacker faces %d, want 0 (north) toward a victim at bearing (1,-5)", got.Facing)
	}
}
