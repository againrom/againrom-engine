package ui

import (
	"bytes"
	"image/color"
	"testing"
)

func TestTownListColdFrameUsesOrdinaryRowsAndDisabledGates(t *testing.T) {
	a := newTestApp(t, appRows(1), nil)
	town := &fakeTown{}
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("row-list town unavailable")
	}
	pix, note, err := a.HeadlessFrame()
	if err != nil || note != "" {
		t.Fatal("ordinary row-list frame", err, note)
	}
	for _, c := range []struct {
		x, y int
		want color.RGBA
	}{
		{300, 47, color.RGBA{67, 78, 96, 255}},
		{300, 63, color.RGBA{34, 39, 48, 255}},
		{300, 79, color.RGBA{13, 13, 20, 255}},
	} {
		if got := pix.RGBAAt(c.x, c.y); got != c.want {
			t.Fatalf("row button at %d,%d=%v want%v", c.x, c.y, got, c.want)
		}
	}
	if err := a.HeadlessActivate("a shut door"); err == nil || len(town.chosen) != 0 {
		t.Fatal("disabled row activated", err, town.chosen)
	}
	if err := a.HeadlessActivate("another door"); err != nil {
		t.Fatal(err)
	}
	after, _, err := a.HeadlessFrame()
	if err != nil || bytes.Equal(pix.Pix, after.Pix) || town.Header() != "inside" {
		t.Fatal("ordinary row selection did not change room/frame", err)
	}
	cold := newTestApp(t, appRows(1), nil)
	cold.SetTown(&fakeTown{inside: true})
	if !cold.flow.showTown("") {
		t.Fatal("cold room unavailable")
	}
	again, _, err := cold.HeadlessFrame()
	if err != nil || !bytes.Equal(after.Pix, again.Pix) {
		t.Fatal("cold room composed different header/rows", err)
	}
}
