package ui

import (
	"image"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// TestStageOfIsTheOneFilterRule pins the stage each executor runs for each
// filter at each scale.
func TestStageOfIsTheOneFilterRule(t *testing.T) {
	const (
		none    = scaleNone
		nearest = scaleNearest
		catmull = scaleCatmull
		sharp   = scaleSharp
	)
	cases := []struct {
		name   string
		filter frameFilter
		s      float64
		want   scaleChoice
	}{
		{"smooth off the window", filterSmooth, -1, scaleChoice{none, none}},
		{"smooth at one", filterSmooth, 1, scaleChoice{nearest, nearest}},
		{"smooth fractional", filterSmooth, 2.5, scaleChoice{catmull, catmull}},
		{"smooth whole", filterSmooth, 2, scaleChoice{catmull, catmull}},
		{"pixel below two", filterPixel, 1.5, scaleChoice{nearest, nearest}},
		{"pixel at zero", filterPixel, 0, scaleChoice{sharp, none}},
		{"pixel whole from two", filterPixel, 2, scaleChoice{sharp, nearest}},
		{"pixel fractional from two", filterPixel, 2.5, scaleChoice{sharp, sharp}},
		{"sharp at one", filterSharp, 1, scaleChoice{sharp, nearest}},
		{"sharp whole", filterSharp, 3, scaleChoice{sharp, nearest}},
		{"sharp fractional", filterSharp, 1.5, scaleChoice{sharp, sharp}},
	}
	for _, c := range cases {
		if got := stageOf(c.filter, c.s); got != c.want {
			t.Errorf("%s: stageOf(%d, %v) = %+v, want %+v", c.name, c.filter, c.s, got, c.want)
		}
	}
}

// TestFinalFrameExecutorsRunTheStageOf checks that both executors take the
// path stageOf names for every filter and scale, so neither keeps a rule.
func TestFinalFrameExecutorsRunTheStageOf(t *testing.T) {
	var ran string
	origSharp, origCatmull, origNearest := sharpBilinearBlit, catmullRomBlit, nearestBlit
	t.Cleanup(func() { sharpBilinearBlit, catmullRomBlit, nearestBlit = origSharp, origCatmull, origNearest })
	sharpBilinearBlit = func(dst, src *ebiten.Image, op *ebiten.DrawImageOptions, buf **ebiten.Image) { ran = "sharp" }
	catmullRomBlit = func(dst, src *ebiten.Image, op *ebiten.DrawImageOptions) bool { ran = "catmull"; return true }
	nearestBlit = func(dst, src *ebiten.Image, op *ebiten.DrawImageOptions) { ran = "nearest" }
	names := map[scaleStage]string{scaleNone: "", scaleNearest: "nearest", scaleCatmull: "catmull", scaleSharp: "sharp"}
	dst, src := new(ebiten.Image), new(ebiten.Image)
	dstRGBA, srcRGBA := image.NewRGBA(image.Rect(0, 0, 8, 8)), image.NewRGBA(image.Rect(0, 0, 4, 4))
	for _, filter := range []frameFilter{filterSmooth, filterSharp, filterPixel} {
		for _, s := range []float64{-1, 0, 0.5, 1, 1.5, 2, 2.5, 3} {
			want := stageOf(filter, s)
			var op ebiten.DrawImageOptions
			op.GeoM.Scale(s, s)
			ran = ""
			var buf *ebiten.Image
			drawFinalFrame(dst, src, &op, &buf, filter)
			if ran != names[want.window] {
				t.Errorf("drawFinalFrame(filter %d, s %v) ran %q, want %q", filter, s, ran, names[want.window])
			}
			label := referenceFinalFrame(dstRGBA, srcRGBA, s, 0, 0, filter)
			if got := labelStage(label); got != want.cpu {
				t.Errorf("referenceFinalFrame(filter %d, s %v) = %q, want stage %d", filter, s, label, want.cpu)
			}
		}
	}
}

// labelStage maps the name referenceFinalFrame answers to its stage.
func labelStage(label string) scaleStage {
	switch {
	case label == "none":
		return scaleNone
	case label == "catmull-rom":
		return scaleCatmull
	case strings.HasPrefix(label, "sharp bilinear"):
		return scaleSharp
	}
	return scaleNearest
}
