package main

import (
	"crypto/sha256"
	"fmt"
	"image"
	"testing"
)

// frameRingHash hashes the pixels of every framed control and of the log box
// along their two outer rings, where a frame's border and the first row of
// its fill lie and no caption is drawn.
func frameRingHash(a *app) string {
	img := a.render()
	h := sha256.New()
	ring := func(r image.Rectangle) {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				if x < r.Min.X+2 || y < r.Min.Y+2 || x >= r.Max.X-2 || y >= r.Max.Y-2 {
					o := img.PixOffset(x, y)
					h.Write(img.Pix[o : o+4])
				}
			}
		}
	}
	ring(image.Rect(10, 488+panelH, winW-10, winH-6))
	for _, it := range a.layout() {
		switch it.kind {
		case iButton, iToggle, iField:
			fmt.Fprintf(h, "%v|", it.rect)
			ring(it.rect)
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

// TestStarterFramesKeepTheirPixels pins the border and fill pixels of every
// framed control in both languages, both views and with a field focused.
func TestStarterFramesKeepTheirPixels(t *testing.T) {
	for _, tc := range []struct{ language, lang string }{{"english", "en"}, {"russian", "ru"}} {
		for _, view := range []viewKind{viewMain, viewSettings} {
			for _, focus := range []fieldID{fNone, fAddBase, fVolume, fSaves} {
				a := rigWithLanguage(t, tc.language).app()
				a.setView(view)
				a.focus = focus
				key := fmt.Sprintf("%s/%d/%d", tc.lang, view, focus)
				got := frameRingHash(a)
				if starterFrameHashes[key] != got {
					t.Errorf("%s: ring hash %s, pinned %s", key, got, starterFrameHashes[key])
				}
			}
		}
	}
}

// starterFrameHashes were recorded before the frames became specs.
var starterFrameHashes = map[string]string{
	"en/0/0": "6efe93b4b498d0f2",
	"en/0/1": "e54151fd48054503",
	"en/0/3": "6efe93b4b498d0f2",
	"en/0/4": "6efe93b4b498d0f2",
	"en/1/0": "4c625383ee70e1f8",
	"en/1/1": "4c625383ee70e1f8",
	"en/1/3": "9d835f55bfa1c3bd",
	"en/1/4": "2479d5f46a5879f0",
	"ru/0/0": "6efe93b4b498d0f2",
	"ru/0/1": "e54151fd48054503",
	"ru/0/3": "6efe93b4b498d0f2",
	"ru/0/4": "6efe93b4b498d0f2",
	"ru/1/0": "4c625383ee70e1f8",
	"ru/1/1": "4c625383ee70e1f8",
	"ru/1/3": "9d835f55bfa1c3bd",
	"ru/1/4": "2479d5f46a5879f0",
}
