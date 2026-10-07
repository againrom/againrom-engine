package sim

import "testing"

// These tests are asset-free: motionFixture1115 builds the whole scenario by
// hand, so nothing here requires a lawful install or the save corpus.

// TestSavedMotion1134ContinuationFromImportWalksToRouteGoal proves the
// import-time entry point (a motion that starts already centered, so
// advanceSavedMotion never runs for it) hands the saved route to native
// movement, and that ordinary Step()s walk it all the way to the route's
// last cell and end the order there (restAt), the same as any other native
// destination.
func TestSavedMotion1134ContinuationFromImportWalksToRouteGoal(t *testing.T) {
	w, m, cs, bs := motionFixture1115(t, 128, 128, 0, 0, 0, 0, 0)
	m.StaticRoute = []uint16{uint16(16)<<8 | 16, uint16(16)<<8 | 17}
	m.DynamicRoute = nil
	importMotion1115(t, w, m, cs, bs)

	motions, _, _, present := w.SavedActorMotions()
	if !present || motions[0].Issue != "native movement continues the imported route" || motions[0].Current || motions[0].Active {
		t.Fatalf("continuation should activate at import: present=%v issue=%q current=%v active=%v",
			present, motions[0].Issue, motions[0].Current, motions[0].Active)
	}
	at := indexOfEntity(w.entities, motions[0].Entity)
	if e := w.entities[at]; !e.HasTarget || e.TargetX != 17 || e.TargetY != 16 {
		t.Fatalf("native target should be the route's last cell: hasTarget=%v x=%d y=%d", e.HasTarget, e.TargetX, e.TargetY)
	}

	arrived := false
	for i := 0; i < 400; i++ {
		Step(w, nil)
		at = indexOfEntity(w.entities, EntityID(7))
		if e := w.entities[at]; !e.HasTarget {
			arrived = true
			if e.X != 17 || e.Y != 16 {
				t.Fatalf("order ended away from the route's last cell: x=%d y=%d", e.X, e.Y)
			}
			break
		}
	}
	if !arrived {
		t.Fatal("unit never reached the saved route's goal and ended its order")
	}
	// The carried fields still read exactly as imported: continuation walks a
	// copy in w.routes, never the SavedActorMotion's own StaticRoute/DynamicRoute.
	after, _, _, _ := w.SavedActorMotions()
	if len(after[0].StaticRoute) != 2 {
		t.Fatalf("carried static route should stay untouched by walking it: %v", after[0].StaticRoute)
	}
}

// TestSavedMotion1134ContinuationPrefersDynamicOverStatic proves DIV-951's
// policy: when a motion's DynamicRoute survives trimming non-empty, that is
// the whole walked route; StaticRoute is left carried and unconsulted, never
// appended after it. A first candidate concatenated the two lists in this
// same order; a real release-corpus save produced a dynamic-to-static seam
// of Chebyshev distance 2, which the wire form's own adjacency refusal
// (binary.go's neighbours()) caught — the direct evidence, not merely an
// absent reading, that concatenation is the wrong policy (savedmotionroute.go).
func TestSavedMotion1134ContinuationPrefersDynamicOverStatic(t *testing.T) {
	w, m, cs, bs := motionFixture1115(t, 128, 128, 0, 0, 0, 0, 0)
	m.DynamicRoute = []uint16{uint16(17)<<8 | 17}
	m.StaticRoute = []uint16{uint16(18)<<8 | 18, uint16(19)<<8 | 19}
	importMotion1115(t, w, m, cs, bs)

	at := indexOfEntity(w.entities, EntityID(7))
	got := w.routes[at]
	want := []cell{{x: 17, y: 17}}
	if len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("route should be the dynamic list alone: got %v want %v", got, want)
	}
	if e := w.entities[at]; e.TargetX != 17 || e.TargetY != 17 {
		t.Fatalf("target should be the dynamic route's own last cell, not the static one's: x=%d y=%d", e.TargetX, e.TargetY)
	}
	after, _, _, _ := w.SavedActorMotions()
	if len(after[0].StaticRoute) != 2 || after[0].StaticRoute[0] != uint16(18)<<8|18 || after[0].StaticRoute[1] != uint16(19)<<8|19 {
		t.Fatalf("unconsulted static route should stay carried untouched: %v", after[0].StaticRoute)
	}
}

// TestSavedMotion1134ContinuationTrimsLeadingCurrentCell proves a leading run
// of route elements equal to the unit's own current cell is dropped before
// the remainder is handed to native movement as a sub-goal, rather than
// asking the near search to walk a zero-distance first step. Here that
// leaves DynamicRoute empty, so this also exercises the StaticRoute fallback:
// trimming, not merely emptiness, is what triggers it.
func TestSavedMotion1134ContinuationTrimsLeadingCurrentCell(t *testing.T) {
	w, m, cs, bs := motionFixture1115(t, 128, 128, 0, 0, 0, 0, 0)
	here := uint16(16)<<8 | 15 // matches motionFixture1115's own Position.Cell (0x100f)
	m.StaticRoute = []uint16{here, uint16(16)<<8 | 16}
	m.DynamicRoute = []uint16{here}
	importMotion1115(t, w, m, cs, bs)

	at := indexOfEntity(w.entities, EntityID(7))
	got := w.routes[at]
	if len(got) != 1 || got[0] != (cell{x: 16, y: 16}) {
		t.Fatalf("leading current-cell entries should be trimmed: %v", got)
	}
}

// TestSavedMotion1134ContinuationFromArrivalNeedingNoBoundaryTransition
// proves the second entry point: a crossing that finishes without ever
// changing native cell (so it carries none of advanceSavedMotion's own
// boundary-transition Issue text) hands its saved route to native movement
// the moment it arrives, exactly as the centered-at-import case does.
func TestSavedMotion1134ContinuationFromArrivalNeedingNoBoundaryTransition(t *testing.T) {
	// fineY already centered; fineX 120->128 in one tick of stepX=8 never
	// overflows a cell boundary, so next.Cell == Position.Cell throughout.
	w, m, cs, bs := motionFixture1115(t, 120, 128, 8, 0, 2, 0, 1)
	m.StaticRoute = []uint16{uint16(17)<<8 | 17}
	m.DynamicRoute = nil
	importMotion1115(t, w, m, cs, bs)

	motions, _, _, present := w.SavedActorMotions()
	if !present || !motions[0].Active {
		t.Fatalf("crossing should still be in flight at import: present=%v active=%v", present, motions[0].Active)
	}
	Step(w, nil)

	after, _, _, _ := w.SavedActorMotions()
	if after[0].Issue != "native movement continues the imported route" || after[0].Current || after[0].Active {
		t.Fatalf("clean arrival should hand off to continuation: issue=%q current=%v active=%v",
			after[0].Issue, after[0].Current, after[0].Active)
	}
	at := indexOfEntity(w.entities, EntityID(7))
	if e := w.entities[at]; !e.HasTarget || e.TargetX != 17 || e.TargetY != 17 {
		t.Fatalf("native target should be the route's last cell: hasTarget=%v x=%d y=%d", e.HasTarget, e.TargetX, e.TargetY)
	}
}

// TestSavedMotion1134ContinuationSurvivesBinaryRoundTrip proves the state
// beginSavedRouteContinuation leaves behind — the cleared saved-motion
// authority, the native target and the copied w.routes entry — is exactly
// what a MarshalBinary/UnmarshalBinary round trip reproduces, the same
// mechanism a native .ags resume (pkg/game/resume.go) uses to carry a
// mission across a session boundary.
func TestSavedMotion1134ContinuationSurvivesBinaryRoundTrip(t *testing.T) {
	w, m, cs, bs := motionFixture1115(t, 128, 128, 0, 0, 0, 0, 0)
	m.StaticRoute = []uint16{uint16(16)<<8 | 16, uint16(16)<<8 | 17}
	m.DynamicRoute = nil
	importMotion1115(t, w, m, cs, bs)

	motions, _, _, present := w.SavedActorMotions()
	if !present || motions[0].Current || motions[0].Active {
		t.Fatalf("continuation should have already cleared saved-motion authority: present=%v current=%v active=%v",
			present, motions[0].Current, motions[0].Active)
	}
	at := indexOfEntity(w.entities, EntityID(7))
	if !w.entities[at].HasTarget || len(w.routes[at]) != 2 {
		t.Fatalf("expected a native target and a two-cell route before marshal: hasTarget=%v route=%v",
			w.entities[at].HasTarget, w.routes[at])
	}

	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if back.Hash() != w.Hash() {
		t.Fatal("route continuation state did not survive a binary round trip")
	}
	backAt := indexOfEntity(back.entities, EntityID(7))
	if !back.entities[backAt].HasTarget || len(back.routes[backAt]) != 2 {
		t.Fatalf("native target/route should survive independently of Hash: hasTarget=%v route=%v",
			back.entities[backAt].HasTarget, back.routes[backAt])
	}
}

// A crossing finishes before its pending turn; route handoff waits for both.
func TestSavedMotion1134PendingTurnDefersContinuation(t *testing.T) {
	w, m, cs, bs := motionFixture1115(t, 120, 128, 8, 0, 2, 0, 1)
	m.Mover[0xa0] = 1 // pendingSavedMotionTurn's second operand
	m.StaticRoute = []uint16{uint16(17)<<8 | 17}
	m.DynamicRoute = []uint16{uint16(18)<<8 | 18}
	importMotion1115(t, w, m, cs, bs)
	Step(w, nil)

	after, _, _, _ := w.SavedActorMotions()
	if after[0].Issue != "" || after[0].Active || !w.ActorMotionActive(7) {
		t.Fatalf("pending turn should defer continuation: issue=%q active=%v", after[0].Issue, after[0].Active)
	}
	if len(after[0].DynamicRoute) != 0 {
		t.Fatalf("DynamicRoute should still be spent on a deferred arrival: %v", after[0].DynamicRoute)
	}
	at := indexOfEntity(w.entities, EntityID(7))
	if e := w.entities[at]; e.HasTarget {
		t.Fatalf("no native target should be given while continuation is deferred: x=%d y=%d", e.TargetX, e.TargetY)
	}
	Step(w, nil)
	if w.ActorMotionActive(7) || !w.entities[at].HasTarget {
		t.Fatal("finished crossing turn did not hand off the remaining static route")
	}
}
