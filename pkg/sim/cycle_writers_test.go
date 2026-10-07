package sim

import "testing"

// More writers of a pending order against a loaded attack cycle. Each of these
// stores a state, a destination or a building and no attack progress, and the
// original's order machine consumes nonzero attack progress before any pending
// order, so a charge already loaded resolves its blow first (AI-ORDER-039,
// AI-RETREAT-272, HERO-CADENCE-112). The fixtures load the cycle the ordinary
// way, through an attack order and production ticks, and read the blow off the
// victim's health.

const lcGroup = uint32(5)

// lcBuild is the ordinary route to a loaded cycle for a unit of owner. The
// archer names group lcGroup so the script's group commands reach it. With
// saved set the world also carries a saved Group holding the archer under the
// given order, whose Swarm destination is (30, 20).
func lcBuild(t *testing.T, owner uint32, saved bool, order uint8, extra ...Entity) *World {
	t.Helper()
	rel := engRel(t, [3]uint32{owner, 2, relationHostile}, [3]uint32{2, owner, 2})
	archer := lcArcher(lcArcherID, owner, 20, 20)
	archer.Group = lcGroup
	ents := append([]Entity{archer, lcVictim(lcVictimID, 2, 24, 20)}, extra...)
	if saved && owner != SelfSlot {
		// A retained Group of the map decides only while a unit of the
		// participant stands near it.
		ents = append(ents, withdrawalFighter(9, SelfSlot, 18, 22, 100))
	}
	w := engWorld(t, rel, ents...)
	if saved {
		g := SavedGroup{ID: 71, Selector: lcGroup, Words: []uint16{97}, Path: []uint16{0x1111}}
		g.Members = []SavedGroupMember{{1, lcArcherID, true}}
		g.AI[0x20], g.AI[0x45], g.AI[0x38], g.AI[0x44] = order, 1, 9, 7
		g.AI[10], g.AI[11] = 30, 20
		orders := []SavedActorOrder{{Entity: lcArcherID}}
		orders[0].Raw[0], orders[0].Raw[1] = 20, 20
		orders[0].Raw[10], orders[0].Raw[11] = 20, 20
		if err := w.ImportSavedGroups([]SavedGroup{g}, orders); err != nil {
			t.Fatalf("ImportSavedGroups: %v", err)
		}
	}
	Step(w, []Command{Attack(lcArcherID, lcVictimID)})
	lcRequireLoaded(t, w, lcArcherID, lcVictimID)
	return w
}

// lcRequireRetained fails unless id still holds the cycle it had loaded on
// victim, whatever else the order it was given wrote.
func lcRequireRetained(t *testing.T, w *World, id, victim EntityID) {
	t.Helper()
	e := laEnt(t, w, id)
	if !e.HasAttackTarget || e.AttackTarget != victim || e.AttackPhase == AttackReady {
		t.Fatalf("the order discarded the loaded cycle: attack %t victim %d phase %d countdown %d",
			e.HasAttackTarget, e.AttackTarget, e.AttackPhase, e.AttackCountdown)
	}
}

// lcGroupOrder is the script's group command Args[0] applied to group.
func lcGroupOrder(group uint32, unit EntityID, args ...int32) ScriptInstant {
	in := ScriptInstant{Op: ScriptInstantGroupOrder, Group: group, HasGroup: true}
	if unit != 0 {
		in.Unit, in.HasUnit = unit, true
	}
	copy(in.Args[:], args)
	return in
}

// lcCompanion is a second unit of owner beside the archer.
func lcCompanion(owner uint32) Entity { return lcArcher(3, owner, 16, 20) }

// TestADefendOrderWaitsBehindALoadedAttackCycle covers the player's Defend
// command for the unit that acquires in place and for a unit that escorts a
// companion: the setter stores the state, the escort fields and the pending
// order and no progress (AI-CMD-054, AI-FOLLOWSET-116, AI-CMD-033).
func TestADefendOrderWaitsBehindALoadedAttackCycle(t *testing.T) {
	for _, tc := range []struct {
		name    string
		subject EntityID
		state   uint8
	}{
		{"acquires in place", lcArcherID, actorStateAcquire},
		{"escorts a companion", 3, actorStateDefend},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := lcBuild(t, SelfSlot, false, 0, lcCompanion(SelfSlot))
			Step(w, []Command{GroupDefend(lcArcherID, tc.subject, 1)})
			lcRequireRetained(t, w, lcArcherID, lcVictimID)
			if got := laEnt(t, w, lcArcherID).ActorState; got != tc.state {
				t.Fatalf("actor state = %d, want %d written by the order", got, tc.state)
			}
			lcBlowFirst(t, w, lcArcherID, lcVictimID)
		})
	}
}

// TestAPatrolOrderWaitsBehindALoadedAttackCycle covers the player's Patrol
// command, the script's Patrol on a native group and the player's Patrol in a
// world that carries saved Groups: the ring and the post are stored and no
// progress is (AI-PATROL-018, AI-PATROL-017, AI-CMD-033).
func TestAPatrolOrderWaitsBehindALoadedAttackCycle(t *testing.T) {
	dest := CellPoint{X: 17, Y: 20}
	for _, tc := range []struct {
		name  string
		build func(t *testing.T) *World
		order func(w *World)
	}{
		{"player patrol", func(t *testing.T) *World { return lcBuild(t, SelfSlot, false, 0) },
			func(w *World) { Step(w, []Command{GroupPatrolTo(lcArcherID, dest, 1)}) }},
		{"script patrol", func(t *testing.T) *World { return lcBuild(t, SelfSlot, false, 0) },
			func(w *World) { w.runInstant(lcGroupOrder(lcGroup, 0, subCommandPatrol, dest.X, dest.Y)) }},
		{"player patrol with saved groups", func(t *testing.T) *World { return lcBuild(t, SelfSlot, true, orderSwarm) },
			func(w *World) { Step(w, []Command{GroupPatrolTo(lcArcherID, dest, 1)}) }},
		{"script patrol with saved groups", func(t *testing.T) *World { return lcBuild(t, SelfSlot, true, orderSwarm) },
			func(w *World) { w.runInstant(lcGroupOrder(lcGroup, 0, subCommandPatrol, dest.X, dest.Y)) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := tc.build(t)
			tc.order(w)
			lcRequireRetained(t, w, lcArcherID, lcVictimID)
			if got := laEnt(t, w, lcArcherID).ActorState; got != actorStatePatrol {
				t.Fatalf("actor state = %d, want patrol %d written by the order", got, actorStatePatrol)
			}
			lcBlowFirst(t, w, lcArcherID, lcVictimID)
		})
	}
}

// TestAScriptGroupStopWaitsBehindALoadedAttackCycle covers the stop every
// script group command begins with: Defend and Follow hand the members an
// escort state, and Roam hands them to the group layer. The dispatcher calls
// the reset helper per member and stores no progress (TRIG-GRPARM-047,
// AI-GROUPCMD-020, AI-PATROL-018).
func TestAScriptGroupStopWaitsBehindALoadedAttackCycle(t *testing.T) {
	for _, tc := range []struct {
		name  string
		saved bool
		order ScriptInstant
		state uint8
	}{
		{"defend", false, lcGroupOrder(lcGroup, 3, subCommandDefend, 3), actorStateDefend},
		{"follow", false, lcGroupOrder(lcGroup, 3, subCommandFollow, 3), actorStateFollow},
		{"roam", false, lcGroupOrder(lcGroup, 0, int32(orderRoam)), actorStateEngage},
		{"defend with saved groups", true, lcGroupOrder(lcGroup, 3, subCommandDefend, 3), actorStateDefend},
		{"roam with saved groups", true, lcGroupOrder(lcGroup, 0, int32(orderRoam)), actorStateEngage},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := lcBuild(t, 3, tc.saved, orderSwarm, lcCompanion(3))
			w.runInstant(tc.order)
			lcRequireRetained(t, w, lcArcherID, lcVictimID)
			if got := laEnt(t, w, lcArcherID).ActorState; got != tc.state {
				t.Fatalf("actor state = %d, want %d", got, tc.state)
			}
			lcBlowFirst(t, w, lcArcherID, lcVictimID)
		})
	}
}

// TestASwarmWalkOnWaitsBehindALoadedAttackCycle covers the walk to the
// commanded cell a Swarm member takes when its group scores nothing, for a
// native group and for a saved Group. The victim stops being hostile while the
// cycle is loaded, so at the next decision the member scores nothing; the walk
// is an ordinary move order (AI-SWARM-022, AI-MOVE-023) and waits behind the
// cycle exactly as a move does.
func TestASwarmWalkOnWaitsBehindALoadedAttackCycle(t *testing.T) {
	for _, tc := range []struct {
		name  string
		saved bool
	}{
		{"native group", false},
		{"saved group", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := lcBuild(t, 3, tc.saved, orderSwarm)
			if !tc.saved {
				w.runInstant(lcGroupOrder(lcGroup, 0, int32(orderSwarm), 30, 20))
			}
			w.runInstant(relationInstant(3, 2, 2))
			cmdToDecision(w)
			Step(w, nil)
			lcHold(t, w, lcArcherID, lcVictimID, 20, 20, 30, 20)
			lcResolve(t, w, lcArcherID, lcVictimID)
		})
	}
}

// lcStructureWorld is lcBuild's archer beside a lever six cells away.
func lcStructureWorld(t *testing.T, victimX int32) *World {
	t.Helper()
	rel := engRel(t, [3]uint32{SelfSlot, 2, relationHostile}, [3]uint32{2, SelfSlot, 2})
	lever := Structure{ID: 0, Col: 14, Row: 20, MaxHealth: 1, Width: 1, Height: 1, Kind: 28, Attach: 1, Blocking: 1}
	w, err := NewStructuredWorld(1, engBounds, ModeCanonical, Terrain{},
		[]Entity{lcArcher(lcArcherID, SelfSlot, 20, 20), lcVictim(lcVictimID, 2, victimX, 20)},
		nil, rel, nil, nil, nil, GhostTemplate{}, []Structure{lever})
	if err != nil {
		t.Fatalf("NewStructuredWorld: %v", err)
	}
	Step(w, []Command{Attack(lcArcherID, lcVictimID)})
	return w
}

// TestAStructureUseWaitsBehindALoadedAttackCycle sends a loaded archer at a
// lever: the setter stores the state, the building and a destination and no
// progress, so the blow lands first, the approach starts once the cycle has
// ended, no second cycle loads before it, and the lever is used on arrival
// (AI-STRUCTUSE-306, AI-ORDER-039).
func TestAStructureUseWaitsBehindALoadedAttackCycle(t *testing.T) {
	w := lcStructureWorld(t, 24)
	lcRequireLoaded(t, w, lcArcherID, lcVictimID)
	Step(w, []Command{UseStructure(lcArcherID, 0)})
	lcRequireRetained(t, w, lcArcherID, lcVictimID)
	if len(w.StructureUses()) != 1 {
		t.Fatalf("structure uses = %v, want the one just ordered", w.StructureUses())
	}
	if e := laEnt(t, w, lcArcherID); e.X != 20 || e.Y != 20 {
		t.Fatalf("the archer walked to (%d,%d) with its cycle loaded", e.X, e.Y)
	}
	lcResolve(t, w, lcArcherID, lcVictimID)
	for i := 0; i < 400 && len(w.StructureUses()) != 0; i++ {
		Step(w, nil)
	}
	if len(w.StructureUses()) != 0 || w.structures[0].Field42 != 1 {
		t.Fatalf("lever not used after the walk: uses %v field %d", w.StructureUses(), w.structures[0].Field42)
	}
}

// TestAStructureUseHeldBehindALoadedCycleSurvivesTheByteFormAndResumesIdentically
// encodes the world the tick the order arrives: the held use is attack fields
// plus the structure use list, both already in the byte form, so a world
// decoded from it continues to the same hash.
func TestAStructureUseHeldBehindALoadedCycleSurvivesTheByteFormAndResumesIdentically(t *testing.T) {
	w := lcStructureWorld(t, 24)
	Step(w, []Command{UseStructure(lcArcherID, 0)})
	lcRequireRetained(t, w, lcArcherID, lcVictimID)
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var cold World
	if err := cold.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	for i := 0; i < 260; i++ {
		Step(w, nil)
		Step(&cold, nil)
		if w.Hash() != cold.Hash() {
			t.Fatalf("the decoded world diverged at step %d", i+1)
		}
	}
	if len(w.StructureUses()) != 0 {
		t.Fatal("the held use never finished")
	}
}

// TestTheseOrdersBetweenCyclesStillReplaceTheAttackAtOnce is the control: an
// archer still approaching its victim owes no cycle, so each of these orders
// takes the attack order away on the tick it is given.
func TestTheseOrdersBetweenCyclesStillReplaceTheAttackAtOnce(t *testing.T) {
	for _, tc := range []struct {
		name  string
		order func(w *World)
	}{
		{"defend", func(w *World) { Step(w, []Command{GroupDefend(lcArcherID, lcArcherID, 1)}) }},
		{"patrol", func(w *World) { Step(w, []Command{GroupPatrolTo(lcArcherID, CellPoint{X: 17, Y: 20}, 1)}) }},
		{"script patrol", func(w *World) {
			w.runInstant(lcGroupOrder(lcGroup, 0, subCommandPatrol, 17, 20))
		}},
		{"script defend", func(w *World) { w.runInstant(lcGroupOrder(lcGroup, 3, subCommandDefend, 3)) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rel := engRel(t, [3]uint32{SelfSlot, 2, relationHostile}, [3]uint32{2, SelfSlot, 2})
			archer := lcArcher(lcArcherID, SelfSlot, 20, 20)
			archer.Group = lcGroup
			w := engWorld(t, rel, archer, lcVictim(lcVictimID, 2, 32, 20), lcCompanion(SelfSlot))
			Step(w, []Command{Attack(lcArcherID, lcVictimID)})
			if e := laEnt(t, w, lcArcherID); !e.HasAttackTarget || e.AttackPhase != AttackReady {
				t.Fatalf("fixture: attack %t phase %d, want an archer still approaching", e.HasAttackTarget, e.AttackPhase)
			}
			tc.order(w)
			if got := laEnt(t, w, lcArcherID); got.HasAttackTarget || got.AttackPhase != AttackReady {
				t.Errorf("the order left attack %t phase %d, want the attack order gone", got.HasAttackTarget, got.AttackPhase)
			}
		})
	}
}

// TestAStructureUseBetweenCyclesStillReplacesTheAttackAtOnce is the control
// for the lever: the archer is still walking to a victim eight cells off.
func TestAStructureUseBetweenCyclesStillReplacesTheAttackAtOnce(t *testing.T) {
	w := lcStructureWorld(t, 32)
	if e := laEnt(t, w, lcArcherID); !e.HasAttackTarget || e.AttackPhase != AttackReady {
		t.Fatalf("fixture: attack %t phase %d, want an archer still approaching", e.HasAttackTarget, e.AttackPhase)
	}
	Step(w, []Command{UseStructure(lcArcherID, 0)})
	if got := laEnt(t, w, lcArcherID); got.HasAttackTarget {
		t.Errorf("the use left the attack on %d, want the attack order gone", got.AttackTarget)
	}
}

// lcLoadsAfter orders w with order after offset ticks and counts the cycles the
// archer loads in the 200 ticks that follow, and whether it held a cycle when
// the order landed.
func lcLoadsAfter(t *testing.T, w *World, offset int, order func(*World)) (loads int, loaded bool) {
	t.Helper()
	for i := 0; i < offset; i++ {
		Step(w, nil)
	}
	loaded = laEnt(t, w, lcArcherID).AttackPhase != AttackReady
	order(w)
	prev := laEnt(t, w, lcArcherID).AttackPhase
	for i := 0; i < 200; i++ {
		Step(w, nil)
		now := laEnt(t, w, lcArcherID).AttackPhase
		if prev == AttackReady && now != AttackReady {
			loads++
		}
		prev = now
	}
	return loads, loaded
}

// TestAStateOnlyOrderLoadsNoSecondCycleBehindAFinishingOne gives the escort
// states of Defend and Follow in every tick of a cycle. Each setter stores a
// state and pending order 0 and no progress, so a member whose cycle ends
// before the next decision pass has nothing that loads another one
// (AI-CMD-054, AI-FOLLOWSET-116, AI-GROUPCMD-020, AI-ORDER-039,
// HERO-CADENCE-112; DIV-1577). Without the retained wait the archer loads one
// more cycle on the victim when the order lands in the last ticks of its cycle.
func TestAStateOnlyOrderLoadsNoSecondCycleBehindAFinishingOne(t *testing.T) {
	for _, tc := range []struct {
		name  string
		build func(t *testing.T) *World
		order func(w *World)
	}{
		{"player defend", func(t *testing.T) *World { return lcBuild(t, SelfSlot, false, 0, lcCompanion(SelfSlot)) },
			func(w *World) { Step(w, []Command{GroupDefend(lcArcherID, 3, 1)}) }},
		{"script defend", func(t *testing.T) *World { return lcBuild(t, 3, false, orderSwarm, lcCompanion(3)) },
			func(w *World) { w.runInstant(lcGroupOrder(lcGroup, 3, subCommandDefend, 3)) }},
		{"script follow", func(t *testing.T) *World { return lcBuild(t, 3, false, orderSwarm, lcCompanion(3)) },
			func(w *World) { w.runInstant(lcGroupOrder(lcGroup, 3, subCommandFollow, 3)) }},
		{"script defend with saved groups", func(t *testing.T) *World { return lcBuild(t, 3, true, orderSwarm, lcCompanion(3)) },
			func(w *World) { w.runInstant(lcGroupOrder(lcGroup, 3, subCommandDefend, 3)) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			finishing := 0
			for offset := 0; offset < 90; offset++ {
				loads, loaded := lcLoadsAfter(t, tc.build(t), offset, tc.order)
				if loaded && offset >= 70 {
					finishing++
				}
				if loads != 0 {
					t.Fatalf("an order given %d ticks after the cycle loaded, with it %v, let the archer load %d more cycles", offset, loaded, loads)
				}
			}
			if finishing == 0 {
				t.Fatal("the sweep never gave the order while a cycle was ending")
			}
		})
	}
}

// TestAStateOnlyOrderHeldBehindALoadedCycleSurvivesTheByteFormAndResumesIdentically
// encodes the world the tick a Defend order lands in the last ticks of a cycle:
// the retained wait is the victim and a destination at the member's own cell,
// both already in the byte form, so a decoded world continues to the same hash
// through the end of the cycle and the walk to the companion.
func TestAStateOnlyOrderHeldBehindALoadedCycleSurvivesTheByteFormAndResumesIdentically(t *testing.T) {
	w := lcBuild(t, SelfSlot, false, 0, lcCompanion(SelfSlot))
	for i := 0; i < 72; i++ {
		Step(w, nil)
	}
	Step(w, []Command{GroupDefend(lcArcherID, 3, 1)})
	held := laEnt(t, w, lcArcherID)
	if !held.HasAttackTarget || !held.HasTarget || held.TargetX != held.X || held.TargetY != held.Y {
		t.Fatalf("fixture: the order left victim %t destination %t (%d,%d) at (%d,%d), want the victim and a wait at its own cell",
			held.HasAttackTarget, held.HasTarget, held.TargetX, held.TargetY, held.X, held.Y)
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal a world holding a state order behind a loaded cycle: %v", err)
	}
	var copyOf World
	if err := copyOf.UnmarshalBinary(form); err != nil {
		t.Fatalf("decode a world holding a state order behind a loaded cycle: %v", err)
	}
	if w.Hash() != copyOf.Hash() {
		t.Fatalf("decoded hash %#x differs from the live hash %#x", copyOf.Hash(), w.Hash())
	}
	for i := 0; i < 150; i++ {
		Step(w, nil)
		Step(&copyOf, nil)
		if w.Hash() != copyOf.Hash() {
			t.Fatalf("tick %d: decoded world diverged: live %#x decoded %#x", w.tick, w.Hash(), copyOf.Hash())
		}
	}
	if got := laEnt(t, w, lcArcherID); got.X == 20 && got.Y == 20 {
		t.Error("the archer never left its cell in the resumed span, so the comparison did not cross the walk")
	}
}
