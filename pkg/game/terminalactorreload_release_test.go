package game

import (
	"os"
	"reflect"
	"testing"

	"againrom/pkg/sim"
)

// A non-ground mover leaves a Stage 1 terminal row, which its SAV root reads
// as dying. SAVE then LOAD must resume the saved hash, twice.
func TestReleaseNonGroundTerminalActorReloads(t *testing.T) {
	hop := func(from *FrontEnd, label string) *FrontEnd {
		t.Helper()
		before, _ := from.LiveWorld()
		want := before.CurrentTerminalActors()
		raw, err := os.ReadFile(saveCorpseMission(t, from, t.TempDir()))
		if err != nil {
			t.Fatal(err)
		}
		next := releaseFront(t)
		open, town, err := next.RestoreOriginal(raw)
		if err != nil || town {
			t.Fatalf("%s: LOAD: town=%t %v", label, town, err)
		}
		if err := next.App(label).OpenMission(open); err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		after, _ := next.LiveWorld()
		if !reflect.DeepEqual(after.CurrentTerminalActors(), want) {
			t.Fatalf("%s: terminal rows %+v, want %+v", label, after.CurrentTerminalActors(), want)
		}
		if after.Hash() != before.Hash() || after.Tick() != before.Tick() {
			t.Fatalf("%s: LOAD hash %016x tick %d, saved %016x tick %d", label, after.Hash(), after.Tick(), before.Hash(), before.Tick())
		}
		return next
	}
	f := releaseFront(t)
	open := f.MissionOpenerWith(10, f.NextParty())
	if _, _, _, _, _, _, _, _, _, _, err := open(); err != nil {
		t.Fatal(err)
	}
	f = hop(f, "fresh mission save")
	w, _ := f.LiveWorld()
	var victim sim.EntityID
	for _, e := range w.Entities() {
		if e.Domain != sim.DomainGround && e.Alive() && e.Owner != sim.SelfSlot {
			victim = e.ID
			break
		}
	}
	if victim == 0 {
		t.Fatal("mission 10 has no living non-ground mover")
	}
	f.LiveKill(uint32(victim))
	var rows []sim.CurrentTerminalActor
	for i := 0; i < 100 && len(rows) == 0; i++ {
		f.LiveAdvance(20)
		w, _ = f.LiveWorld()
		rows = w.CurrentTerminalActors()
	}
	if len(rows) != 1 || rows[0].ID != victim || rows[0].Stage != uint8(sim.DecayFallen) {
		t.Fatalf("killed non-ground mover %d left terminal rows %+v, want one Stage 1 row", victim, rows)
	}
	hop(hop(f, "terminal actor save"), "reloaded terminal actor save")
}
