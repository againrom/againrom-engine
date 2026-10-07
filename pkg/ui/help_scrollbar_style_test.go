package ui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"reflect"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
)

type helpStyleCalls struct {
	requests, ticks, cadence, orders, affects, advances, attacks, grabs, stances, marches int
	openingRequests, openingTicks, heldCadence                                            int
	stopped                                                                               bool
}

func helpStyleBody() string {
	var body strings.Builder
	for i := 0; i < 60; i++ {
		fmt.Fprintf(&body, "Help %02d %s\r\n", i, strings.Repeat("x", 1+i%9))
	}
	return body.String()
}

func helpStyleApp(t *testing.T, frames []*image.RGBA, width, height int) (*App, *Viewer, *helpStyleCalls, MapOpener, time.Time) {
	t.Helper()
	calls := &helpStyleCalls{}
	open := func() (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
		v, err := NewViewer("help", grid(60, 60), &terrain.Tileset{})
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		v.SetFont(solidFont15())
		v.SetEntities([]MapEntity{{ID: 2, Cell: image.Pt(5, 5), Life: LifeAlive, Owner: 1}})
		tick := func() {
			calls.requests++
			if !calls.stopped {
				calls.ticks++
			}
		}
		order := func(uint32, int, int) { calls.orders++ }
		cadence := func(_ int, stopped, _, _ bool) { calls.cadence++; calls.stopped = stopped }
		affect := func(uint32, bool) { calls.affects++ }
		advance := func(...NoticeAction) (NoticeDest, string, MapOpener) { calls.advances++; return NoticeStay, "", nil }
		attack := func(uint32, uint32, uint32, int, int, bool) { calls.attacks++ }
		grab := func(uint32, int, int, bool) { calls.grabs++ }
		stance := func(uint32, bool) { calls.stances++ }
		march := func(uint32, bool, int, int) { calls.marches++ }
		return v, tick, order, cadence, affect, advance, attack, grab, stance, march, nil
	}
	a := newTestApp(t, nil, nil)
	w := AuthoredWords()
	w.HelpText = helpStyleBody()
	a.SetWords(w, solidFont15(), nil)
	a.SetCutsceneScrollArt(frames)
	if err := a.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	mapAtScaleOne(a)
	v := a.flow.viewer
	if width > 0 {
		v.Layout(width, height)
		a.winW, a.winH = width, height
	}
	v.sel = selection{2}
	*calls = helpStyleCalls{stopped: calls.stopped}
	now := time.Unix(600, 0)
	a.step(appInput{Help: true}, now)
	if !v.HelpOpen() || !a.flow.popupOpen() {
		t.Fatal("App F1 did not open help")
	}
	if lines, visible := v.HelpLines(); lines != 60 || visible != 12 {
		t.Fatalf("authored F1 lines/visible = %d/%d, want 60/12", lines, visible)
	}
	if first, last := v.HelpScroll(); first != 0 || last != 48 {
		t.Fatalf("F1 initial range = %d/%d, want 0/48", first, last)
	}
	l, body, ok := v.HelpPanel()
	if !ok || body != helpStyleBody() || l.Box != image.Rect(76, 60, 564, 420) || l.Text != image.Rect(40, 56, 422, 267) || l.Scrollbar != image.Rect(422, 56, 446, 267) || l.Button != image.Rect(196, 300, 292, 324) {
		t.Fatalf("F1 layout/body changed: %v / %v / %v / %v / %t", l.Box, l.Text, l.Scrollbar, l.Button, ok)
	}
	now = helpStyleQualifyPostOpen(t, a, calls, now)
	return a, v, calls, open, now
}

func helpStyleQualifyPostOpen(t *testing.T, a *App, calls *helpStyleCalls, now time.Time) time.Time {
	t.Helper()
	calls.openingRequests, calls.openingTicks = calls.requests, calls.ticks
	openingCadence, openingStopped := calls.cadence, calls.stopped
	now = now.Add(time.Millisecond)
	a.step(appInput{}, now)
	if calls.requests != calls.openingRequests+1 || calls.ticks != calls.openingTicks || calls.cadence == 0 || !calls.stopped {
		t.Fatalf("F1 first post-open idle did not consume a stopped clock request: opening %d/%d/%d/%t, idle %+v", calls.openingRequests, calls.openingTicks, openingCadence, openingStopped, *calls)
	}
	calls.heldCadence = calls.cadence
	t.Logf("F1 opening requests/ticks/cadence/stopped=%d/%d/%d/%t; post-open idle=%d/%d/%d/%t", calls.openingRequests, calls.openingTicks, openingCadence, openingStopped, calls.requests, calls.ticks, calls.cadence, calls.stopped)
	return now
}

func helpStylePicture(t *testing.T, v *Viewer) *image.RGBA {
	t.Helper()
	pix, _, _, ok := v.noticePresent()
	if !ok || pix == nil || pix.Bounds() != image.Rect(0, 0, 488, 360) {
		t.Fatal("F1 did not compose the native panel")
	}
	copy := image.NewRGBA(pix.Bounds())
	copy.Pix = append(copy.Pix[:0], pix.Pix...)
	return copy
}

func helpStylePictures(t *testing.T, a *App, v *Viewer, frames []*image.RGBA) (*image.RGBA, *image.RGBA) {
	t.Helper()
	a.SetCutsceneScrollArt(nil)
	fallback := helpStylePicture(t, v)
	a.SetCutsceneScrollArt(frames)
	return helpStylePicture(t, v), fallback
}

func assertHelpStyleFrozen(t *testing.T, calls *helpStyleCalls) {
	t.Helper()
	if !calls.stopped || calls.requests <= calls.openingRequests || calls.ticks != calls.openingTicks || calls.heldCadence == 0 || calls.cadence != calls.heldCadence || calls.orders != 0 || calls.affects != 0 || calls.advances != 0 || calls.attacks != 0 || calls.grabs != 0 || calls.stances != 0 || calls.marches != 0 {
		t.Fatalf("F1 crossed a post-open map effect seam: %+v", *calls)
	}
}

func assertHelpStyleOutsideBar(t *testing.T, pix, fallback *image.RGBA) {
	t.Helper()
	bar := image.Rect(422, 56, 446, 267)
	for y := 0; y < 360; y++ {
		for x := 0; x < 488; x++ {
			if !image.Pt(x, y).In(bar) && pix.RGBAAt(x, y) != fallback.RGBAAt(x, y) {
				t.Fatalf("F1 skin changed text, OK or frame outside bar at %d,%d", x, y)
			}
		}
	}
}

func assertHelpStyleSourcePixels(t *testing.T, pix, background *image.RGBA, frames []*image.RGBA, thumbY int) {
	t.Helper()
	counts := map[int]int{}
	for y := 56; y < 267; y++ {
		for x := 422; x < 446; x++ {
			want := color.RGBA{}
			if background != nil {
				want = background.RGBAAt(x, y)
			}
			index, sy := 19, (y-80)%24
			switch {
			case y < 80:
				index, sy = 18, y-56
			case y >= 243:
				index, sy = 20, y-243
			}
			under := frames[index].RGBAAt(x-422, sy)
			if under.A == 255 {
				want = under
			} else if under.A != 0 {
				t.Fatal("synthetic F1 source has unexpected partial alpha")
			}
			if y >= thumbY && y < thumbY+32 {
				thumb := frames[16].RGBAAt(x-422, (y-thumbY)*24/32)
				if thumb.A == 255 {
					want = thumb
					index = 16
				} else if thumb.A != 0 {
					t.Fatal("synthetic F1 thumb has unexpected partial alpha")
				}
			}
			if got := pix.RGBAAt(x, y); got != want {
				t.Fatalf("F1 source frame %d at %d,%d = %v, want %v (thumb %d..%d)", index, x, y, got, want, thumbY, thumbY+32)
			}
			counts[index]++
		}
	}
	for _, index := range []int{16, 18, 19, 20} {
		if counts[index] == 0 {
			t.Fatalf("F1 source frame %d checked no pixels", index)
		}
		if background == nil {
			want := 576
			if index == 16 {
				want = 768
			} else if index == 19 {
				want = 3144
			}
			if counts[index] != want {
				t.Fatalf("F1 source frame %d checked %d pixels, want %d", index, counts[index], want)
			}
		}
	}
}

func TestHelpScrollbarSkinKeepsHeldDragGeometryAndWorld(t *testing.T) {
	for _, size := range helpWindowSizes {
		t.Run(size.name, func(t *testing.T) {
			frames := loadScrollTestFrames()
			a, v, calls, _, now := helpStyleApp(t, frames, size.w, size.h)
			initial, fallback0 := helpStylePictures(t, a, v, frames)
			x, y := helpWindowPoint(t, v, image.Pt(434, 96))
			a.step(helpPressAt(x, y), now.Add(time.Millisecond))
			if first, _ := v.HelpScroll(); first != 0 {
				t.Fatalf("F1 initial thumb press moved to %d", first)
			}
			x, y = helpWindowPoint(t, v, image.Pt(434, 162))
			a.step(helpDown(x, y), now.Add(2*time.Millisecond))
			if first, last := v.HelpScroll(); first != 24 || last != 48 || v.help.hold != helpHoldThumb {
				t.Fatalf("held F1 midpoint before release = %d/%d/%v, want 24/48/thumb", first, last, v.help.hold)
			}
			middle, fallback24 := helpStylePictures(t, a, v, frames)
			x, y = helpWindowPoint(t, v, image.Pt(434, 227))
			a.step(helpDown(x, y), now.Add(3*time.Millisecond))
			if first, last := v.HelpScroll(); first != 48 || last != 48 {
				t.Fatalf("held F1 bottom before release = %d/%d, want 48/48", first, last)
			}
			bottom, fallback48 := helpStylePictures(t, a, v, frames)
			x, y = helpWindowPoint(t, v, image.Pt(120, 227))
			a.step(appInput{CursorX: x, CursorY: y, PrimaryReleased: true}, now.Add(4*time.Millisecond))
			x, y = helpWindowPoint(t, v, image.Pt(434, 96))
			a.step(appInput{CursorX: x, CursorY: y}, now.Add(5*time.Millisecond))
			a.step(helpDown(x, y), now.Add(6*time.Millisecond))
			if first, _ := v.HelpScroll(); first != 48 || v.help.hold != helpHoldNone || !v.HelpOpen() {
				t.Fatalf("outside release/next idle retained F1 drag: %d/%v/%t", first, v.help.hold, v.HelpOpen())
			}
			assertHelpStyleFrozen(t, calls)
			for _, stage := range []struct {
				name          string
				pix, fallback *image.RGBA
				thumbY        int
			}{{"initial", initial, fallback0, 80}, {"middle", middle, fallback24, 145}, {"bottom", bottom, fallback48, 211}} {
				t.Run(stage.name, func(t *testing.T) {
					assertHelpStyleOutsideBar(t, stage.pix, stage.fallback)
					assertHelpStyleSourcePixels(t, stage.pix, nil, frames, stage.thumbY)
				})
			}
			if got, want := middle.RGBAAt(434, 96), frames[19].RGBAAt(12, 16); got != want || got == initial.RGBAAt(434, 96) {
				t.Errorf("F1 old thumb did not restore track: %v, want %v", got, want)
			}
			a.step(appInput{Escape: true}, now.Add(7*time.Millisecond))
			a.step(appInput{}, now.Add(8*time.Millisecond))
			if v.HelpOpen() || calls.ticks <= calls.openingTicks || calls.advances != 0 {
				t.Fatalf("F1 clock/advance positive control = %+v / help %t", *calls, v.HelpOpen())
			}
			v.sel = selection{2}
			ox, oy := cellPoint(v, 10, 8)
			a.step(appInput{CursorX: ox, CursorY: oy, PrimaryPressed: true, PrimaryReleased: true, Viewer: Input{CursorX: ox, CursorY: oy}}, now.Add(9*time.Millisecond))
			if calls.orders != 1 || calls.attacks != 0 || calls.grabs != 0 || calls.advances != 0 {
				t.Fatalf("ordinary map input did not qualify F1 command spy: %+v", *calls)
			}
		})
	}
}

func TestHelpScrollbarSkinRefreshesCacheAndTravelsToNewViewer(t *testing.T) {
	frames := loadScrollTestFrames()
	a, v, calls, open, now := helpStyleApp(t, frames, 0, 0)
	shared := &MenuPanelArt{Minimap: image.NewRGBA(image.Rect(0, 0, 2, 2))}
	a.SetGameMenuArt(shared)
	saved := *shared
	a.SetCutsceneScrollArt(frames)
	x, y := helpWindowPoint(t, v, image.Pt(434, 96))
	a.step(helpPressAt(x, y), now.Add(time.Millisecond))
	x, y = helpWindowPoint(t, v, image.Pt(434, 162))
	a.step(helpDown(x, y), now.Add(2*time.Millisecond))
	a.step(appInput{CursorX: x, CursorY: y, PrimaryReleased: true}, now.Add(3*time.Millisecond))
	before := helpStylePicture(t, v)
	builds := v.noticeBuilds
	helpStylePicture(t, v)
	if v.noticeBuilds != builds {
		t.Fatal("unchanged F1 recomposed its cache")
	}
	replacement := loadScrollTestFrames()
	for _, index := range []int{16, 18, 19, 20} {
		for y := 0; y < 24; y++ {
			for x := 0; x < 24; x++ {
				c := replacement[index].RGBAAt(x, y)
				c.R++
				replacement[index].SetRGBA(x, y, c)
			}
		}
	}
	a.SetCutsceneScrollArt(replacement)
	after := helpStylePicture(t, v)
	if first, last := v.HelpScroll(); first != 24 || last != 48 || v.noticeBuilds != builds+1 || bytes.Equal(before.Pix, after.Pix) {
		t.Fatalf("F1 replacement did not refresh cache at same scroll: %d/%d builds %d/%d", first, last, v.noticeBuilds, builds)
	}
	assertHelpStyleOutsideBar(t, after, before)
	assertHelpStyleSourcePixels(t, after, nil, replacement, 145)
	layout, _, _ := v.HelpPanel()
	if !reflect.DeepEqual(*shared, saved) || a.flow.menuArt != shared || v.dialogFrame != shared || layout.Frame == nil || layout.Frame == shared || layout.Frame.Minimap != shared.Minimap {
		t.Fatal("F1 resolved frame did not preserve shared caller frame and artwork")
	}
	assertHelpStyleFrozen(t, calls)
	a.step(appInput{Escape: true}, now.Add(4*time.Millisecond))
	if err := a.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	if a.flow.viewer == v {
		t.Fatal("mission adoption reused old Viewer")
	}
	mapAtScaleOne(a)
	*calls = helpStyleCalls{stopped: calls.stopped}
	a.step(appInput{Help: true}, now.Add(5*time.Millisecond))
	helpStyleQualifyPostOpen(t, a, calls, now.Add(5*time.Millisecond))
	v = a.flow.viewer
	if lines, visible := v.HelpLines(); !v.HelpOpen() || lines != 60 || visible != 12 {
		t.Fatalf("new Viewer lost F1 body: %d/%d/%t", lines, visible, v.HelpOpen())
	}
	assertHelpStyleSourcePixels(t, helpStylePicture(t, v), nil, replacement, 80)
	assertHelpStyleFrozen(t, calls)
}

func TestHelpScrollbarSkinRequiresOnlyUsableFrames(t *testing.T) {
	for _, name := range []string{"21 frames", "unrelated nil"} {
		t.Run(name, func(t *testing.T) {
			frames := loadScrollTestFrames()
			if name == "21 frames" {
				frames = frames[:21]
			}
			a, v, calls, _, _ := helpStyleApp(t, frames, 0, 0)
			pix, fallback := helpStylePictures(t, a, v, frames)
			assertHelpStyleOutsideBar(t, pix, fallback)
			assertHelpStyleSourcePixels(t, pix, nil, frames, 80)
			assertHelpStyleFrozen(t, calls)
		})
	}
	for _, index := range []int{16, 18, 19, 20} {
		for _, damage := range []string{"missing", "nil", "empty"} {
			t.Run(fmt.Sprintf("frame-%d-%s", index, damage), func(t *testing.T) {
				frames := loadScrollTestFrames()
				switch damage {
				case "missing":
					frames = frames[:index]
				case "nil":
					frames[index] = nil
				case "empty":
					frames[index] = image.NewRGBA(image.Rectangle{})
				}
				a, v, calls, _, _ := helpStyleApp(t, nil, 0, 0)
				fallback := helpStylePicture(t, v)
				a.SetCutsceneScrollArt(frames)
				if got := helpStylePicture(t, v); !bytes.Equal(got.Pix, fallback.Pix) {
					t.Fatalf("F1 frame %d (%s) did not retain complete fallback", index, damage)
				}
				assertHelpStyleFrozen(t, calls)
			})
		}
	}
}

func TestHelpScrollbarSkinKeepsOpaqueBlackAndStructuralTransparency(t *testing.T) {
	frames := loadScrollTestFrames()
	for _, index := range []int{16, 18, 19, 20} {
		frames[index].SetRGBA(12, 10, color.RGBA{})
		frames[index].SetRGBA(13, 10, color.RGBA{0, 0, 0, 255})
	}
	frames[16].SetRGBA(12, 9, color.RGBA{})
	frames[16].SetRGBA(13, 9, color.RGBA{0, 0, 0, 255})
	a, v, calls, _, _ := helpStyleApp(t, frames, 0, 0)
	pix, fallback := helpStylePictures(t, a, v, frames)
	l, body, _ := v.HelpPanel()
	l.Scrollbar = image.Rectangle{}
	background := RenderNotice(l, v.font, body, nil)
	assertHelpStyleOutsideBar(t, pix, fallback)
	assertHelpStyleSourcePixels(t, pix, background, frames, 80)
	for _, p := range []image.Point{{435, 66}, {435, 92}, {435, 138}, {435, 253}} {
		if got := pix.RGBAAt(p.X, p.Y); got != (color.RGBA{0, 0, 0, 255}) {
			t.Fatalf("F1 opaque RGB black became a hole at %v: %v", p, got)
		}
	}
	if got, want := pix.RGBAAt(434, 92), frames[19].RGBAAt(12, 12); got != want {
		t.Fatalf("F1 structural thumb hole did not reveal track: %v, want %v", got, want)
	}
	if got, want := pix.RGBAAt(434, 138), background.RGBAAt(434, 138); got != want {
		t.Fatalf("F1 structural track hole did not reveal panel: %v, want %v", got, want)
	}
	assertHelpStyleFrozen(t, calls)
}

func helpStyleFrame(ink color.RGBA) (*MenuPanelArt, *MenuPanelArt) {
	art, snapshot := &MenuPanelArt{}, &MenuPanelArt{}
	for i := range art.Pieces {
		pic := image.NewRGBA(image.Rect(0, 0, 48, 48))
		for y := 0; y < 48; y++ {
			for x := 0; x < 48; x++ {
				pic.SetRGBA(x, y, ink)
			}
		}
		art.Pieces[i] = pic
		frozen := image.NewRGBA(pic.Bounds())
		copy(frozen.Pix, pic.Pix)
		snapshot.Pieces[i] = frozen
	}
	return art, snapshot
}

func helpStyleHeldMiddle(t *testing.T, a *App, v *Viewer, now time.Time) {
	t.Helper()
	x, y := helpWindowPoint(t, v, image.Pt(434, 96))
	a.step(helpPressAt(x, y), now.Add(time.Millisecond))
	x, y = helpWindowPoint(t, v, image.Pt(434, 162))
	a.step(helpDown(x, y), now.Add(2*time.Millisecond))
	assertHelpStyleHeldMiddle(t, v)
}

func assertHelpStyleHeldMiddle(t *testing.T, v *Viewer) {
	t.Helper()
	_, body, ok := v.HelpPanel()
	first, last := v.HelpScroll()
	if !ok || body != helpStyleBody() || first != 24 || last != 48 || v.help.hold != helpHoldThumb || v.help.grab != 16 {
		t.Fatalf("F1 setters changed body, position or held capture: %t/%d/%d/%v/%d", ok, first, last, v.help.hold, v.help.grab)
	}
}

func TestHelpScrollbarSkinSurvivesFrameSetterOrdersAndReplacement(t *testing.T) {
	for _, order := range []string{"scroll before frame", "frame before scroll"} {
		t.Run(order, func(t *testing.T) {
			a, v, calls, _, now := helpStyleApp(t, nil, 0, 0)
			helpStyleHeldMiddle(t, a, v, now)
			helpStylePicture(t, v)
			builds := v.noticeBuilds
			frames := loadScrollTestFrames()
			firstInk, secondInk := color.RGBA{93, 31, 48, 255}, color.RGBA{41, 86, 102, 255}
			firstArt, firstSnapshot := helpStyleFrame(firstInk)
			secondArt, secondSnapshot := helpStyleFrame(secondInk)
			if order == "scroll before frame" {
				a.SetCutsceneScrollArt(frames)
				a.SetGameMenuArt(firstArt)
			} else {
				a.SetGameMenuArt(firstArt)
				a.SetCutsceneScrollArt(frames)
			}
			for index, supplied := range []*MenuPanelArt{firstArt, secondArt} {
				if index == 1 {
					a.SetGameMenuArt(supplied)
				}
				pix := helpStylePicture(t, v)
				assertHelpStyleHeldMiddle(t, v)
				if v.noticeBuilds != builds+1 {
					t.Fatalf("F1 frame setter did not refresh cached pixels: builds %d, want %d", v.noticeBuilds, builds+1)
				}
				builds = v.noticeBuilds
				ink := firstInk
				if index == 1 {
					ink = secondInk
				}
				if got := pix.RGBAAt(10, 10); got != ink {
					t.Fatalf("F1 frame %d did not reach cached panel: %v, want %v", index, got, ink)
				}
				layout, body, _ := v.HelpPanel()
				layout.Frame = supplied
				assertHelpStyleOutsideBar(t, pix, RenderNotice(layout, v.font, body, nil))
				assertHelpStyleSourcePixels(t, pix, nil, frames, 145)
				if !reflect.DeepEqual(*firstArt, *firstSnapshot) || !reflect.DeepEqual(*secondArt, *secondSnapshot) {
					t.Fatal("F1 setters mutated supplied frame or source pixels")
				}
				assertHelpStyleFrozen(t, calls)
			}
			x, y := helpWindowPoint(t, v, image.Pt(434, 227))
			a.step(helpDown(x, y), now.Add(3*time.Millisecond))
			if first, _ := v.HelpScroll(); first != 48 {
				t.Fatalf("F1 frame replacement broke held continuation: %d, want 48", first)
			}
			assertHelpStyleFrozen(t, calls)
		})
	}
}

func TestHelpScrollbarSkinClearsIndependentlyFromFrame(t *testing.T) {
	a, v, calls, _, now := helpStyleApp(t, nil, 0, 0)
	helpStyleHeldMiddle(t, a, v, now)
	nilFallback := helpStylePicture(t, v)
	ink := color.RGBA{93, 31, 48, 255}
	art, snapshot := helpStyleFrame(ink)
	a.SetGameMenuArt(art)
	frameFallback := helpStylePicture(t, v)
	frames := loadScrollTestFrames()
	a.SetCutsceneScrollArt(frames)
	skinned := helpStylePicture(t, v)
	assertHelpStyleSourcePixels(t, skinned, nil, frames, 145)
	assertHelpStyleOutsideBar(t, skinned, frameFallback)
	a.SetCutsceneScrollArt(nil)
	cleared := helpStylePicture(t, v)
	assertHelpStyleHeldMiddle(t, v)
	if !bytes.Equal(cleared.Pix, frameFallback.Pix) || cleared.RGBAAt(10, 10) != ink {
		t.Fatal("clearing F1 scroll art changed frame, body or fallback pixels")
	}
	a.SetCutsceneScrollArt(frames)
	helpStylePicture(t, v)
	a.SetGameMenuArt(nil)
	nilFrame := helpStylePicture(t, v)
	assertHelpStyleHeldMiddle(t, v)
	assertHelpStyleOutsideBar(t, nilFrame, nilFallback)
	assertHelpStyleSourcePixels(t, nilFrame, nil, frames, 145)
	a.SetCutsceneScrollArt(nil)
	if cleared := helpStylePicture(t, v); !bytes.Equal(cleared.Pix, nilFallback.Pix) {
		t.Fatal("nil frame plus cleared F1 scroll art lost original fallback panel")
	}
	assertHelpStyleHeldMiddle(t, v)
	if !reflect.DeepEqual(*art, *snapshot) {
		t.Fatal("clearing or nil frame setter mutated supplied artwork")
	}
	x, y := helpWindowPoint(t, v, image.Pt(434, 227))
	a.step(helpDown(x, y), now.Add(3*time.Millisecond))
	if first, _ := v.HelpScroll(); first != 48 {
		t.Fatalf("F1 cleared/nil frame broke held continuation: %d, want 48", first)
	}
	assertHelpStyleFrozen(t, calls)
}

func TestHelpScrollbarSkinKeepsOrdinaryNoticeAndPortraitPixels(t *testing.T) {
	for _, frameName := range []string{"nil frame", "supplied frame"} {
		for _, portrait := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s-portrait-%t", frameName, portrait), func(t *testing.T) {
				a, v, _, _, _ := helpStyleApp(t, nil, 0, 0)
				var art, snapshot *MenuPanelArt
				if frameName == "supplied frame" {
					art, snapshot = helpStyleFrame(color.RGBA{93, 31, 48, 255})
				}
				a.SetGameMenuArt(art)
				v.ClearNotice()
				dialogue := Dialogue{Text: "Ordinary notice", Portrait: portrait, Speaks: true}
				v.SetDialogue(dialogue)
				omitted, _, _, ok := v.noticePresent()
				if !ok || omitted == nil {
					t.Fatal("ordinary notice omission control did not compose")
				}
				omittedPixels := append([]byte(nil), omitted.Pix...)
				if portrait {
					face := image.NewRGBA(image.Rect(0, 0, 160, 240))
					for y := 0; y < 240; y++ {
						for x := 0; x < 160; x++ {
							face.SetRGBA(x, y, color.RGBA{64, 80, 96, 255})
						}
					}
					dialogue.Face = face
					v.PageDialogue(dialogue)
				}
				before, _, _, ok := v.noticePresent()
				if !ok || before == nil || v.HelpOpen() {
					t.Fatal("ordinary dialogue fixture still uses Help")
				}
				bounds, pixels := before.Bounds(), append([]byte(nil), before.Pix...)
				if portrait && bytes.Equal(pixels, omittedPixels) {
					t.Fatal("ordinary portrait omission control is insensitive")
				}
				for _, frames := range [][]*image.RGBA{loadScrollTestFrames(), loadScrollTestFrames()[:21], nil} {
					a.SetCutsceneScrollArt(frames)
					after, _, _, ok := v.noticePresent()
					if !ok || after == nil || after.Bounds() != bounds || !bytes.Equal(after.Pix, pixels) {
						t.Fatal("Help scroll resource changed ordinary notice or portrait pixels")
					}
					if a.flow.menuArt != art || v.dialogFrame != art || v.noticeLayout().Frame != art {
						t.Fatal("Help scroll resource replaced actual non-Help frame")
					}
					if art != nil && !reflect.DeepEqual(*art, *snapshot) {
						t.Fatal("Help scroll resource mutated supplied ordinary frame")
					}
				}
			})
		}
	}
}

func TestHelpScrollbarSkinKeepsNilFrameSaveConfirmationPixels(t *testing.T) {
	spy := &saveDialogSpy{paths: []string{"saves/save.sav"}, existing: []string{"saves/save.sav"}}
	a := newSaveDialogApp(t, spy, ScreenTown)
	a.SetWords(AuthoredWords(), chargenTestFont(), nil)
	if err := a.HeadlessSaveEdit("", "save", SaveSAV); err != nil {
		t.Fatal(err)
	}
	mustSaveAction(t, a, "save")
	if a.flow.menuArt != nil || a.flow.saveDialog.prepared == nil || len(spy.requests) != 1 || len(spy.commits) != 0 {
		t.Fatal("nil-frame Save confirmation fixture did not prepare without writing")
	}
	before, note, err := a.HeadlessFrame()
	if err != nil || note != "" || before == nil {
		t.Fatal("nil-frame Save confirmation did not compose", note, err)
	}
	bounds, pixels := before.Bounds(), append([]byte(nil), before.Pix...)
	state, _ := a.HeadlessSaveState()
	for _, frames := range [][]*image.RGBA{loadScrollTestFrames(), loadScrollTestFrames()[:21], nil} {
		a.SetCutsceneScrollArt(frames)
		after, note, err := a.HeadlessFrame()
		if err != nil || note != "" || after == nil || after.Bounds() != bounds || !bytes.Equal(after.Pix, pixels) {
			t.Fatal("Help scroll resource changed nil-frame Save confirmation pixels", note, err)
		}
		got, _ := a.HeadlessSaveState()
		if a.flow.menuArt != nil || !reflect.DeepEqual(got, state) || len(spy.requests) != 1 || len(spy.commits) != 0 {
			t.Fatal("Help scroll resource changed nil-frame Save target or prepared output")
		}
	}
}
