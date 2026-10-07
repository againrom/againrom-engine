package game

import (
	"testing"

	"image"
)

func TestPushCarriesTheClassSoundSlots(t *testing.T) {
	mw := swingWorld(t, swingAnimDesc(), swingSoundUnit(1, 4, 4), swingSoundUnit(2, 5, 4))
	mw.setSwingSound(map[int32]UnitSound{
		// Class 1 is the class both fixture units are placed under. Class 2 is
		// a row NO entity holds, and it carries a distinct array so a lookup
		// that keyed on anything but the entity's own class id would be
		// visible here rather than merely unlucky.
		1: {Slots: []int32{11, 0, 22, 33, 44}, AttackDelay: 5},
		2: {Slots: []int32{99, 99, 99, 99, 99}, AttackDelay: 5},
	}, func(int, uint32, image.Point) {})

	got := swingDraw(t, mw, 1).Sound
	want := []int{11, 0, 22, 33, 44}
	if len(got) != len(want) {
		t.Fatalf("Sound = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Sound = %v, want %v", got, want)
		}
	}
}

// TestPushCarriesNoSlotsForAnUnnamedClass is the same statement's other half:
// a class the table does not name, and a world wired with no table at all,
// both reach the seam as nil — the silence every entity built before this
// story already carried.
func TestPushCarriesNoSlotsForAnUnnamedClass(t *testing.T) {
	t.Run("a table naming another class", func(t *testing.T) {
		mw := swingWorld(t, swingAnimDesc(), swingSoundUnit(1, 4, 4), swingSoundUnit(2, 5, 4))
		mw.setSwingSound(map[int32]UnitSound{7: {Slots: []int32{11}}}, func(int, uint32, image.Point) {})
		if got := swingDraw(t, mw, 1).Sound; got != nil {
			t.Fatalf("Sound = %v, want nil", got)
		}
	})
	t.Run("no table at all", func(t *testing.T) {
		mw := swingWorld(t, swingAnimDesc(), swingSoundUnit(1, 4, 4), swingSoundUnit(2, 5, 4))
		if got := swingDraw(t, mw, 1).Sound; got != nil {
			t.Fatalf("Sound = %v, want nil", got)
		}
	})
}
