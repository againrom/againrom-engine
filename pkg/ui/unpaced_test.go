package ui

import (
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
)

func ownerCadenceInput(unpaced, paced bool) appInput {
	in := cadenceInput(false, false, false)
	in.Unpaced = unpaced
	in.Paced = paced
	in.Viewer.Ctrl = true
	return in
}

func TestUnpacedBindingUsesOnlyTheCtrlNumpadPair(t *testing.T) {
	got := bindingSource(t)
	for _, tc := range []struct {
		field string
		key   string
	}{
		{field: "Unpaced", key: "ebiten.KeyNumpadAdd"},
		{field: "Paced", key: "ebiten.KeyNumpadSubtract"},
	} {
		expr, ok := got[tc.field]
		if !ok {
			t.Fatalf("readAppInput has no %s binding", tc.field)
		}
		if !strings.Contains(expr, "inpututil.IsKeyJustPressed("+tc.key+")") {
			t.Errorf("%s binding = %q, want a %s press edge", tc.field, expr, tc.key)
		}
		if !strings.Contains(expr, "ctrlHeld()") || strings.Contains(expr, "!ctrlHeld()") {
			t.Errorf("%s binding = %q, want Ctrl held", tc.field, expr)
		}
		if strings.Contains(expr, "ebiten.KeyEqual") || strings.Contains(expr, "ebiten.KeyMinus)") {
			t.Errorf("%s binding = %q, want the numpad key alone", tc.field, expr)
		}
	}

	// The original ordinary pair stays disjoint from the modified pair.
	if expr := got["Faster"]; !strings.Contains(expr, "!ctrlHeld()") {
		t.Errorf("Faster binding = %q, want bare numpad + guarded from Ctrl", expr)
	}
	if expr := got["Slower"]; !strings.Contains(expr, "!ctrlHeld()") {
		t.Errorf("Slower binding = %q, want bare numpad - guarded from Ctrl", expr)
	}
}

func TestUnpacedMapFramesAdvanceOnceWithoutADeadlineAndRestoreCleanly(t *testing.T) {
	a, v, seam, at := cadenceApp(t)
	base := cadenceAt(at)
	before := v.AnimationCounter()

	// The press takes effect before this frame's advance. A timestamp equal to
	// the baseline would fire no paced tick, so this first +1 discriminates the
	// separate deadline-free arm.
	a.step(ownerCadenceInput(true, false), base)
	if call := mustLast(t, seam); !call.unpaced || call.stopped || call.reset {
		t.Fatalf("Ctrl+numpad + crossed as %+v, want unpaced and running", call)
	}
	if got := v.AnimationCounter() - before; got != 1 {
		t.Fatalf("the selecting frame advanced ambient ticks by %d, want exactly 1", got)
	}

	// Same instant, backwards instant, and a day-long gap are deliberately
	// incomparable as deadlines. Each is one eligible map frame and therefore
	// advances exactly once.
	instants := []time.Time{base, base.Add(-time.Hour), base.Add(24 * time.Hour)}
	for i, now := range instants {
		prior := v.AnimationCounter()
		a.step(ownerCadenceInput(false, false), now)
		if got := v.AnimationCounter() - prior; got != 1 {
			t.Fatalf("unpaced frame %d advanced %d ticks, want 1", i+1, got)
		}
	}
	if got := len(seam.calls); got != 1 {
		t.Fatalf("held/idempotent unpaced frames made %d cadence writes, want the selecting write alone", got)
	}

	// A bare + still walks the stored ordinary ladder while the separate arm is
	// selected. That period is what the existing readout and later restore keep.
	oldPeriod := seam.clock.Period()
	a.step(cadenceInput(false, true, false), base.Add(24*time.Hour+time.Millisecond))
	if call := mustLast(t, seam); !call.unpaced || call.periodUS >= oldPeriod {
		t.Fatalf("bare + under unpaced crossed as %+v from %d us, want a faster stored rung and unpaced still set", call, oldPeriod)
	}
	storedPeriod := seam.clock.Period()

	// Clearing the arm resets the viewer baseline and phase. The restore frame
	// fires no ambient tick; a whole stored period is required afterwards.
	restoreAt := base.Add(48 * time.Hour)
	prior := v.AnimationCounter()
	a.step(ownerCadenceInput(false, true), restoreAt)
	if call := mustLast(t, seam); call.unpaced || !call.reset || call.periodUS != storedPeriod {
		t.Fatalf("Ctrl+numpad - crossed as %+v, want paced at stored period %d", call, storedPeriod)
	}
	if got := v.AnimationCounter() - prior; got != 0 {
		t.Fatalf("restore frame advanced %d ambient ticks, want 0 from a fresh baseline", got)
	}
	a.step(ownerCadenceInput(false, false), restoreAt.Add(time.Duration(storedPeriod-1)*time.Microsecond))
	if got := v.AnimationCounter() - prior; got != 0 {
		t.Fatalf("less than one restored period advanced %d ticks, want 0", got)
	}
	a.step(ownerCadenceInput(false, false), restoreAt.Add(time.Duration(storedPeriod)*time.Microsecond))
	if got := v.AnimationCounter() - prior; got != 1 {
		t.Fatalf("one restored period advanced %d ticks, want 1", got)
	}

	// Minus carries a phase/epoch reset even when the mode is already paced.
	// Seed half a period, press it again, and verify the explicit reset crossed
	// instead of being optimised away as an idempotent state write.
	halfAt := restoreAt.Add(time.Duration(storedPeriod+storedPeriod/2) * time.Microsecond)
	a.step(ownerCadenceInput(false, false), halfAt)
	if got := v.anim.Remainder(); got != storedPeriod/2 {
		t.Fatalf("setup paced remainder = %d, want %d", got, storedPeriod/2)
	}
	prior = v.AnimationCounter()
	a.step(ownerCadenceInput(false, true), halfAt)
	if call := mustLast(t, seam); call.unpaced || !call.reset {
		t.Fatalf("Ctrl+numpad - while already paced crossed as %+v, want an explicit paced reset", call)
	}
	if got := v.anim.Remainder(); got != 0 {
		t.Fatalf("paced reset left remainder %d, want 0", got)
	}
	if got := v.AnimationCounter() - prior; got != 0 {
		t.Fatalf("paced reset frame advanced %d ticks, want 0", got)
	}
}

func TestUnpacedSelectionObeysStopPopupAndNonMapBoundaries(t *testing.T) {
	a, v, seam, at := cadenceApp(t)

	// A player stop and owner-loop selection can arrive together. Both the
	// world and its ambient presentation retain their frame under that stop.
	before := v.AnimationCounter()
	in := ownerCadenceInput(true, false)
	in.Pause = true
	at++
	a.step(in, cadenceAt(at))
	if call := mustLast(t, seam); !call.unpaced || !call.stopped {
		t.Fatalf("stopped unpaced selection crossed as %+v", call)
	}
	if got := v.AnimationCounter() - before; got != 0 {
		t.Fatalf("player-stopped unpaced frame advanced ambient by %d, want 0", got)
	}

	// A modal surface consumes the clear command. syncCadence may write its
	// stop, but the requested owner mode itself must not move.
	v.SetFont(panelFont())
	v.SetNotice("modal", NoticeDialogue)
	at++
	a.step(ownerCadenceInput(false, true), cadenceAt(at))
	if !a.flow.unpaced {
		t.Fatal("a modal Ctrl+numpad - cleared the owner-loop mode")
	}
	v.ClearNotice()

	// Town and load/text-like screens have no cadence arm at all, even if a
	// stale viewer and its world seam are still retained by the fixture.
	for _, screen := range []Screen{ScreenTown, ScreenLoad} {
		a.flow.screen = screen
		at++
		a.step(ownerCadenceInput(false, true), cadenceAt(at))
		if !a.flow.unpaced {
			t.Fatalf("%s consumed Ctrl+numpad - as a map cadence command", screen)
		}
	}
}

func TestOrdinaryCadenceReadoutKeepsTheStoredLadderUnderUnpaced(t *testing.T) {
	a, v, seam, at := readoutKeyApp(t)
	openRate, _ := statedRate(t, v)

	at++
	a.step(ownerCadenceInput(true, false), cadenceAt(at))
	if rate, stopped := statedRate(t, v); rate != openRate || stopped {
		t.Fatalf("selecting unpaced changed the ordinary readout to %d/s stopped=%v, want stored %d/s running", rate, stopped, openRate)
	}

	at++
	a.step(cadenceInput(false, true, false), cadenceAt(at))
	stored := terrain.RateOf(seam.clock.Period())
	if rate, _ := statedRate(t, v); rate != stored || rate == openRate {
		t.Fatalf("bare + under unpaced made readout %d/s, want new stored ladder rate %d/s", rate, stored)
	}

	at++
	a.step(ownerCadenceInput(false, true), cadenceAt(at))
	if rate, _ := statedRate(t, v); rate != stored {
		t.Fatalf("restoring paced mode made readout %d/s, want stored %d/s", rate, stored)
	}
}

func TestNormalCadencePreferenceRestoresPersistsAndNeverRestoresUnpaced(t *testing.T) {
	start := terrain.CadenceShippedHi
	var seam *cadenceSeam
	var persisted []int
	a := newTestApp(t, appRows(3), cadenceLoader(t, &seam))
	a.SetMapCadencePreference(start, func(rung int) {
		persisted = append(persisted, rung)
	})
	if got := a.MapCadencePreference(); got != start {
		t.Fatalf("installed preference = rung %d, want %d", got, start)
	}

	a.flow.screen = ScreenPicker
	a.step(appInput{Enter: true}, cadenceAt(0))
	if a.Screen() != ScreenMap || seam == nil {
		t.Fatalf("opening stored speed reached screen %v seam %v", a.Screen(), seam)
	}
	if len(seam.calls) != 1 {
		t.Fatalf("stored nondefault entry crossed %d cadence calls, want exactly 1", len(seam.calls))
	}
	if call := seam.calls[0]; call.periodUS != terrain.CadencePeriod(start) || call.unpaced || call.stopped || call.reset {
		t.Fatalf("stored entry crossed as %+v, want paced rung %d", call, start)
	}
	if got := a.flow.viewer.anim.Period(); got != terrain.CadencePeriod(start) {
		t.Fatalf("viewer opened at %d us, want stored %d us", got, terrain.CadencePeriod(start))
	}
	if len(persisted) != 0 {
		t.Fatalf("merely restoring the preference wrote it again: %v", persisted)
	}

	// The unpaced selector remains session-only. A bare + while that arm is
	// selected still changes and persists the normal rung waiting underneath.
	a.step(ownerCadenceInput(true, false), cadenceAt(1))
	if !a.flow.unpaced || len(persisted) != 0 {
		t.Fatalf("selecting unpaced left mode=%v persisted=%v, want true and no write", a.flow.unpaced, persisted)
	}
	want := start + 1
	a.step(cadenceInput(false, true, false), cadenceAt(2))
	if got := a.MapCadencePreference(); got != want {
		t.Fatalf("bare + under unpaced stored rung %d, want %d", got, want)
	}
	if len(persisted) != 1 || persisted[0] != want {
		t.Fatalf("bare + persistence = %v, want [%d]", persisted, want)
	}
	if call := mustLast(t, seam); call.periodUS != terrain.CadencePeriod(want) || !call.unpaced {
		t.Fatalf("bare + under unpaced crossed as %+v, want stored rung %d with arm retained", call, want)
	}

	// A new map in the same App restores the updated normal rung but starts in
	// paced mode. Entry is a read/apply, not another persistence write.
	a.flow.leaveMap()
	a.flow.screen = ScreenPicker
	a.step(appInput{Enter: true}, cadenceAt(3))
	if a.Screen() != ScreenMap || seam == nil {
		t.Fatalf("reopen reached screen %v seam %v", a.Screen(), seam)
	}
	if a.flow.unpaced {
		t.Fatal("new map restored the previous map's unpaced selector")
	}
	if len(seam.calls) != 1 {
		t.Fatalf("reopen crossed %d cadence calls, want one restored rung", len(seam.calls))
	}
	if call := seam.calls[0]; call.periodUS != terrain.CadencePeriod(want) || call.unpaced || call.reset {
		t.Fatalf("reopen crossed as %+v, want ordinary stored rung %d", call, want)
	}
	if len(persisted) != 1 {
		t.Fatalf("reopen rewrote preference: %v", persisted)
	}
}
