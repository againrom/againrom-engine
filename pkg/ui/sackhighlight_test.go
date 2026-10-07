package ui

import (
	"image"
	"reflect"
	"testing"

	"againrom/pkg/render/terrain"
)

func twoSackViewer(t *testing.T) *Viewer {
	t.Helper()
	v := sackIdentityViewer(t)
	v.SetSackFrames([]*terrain.StaticFrame{sackFrame(8, 8)})
	v.SetSacks([]MapSack{{Cell: image.Pt(0, 0), FrameIndex: 0}, {Cell: image.Pt(2, 3), FrameIndex: 0}})
	return v
}

func countOutlines(rs []staticScreenRect) (outlines int) {
	for _, r := range rs {
		if r.SackOutline {
			outlines++
		}
	}
	return outlines
}

func TestSackOutlineEntriesFollowTheFlag(t *testing.T) {
	v := twoSackViewer(t)
	plain := v.planeSprites()
	if len(plain) != 2 || countOutlines(plain) != 0 {
		t.Fatalf("flag down: %d entries, %d outlines; want 2 and 0", len(plain), countOutlines(plain))
	}
	v.setSackHighlight(true)
	got := v.planeSprites()
	if len(got) != 4 || countOutlines(got) != 2 {
		t.Fatalf("flag up: %d entries, %d outlines; want 4 and 2", len(got), countOutlines(got))
	}
	for i, r := range got {
		if !r.SackOutline && i >= 2 {
			t.Errorf("entry %d: a sprite follows an outline", i)
		}
	}
	v.setSackHighlight(false)
	if n := countOutlines(v.planeSprites()); n != 0 {
		t.Errorf("flag released: %d outlines remain", n)
	}
}

func TestSackOutlineSkipsFoggedSack(t *testing.T) {
	v := twoSackViewer(t)
	v.setSackHighlight(true)
	v.SetSacks([]MapSack{{Cell: image.Pt(0, 0), FrameIndex: 0}, {Cell: image.Pt(9, 9), FrameIndex: 0}})
	if n := countOutlines(v.planeSprites()); n != 1 {
		t.Errorf("outlines = %d, want 1: a sack off the grid has none", n)
	}
}

func TestSackOutlinePixelsFollowSilhouette(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 5, 5))
	for _, p := range []image.Point{{2, 2}, {3, 2}} {
		src.Pix[src.PixOffset(p.X, p.Y)+3] = 255
	}
	out := sackOutlinePixels(src, 1)
	if out.Bounds().Dx() != 7 || out.Bounds().Dy() != 7 {
		t.Fatalf("outline size %v, want 7x7", out.Bounds())
	}
	lit := func(x, y int) bool { return out.Pix[out.PixOffset(x+1, y+1)+3] != 0 }
	for _, p := range []image.Point{{2, 2}, {3, 2}, {0, 0}, {4, 4}, {5, 5}} {
		if lit(p.X, p.Y) {
			t.Errorf("pixel %v is lit, want clear (silhouette interior or far)", p)
		}
	}
	for _, p := range []image.Point{{1, 1}, {1, 2}, {1, 3}, {2, 1}, {4, 2}, {4, 3}, {4, 1}, {3, 3}, {2, 3}} {
		if !lit(p.X, p.Y) {
			t.Errorf("pixel %v is clear, want contour", p)
		}
	}
}

func TestSackOutlineWidthByZoom(t *testing.T) {
	for zoom, want := range map[float64]int{4: 1, 1: 1, 0.5: 2, 0.25: 4, 0.1: 4} {
		if got := sackOutlineWidth(zoom); got != want {
			t.Errorf("sackOutlineWidth(%v) = %d, want %d", zoom, got, want)
		}
	}
}

// Every outline is painted after every sprite, so an outline is never covered
// by a sack, tree or building drawn nearer than the sack it traces.
func TestSackOutlineDrawsAboveTheWholeArtLayer(t *testing.T) {
	v := twoSackViewer(t)
	v.setSackHighlight(true)
	var rec staticRecorder
	v.drawArt(&rec)
	if len(rec.imgs) != 4 || rec.imgs[0] != rec.imgs[1] || rec.imgs[2] != rec.imgs[3] || rec.imgs[1] == rec.imgs[2] {
		t.Fatalf("draws = %d, want both sprites and then both outline textures", len(rec.imgs))
	}
	if rec.geo[2][2] >= rec.geo[0][2] {
		t.Errorf("outline x %v is not left of sprite x %v by the contour width", rec.geo[2][2], rec.geo[0][2])
	}
}

func TestHoldingZChangesOnlyTheDrawingFlag(t *testing.T) {
	a, v, seam := atOnMap(t)
	v.SetSackFrames([]*terrain.StaticFrame{sackFrame(8, 8)})
	v.SetSacks([]MapSack{{Cell: image.Pt(1, 1), FrameIndex: 0}, {Cell: image.Pt(2, 3), FrameIndex: 0}})
	entities := append([]MapEntity(nil), v.entities...)
	sacks := append([]MapSack(nil), v.sacks...)
	before := *seam

	x, y := cellPoint(v, atEmptyCol, atEmptyRow)
	held := atFrame(x, y)
	held.HighlightHeld = true
	a.step(held, atAt)
	if !v.SackHighlighted() {
		t.Fatal("flag not set while Z is held")
	}
	a.step(held, atAt)
	if !v.SackHighlighted() {
		t.Fatal("flag dropped while Z is still held")
	}
	after := *seam
	after.ticks, before.ticks = 0, 0
	if !reflect.DeepEqual(after, before) || !reflect.DeepEqual(entities, v.entities) || !reflect.DeepEqual(sacks, v.sacks) {
		t.Error("holding Z changed simulation-side state")
	}

	a.step(atFrame(x, y), atAt)
	if v.SackHighlighted() {
		t.Error("flag still set after Z is released")
	}

	held.Unfocused = true
	a.step(held, atAt)
	if v.SackHighlighted() {
		t.Error("flag set on an unfocused tick")
	}
}
