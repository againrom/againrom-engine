package game

import (
	"bytes"
	"image"
	"slices"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// closeMapPanel closes the Map panel the editor opens a map with, for a test
// whose gestures are on the map under it.
func closeMapPanel(t *testing.T, e *ui.MapEditor) {
	t.Helper()
	if !e.PanelOpen() {
		t.Fatal("the Map panel did not open with the map")
	}
	_ = e.Key("escape")
	if e.PanelOpen() {
		t.Fatal("Esc did not close the Map panel")
	}
}

// pastNotice presses key and continues past the notice a map with no start
// point or no win gets before test play and a save into a mod.
func pastNotice(e *ui.MapEditor, key string) error {
	if err := e.Key(key); err != nil || !e.NoticeOpen() {
		return err
	}
	return e.Key("enter")
}

// A new map is single-player unless the author asks for multiplayer, and the
// party starts at its centre. Both are the map's first bytes: the map opens
// clean with no history, and differs from the original editor's new map only
// in +0x70 and the drop action.
func TestNewMapIsSingleByDefaultWithADropAtTheCentre(t *testing.T) {
	x := &MapInspector{tiles: &terrain.Tileset{}}
	for _, c := range []struct {
		source string
		mode   uint32
		drop   mapload.Cell
	}{
		{"new/80x80", ui.MapModeSingle, mapload.Cell{X: 40, Y: 40}},
		{"new/80x80" + ui.NewMapMultiplayer, ui.MapModeMultiplayer, mapload.Cell{X: 40, Y: 40}},
		{"new/32x48", ui.MapModeSingle, mapload.Cell{X: 16, Y: 24}},
	} {
		d, err := x.Open(c.source)
		if err != nil {
			t.Fatalf("%s: %v", c.source, err)
		}
		s := d.Edits.(*mapInspectionEdits)
		m, err := s.model.Map()
		if err != nil {
			t.Fatal(err)
		}
		if m.Meta.Word70 != c.mode || !slices.Equal(mapload.DropCells(m), []mapload.Cell{c.drop}) {
			t.Errorf("%s: +0x70 %d drops %v, want %d %v", c.source, m.Meta.Word70, mapload.DropCells(m), c.mode, c.drop)
		}
		if s.Dirty() || s.CanUndo() {
			t.Errorf("%s: a new map opens dirty %v or with history %v", c.source, s.Dirty(), s.CanUndo())
		}
		spec, _, err := newMapSpec(c.source)
		if err != nil {
			t.Fatal(err)
		}
		want, err := mapedit.NewMap(spec)
		if err != nil {
			t.Fatal(err)
		}
		if err := want.SetWord70(c.mode); err != nil {
			t.Fatal(err)
		}
		if err := want.SetDropCell(uint32(c.drop.X), uint32(c.drop.Y)); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(s.model.Bytes(), want.Bytes()) {
			t.Errorf("%s: the bytes differ from the original new map with the mode and the drop cell", c.source)
		}
		// The new map has no win: its report names it endless, and only that.
		rows := s.Report()
		if len(rows) != 1 || rows[0].Rule != ui.RuleVictory {
			t.Errorf("%s: report %+v", c.source, rows)
		}
	}
	if _, _, err := newMapSpec("new/80x80/single"); err == nil {
		t.Fatal("an unknown mode suffix was accepted")
	}
}

// The Map panel's facts state the map, and reading them and the report writes
// nothing.
func TestMapFactsStateTheMapAndWriteNothing(t *testing.T) {
	o := cleanOptions()
	o.Name, o.Description = "Bridge", "Hold the bridge."
	lose := scriptNode{label: "Mission failed", opcode: scriptOpLose, id: 50}
	vip := scriptNode{label: "Protect", opcode: scriptCheckVIP, id: 60, values: []uint32{10}, tags: []uint32{scriptUnitParam}}
	o.Type7Payload = scriptPayloadWithDrop([]scriptNode{lose}, []scriptNode{vip})
	m := built(t, o, nil, nil)
	before := m.Bytes()
	s := &mapInspectionEdits{inspector: checkInspector(), model: m, source: "x", saved: before}
	f := s.MapFacts()
	want := ui.InspectionMapFacts{Name: "Bridge", Description: "Hold the bridge.", Players: 1, Sides: 1,
		Drops: []image.Point{{X: 16, Y: 16}}, Wins: []string{"Win"}, Protects: 1}
	if f.Name != want.Name || f.Description != want.Description || f.Mode != 0 || f.Players != 1 || f.Sides != 1 ||
		!slices.Equal(f.Drops, want.Drops) || !slices.Equal(f.Wins, want.Wins) || len(f.Losses) != 0 || f.Protects != 1 {
		t.Fatalf("facts %+v", f)
	}
	_ = s.Report()
	if !bytes.Equal(m.Bytes(), before) || m.CanUndo() || s.Dirty() {
		t.Fatal("the panel's facts or the report wrote the map")
	}
}

// Two players are one side when each names the other an ally.
func TestMapSidesJoinMutualAllies(t *testing.T) {
	g := func(rel ...uint16) alm.Group {
		var x alm.Group
		copy(x.Relation[:], rel)
		return x
	}
	for _, c := range []struct {
		name   string
		groups []alm.Group
		want   int
	}{
		{"none", nil, 0},
		{"two hostile", []alm.Group{g(2, 1), g(1, 2)}, 2},
		{"two allied", []alm.Group{g(2, 2), g(2, 2)}, 1},
		{"one-way ally", []alm.Group{g(2, 2), g(1, 2)}, 2},
		{"a chain of three", []alm.Group{g(2, 2, 1), g(2, 2, 2), g(1, 2, 2)}, 1},
	} {
		if got := mapSides(c.groups); got != c.want {
			t.Errorf("%s: %d sides, want %d", c.name, got, c.want)
		}
	}
}

// The metadata's +0x70 field is Mode: the two values a new map is given by
// name, and another stored value offered as itself and kept.
func TestMapModeFieldNamesTheTwoModesAndKeepsAnotherValue(t *testing.T) {
	_, e := workspaceFixture(t)
	w := e.Document().Edits.(ui.InspectionWorkspace)
	field := func() ui.InspectionField {
		for _, f := range w.WorkspaceFields("metadata") {
			if f.Key == ui.ModeFieldKey {
				return f
			}
		}
		t.Fatal("no mode field")
		return ui.InspectionField{}
	}
	f := field()
	if f.Label != "Mode" || f.Value != 1 || len(f.Choices) != 2 || f.Choices[0] != (ui.InspectionChoice{Value: 1, Label: "Single-player"}) ||
		f.Choices[1] != (ui.InspectionChoice{Value: 4, Label: "Multiplayer"}) {
		t.Fatalf("mode field %+v", f)
	}
	d, err := w.SetWorkspaceField("metadata", ui.ModeFieldKey, "16")
	if err != nil {
		t.Fatal(err)
	}
	if f = field(); f.Value != 16 || len(f.Choices) != 3 || f.Choices[2] != (ui.InspectionChoice{Value: 16, Label: "16"}) {
		t.Fatalf("a stored 16: %+v", f)
	}
	m, _ := alm.Open(d.SourceBytes)
	if m.Meta.Word70 != 16 {
		t.Fatalf("+0x70 %d", m.Meta.Word70)
	}
}
