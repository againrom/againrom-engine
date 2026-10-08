package sim

import "testing"

// refusedPursuitWorld is the escort fixture: a 2x2 attacker whose only anchor
// row is walled off by five bodies, holding a victim it can see at the far end.
// owner is the attacker's owner; a human participant's attacker is given its
// victim by command.
func refusedPursuitWorld(t *testing.T, owner uint32) *World {
	t.Helper()
	return refusedPursuitWorldAt(t, owner, 20)
}

// refusedPursuitWorldAt is refusedPursuitWorld with the victim at column vx.
func refusedPursuitWorldAt(t *testing.T, owner uint32, vx int32) *World {
	t.Helper()
	const width, height = 24, 10
	b, grid := escortWallGrid(width, height)
	troll := engFighter(1, owner, 2, 4)
	troll.TokenSize = 2
	troll.ScanRange = 100
	ents := []Entity{troll, engFighter(2, 3, vx, 4)}
	for i, x := range []int32{8, 9, 10, 11, 12} {
		ents = append(ents, engFighter(EntityID(10+i), 5, x, 4))
	}
	w, err := NewRelatedWorld(1, b, ModeCanonical, Terrain{Block: grid}, ents, nil,
		engRel(t, [3]uint32{owner, 3, 1}, [3]uint32{3, owner, 1}))
	if err != nil {
		t.Fatalf("NewRelatedWorld: %v", err)
	}
	return w
}

func refusedAttacker(w *World) Entity { return w.entities[indexOfEntity(w.entities, 1)] }

// stepUntilIdle steps until the attacker's refused pursuit is idle.
func stepUntilIdle(t *testing.T, w *World, cmds []Command) {
	t.Helper()
	for k := 0; k < 200; k++ {
		Step(w, cmds)
		cmds = nil
		if refusedAttacker(w).PursuitIdle {
			return
		}
	}
	t.Fatal("the refused attacker never went idle")
}

func TestARefusedPursuitKeepsTheVictimAndGoesIdle(t *testing.T) {
	t.Parallel()
	w := refusedPursuitWorld(t, SelfSlot)
	stepUntilIdle(t, w, []Command{Attack(1, 2)})
	e := refusedAttacker(w)
	x, y := e.X, e.Y
	if !e.HasAttackTarget || e.AttackTarget != 2 || e.HasTarget || e.Stall != 0 || len(w.Route(1)) != 0 {
		t.Fatalf("idle attacker: victim %v/%d, walk %v, stall %d, route %d; want victim 2 kept and no walk",
			e.HasAttackTarget, e.AttackTarget, e.HasTarget, e.Stall, len(w.Route(1)))
	}
	// A human participant's order is not rewritten by any decision, the route
	// opening changes nothing, and the idle order neither walks nor strikes.
	w.entities[indexOfEntity(w.entities, 10)].HP = 0
	for k := 0; k < 160; k++ {
		Step(w, nil)
		e = refusedAttacker(w)
		if !e.PursuitIdle || !e.HasAttackTarget || e.AttackTarget != 2 || e.HasTarget || e.X != x || e.Y != y {
			t.Fatalf("tick %d: idle %v victim %v/%d walk %v at %d,%d; want the idle order unchanged at %d,%d",
				k, e.PursuitIdle, e.HasAttackTarget, e.AttackTarget, e.HasTarget, e.X, e.Y, x, y)
		}
	}
}

func TestAnIdleRefusedPursuitEndsOnTheNextOrder(t *testing.T) {
	t.Parallel()
	for name, cmd := range map[string]Command{
		"same victim": Attack(1, 2),
		"move":        MoveTo(1, CellPoint{X: 2, Y: 5}),
	} {
		w := refusedPursuitWorld(t, SelfSlot)
		stepUntilIdle(t, w, []Command{Attack(1, 2)})
		Step(w, []Command{cmd})
		if e := refusedAttacker(w); e.PursuitIdle {
			t.Errorf("%s: the order written after the refusal left the pursuit idle", name)
		}
	}
}

// The victim stands just past the escort, so the troll's static list falls
// to five nodes and its near search, aimed at the route end, finds no step.
func TestAnAIAttackerRefusedItsRouteIsReissuedByItsGroup(t *testing.T) {
	t.Parallel()
	w := refusedPursuitWorldAt(t, 2, 13)
	engRun(w, 1)
	stepUntilIdle(t, w, nil)
	e := refusedAttacker(w)
	if !e.HasAttackTarget || e.AttackTarget != 2 || e.HasTarget {
		t.Fatalf("idle AI attacker: victim %v/%d walk %v; want victim 2 kept, no walk", e.HasAttackTarget, e.AttackTarget, e.HasTarget)
	}
	for k := 0; k < scriptCycle+1; k++ {
		Step(w, nil)
		if !refusedAttacker(w).PursuitIdle {
			return
		}
	}
	t.Fatal("the group's next decision never reissued the idle attacker's order")
}

func TestAnIdleRefusedPursuitSurvivesTheByteForm(t *testing.T) {
	t.Parallel()
	w := refusedPursuitWorld(t, SelfSlot)
	stepUntilIdle(t, w, []Command{Attack(1, 2)})
	data, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back World
	if err := back.UnmarshalBinary(data); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if e := refusedAttacker(&back); !e.PursuitIdle || !e.HasAttackTarget || e.AttackTarget != 2 {
		t.Fatalf("restored attacker: idle %v victim %v/%d", e.PursuitIdle, e.HasAttackTarget, e.AttackTarget)
	}
}

// A friendly body parked in the only gap near the attacker stalls its near
// search past the stall limit, but a route round it exists, so the pursuit is
// not refused: it never idles, and once the body walks off the attacker closes
// on its victim and strikes.
func TestAnAttackerHeldUpByATrafficJamKeepsAttackingOnceItClears(t *testing.T) {
	t.Parallel()
	const width, height = 40, 50
	grid := make([]byte, width*height)
	for y := int32(0); y < height; y++ {
		if y == 4 || y == 10 {
			continue
		}
		grid[y*width+8] = blockGround
	}
	attacker := engFighter(1, SelfSlot, 2, 4)
	attacker.ScanRange = 100
	victim := engFighter(2, 3, 20, 4)
	blocker := engFighter(3, 5, 8, 4)
	w, err := NewRelatedWorld(1, Bounds{Width: width, Height: height}, ModeCanonical, Terrain{Block: grid},
		[]Entity{attacker, victim, blocker}, nil, engRel(t, [3]uint32{SelfSlot, 3, 1}, [3]uint32{3, SelfSlot, 1}))
	if err != nil {
		t.Fatalf("NewRelatedWorld: %v", err)
	}
	stalled := false
	for k := 0; k < 600; k++ {
		var cmds []Command
		switch k {
		case 0:
			cmds = []Command{Attack(1, 2)}
		case 60:
			cmds = []Command{MoveTo(3, CellPoint{X: 3, Y: 8})}
		}
		Step(w, cmds)
		e := w.entities[indexOfEntity(w.entities, 1)]
		if e.PursuitIdle {
			t.Fatalf("tick %d: the attacker idled although a route round the jam exists", k)
		}
		if k < 60 && e.Stall >= stallLimit-1 {
			stalled = true
		}
	}
	if !stalled {
		t.Fatal("the attacker never stalled against the parked body, so the jam did not bind")
	}
	if v := w.entities[indexOfEntity(w.entities, 2)]; v.HP >= 20 {
		t.Fatalf("the victim holds %d health: the attacker never reached it after the jam cleared", v.HP)
	}
}
