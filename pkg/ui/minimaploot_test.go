package ui

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/render/terrain"
)

// lootViewer is a 3x3 map whose fog plane is set by plane (row-major) and
// whose only sack stands on (1,1) with a resolvable frame.
func lootViewer(t *testing.T, plane [9]byte) *Viewer {
	t.Helper()
	v, err := NewViewer("mm-loot", grid(3, 3), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	v.SetLocalOwner(1)
	layoutViewport(v, DefaultWindowW, DefaultWindowH)
	v.SetFog(plane[:], 3, 3)
	v.SetSackFrames([]*terrain.StaticFrame{{Width: 4, Height: 4}})
	v.SetSacks([]MapSack{{Cell: image.Pt(1, 1), FrameIndex: 0}})
	return v
}

func allFog(state byte) (p [9]byte) {
	for i := range p {
		p[i] = state
	}
	return p
}

// lootPixel composes the minimap the way the framed picture does and reports
// whether the pixel block of cell holds the loot colour.
func lootPixel(t *testing.T, v *Viewer, cell image.Point) bool {
	t.Helper()
	g, ok := v.minimapGeometry()
	if !ok {
		t.Fatal("no minimap geometry")
	}
	colours, fog, marks, view := v.minimapWindowed(g)
	img := composeMinimap(colours, g.Cols, g.Rows, fog, marks, view, g.extent)
	r := minimapMarkRect(minimapMark{Cell: cell.Sub(g.Origin)}, g.Cols, g.Rows, g.Num, g.Den, img.Bounds())
	if r.Empty() {
		t.Fatalf("cell %v has no pixel", cell)
	}
	return img.RGBAAt(r.Min.X+r.Dx()/2, r.Min.Y+r.Dy()/2) == minimapLootColor
}

// A sack in a visible cell gets a loot dot. Loss control: the same viewer with
// the sack list emptied shows none, so the dot comes from the sack.
func TestMinimapLootDotForAVisibleSack(t *testing.T) {
	v := lootViewer(t, allFog(FogVisible))
	if !lootPixel(t, v, image.Pt(1, 1)) {
		t.Fatal("a visible sack has no loot dot")
	}
	v.SetSacks(nil)
	if lootPixel(t, v, image.Pt(1, 1)) {
		t.Fatal("loss control: no sack still drew a loot dot")
	}
}

// A sack in an explored or an unseen cell gets no dot, the cells where the map
// draws no sack. Loss control: the same sack in a visible cell does.
func TestMinimapLootDotHiddenByFog(t *testing.T) {
	for _, c := range []struct {
		name  string
		state byte
	}{{"explored", FogExplored}, {"unseen", FogUnseen}} {
		v := lootViewer(t, allFog(c.state))
		if got := v.minimapLootMarks(); len(got) != 0 {
			t.Errorf("%s sack marked: %+v", c.name, got)
		}
		if lootPixel(t, v, image.Pt(1, 1)) {
			t.Errorf("%s sack drew a loot pixel", c.name)
		}
		if got := v.sackLayer(); len(got) != 0 {
			t.Errorf("%s: the map draws %d sacks", c.name, len(got))
		}
	}
	if !lootPixel(t, lootViewer(t, allFog(FogVisible)), image.Pt(1, 1)) {
		t.Fatal("loss control: the visible sack lost its dot")
	}
}

// The minimap marks exactly the sacks the map's sackLayer places, over every
// fog state and frame index. Loss control: the population holds both drawn and
// undrawn sacks.
func TestMinimapLootDotsMatchTheMapsSacks(t *testing.T) {
	var plane [9]byte
	plane[0], plane[1], plane[2], plane[3] = FogVisible, FogExplored, FogUnseen, FogVisible
	v := lootViewer(t, plane)
	v.SetSacks([]MapSack{
		{Cell: image.Pt(0, 0), FrameIndex: 0},  // visible: drawn
		{Cell: image.Pt(1, 0), FrameIndex: 0},  // explored: not drawn
		{Cell: image.Pt(2, 0), FrameIndex: 0},  // unseen: not drawn
		{Cell: image.Pt(0, 1), FrameIndex: 5},  // visible, no frame: not drawn
		{Cell: image.Pt(9, 9), FrameIndex: 0},  // outside the grid: not drawn
		{Cell: image.Pt(0, 1), FrameIndex: -1}, // visible, no frame: not drawn
	})
	drawn := map[image.Point]bool{}
	for _, p := range v.sackLayer() {
		drawn[p.Cell] = true
	}
	marked := map[image.Point]bool{}
	for _, m := range v.minimapLootMarks() {
		marked[m.Cell] = true
	}
	if len(drawn) != 1 || len(marked) != len(drawn) {
		t.Fatalf("map draws %v, minimap marks %v; want the same single cell", drawn, marked)
	}
	for c := range drawn {
		if !marked[c] {
			t.Fatalf("map draws a sack on %v the minimap does not mark", c)
		}
	}
}

// With the option off no dot is drawn. Loss control: the same viewer with the
// option on draws one.
func TestMinimapLootDotOptionOff(t *testing.T) {
	v := lootViewer(t, allFog(FogVisible))
	v.SetGraphicsOptions(GraphicsOptions{HideLootDots: true})
	if lootPixel(t, v, image.Pt(1, 1)) || len(v.minimapMarks()) != 0 {
		t.Fatal("a loot dot is drawn with the option off")
	}
	v.SetGraphicsOptions(GraphicsOptions{})
	if !lootPixel(t, v, image.Pt(1, 1)) {
		t.Fatal("loss control: the option on draws no dot")
	}
}

// A unit standing on a sack draws its own colour over the dot, whatever the
// order of the marks. Loss control: the sack alone draws the loot colour.
func TestMinimapUnitOverLootDot(t *testing.T) {
	cell := image.Pt(1, 1)
	for _, local := range []bool{true, false} {
		want := minimapOtherColor
		if local {
			want = minimapLocalColor
		}
		for _, order := range [][]minimapMark{
			{{Cell: cell, Loot: true}, {Cell: cell, Local: local}},
			{{Cell: cell, Local: local}, {Cell: cell, Loot: true}},
		} {
			img := composeMinimap(nil, 3, 3, solidFog(FogVisible), order, image.Rectangle{}, image.Pt(3, 3))
			if got := img.RGBAAt(1, 1); got != want {
				t.Errorf("local=%v order %+v: pixel %v, want %v", local, order, got, want)
			}
		}
	}
	img := composeMinimap(nil, 3, 3, solidFog(FogVisible), []minimapMark{{Cell: cell, Loot: true}}, image.Rectangle{}, image.Pt(3, 3))
	if got := img.RGBAAt(1, 1); got != minimapLootColor {
		t.Fatalf("loss control: sack alone draws %v, want %v", got, minimapLootColor)
	}
}

// The loot colour is apart from the unit marks and the view outline.
func TestMinimapLootColourIsDistinct(t *testing.T) {
	others := map[string]color.RGBA{"local": minimapLocalColor, "other": minimapOtherColor, "view": minimapViewColor, "unseen": minimapUnseenColor}
	for name, c := range others {
		if c == minimapLootColor {
			t.Errorf("loot colour equals the %s colour", name)
		}
	}
}
