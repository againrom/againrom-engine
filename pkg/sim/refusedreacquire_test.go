package sim

import "testing"

func reacquireWall(t *testing.T, owner uint32, reach uint8, victimX int32, bodyXs []int32, bodyHostile bool) *World {
	t.Helper()
	const width, height = 30, 10
	b, grid := escortWallGrid(width, height)
	troll := engFighter(1, owner, 2, 4)
	troll.TokenSize = 2
	troll.ScanRange = 100
	troll.Reach = reach
	ents := []Entity{troll, engFighter(2, 3, victimX, 4)}
	for i, x := range bodyXs {
		body := engFighter(EntityID(10+i), 5, x, 4)
		body.ScanRange = 0
		ents = append(ents, body)
	}
	cells := [][3]uint32{{owner, 3, 1}, {3, owner, 1}}
	if bodyHostile {
		cells = append(cells, [3]uint32{owner, 5, 1}, [3]uint32{5, owner, 1})
	}
	w, err := NewRelatedWorld(1, b, ModeCanonical, Terrain{Block: grid}, ents, nil, engRel(t, cells...))
	if err != nil {
		t.Fatalf("NewRelatedWorld: %v", err)
	}
	return w
}

func reacquirer(w *World) Entity { return w.entities[indexOfEntity(w.entities, 1)] }

func stepsUntilRefused(t *testing.T, w *World, cmds []Command) (int, Entity) {
	t.Helper()
	still, lastX, lastY := 0, reacquirer(w).X, reacquirer(w).Y
	for k := 0; k < 400; k++ {
		Step(w, cmds)
		cmds = nil
		e := reacquirer(w)
		if e.PursuitIdle || e.AcquirePursuit {
			return still, e
		}
		if e.X == lastX && e.Y == lastY && e.Transit == 0 {
			still++
		} else {
			still = 0
		}
		lastX, lastY = e.X, e.Y
	}
	t.Fatal("the attacker's pursuit was never refused")
	return 0, Entity{}
}

func TestARefusedPursuitTakesTheNearestHostileWithinReach(t *testing.T) {
	t.Parallel()
	w := reacquireWall(t, SelfSlot, 3, 20, []int32{8, 9, 10, 11, 12}, true)
	_, e := stepsUntilRefused(t, w, []Command{Attack(1, 2)})
	if !e.AcquirePursuit || e.PursuitIdle || !e.HasAttackTarget || e.AttackTarget != 10 || e.HasTarget || e.Stall != 0 {
		t.Fatalf("acquisition %v idle %v victim %v/%d walk %v stall %d; want the nearest body 10 taken",
			e.AcquirePursuit, e.PursuitIdle, e.HasAttackTarget, e.AttackTarget, e.HasTarget, e.Stall)
	}
	before := w.entities[indexOfEntity(w.entities, 10)].HP
	for k := 0; k < 120; k++ {
		Step(w, nil)
	}
	if after := w.entities[indexOfEntity(w.entities, 10)].HP; after >= before {
		t.Errorf("the nearest body holds %d health, from %d: the attacker never struck its pick", after, before)
	}
}

func TestARefusedPursuitWithNoHostileWithinReachKeepsItsVictimIdle(t *testing.T) {
	t.Parallel()
	for name, w := range map[string]*World{
		"bodies not hostile": reacquireWall(t, SelfSlot, 3, 20, []int32{8, 9, 10, 11, 12}, false),
		"hostile past reach": reacquireWall(t, SelfSlot, 1, 20, []int32{8, 9, 10, 11, 12}, true),
	} {
		_, e := stepsUntilRefused(t, w, []Command{Attack(1, 2)})
		if !e.PursuitIdle || e.AcquirePursuit || !e.HasAttackTarget || e.AttackTarget != 2 {
			t.Errorf("%s: idle %v acquisition %v victim %v/%d; want the idle order on victim 2",
				name, e.PursuitIdle, e.AcquirePursuit, e.HasAttackTarget, e.AttackTarget)
		}
	}
}

func gapWorld(t *testing.T, owner uint32) *World {
	t.Helper()
	const width, height = 30, 10
	b, grid := escortWallGrid(width, height)
	attacker := engFighter(1, owner, 2, 4)
	attacker.Reach = 1
	attacker.ScanRange = 100
	ents := []Entity{attacker, engFighter(2, 3, 13, 4)}
	for i, y := range []int32{4, 5} {
		body := engFighter(EntityID(10+i), 5, 11, y)
		body.ScanRange = 0
		ents = append(ents, body)
	}
	cells := [][3]uint32{{owner, 3, 1}, {3, owner, 1}, {owner, 5, 1}, {5, owner, 1}}
	w, err := NewRelatedWorld(1, b, ModeCanonical, Terrain{Block: grid}, ents, nil, engRel(t, cells...))
	if err != nil {
		t.Fatalf("NewRelatedWorld: %v", err)
	}
	return w
}

func TestAHumanRefusedPursuitWaitsForTheStallCountAtTheSearchThatAimsAtTheVictim(t *testing.T) {
	t.Parallel()
	still, e := stepsUntilRefused(t, gapWorld(t, SelfSlot), []Command{Attack(1, 2)})
	if !e.AcquirePursuit || e.AttackTarget != 10 {
		t.Fatalf("acquisition %v victim %d; want the nearest body 10 taken", e.AcquirePursuit, e.AttackTarget)
	}
	if still < stallLimit-2 {
		t.Fatalf("the attacker stood %d ticks before the refusal, want about the stall limit %d", still, stallLimit)
	}
}

func TestAnAIAttackerBlockedByAnAllyInACorridorIsRefusedAtTheSearchThatAimsAtTheVictim(t *testing.T) {
	t.Parallel()
	w := corridorAllyWorld(t, 2)
	engRun(w, 1)
	still, e := stepsUntilRefused(t, w, nil)
	if !e.PursuitIdle || e.AttackTarget != 2 {
		t.Fatalf("idle %v victim %d; want the order idle with victim 2 kept", e.PursuitIdle, e.AttackTarget)
	}
	if still >= stallLimit/2 {
		t.Fatalf("the attacker stood %d ticks before the refusal, want it raised by the search itself", still)
	}
}

func corridorAllyWorld(t *testing.T, owner uint32) *World {
	t.Helper()
	const width, height = 30, 10
	b := Bounds{Width: width, Height: height}
	grid := make([]byte, width*height)
	for y := int32(0); y < height; y++ {
		for x := int32(0); x < width; x++ {
			if y != 4 && !(y == 5 && x == 11) {
				grid[y*width+x] = blockGround
			}
		}
	}
	attacker := engFighter(1, owner, 2, 4)
	attacker.Reach = 1
	attacker.ScanRange = 100
	ally := engFighter(3, owner, 10, 4)
	ally.ScanRange = 0
	ents := []Entity{attacker, engFighter(2, 3, 12, 4), ally}
	w, err := NewRelatedWorld(1, b, ModeCanonical, Terrain{Block: grid}, ents, nil, engRel(t, [][3]uint32{{owner, 3, 1}, {3, owner, 1}}...))
	if err != nil {
		t.Fatalf("NewRelatedWorld: %v", err)
	}
	return w
}

func TestAHumanAttackOrderSurvivesAnAllyPassingInAOneCellCorridor(t *testing.T) {
	t.Parallel()
	w := corridorAllyWorld(t, SelfSlot)
	for k := 0; k < 300; k++ {
		var cmds []Command
		switch k {
		case 0:
			cmds = []Command{Attack(1, 2)}
		case 12:
			cmds = []Command{MoveTo(3, CellPoint{X: 11, Y: 5})}
		}
		Step(w, cmds)
	}
	if v := w.entities[indexOfEntity(w.entities, 2)]; v.HP >= 20 {
		a := reacquirer(w)
		t.Fatalf("the victim kept %d health; attacker at %d idle %v acquisition %v", v.HP, a.X, a.PursuitIdle, a.AcquirePursuit)
	}
}

func TestARefusedPursuitFarFromTheVictimWaitsForTheStallCount(t *testing.T) {
	t.Parallel()
	w := reacquireWall(t, SelfSlot, 3, 24, []int32{8, 9, 10, 11, 12}, true)
	still, e := stepsUntilRefused(t, w, []Command{Attack(1, 2)})
	if !e.AcquirePursuit {
		t.Fatalf("acquisition %v idle %v", e.AcquirePursuit, e.PursuitIdle)
	}
	if still < stallLimit-2 {
		t.Fatalf("the attacker stood %d ticks before the refusal, want about the stall limit %d", still, stallLimit)
	}
}

func islandWorld(t *testing.T, hostileBeside bool, reach uint8, mage bool) *World {
	t.Helper()
	const width, height = 30, 10
	grid := make([]byte, width*height)
	for y := int32(0); y < height; y++ {
		grid[y*width+14] = blockGround
	}
	attacker := engFighter(1, SelfSlot, 10, 4)
	attacker.Reach = reach
	attacker.ScanRange = 100
	if mage {
		attacker.MaxMana, attacker.Mana = 10, 10
	}
	victim := engFighter(2, 3, 20, 4)
	ents := []Entity{attacker, victim}
	cells := [][3]uint32{{SelfSlot, 3, 1}, {3, SelfSlot, 1}}
	if hostileBeside {
		beside := engFighter(11, 5, 11, 5)
		beside.ScanRange = 0
		ents = append(ents, beside)
		cells = append(cells, [3]uint32{SelfSlot, 5, 1}, [3]uint32{5, SelfSlot, 1})
	}
	w, err := NewRelatedWorld(1, Bounds{Width: width, Height: height}, ModeCanonical, Terrain{Block: grid}, ents, nil, engRel(t, cells...))
	if err != nil {
		t.Fatalf("NewRelatedWorld: %v", err)
	}
	return w
}

func TestAVictimNoTerrainRouteServesRefusesThePursuit(t *testing.T) {
	t.Parallel()
	_, e := stepsUntilRefused(t, islandWorld(t, true, 2, false), []Command{Attack(1, 2)})
	if !e.AcquirePursuit || e.AttackTarget != 11 {
		t.Fatalf("acquisition %v victim %d; want the hostile in reach, 11, taken", e.AcquirePursuit, e.AttackTarget)
	}
	_, e = stepsUntilRefused(t, islandWorld(t, false, 2, false), []Command{Attack(1, 2)})
	if !e.PursuitIdle || e.AttackTarget != 2 {
		t.Fatalf("idle %v victim %d; want the idle order on victim 2", e.PursuitIdle, e.AttackTarget)
	}
}

func TestARefusedPursuitOfAShortReachHumanMageTakesNoPick(t *testing.T) {
	t.Parallel()
	_, e := stepsUntilRefused(t, islandWorld(t, true, 1, false), []Command{Attack(1, 2)})
	if !e.AcquirePursuit || e.AttackTarget != 11 {
		t.Fatalf("fighter of reach 1: acquisition %v victim %d; want the adjacent hostile taken", e.AcquirePursuit, e.AttackTarget)
	}
	_, e = stepsUntilRefused(t, islandWorld(t, true, 1, true), []Command{Attack(1, 2)})
	if !e.PursuitIdle || e.AcquirePursuit || e.AttackTarget != 2 {
		t.Fatalf("mage of reach 1: idle %v acquisition %v victim %d; want the idle order on victim 2", e.PursuitIdle, e.AcquirePursuit, e.AttackTarget)
	}
}
