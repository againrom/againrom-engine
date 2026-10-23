package sim

import "testing"

// The end-of-mission cull stores nothing at the action-run end (SAV-1353), so
// a survivor leaves the mission with the run end the mission set.
func TestMissionReturnKeepsTheActionRunEnd(t *testing.T) {
	w := sourceMutationWorld(t, PlainItem(0xe01))
	e := &w.entities[0]
	e.Owner, e.TypeID = SelfSlot, HumanTypeID
	e.ActionClock = ActionClock{Known: true, End: 577}
	if err := w.NormalizeMissionSurvivors(SelfSlot); err != nil {
		t.Fatal(err)
	}
	if got := w.entities[0].ActionClock; got != (ActionClock{Known: true, End: 577}) {
		t.Fatalf("survivor action clock = %+v, want known end 577", got)
	}
}

// The second game's cull still clears the run end: no second-game claim states
// that it survives the cull.
func TestSecondGameMissionReturnClearsTheActionRunEnd(t *testing.T) {
	w := sourceMutationWorld(t, PlainItem(0xe01))
	script, err := NewROM2Script(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	w.script = script
	e := &w.entities[0]
	e.Owner, e.TypeID = SelfSlot, HumanTypeID
	e.ActionClock = ActionClock{Known: true, End: 577}
	if err := w.NormalizeMissionSurvivors(SelfSlot); err != nil {
		t.Fatal(err)
	}
	if got := w.entities[0].ActionClock; got != (ActionClock{}) {
		t.Fatalf("second-game survivor action clock = %+v, want unknown", got)
	}
}

// A kept stamp S keeps the regeneration rate at 1 until the restarted clock
// passes S by more than 80 sub-ticks (SAV-1355).
func TestKeptStampHoldsRateOneUntilEightyPast(t *testing.T) {
	e := Entity{ActionClock: ActionClock{Known: true, End: 577}}
	for _, c := range []struct {
		now  uint64
		want int32
	}{{0, 1}, {80, 1}, {577, 1}, {657, 1}, {658, 3}} {
		if got := e.regenerationRate(c.now); got != c.want {
			t.Fatalf("rate at %d = %d, want %d", c.now, got, c.want)
		}
	}
}
