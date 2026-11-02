package game

import (
	"bytes"
	"image"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// The palette table puts each structure kind and object class in one
// palette: what can be selected, destroyed or switched and the bridges in
// Structures, decoration in Objects, trees in Trees, and a kind the data
// does not settle in Structures with the field that fails.
func TestEditorPaletteTableSettlesEachRecord(t *testing.T) {
	for _, c := range []struct {
		name    string
		class   *terrain.StructureClass
		hp      int32
		hpKnown bool
		want    ui.Palette
		why     string
	}{
		{"no class", nil, 0, true, ui.PaletteStructures, "structures.reg: no class"},
		{"a bridge", &terrain.StructureClass{VariableSize: true, Indestructible: true}, 0, true, ui.PaletteStructures, ""},
		{"a usable gate", &terrain.StructureClass{Usable: true, Indestructible: true}, 0, true, ui.PaletteStructures, ""},
		{"a house that can be destroyed", &terrain.StructureClass{}, 500, true, ui.PaletteStructures, ""},
		{"a flat mark", &terrain.StructureClass{Flat: true, Indestructible: true}, 0, true, ui.PaletteStructures, "Flat: no field marks a fixed bridge"},
		{"no Buildings entry", &terrain.StructureClass{Indestructible: true}, 0, false, ui.PaletteStructures, "Buildings: no maximum health"},
		{"indestructible with health", &terrain.StructureClass{Indestructible: true}, 1000, true, ui.PaletteStructures, "Indestructible with Buildings maximum health above zero"},
		{"a fence", &terrain.StructureClass{Indestructible: true}, 0, true, ui.PaletteObjects, ""},
	} {
		if p, why := structurePalette(c.class, c.hp, c.hpKnown); p != c.want || why != c.why {
			t.Errorf("%s: palette %d (%q), want %d (%q)", c.name, p, why, c.want, c.why)
		}
	}
	for _, c := range []struct {
		name  string
		class *data.ObjectClass
		want  ui.Palette
	}{
		{"no class", nil, ui.PaletteObjects},
		{"a stone", &data.ObjectClass{DeadObject: -1, FireObject: -1}, ui.PaletteObjects},
		{"a standing tree", &data.ObjectClass{DeadObject: 4, FireObject: -1}, ui.PaletteTrees},
		{"a dead tree", &data.ObjectClass{DeadObject: -1, FireObject: -2}, ui.PaletteTrees},
	} {
		if p := objectPalette(c.class); p != c.want {
			t.Errorf("%s: palette %d, want %d", c.name, p, c.want)
		}
	}

	// The object list reads each row's palette from the registry.
	class := func(id int32, desc string, dead int32) []synth.RegNode {
		return []synth.RegNode{
			{Name: "ID", Kind: 0x02, Int: id}, {Name: "File", Kind: 0x02, Int: 0}, {Name: "Index", Kind: 0x02, Int: 0},
			{Name: "Width", Kind: 0x02, Int: 8}, {Name: "Height", Kind: 0x02, Int: 8},
			{Name: "DescText", Kind: 0x00, Str: desc}, {Name: "DeadObject", Kind: 0x02, Int: dead},
		}
	}
	catalog, err := LoadObjectCatalog(memorySource{ObjectRegistry: synth.ObjectsReg([]string{`trees\oak`}, class(0, "Oak", 1), class(1, "Oak dead", -1), class(4, "Rock", -1))})
	if err != nil {
		t.Fatal(err)
	}
	listed := &terrain.StaticSet{}
	for b := 1; b <= 0xff; b++ {
		if _, ok := catalog.ByCode(byte(b)); ok {
			listed.Classes[b] = &terrain.StaticClass{Frame: &terrain.StaticFrame{}}
		}
	}
	got := map[string]ui.Palette{}
	for _, r := range objectBrushRows(catalog, listed)[1:] {
		got[r.Name] = r.Palette
	}
	if got["Oak"] != ui.PaletteTrees || got["Rock"] != ui.PaletteObjects || len(got) != 3 {
		t.Fatalf("the object list's palettes are %v", got)
	}
}

// paletteMapCell is an open land cell of the forest fixture map.
var paletteMapCell = image.Pt(8, 8)

// paletteInspector is the overlap fixture's kinds 1 (a house) and 5, with
// kind 2 a fence that only decorates and kind 3 a flat mark the data does
// not settle, and an object list of an oak tree and a stone.
func paletteInspector() *MapInspector {
	x := overlapInspector()
	x.structures.Classes[2] = &terrain.StructureClass{ID: 2, Name: "Fence", TileWidth: 1, TileHeight: 1, FullHeight: 1, Indestructible: true}
	x.structures.Classes[3] = &terrain.StructureClass{ID: 3, Name: "Mark", TileWidth: 1, TileHeight: 1, Flat: true, Indestructible: true}
	x.objectRows = []ui.BrushObject{{Code: 0}, {Code: forestOakish, Name: "Oak", Palette: ui.PaletteTrees}, {Code: 7, Name: "Stone", Palette: ui.PaletteObjects}}
	x.statics = forestStatics(forestOakish, 7)
	return x
}

// paletteEditorOn opens raw at 1920x1080 over paletteInspector.
func paletteEditorOn(t *testing.T, raw []byte) (*ui.MapEditor, *editorClicker) {
	t.Helper()
	e := paletteInspector().Editor()
	openSelectionMap(t, e, "palettes.alm", raw)
	e.Layout(1920, 1080)
	return e, &editorClicker{t: t, now: time.Unix(1, 0)}
}

// clickMapCell clicks the middle of cell c with the view on it.
func clickMapCell(c *editorClicker, e *ui.MapEditor, cell image.Point) {
	e.ShowCell(cell)
	p := e.CellPoint(cell)
	c.click(e, image.Rect(p.X, p.Y, p.X+1, p.Y+1), true, "the map cell")
}

// On the game's host the palettes hold what its table gives: Structures the
// house, the unsettled mark and kind 5; Objects the fence and the stone;
// Trees the oak.
func TestEditorPalettesHoldTheTablesRecords(t *testing.T) {
	e, _ := paletteEditorOn(t, forestMap(t))
	_, unsettled := structurePalettes(paletteInspector().structures, paletteInspector().table)
	if len(unsettled) != 1 || unsettled[3] == "" {
		t.Fatalf("the unsettled kinds are %v, want the mark", unsettled)
	}
	for _, c := range []struct {
		p       ui.Palette
		kinds   []int64
		objects []int
	}{
		{ui.PaletteStructures, []int64{1, 3, 5}, nil},
		{ui.PaletteObjects, []int64{2}, []int{7}},
		{ui.PaletteTrees, nil, []int{forestOakish}},
	} {
		kinds, objects := e.PaletteHolds(c.p)
		if !slices.Equal(kinds, c.kinds) || !slices.Equal(objects, c.objects) {
			t.Errorf("palette %d holds kinds %v and objects %v, want %v and %v", c.p, kinds, objects, c.kinds, c.objects)
		}
	}
}

// saveTyped saves a map of no file of its own through Save As to a typed
// path and reads the file back.
func saveTyped(t *testing.T, e *ui.MapEditor) []byte {
	t.Helper()
	path := filepath.Join(t.TempDir(), "saved.alm")
	if err := e.Key("saveas"); err != nil {
		t.Fatal(err)
	}
	e.TextInput(path)
	if err := pastNotice(e, "enter"); err != nil {
		t.Fatalf("Save As: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// A record writes the same bytes from its new palette as from the old one:
// the fence placed from the Objects grid as from the Structure key's Add
// window, which the Structures grid opened for it before; the oak planted
// from Forest, one kind, solid, on one cell, as the object brush's row of it
// painted before. The placed fence is still a structure record. Opening
// every palette and saving with no edit writes the opened bytes.
func TestEditorPaletteRoutesWriteTheSameBytes(t *testing.T) {
	raw := forestMap(t)

	e, c := paletteEditorOn(t, raw)
	if err := e.Key("objects"); err != nil {
		t.Fatal(err)
	}
	if n := e.PlaceGridLen(); n != 1 {
		t.Fatalf("the Objects grid holds %d kinds, want the fence", n)
	}
	r, ok := e.PlaceGridCellRect(0)
	c.click(e, r, ok, "the Objects grid's fence")
	clickMapCell(c, e, paletteMapCell)
	_ = e.Key("escape")
	_ = e.Key("escape")
	if e.PlaceGridLen() != 1 {
		t.Fatal("Esc did not end the placing back in the object brush")
	}
	byGrid := bytes.Clone(e.Document().SourceBytes)
	if bytes.Equal(byGrid, raw) {
		t.Fatalf("the Objects grid placed nothing (%q)", e.Message())
	}
	if !slices.ContainsFunc(e.Document().Records, func(rec ui.InspectionRecord) bool {
		return rec.Kind == ui.KindStructure && rec.Cell == paletteMapCell && slices.Contains(rec.Details, "Kind: 2 (art key 2)")
	}) {
		t.Fatalf("the fence placed from the Objects grid is no structure record: %+v", e.Document().Records)
	}

	w, _ := paletteEditorOn(t, raw)
	addWindow(t, w, "structure", map[string]string{"kind": "2", "cell": "8,8"})
	if !bytes.Equal(w.Document().SourceBytes, byGrid) {
		t.Fatal("the Add window and the Objects grid write different bytes for the fence")
	}

	f, fc := paletteEditorOn(t, raw)
	_ = f.Key("objects")
	r, ok = f.ListButtonRect(2, 1)
	fc.click(f, r, ok, "the Forest button")
	r, ok = f.PlaceGridCellRect(0)
	fc.click(f, r, ok, "the oak's Forest cell")
	r, ok = f.ListButtonRect(4, 4)
	fc.click(f, r, ok, "Solid")
	for range 20 {
		_ = f.Key("radiusdown")
	}
	clickMapCell(fc, f, paletteMapCell)
	// The object brush's stroke of a row it lists, radius 0 on one cell.
	o, _ := paletteEditorOn(t, raw)
	row, err := objectEdits(t, o).Paint(ui.BrushStroke{Plane: ui.BrushPlaneObjects, Value: forestOakish, Cells: []image.Point{paletteMapCell}})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(f.Document().SourceBytes, raw) || !bytes.Equal(f.Document().SourceBytes, row.SourceBytes) {
		t.Fatalf("Forest wrote other bytes than the object brush's oak row (%q)", f.Message())
	}

	// The fixture map's unresolved art refuses a save; a new map saves.
	n := paletteInspector().Editor()
	n.Layout(1920, 1080)
	if err := n.Open(NewMapSource("40x40")); err != nil {
		t.Fatal(err)
	}
	opened := bytes.Clone(n.Document().SourceBytes)
	nc := &editorClicker{t: t, now: time.Unix(1, 0)}
	for _, k := range []string{"escape", "structures", "objects"} {
		if err := n.Key(k); err != nil {
			t.Fatal(err)
		}
	}
	r, ok = n.ListButtonRect(2, 1)
	nc.click(n, r, ok, "the Forest button")
	if n.Dirty() {
		t.Fatal("opening the palettes edited the map")
	}
	if got := saveTyped(t, n); !bytes.Equal(got, opened) {
		t.Fatalf("a save with no edit wrote %d bytes, not the opened %d", len(got), len(opened))
	}
}
