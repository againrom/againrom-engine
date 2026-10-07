package ui

import (
	"image"
	"testing"
	"time"
)

// A press on the painted merchant crosses no seam, plays no cue and leaves the
// message line as it was (TOWN-478). The shelf pick beside him does all three,
// which proves the route the merchant's press would take is open.
func TestApplicationTakesNoPressOnTheMerchant(t *testing.T) {
	town := &fakeShopTown{}
	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("showTown refused the shop town")
	}
	rec := &recordingPlayer{}
	a.SetAudio(rec, fixedUISoundBank())
	a.flow.msg = "kept line"
	now := time.Unix(1_700_000_000, 0)

	press := func(p image.Point) {
		t.Helper()
		a.step(appInput{CursorX: p.X, CursorY: p.Y, PrimaryPressed: true}, now)
		a.step(appInput{CursorX: p.X, CursorY: p.Y, PrimaryReleased: true}, now)
	}

	onMerchant := shopMerchantRect.Min.Add(image.Pt(20, 60))
	if c, ok := ShopControlAt(onMerchant); !ok || c.Kind != ShopControlMerchant {
		t.Fatalf("the point %v answers %+v, want the merchant", onMerchant, c)
	}
	press(onMerchant)
	if len(town.clicked) != 0 || len(rec.plays) != 0 || a.flow.msg != "kept line" {
		t.Fatalf("a press on the merchant crossed %v, played %d cue(s) and left the line %q; want none, none and the kept line",
			town.clicked, len(rec.plays), a.flow.msg)
	}

	onPick := shopShelfPickRects[0].Min.Add(image.Pt(4, 4))
	if c, ok := ShopControlAt(onPick); !ok || c.Kind != ShopControlShelfPick || c.Index != 0 {
		t.Fatalf("the point %v answers %+v, want shelf pick 0", onPick, c)
	}
	press(onPick)
	if len(town.clicked) != 1 || len(rec.plays) != 1 || a.flow.msg != "moved" {
		t.Fatalf("a press on a shelf pick crossed %v, played %d cue(s) and left the line %q; want one, one and the answer",
			town.clicked, len(rec.plays), a.flow.msg)
	}
}
