package ui

import (
	"image"
	"testing"
	"time"
)

func crossRegistry() *CursorRegistry {
	return NewCursorRegistry([]CursorSlot{
		{Name: "default", Frames: []*image.RGBA{{}}, Hotspot: image.Pt(5, 5), FrameCount: 1, PeriodMillis: 2000000000},
		{Name: "cantput", Frames: []*image.RGBA{{}}, Hotspot: image.Pt(38, 36), FrameCount: 1, PeriodMillis: 2000000000},
	})
}

func TestShopCrossCursorFollowsAHeldShelfStamp(t *testing.T) {
	panel := TownCharacterRegion.Min.Add(image.Pt(60, 60))
	outside := ShopTableCellRect(2).Min.Add(image.Pt(30, 30))
	for _, tc := range []struct {
		name  string
		from  image.Point
		to    image.Point
		cross bool
	}{
		{"shelf over the panel", shopDragShelfPoint, panel, true},
		{"shelf outside the panel", shopDragShelfPoint, outside, false},
		{"merchant table place over the panel", ShopTableCellRect(0).Min.Add(image.Pt(4, 4)), panel, true},
		{"owned table place over the panel", ShopTableCellRect(1).Min.Add(image.Pt(4, 4)), panel, false},
		{"worn object over the panel", shopDragDollPoint, panel, false},
		{"pack object over the panel", ShopPackCellRect(1).Min.Add(image.Pt(4, 4)), panel, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, _ := shopDragTestApp(t)
			a.SetCursorRegistry(crossRegistry())
			a.flow.cursor.SetCursor("default")
			now := time.Unix(1_700_000_000, 0)
			a.step(appInput{CursorX: tc.from.X, CursorY: tc.from.Y, PrimaryPressed: true}, now)
			a.step(appInput{CursorX: tc.to.X, CursorY: tc.to.Y}, now)
			a.step(appInput{CursorX: tc.to.X, CursorY: tc.to.Y}, now)
			want := "default"
			if tc.cross {
				want = "cantput"
			}
			if got := a.flow.cursor.CurrentName(); got != want {
				t.Fatalf("held cursor = %q, want %q", got, want)
			}
			a.step(appInput{CursorX: tc.to.X, CursorY: tc.to.Y, PrimaryReleased: true}, now)
			if got := a.flow.cursor.CurrentName(); got != "default" {
				t.Fatalf("cursor after the release = %q, want default", got)
			}
		})
	}
}
