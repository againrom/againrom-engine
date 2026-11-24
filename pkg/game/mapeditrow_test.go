package game

import (
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/ui"
)

// The first game's row has every capability on; the second game's row takes
// units, structures and sacks, writes script words, and offers the seams the
// terrain brush, the palettes and the save use.
func TestEditRowsHoldBothFormats(t *testing.T) {
	first, second := editRows[alm.FirstFormat], editRows[alm.SecondFormat]
	if first == nil || second == nil || rowFor(nil) != first {
		t.Fatal("a format has no row")
	}
	if first.moves != nil || first.adds != nil || first.deletes != nil || first.sets != nil || first.seams != nil || first.newSteps != nil || first.admissions != nil ||
		!first.fields || !first.randomTerrain || !first.manyValues || !first.wholeScripts || !first.unitOverrides || !first.spellSets || !first.playerRecords ||
		!first.storedKeys || !first.playableRing || !first.workspace || first.templatePlacing || first.centredCells || first.structureKind == nil {
		t.Fatalf("the first game's row leaves a capability off: %+v", first)
	}
	for _, k := range []ui.RecordKind{ui.KindUnit, ui.KindStructure, ui.KindSack} {
		if !second.moves.has(k) || !second.adds.has(k) || !second.deletes.has(k) {
			t.Fatalf("the second game's row refuses kind %v", k)
		}
	}
	for _, k := range []ui.RecordKind{ui.KindScenery, ui.KindStock, ui.KindEnchantment, ui.KindPlayer, ui.KindAction, ui.KindCondition} {
		node := k == ui.KindAction || k == ui.KindCondition
		if second.moves.has(k) || second.deletes.has(k) != node || second.adds.has(k) != (k == ui.KindPlayer || node) {
			t.Fatalf("the second game's row takes kind %v", k)
		}
	}
	for _, k := range []ui.RecordKind{ui.KindUnit, ui.KindStructure, ui.KindSack, ui.KindStock, ui.KindPlayer} {
		if !second.fieldKinds.has(k) {
			t.Fatalf("the second game's row lists no fields for %v", k)
		}
	}
	if second.fieldKinds.has(ui.KindScenery) || second.fieldKinds.has(ui.KindEnchantment) || first.unitNoRowWord == "" || second.unitNoRowWord == "" || first.unitNoRowID == nil || second.unitNoRowID == nil || !second.playerRecords || !second.structureOwner || !first.structureOwner {
		t.Fatalf("the second game's row lists the wrong fields: %+v", second)
	}
	for _, k := range []ui.RecordKind{ui.KindAction, ui.KindCondition, ui.KindTrigger} {
		if !second.sets.has(k) {
			t.Fatalf("the second game's row sets no kind %v", k)
		}
	}
	if second.sets.has(ui.KindUnit) || second.wholeScripts || second.fields || second.randomTerrain || second.refusal == "" {
		t.Fatalf("the second game's row is wider than its tools: %+v", second)
	}
	offered := func(r *editRow, seam reflect.Type) bool { return r.seams == nil || slices.Contains(r.seams, seam) }
	for _, seam := range []reflect.Type{reflect.TypeFor[ui.InspectionChecker](), reflect.TypeFor[ui.InspectionTestPlay](), reflect.TypeFor[ui.InspectionBrush](), reflect.TypeFor[ui.InspectionNewRecords](), reflect.TypeFor[ui.InspectionFieldSet](),
		reflect.TypeFor[ui.InspectionInPlace](), reflect.TypeFor[ui.InspectionSaveFolder](),
		reflect.TypeFor[ui.InspectionSelection](), reflect.TypeFor[ui.InspectionGroups](),
		reflect.TypeFor[ui.InspectionItemLook]()} {
		if !offered(second, seam) || !offered(first, seam) {
			t.Fatalf("seam %v is not offered", seam)
		}
	}
	for _, seam := range []reflect.Type{reflect.TypeFor[ui.InspectionRecords](), reflect.TypeFor[ui.InspectionMessages](),
		reflect.TypeFor[ui.InspectionMapSet](), reflect.TypeFor[ui.InspectionAreaCopy](), reflect.TypeFor[ui.InspectionScript]()} {
		if offered(second, seam) || !offered(first, seam) {
			t.Fatalf("seam %v is offered wrongly", seam)
		}
	}
}

// A host with no map open offers every seam; the editor reads it as the first
// game's.
func TestHostOffersEverySeamOnTheFirstRow(t *testing.T) {
	s := &mapInspectionEdits{}
	if !s.Offers(reflect.TypeFor[ui.InspectionRecords]()) || !s.Offers(reflect.TypeFor[ui.InspectionChecker]()) {
		t.Fatal("a seam of the first game's row is off")
	}
	if _, ok := ui.Seam[ui.InspectionChecker](s); !ok {
		t.Fatal("the checker seam is not read from the first game's host")
	}
}
