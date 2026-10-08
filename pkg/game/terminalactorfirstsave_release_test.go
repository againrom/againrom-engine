package game

import (
	"os"
	"reflect"
	"testing"

	"againrom/pkg/sim"
)

// terminalSaveReload saves through the ordinary SAVE seam, loads the file in a
// fresh front end and requires the saved hash, tick and terminal rows.
func terminalSaveReload(t *testing.T, f *FrontEnd, label string) *FrontEnd {
	t.Helper()
	before, _ := f.LiveWorld()
	raw, err := os.ReadFile(saveCorpseMission(t, f, t.TempDir()))
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
	if !reflect.DeepEqual(after.CurrentTerminalActors(), before.CurrentTerminalActors()) {
		t.Fatalf("%s: terminal rows %+v, want %+v", label, after.CurrentTerminalActors(), before.CurrentTerminalActors())
	}
	if after.Hash() != before.Hash() || after.Tick() != before.Tick() {
		logCurrentCarrierDiff(t, before, after)
		t.Fatalf("%s: LOAD hash %016x tick %d, saved %016x tick %d", label, after.Hash(), after.Tick(), before.Hash(), before.Tick())
	}
	return next
}

// killUntilTerminal kills one actor and advances until its terminal row exists.
func killUntilTerminal(t *testing.T, f *FrontEnd, victim sim.EntityID, ticks int) {
	t.Helper()
	f.LiveKill(uint32(victim))
	for done := 0; done < ticks; done += 50 {
		f.LiveAdvance(50)
		w, _ := f.LiveWorld()
		for _, row := range w.CurrentTerminalActors() {
			if row.ID == victim {
				return
			}
		}
	}
	t.Fatalf("actor %d left no terminal row within %d ticks", victim, ticks)
}

// A mission never saved to SAV has no document object for an actor that left
// before the first SAVE; the terminal row alone carries it.
func TestReleaseTerminalActorBeforeFirstSAV(t *testing.T) {
	for _, ground := range []bool{false, true} {
		name := "nonground"
		if ground {
			name = "ground"
		}
		t.Run(name, func(t *testing.T) {
			f := releaseFront(t)
			open := f.MissionOpenerWith(10, f.NextParty())
			if _, _, _, _, _, _, _, _, _, _, err := open(); err != nil {
				t.Fatal(err)
			}
			w, _ := f.LiveWorld()
			var victim sim.EntityID
			found := false
			for _, e := range w.Entities() {
				if (e.Domain == sim.DomainGround) == ground && e.Alive() && e.Owner != sim.SelfSlot && !e.Humanoid {
					victim, found = e.ID, true
					break
				}
			}
			if !found {
				t.Fatal("mission 10 has no candidate actor")
			}
			killUntilTerminal(t, f, victim, 24000)
			terminalSaveReload(t, terminalSaveReload(t, f, "first SAV"), "second SAV")
		})
	}
}

// Entity ID 0 is an ordinary ID. Its terminal row must SAVE and LOAD.
func TestReleaseTerminalActorZeroIDSaves(t *testing.T) {
	_, raw := groundCorpusFile(t, "2026-09-24/exp-engine-lineage/game1017-engine-from-0017.sav",
		"1c7cb03aecacc2a80b89f1d86be4f9d5bfbadf7214c7eaaa334197ebf0d2e985")
	f := releaseFront(t)
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatalf("LOAD: town=%t %v", town, err)
	}
	if err := f.App("zero ID terminal").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	w, _ := f.LiveWorld()
	present := false
	for _, e := range w.Entities() {
		present = present || e.ID == 0 && e.Alive()
	}
	if !present {
		t.Fatal("fixture has no living entity 0")
	}
	killUntilTerminal(t, f, 0, 24000)
	terminalSaveReload(t, f, "zero ID terminal SAV")
}
