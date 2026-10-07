package game

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func typedCellSave(t *testing.T, f *FrontEnd) []byte {
	t.Helper()
	dir := t.TempDir()
	prepared, err := f.SaveDialogSeams(SaveStore{Dir: dir}, OriginalStore{}).Prepare(ui.SaveRequest{
		OnMap: true, Directory: dir, Name: "typed cell", Format: ui.SaveSAV,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := prepared.Commit(true); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "typed cell.sav"))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func typedCellLoad(t *testing.T, raw []byte) *FrontEnd {
	t.Helper()
	f := releaseFront(t)
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatalf("mission LOAD: town=%t error=%v", town, err)
	}
	if err := f.App("typed actor cell").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	return f
}

func typedCellWorld(t *testing.T, f *FrontEnd, cell uint16, key uint32) *sim.World {
	t.Helper()
	w, ok := f.LiveWorld()
	if !ok {
		t.Fatal("mission has no live World")
	}
	_, cells, _, present := w.SavedActorMotions()
	if !present {
		t.Fatal("mission lost saved actor cells")
	}
	for _, c := range cells {
		if c.Cell != cell {
			continue
		}
		if c.Ground.Key == key {
			t.Fatalf("cell %#04x retains terminal ground=%+v", cell, c.Ground)
		}
		return w
	}
	t.Fatalf("saved actor cell %#04x is absent", cell)
	return nil
}

func typedCellDocument(t *testing.T, raw []byte, cell uint16, key uint32, actor sim.EntityID, checkAction bool) {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	if doc.World == nil {
		t.Fatal("written SAV has no World")
	}
	cellFound := false
	for _, c := range doc.World.Cells {
		if c.Cell == cell {
			cellFound = true
			if c.GroundActor == key {
				t.Fatalf("written cell %#04x retained terminal GroundActor key: %#x", cell, c.GroundActor)
			}
		}
	}
	if !cellFound {
		t.Fatalf("written SAV lacks cell %#04x", cell)
	}
	a, err := readCurrentActions(&doc)
	if err != nil || a == nil {
		t.Fatal("written SAV lacks readable current actions", err)
	}
	bindingFound := false
	for _, b := range a.Bindings {
		if b.ID != actor || b.Structure {
			continue
		}
		if b.Missing || b.Object == 0 || int(b.Object) > len(doc.Objects) || doc.Objects[b.Object-1].Class != "Unit" || !slices.Contains(doc.DeadActors, b.Object) {
			t.Fatalf("actor %d lost its exact Unit/DeadActors root: %+v", actor, b)
		}
		identity, err := savedStructureValue(&doc.Objects[b.Object-1], "Identity")
		if err != nil || identity != key {
			t.Fatalf("actor %d Unit identity=%#x, want GroundActor key %#x: %v", actor, identity, key, err)
		}
		bindingFound = true
	}
	if !bindingFound {
		t.Fatalf("current action binding for corpse actor %d is absent", actor)
	}
	if checkAction {
		for _, action := range a.Actions.Actors {
			if action.Entity == actor {
				if action.ImportedMotion || action.MotionIssue != "" {
					t.Fatalf("corpse actor %d gained a current imported motion: %+v", actor, action)
				}
				return
			}
		}
		t.Fatalf("current action for corpse actor %d is absent", actor)
	}
}

func TestReleaseSavedTypedActorCellColdReload(t *testing.T) {
	cases := []struct {
		file, hash string
		cell       uint16
		key        uint32
		actor      sim.EntityID
		runtime    uint32
		mapUnit    uint16
		stage      uint8
		hp         int16
	}{
		{"game0005.sav", "715d7f9d20b90a966ab6e9f14fb8fa4cea90097a1a41f2488ed8d5d38da414a4", 0x793e, 0x0316a100, 89, 90, 173, 3, -40},
		{"game0006.sav", "b652cb4c5745b6dfa1a4cec12ea3be1dbb5991f44bbde97716755d6a5b8197ba", 0x6242, 0x02f76d70, 78, 79, 160, 4, -261},
	}
	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			_, source := groundCorpusFile(t, "2026-09-27/oldsaves7/"+c.file, c.hash)
			first := typedCellLoad(t, source)
			w := typedCellWorld(t, first, c.cell, c.key)
			deadFound := false
			for _, d := range w.OriginalDeadActors() {
				if d.ID == c.actor {
					deadFound = d.Source.Identity == c.key && d.Source.MapUnitID == c.mapUnit && d.Current.RuntimeID == c.runtime && d.Current.Stage == c.stage && d.Current.HP == c.hp
				}
			}
			if !deadFound {
				t.Fatalf("original dead actor %d differs from the source tuple", c.actor)
			}
			beforeHash := w.Hash()
			written := typedCellSave(t, first)
			if after, _ := first.LiveWorld(); after.Hash() != beforeHash {
				t.Fatal("SAVE changed the source World")
			}
			typedCellDocument(t, written, c.cell, c.key, c.actor, true)
			cold := typedCellLoad(t, written)
			if got := typedCellWorld(t, cold, c.cell, c.key).Hash(); got != beforeHash {
				t.Fatalf("first cold LOAD changed World: %016x -> %016x", beforeHash, got)
			}
			for tick := 0; tick < 25; tick++ {
				first.LiveAdvance(1)
				cold.LiveAdvance(1)
				left, _ := first.LiveWorld()
				right, _ := cold.LiveWorld()
				if left.Hash() != right.Hash() {
					t.Fatalf("World diverged on continued tick %d: %016x != %016x", tick+1, left.Hash(), right.Hash())
				}
			}
			beforeSecond, _ := cold.LiveWorld()
			secondHash := beforeSecond.Hash()
			written = typedCellSave(t, cold)
			if after, _ := cold.LiveWorld(); after.Hash() != secondHash {
				t.Fatal("second SAVE changed the source World")
			}
			typedCellDocument(t, written, c.cell, c.key, c.actor, false)
			second := typedCellLoad(t, written)
			if got := typedCellWorld(t, second, c.cell, c.key).Hash(); got != secondHash {
				t.Fatalf("second cold LOAD changed World: %016x -> %016x", secondHash, got)
			}
		})
	}
}
