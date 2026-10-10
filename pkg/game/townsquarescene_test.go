package game

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// The square scene answers the four doors in the row order Choose(i) takes
// and the statue's menu from the five significant mask bytes, and nothing for
// any other byte or a point off the mask (TOWN-161, DIV-861).
func TestTownSquareSceneControlsFromTheMaskBytes(t *testing.T) {
	f := shellFrontEnd()
	f.TownSquareArt = resolved(exteriorTestArt(t), nil)
	f.Town.announceMission(f.Town.currentMain())
	_, screen := exteriorApp(t, f)
	scene := townSquareScene{screen}
	door := func(i int) ui.TownSquareControl {
		return ui.TownSquareControl{Kind: ui.TownSquareControlDoor, Door: i}
	}
	for _, c := range []struct {
		name string
		p    image.Point
		want ui.TownSquareControl
	}{
		{"tavern 0x80", image.Pt(10, 100), door(0)},
		{"shop 0x90", image.Pt(20, 100), door(1)},
		{"school 0xc0", image.Pt(30, 100), door(2)},
		{"gate 0xa0", image.Pt(40, 100), door(3)},
		{"statue 0xb0", image.Pt(50, 100), ui.TownSquareControl{Kind: ui.TownSquareControlMenu}},
	} {
		if got, ok := scene.ControlAt(c.p); !ok || got != c.want {
			t.Errorf("%s: ControlAt(%v) = %+v, %v; want %+v", c.name, c.p, got, ok, c.want)
		}
	}
	f.TownSquareArt.Value().Mask.SetColorIndex(5, 5, 7)
	for _, p := range []image.Point{{0, 0}, {5, 5}, {-1, 0}, {640, 100}} {
		if got, ok := scene.ControlAt(p); ok {
			t.Errorf("ControlAt(%v) = %+v; want no hit", p, got)
		}
	}
}

// Each hover label is copied at its own origin over the base picture: a pixel
// inside the placed rectangle carries the label, one just outside the base
// (TOWN-161).
func TestTownSquareLabelsCopyAtTheirOrigins(t *testing.T) {
	solid := func(w, h int, v uint8) image.Image {
		pic := image.NewRGBA(image.Rect(0, 0, w, h))
		for i := range pic.Pix {
			pic.Pix[i] = v
		}
		return pic
	}
	mask := image.NewPaletted(image.Rect(0, 0, 640, 480), make(color.Palette, 256))
	mask.SetColorIndex(1, 1, 0x90)
	mask.SetColorIndex(2, 1, 0x80)
	mask.SetColorIndex(3, 1, 0xc0)
	art := squareArt(mask, map[string][]image.Image{
		"base":         {solid(640, 480, 0x40)},
		"label-shop":   {solid(52, 76, 0x60)},
		"label-tavern": {solid(28, 64, 0x61)},
		"label-school": {solid(140, 116, 0x62)},
	})
	for i, c := range []struct {
		hover  image.Point
		origin image.Point
	}{{image.Pt(1, 1), image.Pt(264, 264)}, {image.Pt(2, 1), image.Pt(144, 332)}, {image.Pt(3, 1), image.Pt(436, 300)}} {
		v := town.NewScene(ROM1TownDescription(), "square", stillHost{art}, nil)
		v.Pointer(c.hover)
		dst := image.NewRGBA(image.Rectangle{Max: v.Size()})
		v.Paint(dst, "")
		if got, want := dst.RGBAAt(c.origin.X+1, c.origin.Y+1).R, uint8(0x60+i); got != want {
			t.Errorf("label %d inside its origin = %#x, want %#x", i, got, want)
		}
		if got := dst.RGBAAt(c.origin.X-1, c.origin.Y-1).R; got != 0x40 {
			t.Errorf("label %d just outside its origin = %#x, want the base 0x40", i, got)
		}
	}
}
