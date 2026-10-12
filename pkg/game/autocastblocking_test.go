package game

import (
	"testing"

	"againrom/pkg/sim"
)

// A changed structure's saved mask (2, mission start 1) survives resume under
// the automatic-casting wrapper.
func TestSnapshotResumeKeepsSavedBlockingUnderAutoCast(t *testing.T) {
	makeWorld := func(mask uint32) *sim.World {
		w, err := sim.NewStructuredWorld(7, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, sim.Terrain{},
			[]sim.Entity{{ID: 1, X: 3, Y: 3, HP: 100, MaxHP: 100}}, nil, sim.Relations{}, nil, nil, nil, sim.GhostTemplate{},
			[]sim.Structure{{ID: 3, Col: 6, Row: 6, Width: 1, Height: 1, Attach: 1, Blocking: mask}})
		if err != nil {
			t.Fatal(err)
		}
		return w
	}
	saved := makeWorld(2)
	if !saved.SetAutoCast(0, saved.AutoCast(0).WithLevel(sim.AutoCastLevel(2))) {
		t.Fatal("automatic-casting level refused")
	}
	raw, err := saved.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if raw[0] != 121 || string(raw[len(raw)-4:]) != "ACS1" {
		t.Fatalf("form %d does not carry the automatic-casting wrapper", raw[0])
	}
	if !sim.HasStructureBlockingForm(raw) {
		t.Error("automatic-casting wrapper hid the saved structure blocking")
	}
	ms := &Mission{World: makeWorld(1)}
	if err := resumeWorld(ms, &Snapshot{World: raw}, nil); err != nil {
		t.Fatal(err)
	}
	if got := ms.World.Structures()[0].Blocking; got != 2 {
		t.Fatalf("resumed blocking %d, want the saved 2 (mission start is 1)", got)
	}
	if ms.World.AutoCast(0).LevelNumber() != 2 {
		t.Fatal("resumed automatic-casting level lost")
	}
}
