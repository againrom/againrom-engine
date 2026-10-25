package main

import (
	"bytes"
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/internal/appicon"
)

// The starter's window gets the starter icon before the window runs, and no
// window opens.
func TestRunWindowHandsTheStarterItsOwnIcon(t *testing.T) {
	saved := desktop
	t.Cleanup(func() { desktop = saved })
	var calls []string
	var icons []image.Image
	desktop.setIcon = func(images []image.Image) { calls = append(calls, "icon"); icons = images }
	desktop.run = func(ebiten.Game) error { calls = append(calls, "run"); return nil }
	a := newApp(t.TempDir()+"/starter.ini", "starter.exe", "windows", "0.0.0", deps{})
	if err := runWindow(a); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 || calls[0] != "icon" || calls[1] != "run" {
		t.Fatalf("the window system was asked %v", calls)
	}
	want := appicon.Images(appicon.Starter)
	if len(icons) != len(want) {
		t.Fatalf("%d images, want %d", len(icons), len(want))
	}
	for i := range want {
		if !bytes.Equal(icons[i].(*image.NRGBA).Pix, want[i].(*image.NRGBA).Pix) {
			t.Errorf("image %d is not the starter icon", i)
		}
	}
}
