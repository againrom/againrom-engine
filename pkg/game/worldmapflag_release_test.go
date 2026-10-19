package game

import (
	"image"
	"testing"

	"againrom/pkg/ui"
)

func TestReleaseWorldMapFlagStandsAboveTheCurrentPoint(t *testing.T) {
	w := openCrossWitness(t)
	v := w.screen.WorldMapView()
	if v.Flag == nil || v.Flag.Bounds().Size() != image.Pt(20, 24) {
		t.Fatal("installed Flag is not a 20x24 frame")
	}
	frame := ui.ComposeWorldMap(v)
	bare := v
	bare.Flag = image.NewRGBA(v.Flag.Bounds())
	want := image.Rect(v.Position.X-10, v.Position.Y-24, v.Position.X+10, v.Position.Y)
	in, out := secondMapChanged(frame, ui.ComposeWorldMap(bare), want)
	if in == 0 || out != 0 {
		t.Fatalf("Flag rectangle %v: %d changed pixels inside, %d outside", want, in, out)
	}
	crop := frame.SubImage(want.Inset(-8)).(*image.RGBA)
	writeModShot(t, "worldmap-current-flag-crop", crop)
	t.Logf("Flag at current point %v: frame %v, %d changed pixels inside, %d outside", v.Position, want, in, out)
}
