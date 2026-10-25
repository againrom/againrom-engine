package game

import (
	"bytes"
	"image"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// terrainFrame writes the owner's frame name of the window into
// AGAINROM_EDITOR_FRAMES, prefixed with the install's language, when set;
// without it the frame is drawn and checked, not written.
func terrainFrame(t *testing.T, x *MapInspector, e *ui.MapEditor, name string) {
	t.Helper()
	dir := os.Getenv("AGAINROM_EDITOR_FRAMES")
	writeEditorFrame(t, e, filepath.Join(dir, installLanguage(x)+"-"+name+".png"), dir != "", image.Rectangle{})
}

// runSeeded runs the whole-map command key through its seed prompt with seed
// typed in, and returns the map's bytes afterwards.
func runSeeded(t *testing.T, e *ui.MapEditor, key, seed string) []byte {
	t.Helper()
	if err := e.Key(key); err != nil {
		t.Fatal(err)
	}
	_ = e.Key("clear")
	e.TextInput(seed)
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	if e.Message() != "" {
		t.Fatalf("%s seed %s: %s", key, seed, e.Message())
	}
	return slices.Clone(e.Document().SourceBytes)
}

// without is m with the planes named cleared, for comparing what is left.
func without(t *testing.T, data []byte, tiles, heights, objects bool) *alm.Map {
	t.Helper()
	m, err := alm.Open(data)
	if err != nil {
		t.Fatal(err)
	}
	if tiles {
		m.Tiles = nil
	}
	if heights {
		m.Altitudes = nil
	}
	if objects {
		m.Overlay = nil
	}
	return m
}

// playFirstTicks builds the game's world from the map's bytes and plays its
// first 40 ticks headless.
func playFirstTicks(t *testing.T, x *MapInspector, data []byte, what string) {
	t.Helper()
	m, err := alm.Open(data)
	if err != nil {
		t.Fatalf("%s: %v", what, err)
	}
	w, err := mapload.FromALMWith(m, x.table, mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("%s: the game refuses the map: %v", what, err)
	}
	for range 40 {
		sim.Step(w, nil)
	}
	_ = w.Hash()
}

// On the installed map, through the window's keys: random terrain and random
// heights each follow their seed (one seed twice gives one map, another seed
// another), write only their own plane, leave every unit, structure, tree
// and record byte as it was, are one undo step, and give a map the game
// builds and plays.
func TestReleaseMapEditorWholeMapCommandsFollowTheSeed(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	for _, c := range []struct {
		key                     string
		tiles, heights, objects bool
	}{{"randomterrain", true, false, false}, {"randomheights", false, true, false}} {
		e := openBrushEditor(t, x, "scenario/10.alm")
		before := slices.Clone(e.Document().SourceBytes)
		_ = e.Key(c.key)
		terrainFrame(t, x, e, c.key+"-seed")
		_ = e.Key("escape")
		a := runSeeded(t, e, c.key, "1")
		_ = e.Key("undo")
		if !bytes.Equal(e.Document().SourceBytes, before) {
			t.Fatalf("%s: one undo did not give back the map", c.key)
		}
		b := runSeeded(t, e, c.key, "1")
		_ = e.Key("undo")
		other := runSeeded(t, e, c.key, "2")
		if bytes.Equal(a, before) || !bytes.Equal(a, b) || bytes.Equal(a, other) {
			t.Fatalf("%s: changed %v, seed 1 twice equal %v, seeds 1 and 2 equal %v", c.key, !bytes.Equal(a, before), bytes.Equal(a, b), bytes.Equal(a, other))
		}
		if !reflect.DeepEqual(without(t, before, c.tiles, c.heights, false), without(t, a, c.tiles, c.heights, false)) {
			t.Fatalf("%s wrote outside its plane", c.key)
		}
		terrainFrame(t, x, e, c.key)
		playFirstTicks(t, x, a, c.key)
	}
}

// boxCells drags a plain selection box over the cell rectangle box.
func boxCells(e *ui.MapEditor, box image.Rectangle) {
	now := time.Unix(1, 0)
	e.ShowCell(image.Pt((box.Min.X+box.Max.X)/2, (box.Min.Y+box.Max.Y)/2))
	a, b := e.CellPoint(box.Min), e.CellPoint(box.Max.Sub(image.Pt(1, 1)))
	e.Pointer(ui.Input{CursorX: a.X, CursorY: a.Y, PrimaryDown: true}, now)
	e.Pointer(ui.Input{CursorX: b.X, CursorY: b.Y, PrimaryDown: true}, now)
	e.Pointer(ui.Input{CursorX: b.X, CursorY: b.Y}, now)
}

// hoverPaste puts the pointer on cell and presses Ctrl+V.
func hoverPaste(t *testing.T, e *ui.MapEditor, cell image.Point) {
	t.Helper()
	e.ShowCell(cell)
	p := e.CellPoint(cell)
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, time.Unix(2, 0))
	if err := e.Key("paste"); err != nil {
		t.Fatal(err)
	}
}

// treeBox is an 8 x 6 rectangle of the map below its top eight rows with at
// least four object cells, and a target of that size elsewhere whose tiles,
// heights and objects differ from it.
func treeBox(t *testing.T, m *alm.Map) (box image.Rectangle, target image.Point) {
	t.Helper()
	const w, h = 8, 6
	trees := func(r image.Rectangle) int {
		n := 0
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				if m.Overlay[y*m.Width+x] != 0 {
					n++
				}
			}
		}
		return n
	}
	for y := 10; y+h < m.Height-2; y += 3 {
		for x := 2; x+w < m.Width-2; x += 3 {
			if r := image.Rect(x, y, x+w, y+h); trees(r) >= 4 {
				for _, to := range []image.Point{{m.Width - w - 12, m.Height - h - 12}, {12, m.Height - h - 12}, {m.Width - w - 12, 12}, {12, 12}} {
					there := image.Rectangle{Min: to, Max: to.Add(image.Pt(w, h))}
					if !there.Overlaps(r) && trees(there) < trees(r) {
						return r, to
					}
				}
			}
		}
	}
	t.Fatal("no rectangle of the map holds four object cells")
	return
}

// samePatch reports whether the cells of now at to show what the cells of
// src at box show, as far as the map reaches.
func samePatch(src, now *alm.Map, box image.Rectangle, to image.Point) (cells int, ok bool) {
	for y := 0; y < box.Dy(); y++ {
		for x := 0; x < box.Dx(); x++ {
			tx, ty := to.X+x, to.Y+y
			if tx >= now.Width || ty >= now.Height {
				continue
			}
			a, b := (box.Min.Y+y)*src.Width+box.Min.X+x, ty*now.Width+tx
			if ty >= 8 && now.Tiles[b] != src.Tiles[a] || now.Altitudes[b] != src.Altitudes[a] || now.Overlay[b] != src.Overlay[a] {
				return cells, false
			}
			cells++
		}
	}
	return cells, true
}

// On the installed map, through the window: a selection box over trees and
// Ctrl+C copy its tiles, heights and objects; Ctrl+V with the pointer on a
// cell writes them there, and at the map's edge the patch is cut; each paste
// is one undo step, the records outside the box stay, and the game builds
// and plays the pasted map.
func TestReleaseMapEditorTerrainPatchPastesTerrainHeightsAndTrees(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	e := openBrushEditor(t, x, "scenario/10.alm")
	before := slices.Clone(e.Document().SourceBytes)
	src := shownMap(t, e)
	box, target := treeBox(t, src)
	if _, ok := samePatch(src, src, box, target); ok {
		t.Fatal("the target already shows the patch; the control proves nothing")
	}
	boxCells(e, box)
	if err := e.Key("copy"); err != nil {
		t.Fatal(err)
	}
	terrainFrame(t, x, e, "patch-box")
	edge := image.Pt(src.Width-3, src.Height-2)
	for _, to := range []image.Point{target, edge} {
		hoverPaste(t, e, to)
		now := shownMap(t, e)
		cells, ok := samePatch(src, now, box, to)
		if !ok || cells == 0 || to == edge && cells != 3*2 {
			t.Fatalf("paste at %v: %d cells match the patch (%v)", to, cells, ok)
		}
		if len(now.Units) < len(src.Units) || len(now.Objects) < len(src.Objects) {
			t.Fatalf("paste at %v lost records", to)
		}
		if to == target {
			terrainFrame(t, x, e, "patch-pasted")
			playFirstTicks(t, x, e.Document().SourceBytes, "the pasted map")
		}
		_ = e.Key("undo")
		if !bytes.Equal(e.Document().SourceBytes, before) {
			t.Fatalf("paste at %v: one undo did not give back the map", to)
		}
	}
}

// On the installed tables, through the window: the new-map question holds 64;
// a typed starting height opens a map whose every height cell holds it, the
// game builds and plays it, and 127 is refused with the question kept.
func TestReleaseMapEditorNewMapStartingHeight(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	for _, c := range []struct {
		typed string
		want  uint8
	}{{"", 64}, {"0", 0}, {"126", 126}, {"127", 0}} {
		e := x.Editor()
		e.Layout(1280, 800)
		if err := e.Key("new"); err != nil {
			t.Fatal(err)
		}
		_ = e.Key("clear")
		e.TextInput("32x32")
		_ = e.Key("enter")
		if c.typed != "" {
			_ = e.Key("clear")
			e.TextInput(c.typed)
		}
		if c.typed == "126" {
			terrainFrame(t, x, e, "newmap-height")
		}
		_ = e.Key("enter")
		if c.typed == "127" {
			if e.Document() != nil || e.Message() == "" {
				t.Fatal("height 127 opened a map")
			}
			terrainFrame(t, x, e, "newmap-height-refused")
			continue
		}
		m := shownMap(t, e)
		for i, h := range m.Altitudes {
			if h != c.want {
				t.Fatalf("height %q: cell %d holds %d", c.typed, i, h)
			}
		}
		playFirstTicks(t, x, e.Document().SourceBytes, "new map at "+c.typed)
	}
}

// The height numbers on the installed map, through the View menu's key: off
// at first, then each shows the height of its corner minus 64, and after a
// random-heights edit they show the new heights. Frames of both states for
// the owner.
func TestReleaseMapEditorHeightNumbers(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	e := openBrushEditor(t, x, "scenario/10.alm")
	e.ShowCell(image.Pt(20, 20))
	if rects, _ := e.HeightLabels(); len(rects) != 0 {
		t.Fatal("height numbers with the switch off")
	}
	_ = e.Key("heightlabels")
	cam := e.Document().Viewer.Camera()
	cam.SetZoom(2)
	e.ShowCell(image.Pt(20, 20))
	_, before := e.HeightLabels()
	if len(before) == 0 {
		t.Fatal("no height numbers zoomed in")
	}
	terrainFrame(t, x, e, "heights-on")
	runSeeded(t, e, "randomheights", "5")
	_, after := e.HeightLabels()
	if len(after) == 0 || slices.Equal(before, after) {
		t.Fatalf("the numbers did not follow the edit: %d before, %d after", len(before), len(after))
	}
	terrainFrame(t, x, e, "heights-after-random")
	_ = e.Key("heightlabels")
	if rects, _ := e.HeightLabels(); len(rects) != 0 {
		t.Fatal("V did not turn the numbers off")
	}
}
