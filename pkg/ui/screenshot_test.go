package ui

import (
	"errors"
	"image"
	"image/color"
	"testing"
)

func TestScreenshotAltQueuesOneDetachedFrame(t *testing.T) {
	a, _ := mkOnMap(t)
	var delivered []image.Image
	a.SetScreenshotSink(func(frame image.Image) error {
		delivered = append(delivered, frame)
		return nil
	})
	for range 2 {
		if err := a.HeadlessKey("alt-s"); err != nil {
			t.Fatal(err)
		}
	}
	if !a.screenshot.requested || len(delivered) != 0 {
		t.Fatal("Alt+S must wait for the next completed frame")
	}
	source := image.NewRGBA(image.Rect(7, 11, 12, 14))
	source.SetRGBA(9, 12, color.RGBA{R: 41, G: 73, B: 99, A: 255})
	if err := a.HeadlessScreenshotFrame(source); err != nil {
		t.Fatal(err)
	}
	if len(delivered) != 1 || a.screenshot.requested || delivered[0].Bounds() != image.Rect(0, 0, 5, 3) {
		t.Fatal("request must deliver one frame with normalized bounds")
	}
	source.SetRGBA(9, 12, color.RGBA{A: 255})
	if got := color.RGBAModel.Convert(delivered[0].At(2, 1)).(color.RGBA); got != (color.RGBA{R: 41, G: 73, B: 99, A: 255}) {
		t.Fatal("delivered frame aliases the supplied frame", got)
	}
	if err := a.HeadlessScreenshotFrame(source); err != nil || len(delivered) != 1 {
		t.Fatal("completed request was delivered again", err)
	}
}

func TestScreenshotFailureConsumesRequestAndAllowsNextFrame(t *testing.T) {
	a, _ := mkOnMap(t)
	want := errors.New("publication refused")
	calls := 0
	a.SetScreenshotSink(func(image.Image) error {
		calls++
		a.requestScreenshot()
		return want
	})
	a.requestScreenshot()
	if err := a.HeadlessScreenshotFrame(image.NewRGBA(image.Rect(0, 0, 1, 1))); !errors.Is(err, want) || calls != 1 || !a.screenshot.requested {
		t.Fatal("callback request for a later frame was lost", err, calls)
	}
	a.SetScreenshotSink(nil)
	if a.screenshot.requested {
		t.Fatal("removing the sink retained a pending capture")
	}
	if err := a.HeadlessKey("alt-s"); err != nil || a.screenshot.requested {
		t.Fatal("unconfigured screenshot route queued a capture", err)
	}
}

type screenshotExtent struct{ bounds image.Rectangle }

func (s screenshotExtent) Bounds() image.Rectangle { return s.bounds }
func (s screenshotExtent) ColorModel() color.Model { return color.RGBAModel }
func (s screenshotExtent) At(int, int) color.Color {
	panic("invalid extent must be refused before reading pixels")
}

func TestScreenshotBoundsRefuseBeforePixelAllocation(t *testing.T) {
	a, _ := mkOnMap(t)
	calls := 0
	a.SetScreenshotSink(func(image.Image) error { calls++; return nil })
	for _, frame := range []image.Image{nil, screenshotExtent{image.Rect(0, 0, 0, 10)}, screenshotExtent{image.Rect(0, 0, 8193, 1)}, screenshotExtent{image.Rect(0, 0, 8192, 8192)}} {
		a.requestScreenshot()
		if err := a.HeadlessScreenshotFrame(frame); err == nil || a.screenshot.requested {
			t.Fatal("invalid frame was accepted or retained", err)
		}
	}
	if calls != 0 {
		t.Fatal("invalid frame reached output sink")
	}
}
