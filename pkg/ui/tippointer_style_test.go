package ui

import (
	"image"
	"testing"
	"time"
)

func sameTipControlPixels(a, b *image.RGBA, r image.Rectangle) bool {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				return false
			}
		}
	}
	return true
}

func TestTownTipClosePresentationFollowsPointerAndLatch(t *testing.T) {
	for _, room := range []string{"square", "tavern", "school", "shop"} {
		t.Run(room, func(t *testing.T) {
			a := newTestApp(t, appRows(3), okLoader(t))
			v := TipPanelView{Text: "tip", Art: tipTestArt(), Font: shopTipTestFont()}
			switch room {
			case "square":
				v.Rect = TownTipRect
				a.SetTown(&fakeTipSquareEnumTown{scene: &fakeSquareScene{}, tip: v})
			case "tavern", "school":
				kind := TownSurfaceTavern
				v.Rect = TavernTipRect
				if room == "school" {
					kind, v.Rect = TownSurfaceSchool, SchoolTipRect
				}
				a.SetTown(&fakeTipSurfaceEnumTown{view: TownSurfaceView{Kind: kind, Tip: v}})
			case "shop":
				v.Rect = ShopTipRect()
				a.SetTown(&fakeTipShopCrossingTown{tip: v})
			}
			if !a.flow.showTown("") {
				t.Fatal("fixture could not show room")
			}
			control := TipPanelCloseRect(v.Rect)
			p, _ := sampleInside(control)
			paint := func(in appInput) *image.RGBA {
				t.Helper()
				a.step(in, time.Unix(1, 0))
				pic, err := a.composeTownRoom()
				if err != nil {
					t.Fatal(err)
				}
				return pic
			}
			idle := paint(appInput{CursorX: 0, CursorY: 479})
			hover := paint(appInput{CursorX: p.X, CursorY: p.Y})
			if sameTipControlPixels(idle, hover, control) {
				t.Fatal("Close hover did not change its painted control")
			}
			pressed := paint(appInput{CursorX: p.X, CursorY: p.Y, PrimaryPressed: true})
			if sameTipControlPixels(hover, pressed, control) {
				t.Fatal("registered Close press did not change its bevel and label shadow")
			}
			outside := paint(appInput{CursorX: 0, CursorY: 479})
			if !sameTipControlPixels(idle, outside, control) {
				t.Fatal("pressed Close stayed sunken outside its rectangle")
			}
			paint(appInput{CursorX: 0, CursorY: 479, PrimaryReleased: true})
			released := paint(appInput{CursorX: p.X, CursorY: p.Y})
			if !sameTipControlPixels(hover, released, control) {
				t.Fatal("outside release retained the Close press presentation")
			}
		})
	}
}

func TestChargenTipKeepsOriginalRectAndClosePresentation(t *testing.T) {
	for _, detail := range []bool{false, true} {
		t.Run(map[bool]string{false: "pre-create", true: "detailed"}[detail], func(t *testing.T) {
			setup := chargenTipSetup()
			setup.TipTextDetail = "detail tip"
			c := NewChargen(setup)
			if detail {
				c.Forward()
			}
			want := PreCreateTipRect
			if detail {
				want = ChargenTipRect
			}
			if got := c.TipPanel().Rect; got != want {
				t.Errorf("generator tip rect = %v, want %v", got, want)
			}
			a := newTestApp(t, appRows(3), okLoader(t))
			if err := a.OpenChargen(c, nil); err != nil {
				t.Fatal(err)
			}
			control := TipPanelCloseRect(c.TipPanel().Rect)
			p, _ := sampleInside(control)
			paint := func(in appInput) *image.RGBA {
				t.Helper()
				a.step(in, time.Unix(1, 0))
				pic, err := a.composeChargenScreen()
				if err != nil {
					t.Fatal(err)
				}
				return pic
			}
			idle := paint(appInput{CursorX: 0, CursorY: 479})
			hover := paint(appInput{CursorX: p.X, CursorY: p.Y})
			if sameTipControlPixels(idle, hover, control) {
				t.Fatal("generator Close hover did not reach the shared painter")
			}
			pressed := paint(appInput{CursorX: p.X, CursorY: p.Y, PrimaryPressed: true})
			if sameTipControlPixels(hover, pressed, control) {
				t.Fatal("generator Close press did not reach the shared painter")
			}
			if a.chargenPress.Holds() {
				t.Fatal("tip press armed a page control")
			}
			outside := paint(appInput{CursorX: 0, CursorY: 479})
			if !sameTipControlPixels(idle, outside, control) {
				t.Fatal("generator Close stayed sunken outside")
			}
			paint(appInput{CursorX: 0, CursorY: 479, PrimaryReleased: true})
			released := paint(appInput{CursorX: p.X, CursorY: p.Y})
			if !sameTipControlPixels(hover, released, control) {
				t.Fatal("generator outside release retained a pressed tip")
			}
		})
	}
}
