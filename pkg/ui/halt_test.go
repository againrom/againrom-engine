package ui

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
)

const (
	// haltStride is how many of cadence_test.go's frames one driven frame here
	// spans, so haltAt stays a literal offset from the one shared base.
	haltStride  = 10
	haltFrameUS = haltStride * cadenceFrameUS // 100_000 us

	// The cadence a map opens at, restated so a change to terrain's ladder has to
	// be made twice rather than silently re-basing every count below.
	haltPeriodUS = 62_000

	// One run, and what it is worth at that cadence: 31 * 100_000 us is
	// 3_100_000 us, and 3_100_000 / 62_000 is 50 with nothing left over.
	haltRun      = 31
	haltRunTicks = 50

	// The first frame after a dismissal, where "no backlog" is measured sharpest.
	// A dismissal frame reached with the remainder at 0 leaves it at
	// 100_000 - 62_000 = 38_000 us; the frame after it carries
	// 38_000 + 100_000 = 138_000 us, which is two whole 62_000 us ticks and
	// 14_000 us over. Two — not fifty, and not five hundred.
	haltFirstResumed = 2
)

// haltAt is the instant of driven frame n, a literal offset from cadence_test.go's
// one base.
func haltAt(n int) time.Time { return cadenceAt(n * haltStride) }

// haltNeutral is a map-screen frame pressing nothing, with the cursor at the
// centre of the frame so no edge scrolls.
func haltNeutral() appInput { return cadenceInput(false, false, false) }

// haltWorld is one load's whole share of the map seam as the far side sees it,
// modelled as the contract's column B requires.
//
// The clock is born at the map-load period, exactly as pkg/game builds the
// world's own, so an opened map needs no declaration to agree with it.
type haltWorld struct {
	clock   *terrain.Ticker
	calls   []cadenceCall
	stopped bool

	// The wall clock the far side reads. The front-end and the far side of a real
	// seam read ONE clock; the harness writes this to the instant of the frame it
	// is about to drive, so a frame that reaches MapTick and a frame that does not
	// both cost the same wall time.
	now     time.Time
	last    time.Time
	started bool

	ticks int // MapTick calls, suspended or not
	world int // ticks the world actually ran

	v        *Viewer
	advances int
	dest     NoticeDest
	msg      string
}

func newHaltWorld(born time.Time) *haltWorld {
	return &haltWorld{
		clock: terrain.NewTicker(terrain.SpeedIndexPeriod(terrain.DefaultSpeedIndex)),
		now:   born,
	}
}

// tick is MapTick. The first call takes the baseline and runs nothing — the
// pinned rule the map screen already inherits. Every call after it consumes the
// elapsed span WHETHER OR NOT the world is stopped, and credits the ticks it
// bought only when it is not.
func (w *haltWorld) tick() {
	w.ticks++
	if !w.started {
		w.started, w.last = true, w.now
		return
	}
	elapsed := int(w.now.Sub(w.last) / time.Microsecond)
	w.last = w.now
	fired := w.clock.AdvanceMicros(elapsed)
	if !w.stopped {
		w.world += fired
	}
}

// setCadence is MapCadence. It ADOPTS the period verbatim and holds the stop
// and owner-loop selector.
func (w *haltWorld) setCadence(periodUS int, stopped, unpaced, reset bool) {
	w.calls = append(w.calls, cadenceCall{periodUS: periodUS, stopped: stopped, unpaced: unpaced, reset: reset})
	w.clock.SetPeriod(periodUS)
	w.stopped = stopped
}

// lastCadence is the cadence the far side is holding now, and whether it was
// ever told anything at all.
func (w *haltWorld) lastCadence() (cadenceCall, bool) {
	if len(w.calls) == 0 {
		return cadenceCall{}, false
	}
	return w.calls[len(w.calls)-1], true
}

// advance is MapAdvance: the far side's answer when the player dismisses a
// notice. A stay destination closes it, which is the shipped seam's own
// habit. It never answers NoticeToMission — this fixture is 0073's, extended
// only for the seams 0077 added, and every case that reaches this file drives
// a destination that shipped before 0131.
func (w *haltWorld) advance(_ ...NoticeAction) (NoticeDest, string, MapOpener) {
	w.advances++
	if w.dest == NoticeStay && w.v != nil {
		w.v.ClearNotice()
	}
	return w.dest, w.msg, nil
}

// haltOpts names the two absences the contract has cases for: a map screen
// whose lettering failed to load (AC-8) and one the loader put no cadence
// under (AC-9).
type haltOpts struct {
	noFont    bool
	noCadence bool
}

func haltLoader(t *testing.T, out **haltWorld, o haltOpts) MapLoader {
	t.Helper()
	return func(int) (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
		v, err := NewViewer("halt", grid(60, 60), &terrain.Tileset{})
		if err != nil {
			t.Fatalf("NewViewer: %v", err)
		}
		hideBottomPanels(v)
		if !o.noFont {
			v.SetFont(panelFont())
		}
		w := newHaltWorld(haltAt(0))
		w.v = v
		*out = w
		var cadence MapCadence
		if !o.noCadence {
			cadence = w.setCadence
		}
		return v, w.tick, nil, cadence, nil, w.advance, nil, nil, nil, nil, nil
	}
}

// haltFix is the front-end parked on the map screen over a haltWorld, REACHED
// through the picker rather than assigned, with the far side's baseline already
// taken.
type haltFix struct {
	t *testing.T
	a *App
	v *Viewer
	w *haltWorld
	n int // the driven frame the run stands on
}

func newHaltFix(t *testing.T, o haltOpts) *haltFix {
	t.Helper()

	// The fixture's own premises, falsified rather than believed: the arithmetic
	// in this file's header, and the cadence it is written over.
	if got := terrain.SpeedIndexPeriod(terrain.DefaultSpeedIndex); got != haltPeriodUS {
		t.Fatalf("a map opens at %d us, not the %d every count in this file is computed over", got, haltPeriodUS)
	}
	if haltRun*haltFrameUS != haltRunTicks*haltPeriodUS {
		t.Fatalf("a %d-frame run is %d us and %d ticks is %d us — the schedule does not divide",
			haltRun, haltRun*haltFrameUS, haltRunTicks, haltRunTicks*haltPeriodUS)
	}

	var w *haltWorld
	a := newTestApp(t, appRows(3), haltLoader(t, &w, o))
	a.flow.screen = ScreenPicker
	a.step(appInput{Enter: true}, haltAt(0))
	if a.Screen() != ScreenMap {
		t.Fatalf("setup: screen = %v, want ScreenMap", a.Screen())
	}
	mapAtScaleOne(a)
	if w == nil || a.flow.viewer == nil {
		t.Fatalf("setup: far side %v, viewer %v — both are needed", w, a.flow.viewer)
	}
	if w.ticks != 0 {
		t.Fatalf("setup: the frame that opened the map made %d advance requests, want 0 — every count "+
			"below is offset from a map that opened having advanced nothing", w.ticks)
	}

	h := &haltFix{t: t, a: a, v: a.flow.viewer, w: w}
	h.frame(haltNeutral()) // the far side's baseline frame
	if w.ticks != 1 || w.world != 0 {
		t.Fatalf("setup: the baseline frame made %d requests and ran %d ticks, want 1 and 0",
			w.ticks, w.world)
	}
	if o.noCadence && len(w.calls) != 0 {
		t.Fatalf("setup: %d declarations crossed a seam the loader did not hand over", len(w.calls))
	}
	return h
}

// frame drives one frame, moving the wall clock BOTH sides read.
func (h *haltFix) frame(in appInput) {
	h.n++
	h.w.now = haltAt(h.n)
	h.a.step(in, haltAt(h.n))
}

// run drives k frames of one input and reports how far the WORLD moved over them
// — not how many requests were made, which is a different quantity and is read
// separately where it matters.
func (h *haltFix) run(k int, in appInput) int {
	before := h.w.world
	for i := 0; i < k; i++ {
		h.frame(in)
	}
	return h.w.world - before
}

// haltDismissal is one of the three routes the player has out of a notice, and
// the geometry the button one is computed from is the authored layout's own.
type haltDismissal struct {
	name string
	in   appInput

	// carries is the EXTRA world ticks this route's own dismissal frame leaves
	// for the run that follows it, and it is a property of 0066 rather than of
	// anything this story does.
	//
	// ESCAPE is read before any per-screen dispatch and RETURNS THERE, so the
	// frame that dismisses with it never reaches the map arm and never asks for
	// an advance — the pre-existing rule that the leaving tick advances nothing,
	// which a non-leaving key now reaches too. One frame of 100_000 us is
	// therefore still owed when the next run starts: 3_100_000 + 100_000 is
	// 3_200_000, and 3_200_000 / 62_000 is 51 with 38_000 left over. One extra
	// tick, and one only.
	//
	// It is MEASURED ON A FIXTURE THE LOADER GAVE NO CADENCE SEAM as well as on
	// one it did, and it is the same single tick either way — which is what says
	// it is the frame schedule and not the suspension: a screen with no seam
	// suspends nothing at all.
	carries int
}

func haltDismissals() []haltDismissal {
	l := AuthoredDialogueLayout()
	btn := l.Button.Add(l.Box.Min)
	return []haltDismissal{
		{name: "RETURN", in: appInput{Enter: true}},
		{name: "ESCAPE", in: appInput{Escape: true}, carries: 1},
		{name: "the button", in: appInput{
			PrimaryPressed:  true,
			PrimaryReleased: true,
			CursorX:         (btn.Min.X + btn.Max.X) / 2,
			CursorY:         (btn.Min.Y + btn.Max.Y) / 2,
		}},
	}
}

// TestAC1AnOpenNoticeStopsEveryAdvance — spec AC-1: "a notice opens and
// frames keep being driven → not one further advance reaches the world
// until it is dismissed."
//
// The control comes FIRST and is the same 31 frames, so a run that suspended
// nothing and a run that stopped the clock are told apart by 50 against 0 rather
// than by an assertion that could hold over a world nothing ever moved.
func TestAC1AnOpenNoticeStopsEveryAdvance(t *testing.T) {
	h := newHaltFix(t, haltOpts{})

	if got := h.run(haltRun, haltNeutral()); got != haltRunTicks {
		t.Fatalf("the control run of %d frames ran %d world ticks, want %d — with a still world the "+
			"suspension below says nothing", haltRun, got, haltRunTicks)
	}
	if n := len(h.w.calls); n != 0 {
		t.Fatalf("a map advancing normally declared %d cadences, want 0 — both sides are born at the "+
			"map-load period", n)
	}

	h.v.SetNotice("the mission opens", NoticeDialogue)
	if !h.v.NoticeOpen() {
		t.Fatal("setup: the notice is not open, so nothing below is about an open notice")
	}

	ticksBefore := h.w.ticks
	if got := h.run(haltRun, haltNeutral()); got != 0 {
		t.Errorf("%d advances reached the world over %d frames with a notice open, want 0 (FR-1, AC-1)",
			got, haltRun)
	}

	// The contract's column B, from this side: the front-end keeps ASKING at the
	// same rate and declares the stop, so nothing new crosses and the far side's
	// own diagnostics stay live. Column A — the front-end stops asking — is what
	// AC-2 below rules out by measurement; this is the same choice read at the
	// call rather than at the resume.
	if got, want := h.w.ticks-ticksBefore, haltRun; got != want {
		t.Errorf("%d advance requests were made over %d suspended frames, want %d — the suspension is "+
			"declared, not withheld (Constraints, column B)", got, haltRun, want)
	}
	if got, want := len(h.w.calls), 1; got != want {
		t.Fatalf("%d cadences crossed while the notice was open, want exactly %d — the stop", got, want)
	}
	if got, want := h.w.calls[0], (cadenceCall{periodUS: haltPeriodUS, stopped: true}); got != want {
		t.Errorf("the notice declared %+v, want %+v — a suspension moves no rate", got, want)
	}

	h.v.Camera().X, h.v.Camera().Y = 400, 400
	h.v.Camera().Clamp()
	camX := h.v.Camera().X
	pan := haltNeutral()
	pan.Viewer.PanRight = true
	before := h.w.world
	for i := 0; i < 3; i++ {
		h.frame(pan)
	}
	if h.v.Camera().X != camX {
		t.Errorf("the camera panned under an open notice: X %v -> %v (0077 FR-3)",
			camX, h.v.Camera().X)
	}
	if got := h.w.world - before; got != 0 {
		t.Errorf("three panning frames under an open notice ran %d world ticks, want 0", got)
	}

	// ...and it is a suspension rather than a stop: the world runs again.
	h.frame(appInput{Enter: true})
	if h.v.NoticeOpen() {
		t.Fatal("RETURN did not dismiss the notice")
	}
	if got := h.run(haltRun, haltNeutral()); got != haltRunTicks {
		t.Errorf("the %d frames after the dismissal ran %d world ticks, want %d", haltRun, got, haltRunTicks)
	}
}

// TestAC2DismissalResumesWithoutRepaying — spec AC-2: "a notice held open
// across many frames' worth of elapsed time ... the advances that follow run
// the same world time as the same frames run with no notice — no burst, no
// backlog."
//
// THREE LEGS OVER ONE SCHEDULE, differing only in how long the notice was held:
// not at all, one run, and ten runs — 3.1 s and 31 s of held wall-clock time. Each
// leg is measured over the SAME 31 frames after the dismissal frame, and each must
// give the same 50, which is what "a notice held for an hour and a notice held for
// a frame resume identically" means when it is measured.
//
// The sharpest single number is the FIRST frame after the dismissal: 2 ticks,
// computed in this file's header from a remainder of 38_000 us. A front-end that
// stopped asking for advances repays the whole held span there — 50 on the short
// leg and 500 on the long one — and the two legs disagree, which is the shape no
// arrangement of a single leg's assertions could detect.
func TestAC2DismissalResumesWithoutRepaying(t *testing.T) {
	for _, hold := range []int{0, haltRun, 10 * haltRun} {
		t.Run(fmt.Sprintf("held for %d frames", hold), func(t *testing.T) {
			h := newHaltFix(t, haltOpts{})

			// One run first, so every leg reaches its dismissal frame with the
			// remainder at 0 and the numbers below are the header's.
			if got := h.run(haltRun, haltNeutral()); got != haltRunTicks {
				t.Fatalf("the run before the notice ran %d world ticks, want %d", got, haltRunTicks)
			}

			dismiss := haltNeutral()
			if hold > 0 {
				h.v.SetNotice("held open", NoticeDialogue)
				if !h.v.NoticeOpen() {
					t.Fatal("setup: the notice is not open")
				}
				if got := h.run(hold, haltNeutral()); got != 0 {
					t.Fatalf("%d frames with a notice open ran %d world ticks, want 0", hold, got)
				}
				dismiss = appInput{Enter: true}
			}

			h.frame(dismiss)
			if hold > 0 {
				if h.w.advances != 1 {
					t.Fatalf("%d advances, want 1", h.w.advances)
				}
				if h.v.NoticeOpen() {
					t.Fatal("the notice survived its dismissal")
				}
			}

			if got := h.run(1, haltNeutral()); got != haltFirstResumed {
				t.Errorf("the first frame after the dismissal ran %d world ticks, want %d — a held span "+
					"is spent, not owed (FR-2, AC-2)", got, haltFirstResumed)
			}
			if got, want := h.run(haltRun-1, haltNeutral()), haltRunTicks-haltFirstResumed; got != want {
				t.Errorf("the next %d frames ran %d world ticks, want %d — %d over the whole run, the "+
					"same %d the run before the notice ran", haltRun-1, got, want, haltRunTicks, haltRunTicks)
			}
		})
	}

	// THE HELD SPAN IS SPENT ON EVERY ROUTE, and the DISMISSAL FRAME is a separate
	// quantity this case keeps apart from it.
	//
	// IT IS 0066'S RULE AND NOT THIS STORY'S, which is measured rather than
	// asserted: the same single extra tick appears on a fixture the loader gave NO
	// CADENCE SEAM, where nothing is ever suspended and no stop can be declared.
	// A suspension that was repaid would show the whole held span here — fifty
	// ticks, or five hundred — not one.
	//
	// So the expectation is DERIVED from the request count this case measures
	// rather than written beside it: a route that asks for an advance owes
	// nothing, a route that does not owes exactly one frame. If a later story
	// makes the ESCAPE frame reach the advance, the count moves, the two disagree,
	// and this fails asking for the table to be corrected.
	t.Run("every dismissal route resumes identically", func(t *testing.T) {
		for _, route := range haltDismissals() {
			t.Run(route.name, func(t *testing.T) {
				h := newHaltFix(t, haltOpts{})
				if got := h.run(haltRun, haltNeutral()); got != haltRunTicks {
					t.Fatalf("the run before the notice ran %d world ticks, want %d", got, haltRunTicks)
				}
				h.v.SetNotice("held open", NoticeDialogue)
				if got := h.run(haltRun, haltNeutral()); got != 0 {
					t.Fatalf("%d frames with a notice open ran %d world ticks, want 0", haltRun, got)
				}

				requests := h.w.ticks
				h.frame(route.in)
				requests = h.w.ticks - requests
				if h.v.NoticeOpen() {
					t.Fatalf("%s did not dismiss the notice", route.name)
				}

				// The table's claim about this route, checked against what the frame
				// actually did before anything is derived from it.
				if owed := 0; requests == 0 {
					owed = 1
					if route.carries != owed {
						t.Fatalf("%s asked for %d advances on its dismissal frame, so it owes %d "+
							"frame, but the route table says %d", route.name, requests, owed, route.carries)
					}
				} else if route.carries != 0 {
					t.Fatalf("%s asked for %d advances on its dismissal frame, so it owes nothing, "+
						"but the route table says %d", route.name, requests, route.carries)
				}

				if got, want := h.run(haltRun, haltNeutral()), haltRunTicks+route.carries; got != want {
					t.Errorf("the %d frames after a dismissal by %s ran %d world ticks, want %d — the "+
						"%d the same frames run with no notice, plus the %d this route's own dismissal "+
						"frame owes. A repaid SUSPENSION would show the whole held span here (FR-2, AC-2)",
						haltRun, route.name, got, want, haltRunTicks, route.carries)
				}
			})
		}
	})
}

// TestAC3ThePlayersPauseSurvivesTheNotice — spec AC-3: "the player has
// paused the world, then a notice opens ... it is dismissed → the world is
// still paused; and the same drive with the player NOT paused leaves the
// world running."
//
// BOTH ARMS THROUGH EACH OF THE THREE DISMISSAL ROUTES, because a route that
// wrote the player's setting on its way out would be invisible to a case that
// drove only one of them. The paused arm additionally presses the key afterwards
// and watches the world run, so "still paused" is a setting rather than a world
// that has stopped working.
func TestAC3ThePlayersPauseSurvivesTheNotice(t *testing.T) {
	arms := []struct {
		name   string
		paused bool
		want   int
	}{
		{"the player had paused", true, 0},
		{"the player had not", false, haltRunTicks},
	}
	for _, route := range haltDismissals() {
		for _, arm := range arms {
			t.Run(route.name+"/"+arm.name, func(t *testing.T) {
				h := newHaltFix(t, haltOpts{})

				// The player's own key, or a frame that presses nothing.
				h.frame(cadenceInput(arm.paused, false, false))

				h.v.SetNotice("the mission opens", NoticeDialogue)
				if !h.v.NoticeOpen() {
					t.Fatal("setup: the notice is not open")
				}
				if got := h.run(haltRun, haltNeutral()); got != 0 {
					t.Fatalf("%d frames with a notice open ran %d world ticks, want 0", haltRun, got)
				}

				h.frame(route.in)
				if h.v.NoticeOpen() {
					t.Fatalf("%s did not dismiss the notice", route.name)
				}
				if h.a.Screen() != ScreenMap {
					t.Fatalf("%s left the map screen for %v", route.name, h.a.Screen())
				}

				// One settling frame, deliberately NOT measured. The three routes differ
				// by whether the dismissal frame itself asked for an advance, and a
				// route that did not leaves that frame's span to be repaid on the next
				// one.
				h.frame(haltNeutral())

				if got := h.run(haltRun, haltNeutral()); got != arm.want {
					t.Errorf("%d frames after the dismissal ran %d world ticks, want %d — %s before the "+
						"notice opened (FR-3, AC-3)", haltRun, got, arm.want, arm.name)
				}

				if arm.paused {
					h.frame(cadenceInput(true, false, false))
					if got := h.run(haltRun, haltNeutral()); got != haltRunTicks {
						t.Errorf("the key that clears the player's pause left %d world ticks over %d "+
							"frames, want %d — what survived must be a PAUSE", got, haltRun, haltRunTicks)
					}
				}
			})
		}
	}
}

// TestAC4AnOutcomeNoticeSuspendsLikeADialogueOne — spec AC-4: "an outcome
// notice rather than a dialogue one → the world is suspended ... exactly
// as under a dialogue notice."
func TestAC4AnOutcomeNoticeSuspendsLikeADialogueOne(t *testing.T) {
	kinds := []struct {
		name string
		push func(v *Viewer)
	}{
		{"a dialogue notice", func(v *Viewer) { v.SetNotice("the mission opens", NoticeDialogue) }},
		{"an outcome notice", func(v *Viewer) { v.SetNotice("the mission is over", NoticeOutcome) }},
	}

	type result struct {
		control   int
		suspended int
		resumed   int
		calls     []cadenceCall
	}
	got := make([]result, len(kinds))

	for i, k := range kinds {
		t.Run(k.name, func(t *testing.T) {
			h := newHaltFix(t, haltOpts{})
			r := result{}

			r.control = h.run(haltRun, haltNeutral())
			k.push(h.v)
			if !h.v.NoticeOpen() {
				t.Fatalf("setup: %s is not open", k.name)
			}
			r.suspended = h.run(haltRun, haltNeutral())
			h.frame(appInput{Enter: true})
			if h.v.NoticeOpen() {
				t.Fatalf("%s survived its dismissal", k.name)
			}
			r.resumed = h.run(haltRun, haltNeutral())
			r.calls = append([]cadenceCall(nil), h.w.calls...)
			got[i] = r

			if r.control != haltRunTicks || r.resumed != haltRunTicks {
				t.Errorf("%s: %d world ticks before it and %d after it, want %d each",
					k.name, r.control, r.resumed, haltRunTicks)
			}
			if r.suspended != 0 {
				t.Errorf("%s: %d advances reached the world while it was open, want 0 (FR-4, AC-4)",
					k.name, r.suspended)
			}
			want := []cadenceCall{
				{periodUS: haltPeriodUS, stopped: true},
				{periodUS: haltPeriodUS, stopped: false},
			}
			if !reflect.DeepEqual(r.calls, want) {
				t.Errorf("%s declared %+v, want %+v — the stop, then the resume, at one rate",
					k.name, r.calls, want)
			}
		})
	}

	if !reflect.DeepEqual(got[0], got[1]) {
		t.Errorf("the dialogue leg measured %+v and the outcome leg %+v — not one requirement of this "+
			"contract is a function of which kind is showing (FR-4)", got[0], got[1])
	}
}

// TestAC5TheCadenceKeysAreInertWhileANoticeIsOpen — spec AC-5: "the pause
// key and both speed keys are pressed → nothing reaches the world's
// cadence, and after dismissal the rate and the pause setting are the ones
// that were in force when the notice opened."
//
// The rate afterwards is measured rather than read: a run of 31 frames at the
// map-load 62_000 us is 50 world ticks, and one leaked press of the speed key
// would put the far side on the next rung of the ladder — 50_000 us — where the
// same 31 frames are 3_100_000/50_000 = 62. One leaked pause press is 0. The three
// answers are far apart, so the count names which key leaked and not merely that
// one did.
func TestAC5TheCadenceKeysAreInertWhileANoticeIsOpen(t *testing.T) {
	// The rung a leaked speed press would land on, stated here and checked against
	// terrain's own ladder, so the discrimination above is a fact rather than a
	// hope.
	const leakedPeriodUS, leakedTicks = 50_000, 62
	if got := terrain.CadencePeriod(terrain.CadenceRung(haltPeriodUS) + 1); got != leakedPeriodUS {
		t.Fatalf("one rung up from %d us is %d us, not the %d this case discriminates against",
			haltPeriodUS, got, leakedPeriodUS)
	}
	if haltRun*haltFrameUS/leakedPeriodUS != leakedTicks {
		t.Fatalf("a run at %d us is %d ticks, not %d", leakedPeriodUS,
			haltRun*haltFrameUS/leakedPeriodUS, leakedTicks)
	}

	h := newHaltFix(t, haltOpts{})
	h.frame(haltNeutral())

	h.v.SetNotice("the mission opens", NoticeDialogue)
	if !h.v.NoticeOpen() {
		t.Fatal("setup: the notice is not open")
	}
	h.frame(haltNeutral()) // the frame the stop is declared on
	if got, want := len(h.w.calls), 1; got != want {
		t.Fatalf("setup: %d cadences crossed when the notice opened, want %d", got, want)
	}
	opened := h.w.calls[0]
	if want := (cadenceCall{periodUS: haltPeriodUS, stopped: true}); opened != want {
		t.Fatalf("setup: the notice declared %+v, want %+v", opened, want)
	}

	// The pause key and both speed keys, one per frame, twice each.
	presses := []appInput{
		cadenceInput(true, false, false),
		cadenceInput(false, true, false),
		cadenceInput(false, false, true),
		cadenceInput(true, false, false),
		cadenceInput(false, true, false),
		cadenceInput(false, false, true),
	}
	before := h.w.world
	for _, in := range presses {
		h.frame(in)
	}
	if got := h.w.world - before; got != 0 {
		t.Errorf("the six cadence presses ran %d world ticks, want 0", got)
	}
	if got, want := len(h.w.calls), 1; got != want {
		t.Errorf("%d cadences crossed over six presses under an open notice, want the %d the notice "+
			"itself declared — the three keys change nothing at all (FR-5, AC-5)", got, want)
	}
	if !h.w.stopped || h.w.clock.Period() != haltPeriodUS {
		t.Errorf("the far side stands at %d us stopped=%v, want %d us stopped — not the rate, and not "+
			"the setting", h.w.clock.Period(), h.w.stopped, haltPeriodUS)
	}

	h.frame(appInput{Enter: true})
	if h.v.NoticeOpen() {
		t.Fatal("the notice survived its dismissal")
	}

	// The cadence after the dismissal is the one that was in force when it opened
	// — measured over a run, then read at the seam.
	if got := h.run(haltRun, haltNeutral()); got != haltRunTicks {
		t.Errorf("the %d frames after the dismissal ran %d world ticks, want %d — %d would mean a speed "+
			"press leaked and 0 a pause press (FR-5, AC-5)", haltRun, got, haltRunTicks, leakedTicks)
	}
	last, ok := h.w.lastCadence()
	if !ok {
		t.Fatal("nothing crossed the cadence seam at all")
	}
	if want := (cadenceCall{periodUS: opened.periodUS, stopped: false}); last != want {
		t.Errorf("the cadence after the dismissal is %+v, want %+v — the rate the notice opened over, "+
			"running", last, want)
	}
}

// TestAC8AMapScreenThatCannotDrawANoticeIsNotSuspended — spec AC-8: "a map
// screen whose lettering failed to load, so no notice can be drawn ... the
// world is NOT suspended, nothing is dimmed, and the screen behaves exactly
// as it does today."
//
// This is the error case, and the spec says why: a notice that cannot be seen
// must not stop a world the player is looking at, for the same reason it already
// must not swallow the key that leaves the screen — which is the last thing this
// case drives.
func TestAC8AMapScreenThatCannotDrawANoticeIsNotSuspended(t *testing.T) {
	h := newHaltFix(t, haltOpts{noFont: true})

	if got := h.run(haltRun, haltNeutral()); got != haltRunTicks {
		t.Fatalf("the control run ran %d world ticks, want %d", got, haltRunTicks)
	}

	h.v.SetNotice("nobody can read this", NoticeDialogue)
	if h.v.NoticeOpen() {
		t.Fatal("setup: a fontless viewer reports a notice open, so this case is not the error case")
	}

	if got := h.run(haltRun, haltNeutral()); got != haltRunTicks {
		t.Errorf("%d world ticks over %d frames with a notice pushed to a fontless screen, want %d — a "+
			"notice that cannot be seen suspends nothing (FR-1, AC-8)", got, haltRun, haltRunTicks)
	}
	if n := len(h.w.calls); n != 0 {
		t.Errorf("%d cadences crossed, want 0 — nothing was declared, so nothing has to be undeclared", n)
	}

	// ...and the screen behaves exactly as it does today: the key that leaves it
	// is not swallowed by a box nobody can see.
	h.frame(appInput{Escape: true})
	leaveViaMenu(h.a.flow) //
	if h.a.Screen() != ScreenPicker {
		t.Errorf("Escape over an undrawable notice landed on %v, want ScreenPicker", h.a.Screen())
	}
}

// TestAC9AMapScreenWithNoCadenceSeamIsUnchanged — spec AC-9: "a map screen
// the loader put no cadence under ... a notice opens and is dismissed →
// nothing crosses, nothing panics, and the screen is otherwise unchanged."
func TestAC9AMapScreenWithNoCadenceSeamIsUnchanged(t *testing.T) {
	h := newHaltFix(t, haltOpts{noCadence: true})

	if got := h.run(haltRun, haltNeutral()); got != haltRunTicks {
		t.Fatalf("the control run ran %d world ticks, want %d", got, haltRunTicks)
	}

	for _, route := range haltDismissals() {
		h.v.SetNotice("the mission opens", NoticeDialogue)
		if !h.v.NoticeOpen() {
			t.Fatalf("%s: setup: the notice is not open", route.name)
		}
		if got := h.run(haltRun, haltNeutral()); got != haltRunTicks {
			t.Errorf("%s: %d world ticks over %d frames with a notice open and no cadence seam, want "+
				"%d — with nothing to declare a stop through the screen is unchanged (AC-9)",
				route.name, got, haltRun, haltRunTicks)
		}
		in := route.in
		if in.PrimaryPressed {
			_, at, scale, ok := h.v.noticePresent()
			if !ok {
				t.Fatal("the notice has no painted button")
			}
			button := h.v.noticeLayout().Button
			centre := button.Min.Add(button.Max).Div(2)
			// A window smaller than the frame maps no window pixel to some frame
			// pixels, so the press is taken at the first painted pixel near the
			// button's centre that has one.
			ok = false
			for _, d := range [][2]int{{0, 0}, {1, 0}, {0, 1}, {-1, 0}, {0, -1}, {1, 1}, {-1, -1}, {2, 0}, {0, 2}} {
				point := centre
				point.X, point.Y = at.X+int(float64(point.X+d[0])*scale), at.Y+int(float64(point.Y+d[1])*scale)
				if in.CursorX, in.CursorY, ok = h.v.place.FrameToWindow(point); ok {
					break
				}
			}
			if !ok {
				t.Fatal("the painted button has no window position")
			}
		}
		h.frame(in)
		if h.v.NoticeOpen() {
			t.Errorf("%s: the notice survived its dismissal", route.name)
		}
		if h.a.Screen() != ScreenMap {
			t.Fatalf("%s: the dismissal left the map screen for %v", route.name, h.a.Screen())
		}
		// One settling frame between the routes, unmeasured, for the reason given
		// in AC-3: the routes differ by one frame's advance request and that
		// difference is AC-2's to measure, not this case's.
		h.frame(haltNeutral())
	}
	if h.w.advances != len(haltDismissals()) {
		t.Errorf("%d advances over %d dismissals, want one each", h.w.advances, len(haltDismissals()))
	}

	// The three cadence keys reach nothing, before or after a notice, because
	// there is nothing for them to reach.
	for _, in := range []appInput{
		cadenceInput(true, false, false),
		cadenceInput(false, true, false),
		cadenceInput(false, false, true),
	} {
		h.frame(in)
	}
	if got := h.run(haltRun, haltNeutral()); got != haltRunTicks {
		t.Errorf("%d world ticks over %d frames after the three cadence keys, want %d — nothing crosses "+
			"a seam the loader did not hand over", got, haltRun, haltRunTicks)
	}
	if got := h.w.ticks; got == 0 {
		t.Error("no advance was ever requested, so nothing above measured a working map screen")
	}
}

func TestP1NoNoticeSuspendsAScreenThatCannotDrawOne(t *testing.T) {
	const pushes = 8

	h := newHaltFix(t, haltOpts{noFont: true})
	before := h.w.world
	for i := 0; i < pushes; i++ {
		if i%2 == 0 {
			h.v.SetNotice("the mission opens", NoticeDialogue)
		} else {
			h.v.SetNotice("the mission is over", NoticeOutcome)
		}
		if h.v.NoticeOpen() {
			t.Fatalf("push %d: a fontless viewer reports a notice open", i)
		}
		h.run(haltRun, haltNeutral())
	}

	if got, want := h.w.world-before, pushes*haltRunTicks; got != want {
		t.Errorf("%d notices over %d runs left %d world ticks, want %d — %d a run, suspended by none of "+
			"them (P-1)", pushes, pushes, got, want, haltRunTicks)
	}
	if n := len(h.w.calls); n != 0 {
		t.Errorf("%d cadences crossed over %d pushed notices, want 0 (P-1)", n, pushes)
	}
}

func TestP2ThePauseSettingIsChangedByThePauseKeyAndByNothingElse(t *testing.T) {
	h := newHaltFix(t, haltOpts{})

	cycle := func(round int, want int) {
		t.Helper()
		for i := 0; i < 3; i++ {
			if i%2 == 0 {
				h.v.SetNotice("the mission opens", NoticeDialogue)
			} else {
				h.v.SetNotice("the mission is over", NoticeOutcome)
			}
			if !h.v.NoticeOpen() {
				t.Fatalf("round %d, notice %d: it is not open", round, i)
			}
			if got := h.run(haltRun, haltNeutral()); got != 0 {
				t.Fatalf("round %d, notice %d: %d world ticks while it was open, want 0", round, i, got)
			}
			h.frame(appInput{Enter: true})
			if h.v.NoticeOpen() {
				t.Fatalf("round %d, notice %d: it survived its dismissal", round, i)
			}
			if got := h.run(haltRun, haltNeutral()); got != want {
				t.Errorf("round %d, after notice %d: %d world ticks over %d frames, want %d — a notice "+
					"opening and closing writes no pause setting (P-2)", round, i, got, haltRun, want)
			}
		}
	}

	// Three notices over a running world leave it running.
	cycle(1, haltRunTicks)

	// The key, and only the key, pauses it.
	h.frame(cadenceInput(true, false, false))
	if got := h.run(haltRun, haltNeutral()); got != 0 {
		t.Fatalf("the pause key left %d world ticks over %d frames, want 0", got, haltRun)
	}

	// Three more over a paused world leave it paused.
	cycle(2, 0)

	// ...and the key, and only the key, frees it.
	h.frame(cadenceInput(true, false, false))
	if got := h.run(haltRun, haltNeutral()); got != haltRunTicks {
		t.Errorf("the pause key left %d world ticks over %d frames, want %d — after six notices the "+
			"setting is still the one the key wrote (P-2)", got, haltRun, haltRunTicks)
	}
}

func TestP4TheCadenceIsDeclaredOnlyWhenItChanges(t *testing.T) {
	h := newHaltFix(t, haltOpts{})

	var want []cadenceCall
	check := func(step string) {
		t.Helper()
		if !reflect.DeepEqual(h.w.calls, want) {
			t.Fatalf("%s: the far side was told %+v, want %+v", step, h.w.calls, want)
		}
	}

	// 1. Nothing has changed, so nothing is declared, over a whole run.
	if got := h.run(haltRun, haltNeutral()); got != haltRunTicks {
		t.Fatalf("step 1: %d world ticks, want %d", got, haltRunTicks)
	}
	check("step 1, a map running as it opened")

	// 2. A notice opens: the cadence in force changes, so exactly one declaration.
	h.v.SetNotice("one", NoticeDialogue)
	if got := h.run(haltRun, haltNeutral()); got != 0 {
		t.Fatalf("step 2: %d world ticks with a notice open, want 0", got)
	}
	want = append(want, cadenceCall{periodUS: haltPeriodUS, stopped: true})
	check("step 2, a notice opens")

	// 3. It is dismissed: one more, and one only.
	h.frame(appInput{Enter: true})
	if got := h.run(haltRun, haltNeutral()); got != haltRunTicks {
		t.Fatalf("step 3: %d world ticks after the dismissal, want %d", got, haltRunTicks)
	}
	want = append(want, cadenceCall{periodUS: haltPeriodUS, stopped: false})
	check("step 3, the notice is dismissed")

	// 4. The player's own key: one more.
	h.frame(cadenceInput(true, false, false))
	if got := h.run(haltRun, haltNeutral()); got != 0 {
		t.Fatalf("step 4: %d world ticks after the pause key, want 0", got)
	}
	want = append(want, cadenceCall{periodUS: haltPeriodUS, stopped: true})
	check("step 4, the player pauses")

	// 5. A notice opens over an already paused world: NO change, no declaration.
	h.v.SetNotice("two", NoticeDialogue)
	if !h.v.NoticeOpen() {
		t.Fatal("step 5: setup: the notice is not open")
	}
	if got := h.run(haltRun, haltNeutral()); got != 0 {
		t.Fatalf("step 5: %d world ticks, want 0", got)
	}
	check("step 5, a notice opens over a world the player had paused")

	// 6. ...and its dismissal is no change either.
	h.frame(appInput{Enter: true})
	if h.v.NoticeOpen() {
		t.Fatal("step 6: the notice survived its dismissal")
	}
	if got := h.run(haltRun, haltNeutral()); got != 0 {
		t.Fatalf("step 6: %d world ticks after the dismissal, want 0 — the player is still paused", got)
	}
	check("step 6, that notice is dismissed")

	// 7. The key frees it: one more, the fourth and last.
	h.frame(cadenceInput(true, false, false))
	if got := h.run(haltRun, haltNeutral()); got != haltRunTicks {
		t.Fatalf("step 7: %d world ticks after the pause key, want %d", got, haltRunTicks)
	}
	want = append(want, cadenceCall{periodUS: haltPeriodUS, stopped: false})
	check("step 7, the player un-pauses")

	if got, wantN := len(h.w.calls), 4; got != wantN {
		t.Errorf("%d declarations over %d driven frames, want %d — one per change of the cadence in "+
			"force and not one more (P-4)", got, h.n, wantN)
	}
}
