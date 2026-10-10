package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func panelTestPicture(r image.Rectangle, c color.RGBA) *image.RGBA {
	pic := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	draw.Draw(pic, pic.Bounds(), &image.Uniform{C: c}, image.Point{}, draw.Src)
	return pic
}

// A pressed-only plaque draws its picture only while held inside; a pair
// draws its off picture at rest; a plaque with no art is the shell box only
// where the composition says so.
func TestButtonPanelPlaqueModes(t *testing.T) {
	r0, r1, r2 := image.Rect(10, 10, 30, 20), image.Rect(10, 30, 30, 40), image.Rect(10, 50, 30, 60)
	on, off := color.RGBA{R: 200, A: 255}, color.RGBA{G: 200, A: 255}
	art := panelArt{Plaques: [][2]image.Image{
		{nil, panelTestPicture(r0, on)},
		{panelTestPicture(r1, off), panelTestPicture(r1, on)},
		{},
	}}
	compose := func(bare bool, held bool) *image.RGBA {
		c := panelComposition{Plaques: []image.Rectangle{r0, r1, r2}, Bare: bare, Ink: plaqueCommandInk}
		buttons := []panelButton{{Pressed: held, Inside: held}, {Pressed: held, Inside: held}, {}}
		dst := image.NewRGBA(image.Rect(0, 0, 40, 70))
		p := buildButtonPanel(c, art, buttons)
		p.drawBody(dst)
		p.drawButtons(dst, nil)
		return dst
	}
	rest, held := compose(false, false), compose(false, true)
	if got := rest.RGBAAt(15, 15); got.A != 0 {
		t.Errorf("pressed-only plaque at rest = %v, want nothing drawn", got)
	}
	if got := held.RGBAAt(15, 15); got != on {
		t.Errorf("pressed-only plaque held = %v, want %v", got, on)
	}
	if got := rest.RGBAAt(15, 35); got != off {
		t.Errorf("pair at rest = %v, want %v", got, off)
	}
	if got := held.RGBAAt(15, 35); got != on {
		t.Errorf("pair held = %v, want %v", got, on)
	}
	if got := rest.RGBAAt(15, 55); got.A != 0 {
		t.Errorf("artless plaque without Bare = %v, want nothing", got)
	}
	if got := compose(true, false).RGBAAt(15, 55); got != townShellPanel {
		t.Errorf("artless plaque with Bare = %v, want the shell box %v", got, townShellPanel)
	}
}

// A keyed body draws over what lies under it; an unkeyed one is copied; a
// missing body draws the composition's box; the seam draws over the body.
func TestButtonPanelBody(t *testing.T) {
	body := image.Rect(0, 0, 20, 20)
	under := color.RGBA{B: 200, A: 255}
	pic := panelTestPicture(body, color.RGBA{R: 50, A: 255})
	pic.SetRGBA(0, 0, color.RGBA{})
	for _, over := range []bool{true, false} {
		dst := panelTestPicture(image.Rect(0, 0, 40, 40), under)
		buildButtonPanel(panelComposition{Body: body, BodyOver: over}, panelArt{Body: pic}, nil).drawBody(dst)
		want := under
		if !over {
			want = color.RGBA{}
		}
		if got := dst.RGBAAt(0, 0); got != want {
			t.Errorf("over %t: transparent body pixel = %v, want %v", over, got, want)
		}
	}
	dst := panelTestPicture(image.Rect(0, 0, 40, 40), under)
	buildButtonPanel(panelComposition{Body: body, MissingBox: image.Rect(0, 0, 30, 30)}, panelArt{}, nil).drawBody(dst)
	if got := dst.RGBAAt(10, 10); got != townShellPanel {
		t.Errorf("missing body = %v, want the shell box", got)
	}
	seam := panelTestPicture(image.Rect(0, 0, 4, 20), color.RGBA{G: 90, A: 255})
	dst = panelTestPicture(image.Rect(0, 0, 40, 40), under)
	buildButtonPanel(panelComposition{Body: body, Seam: image.Rect(20, 0, 24, 20)}, panelArt{Body: pic, Seam: seam}, nil).drawBody(dst)
	if got := dst.RGBAAt(21, 5); got != (color.RGBA{G: 90, A: 255}) {
		t.Errorf("seam = %v", got)
	}
}

// A description's command needs an on picture, and no two commands overlap.
func TestGeneratorCommandsRefuseMissingArtAndOverlap(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "game", "generators", "rom1.json"))
	if err != nil {
		t.Fatal(err)
	}
	noOn := bytes.Replace(b, []byte(`"on": "graphics/interface/shopbutton4.bmp", `), nil, 1)
	if _, err := DecodeGenerator(noOn); err == nil || !strings.Contains(err.Error(), "no on picture") {
		t.Errorf("a command without an on picture decoded: %v", err)
	}
	overlap := bytes.Replace(b, []byte(`"rect": [494, 160, 614, 212]`), []byte(`"rect": [494, 150, 614, 202]`), 1)
	if _, err := DecodeGenerator(overlap); err == nil || !strings.Contains(err.Error(), "overlaps") {
		t.Errorf("overlapping commands decoded: %v", err)
	}
}
