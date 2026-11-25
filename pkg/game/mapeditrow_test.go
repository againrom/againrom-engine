package game

import (
	"reflect"
	"slices"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/base"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// The first game's row has every capability on; the second game's row takes
// units, structures and sacks, adds and removes type-9 cell records, writes
// script words, and offers the seams the
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
		node := k == ui.KindAction || k == ui.KindCondition || k == ui.KindEnchantment
		if second.moves.has(k) || second.deletes.has(k) != node || second.adds.has(k) != (k == ui.KindPlayer || node) {
			t.Fatalf("the second game's row takes kind %v", k)
		}
	}
	for _, k := range []ui.RecordKind{ui.KindUnit, ui.KindStructure, ui.KindSack, ui.KindStock, ui.KindPlayer, ui.KindEnchantment} {
		if !second.fieldKinds.has(k) {
			t.Fatalf("the second game's row lists no fields for %v", k)
		}
	}
	if second.fieldKinds.has(ui.KindScenery) || first.unitNoRowWord == "" || second.unitNoRowWord == "" || first.unitNoRowID == nil || second.unitNoRowID == nil || !second.playerRecords || !second.structureOwner || !first.structureOwner {
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

// The second game's row derives the ring and the marked cells as arms, notes the
// passability view's limit and shows heights as stored; the first game's row
// derives the ground arms of its tiles. Only the second game's files state a
// new map's format.
func TestSecondRowDerivesTheRingAndTheMarksAndNotesThePassabilityView(t *testing.T) {
	m := &alm.Map{Width: 20, Height: 20, Tiles: make([]uint16, 400), Overlay: make([]uint8, 400)}
	arms := editRows[alm.SecondFormat].marks.GroundArms(m, nil)
	for i, a := range arms {
		want := mapload.GroundArm(0)
		if mapload.InBorderRing(m, i%20, i/20) {
			want = mapload.GroundBorder
		}
		if a != want {
			t.Fatalf("cell %d: arm %v, want %v", i, a, want)
		}
	}
	second, first := editRows[alm.SecondFormat], editRows[alm.FirstFormat]
	if second.passNoteWord == "" || first.passNoteWord != "" || !second.rawHeights || first.rawHeights {
		t.Errorf("rows: second %q raw %v, first %q raw %v", second.passNoteWord, second.rawHeights, first.passNoteWord, first.rawHeights)
	}
	if filesOf(base.GameROM2).newMapNote == "" || filesOf(base.GameROM1).newMapNote != "" {
		t.Errorf("new map notes: second %q, first %q", filesOf(base.GameROM2).newMapNote, filesOf(base.GameROM1).newMapNote)
	}
}

func TestInspectionReadsMarksAndValuesFromEachFormatRow(t *testing.T) {
	for _, tc := range []struct {
		name   string
		format alm.Format
		raw    []byte
	}{
		{"first", alm.FirstFormat, synth.ALM(synth.ALMOptions{Width: 32, Height: 32, Type7Payload: make([]byte, 12)})},
		{"second", alm.SecondFormat, secondFixtureMap()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := editRows[tc.format]
			row := *original
			ring := row.marks.Rows[0]
			ring.Word, ring.Value = "ring", 19
			row.marks.Rows = []mapload.MarkRow{ring}
			row.marks.PlaneValueRows = []mapload.PlaneValueRow{{Value: 19, Kinds: 4, Word: "edge"}}
			editRows[tc.format] = &row
			t.Cleanup(func() { editRows[tc.format] = original })

			x := &MapInspector{tiles: &terrain.Tileset{}}
			d, err := x.inspect(tc.raw, "marks.alm")
			if err != nil {
				t.Fatal(err)
			}
			wantArm := ui.PassArmRow{Arm: ui.PassArm(mapload.GroundBorder), Word: "ring", Border: true}
			wantValue := ui.PassValueRow{Value: 19, Kinds: 4, Word: "edge"}
			if !slices.Equal(d.PassArms, []ui.PassArmRow{wantArm}) || !slices.Equal(d.PassValueRows, []ui.PassValueRow{wantValue}) {
				t.Fatalf("arm rows %+v, value rows %+v", d.PassArms, d.PassValueRows)
			}
			if len(d.PassValues) != d.Width*d.Height || d.PassValues[0] != 19 || d.Passability[0] != wantArm.Arm ||
				d.PassValues[16*d.Width+16] != 0 || d.Passability[16*d.Width+16] != 0 {
				t.Fatal("the ring and interior cells did not read the selected mark")
			}
		})
	}
}
