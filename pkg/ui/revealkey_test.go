package ui

import (
	"bytes"
	"image"
	"image/color"
	"io"
	"reflect"
	"testing"
	"time"
)

func TestShiftF4HeadlessRevealsMinimapWithoutSaving(t *testing.T) {
	h := newHaltFix(t, haltOpts{})
	h.frame(cadenceInput(true, false, false))
	h.a.headlessClock = haltAt(h.n)
	plane := make([]byte, 60*60)
	plane[49*60+40] = FogExplored
	h.v.SetFog(plane, 60, 60)
	beforePlane := append([]byte(nil), plane...)
	h.v.minimapColours = make([]color.RGBA, 60*60)
	for i := range h.v.minimapColours {
		h.v.minimapColours[i] = color.RGBA{R: 200, G: 100, B: 40, A: 255}
	}
	quick, ordinary, autosaves, polls := 0, 0, 0, 0
	h.a.SetQuickSaveControls(QuickSaveControls{Save: func(bool) error { quick++; return nil }})
	h.a.SetSaveSeams(func(bool) (string, error) { ordinary++; return "unused.sav", nil }, nil, nil)
	frozen := time.Unix(100, 0)
	due := frozen.Add(time.Minute)
	h.a.SetTimedAutosaveControls(TimedAutosaveControls{Poll: func(_ *Viewer, onMap, ready bool) error {
		polls++
		if onMap && ready && !frozen.Before(due) {
			autosaves++
		}
		return nil
	}})
	beforeApp, world := h.v.SaveApplication(), h.w.world
	picture := func() *image.RGBA {
		t.Helper()
		pic, err := h.a.HeadlessMinimap()
		if err != nil {
			t.Fatal(err)
		}
		if pic.Bounds() != image.Rect(0, 0, 158, 158) {
			t.Fatalf("fixture minimap bounds = %v", pic.Bounds())
		}
		return pic
	}
	checkPixel := func(pic *image.RGBA, x int, want color.RGBA) {
		t.Helper()
		if got := pic.RGBAAt(x, 130); got != want {
			t.Fatalf("minimap pixel (%d,130) = %v, want literal %v", x, got, want)
		}
	}
	off := picture()
	checkPixel(off, 130, color.RGBA{A: 255})
	checkPixel(off, 106, color.RGBA{R: 100, G: 50, B: 20, A: 255})
	if err := h.a.HeadlessKey("shift-f4"); err != nil {
		t.Fatal(err)
	}
	if !h.v.FogRevealed() {
		t.Fatal("Shift+F4 did not reveal the map")
	}
	on := picture()
	checkPixel(on, 130, color.RGBA{R: 200, G: 100, B: 40, A: 255})
	checkPixel(on, 106, color.RGBA{R: 200, G: 100, B: 40, A: 255})
	for range 3 {
		in := haltNeutral()
		in.AnyHeld, in.ShiftHeld, in.Viewer.Shift = true, true, true
		h.frame(in)
		if !h.v.FogRevealed() {
			t.Fatal("held levels toggled without an F4 edge")
		}
	}
	h.a.headlessClock = haltAt(h.n)
	if err := h.a.HeadlessKey("shift-f4"); err != nil {
		t.Fatal(err)
	}
	restored := picture()
	if h.v.FogRevealed() || !bytes.Equal(off.Pix, restored.Pix) {
		t.Fatal("second Shift+F4 did not restore the minimap bytes")
	}
	if !bytes.Equal(beforePlane, plane) || !bytes.Equal(beforePlane, h.v.fogPlane) || h.w.world != world || !reflect.DeepEqual(beforeApp, h.v.SaveApplication()) || h.a.Screen() != ScreenMap || h.a.flow.viewer != h.v {
		t.Fatal("reveal changed the plane, world, application or screen")
	}
	if quick != 0 || ordinary != 0 || autosaves != 0 || polls == 0 {
		t.Fatalf("reveal save counts: quick=%d ordinary=%d timed=%d polls=%d", quick, ordinary, autosaves, polls)
	}
	if err := h.a.HeadlessKey("f4"); err != nil || quick != 1 || h.v.FogRevealed() {
		t.Fatalf("bare F4 loss control: quick=%d reveal=%v err=%v", quick, h.v.FogRevealed(), err)
	}
}

func TestShiftF4RespectsMapInputOwners(t *testing.T) {
	for _, owner := range []string{"focus", "notice", "notice closes", "popup", "town", "menu", "text", "save", "load", "picker", "documents"} {
		t.Run(owner, func(t *testing.T) {
			h := newHaltFix(t, haltOpts{})
			calls := 0
			h.a.SetQuickSaveControls(QuickSaveControls{Save: func(bool) error { calls++; return nil }})
			in := haltNeutral()
			in.Reveal, in.ShiftHeld, in.Viewer.Shift = true, true, true
			switch owner {
			case "focus":
				in.Unfocused = true
			case "notice", "notice closes":
				h.v.SetNotice("notice", NoticeDialogue)
				in.Enter = owner == "notice closes"
			case "popup":
				h.v.menuUp = true
			case "town":
				h.a.SetTown(&stubTown{rows: []TownRow{{Text: "town", Choosable: true}}})
				if !h.a.flow.showTown("") {
					t.Fatal("fixture town did not open")
				}
			case "menu":
				h.a.flow.screen = ScreenMenu
			case "text":
				h.a.flow.screen = ScreenChargen
			case "save":
				h.a.flow.screen = ScreenSave
			case "load":
				h.a.flow.screen = ScreenLoad
			case "picker":
				h.a.flow.screen = ScreenPicker
			case "documents":
				h.a.flow.screen = ScreenDocuments
			}
			before := h.a.Screen()
			h.frame(in)
			if h.v.FogRevealed() || calls != 0 || h.a.Screen() != before {
				t.Fatalf("%s admitted reveal/save/navigation", owner)
			}
			if owner == "notice closes" {
				if h.v.NoticeOpen() || h.w.advances != 1 {
					t.Fatal("notice dismissal did not reach its production seam")
				}
				in.Enter = false
				h.frame(in)
				if !h.v.FogRevealed() {
					t.Fatal("a fresh edge after dismissal remained blocked")
				}
			}
		})
	}
}

func TestShiftF4CutsceneSkipAndHeldDrainOwnTheKey(t *testing.T) {
	h := newHaltFix(t, haltOpts{})
	calls := 0
	h.a.SetQuickSaveControls(QuickSaveControls{Save: func(bool) error { calls++; return nil }})
	r, w := io.Pipe()
	t.Cleanup(func() { h.a.StopCutscene(); w.Close() })
	h.a.SetCutscenes(&testCutsceneSource{stream: r})
	if !h.a.PlayCutscene("fixture") {
		t.Fatal("fixture cutscene did not open")
	}
	ticks, world := h.w.ticks, h.w.world
	in := haltNeutral()
	in.AnyKey, in.AnyHeld, in.Reveal, in.ShiftHeld, in.Viewer.Shift = true, true, true, true, true
	h.frame(in)
	if h.a.Screen() != ScreenMap || !h.a.cutsceneDrain {
		t.Fatal("skip did not return to the draining map")
	}
	in.AnyKey = false
	for range 3 {
		h.frame(in)
		if !h.a.cutsceneDrain {
			t.Fatal("held input left the drain")
		}
	}
	if h.v.FogRevealed() || calls != 0 || h.w.ticks != ticks || h.w.world != world {
		t.Fatal("cutscene skip or held drain reached reveal/save/map tick")
	}
	h.frame(haltNeutral())
	if h.a.cutsceneDrain || h.v.FogRevealed() || h.w.ticks != ticks {
		t.Fatal("draining release reached the map")
	}
	h.a.headlessClock = haltAt(h.n)
	if err := h.a.HeadlessKey("shift-f4"); err != nil || !h.v.FogRevealed() || calls != 0 {
		t.Fatalf("fresh edge after drain: reveal=%v calls=%d err=%v", h.v.FogRevealed(), calls, err)
	}
}

func TestShiftF4PreservesTheAccompanyingPointerRelease(t *testing.T) {
	for _, suppressed := range []bool{false, true} {
		t.Run(map[bool]string{false: "ordinary release", true: "already suppressed release"}[suppressed], func(t *testing.T) {
			control, reveal := newPopupFix(t, haltOpts{}), newPopupFix(t, haltOpts{})
			for _, f := range []*popupFix{control, reveal} {
				f.frame(cadenceInput(true, false, false))
				f.selectA()
				f.s.orders = nil
				press := f.at(popupEmptyCol, popupEmptyRow)
				press.PrimaryPressed, press.Viewer.PrimaryDown = true, true
				press.ShiftHeld, press.Viewer.Shift = true, true
				f.frame(press)
				if !f.v.held {
					t.Fatal("fixture pointer gesture did not arm")
				}
				f.a.suppressPrimaryRelease = suppressed
			}
			world := reveal.w.world
			for _, f := range []*popupFix{control, reveal} {
				in := f.at(popupEmptyCol, popupEmptyRow)
				in.PrimaryReleased, in.ShiftHeld, in.Viewer.Shift = true, true, true
				in.Reveal = f == reveal
				f.frame(in)
			}
			if !reflect.DeepEqual(control.s.orders, reveal.s.orders) || !reflect.DeepEqual(control.v.SaveApplication(), reveal.v.SaveApplication()) || control.a.suppressPrimaryRelease != reveal.a.suppressPrimaryRelease || reveal.a.suppressPrimaryRelease || reveal.w.world != world || !reveal.v.FogRevealed() {
				t.Fatal("reveal changed the accompanying release's map owner")
			}
			if !suppressed && len(reveal.s.orders) == 0 {
				t.Fatal("ordinary release control issued no order")
			}
			if suppressed && len(reveal.s.orders) != 0 {
				t.Fatal("previously suppressed release issued an order")
			}
		})
	}
}
