package ui

import (
	"errors"
	"image"
	"image/draw"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

type screenshotState struct {
	sink      func(image.Image) error
	requested bool
}

func (a *App) SetScreenshotSink(sink func(image.Image) error) {
	a.screenshot.sink = sink
	if sink == nil {
		a.screenshot.requested = false
	}
}

func (a *App) requestScreenshot() {
	if a.screenshot.sink != nil {
		a.screenshot.requested = true
	}
}

func (a *App) captureScreenshot(screen *ebiten.Image) {
	_ = a.finishScreenshot(screen)
}

func (a *App) HeadlessScreenshotFrame(frame image.Image) error {
	return a.finishScreenshot(frame)
}

func (a *App) finishScreenshot(frame image.Image) error {
	if !a.screenshot.requested || a.screenshot.sink == nil {
		return nil
	}
	sink := a.screenshot.sink
	a.screenshot.requested = false
	copy, err := screenshotCopy(frame)
	if err == nil {
		err = sink(copy)
	}
	if err != nil && a.flow != nil && a.flow.viewer != nil {
		a.flow.viewer.PostMessage("Screenshot failed: "+err.Error(), MessageWhite, 3*time.Second)
	}
	return err
}

func screenshotCopy(frame image.Image) (*image.RGBA, error) {
	if frame == nil {
		return nil, errors.New("no completed screenshot frame")
	}
	bounds := frame.Bounds()
	w, h := uint64(bounds.Max.X)-uint64(bounds.Min.X), uint64(bounds.Max.Y)-uint64(bounds.Min.Y)
	if bounds.Max.X <= bounds.Min.X || bounds.Max.Y <= bounds.Min.Y || w > 8192 || h > 8192 || w*h > 1<<24 {
		return nil, errors.New("screenshot frame dimensions exceed the safe limit")
	}
	copy := image.NewRGBA(image.Rect(0, 0, int(w), int(h)))
	if screen, ok := frame.(*ebiten.Image); ok {
		screen.ReadPixels(copy.Pix)
	} else {
		draw.Draw(copy, copy.Bounds(), frame, bounds.Min, draw.Src)
	}
	return copy, nil
}
