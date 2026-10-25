package main

import (
	"bytes"
	"image"
	"testing"

	"againrom/internal/appicon"
	"againrom/pkg/ui"
)

// The editor's window gets the map editor icon before the editor runs, and no
// window opens.
func TestOpenWindowHandsTheEditorItsOwnIcon(t *testing.T) {
	saved := desktop
	t.Cleanup(func() { desktop = saved })
	var calls []string
	var icons []image.Image
	desktop.setIcon = func(images []image.Image) { calls = append(calls, "icon"); icons = images }
	desktop.run = func(*ui.MapEditor) error { calls = append(calls, "run"); return nil }
	if err := openWindow(nil); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 || calls[0] != "icon" || calls[1] != "run" {
		t.Fatalf("the window system was asked %v", calls)
	}
	want := appicon.Images(appicon.MapEditor)
	if len(icons) != len(want) {
		t.Fatalf("%d images, want %d", len(icons), len(want))
	}
	for i := range want {
		if !bytes.Equal(icons[i].(*image.NRGBA).Pix, want[i].(*image.NRGBA).Pix) {
			t.Errorf("image %d is not the map editor icon", i)
		}
	}
	if bytes.Equal(icons[len(icons)-1].(*image.NRGBA).Pix, appicon.Images(appicon.Game)[len(want)-1].(*image.NRGBA).Pix) {
		t.Error("the editor got the game icon")
	}
}
