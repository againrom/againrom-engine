package game

import (
	"bytes"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/mapedit"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// overlapMap is a 40x40 map with two players, all owned by player 1:
// structures of kind 5 (3x2) at 10,10, 13,10 and 20,20, kind-1 structures
// (1x1) at 14,9 and 24,9, and units of class 40 (2x2) at 12,20 and 14,20.
// at moves structure 1 when it is not zero.
func overlapMap(t *testing.T, at image.Point) []byte {
	t.Helper()
	s1 := image.Pt(13, 10)
	if at != (image.Point{}) {
		s1 = at
	}
	obj := func(c image.Point, kind uint32) synth.ALMObject {
		return synth.ALMObject{X: uint32(c.X)*256 + 128, Y: uint32(c.Y)*256 + 128, Kind: kind}
	}
	raw := synth.ALM(synth.ALMOptions{Width: 40, Height: 40, Type7Payload: make([]byte, 12),
		Units: []synth.ALMUnit{
			{X: 12*256 + 128, Y: 20*256 + 128, ClassID: 40, ClassSubID: 2},
			{X: 14*256 + 128, Y: 20*256 + 128, ClassID: 40, ClassSubID: 2},
		},
		Objects: []synth.ALMObject{obj(image.Pt(10, 10), 5), obj(s1, 5), obj(image.Pt(20, 20), 5), obj(image.Pt(14, 9), 1), obj(image.Pt(24, 9), 1)},
	})
	m, err := mapedit.New(raw)
	if err != nil {
		t.Fatal(err)
	}
	for m.Count(mapedit.KindPlayer) < 2 {
		rec, err := m.NewPlayerRecord()
		if err == nil {
			_, err = m.Place(mapedit.KindPlayer, rec)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < m.Count(mapedit.KindStructure); i++ {
		if err := m.SetField(mapedit.KindStructure, i, "owner", 1); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < m.Count(mapedit.KindUnit); i++ {
		if err := m.SetField(mapedit.KindUnit, i, "owner", 1); err != nil {
			t.Fatal(err)
		}
	}
	return m.Bytes()
}

// overlapInspector is an inspector whose tables size kind 5 3x2, kind 1 1x1
// and class 40 two cells square.
func overlapInspector() *MapInspector {
	table := authoringTable()
	building := func(w, h int32) dbEntry { return dbEntry{name: "b", params: []int32{w, h, 0, 0, -1, -1, 0, 0}} }
	table.Buildings = dbCollection{{}, building(1, 1), building(1, 1), building(1, 1), building(1, 1), building(3, 2)}
	set := &terrain.StructureSet{}
	set.Classes[1] = &terrain.StructureClass{ID: 1, Name: "one", TileWidth: 1, TileHeight: 1, FullHeight: 1}
	set.Classes[5] = &terrain.StructureClass{ID: 5, Name: "s", TileWidth: 3, TileHeight: 2, FullHeight: 2}
	units := &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{40: {TileSize: 2}}}
	return &MapInspector{tiles: &terrain.Tileset{}, table: table, structures: set, units: units}
}

// overlapHost opens raw as a file for editing.
func overlapHost(t *testing.T, raw []byte) *mapInspectionEdits {
	t.Helper()
	path := filepath.Join(t.TempDir(), "overlap.alm")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	d, err := overlapInspector().Open(path)
	if err != nil {
		t.Fatal(err)
	}
	return d.Edits.(*mapInspectionEdits)
}

// refusedEdit checks that err refuses an overlap naming other and that the
// map is unchanged.
func refusedEdit(t *testing.T, s *mapInspectionEdits, before []byte, err error, other string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), "overlaps another object ("+other+")") {
		t.Fatalf("err %v, want an overlap of %s", err, other)
	}
	if !bytes.Equal(s.model.Bytes(), before) {
		t.Fatal("a refused edit changed the map")
	}
}

func acceptedEdit(t *testing.T, s *mapInspectionEdits, before []byte, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("err %v on a free placement", err)
	}
	if bytes.Equal(s.model.Bytes(), before) {
		t.Fatal("an accepted edit changed nothing")
	}
}

func TestMapEditorAddRefusesAnOverlappingStructureOrUnit(t *testing.T) {
	raw := overlapMap(t, image.Point{})
	for _, c := range []struct {
		kind, key string
		value     int64
		on, free  image.Point
		other     string
	}{
		{"Structure", "kind", 5, image.Pt(11, 11), image.Pt(10, 13), "structure 0"},
		{"Unit", "class", 40, image.Pt(13, 21), image.Pt(16, 20), "unit 0"},
	} {
		s := overlapHost(t, raw)
		add := func(cell image.Point) error {
			_, err := s.AddRecord(ui.InspectionNewRecord{Kind: c.kind, Cell: cell, HasCell: true, Values: []ui.InspectionFieldValue{{Key: c.key, Value: c.value}}})
			return err
		}
		refusedEdit(t, s, raw, add(c.on), c.other)
		acceptedEdit(t, s, raw, add(c.free))
	}
}

func TestMapEditorMoveRefusesAnOverlappingStructureOrUnit(t *testing.T) {
	raw := overlapMap(t, image.Point{})
	for _, c := range []struct {
		kind     string
		index    int
		on, free image.Point
		other    string
	}{
		{"Structure", 1, image.Pt(19, 19), image.Pt(16, 10), "structure 2"},
		{"Unit", 1, image.Pt(11, 21), image.Pt(12, 22), "unit 0"},
	} {
		s := overlapHost(t, raw)
		_, err := s.Move(c.kind, c.index, c.on)
		refusedEdit(t, s, raw, err, c.other)
		// The Object window's cell field moves through the same check.
		_, err = s.SetObject(c.kind, c.index, nil, &c.on)
		refusedEdit(t, s, raw, err, c.other)
		_, err = s.Move(c.kind, c.index, c.free)
		acceptedEdit(t, s, raw, err)
	}
}

func TestMapEditorSetFieldRefusesAFootprintGrownOntoAStructure(t *testing.T) {
	raw := overlapMap(t, image.Point{})
	s := overlapHost(t, raw)
	// Kind 5 grows structure 3 at 14,9 to 14..16 x 9..10, onto structure 1.
	_, err := s.SetFields("Structure", 3, []ui.InspectionFieldValue{{Key: "kind", Value: 5}})
	refusedEdit(t, s, raw, err, "structure 1")
	// Structure 4 at 24,9 grows onto free cells.
	_, err = s.SetFields("Structure", 4, []ui.InspectionFieldValue{{Key: "kind", Value: 5}})
	acceptedEdit(t, s, raw, err)
}

func TestMapEditorPasteRefusesAnOverlapByName(t *testing.T) {
	raw := overlapMap(t, image.Point{})
	for _, c := range []struct {
		ref      ui.RecordRef
		on, free image.Point
		want     string
	}{
		{ui.RecordRef{Kind: "Structure", Index: 2}, image.Pt(11, 11), image.Pt(30, 30), "structure 2: footprint 11,11 3x2 overlaps another object (structure 0)"},
		{ui.RecordRef{Kind: "Unit", Index: 0}, image.Pt(15, 21), image.Pt(20, 24), "unit 0: footprint 15,21 2x2 overlaps another object (unit 1)"},
	} {
		s := overlapHost(t, raw)
		clip, _, err := s.CopyRecords([]ui.RecordRef{c.ref})
		if err != nil {
			t.Fatal(err)
		}
		r, err := s.PasteRecords(clip, c.on)
		if err != nil || r.Doc != nil || len(r.Refusals) != 1 || r.Refusals[0] != c.want {
			t.Fatalf("paste on %v: %v, %+v; want the refusal %q", c.on, err, r, c.want)
		}
		if len(r.Refused) != 1 || r.Refused[0].Footprint.Min != c.on {
			t.Fatalf("paste on %v answers the refused %+v", c.on, r.Refused)
		}
		r, err = s.PasteRecords(clip, c.free)
		if err != nil || r.Doc == nil || len(r.Refusals) != 0 {
			t.Fatalf("paste on %v: %v, %+v; want it placed", c.free, err, r)
		}
	}
}

func TestMapEditorGroupMoveJudgesTheSettledEdit(t *testing.T) {
	raw := overlapMap(t, image.Point{})
	pair := []ui.RecordRef{{Kind: "Structure", Index: 0}, {Kind: "Structure", Index: 1}}
	units := []ui.RecordRef{{Kind: "Unit", Index: 0}, {Kind: "Unit", Index: 1}}

	// A row moved along itself by one footprint: the first lands on the
	// second's old cells.
	for _, c := range []struct {
		refs []ui.RecordRef
		by   image.Point
	}{{pair, image.Pt(3, 0)}, {units, image.Pt(2, 0)}} {
		s := overlapHost(t, raw)
		r, err := s.MoveRecords(c.refs, c.by)
		if err != nil || r.Doc == nil || len(r.Refusals) != 0 {
			t.Fatalf("%v moved along their row: %v, %+v", c.refs, err, r)
		}
	}
	// Control: the first alone lands on the second.
	s := overlapHost(t, raw)
	r, err := s.MoveRecords(pair[:1], image.Pt(3, 0))
	if err != nil || r.Doc != nil || len(r.Refusals) != 1 || !strings.Contains(r.Refusals[0], "(structure 1)") {
		t.Fatalf("structure 0 alone onto structure 1: %v, %+v", err, r)
	}
	r, err = s.MoveRecords(units[:1], image.Pt(2, 0))
	if err != nil || r.Doc != nil || len(r.Refusals) != 1 || !strings.Contains(r.Refusals[0], "(unit 1)") {
		t.Fatalf("unit 0 alone onto unit 1: %v, %+v", err, r)
	}

	// Moved by 5,10, structure 1 lands on structure 2, outside the group;
	// structure 0 is placed.
	r, err = s.MoveRecords(pair, image.Pt(5, 10))
	want := "structure 1: footprint 18,20 3x2 overlaps another object (structure 2)"
	if err != nil || r.Doc == nil || len(r.Refusals) != 1 || r.Refusals[0] != want || !strings.Contains(r.Note, "Moved 1 of 2") {
		t.Fatalf("the pair moved by 5,10: %v, %+v; want %q", err, r, want)
	}
	// Structure 1 at 13,10 moved by 7,10 lands on structure 2.
	if got := s.CheckMove(pair, image.Rectangle{}, image.Pt(7, 10)); len(got) != 1 || got[0] != pair[1] {
		t.Fatalf("CheckMove of the pair onto structure 2 answers %v", got)
	}
}

func TestMapEditorRecordDragRefusesAnOverlap(t *testing.T) {
	raw := overlapMap(t, image.Point{})
	x := overlapInspector()
	e := x.Editor()
	openSelectionMap(t, e, "drag.alm", raw)
	pointAt(e, image.Pt(11, 11))
	dragInspectionRecord(e, image.Pt(10, 10), image.Pt(10, 12))
	moved := e.Document().SourceBytes
	if bytes.Equal(moved, raw) {
		t.Fatalf("a drag of structure 0 onto free cells did not move it: %q", e.Message())
	}
	// From 10,12 to 13,11 it takes row 11 of structure 1.
	dragInspectionRecord(e, image.Pt(10, 12), image.Pt(13, 11))
	if !bytes.Equal(e.Document().SourceBytes, moved) || !strings.Contains(e.Message(), "overlaps another object (structure 1)") {
		t.Fatalf("a drag of structure 0 onto structure 1: message %q", e.Message())
	}
}

func TestMapEditorPreviewHoldsARefusedAddAndTheWorldFootprint(t *testing.T) {
	s := overlapHost(t, overlapMap(t, image.Point{}))
	feet := func(d *ui.InspectionDocument, kind string, index int) image.Rectangle {
		for _, r := range d.Records {
			if r.Kind == kind && r.Index == index {
				return r.Footprint
			}
		}
		return image.Rectangle{}
	}
	add := ui.InspectionNewRecord{Kind: "Structure", Cell: image.Pt(10, 13), HasCell: true, Values: []ui.InspectionFieldValue{{Key: "kind", Value: 5}}}
	d, err := s.PreviewPlacement(add)
	if err != nil || d == nil || feet(d, "Structure", 5).Size() != image.Pt(3, 2) || feet(d, "Unit", 0).Size() != image.Pt(2, 2) {
		t.Fatalf("free Add preview: err %v", err)
	}
	add.Cell = image.Pt(11, 11)
	d, err = s.PreviewPlacement(add)
	if err == nil || d == nil || feet(d, "Structure", 5).Size() != image.Pt(3, 2) {
		t.Fatalf("refused Add preview: err %v, document %v", err, d != nil)
	}
}

func TestMapEditorLoadsAMapThatHoldsAnOverlap(t *testing.T) {
	// Structure 1 at 12,10 shares column 12 with structure 0.
	raw := overlapMap(t, image.Pt(12, 10))
	s := overlapHost(t, raw)
	_, err := s.Move("Unit", 0, image.Pt(12, 24))
	acceptedEdit(t, s, raw, err)
	before := s.model.Bytes()
	_, err = s.SetFields("Structure", 1, []ui.InspectionFieldValue{{Key: "owner", Value: 2}})
	acceptedEdit(t, s, before, err)
	before = s.model.Bytes()
	_, err = s.Move("Structure", 2, image.Pt(26, 20))
	acceptedEdit(t, s, before, err)
	before = s.model.Bytes()
	_, err = s.Move("Structure", 1, image.Pt(11, 11))
	refusedEdit(t, s, before, err, "structure 0")
}
