package game

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// forestScenario is the shipped map the Forest release tests plant on.
const forestScenario = "scenario/10.alm"

// forestTrees are the class codes the release tests tick: Pine 1,
// Vallenwood 1, Maple 1 and Oak 1 of the installed object registry.
var forestTrees = []int{1, 4, 16, 19}

// forestGridIndex is the forest grid cell of class code: the object list's
// row less the clear row.
func forestGridIndex(t *testing.T, x *MapInspector, code int) int {
	t.Helper()
	n := slices.IndexFunc(x.objectRows, func(r ui.BrushObject) bool { return r.Code == code })
	if n < 1 {
		t.Fatalf("the installed object list does not offer class %d", code)
	}
	return n - 1
}

// installedForest opens key at 1280x800, closes the Map panel, turns the
// object brush to Forest, ticks forestTrees and sets a square brush of
// radius r.
func installedForest(t *testing.T, x *MapInspector, key string, r int) (*ui.MapEditor, *editorClicker) {
	t.Helper()
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open(key); err != nil {
		t.Fatalf("%s: %v", key, err)
	}
	if e.PanelOpen() {
		closeMapPanel(t, e)
	}
	c := &editorClicker{t: t, now: time.Unix(50, 0)}
	if err := e.Key("objects"); err != nil {
		t.Fatal(err)
	}
	rect, ok := e.ListButtonRect(2, 1)
	c.click(e, rect, ok, "the Forest button")
	for _, code := range forestTrees {
		rect, ok = e.PlaceGridCellRect(forestGridIndex(t, x, code))
		c.click(e, rect, ok, fmt.Sprintf("the grid cell of class %d", code))
	}
	rect, ok = e.ListButtonRect(1, 1)
	c.click(e, rect, ok, "the Square button")
	for i := 0; i < 20; i++ {
		_ = e.Key("radiusdown")
	}
	for i := 0; i < r; i++ {
		_ = e.Key("radiusup")
	}
	return e, c
}

// forestCellRule is the test's reading of the cells a forest may take on m:
// an empty type-3 cell outside the water index range, which the classifier
// does not call Road and no structure footprint attaches.
func forestCellRule(m *alm.Map, table *mapload.Table) func(i int) bool {
	fenced := mapload.StructureCells(m, table)
	return func(i int) bool {
		w := m.Tiles[i]
		return m.Overlay[i] == 0 && alm.TileIndex(w)&0x300 != 0x200 && mapload.TileClass(w) != 10 && (fenced == nil || !fenced[i])
	}
}

// forestCentre is an interior cell of m whose radius-r square holds water or
// road, a structure cell and the most open cells, so one stroke shows both
// planting and refusal.
func forestCentre(t *testing.T, m *alm.Map, table *mapload.Table, r int) image.Point {
	t.Helper()
	open := forestCellRule(m, table)
	fenced := mapload.StructureCells(m, table)
	best, score := image.Point{}, -1
	for y := 12 + r; y < m.Height-12-r; y += 2 {
		for x := 12 + r; x < m.Width-12-r; x += 2 {
			n, wet, built := 0, false, false
			for dy := -r; dy <= r; dy++ {
				for dx := -r; dx <= r; dx++ {
					i := (y+dy)*m.Width + x + dx
					if open(i) {
						n++
					}
					w := m.Tiles[i]
					wet = wet || alm.TileIndex(w)&0x300 == 0x200 || mapload.TileClass(w) == 10
					built = built || fenced != nil && fenced[i]
				}
			}
			if wet && built && n > score {
				best, score = image.Pt(x, y), n
			}
		}
	}
	if score < 0 {
		t.Fatal("no cell of the map has water or road and a structure within the brush")
	}
	return best
}

// forestPress drags the brush once at centre, left to plant or right to
// thin, with the view placed so the popup stands clear of the centre.
func forestPress(e *ui.MapEditor, centre image.Point, right bool) {
	e.ShowCell(centre.Sub(image.Pt(8, 0)))
	p := e.CellPoint(centre)
	now := time.Unix(60, 0)
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: !right, SecondaryDown: right}, now)
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, now)
}

// TestReleaseMapEditorForestStrokeSavesAMapTheGameReads plants a dense
// forest of four installed trees on a shipped map through the window. Every
// planted cell was open by the test's own reading of water, road, structure
// and standing objects; no standing object changes. The map saved through
// Save As reopens in the editor with the same bytes, and the game's reader
// gives the planted cells the trees and a world over it builds with each
// planted cell closed to ground movers. One Undo restores the shipped bytes.
func TestReleaseMapEditorForestStrokeSavesAMapTheGameReads(t *testing.T) {
	forestSeed(t, 21)
	x := installedInspector(t)
	raw, err := x.archives.Containers.ReadFile(forestScenario)
	if err != nil {
		t.Fatal(err)
	}
	before, err := alm.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	const radius = 6
	centre := forestCentre(t, before, x.table, radius)
	e, _ := installedForest(t, x, forestScenario, radius)
	r, ok := e.ListButtonRect(4, 2)
	(&editorClicker{t: t, now: time.Unix(55, 0)}).click(e, r, ok, "Dense")
	forestPress(e, centre, false)
	if !e.Dirty() {
		t.Fatalf("the stroke at %v planted nothing (%q)", centre, e.Message())
	}
	open := forestCellRule(before, x.table)
	planted := 0
	after, err := alm.Open(e.Document().SourceBytes)
	if err != nil {
		t.Fatal(err)
	}
	for i := range after.Overlay {
		if after.Overlay[i] == before.Overlay[i] {
			continue
		}
		c := image.Pt(i%before.Width, i/before.Width)
		if d := c.Sub(centre); max(d.X, -d.X, d.Y, -d.Y) > radius {
			t.Fatalf("cell %v outside the brush changed", c)
		}
		if !open(i) {
			t.Fatalf("cell %v (tile %#x, object %d) was not open and took %d", c, before.Tiles[i], before.Overlay[i], after.Overlay[i])
		}
		if !slices.Contains(forestTrees, int(after.Overlay[i])) {
			t.Fatalf("cell %v took %d, which is not ticked", c, after.Overlay[i])
		}
		planted++
	}
	t.Logf("%s: %d trees planted around %v", forestScenario, planted, centre)

	target := filepath.Join(t.TempDir(), "forest.alm")
	_ = e.Key("saveas")
	e.TextInput(target)
	if err := e.Key("enter"); err != nil {
		t.Fatalf("Save As: %v", err)
	}
	saved, err := os.ReadFile(target)
	if err != nil || !bytes.Equal(saved, e.Document().SourceBytes) {
		t.Fatalf("the saved file is not the edited map: %v", err)
	}
	again := x.Editor()
	if err := again.Open(target); err != nil || !bytes.Equal(again.Document().SourceBytes, saved) {
		t.Fatalf("the saved map does not reopen in the editor: %v", err)
	}
	back, err := alm.Open(saved)
	if err != nil {
		t.Fatalf("the game's reader refuses the saved map: %v", err)
	}
	if !bytes.Equal(back.Overlay, after.Overlay) {
		t.Fatal("the game's reader gives other object cells than the editor wrote")
	}
	if _, err := mapload.FromALMWith(back, x.table, mapload.DifficultyNormal); err != nil {
		t.Fatalf("no world builds over the saved map: %v", err)
	}
	plane := mapload.PassabilityWith(back, x.table)
	for i := range back.Overlay {
		if back.Overlay[i] != before.Overlay[i] && plane[i]&1 == 0 {
			t.Fatalf("planted cell %d is open to ground movers in the game's plane", i)
		}
	}
	if err := e.Key("undo"); err != nil || !bytes.Equal(e.Document().SourceBytes, raw) {
		t.Fatalf("one Undo did not restore the shipped bytes: %v", err)
	}
}

// TestReleaseMapEditorForestWithoutAStrokeSavesUnchanged opens four shipped
// maps, turns Forest on, ticks kinds, presses every density and the typed
// per cent, and saves without a stroke: each saved file keeps the installed
// bytes.
func TestReleaseMapEditorForestWithoutAStrokeSavesUnchanged(t *testing.T) {
	x := installedInspector(t)
	dir := t.TempDir()
	keys := installedMapKeys(x)
	for n, key := range []string{keys[0], forestScenario, keys[len(keys)/2], keys[len(keys)-1]} {
		want, err := x.Open(key)
		if err != nil {
			t.Fatal(err)
		}
		e, c := installedForest(t, x, key, 4)
		for k := 0; k < 4; k++ {
			r, ok := e.ListButtonRect(4, k)
			c.click(e, r, ok, "a density button")
		}
		r, ok := e.ListButtonRect(3, 0)
		c.click(e, r, ok, "the per cent button")
		_ = e.Key("clear")
		e.TextInput("40")
		if err := e.Key("enter"); err != nil {
			t.Fatal(err)
		}
		if e.Dirty() {
			t.Fatalf("%s: setting the forest up edited the map", key)
		}
		target := filepath.Join(dir, fmt.Sprintf("m%d.alm", n))
		_ = e.Key("saveas")
		e.TextInput(target)
		if err := e.Key("enter"); err != nil {
			t.Fatalf("%s: Save As: %v", key, err)
		}
		got, err := os.ReadFile(target)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want.SourceBytes) {
			t.Errorf("%s: saved %d bytes differ from the installed %d", key, len(got), len(want.SourceBytes))
		}
	}
}

// TestReleaseMapEditorForestFrames writes the owner's frames into
// AGAINROM_EDITOR_FRAMES, rendered headless from the install: the forest
// grid with four ticked trees and the density row, a map area before a
// stroke, after a dense stroke and after a right-button thinning. Without
// the variable the same steps run and nothing is written.
func TestReleaseMapEditorForestFrames(t *testing.T) {
	forestSeed(t, 21)
	x := installedInspector(t)
	dir, lang := os.Getenv("AGAINROM_EDITOR_FRAMES"), installLanguage(x)
	frame := func(e *ui.MapEditor, name string) {
		writeEditorFrame(t, e, filepath.Join(dir, "forest-"+name+"-"+lang+".png"), dir != "", image.Rectangle{})
	}
	raw, err := x.archives.Containers.ReadFile(forestScenario)
	if err != nil {
		t.Fatal(err)
	}
	m, err := alm.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	const radius = 6
	centre := forestCentre(t, m, x.table, radius)
	e, c := installedForest(t, x, forestScenario, radius)
	r, ok := e.ListButtonRect(4, 2)
	c.click(e, r, ok, "Dense")
	e.ShowCell(centre.Sub(image.Pt(8, 0)))
	frame(e, "grid")
	p := e.CellPoint(centre)
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, time.Unix(59, 0))
	frame(e, "before")
	forestPress(e, centre, false)
	if !e.Dirty() {
		t.Fatalf("the stroke planted nothing (%q)", e.Message())
	}
	frame(e, "after")
	planted := e.Document().SourceBytes
	forestPress(e, centre, true)
	if bytes.Equal(e.Document().SourceBytes, planted) {
		t.Fatal("the right button thinned nothing")
	}
	frame(e, "thinned")
}
