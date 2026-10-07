package ui

import (
	"image"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/render/frame"
	"againrom/pkg/render/terrain"
)

// cadenceCall is one (periodUS, stopped, unpaced, reset) tuple as the far side
// of the seam received it — the four scalars MapCadence carries and nothing
// else.
type cadenceCall struct {
	periodUS int
	stopped  bool
	unpaced  bool
	reset    bool
}

// cadenceSeam is one load's whole share of the map seam as the far side sees it:
// every cadence call, every order, how many ticks, the clock those cadence calls
// re-rated, and the ORDER in which the front-end made the calls within a frame.
//
// The clock is born at the map-load period, which is what lets "an opened map
// needs no call to agree" be asserted rather than assumed.
type cadenceSeam struct {
	calls   []cadenceCall
	orders  []issued
	ticks   int
	clock   *terrain.Ticker
	stopped bool
	unpaced bool
	log     []string
}

func newCadenceSeam() *cadenceSeam {
	return &cadenceSeam{clock: terrain.NewTicker(terrain.SpeedIndexPeriod(terrain.DefaultSpeedIndex))}
}

func (s *cadenceSeam) tick() {
	s.ticks++
	s.log = append(s.log, "tick")
}

func (s *cadenceSeam) order(entity uint32, x, y int) {
	s.orders = append(s.orders, issued{entity: entity, x: x, y: y})
	s.log = append(s.log, "order")
}

// It ADOPTS the period verbatim, which is what mapWorld.setCadence does: the
// arithmetic happened once, on the far side, before the call was made. A
// stand-in that divided anything here would be measuring a seam this tree
// does not have.
func (s *cadenceSeam) setCadence(periodUS int, stopped, unpaced, reset bool) {
	s.calls = append(s.calls, cadenceCall{periodUS: periodUS, stopped: stopped, unpaced: unpaced, reset: reset})
	s.clock.SetPeriod(periodUS)
	s.stopped = stopped
	s.unpaced = unpaced
	s.log = append(s.log, "cadence")
}

// last is the cadence the far side is holding now, and whether it was ever told
// anything at all.
func (s *cadenceSeam) last() (cadenceCall, bool) {
	if len(s.calls) == 0 {
		return cadenceCall{}, false
	}
	return s.calls[len(s.calls)-1], true
}

// cadenceLoader hands every successful load a FRESH cadenceSeam with all three
// halves of the seam wired to it.
func cadenceLoader(t *testing.T, seam **cadenceSeam) MapLoader {
	t.Helper()
	return func(int) (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
		v, err := NewViewer("t", grid(60, 60), &terrain.Tileset{})
		if err != nil {
			t.Fatalf("NewViewer: %v", err)
		}
		s := newCadenceSeam()
		*seam = s
		return v, s.tick, s.order, s.setCadence, nil, nil, nil, nil, nil, nil, nil
	}
}

// The frame schedule every measurement below is driven on: a hundred frames of
// 10 ms is exactly one second, so no elapsed time is left on a sub-frame tail
// and a counted rise is the clock's own.
const (
	cadenceFrames  = 100
	cadenceFrameUS = 10_000
	cadenceSecond  = cadenceFrames * cadenceFrameUS

	// The cursor sits at the centre of the frame, away from every edge, so a
	// neutral frame edge-scrolls nothing and a camera that moved was moved by
	// something this test did.
	cadenceCX = frame.W / 2
	cadenceCY = frame.H / 2
)

var cadenceBase = time.Unix(1_700_000_000, 0)

// mustLast is seam.last with its "was there one at all" turned into a fatal, so
// a case about WHICH cadence crossed is not also a case about whether one did.
func mustLast(t *testing.T, s *cadenceSeam) cadenceCall {
	t.Helper()
	call, ok := s.last()
	if !ok {
		t.Fatal("nothing crossed the cadence seam")
	}
	return call
}

// cadenceAt is the instant of frame n, counted from the base in whole
// microseconds.
func cadenceAt(n int) time.Time {
	return cadenceBase.Add(time.Duration(n*cadenceFrameUS) * time.Microsecond)
}

// cadenceInput is a neutral map-screen frame with the given cadence keys.
func cadenceInput(pause, faster, slower bool) appInput {
	return appInput{
		Viewer:  Input{CursorX: cadenceCX, CursorY: cadenceCY},
		CursorX: cadenceCX,
		CursorY: cadenceCY,
		Pause:   pause,
		Faster:  faster,
		Slower:  slower,
	}
}

// cadenceApp parks the front-end on the map screen over a cadence loader,
// REACHED through the picker rather than assigned, and takes the animation
// baseline on the first map frame — the standalone viewer's own pinned rule,
// which the map screen inherits because it drives the same step.
//
// It returns the frame number the caller's own driving starts from.
func cadenceApp(t *testing.T) (*App, *Viewer, *cadenceSeam, int) {
	t.Helper()
	var seam *cadenceSeam
	a := newTestApp(t, appRows(3), cadenceLoader(t, &seam))
	a.flow.screen = ScreenPicker
	a.step(appInput{Enter: true}, cadenceAt(0))
	if a.Screen() != ScreenMap {
		t.Fatalf("setup: screen = %v, want ScreenMap", a.Screen())
	}
	if seam == nil || a.flow.viewer == nil {
		t.Fatalf("setup: seam %v, viewer %v — both are needed", seam, a.flow.viewer)
	}
	a.step(cadenceInput(false, false, false), cadenceAt(1))
	if got := a.flow.viewer.AnimationCounter(); got != 0 {
		t.Fatalf("setup: the baseline frame moved the water counter to %d, want 0", got)
	}
	return a, a.flow.viewer, seam, 1
}

// driveSecond drives one second of wall-clock time through App.step — the
// production path, map arm and all — and reports how far the WATER counter rose
// over it and which frame the run ended on.
func driveSecond(t *testing.T, a *App, v *Viewer, from int) (water, to int) {
	t.Helper()
	before := v.AnimationCounter()
	for i := 1; i <= cadenceFrames; i++ {
		a.step(cadenceInput(false, false, false), cadenceAt(from+i))
	}
	return int(v.AnimationCounter() - before), from + cadenceFrames
}

// TestOneRateReachesBothConsumers — 0041 SC-5 (AC-5): there is ONE rate
// and two readers of it. The water counter and the world clock fire the same
// number of ticks over the same driven second at every rate the ladder
// selects; doubling the rate doubles the counter's rise per driven second;
// and the stop crosses to the world and never to the water.
//
// THE COMPARISON IS BETWEEN TWO CLOCKS, not between a clock and a period read
// back out of it. A front-end that wrote the rate to one consumer and not the
// other leaves the two counts apart, which is what no arrangement of a single
// clock's own assertions could detect.
func TestOneRateReachesBothConsumers(t *testing.T) {
	// The map-load cadence, hand-computed: index 4 is 62 ms, widened to 62000 us,
	// so one second is 1000000/62000 = 16 whole ticks.
	const (
		loadPeriodUS = 62_000
		loadTicks    = 16
	)

	t.Run("an opened map agrees without a call", func(t *testing.T) {
		a, v, seam, at := cadenceApp(t)

		if n := len(seam.calls); n != 0 {
			t.Errorf("opening a map made %d cadence calls, want 0 — both sides are born at the map-load "+
				"speed, so an opened map needs no call to agree (DD-6)", n)
		}
		if got, want := seam.clock.Period(), loadPeriodUS; got != want {
			t.Fatalf("the far side stands at %d us, want the map-load %d", got, want)
		}
		if on, idx := v.Animation(); !on || idx != terrain.DefaultSpeedIndex {
			t.Errorf("an opened map reports animation %v at speed index %d, want true at %d",
				on, idx, terrain.DefaultSpeedIndex)
		}

		water, _ := driveSecond(t, a, v, at)
		world := seam.clock.AdvanceMicros(cadenceSecond)
		if water != loadTicks || world != loadTicks {
			t.Errorf("one driven second ran %d water ticks and %d world ticks, want %d each at the "+
				"map-load period", water, world, loadTicks)
		}
	})

	// AC-5's own case, re-aimed by 0061 (AC-4): a press moves BOTH readers to
	// the same new cadence, and the counter's rise per driven second is the one
	// that cadence names — MEASURED rather than inferred from the period.
	//
	// The presses walk up out of the shipped speed table and into our extension,
	// so both halves of the ladder are driven through the production path. Only
	// the rungs whose period divides a second EXACTLY carry a hand-computed
	// count: the accumulator carries a remainder across a re-rate, so at a rung
	// like 41000 us a driven second runs 24 ticks or 25 depending on where the
	// previous rung left it, and a fixed number there would be a fixture pinning
	// history rather than cadence. Where the count is not exact, "the two
	// consumers agree" is still asserted, and that is the claim this case is for.
	t.Run("a press moves both readers to the same cadence", func(t *testing.T) {
		a, v, seam, at := cadenceApp(t)

		// From the map-load rung: speed 6, 7, 8 and 9 of the shipped nine, then
		// the first two rungs of our extension past its fast end. The periods are
		// terrain's and are pinned by terrain's own hand-written ladder; what is
		// hand-written HERE is which press lands on which, and the tick counts.
		steps := []struct {
			periodUS int
			ticks    int // 0 where 1000000/periodUS is not whole
		}{
			{50_000, 20}, // speed index 5: 1000/20 = 50 ms
			{41_000, 0},  // speed index 6: 1000/24 = 41 ms, truncated
			{35_000, 0},  // speed index 7
			{31_000, 0},  // speed index 8, the table's fastest
			{15_625, 64}, // past it: our extension's first doubling, 64/s
			{7_812, 0},   // and its second, 128/s
		}
		for i, c := range steps {
			at++
			a.step(cadenceInput(false, true, false), cadenceAt(at))

			call, ok := seam.last()
			if !ok || call.periodUS != c.periodUS || call.stopped {
				t.Fatalf("press %d put %+v across the seam (seen %v), want %d us running",
					i+1, call, ok, c.periodUS)
			}
			if want := terrain.CadencePeriod(terrain.CadenceRung(c.periodUS)); want != c.periodUS {
				t.Fatalf("press %d expects %d us, which is on no rung of the ladder", i+1, c.periodUS)
			}

			// The far side's clock is advanced over the SAME span the front-end
			// just drove the water counter over, press frame included, so the two
			// carry the same remainder into the second and a difference between
			// their counts is a difference of cadence and not of bookkeeping.
			seam.clock.AdvanceMicros(cadenceFrameUS)
			water, next := driveSecond(t, a, v, at)
			at = next
			world := seam.clock.AdvanceMicros(cadenceSecond)
			if water != world {
				t.Fatalf("at %d us one driven second ran %d water ticks and %d world ticks — the two "+
					"consumers are not on one cadence (FR-5, C-2, DD-5)", c.periodUS, water, world)
			}
			if c.ticks != 0 && water != c.ticks {
				t.Fatalf("at %d us one driven second ran %d ticks, want %d", c.periodUS, water, c.ticks)
			}
		}
	})

	t.Run("player stop holds world and water at the same cadence", func(t *testing.T) {
		a, v, seam, at := cadenceApp(t)

		at++
		// One press up from the map-load rung: speed index 5 of the shipped nine,
		// 1000/20 = 50 ms, so a driven second is 20 ticks on the nose.
		const nextPeriodUS, nextTicks = 50_000, 20
		a.step(cadenceInput(false, true, false), cadenceAt(at))
		running, next := driveSecond(t, a, v, at)
		at = next
		if running != nextTicks {
			t.Fatalf("one driven second at %d us ran %d water ticks, want %d", nextPeriodUS, running, nextTicks)
		}

		at++
		a.step(cadenceInput(true, false, false), cadenceAt(at))
		call, _ := seam.last()
		if !call.stopped || call.periodUS != nextPeriodUS {
			t.Fatalf("the stop crossed as %+v, want %d us stopped — a stop must move no cadence",
				call, nextPeriodUS)
		}
		if !seam.stopped {
			t.Fatal("the far side is not holding the stop, so nothing was stopped at all")
		}

		stopped, next := driveSecond(t, a, v, at)
		at = next
		if stopped != 0 {
			t.Errorf("the water counter rose %d over a player-paused second, want zero", stopped)
		}

		at++
		a.step(cadenceInput(true, false, false), cadenceAt(at))
		call, _ = seam.last()
		if call.stopped || call.periodUS != nextPeriodUS {
			t.Fatalf("clearing the stop crossed as %+v, want %d us running", call, nextPeriodUS)
		}
		resumed, _ := driveSecond(t, a, v, at)
		if resumed != running {
			t.Errorf("the water counter rose %d over the second after the resume, want the same %d — the "+
				"toggle must move no rate (FR-3)", resumed, running)
		}
	})

	t.Run("the standalone viewer keeps its own speed, counter and snapshot", func(t *testing.T) {
		// Index 6 is 24 ticks a second, so 1000/24 = 41 ms and 41000 us; one
		// second is 1000000/41000 = 24 whole ticks. Chosen away from the map-load
		// index so a viewer that had been re-rated to anything else misses it.
		const (
			standaloneIdx    = 6
			standalonePerUS  = 41_000
			standaloneTicks  = 24
			standaloneFrames = cadenceFrames
		)
		if got := terrain.SpeedIndexPeriod(standaloneIdx); got != standalonePerUS {
			t.Fatalf("speed index %d is %d us, not the %d this case is written over", standaloneIdx, got, standalonePerUS)
		}

		sv, err := NewViewer("standalone", grid(60, 60), &terrain.Tileset{})
		if err != nil {
			t.Fatalf("NewViewer: %v", err)
		}
		layoutViewport(sv, DefaultWindowW, DefaultWindowH)
		sv.SetSpeedIndex(standaloneIdx)
		sv.SetAnimated(true)

		// Driven over the very frames on which the front-end beside it is having
		// its cadence keys pressed.
		a, _, seam, at := cadenceApp(t)
		sv.step(Input{CursorX: cadenceCX, CursorY: cadenceCY}, cadenceAt(at)) // the baseline
		for i := 1; i <= standaloneFrames; i++ {
			at++
			a.step(cadenceInput(i == 3, i == 7, i == 11), cadenceAt(at))
			sv.step(Input{CursorX: cadenceCX, CursorY: cadenceCY}, cadenceAt(at))
		}
		if len(seam.calls) == 0 {
			t.Fatal("the front-end beside the standalone viewer changed no cadence, so this case drives nothing")
		}

		if on, idx := sv.Animation(); !on || idx != standaloneIdx {
			t.Errorf("the standalone viewer reports animation %v at speed index %d, want true at %d — its own "+
				"speed selection, unchanged (FR-5)", on, idx, standaloneIdx)
		}
		if got := sv.AnimationCounter(); got != standaloneTicks {
			t.Errorf("the standalone viewer's counter stands at %d after a driven second, want %d — its own "+
				"counter at its own speed", got, standaloneTicks)
		}

		// The mechanism, not the outcome: Viewer.Input — the snapshot cmd/mapview
		// fills — carries no cadence field at all, so no key of this story can
		// reach the standalone viewer even by a caller that wanted it to.
		it := reflect.TypeOf(Input{})
		for _, banned := range []string{"Pause", "Faster", "Slower", "Stop", "Rate"} {
			if _, ok := it.FieldByName(banned); ok {
				t.Errorf("Viewer.Input carries a %q field; the standalone viewer's own snapshot must gain "+
					"none of this story's keys (FR-5, DD-7)", banned)
			}
		}
	})
}

// TestTheCadenceKeysActOnTheMapScreenAlone — 0041 SC-7 (AC-7): each of the
// three keys acts once per press and not while held, the ladder doubles and
// halves and clamps at both ends, a press moves no camera and changes no
// selection, it takes effect on the frame it is made on, and on every other
// screen all three do nothing.
func TestTheCadenceKeysActOnTheMapScreenAlone(t *testing.T) {
	// The rung an opened map's ladder starts from: the one the map-load speed's
	// own PERIOD stands on, so the first press is a step from exactly where the
	// map opened.
	bornRung := terrain.CadenceRung(terrain.SpeedIndexPeriod(terrain.DefaultSpeedIndex))

	// A press EDGE is what the snapshot carries, and inpututil.IsKeyJustPressed
	// is what fills it — so a held key delivers one true and then falses, which
	// is the sequence every "held" run below drives. Headless, with no key down,
	// the sampler reads all three as not pressed.
	t.Run("the sampler reads a key nobody is pressing as unpressed", func(t *testing.T) {
		in := readAppInput()
		if in.Pause || in.Faster || in.Slower {
			t.Errorf("readAppInput with no key down returned Pause %v, Faster %v, Slower %v, want all false",
				in.Pause, in.Faster, in.Slower)
		}
	})

	t.Run("each key acts once per press and not while held", func(t *testing.T) {
		a, _, seam, at := cadenceApp(t)

		at++
		a.step(cadenceInput(false, true, false), cadenceAt(at))
		if got := len(seam.calls); got != 1 {
			t.Fatalf("one press made %d cadence calls, want 1", got)
		}
		for i := 0; i < 6; i++ { // the key stays down; the edge does not fire again
			at++
			a.step(cadenceInput(false, false, false), cadenceAt(at))
		}
		if got := len(seam.calls); got != 1 {
			t.Errorf("holding the key made %d cadence calls, want the 1 the press made", got)
		}
		if got, want := mustLast(t, seam).periodUS, terrain.CadencePeriod(bornRung+1); got != want {
			t.Errorf("the cadence stands at %d us after one press held for six frames, want the next "+
				"rung's %d us", got, want)
		}

		// The stop toggles once per press, and back.
		at++
		a.step(cadenceInput(true, false, false), cadenceAt(at))
		if !seam.stopped {
			t.Fatal("one press of the stop key did not stop the world")
		}
		for i := 0; i < 4; i++ {
			at++
			a.step(cadenceInput(false, false, false), cadenceAt(at))
		}
		if !seam.stopped {
			t.Error("the stop cleared itself while the key was held")
		}
		at++
		a.step(cadenceInput(true, false, false), cadenceAt(at))
		if seam.stopped {
			t.Error("a second press of the stop key did not clear the stop")
		}
	})

	// 0061 SC-3 (AC-1, AC-4, AC-5): the keys walk the ladder ONE RUNG at a
	// time, through the production path, all the way to both ends - and the
	// cadence a map opened at is one of the rungs walked over.
	t.Run("the keys walk every rung and clamp at both ends", func(t *testing.T) {
		a, _, seam, at := cadenceApp(t)

		press := func(faster bool) {
			at++
			if faster {
				a.step(cadenceInput(false, true, false), cadenceAt(at))
				return
			}
			a.step(cadenceInput(false, false, true), cadenceAt(at))
		}

		// Up to the ceiling, one rung a press.
		for rung := bornRung + 1; rung <= terrain.CadenceRungMax; rung++ {
			press(true)
			if got, want := mustLast(t, seam).periodUS, terrain.CadencePeriod(rung); got != want {
				t.Fatalf("the press onto rung %d put %d us across the seam, want %d us", rung, got, want)
			}
		}
		if got, want := len(seam.calls), terrain.CadenceRungMax-bornRung; got != want {
			t.Fatalf("%d presses made %d cadence calls, want one each", want, got)
		}
		for i := 0; i < 3; i++ { // past the ceiling
			press(true)
		}
		if got, want := len(seam.calls), terrain.CadenceRungMax-bornRung; got != want {
			t.Errorf("pressing past the ceiling made %d cadence calls, want the %d before it - a press that "+
				"changes nothing writes nothing", got, want)
		}
		if got, want := mustLast(t, seam).periodUS, terrain.CadencePeriod(terrain.CadenceRungMax); got != want {
			t.Errorf("the ladder stands at %d us at the top, want the ceiling's %d us", got, want)
		}

		// ...and all the way back down to the floor, over every rung including
		// the nine the game itself offers.
		shipped := 0
		for rung := terrain.CadenceRungMax - 1; rung >= terrain.CadenceRungMin; rung-- {
			press(false)
			got, want := mustLast(t, seam).periodUS, terrain.CadencePeriod(rung)
			if got != want {
				t.Fatalf("the press onto rung %d put %d us across the seam, want %d us", rung, got, want)
			}
			if _, ok := terrain.SpeedIndexOf(got); ok {
				shipped++
			}
		}
		if want := terrain.SpeedIndexMax - terrain.SpeedIndexMin + 1; shipped != want {
			t.Errorf("the walk down passed through %d of the game's own speed settings, want all %d",
				shipped, want)
		}
		before := len(seam.calls)
		for i := 0; i < 3; i++ { // past the floor
			press(false)
		}
		if got := len(seam.calls); got != before {
			t.Errorf("pressing past the floor made %d cadence calls, want the %d before it", got, before)
		}
		if got, want := mustLast(t, seam).periodUS, terrain.CadencePeriod(terrain.CadenceRungMin); got != want {
			t.Errorf("the ladder stands at %d us at the bottom, want the floor's %d us", got, want)
		}
	})

	// 0061 SC-3 (AC-3): THE ROUND TRIP, driven through App.step - the very keys
	// the owner presses. n presses of one key and n of the other, from the
	// cadence a map opens at, come back to it for every n whose run does not
	// reach an end; the runs that do reach one are stated too, because
	// saturating is what an end does and the owner will find it.
	t.Run("n presses each way come back to the cadence the map opened at", func(t *testing.T) {
		open := terrain.SpeedIndexPeriod(terrain.DefaultSpeedIndex)
		for n := 0; n <= terrain.CadenceRungMax-terrain.CadenceRungMin+2; n++ {
			a, _, seam, at := cadenceApp(t)
			for i := 0; i < n; i++ {
				at++
				a.step(cadenceInput(false, true, false), cadenceAt(at))
			}
			for i := 0; i < n; i++ {
				at++
				a.step(cadenceInput(false, false, true), cadenceAt(at))
			}

			up := bornRung + n
			if up > terrain.CadenceRungMax {
				up = terrain.CadenceRungMax
			}
			back := up - n
			if back < terrain.CadenceRungMin {
				back = terrain.CadenceRungMin
			}
			want := terrain.CadencePeriod(back)

			if n == 0 {
				if len(seam.calls) != 0 {
					t.Fatalf("no press made %d cadence calls", len(seam.calls))
				}
				continue
			}
			if got := mustLast(t, seam).periodUS; got != want {
				t.Fatalf("%d presses up then %d down left %d us, want %d us", n, n, got, want)
			}
			if bornRung+n <= terrain.CadenceRungMax && want != open {
				t.Fatalf("%d presses each way is a run that never leaves the ladder, so it must return to "+
					"the map-load %d us; the fixture expects %d us", n, open, want)
			}
		}
	})

	t.Run("a press moves no camera and changes no selection", func(t *testing.T) {
		a, v, _, at := cadenceApp(t)
		v.Camera().X, v.Camera().Y = 400, 400
		v.Camera().Clamp()
		v.sel = selection{3, 9}

		camX, camY, zoom := v.Camera().X, v.Camera().Y, v.Camera().Zoom
		for _, in := range []appInput{
			cadenceInput(true, false, false),
			cadenceInput(false, true, false),
			cadenceInput(false, false, true),
			cadenceInput(true, true, true),
		} {
			at++
			a.step(in, cadenceAt(at))
		}

		if v.Camera().X != camX || v.Camera().Y != camY || v.Camera().Zoom != zoom {
			t.Errorf("the camera moved from (%v,%v) zoom %v to (%v,%v) zoom %v over four cadence presses",
				camX, camY, zoom, v.Camera().X, v.Camera().Y, v.Camera().Zoom)
		}
		if !reflect.DeepEqual([]uint32(v.sel), []uint32{3, 9}) {
			t.Errorf("the selection is %v after four cadence presses, want the {3 9} it held", v.sel)
		}
	})

	t.Run("a press takes effect before the same frame's advance", func(t *testing.T) {
		a, _, seam, at := cadenceApp(t)
		seam.log = nil

		at++
		a.step(cadenceInput(true, false, false), cadenceAt(at))
		if got, want := seam.log, []string{"cadence", "tick"}; !reflect.DeepEqual(got, want) {
			t.Errorf("the frame of the press ran %v, want %v — a press that reached the seam after the "+
				"advance would take effect one frame late", got, want)
		}

		seam.log = nil
		at++
		a.step(cadenceInput(false, false, false), cadenceAt(at))
		if got, want := seam.log, []string{"tick"}; !reflect.DeepEqual(got, want) {
			t.Errorf("a frame with no press ran %v, want %v — the cadence call fires on a change, never "+
				"per tick (DD-6)", got, want)
		}
	})

	t.Run("on every other screen all three do nothing", func(t *testing.T) {
		var seam *cadenceSeam
		a := newTestApp(t, appRows(3), cadenceLoader(t, &seam))

		// The menu.
		for i := 0; i < 3; i++ {
			a.step(cadenceInput(true, true, true), cadenceAt(i+1))
		}
		if a.Screen() != ScreenMenu {
			t.Fatalf("three cadence presses on the menu left screen %v, want ScreenMenu", a.Screen())
		}
		if seam != nil {
			t.Fatal("a cadence press on the menu loaded a map")
		}
		if a.flow.rung != 0 || a.flow.stopped {
			t.Errorf("the menu's cadence presses left rung %d stopped %v, want the untouched zero value",
				a.flow.rung, a.flow.stopped)
		}

		// The picker.
		a.flow.screen = ScreenPicker
		for i := 0; i < 3; i++ {
			a.step(cadenceInput(true, true, true), cadenceAt(i+4))
		}
		if a.Screen() != ScreenPicker {
			t.Fatalf("three cadence presses on the picker left screen %v, want ScreenPicker", a.Screen())
		}
		if seam != nil {
			t.Fatal("a cadence press on the picker loaded a map")
		}

		// ...and back on the picker after a map screen was open and left, where a
		// front-end that read the keys off the wrong arm would still be reading
		// them.
		a.step(appInput{Enter: true}, cadenceAt(10))
		if a.Screen() != ScreenMap || seam == nil {
			t.Fatalf("setup: screen %v, seam %v — the map must be open for the case below", a.Screen(), seam)
		}
		a.step(cadenceInput(false, true, false), cadenceAt(11))
		rung, stopped, calls := a.flow.rung, a.flow.stopped, len(seam.calls)
		if calls != 1 {
			t.Fatalf("setup: %d cadence calls on the map screen, want 1", calls)
		}

		a.step(appInput{Escape: true}, cadenceAt(12))
		leaveViaMenu(a.flow) //
		if a.Screen() != ScreenPicker {
			t.Fatalf("Esc left screen %v, want ScreenPicker", a.Screen())
		}
		for i := 0; i < 4; i++ {
			a.step(cadenceInput(true, true, true), cadenceAt(13+i))
		}
		if a.flow.rung != rung || a.flow.stopped != stopped {
			t.Errorf("the picker's cadence presses moved the front-end's cadence from (%d, %v) to (%d, %v)",
				rung, stopped, a.flow.rung, a.flow.stopped)
		}
		if got := len(seam.calls); got != calls {
			t.Errorf("the picker's cadence presses made %d further cadence calls, want 0", got-calls)
		}
	})
}

// stoppedGesture is the schedule both legs below are driven on, one appInput per
// frame: a keyboard pan, a wheel zoom, then a left press, two moves and a
// release that draws a rectangle between the first and the last cursor point.
//
// It is ONE slice consumed by both legs, so "the same input" is a shared value
// rather than two transcriptions that could drift.
func stoppedGesture() []appInput {
	at := func(x, y int) appInput {
		return appInput{Viewer: Input{CursorX: x, CursorY: y}, CursorX: x, CursorY: y}
	}
	press := at(stoppedPressX, stoppedPressY)
	press.PrimaryPressed = true
	press.Viewer.PrimaryDown = true

	move1 := at(140, 130)
	move1.Viewer.PrimaryDown = true
	move2 := at(stoppedReleaseX, stoppedReleaseY)
	move2.Viewer.PrimaryDown = true

	release := at(stoppedReleaseX, stoppedReleaseY)
	release.PrimaryReleased = true

	pan := at(cadenceCX, cadenceCY)
	pan.Viewer.PanRight = true

	zoom := at(cadenceCX, cadenceCY)
	zoom.Viewer.WheelY = +1

	return []appInput{pan, zoom, press, move1, move2, release}
}

// The gesture's two corners, in screen pixels, and the camera position both legs
// start from.
const (
	stoppedPressX, stoppedPressY     = 80, 80
	stoppedReleaseX, stoppedReleaseY = 200, 175
	stoppedCamX, stoppedCamY         = 400, 400
	stoppedFarCol, stoppedFarRow     = 58, 58 // on the 60x60 grid, nowhere near the rectangle
)

// TestTheStopStopsNoCameraNoSelectionAndNoOutline — 0041 SC-4's front-end
// clauses (AC-4): stopped, a pan moves the camera by its UNSTOPPED delta, a
// wheel zoom zooms, the outline pass is built mid-drag, and a box release
// replaces the selection exactly as it does unstopped.
//
// TWO LEGS OVER ONE INPUT SCHEDULE, differing only in that one of them was
// stopped first. Every quantity is compared BETWEEN the legs rather than against
// a number written here, which is what makes "by its unstopped delta" the
// measurement rather than a paraphrase; and each is also checked to have MOVED,
// so two frozen legs cannot agree their way to a pass.
//
// The fixture's cells are RESOLVED through the camera after the pan and the zoom
// rather than written out: that places the units under the gesture's own two
// corners. It predicts nothing about what the release will select — which cells a
// rectangle covers is 0030's contract and is witnessed there — and both legs
// resolve the same cells, which the camera comparison below independently says.
func TestTheStopStopsNoCameraNoSelectionAndNoOutline(t *testing.T) {
	leg := func(t *testing.T, stop bool) (*Viewer, *cadenceSeam, []screenRect) {
		t.Helper()
		a, v, seam, at := cadenceApp(t)
		v.Camera().X, v.Camera().Y = stoppedCamX, stoppedCamY
		v.Camera().Clamp()

		at++
		a.step(cadenceInput(stop, false, false), cadenceAt(at))
		if seam.stopped != stop {
			t.Fatalf("the far side holds stopped=%v, want %v", seam.stopped, stop)
		}

		gesture := stoppedGesture()
		for _, in := range gesture[:2] { // the keyboard pan and the wheel zoom
			at++
			a.step(in, cadenceAt(at))
		}

		col0, row0, ok0 := v.Camera().ScreenToCell(stoppedPressX, stoppedPressY)
		col1, row1, ok1 := v.Camera().ScreenToCell(stoppedReleaseX, stoppedReleaseY)
		if !ok0 || !ok1 || (col0 == col1 && row0 == row1) {
			t.Fatalf("the gesture's corners resolved to (%d,%d) inside=%v and (%d,%d) inside=%v; both must be "+
				"on the map and on different cells", col0, row0, ok0, col1, row1, ok1)
		}
		v.SetEntities([]MapEntity{
			{ID: 4, Cell: image.Pt(col0, row0)},
			{ID: 9, Cell: image.Pt(col1, row1)},
			{ID: 11, Cell: image.Pt(stoppedFarCol, stoppedFarRow)},
		})

		var outline []screenRect
		for i, in := range gesture[2:] {
			at++
			a.step(in, cadenceAt(at))
			if i == 2 { // the last move, before the release
				outline = marqueeRects(v)
			}
		}
		return v, seam, outline
	}

	running, runSeam, runOutline := leg(t, false)
	frozen, stopSeam, stopOutline := leg(t, true)

	if runSeam.stopped || !stopSeam.stopped {
		t.Fatalf("the legs ran with stopped=%v and %v, want false and true", runSeam.stopped, stopSeam.stopped)
	}

	// The camera moved at all, and moved the same way.
	if running.Camera().X == stoppedCamX && running.Camera().Y == stoppedCamY && running.Camera().Zoom == 1 {
		t.Fatal("the unstopped leg's camera did not move, so the comparison below discriminates nothing")
	}
	if frozen.Camera().X != running.Camera().X || frozen.Camera().Y != running.Camera().Y {
		t.Errorf("the stopped leg's camera is at (%v,%v) and the running leg's at (%v,%v) — a pan and a drag "+
			"must move the camera by their unstopped delta (FR-4)",
			frozen.Camera().X, frozen.Camera().Y, running.Camera().X, running.Camera().Y)
	}
	if frozen.Camera().Zoom != running.Camera().Zoom {
		t.Errorf("the stopped leg zoomed to %v and the running leg to %v (FR-4)",
			frozen.Camera().Zoom, running.Camera().Zoom)
	}

	// The outline was built mid-drag, and the same one.
	if len(runOutline) == 0 {
		t.Fatal("the unstopped leg built no outline mid-drag, so the comparison below discriminates nothing")
	}
	if !reflect.DeepEqual(stopOutline, runOutline) {
		t.Errorf("the stopped leg's outline is %+v and the running leg's %+v — the rectangle must be drawn "+
			"while the world is stopped (FR-4)", stopOutline, runOutline)
	}

	// The release replaced the selection, and replaced it the same way.
	if len(running.sel) == 0 {
		t.Fatal("the unstopped leg's box release selected nothing, so the comparison below discriminates nothing")
	}
	if !reflect.DeepEqual([]uint32(frozen.sel), []uint32(running.sel)) {
		t.Errorf("the stopped leg selected %v and the running leg %v (FR-4)", frozen.sel, running.sel)
	}
	for _, id := range running.sel {
		if id == 11 {
			t.Errorf("the release selected the unit far outside the rectangle, so it covered the whole " +
				"snapshot and says nothing about which cells it caught")
		}
	}
}
