package ui

import (
	"image"
	"image/color"
	"testing"
)

func TestWorldMap1088ReturnPaintsHomeCrossWithoutScrolls(t *testing.T) {
	background := image.NewRGBA(image.Rect(0, 0, 640, 480))
	for y := 0; y < 480; y++ {
		for x := 0; x < 640; x++ {
			background.SetRGBA(x, y, color.RGBA{11, 22, 33, 255})
		}
	}
	v := WorldMapView{
		Background: background, Position: image.Pt(500, 400), Destination: image.Pt(350, 300),
		Returning: true, HideScrolls: true, Selected: -1, Hovered: -1,
		Missions: []WorldMapMission{{Number: 40, Title: "NEXT", Enabled: true}},
	}
	got := ComposeWorldMap(v)
	// Independent literal rectangle covers both rows of cards. No production
	// card geometry or second production composition constructs the oracle.
	for y := 8; y < 162; y++ {
		for x := 8; x < 632; x++ {
			if got.RGBAAt(x, y) != (color.RGBA{11, 22, 33, 255}) {
				t.Fatalf("scroll pixel remained at %d,%d", x, y)
			}
		}
	}
	if got.RGBAAt(350, 300) != (color.RGBA{190, 41, 31, 255}) {
		t.Fatal("home destination cross not painted")
	}
	for _, p := range []image.Point{{10, 10}, {170, 10}} {
		if _, hit := WorldMapCardAt(v, p); hit {
			t.Fatalf("hidden scroll accepts %v", p)
		}
	}
}
