package ui

import (
	"flag"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/render/terrain"
)

var updateLightPlane = flag.Bool("update-light-plane", false,
	"rewrite testdata/lightplane.golden from the current code")

// lightFrame is one frame's light sources: the cell-bit rows and the stamps.
type lightFrame struct {
	name   string
	cells  []SpellLightCell
	stamps []LightStamp
}

// setLightFrame hands the viewer one frame's light sources.
func setLightFrame(v *Viewer, f lightFrame) {
	v.SetLightSources(LightSources(f.cells, f.stamps))
}

// cellCorners is the four light stamps of a cell's corners at one level.
func cellCorners(cell image.Point, level uint8, point bool) []LightStamp {
	var out []LightStamp
	for _, c := range [4]image.Point{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
		out = append(out, LightStamp{Vertex: cell.Add(c), Level: level, Point: point})
	}
	return out
}

// lightFrames are the fixture frames: cell sources only, stamps only, both
// overlapping, a bolt's path stamps under a later point stamp, and none.
// Repeated places pin each source kind's merge: the cell plane's maximum for
// a unit level, later-overwrites for a vertex.
func lightFrames() []lightFrame {
	light := func(x, y int) SpellLightCell { return SpellLightCell{Cell: image.Pt(x, y), Terrain: 3, Sprite: 2} }
	dark := func(x, y int) SpellLightCell { return SpellLightCell{Cell: image.Pt(x, y), Terrain: 0.5, Sprite: 0.5} }
	both := cellCorners(image.Pt(1, 1), 40, true)
	both = append(both, LightStamp{Vertex: image.Pt(2, 3), Level: 0}, LightStamp{Vertex: image.Pt(1, 2), Level: 70, Point: true})
	bolt := cellCorners(image.Pt(1, 2), 0, false)
	bolt = append(bolt, cellCorners(image.Pt(0, 1), 16, false)...)
	bolt = append(bolt, LightStamp{Vertex: image.Pt(1, 2), Level: 50, Point: true}, LightStamp{Vertex: image.Pt(3, 4), Level: 8, Point: true})
	return []lightFrame{
		{name: "cells", cells: []SpellLightCell{
			light(1, 1), dark(2, 1), dark(1, 1),
			{Cell: image.Pt(0, 3), Sprite: 2}, {Cell: image.Pt(2, 3), Terrain: 3},
			dark(0, 0), light(0, 0),
		}},
		{name: "stamps", stamps: []LightStamp{
			{Vertex: image.Pt(1, 1), Level: 0, Point: true}, {Vertex: image.Pt(1, 1), Level: 40},
			{Vertex: image.Pt(2, 2), Level: 10, Point: true}, {Vertex: image.Pt(3, 4), Level: 0},
			{Vertex: image.Pt(0, 0), Level: 60}, {Vertex: image.Pt(0, 0), Level: 5, Point: true},
			{Vertex: image.Pt(2, 1), Level: 20},
		}},
		{name: "both", cells: []SpellLightCell{light(1, 1), dark(1, 2), light(2, 3), {Cell: image.Pt(0, 0), Sprite: 2}}, stamps: both},
		{name: "bolt", cells: []SpellLightCell{dark(1, 2), light(0, 1)}, stamps: bolt},
		{name: "none"},
	}
}

// lightOption is one viewer state a frame is merged in.
type lightOption struct {
	name     string
	ambient  int // 0 keeps the viewer's own sun
	off      bool
	unshaded bool
}

var lightOptions = []lightOption{
	{name: "own-sun"},
	{name: "ambient-48", ambient: 48},
	{name: "ambient-120", ambient: 120},
	{name: "ambient-48-dynamic-off", ambient: 48, off: true},
	{name: "ambient-48-unshaded", ambient: 48, unshaded: true},
}

func lightFloat(f float32) string { return strconv.FormatFloat(float64(f), 'g', -1, 32) }

// TestTheMergedLightOfEachFrameIsUnchanged records cornerScales at every
// corner and spellSpriteFactor at every cell of each fixture frame, on a lit
// map and on one whose left column is the black margin, in each viewer
// state. The recorded answers are the two-setter viewer's.
func TestTheMergedLightOfEachFrameIsUnchanged(t *testing.T) {
	maps := []struct {
		name string
		grid terrain.Grid
	}{
		{"lit", cliffGrid()},
		{"margin", withBlock(cliffGrid(), marginPlane(cliffW, cliffH, [2]int{0, 0}, [2]int{0, 1}, [2]int{0, 2}, [2]int{0, 3}))},
	}
	var b strings.Builder
	for _, m := range maps {
		for _, o := range lightOptions {
			v, err := NewViewer("light-plane", m.grid, litTileset())
			if err != nil {
				t.Fatalf("NewViewer: %v", err)
			}
			if !v.Lit() {
				t.Fatal("fixture must be lit")
			}
			if o.ambient != 0 {
				v.sun = terrain.Light{Ambient: uint8(o.ambient)}
			}
			v.SetGraphicsOptions(GraphicsOptions{DisableLighting: o.off})
			v.SetUnshaded(o.unshaded)
			for _, f := range lightFrames() {
				setLightFrame(v, f)
				for ty := 0; ty < cliffH; ty++ {
					for tx := 0; tx < cliffW; tx++ {
						sc := v.cornerScales(tx, ty)
						fmt.Fprintf(&b, "%s %s %s %d,%d corners %s %s %s %s sprite %s\n", m.name, o.name, f.name, tx, ty,
							lightFloat(sc[0]), lightFloat(sc[1]), lightFloat(sc[2]), lightFloat(sc[3]),
							lightFloat(v.spellSpriteFactor(image.Pt(tx, ty))))
					}
				}
			}
		}
	}
	got := b.String()
	path := filepath.Join("testdata", "lightplane.golden")
	if *updateLightPlane {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		gotLines, wantLines := strings.Split(got, "\n"), strings.Split(string(want), "\n")
		for i := range min(len(gotLines), len(wantLines)) {
			if gotLines[i] != wantLines[i] {
				t.Fatalf("line %d: got %q, want %q", i+1, gotLines[i], wantLines[i])
			}
		}
		t.Fatalf("got %d lines, want %d", len(gotLines), len(wantLines))
	}
}
