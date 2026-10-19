package game

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	"againrom/pkg/ui"
)

func TestWorldMapFlagStandsAboveTheCurrentPoint(t *testing.T) {
	s := travelInputsScreen(image.Pt(320, 240))
	flag := image.NewRGBA(image.Rect(0, 0, 20, 24))
	draw.Draw(flag, flag.Bounds(), &image.Uniform{C: color.RGBA{R: 251, G: 17, A: 255}}, image.Point{}, draw.Src)
	s.worldMap.assets.modes = []worldMapModeArt{{party: []*image.RGBA{flag}}}
	v := s.WorldMapView()
	frame := ui.ComposeWorldMap(v)
	v.Flag = image.NewRGBA(flag.Bounds())
	bare := ui.ComposeWorldMap(v)
	want := image.Rect(310, 216, 330, 240)
	in, out := secondMapChanged(frame, bare, want)
	if in != 480 || out != 0 {
		t.Fatalf("Flag rectangle %v: %d changed pixels inside, %d outside; want 480 and 0", want, in, out)
	}
}
