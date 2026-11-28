package sim

import "testing"

func TestRouteSearchFootprintAndStepAdmission(t *testing.T) {
	walkStepGames(t, func(t *testing.T, second bool) {
		mover := walkStepUnit(1, 5, 5, 0)
		mover.TokenSize = 2
		w := walkStepWorld(t, second, mover, walkStepUnit(2, 7, 6, 0))
		s := newRouteScratch(w)
		route, ok := w.searchRoute(s, 0, unitRelation, dynamicWindow, scaledBudget, exactGoal, 6, 5)
		if ok == second || !second && (len(route) != 1 || route[0] != (cell{6, 5})) {
			t.Fatalf("near route %v, ok %t, second game %t", route, ok, second)
		}
		if w.advanceStep(s, 0, cell{6, 5}, false) != stepBlocked || w.entities[0].X != 5 {
			t.Fatal("wide mover entered a held footprint corner")
		}
	})
}

func TestNearRouteReadsHeldPlaneWithoutAnOccupant(t *testing.T) {
	w := walkStepWorld(t, false, walkStepUnit(1, 5, 5, 0))
	p := w.CellPlanes(CellPlaneStart{})
	key := uint16(5<<8 | 6)
	p.Dynamic[key] = 0x60
	w.savedCellPlanes = p
	s := newRouteScratch(w)
	if !w.open(s, terrainRelation, 0, 6, 5) || w.open(s, unitRelation, 0, 6, 5) {
		t.Fatal("a ground reservation must block only the local search")
	}
	p.Dynamic[key] = 0x80
	if !w.open(s, unitRelation, 0, 6, 5) {
		t.Fatal("an air reservation blocked a ground mover")
	}
}

func TestRouteRepathChoosesConfiguredListWord(t *testing.T) {
	walkStepGames(t, func(t *testing.T, second bool) {
		w := walkStepWorld(t, second, walkStepUnit(1, 3, 5, 0))
		path := ShippedPathFinding()
		path.DynamicByStaticLookup = 1
		w.SetBuildRules(BuildRules{Path: &path})
		route := []cell{{4, 5}, {5, 6}, {6, 7}, {6, 8}, {7, 9}, {8, 10}}
		w.routes[0] = route
		e := &w.entities[0]
		e.HasTarget, e.TargetX, e.TargetY = true, 8, 10
		if got, ok := w.subGoal(newRouteScratch(w), 0); !ok || got != route[1] {
			t.Fatalf("lookup 1 selected %v, want %v", got, route[1])
		}
	})
}

func TestRouteTraceStopsBeyondAThousandSteps(t *testing.T) {
	walkStepGames(t, func(t *testing.T, second bool) {
		w := mustWorld(t, 1, Bounds{Width: 1002, Height: 1}, nil)
		if second {
			w.game = newGameProfile(ScriptROM2)
		}
		s := newRouteScratch(w)
		for x := range s.plane {
			s.plane[x] = uint64(1 + 2*x)
		}
		for _, n := range []int{1000, 1001} {
			route, ok := w.walkBack(s, DomainAir, window{half: noWindow}, cell{}, cell{x: int32(n)})
			if n == 1000 && (!ok || len(route) != n) || n == 1001 && (ok || len(route) != 0) {
				t.Fatalf("trace of %d steps returned %d, ok %t", n, len(route), ok)
			}
		}
	})
}

func TestStandingAfterZeroVectorSurvivesReload(t *testing.T) {
	w := gameProfileCopy(t, strideLawWorld(t, ScriptROM1, 1), true)
	if _, advances, _, ok := w.StepRate(7, 9, 9); !ok || advances != 1 {
		t.Errorf("zero-vector step takes %d advances before standing", advances)
	}
	Step(w, []Command{MoveTo(7, CellPoint{X: 9, Y: 9})})
	e := w.entities[0]
	if e.Transit != 0 || !e.HasTarget || !e.Walk.Listed || e.X != 8 || e.Y != 8 {
		t.Fatalf("first advance: at (%d,%d), transit %d, listed %t, target %t", e.X, e.Y, e.Transit, e.Walk.Listed, e.HasTarget)
	}
	data, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	if err := cold.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick < 260; tick++ {
		Step(w, nil)
		Step(&cold, nil)
		e = w.entities[0]
		if e.X != 8 || e.Y != 8 || e.Transit != 0 || cold.Hash() != w.Hash() {
			t.Fatalf("tick %d: at (%d,%d), transit %d, replay equal %t", tick, e.X, e.Y, e.Transit, cold.Hash() == w.Hash())
		}
	}
	Step(w, []Command{MoveTo(7, CellPoint{X: 9, Y: 8})})
	if e := w.entities[0]; !e.stepping() || e.Stride.StepX != 1 || e.Stride.StepY != 0 {
		t.Fatal("the standing mover could not start a straight step")
	}
}

func TestLoadedStepReadsBothFractionBytes(t *testing.T) {
	for _, y := range []byte{127, 128} {
		w, m, cs, bs := savedMotionFixture(t, 120, y, 8, 0, 2, 0, 10)
		w.game, w.rom2 = newGameProfile(ScriptROM2), &rom2ScriptState{}
		importSavedMotion(t, w, m, cs, bs)
		Step(w, nil)
		motion := w.motionFor(7)
		if motion == nil || motion.Active != (y != 128) {
			t.Fatalf("fine (128,%d) after advance: active %v", y, motion != nil && motion.Active)
		}
	}
}

func TestPursuitRefreshCountsCentredCalls(t *testing.T) {
	w := walkStepWorld(t, false, puFighter(1, 0, 10), cbEnt(2, 14, 10), cbEnt(3, 1, 11))
	Step(w, []Command{Attack(1, 2)})
	e := &w.entities[0]
	e.Walk = WalkRefresh{Age: 31, Listed: true, NextX: e.X, NextY: e.Y + 1}
	passes := e.Pursuit.Passes
	s := newRouteScratch(w)
	w.pursue(s, 0, 1, false)
	e = &w.entities[0]
	if e.Walk.Age != 32 || e.Pursuit.Passes != passes || e.Y != 10 {
		t.Fatalf("call 32: age %d, passes %d (was %d), cell %d,%d", e.Walk.Age, e.Pursuit.Passes, passes, e.X, e.Y)
	}
	w.pursue(s, 0, 1, false)
	if e.Walk.Age != 0 || e.Pursuit.Passes != passes+1 {
		t.Fatalf("call 33: age %d, passes %d", e.Walk.Age, e.Pursuit.Passes)
	}
	e.Pursuit.EndX, e.Pursuit.EndY = e.X, e.Y
	e.Walk.Age = 7
	w.pursue(s, 0, 1, false)
	if e.Pursuit.Held || e.Walk.Age != 8 {
		t.Fatalf("endpoint match: held %t, age %d", e.Pursuit.Held, e.Walk.Age)
	}
}

func TestNearSearchUnlinksTheHeadBeforeTheStep(t *testing.T) {
	walkStepGames(t, func(t *testing.T, second bool) {
		e := walkStepUnit(1, 5, 5, 16)
		e.Facing, e.RotationSpeed = 0, 8
		w := walkStepWorld(t, second, e)
		Step(w, []Command{MoveTo(1, CellPoint{X: 15, Y: 5})})
		head, listed := w.walkNext(0)
		route := w.Route(1)
		if !listed || head != (cell{6, 5}) || w.entities[0].X != 5 || len(route) != 9 || route[0] != [2]int32{7, 5} {
			t.Fatalf("turning at %d: listed %v, %t, static %v", w.entities[0].X, head, listed, route)
		}
	})
}

func TestSecondGameEmptyDestinationRepathStopsComputerWalk(t *testing.T) {
	w := secondProfileOf(jammedCorridor(t, 1, 25))
	e := &w.entities[0]
	e.X, e.Y = 7, 4
	e.HP--
	w.walkTo(newRouteScratch(w), 0, 8, 4)
	Step(w, nil)
	if e.HasTarget || e.X != 7 || e.Y != 4 {
		t.Fatalf("empty destination replan left target %t at %d,%d", e.HasTarget, e.X, e.Y)
	}
}

func TestSecondGameZeroCostCentreCanWinTheRouteTie(t *testing.T) {
	b := Bounds{Width: 2, Height: 1}
	w := secondProfileOf(csWorld(t, b, ModeCanonical, openGrid(b), []byte{8, 0}, DomainGround, 0, 0))
	if route, ok := w.searchRoute(newRouteScratch(w), 0, terrainRelation, noWindow, flatBudget, exactGoal, 1, 0); ok || len(route) != 0 {
		t.Fatalf("self predecessor at a zero-cost goal returned %v", route)
	}
}
