package ui

import (
	"image"
	"testing"
	"time"
)

// The map choice page's control rectangles, written out so the tests do not
// read the geometry they check from the code under test.
var (
	testMapOKRect     = image.Rect(154, 338, 250, 362)
	testMapCancelRect = image.Rect(398, 338, 494, 362)
)

func clickMapChoiceRect(t *testing.T, a *App, r image.Rectangle, now time.Time) {
	t.Helper()
	x, y, ok := a.nativeFrameToWindow(r.Min.Add(r.Size().Div(2)))
	if !ok {
		t.Fatalf("map choice button at %v is not in the window", r)
	}
	a.step(appInput{CursorX: x, CursorY: y}, now)
	clickAt(a, now, image.Pt(x, y))
}

func mapChoiceSampleItems() []MapChoiceItem {
	return []MapChoiceItem{
		{Source: "alpha.alm", Row: PickerRow{Text: "alpha.alm - Alpha", Choosable: true}},
		{Source: "beta.alm", Row: PickerRow{Text: "beta.alm - Beta", Choosable: true}},
		{Source: "gamma.alm", Row: PickerRow{Text: "gamma.alm - Gamma", Choosable: true}},
	}
}

func TestMapChoiceSelectOkAndCancel(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	apps := multiplayerApps(t)
	for _, name := range []string{"first", "second"} {
		t.Run(name, func(t *testing.T) {
			a := apps[name]
			a.SetWords(AuthoredWords(), chargenTestFont(), nil)
			a.SetMapChoiceWords(MapChoiceWords{Title: "Select Map", OK: "OK", Cancel: "Cancel", Empty: "No maps"})

			// Choosing a map with Down + Ok button returns it to the caller.
			var got MapChoiceResult
			called := 0
			open := func(items []MapChoiceItem, stale MapChoiceResult) {
				got, called = stale, 0
				a.OpenMapChoice(items, ScreenMenu, func(res MapChoiceResult) {
					got = res
					called++
				})
			}
			open(mapChoiceSampleItems(), MapChoiceResult{})
			if rows := a.HeadlessRows(); len(rows) != 3 || rows[1].Text != "beta.alm - Beta" {
				t.Fatalf("rows = %+v, want 3 sample rows", rows)
			}
			if err := a.HeadlessKey("down"); err != nil {
				t.Fatal(err)
			}
			clickMapChoiceRect(t, a, testMapOKRect, now)
			if a.Screen() != ScreenMenu || called != 1 || !got.OK || got.Chosen != "beta.alm" {
				t.Fatalf("Ok returned screen=%v called=%d result=%+v, want ScreenMenu beta.alm", a.Screen(), called, got)
			}

			// HeadlessActivate picks a row by text and confirms with Enter.
			open(mapChoiceSampleItems(), MapChoiceResult{})
			if err := a.HeadlessActivate("gamma.alm - Gamma"); err != nil {
				t.Fatal(err)
			}
			if a.Screen() != ScreenMenu || called != 1 || !got.OK || got.Chosen != "gamma.alm" {
				t.Fatalf("Enter returned screen=%v called=%d result=%+v, want ScreenMenu gamma.alm", a.Screen(), called, got)
			}

			// Cancel button returns nothing chosen.
			open(mapChoiceSampleItems(), MapChoiceResult{Chosen: "stale", OK: true})
			clickMapChoiceRect(t, a, testMapCancelRect, now)
			if a.Screen() != ScreenMenu || called != 1 || got.OK || got.Chosen != "" {
				t.Fatalf("Cancel returned screen=%v called=%d result=%+v, want ScreenMenu none", a.Screen(), called, got)
			}

			// Escape also cancels with nothing chosen.
			open(mapChoiceSampleItems(), MapChoiceResult{Chosen: "stale", OK: true})
			if err := a.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
			if a.Screen() != ScreenMenu || called != 1 || got.OK || got.Chosen != "" {
				t.Fatalf("Escape returned screen=%v called=%d result=%+v, want ScreenMenu none", a.Screen(), called, got)
			}

			// Empty list disables Ok and keeps the page open until Cancel.
			open(nil, MapChoiceResult{})
			shown := a.Screen()
			clickMapChoiceRect(t, a, testMapOKRect, now)
			if err := a.HeadlessKey("enter"); err != nil {
				t.Fatal(err)
			}
			if a.Screen() != shown || called != 0 {
				t.Fatalf("Ok on empty list left screen=%v called=%d, want %v and 0", a.Screen(), called, shown)
			}
			clickMapChoiceRect(t, a, testMapCancelRect, now)
			if a.Screen() != ScreenMenu || called != 1 || got.OK {
				t.Fatalf("Cancel on empty list returned screen=%v called=%d result=%+v", a.Screen(), called, got)
			}
		})
	}
}

// TestMapChoiceDrawsEachCaptionAtItsOwnButton renders the page under swapped
// and single-changed captions and compares the pixels of the two button
// rectangles: each caption must change its own button and no other, and the
// button's disabled look must be OK's alone.
func TestMapChoiceDrawsEachCaptionAtItsOwnButton(t *testing.T) {
	render := func(items []MapChoiceItem, ok, cancel string) *image.RGBA {
		t.Helper()
		a := newTestApp(t, appRows(1), okLoader(t))
		a.SetWords(AuthoredWords(), chargenTestFont(), nil)
		a.SetMapChoiceWords(MapChoiceWords{Title: "Select Map", OK: ok, Cancel: cancel, Empty: "No maps"})
		a.OpenMapChoice(items, ScreenMenu, nil)
		pix, err := a.composeScreen()
		if err != nil {
			t.Fatal(err)
		}
		return pix
	}
	diff := func(a, b *image.RGBA, r image.Rectangle) int {
		n := 0
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
					n++
				}
			}
		}
		return n
	}
	items := mapChoiceSampleItems()
	base := render(items, "Okay", "No")
	swapped := render(items, "No", "Okay")
	if diff(base, swapped, testMapOKRect) == 0 || diff(base, swapped, testMapCancelRect) == 0 {
		t.Errorf("swapping the captions changed %d pixels at OK and %d at Cancel, want both changed",
			diff(base, swapped, testMapOKRect), diff(base, swapped, testMapCancelRect))
	}
	okOnly := render(items, "Okay and more", "No")
	if diff(base, okOnly, testMapOKRect) == 0 || diff(base, okOnly, testMapCancelRect) != 0 {
		t.Errorf("changing the OK caption changed %d pixels at OK and %d at Cancel, want OK only",
			diff(base, okOnly, testMapOKRect), diff(base, okOnly, testMapCancelRect))
	}
	noOnly := render(items, "Okay", "No and more")
	if diff(base, noOnly, testMapCancelRect) == 0 || diff(base, noOnly, testMapOKRect) != 0 {
		t.Errorf("changing the Cancel caption changed %d pixels at Cancel and %d at OK, want Cancel only",
			diff(base, noOnly, testMapCancelRect), diff(base, noOnly, testMapOKRect))
	}
	empty := render(nil, "Okay", "No")
	if diff(base, empty, testMapOKRect) == 0 || diff(base, empty, testMapCancelRect) != 0 {
		t.Errorf("an empty list changed %d pixels at OK and %d at Cancel, want OK only (disabled)",
			diff(base, empty, testMapOKRect), diff(base, empty, testMapCancelRect))
	}
}
