package menu_test

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/render/menu"
)

var secondRects = [8]image.Rectangle{
	rect(204, 52, 104, 96), rect(124, 156, 108, 76), rect(124, 252, 96, 88), rect(208, 340, 100, 100),
	rect(340, 52, 88, 100), rect(424, 152, 84, 88), rect(412, 260, 96, 84), rect(344, 348, 72, 80),
}

var secondCaption = rect(232, 200, 180, 80)

func secondFiles(edits map[string][]byte) mapSource {
	files := synth.MenuFiles(synth.MenuOptions{
		Prefix:    specPrefix,
		Hover:     secondRects,
		Pressed:   secondRects,
		MaskIndex: specHot,
	})
	for i := 1; i <= 8; i++ {
		c := image.NewRGBA(image.Rect(0, 0, 180, 80))
		draw.Draw(c, c.Bounds(), image.NewUniform(color.RGBA{0, uint8(0x20 * i), 0xff, 0xff}), image.Point{}, draw.Src)
		files[specPrefix+fmt.Sprintf("text%d.bmp", i)] = synth.BMP24(c)
	}
	out := mapSource(files)
	for k, v := range edits {
		if v == nil {
			delete(out, k)
		} else {
			out[k] = v
		}
	}
	return out
}

func TestLoadSecondPlacesOverlaysAndCaptions(t *testing.T) {
	a, err := menu.LoadSecond(secondFiles(nil))
	if err != nil {
		t.Fatal(err)
	}
	base := a.Compose(menu.State{})
	for b := 1; b <= 8; b++ {
		for _, pressed := range []bool{false, true} {
			s := menu.State{Selected: b, Pressed: pressed}
			_, at, ok := a.Overlay(s)
			if !ok || at != secondRects[b-1] {
				t.Fatalf("button %d pressed %v: rect %v ok %v, want %v", b, pressed, at, ok, secondRects[b-1])
			}
			frame := a.Compose(s)
			if got := frame.RGBAAt(secondCaption.Min.X, secondCaption.Min.Y); got != (color.RGBA{0, uint8(0x20 * b), 0xff, 0xff}) {
				t.Errorf("button %d: caption corner %v", b, got)
			}
			for y := 0; y < 480; y++ {
				for x := 0; x < 640; x++ {
					p := image.Pt(x, y)
					if p.In(at) || p.In(secondCaption) {
						continue
					}
					if frame.RGBAAt(x, y) != base.RGBAAt(x, y) {
						t.Fatalf("button %d: pixel %v changed outside overlay and caption", b, p)
					}
				}
			}
		}
	}
	if got := a.Compose(menu.State{}); got.RGBAAt(secondCaption.Min.X, secondCaption.Min.Y) != base.RGBAAt(secondCaption.Min.X, secondCaption.Min.Y) {
		t.Error("an unselected frame carries a caption")
	}
}

func TestFirstGameSetHasNoCaptionAndKeepsItsRects(t *testing.T) {
	a := mustLoad(t, filesWith(nil))
	for b := 1; b <= 8; b++ {
		_, at, _ := a.Overlay(menu.State{Selected: b})
		if at != specHover[b-1] {
			t.Errorf("button %d hover rect %v, want %v", b, at, specHover[b-1])
		}
	}
	if a.Compose(menu.State{Selected: 1}).RGBAAt(secondCaption.Min.X, secondCaption.Min.Y) != a.Compose(menu.State{}).RGBAAt(secondCaption.Min.X, secondCaption.Min.Y) {
		t.Error("the first game's frame draws a caption")
	}
}

func TestLoadSecondRefuses(t *testing.T) {
	cases := []struct {
		name  string
		files mapSource
		want  string
	}{
		{"missing caption", secondFiles(map[string][]byte{specPrefix + "text4.bmp": nil}), ""},
		{"first game's overlay sizes", func() mapSource {
			m := secondFiles(nil)
			for k, v := range filesWith(nil) {
				if strings.Contains(k, "button") {
					m[k] = v
				}
			}
			return m
		}(), "button1.bmp"},
		{"wrong caption size", secondFiles(map[string][]byte{specPrefix + "text2.bmp": synth.BMP24(image.NewRGBA(image.Rect(0, 0, 100, 80)))}), "text2.bmp"},
	}
	for _, c := range cases {
		a, err := menu.LoadSecond(c.files)
		if err == nil || a != nil {
			t.Errorf("%s: loaded", c.name)
			continue
		}
		if c.want != "" && !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: error %q does not name %s", c.name, err, c.want)
		}
	}
	if _, err := menu.Load(secondFiles(nil)); err == nil {
		t.Error("the first game's loader accepted the second game's overlay sizes")
	}
}
