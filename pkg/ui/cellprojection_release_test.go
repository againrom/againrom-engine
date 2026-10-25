package ui

import (
	"image"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/alm"
	"againrom/pkg/render/terrain"
	"againrom/pkg/vfs"
)

// TestReleaseEditorCrossesStandOnThePathCentreOfAnInstalledSlope runs the
// same-height proof on the installed map scenario/20.alm: on the interior
// cell where the former midpoint rule and the anchor height differ most, the
// editor's unit marker, footprint cross and area centre are the game's
// path-line centre, and the midpoint rule is not.
func TestReleaseEditorCrossesStandOnThePathCentreOfAnInstalledSlope(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: the slope witness requires an installed map")
	}
	source, err := vfs.Open([]string{filepath.Join(root, "scenario.res"), filepath.Join(root, "graphics.res")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := source.ReadFile("scenario/20.alm")
	if err != nil {
		t.Fatal(err)
	}
	m, err := alm.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	g := terrain.Grid{Width: m.Width, Height: m.Height, Tiles: terrain.RenderTileWords(m.Tiles), Altitudes: m.Altitudes}
	p := terrain.Project(m.Altitudes, m.Width, m.Height)
	best, gap := image.Point{}, 0
	for row := 4; row < m.Height-4; row++ {
		for col := 4; col < m.Width-4; col++ {
			_, top := p.WorldCorner(col, row)
			_, bottom := p.WorldCorner(col+1, row+1)
			anchor := row*terrain.CellSize + terrain.CellSize/2 - p.AnchorHeight(col, row) - p.MinV
			if d := max((top+bottom)/2-anchor, anchor-(top+bottom)/2); d > gap {
				best, gap = image.Pt(col, row), d
			}
		}
	}
	if gap < 2 {
		t.Fatalf("no interior cell of scenario/20.alm separates the two centre rules (largest gap %d)", gap)
	}
	t.Logf("cell %v: anchor centre and corner midpoint %d native pixels apart", best, gap)
	v, err := NewViewer("installed slope", g, terrain.LoadTileset(source))
	if err != nil {
		t.Fatal(err)
	}
	checkCellCentresAgree(t, v, m.Width, m.Height, best)
}
