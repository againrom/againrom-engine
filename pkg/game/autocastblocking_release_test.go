package game

import (
	"encoding/binary"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// changeSavedStructureBlocking returns raw with the Building record of st
// carrying mask as its blocking word: a changed structure.
func changeSavedStructureBlocking(t *testing.T, raw []byte, st sim.Structure, mask uint32) []byte {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil || doc.World == nil {
		t.Fatal("mission SAV", err)
	}
	changed := 0
	for _, index := range doc.World.Buildings {
		r := &doc.Objects[index-1]
		kind, err := savedStructureValue(r, "B40")
		if err != nil || kind != uint32(uint8(st.Kind)) {
			continue
		}
		at := false
		for _, v := range r.Raw {
			if v.Name == "Block12" && len(v.Bytes) >= 2 && int32(v.Bytes[0]) == st.Col && int32(v.Bytes[1]) == st.Row {
				at = true
			}
		}
		if !at {
			continue
		}
		if err := savedStructureSetValue(r, "B64", mask); err != nil {
			t.Fatal(err)
		}
		for _, v := range r.Raw {
			if v.Name == "B52" && len(v.Bytes) == 22 {
				binary.LittleEndian.PutUint32(v.Bytes[18:], mask)
			}
		}
		changed++
	}
	if changed != 1 {
		t.Fatalf("%d Building records at (%d,%d)", changed, st.Col, st.Row)
	}
	out, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func structureByID(t *testing.T, w *sim.World, id sim.StructureID) sim.Structure {
	t.Helper()
	for _, st := range w.Structures() {
		if st.ID == id {
			return st
		}
	}
	t.Fatalf("structure %d absent", id)
	return sim.Structure{}
}

// autoCastChangedBlocking is the witness on either game's root. A mission
// saved with a stored automatic-casting level, its first structure's blocking
// word changed in the SAV, keeps the changed word through a cold LOAD. That
// World's own byte form is the automatic-casting wrapper; its Snapshot resume
// in a fresh process keeps the changed word too, where the started mission
// holds the original one.
func autoCastChangedBlocking(t *testing.T) {
	f := autoCastLevelFront(t)
	autoCastLevelMission(t, f)
	column := f.optionColumn()
	level := -1
	for _, offer := range ui.GameOptionAutoCast.Offers(column) {
		if offer != 0 && offer != ui.GameOptionAutoCast.Default(column) {
			level = offer
			break
		}
	}
	if level < 0 {
		t.Fatal("no stored automatic-casting level is offered")
	}
	if err := f.setGameOption(true, ui.GameOptionAutoCast, level); err != nil {
		t.Fatal(err)
	}
	f.live.tick()
	if !f.live.world.AutoCastStored(sim.SelfSlot) {
		t.Fatalf("level %d is not stored", level)
	}
	structures := f.live.world.Structures()
	if len(structures) == 0 {
		t.Fatal("mission has no structure")
	}
	start := structures[0]
	mask := start.Blocking ^ 1
	if mask == start.Attach {
		mask ^= 2
	}
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(true)
	if err != nil || filepath.Ext(name) != ".sav" {
		t.Fatal("ordinary SAV", name, err)
	}
	raw, err := store.Read(name)
	if err != nil {
		t.Fatal(err)
	}
	cold := autoCastColdLoad(t, changeSavedStructureBlocking(t, raw, start, mask))
	if got := structureByID(t, cold.live.world, start.ID); got.Blocking != mask || !cold.live.world.AutoCastStored(sim.SelfSlot) {
		t.Fatalf("cold LOAD blocking %#x, want the saved %#x (mission start %#x)", got.Blocking, mask, start.Blocking)
	}
	s, _, err := cold.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.World) == 0 || s.World[0] != 121 || !sim.HasStructureBlockingForm(s.World) {
		t.Errorf("snapshot World form %v hides the structure blocking", s.World[:min(1, len(s.World))])
	}
	resumed := autoCastLevelFront(t)
	open, town, err := resumed.Restore(s)
	if err != nil || town {
		t.Fatal("Snapshot resume", town, err)
	}
	if err := resumed.App("automatic-casting Snapshot resume").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	if got := structureByID(t, resumed.live.world, start.ID); got.Blocking != mask {
		t.Fatalf("Snapshot resume blocking %#x, want the saved %#x (mission start %#x)", got.Blocking, mask, start.Blocking)
	}
	t.Logf("%s: level %d; structure %d (kind %d at %d,%d) blocking %#x -> %#x; SAV %d bytes", f.Archives.Game(), level, start.ID, start.Kind, start.Col, start.Row, start.Blocking, mask, len(raw))
}

// TestReleaseAutoCastSAVKeepsChangedStructureBlocking runs on a first-game
// root.
func TestReleaseAutoCastSAVKeepsChangedStructureBlocking(t *testing.T) {
	autoCastChangedBlocking(t)
}

// TestReleaseSecondMissionSaveAutoCastChangedBlocking runs on a second-game
// root.
func TestReleaseSecondMissionSaveAutoCastChangedBlocking(t *testing.T) {
	secondGameRoot(t)
	autoCastChangedBlocking(t)
}
