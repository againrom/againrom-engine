package game

import (
	"slices"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// A stroke beside a cell the author closed (bit 13) keeps it closed. Control:
// the stroke rewrites that cell, and without the flag it would open.
func TestReleaseMapEditorStrokeKeepsTheAuthorsClosedCell(t *testing.T) {
	const flag = 0x2000
	x := installedInspector(t)
	maps, probes, rewritten, wouldOpen := 0, 0, 0, 0
	for _, key := range installedMapKeys(x) {
		d, err := x.Open(key)
		if err != nil {
			t.Fatal(err)
		}
		m, err := alm.Open(d.SourceBytes)
		if err != nil {
			t.Fatal(err)
		}
		w, h := m.Width, m.Height
		var cells []int
		for i, v := range m.Tiles {
			// The left neighbour lies outside the top rows; the 3x3 holds no last cell.
			if cx, cy := i%w, i/w; v&flag != 0 && cx >= 9 && cx < w-2 && cy >= 10 && cy < h-2 {
				cells = append(cells, i)
			}
		}
		if len(cells) == 0 {
			continue
		}
		maps++
		for k := 0; k < 6; k++ {
			i := cells[k*len(cells)/6]
			cx, cy := i%w, i/w
			ed, err := mapedit.New(d.SourceBytes)
			if err != nil {
				t.Fatal(err)
			}
			for _, class := range []uint8{mapedit.ClassGrass, mapedit.ClassWater} {
				if _, err := ed.Paint(mapedit.Brush{Plane: mapedit.PlaneTile, Value: int(class)},
					[]mapedit.Cell{{X: cx - 1, Y: cy}}, mapload.TileClass); err != nil {
					t.Fatalf("%s: %v", key, err)
				}
			}
			out, err := alm.Open(ed.Bytes())
			if err != nil {
				t.Fatalf("%s: %v", key, err)
			}
			probes++
			if out.Tiles[i]&flag == 0 {
				t.Errorf("%s (%d,%d): the stroke beside the closed cell cleared its flag: %#x", key, cx, cy, out.Tiles[i])
				continue
			}
			if !sim.DomainGround.BlockedBy(mapload.Passability(out)[i]) {
				t.Errorf("%s (%d,%d): the closed cell is passable after the stroke", key, cx, cy)
			}
			if out.Tiles[i]&^flag != m.Tiles[i]&^flag {
				rewritten++
				control := *out
				control.Tiles = append([]uint16(nil), out.Tiles...)
				control.Tiles[i] &^= flag
				if !sim.DomainGround.BlockedBy(mapload.Passability(&control)[i]) {
					wouldOpen++
				}
			}
		}
	}
	t.Logf("%d maps with a closed cell, %d strokes, %d rewrote the closed cell, %d of those would open without the flag", maps, probes, rewritten, wouldOpen)
	if maps == 0 || rewritten == 0 || wouldOpen == 0 {
		t.Errorf("the probe measured nothing: %d maps, %d rewrites, %d would open", maps, rewritten, wouldOpen)
	}
}

// The installed maps the drop-cell rule warns on, by catalogue key.
func TestReleaseDropCellRuleWarnsOnTheNamedInstalledMaps(t *testing.T) {
	x := installedInspector(t)
	var got []string
	keys := installedMapKeys(x)
	var want []string
	switch len(keys) {
	case 38: // the EN install
		want = []string{"scenario/50.alm"}
	case 34: // the RU install
		want = []string{"loose/Horror.alm", "scenario/50.alm"}
	default:
		t.Fatalf("%d installed maps: neither the EN install (38) nor the RU install (34)", len(keys))
	}
	for _, key := range keys {
		d, err := x.Open(key)
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range d.Edits.(ui.InspectionChecker).Report() {
			if f.Rule == "drop-cell" {
				got = append(got, key)
			}
		}
	}
	if !slices.Equal(got, want) {
		t.Errorf("drop-cell warns on %q, want %q", got, want)
	}
}
