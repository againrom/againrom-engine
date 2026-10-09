package ui

import (
	"image"
	"testing"
	"time"
)

type pointerDialogueTown struct {
	fakeSurfaceDialogueTown
	revision uint64
}

func (f *pointerDialogueTown) TownDialogueRevision() uint64 { return f.revision }
func (f *pointerDialogueTown) AdvanceTownDialogue() TownAction {
	f.advances++
	f.revision++
	return TownAction{}
}

type dialoguePointerFixture struct {
	app      *App
	button   image.Rectangle
	body     image.Point
	advances func() int
	replace  func()
	close    func()
	reopen   func()
	blocked  func(*testing.T)
}

func newDialoguePointerFixture(t *testing.T, surface string) dialoguePointerFixture {
	t.Helper()
	if surface == "town" {
		town := &pointerDialogueTown{revision: 1}
		town.pic = image.NewRGBA(image.Rect(0, 0, 240, 120))
		a := newTestApp(t, appRows(3), okLoader(t))
		a.SetTown(town)
		if !a.flow.showTown("") {
			t.Fatal("showTown refused dialogue fixture")
		}
		origin := image.Pt((640-240)/2, (480-120)/2)
		return dialoguePointerFixture{
			app: a, button: image.Rect(100, 60, 180, 90).Add(origin), body: origin.Add(image.Pt(10, 10)),
			advances: func() int { return town.advances },
			replace:  func() { town.revision++ },
			close:    func() { town.pic = nil; town.revision++ },
			reopen:   func() { town.pic = image.NewRGBA(image.Rect(0, 0, 240, 120)); town.revision++ },
			blocked: func(t *testing.T) {
				t.Helper()
				if len(town.chosen) != 0 || len(town.surfaceClicks) != 0 {
					t.Fatalf("dialogue leaked input: choices %v, surface clicks %v", town.chosen, town.surfaceClicks)
				}
			},
		}
	}
	a, seam := noticeApp(t)
	seam.v.SetDialogue(Dialogue{Text: "same page"})
	a.flow.advance = func(actions ...NoticeAction) (NoticeDest, string, MapOpener) {
		seam.advances++
		seam.v.SetDialogue(Dialogue{Text: "same page"})
		return NoticeStay, "", nil
	}
	commands := 0
	a.flow.order = func(uint32, int, int) { commands++ }
	a.flow.attack = func(uint32, uint32, uint32, int, int, bool) { commands++ }
	a.flow.grab = func(uint32, int, int, bool) { commands++ }
	l := AuthoredDialogueLayout()
	return dialoguePointerFixture{
		app: a, button: l.Button.Add(l.Box.Min), body: l.Box.Min.Add(image.Pt(4, 4)),
		advances: func() int { return seam.advances },
		replace:  func() { seam.v.SetDialogue(Dialogue{Text: "same page"}) },
		close:    func() { seam.v.ClearNotice() },
		reopen:   func() { seam.v.SetDialogue(Dialogue{Text: "same page"}) },
		blocked: func(t *testing.T) {
			t.Helper()
			if commands != 0 || seam.v.held {
				t.Fatalf("dialogue leaked input: commands %d, selection gesture %v", commands, seam.v.held)
			}
		},
	}
}

func (f dialoguePointerFixture) pointer(t *testing.T, action string, p image.Point, want int) {
	t.Helper()
	if err := f.app.HeadlessPointer(action, p.X, p.Y); err != nil {
		t.Fatal(err)
	}
	if got := f.advances(); got != want {
		t.Fatalf("%s at %v advanced %d pages, want %d", action, p, got, want)
	}
	f.blocked(t)
}

func TestDialoguePointerRequiresOwnedPressAndInsideRelease(t *testing.T) {
	for _, surface := range []string{"town", "mission"} {
		t.Run(surface, func(t *testing.T) {
			for _, gesture := range []string{"click", "orphan", "outside press", "outside release", "out and back", "body", "right", "left edge", "top edge", "right edge", "bottom edge"} {
				t.Run(gesture, func(t *testing.T) {
					f := newDialoguePointerFixture(t, surface)
					inside := f.button.Min.Add(f.button.Max).Div(2)
					outside := image.Pt(f.button.Min.X-1, inside.Y)
					switch gesture {
					case "click":
						f.pointer(t, "press", inside, 0)
						f.pointer(t, "move", inside, 0)
						f.pointer(t, "release", inside, 1)
						f.pointer(t, "release", inside, 1)
					case "orphan":
						f.pointer(t, "release", inside, 0)
					case "outside press":
						f.pointer(t, "press", outside, 0)
						f.pointer(t, "release", inside, 0)
					case "outside release":
						f.pointer(t, "press", inside, 0)
						f.pointer(t, "release", outside, 0)
						f.pointer(t, "release", inside, 0)
					case "out and back":
						f.pointer(t, "press", inside, 0)
						f.pointer(t, "move", outside, 0)
						f.pointer(t, "move", inside, 0)
						f.pointer(t, "release", inside, 1)
					case "body":
						f.pointer(t, "press", f.body, 0)
						f.pointer(t, "release", inside, 0)
					case "right":
						f.pointer(t, "right-press", inside, 0)
						f.pointer(t, "right-release", inside, 0)
						f.pointer(t, "release", inside, 0)
					default:
						p, want := inside, 1
						switch gesture {
						case "left edge":
							p.X = f.button.Min.X
						case "top edge":
							p.Y = f.button.Min.Y
						case "right edge":
							p.X, want = f.button.Max.X, 0
						case "bottom edge":
							p.Y, want = f.button.Max.Y, 0
						}
						f.pointer(t, "press", p, 0)
						f.pointer(t, "release", p, want)
					}
				})
			}
		})
	}
}

func TestDialoguePointerOwnershipEndsAtPageAndLifecycleBoundaries(t *testing.T) {
	for _, surface := range []string{"town", "mission"} {
		t.Run(surface, func(t *testing.T) {
			for _, boundary := range []string{"enter", "escape", "replacement", "close and reopen", "focus", "right cancellation", "window close", "screen replacement"} {
				t.Run(boundary, func(t *testing.T) {
					f := newDialoguePointerFixture(t, surface)
					inside := f.button.Min.Add(f.button.Max).Div(2)
					f.pointer(t, "press", inside, 0)
					want := 0
					switch boundary {
					case "enter", "escape":
						if err := f.app.HeadlessKey(boundary); err != nil {
							t.Fatal(err)
						}
						want = 1
					case "replacement":
						f.replace()
					case "close and reopen":
						f.close()
						f.reopen()
					case "focus":
						if err := f.app.HeadlessFocus(false); err != nil {
							t.Fatal(err)
						}
						if err := f.app.HeadlessFocus(true); err != nil {
							t.Fatal(err)
						}
					case "right cancellation":
						f.pointer(t, "right-press", inside, 0)
					case "window close":
						if !f.app.step(appInput{Close: true}, time.Unix(1_700_000_000, 0)) {
							t.Fatal("close did not request exit")
						}
					case "screen replacement":
						original := f.app.flow.screen
						f.app.flow.screen = ScreenMenu
						if err := f.app.HeadlessStep(); err != nil {
							t.Fatal(err)
						}
						f.app.flow.screen = original
					}
					inside = dialogueWindowPoint(t, f.app, surface, f.button.Min.Add(f.button.Max).Div(2))
					f.pointer(t, "release", inside, want)
					f.pointer(t, "press", inside, want)
					f.pointer(t, "release", inside, want+1)
				})
			}
		})
	}
}

func dialogueWindowPoint(t *testing.T, a *App, surface string, native image.Point) image.Point {
	t.Helper()
	var x, y int
	var ok bool
	if surface == "town" {
		x, y, ok = a.nativeFrameToWindow(native)
	} else {
		v := a.flow.viewer
		for dy := -3; dy <= 3 && !ok; dy++ {
			for dx := -3; dx <= 3 && !ok; dx++ {
				fx, fy, mapped := v.noticePlace().FrameToWindow(native.Add(image.Pt(dx, dy)))
				if mapped {
					x, y, ok = v.place.FrameToWindow(image.Pt(fx, fy))
				}
			}
		}
	}
	if !ok {
		t.Fatal("dialogue point has no window pixel")
	}
	return image.Pt(x, y)
}

func TestDialoguePointerTownReplacementCancelsOwnership(t *testing.T) {
	f := newDialoguePointerFixture(t, "town")
	inside := f.button.Min.Add(f.button.Max).Div(2)
	f.pointer(t, "press", inside, 0)
	replacement := &pointerDialogueTown{revision: 1}
	replacement.pic = image.NewRGBA(image.Rect(0, 0, 240, 120))
	f.app.SetTown(replacement)
	if err := f.app.HeadlessPointer("release", inside.X, inside.Y); err != nil {
		t.Fatal(err)
	}
	if replacement.advances != 0 || f.advances() != 0 {
		t.Fatalf("replacement inherited gesture: old %d, new %d", f.advances(), replacement.advances)
	}
	for _, edge := range []string{"press", "release"} {
		if err := f.app.HeadlessPointer(edge, inside.X, inside.Y); err != nil {
			t.Fatal(err)
		}
	}
	if replacement.advances != 1 {
		t.Fatalf("fresh replacement gesture advanced %d pages, want 1", replacement.advances)
	}
}

func TestDialoguePointerClosedTownConsumesRemainingRelease(t *testing.T) {
	for _, press := range []image.Point{image.Pt(340, 255), image.Pt(210, 190)} {
		t.Run(press.String(), func(t *testing.T) {
			town := &fakeTownDialogue{pic: image.NewRGBA(image.Rect(0, 0, 240, 120))}
			a := newTestApp(t, appRows(3), okLoader(t))
			a.SetTown(town)
			if !a.flow.showTown("") {
				t.Fatal("showTown refused dialogue fixture")
			}
			if err := a.HeadlessPointer("press", press.X, press.Y); err != nil {
				t.Fatal(err)
			}
			town.pic = nil
			if err := a.HeadlessPointer("release", pickerLeft, pickerTop); err != nil {
				t.Fatal(err)
			}
			if len(town.chosen) != 0 || town.advances != 0 {
				t.Fatalf("closed dialogue release chose %v or advanced %d pages", town.chosen, town.advances)
			}
		})
	}
}

func TestDialoguePointerUsesWindowPlacement(t *testing.T) {
	for _, surface := range []string{"town", "mission"} {
		for _, size := range []image.Point{image.Pt(1280, 960), image.Pt(1600, 900)} {
			t.Run(surface+size.String(), func(t *testing.T) {
				f := newDialoguePointerFixture(t, surface)
				f.app.Layout(size.X, size.Y)
				centre := f.button.Min.Add(f.button.Max).Div(2)
				p := dialogueWindowPoint(t, f.app, surface, centre)
				f.pointer(t, "press", p, 0)
				f.pointer(t, "release", p, 1)
			})
		}
	}
}

// An outcome panel's button acts on the release inside it, as every push
// button does (MENU-116); the press only latches.
func TestDialoguePointerOutcomePanelActsOnRelease(t *testing.T) {
	for _, tc := range []struct {
		name      string
		kind      NoticeKind
		secondary bool
		disabled  bool
		want      NoticeAction
	}{
		{"Victory", NoticeSuccess, false, false, NoticeVictory},
		{"Continue", NoticeSuccess, true, false, NoticeContinue},
		{"Exit", NoticeFailure, false, false, NoticeExitMain},
		{"Load", NoticeFailure, true, false, NoticeLoadGame},
		{"disabled Load", NoticeFailure, true, true, NoticeLoadGame},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, seam := noticeApp(t)
			seam.v.setFailureLoadCheck(func() bool { return !tc.disabled })
			seam.v.SetNotice("outcome", tc.kind)
			layout := seam.v.noticeLayout()
			button := layout.Button
			if tc.secondary {
				button = layout.SecondaryButton
			}
			p := button.Add(layout.Box.Min).Min.Add(button.Add(layout.Box.Min).Max).Div(2)
			if err := a.HeadlessPointer("press", p.X, p.Y); err != nil {
				t.Fatal(err)
			}
			wantCount := 1
			if tc.disabled {
				wantCount = 0
			}
			if len(seam.actions) != 0 {
				t.Fatalf("press actions = %v, want none before the release", seam.actions)
			}
			if err := a.HeadlessPointer("release", p.X, p.Y); err != nil {
				t.Fatal(err)
			}
			if len(seam.actions) != wantCount || (wantCount != 0 && seam.actions[0] != tc.want) || seam.v.held {
				t.Fatalf("release actions = %v, selection gesture %v", seam.actions, seam.v.held)
			}
		})
	}
}
