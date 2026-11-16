package ui

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/render/terrain"
)

func minimapWitnessFog(w, h int, state uint8) []byte {
	plane := make([]byte, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			plane[y*w+x] = state
			if state > FogVisible {
				plane[y*w+x] = []byte{FogUnseen, FogExplored, FogVisible}[(x+y)%3]
			}
		}
	}
	return plane
}

// minimapWitnessVariants apply one state of fog and options to a viewer.
// A mod group's own colour reaches the viewer with no paint data: the cell of
// the group shows it under the symbolic fog rule, with and without the
// crystal, while the other cells keep their sampled colour. Loss control:
// without the paint the same cell shows the placeholder.
func TestAModGroupColourReachesAViewerWithNoPaintData(t *testing.T) {
	const group = 20
	own := color.RGBA{R: 12, G: 200, B: 34, A: 255}
	for _, crystal := range []bool{false, true} {
		words := make([]uint16, 8*8)
		words[8*2+3] = group<<6 | 5
		v, err := NewViewer("mm-group", grid(8, 8, words...), &terrain.Tileset{})
		if err != nil {
			t.Fatalf("NewViewer: %v", err)
		}
		v.Layout(MissionFrameW, MissionFrameH)
		v.SetEntities(nil)
		if crystal {
			v.SetDialogFrame(&DialogFrame{Minimap: originalCrystal()})
		}
		paint := &MinimapPaint{}
		paint.GroupColours[group] = own
		v.SetMinimapPaint(paint)
		if v.minimapOriginalActive() {
			t.Fatal("a paint with no lookup turned the installed picture on")
		}
		cellColour := func(cell image.Point) color.RGBA {
			t.Helper()
			g, ok := v.minimapGeometry()
			if !ok {
				t.Fatal("no minimap geometry")
			}
			img := composeMinimapPlan(v.minimapProducePlan(g), g.extent)
			r := minimapMarkRect(minimapMark{Cell: cell}, g.Cols, g.Rows, g.Num, g.Den, img.Bounds())
			return img.RGBAAt(r.Min.X+r.Dx()/2, r.Min.Y+r.Dy()/2)
		}
		if got := cellColour(image.Pt(3, 2)); got != own {
			t.Errorf("crystal %t: mod group's cell shows %v, want its own colour %v", crystal, got, own)
		}
		if got := cellColour(image.Pt(4, 4)); got != terrain.PlaceholderColor {
			t.Errorf("crystal %t: a plain cell shows %v, want the sampled placeholder", crystal, got)
		}
		v.SetFog(minimapWitnessFog(8, 8, FogExplored), 8, 8)
		if got, want := cellColour(image.Pt(3, 2)), (color.RGBA{R: own.R / 2, G: own.G / 2, B: own.B / 2, A: 255}); got != want {
			t.Errorf("crystal %t: explored mod group's cell shows %v, want the symbolic half %v", crystal, got, want)
		}
		v.SetFog(nil, 0, 0)
		v.SetMinimapPaint(nil)
		if got := cellColour(image.Pt(3, 2)); got == own {
			t.Errorf("crystal %t: loss control: the viewer with no paint drew the mod colour", crystal)
		}
	}
}
