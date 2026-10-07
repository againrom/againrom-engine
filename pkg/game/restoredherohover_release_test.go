package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// openRestoredVictory loads the original mission-17 victory SAV through the
// ordinary front end.
func openRestoredVictory(t *testing.T, label string) *mapWorld {
	t.Helper()
	_, saved := groundCorpusFile(t, "2026-09-24/game0017-victory.sav", "a30de01ec3d96d58a3a244b9a536a3e49c04a4d45098cb828dabb908756488c3")
	f := releaseFront(t)
	a := f.App(label)
	a.Layout(1024, 768)
	open, _, err := f.RestoreOriginal(saved)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	return f.live
}

// giveHeroBandClass sets one party actor's class back to its hero-band TypeID,
// the fallback a restored member keeps when its equipment selects no class,
// through the same action restore a saved class uses.
func giveHeroBandClass(t *testing.T, mw *mapWorld, id sim.EntityID) int32 {
	t.Helper()
	a := mw.world.Actions()
	for i := range a.Actors {
		if a.Actors[i].Entity != id || a.Actors[i].Current == nil {
			continue
		}
		e, _ := mw.entity(id)
		class := e.TypeID
		if class < 0x20 || class >= 0x40 || data.ComposesFigure(class) {
			t.Fatalf("party actor %d TypeID %d is not a hero-band class above the compose limit", id, class)
		}
		a.Actors[i].Current.Class = &class
		if err := mw.world.RestoreActions(a, nil); err != nil {
			t.Fatal(err)
		}
		if e, _ := mw.entity(id); e.Class != class {
			t.Fatalf("party actor %d class %d, want %d", id, e.Class, class)
		}
		return class
	}
	t.Fatalf("party actor %d has no current action row", id)
	return 0
}

// A hero restored from an original SAV carries the class its equipment
// selects, and hovering it shows its composed figure. The hero-band class
// survives only as the fallback for a member whose equipment selects none;
// the recorded figure still wins over the class test for it.
func TestReleaseRestoredHeroHoverShowsItsFigure(t *testing.T) {
	mw := openRestoredVictory(t, "restored hero hover")
	checked := 0
	for _, id := range mw.mission.ids {
		e, ok := mw.entity(id)
		if !ok {
			continue
		}
		want := mw.unitFigure(id)
		if want == nil {
			t.Fatalf("party actor %d class %d has no composed figure", id, e.Class)
		}
		if got := mw.inspectionUnitPicture(uint32(id)); !imagesEqual(got, want) {
			t.Fatalf("party actor %d class %d hover picture is not its figure", id, e.Class)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("fixture has no restored party actor")
	}
	id := mw.mission.ids[0]
	class := giveHeroBandClass(t, mw, id)
	want := mw.unitFigure(id)
	if want == nil {
		t.Fatalf("party actor %d has no composed figure", id)
	}
	if got := mw.inspectionUnitPicture(uint32(id)); !imagesEqual(got, want) {
		t.Fatalf("party actor %d hero-band class %d hover picture is not its figure", id, class)
	}
}
