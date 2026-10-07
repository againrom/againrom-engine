package sim

import "testing"

// TestSavedMotion1134OutOfBoundsRouteCellRefusesAdmission proves a saved
// route naming a cell the native map does not have marks the motion
// Issue/inactive rather than being consumed uncontested and indexing outside
// the map later (MOVE-ROUTE-004's packed cell, confirmed by SAV-630's own
// save/load arm reading; motionFixture1115's map is Bounds{32,32}).
func TestSavedMotion1134OutOfBoundsRouteCellRefusesAdmission(t *testing.T) {
	const wantIssue = "saved route names a cell outside the native map"
	for _, tc := range []struct {
		name  string
		apply func(*SavedActorMotion)
	}{
		{"static-x", func(m *SavedActorMotion) { m.StaticRoute = []uint16{250} }},          // x=250
		{"static-y", func(m *SavedActorMotion) { m.StaticRoute = []uint16{250 << 8} }},     // y=250
		{"dynamic", func(m *SavedActorMotion) { m.DynamicRoute = []uint16{250<<8 | 250} }}, // x=250,y=250
		{"second-element", func(m *SavedActorMotion) { m.StaticRoute = append(m.StaticRoute, 250) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, m, cs, bs := motionFixture1115(t, 224, 128, 32, 0, 2, 3, 8)
			tc.apply(&m)
			importMotion1115(t, w, m, cs, bs)
			motions, _, _, present := w.SavedActorMotions()
			if !present || len(motions) != 1 {
				t.Fatalf("motion not carried: present=%v n=%d", present, len(motions))
			}
			if motions[0].Active || motions[0].Issue != wantIssue {
				t.Fatalf("out-of-bounds route cell: active=%v issue=%q", motions[0].Active, motions[0].Issue)
			}
			if _, err := w.MarshalBinary(); err != nil {
				t.Fatalf("unsupported route state lost native saveability: %v", err)
			}
			// An inactive, issue-marked motion must not be silently advanced.
			Step(w, nil)
			after, _, _, _ := w.SavedActorMotions()
			if after[0].Active {
				t.Fatal("Step activated a motion admission had refused")
			}
		})
	}
}

// TestSavedMotion1134InBoundsRouteCellsAtTheMapEdgeAdmit proves the bounds
// check is inclusive at 0 and Width-1/Height-1, not merely "small values
// pass." Each edge cell is its own single-element list: the within-list
// adjacency and domain-crossing checks (motionAdmissionIssue) never compare
// across DynamicRoute/StaticRoute or against the unit's own position, only
// consecutive elements of the same list, so a lone element never trips
// either regardless of distance — what proves that is Issue staying empty
// rather than becoming an "outside the native map" refusal.
//
// motionFixture1115's unit sits at (15,16), so both (0,0) and (31,31) sit
// far outside DIV-953's anchor radius: the dynamic cell that continuation
// would otherwise prefer (DIV-951) is exactly the kind of distant-but-in-
// bounds input that guard exists to withhold, so this is also its direct
// witness. TestSavedMotion1134ContinuationFromImportWalksToRouteGoal already
// covers an anchored route actually being walked; this test's own claim is
// narrower and stays true regardless: neither edge cell is bounds-refused.
func TestSavedMotion1134InBoundsRouteCellsAtTheMapEdgeAdmit(t *testing.T) {
	w, m, cs, bs := motionFixture1115(t, 128, 128, 0, 0, 0, 0, 0)
	m.StaticRoute = []uint16{0}
	m.DynamicRoute = []uint16{uint16(31)<<8 | 31}
	importMotion1115(t, w, m, cs, bs)
	motions, _, _, present := w.SavedActorMotions()
	if !present || motions[0].Issue != "" || !motions[0].Current || motions[0].Active {
		t.Fatalf("map-edge route cells should admit, not bounds-refuse: present=%v issue=%q current=%v active=%v",
			present, motions[0].Issue, motions[0].Current, motions[0].Active)
	}
	// The carried lists stay exactly what was imported: the corpus/release
	// audits and exportOriginalMoverRoutes compare and re-export these same
	// fields against the source file before any tick runs, and an anchor
	// refusal must not desync them from it either (pkg/sim/savedmotionroute.go).
	if len(motions[0].StaticRoute) != 1 || motions[0].StaticRoute[0] != 0 {
		t.Fatalf("carried static route should be untouched: %v", motions[0].StaticRoute)
	}
	if len(motions[0].DynamicRoute) != 1 || motions[0].DynamicRoute[0] != uint16(31)<<8|31 {
		t.Fatalf("carried dynamic route should be untouched: %v", motions[0].DynamicRoute)
	}
	at := indexOfEntity(w.entities, motions[0].Entity)
	e := w.entities[at]
	if e.HasTarget {
		t.Fatalf("an unanchored edge cell should not become a native target: x=%d y=%d", e.TargetX, e.TargetY)
	}
	if got := w.routes[at]; len(got) != 0 {
		t.Fatalf("an unanchored edge cell should not become a native route: %v", got)
	}
}

// TestSavedMotion1134DomainBlockedRouteCellIsKept proves a saved route naming a
// cell the unit's own domain cannot cross is admitted and handed to native
// movement like any other route. MOVE-ROUTE-004 states the original's own
// route extractor "does not re-test passability", so a blocked cell in a saved
// route is expected input, and the world that holds it must still Save.
func TestSavedMotion1134DomainBlockedRouteCellIsKept(t *testing.T) {
	w, m, cs, bs := motionFixture1115(t, 128, 128, 0, 0, 0, 0, 0)
	idx, ok := w.cellIndex(16, 16) // motionFixture1115's own StaticRoute[0] and DynamicRoute[0]
	if !ok {
		t.Fatal("fixture cell (16,16) is unexpectedly out of bounds")
	}
	w.grid[idx] = blockGround // entity 7 names no domain, so it is DomainGround
	importMotion1115(t, w, m, cs, bs)
	motions, _, _, present := w.SavedActorMotions()
	if !present || motions[0].Issue != "native movement continues the imported route" {
		t.Fatalf("domain-blocked route cell: present=%v issue=%q", present, motions[0].Issue)
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("a domain-blocked saved route must still let the world Save: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("the saved world did not load: %v", err)
	}
	at := indexOfEntity(w.entities, motions[0].Entity)
	if len(w.routes[at]) == 0 || len(back.routes[at]) != len(w.routes[at]) {
		t.Fatalf("the route crossing the closed cell was not kept and saved: %v, loaded %v", w.routes[at], back.routes[at])
	}
}
