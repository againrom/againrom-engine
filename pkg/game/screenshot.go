package game

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"
	"time"

	"againrom/pkg/ui"
)

type screenshotBuffer struct{ bytes.Buffer }

func (b *screenshotBuffer) Write(data []byte) (int, error) {
	if len(data) > maxSaveBytes-b.Len() {
		return 0, errors.New("screenshot PNG exceeds the safe byte limit")
	}
	return b.Buffer.Write(data)
}

func writeScreenshot(store SaveStore, fences []string, frame image.Image) (string, error) {
	validateTarget := func(target string) error {
		if len(fences) == 0 {
			return refuseProfileWriteTarget(target, "", store.profile)
		}
		for _, fence := range fences {
			if err := refuseProfileWriteTarget(target, fence, store.profile); err != nil {
				return err
			}
		}
		return nil
	}
	if err := validateTarget(store.Dir); err != nil {
		return "", err
	}
	if frame == nil {
		return "", errors.New("no completed screenshot frame")
	}
	bounds := frame.Bounds()
	w, h := uint64(bounds.Max.X)-uint64(bounds.Min.X), uint64(bounds.Max.Y)-uint64(bounds.Min.Y)
	if bounds.Max.X <= bounds.Min.X || bounds.Max.Y <= bounds.Min.Y || w > 8192 || h > 8192 || w*h > 1<<24 {
		return "", errors.New("screenshot frame dimensions exceed the safe limit")
	}
	var encoded screenshotBuffer
	if err := png.Encode(&encoded, frame); err != nil {
		return "", err
	}
	return store.write(encoded.Bytes(), "save-screenshot", originalSaveExt, func(dir saveDirectoryIdentity) error {
		return validateTarget(dir.path)
	}, func(at int) (string, bool) {
		if at >= 10000 {
			return "", false
		}
		return fmt.Sprintf("screen%04d.png", at), true
	})
}

func (f *FrontEnd) configureScreenshot(app *ui.App, store SaveStore, fences []string) {
	app.SetScreenshotSink(func(frame image.Image) error {
		name, err := writeScreenshot(store, fences, frame)
		if err == nil && f.live != nil && f.live.view != nil {
			f.live.view.PostMessage("Screenshot saved: "+name, ui.MessageWhite, 3*time.Second)
		}
		return err
	})
}
