package game

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func selectionFront(t *testing.T) *FrontEnd {
	t.Helper()
	f := structureFront(t)
	f.Campaign = resolved(saveCampaign(), nil)
	return f
}

// structureSelectionSave is the literal two-Building document whose
// /Objects/Selection holds the given runtime IDs. The first Building's
// RuntimeID is the literal 70 its fixture writes at the record's +12.
func structureSelectionSave(t *testing.T, selection ...int32) []byte {
	t.Helper()
	file, err := sav.Open(completeDocumentTail1115(t, selectionFront(t), structureSave1114(t, 0, false)))
	if err != nil {
		t.Fatal(err)
	}
	if err := file.SetStoreIntArray("Objects", "Selection", selection); err != nil {
		t.Fatal(err)
	}
	return file.Marshal()
}

func openStructureSelection(t *testing.T, raw []byte) (*FrontEnd, *ui.App, error) {
	t.Helper()
	f := selectionFront(t)
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		return f, nil, fmt.Errorf("restore: town=%v %w", town, err)
	}
	app := f.App("structure selection")
	app.Layout(1024, 768)
	return f, app, app.OpenMission(open)
}

func exportedSelection(t *testing.T, f *FrontEnd) []uint32 {
	t.Helper()
	before := f.live.world.Hash()
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, "structure selection")
	if err != nil {
		t.Fatalf("SAVE: %v", err)
	}
	if f.live.world.Hash() != before {
		t.Fatal("SAVE changed the live World")
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	return appLeafInts(t, &doc, "/Objects/Selection")
}

// An original SAV whose selection names a Building loads, the Building is the
// viewer's selected structure, and the next SAVE writes its runtime ID.
func TestSAVSelectionOfAStructureLoadsAndSaves(t *testing.T) {
	const building = 70
	f, app, err := openStructureSelection(t, structureSelectionSave(t, building))
	if err != nil {
		t.Fatalf("LOAD of a structure selection: %v", err)
	}
	view := f.live.view.SaveApplication()
	if len(view.Selection) != 0 || !slices.Equal(view.Structures, []uint32{0}) {
		t.Fatalf("viewer selection after LOAD: actors %v structures %v", view.Selection, view.Structures)
	}
	if got := exportedSelection(t, f); !slices.Equal(got, []uint32{building}) {
		t.Fatalf("next SAVE selection = %v, want [%d]", got, building)
	}
	empty, _, err := openStructureSelection(t, structureSelectionSave(t))
	if err != nil {
		t.Fatal(err)
	}
	if empty.live.world.Hash() != f.live.world.Hash() {
		t.Fatal("the structure selection reached the World hash")
	}
	if got := exportedSelection(t, empty); len(got) != 0 {
		t.Fatalf("empty selection saved as %v", got)
	}
	// A selected unit replaces the structure; the SAVE writes the unit alone.
	var actor sim.Entity
	for _, e := range f.live.world.Entities() {
		if e.SourceBinding.TypeID == 35 {
			actor = e
		}
	}
	if err := app.HeadlessSelectEntity(uint32(actor.ID)); err != nil {
		t.Fatal(err)
	}
	if view := f.live.view.SaveApplication(); !slices.Equal(view.Selection, []uint32{uint32(actor.ID)}) || len(view.Structures) != 0 {
		t.Fatalf("unit selection over a structure: %+v", view)
	}
	if got := exportedSelection(t, f); !slices.Equal(got, []uint32{actor.SourceBinding.RuntimeID}) {
		t.Fatalf("unit SAVE selection = %v, want [%d]", got, actor.SourceBinding.RuntimeID)
	}
}

// A selection id that neither an actor nor a structure carries stays a refused
// LOAD that names the id.
func TestSAVSelectionOfAnUnboundIDIsRefused(t *testing.T) {
	_, _, err := openStructureSelection(t, structureSelectionSave(t, 4242))
	if err == nil || !strings.Contains(err.Error(), "runtime ID 4242") {
		t.Fatalf("an unbound selection id: %v", err)
	}
}
