package game

import (
	"bytes"
	"image"
	"image/color"
	"io/fs"
	"reflect"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

type structurePreviewArchive map[string][]byte

func (a structurePreviewArchive) ReadFile(path string) ([]byte, error) {
	if b, ok := a[path]; ok {
		return b, nil
	}
	return nil, fs.ErrNotExist
}

func TestMapInspectionStructureGridVariableBridgeNeedsLoadedArt(t *testing.T) {
	var classes [][]synth.RegNode
	for _, id := range []int32{33, 37} {
		file := "bridge"
		if id == 37 {
			file = "other"
		}
		classes = append(classes, []synth.RegNode{
			{Name: "ID", Kind: 2, Int: id},
			{Name: "File", Kind: 0, Str: file},
			{Name: "Picture", Kind: 0, Str: "beeh"},
			{Name: "TileWidth", Kind: 2, Int: 3},
			{Name: "TileHeight", Kind: 2, Int: 3},
			{Name: "FullHeight", Kind: 2, Int: 3},
			{Name: "VariableSize", Kind: 2, Int: 1},
		})
	}
	for _, count := range []int{1, 5, 11} {
		sheet := func(frames int) []byte {
			return synth.StructureSheet(synth.StructureSheetOptions{
				Frames: frames, Width: 32, Height: 32,
				Palette: []color.RGBA{{A: 255}, {G: 100, A: 255}, {B: 100, A: 255}},
				Ink:     func(frame, _, _ int) (uint8, bool) { return uint8(frame%2 + 1), true },
			})
		}
		art := structurePreviewArchive{
			StructureRegistry:                synth.StructuresReg(classes...),
			"graphics/structures/bridge.256": sheet(9),
			"graphics/structures/other.256":  sheet(count),
		}
		set, err := LoadStructures(art)
		if err != nil {
			t.Fatal(err)
		}
		if c := set.Classes[33]; c == nil || len(c.Frames) != 9 || c.VariableLayout != terrain.VariableStructureVerticalNinePatch {
			t.Fatal("fixture did not load the supported variable bridge")
		}
		c := set.Classes[37]
		if c == nil || c.Frames != nil || c.VariableLayout != terrain.VariableStructureUnsupported {
			t.Fatalf("%d-frame sheet enabled the unsupported map layout: %+v", count, c)
		}
		g := terrain.Grid{Width: 8, Height: 8, Structures: []terrain.StructureRecord{{ID: 37, Key: 37, X: 256, Y: 256, VariableWidth: 3, VariableHeight: 4}}}
		places, census, animated := terrain.StructurePlacements(g, set, nil, 0)
		want := terrain.StructureCounts{Placements: terrain.StructurePlacementCounts{VariableSize: 1}}
		if len(places) != 0 || len(animated) != 0 || census != want {
			t.Fatalf("%d-frame sheet changed the placed map/census: %d entries, %+v", count, len(places), census)
		}
		c.Portrait = structurePreviewFrame(color.RGBA{R: 190, A: 255}).RGBA()
		s := &mapInspectionEdits{inspector: &MapInspector{structures: set}}
		got := s.NewRecordPreview(ui.KindStructure, "kind-grid", 37)
		if got == nil || got.Bounds().Size() != image.Pt(32, 32) || got.RGBAAt(16, 16) != (color.RGBA{G: 100, A: 255}) {
			t.Errorf("%d-frame sheet: grid does not show its own frame 0", count)
		}
		if s.NewRecordPreview(ui.KindStructure, "kind", 37) != c.Portrait {
			t.Error("the separate Add window changed its picture")
		}
		c.PreviewFrame = nil
		if s.NewRecordPreview(ui.KindStructure, "kind-grid", 37) != nil {
			t.Error("an absent sheet frame fell back to the shared picture")
		}
	}
}

func structurePreviewFrame(ink color.RGBA) *terrain.StaticFrame {
	f := &terrain.StaticFrame{Width: 32, Height: 32, Pixels: make([]terrain.StaticPixel, 32*32)}
	f.Palette[1] = ink
	for i := range f.Pixels {
		f.Pixels[i] = terrain.StaticPixel{Index: 1, Opaque: true}
	}
	return f
}

func TestMapInspectionStructureGridSharedPictureUsesLoadedSprites(t *testing.T) {
	hive := color.RGBA{R: 190, A: 255}
	portrait := structurePreviewFrame(hive).RGBA()
	set := new(terrain.StructureSet)
	s := &mapInspectionEdits{inspector: &MapInspector{structures: set}}
	for _, id := range []int{24, 25, 33, 37, 38, 39, 40, 41, 42, 48, 52, 53} {
		ink := color.RGBA{G: uint8(id), A: 255}
		if id == 24 || id == 25 {
			ink = hive
		}
		c := &terrain.StructureClass{ID: int32(id), Picture: "beeh", Portrait: portrait,
			TileWidth: 2, TileHeight: 1, FullHeight: 2, PreviewFrame: structurePreviewFrame(hive)}
		for range 4 {
			c.Frames = append(c.Frames, structurePreviewFrame(ink))
		}
		set.Classes[id] = c
		got := s.NewRecordPreview(ui.KindStructure, "kind-grid", int64(id))
		if got == nil || got.Bounds().Size() != image.Pt(64, 64) || got.RGBAAt(16, 16) != ink {
			t.Errorf("class %d: grid does not draw its complete loaded sprite", id)
		}
		if got := s.NewRecordPreview(ui.KindStructure, "kind", int64(id)); got != portrait {
			t.Errorf("class %d: separate Add preview changed", id)
		}
	}
	set.Classes[7] = &terrain.StructureClass{Picture: "tower", Portrait: portrait}
	if s.NewRecordPreview(ui.KindStructure, "kind-grid", 7) != portrait {
		t.Error("an unrelated grid picture changed")
	}
}

func TestMapInspectionStructureGridPlacementWordAndRuinDraw(t *testing.T) {
	_, e := openRecords(t)
	s := e.Document().Edits.(*mapInspectionEdits)
	s.inspector.table.Buildings.(dbCollection)[5] = dbEntry{name: "house", params: []int32{3, 2, 0, 300, -1, -1, 0, 0}}
	c := s.inspector.structures.Classes[5]
	for range 12 {
		c.Frames = append(c.Frames, structurePreviewFrame(color.RGBA{G: 100, A: 255}))
	}
	before, _ := s.model.Map()
	first := before.Objects[0]
	r := ui.InspectionNewRecord{Kind: ui.KindStructure, HasCell: true, Cell: image.Pt(18, 10),
		Values: []ui.InspectionFieldValue{{Key: "kind", Value: 5}, {Key: "owner", Value: 1}}}
	standing, err := s.PreviewPlacement(r)
	if err != nil {
		t.Fatal(err)
	}
	r.Values = append(r.Values, ui.InspectionFieldValue{Key: "placement", Value: 0})
	ruined, err := s.PreviewPlacement(r)
	if err != nil {
		t.Fatal(err)
	}
	want := bytes.Clone(standing.SourceBytes)
	diffs := 0
	for i := range want {
		if want[i] != ruined.SourceBytes[i] {
			diffs++
			if want[i] != 1 || ruined.SourceBytes[i] != 0 {
				t.Fatalf("unexpected byte change at %d", i)
			}
		}
	}
	if diffs != 1 {
		t.Fatalf("ruined copy changes %d bytes, want only placement 1 to 0", diffs)
	}
	d, err := s.AddRecord(r)
	if err != nil {
		t.Fatal(err)
	}
	view, _ := s.model.Map()
	if !reflect.DeepEqual(view.Objects[0], first) || view.Objects[2].Field0C != 0 {
		t.Fatal("Add changed an earlier record or left the copy standing")
	}
	if state := mapload.Structures(view, s.inspector.table)[2]; state.Field42 != 0 || state.MaxHealth != 300 {
		t.Fatalf("placed copy loads as %+v", state)
	}
	if entries, ruins := d.Viewer.StructureRuinFrames(2); entries != 6 || ruins != entries {
		t.Fatalf("editor draws %d/%d ruin frames, want 6/6", ruins, entries)
	}
	r.Values = r.Values[:2]
	r.Cell = image.Pt(18, 16)
	if _, err := s.AddRecord(r); err != nil {
		t.Fatal(err)
	}
	view, _ = s.model.Map()
	if view.Objects[2].Field0C != 0 || view.Objects[3].Field0C != 1 || mapload.Structures(view, s.inspector.table)[3].Field42 != 300 {
		t.Fatal("switching off changed an older copy or did not restore standing placement")
	}
}
