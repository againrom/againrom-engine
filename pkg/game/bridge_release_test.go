package game

import (
	"image"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/render/terrain"
)

// TestReleaseMission81VerticalWoodenBridgeDrawsItsAuthorSizedDeck reads the
// shipped mission record and shipped bridge1v sheet, then follows the production
// map-to-viewer route. The release gate runs it once per lawful install.
func TestReleaseMission81VerticalWoodenBridgeDrawsItsAuthorSizedDeck(t *testing.T) {
	f := releaseFront(t)
	addr, _ := MissionMap(81)
	raw, err := f.Archives.Containers.ReadFile(addr)
	if err != nil {
		t.Fatalf("read %s: %v", addr, err)
	}
	m, err := alm.Open(raw)
	if err != nil {
		t.Fatalf("decode %s: %v", addr, err)
	}

	var bridge []alm.Object
	for _, o := range m.Objects {
		if o.Kind == 0x21 {
			bridge = append(bridge, o)
		}
	}
	if len(bridge) != 1 {
		t.Fatalf("mission 81 kind-0x21 placements = %d, want the one vertical bridge", len(bridge))
	}
	col, row := terrain.AnchorCell(bridge[0].X, bridge[0].Y)
	if col != 55 || row != 30 || len(bridge[0].Ext) < 8 || bridge[0].Ext[0] != 3 || bridge[0].Ext[4] != 6 {
		t.Fatalf("mission 81 bridge = anchor (%d,%d), ext % x; want (55,30), 3x6", col, row, bridge[0].Ext)
	}
	class := f.Structures.Classes[33]
	if class == nil || !class.Flat || class.VariableLayout != terrain.VariableStructureVerticalNinePatch || len(class.Frames) != 9 {
		t.Fatalf("installed class 33 = %#v with %d frames, want vertical nine-patch with 9",
			class, func() int {
				if class == nil {
					return 0
				}
				return len(class.Frames)
			}())
	}

	grid := terrain.Grid{Width: m.Width, Height: m.Height, Structures: StructureRecords(bridge)}
	places, counts, _ := terrain.StructurePlacements(grid, f.Structures, nil, 0)
	if len(places) != 18 || counts.Placements.Drawn != 1 || counts.Placements.VariableSize != 0 {
		t.Fatalf("bridge placement = %d entries, counts %+v; want 18 and one drawn", len(places), counts)
	}
	wantFrames := [][]int{{2, 1, 0}, {5, 4, 3}, {5, 4, 3}, {5, 4, 3}, {5, 4, 3}, {8, 7, 6}}
	n := 0
	for dy, rowFrames := range wantFrames {
		for dx, frame := range rowFrames {
			wantCell := image.Point{X: 57 - dx, Y: 30 + dy}
			if places[n].Cell != wantCell || places[n].GridIndex != frame || places[n].Frame != class.Frames[frame] {
				t.Fatalf("bridge entry %d = cell %v frame %d; want %v frame %d",
					n, places[n].Cell, places[n].GridIndex, wantCell, frame)
			}
			n++
		}
	}

	mv, err := LoadMapViewer(f.Tiles, raw, addr, Markers{}, StaticLayer{Set: f.Statics, Art: true},
		StructureLayer{Set: f.Structures, Art: true})
	if err != nil {
		t.Fatalf("production LoadMapViewer(%s): %v", addr, err)
	}
	entries, full := mv.Viewer.Structures()
	if entries < len(places) || full.Placements.Drawn+full.Placements.NoClass+
		full.Placements.Undrawable+full.Placements.VariableSize != len(m.Objects) {
		t.Fatalf("production viewer entries/counts = %d/%+v over %d records", entries, full, len(m.Objects))
	}
}
