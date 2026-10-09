package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"testing"
	"time"
)

func TestTownAdvertisedKeyboardControlsReachTheSameActions(t *testing.T) {
	a := newTestApp(t, nil, nil)
	a.Layout(640, 480)
	shop := &fakeShopTown{}
	a.SetTown(shop)
	a.flow.showTown("")
	for _, key := range []string{"tab", "b", "q"} {
		if err := a.HeadlessKey(key); err != nil {
			t.Fatal(err)
		}
	}
	want := []ShopControlKind{ShopControlCharacterMode, ShopControlBook, ShopControlBook}
	if len(shop.controls) != len(want) {
		t.Fatalf("shop keys: %+v", shop.controls)
	}
	for i, c := range shop.controls {
		if c.Kind != want[i] {
			t.Fatalf("key %d: %v", i, c)
		}
	}
	town := &fakeStatsSurfaceTown{}
	a.SetTown(town)
	a.flow.showTown("")
	for _, key := range []string{"tab", "b", "q"} {
		if err := a.HeadlessKey(key); err != nil {
			t.Fatal(err)
		}
	}
	if len(town.surfaceClicks) != 3 || town.surfaceClicks[0].Kind != TownSurfaceControlMode || town.surfaceClicks[1].Kind != TownSurfaceControlBook || town.surfaceClicks[2].Kind != TownSurfaceControlBook {
		t.Fatalf("town keys: %+v", town.surfaceClicks)
	}
}

func TestShopInventoryArrowsDispatchWithoutTouchingThePack(t *testing.T) {
	a := newTestApp(t, nil, nil)
	a.Layout(640, 480)
	shop := &fakeShopTown{}
	a.SetTown(shop)
	a.flow.showTown("")
	for _, r := range []image.Rectangle{shopPackLeftRect, shopPackRightRect} {
		p := r.Min.Add(image.Pt(10, 40))
		a.step(appInput{CursorX: p.X, CursorY: p.Y, PrimaryPressed: true}, time.Now())
		a.step(appInput{CursorX: p.X, CursorY: p.Y, PrimaryReleased: true}, time.Now())
	}
	if len(shop.controls) != 2 || shop.controls[0].Kind != ShopControlPackLeft || shop.controls[1].Kind != ShopControlPackRight || len(shop.dragged) != 0 {
		t.Fatalf("arrows: %+v", shop.controls)
	}
}

func TestTooltipOmitsEmptySeparators(t *testing.T) {
	font := messageFont()
	bounds := image.Rect(0, 0, 640, 480)
	p := image.Pt(100, 300)
	a, aa, ok := tooltipPicture(tooltipTarget{tooltipItem, "item", []string{"Sword", "#Damage 4", " ", "#", "#Value 9"}, font}, p, bounds, nil)
	b, bb, bok := tooltipPicture(tooltipTarget{tooltipItem, "item", []string{"Sword", "Damage 4", "Value 9"}, font}, p, bounds, nil)
	if !ok || !bok || aa != bb || a.Bounds() != b.Bounds() || !bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("empty separators changed the item popup")
	}
}

func TestTownUpperPanelKeyedEdgeKeepsRoomArt(t *testing.T) {
	room := image.NewRGBA(image.Rect(0, 0, 480, 480))
	ink := color.RGBA{R: 30, G: 80, B: 120, A: 255}
	draw.Draw(room, room.Bounds(), &image.Uniform{C: ink}, image.Point{}, draw.Src)
	for _, kind := range []TownSurfaceKind{TownSurfaceTavern, TownSurfaceSchool} {
		v := TownSurfaceView{Kind: kind}
		if kind == TownSurfaceTavern {
			v.TavernArt = &TownTavernArt{Center: room, CommandUpper: image.NewRGBA(image.Rect(0, 0, 176, 238))}
			v.Buttons = make([]TownSurfaceButton, 4)
		} else {
			v.SchoolArt = &TownSchoolArt{Background: room, Upper: image.NewRGBA(image.Rect(0, 0, 160, 238)), UpperSeam: image.NewRGBA(image.Rect(0, 0, 16, 238))}
		}
		if got := ComposeTownSurface(v).RGBAAt(465, 90); got != ink {
			t.Fatalf("kind %v keyed edge=%v", kind, got)
		}
	}
}

func TestOriginalNoticeFrameHasNoDiagnosticRectangles(t *testing.T) {
	art := &DialogFrame{Portrait: image.NewRGBA(image.Rect(0, 0, 88, 108))}
	for i := range art.Pieces {
		art.Pieces[i] = image.NewRGBA(image.Rect(0, 0, 48, 48))
	}
	l := AuthoredDialogueLayout().WithPortrait(true)
	l.Frame = art
	pic := RenderNotice(l, panelFont(), "", nil)
	if pic.RGBAAt(0, 0).A != 0 {
		t.Fatal("authored outer rectangle remains behind the original frame")
	}
	if got := pic.RGBAAt(l.Portrait.Min.X, l.Portrait.Min.Y); got.A != 0 {
		t.Fatal("authored portrait rectangle remains behind the original border", got)
	}
}
