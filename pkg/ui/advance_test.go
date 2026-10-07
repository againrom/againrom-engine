package ui

// The campaign's own successor, this package's half.
//
// NOTHING HERE DECIDES A CAMPAIGN. Every case drives the seam through a stub
// opener this file builds — pkg/ui cannot read a mission number and nothing
// below asserts anything about one. What is under test is what the front-end
// DOES with a NoticeToMission destination once the far side has already
// decided to hand one over, never which destination a campaign would
// produce — that is pkg/game's (0131 T3).
//
// NOTHING HERE OPENS A WINDOW, READS A CLOCK OR TOUCHES A GAME INSTALL.

import (
	"errors"
	"image"
	"testing"
	"time"

	"againrom/pkg/render/frame"

	"againrom/pkg/render/terrain"
)

// advanceSeam is a map seam that answers a fixed destination on every press —
// noticeSeam's own shape (notice_test.go), widened for the third result this
// story adds.
type advanceSeam struct {
	dest  NoticeDest
	msg   string
	open  MapOpener
	calls int
}

func (s *advanceSeam) advance(_ ...NoticeAction) (NoticeDest, string, MapOpener) {
	s.calls++
	return s.dest, s.msg, s.open
}

// advanceApp builds a front-end parked on the map screen over an ENDED
// mission, with an outcome notice open and an advance seam ready to answer
// whatever a case sets it to — noticeApp's own shape, widened to let a case
// choose the window size the ended mission (and so, at first, the successor)
// opens at, which AC-13 needs to be other than the default.
func advanceApp(t *testing.T, winW, winH int) (a *App, seam *advanceSeam, ended *Viewer) {
	t.Helper()
	seam = &advanceSeam{}
	load := func(int) (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
		v, err := NewViewer("ended", grid(60, 60), &terrain.Tileset{})
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		hideBottomPanels(v)
		v.SetFont(panelFont())
		ended = v
		return v, func() {}, nil, nil, nil, seam.advance, nil, nil, nil, nil, nil
	}
	a = newTestApp(t, appRows(3), load)
	a.Layout(winW, winH)
	a.flow.activateNewGame()
	a.step(appInput{Enter: true}, time.Unix(1_700_000_000, 0))
	if a.Screen() != ScreenMap {
		t.Fatalf("setup: screen = %v, want ScreenMap", a.Screen())
	}
	ended.SetNotice("you won", NoticeOutcome)
	if !ended.NoticeOpen() {
		t.Fatal("setup: the outcome notice is not open")
	}
	return a, seam, ended
}

// successorOpener is a MapOpener stub standing in for what pkg/game's own
// MissionOpener would hand back: a fresh viewer, arming its own start view
// exactly as MissionOpenerWith does, and a MapTick and MapOrder that count
// how many times this frame reached them.
type successorOpener struct {
	v         *Viewer
	startCell image.Point
	ticks     int
	orders    int
}

func (o *successorOpener) open() (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
	v, err := NewViewer("successor", bigGrid(), &terrain.Tileset{})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}
	hideBottomPanels(v)
	v.SetStartView(o.startCell)
	o.v = v
	tick := func() { o.ticks++ }
	order := func(uint32, int, int) { o.orders++ }
	return v, tick, order, nil, nil, nil, nil, nil, nil, nil, nil
}

func TestAdvanceToMissionEntersTheOpenersMapThroughTheSharedEntry(t *testing.T) {
	const winW, winH = 1600, 900 // other than DefaultWindowW x DefaultWindowH (AC-13)
	now := time.Unix(1_700_000_000, 0)
	startCell := image.Pt(100, 90)

	for _, name := range []string{"RETURN", "ESCAPE", "the button"} {
		t.Run(name, func(t *testing.T) {
			a, seam, ended := advanceApp(t, winW, winH)

			// "the button"'s window position is computed from the ENDED
			// mission's own two placements, in the order the draw side
			// composes through them (1026 B2): the notice's design space is
			// fitted into the mission viewport, and the mission frame is
			// fitted into the window. 1600x900 is an integer multiple of
			// neither, so both carry a real offset and a real scale, and the
			// press below is at the window pixel those two inversions name.
			var in appInput
			switch name {
			case "RETURN":
				in = appInput{Enter: true}
			case "ESCAPE":
				in = appInput{Escape: true}
			case "the button":
				l := AuthoredOutcomeLayout()
				btn := l.Button.Add(l.Box.Min)
				centre := image.Pt((btn.Min.X+btn.Max.X)/2, (btn.Min.Y+btn.Max.Y)/2)
				fx, fy, ok := ended.noticePlace().FrameToWindow(centre)
				if !ok {
					t.Fatal("setup: the button's design-space centre maps to no viewport position")
				}
				bx, by, ok := ended.place.FrameToWindow(image.Pt(fx, fy))
				if !ok {
					t.Fatal("setup: the button's frame position maps to no window position")
				}
				in = appInput{PrimaryPressed: true, CursorX: bx, CursorY: by}
			}

			o := &successorOpener{startCell: startCell}
			seam.dest, seam.msg, seam.open = NoticeToMission, "unused: FR-8's sentence, not this seam's", o.open

			a.step(in, now)

			// SC-3, AC-4's own front-end half: the map screen showing is the
			// opener's own, and the map list was never shown.
			if a.Screen() != ScreenMap {
				t.Fatalf("screen = %v, want ScreenMap — never the map list", a.Screen())
			}
			if a.flow.viewer == nil || a.flow.viewer == ended {
				t.Fatalf("a.flow.viewer = %p, want the opener's own viewer, not the ended mission's", a.flow.viewer)
			}
			if a.flow.viewer != o.v {
				t.Fatalf("a.flow.viewer is not the opener's own viewer")
			}
			if seam.calls != 1 {
				t.Fatalf("%d calls to the ended mission's own seam, want 1", seam.calls)
			}

			if !o.v.commandMode {
				t.Error("the successor was not put into command mode — DD-6's own entry was bypassed")
			}
			if got, want := a.flow.rung, terrain.CadenceRung(terrain.SpeedIndexPeriod(terrain.DefaultSpeedIndex)); got != want {
				t.Errorf("rung = %d, want %d — P-6's own cadence, the one any door into a map screen opens at", got, want)
			}
			if a.flow.stopped {
				t.Error("the successor opened stopped — P-6")
			}

			// AC-13, SC-11: the successor's view is sized by the door it came
			// through and not left at whatever the ended mission's camera held
			// or at a Viewer's own placeholder default, and it is centred on
			// the cell the opener armed, exactly as the map list's own door
			// would leave it.
			//
			// The odd 1600x900 window is 16:9, so production Layout expands the
			// logical frame to ceil(1600*768/900)=1366 and leaves 1206 pixels
			// for the map after the right strip.
			if want := image.Pt(1206, 768); o.v.ViewportSize() != want {
				t.Errorf("the successor's view is %v, want expanded mission viewport %v",
					o.v.ViewportSize(), want)
			}
			if got, want := o.v.FrameSize(), image.Pt(1366, 768); got != want {
				t.Errorf("the successor's placement is for a %v frame, want %v: the door never laid it out", got, want)
			}
			if got, want := o.v.place, frame.Fit(1366, 768, winW, winH); got != want {
				t.Errorf("the successor's placement is %+v, want the mission frame fitted into the "+
					"real %dx%d window, %+v", got, winW, winH, want)
			}
			assertCentred(t, o.v, startCell)

			if o.ticks != 0 {
				t.Errorf("%d ticks on the dismissing frame, want 0", o.ticks)
			}
			if o.orders != 0 {
				t.Errorf("%d orders on the dismissing frame, want 0", o.orders)
			}
			if o.v.held {
				t.Error("the dismissing press latched a gesture on the successor's own viewer")
			}

			// The seam is wired, not dead: the settling frame after it asks
			// again, exactly as any other freshly opened map screen would.
			a.step(appInput{}, now)
			if o.ticks != 1 {
				t.Errorf("%d ticks after the settling frame, want 1", o.ticks)
			}
		})
	}
}

func TestAdvanceToAFailingOpenerReportsTheFailuresOwnWords(t *testing.T) {
	a, seam, _ := advanceApp(t, DefaultWindowW, DefaultWindowH)
	boom := errors.New("read successor.alm: no such entry")
	seam.dest = NoticeToMission
	seam.msg = "mission 10 won — your party carries over; mission 20 follows and opens now"
	seam.open = func() (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, boom
	}

	a.step(appInput{Enter: true}, time.Unix(1_700_000_000, 0))

	if a.Screen() != ScreenPicker {
		t.Fatalf("screen = %v, want ScreenPicker — P-2", a.Screen())
	}
	if a.flow.msg != boom.Error() {
		t.Errorf("message = %q, want the failure's own words %q, not the seam's own sentence", a.flow.msg, boom.Error())
	}
	if a.flow.viewer != nil || a.flow.tick != nil || a.flow.advance != nil {
		t.Error("the ended mission's seam outlived the screen it belonged to — P-2")
	}
}

func TestAdvanceWithANilOpenerReachesTheMapListWithTheSeamsSentence(t *testing.T) {
	a, seam, _ := advanceApp(t, DefaultWindowW, DefaultWindowH)
	seam.dest = NoticeToMission
	seam.msg = "mission 10 won — your party carries over; mission 20 follows and opens now"
	seam.open = nil

	a.step(appInput{Enter: true}, time.Unix(1_700_000_000, 0))

	if a.Screen() != ScreenPicker {
		t.Fatalf("screen = %v, want ScreenPicker — P-2", a.Screen())
	}
	if a.flow.msg != seam.msg {
		t.Errorf("message = %q, want the seam's own sentence %q", a.flow.msg, seam.msg)
	}
	if a.flow.viewer != nil {
		t.Error("a nil opener left a viewer on screen — P-2")
	}
}

func TestASecondDismissalOpensNoFurtherMission(t *testing.T) {
	a, seam, _ := advanceApp(t, DefaultWindowW, DefaultWindowH)
	now := time.Unix(1_700_000_000, 0)
	o := &successorOpener{startCell: image.Pt(10, 10)}
	seam.dest, seam.open = NoticeToMission, o.open

	a.step(appInput{Enter: true}, now)
	if a.flow.viewer != o.v || seam.calls != 1 {
		t.Fatalf("setup: viewer = %p (want the opener's %p), %d seam calls (want 1)", a.flow.viewer, o.v, seam.calls)
	}

	a.step(appInput{Enter: true}, now)
	if seam.calls != 1 {
		t.Errorf("%d calls to the ended mission's own seam, want 1 — a second dismissal opens no mission (P-5)", seam.calls)
	}
	if a.flow.viewer != o.v {
		t.Error("a second press over the successor's own screen replaced its viewer")
	}
}
