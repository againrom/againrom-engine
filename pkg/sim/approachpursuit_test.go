package sim

import (
	"bytes"
	"encoding/json"
	"testing"
)

func pursuitEscapeWorld(t *testing.T) *World {
	t.Helper()
	a := laFighter(1, 2, 7, 20, 20)
	a.Reach, a.Facing = 1, 192
	v := laFighter(2, SelfSlot, 9, 21, 20)
	v.Facing, v.Speed = 64, 1000
	return engWorld(t, engRel(t, [3]uint32{2, SelfSlot, relationHostile}, [3]uint32{SelfSlot, 2, 2}), a, v)
}

func TestAcquisitionAndEngagementWritersKeepDistinctPursuits(t *testing.T) {
	for _, tc := range []struct {
		name    string
		acquire bool
		write   func(*World)
	}{
		{"standing", true, func(w *World) { w.acquireStanding(0) }},
		{"refused route", true, func(w *World) { w.reacquireWithinReach(0) }},
		{"move arrival", true, func(w *World) {
			w.commandGroup([]int{0}, orderMove, cell{x: 20, y: 20})
			w.decide(aiGroup{owner: 2, group: effectiveGroup(w.entities[0]), members: []int{0}})
		}},
		{"player attack", false, func(w *World) { Step(w, []Command{Attack(1, 2)}) }},
		{"script attack", false, func(w *World) { w.cmdGroupAttack(ScriptInstant{Group: 7, HasGroup: true, Unit: 2, HasUnit: true}) }},
		{"group guard", false, func(w *World) {
			w.commandGroup([]int{0}, orderGuard, cell{})
			w.decide(aiGroup{owner: 2, group: effectiveGroup(w.entities[0]), members: []int{0}})
		}},
		{"swarm", false, func(w *World) { w.armSwarm(aiGroup{owner: 2, group: 7, members: []int{0}}, []int{1}) }},
		{"guard post", false, func(w *World) { w.postEngage(0, cell{x: 20, y: 20}) }},
		{"defend cover", false, func(w *World) { w.coverEngage(0, 0) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := pursuitEscapeWorld(t)
			w.acquireStanding(0)
			tc.write(w)
			a := w.entities[0]
			if !a.HasAttackTarget || a.AttackTarget != 2 || a.AcquirePursuit != tc.acquire {
				t.Fatalf("writer left victim %d present %t acquisition %t, want true/%t", a.AttackTarget, a.HasAttackTarget, a.AcquirePursuit, tc.acquire)
			}
		})
	}
	w := pursuitEscapeWorld(t)
	w.acquireStanding(0)
	if w.orderAcquire(0, 999) || !w.entities[0].AcquirePursuit {
		t.Fatal("refused order changed the pursuit")
	}
	w.entities[0].clearAttack()
	if w.entities[0].AcquirePursuit {
		t.Fatal("cleared order retained acquisition")
	}
	saved := pursuitEscapeWorld(t)
	saved.acquireStanding(0)
	order := SavedActorOrder{Entity: 1, Authored: true}
	saved.savedEngage(0, &order)
	if saved.entities[0].AcquirePursuit {
		t.Fatal("saved engagement changed arm")
	}
	imported := pursuitEscapeWorld(t)
	imported.acquireStanding(0)
	if err := imported.ImportOriginalActorActions([]OriginalActorAction{{Entity: 1, HasTarget: true, Target: 2}}); err != nil {
		t.Fatal(err)
	}
	if imported.entities[0].AcquirePursuit {
		t.Fatal("original action import inherited acquisition")
	}
}

func TestAcquisitionPursuitWorldAndActionContinuation(t *testing.T) {
	w := pursuitEscapeWorld(t)
	w.orderAttack(0, 2)
	old := mustMarshal(t, w)
	w.acquireStanding(0)
	marked := mustMarshal(t, w)
	const tag = 34 + 3*48*48 + 48
	if old[tag] != 1 || marked[tag] != 3 {
		t.Fatalf("attack tags %d/%d, want 1/3", old[tag], marked[tag])
	}
	stripped := append([]byte(nil), marked...)
	stripped[tag] = 1
	if !bytes.Equal(old, stripped) {
		t.Fatal("unmarked World byte form changed")
	}
	var cold, lost World
	if err := cold.UnmarshalBinary(marked); err != nil {
		t.Fatal(err)
	}
	if err := lost.UnmarshalBinary(stripped); err != nil {
		t.Fatal(err)
	}
	if !cold.entities[0].AcquirePursuit || lost.entities[0].AcquirePursuit || cold.Hash() == lost.Hash() {
		t.Fatal("World loss control did not distinguish the arms")
	}
	a := actionCopy(t, w.Actions())
	cold.entities[0].AcquirePursuit = false
	if err := cold.RestoreActions(a, nil); err != nil {
		t.Fatal(err)
	}
	if !cold.entities[0].AcquirePursuit || cold.Hash() != w.Hash() {
		t.Fatal("action supplement lost acquisition")
	}
	raw, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	raw = bytes.ReplaceAll(raw, []byte(`,"AcquirePursuit":true`), nil)
	var legacy ActionContinuations
	if err := json.Unmarshal(raw, &legacy); err != nil {
		t.Fatal(err)
	}
	if err := lost.RestoreActions(legacy, nil); err != nil {
		t.Fatal(err)
	}
	if lost.entities[0].AcquirePursuit {
		t.Fatal("old supplement did not default to engagement")
	}
	Step(w, []Command{MoveTo(2, CellPoint{X: 28, Y: 20})})
	Step(&cold, []Command{MoveTo(2, CellPoint{X: 28, Y: 20})})
	Step(&lost, []Command{MoveTo(2, CellPoint{X: 28, Y: 20})})
	for range 4 {
		Step(w, nil)
		Step(&cold, nil)
		Step(&lost, nil)
		if w.Hash() != cold.Hash() {
			t.Fatal("restored acquisition diverged on following ticks")
		}
	}
	if w.Hash() == lost.Hash() {
		t.Fatal("marker loss had no behavioral effect")
	}
}

func TestAcquisitionPursuitRejectsInvalidVictims(t *testing.T) {
	w := pursuitEscapeWorld(t)
	w.acquireStanding(0)
	before := w.Hash()
	for _, kind := range []string{"absent", "structure", "self", "missing"} {
		a := actionCopy(t, w.Actions())
		switch kind {
		case "absent":
			a.Actors[0].HasAttackTarget = false
		case "structure":
			a.Actors[0].AttackTargetKind = AttackTargetStructure
		case "self":
			a.Actors[0].AttackTarget = 1
		case "missing":
			a.Actors[0].AttackTarget = 999
		}
		if err := w.RestoreActions(a, nil); err == nil || w.Hash() != before {
			t.Fatalf("invalid %s pursuit accepted or partially applied", kind)
		}
	}
	e := w.entities[0]
	e.AttackTargetKind = AttackTargetStructure
	if err := attackFault(e); err == nil {
		t.Fatal("structure acquisition accepted")
	}
	e.HasAttackTarget = false
	if err := attackFault(e); err == nil {
		t.Fatal("absent acquisition accepted")
	}
	form := mustMarshal(t, w)
	const target = 34 + 3*48*48 + 44
	for i := range 4 {
		form[target+i] = 0xff
	}
	var cold World
	if err := cold.UnmarshalBinary(form); err == nil {
		t.Fatal("tag 3 with absent unit victim accepted")
	}
}

func TestAcquiredVictimBeyondReachTurnsWithoutAPath(t *testing.T) {
	for _, acquired := range []bool{true, false} {
		name := "player attack"
		if acquired {
			name = "standing acquisition"
		}
		t.Run(name, func(t *testing.T) {
			w := pursuitEscapeWorld(t)
			move := MoveTo(2, CellPoint{X: 28, Y: 20})
			if acquired {
				w.acquireStanding(0)
				Step(w, []Command{move})
			} else {
				Step(w, []Command{Attack(1, 2), move})
			}
			if !w.entities[0].HasAttackTarget {
				t.Fatal("production writer took no victim")
			}
			observed := false
			for n := 0; n < 5; n++ {
				beforeA, beforeV := w.entities[0], w.entities[1]
				Step(w, nil)
				a, v := w.entities[0], w.entities[1]
				if inReach(beforeA, beforeV) || beforeA.AttackPhase != AttackReady || !beforeA.HasAttackTarget {
					continue
				}
				observed = true
				if acquired {
					if a.HasTarget || a.X != 20 || a.Y != 20 {
						t.Fatalf("acquisition chased victim at (%d,%d): actor (%d,%d), destination %t", v.X, v.Y, a.X, a.Y, a.HasTarget)
					}
					if a.Facing != 64 && a.DesiredFacing != 64 {
						t.Fatalf("acquisition facing %d desired %d, want east", a.Facing, a.DesiredFacing)
					}
				} else if !a.HasTarget {
					t.Fatal("player attack did not path beyond reach")
				}
			}
			if !observed {
				t.Fatal("ordinary move never exposed a ready pursuit beyond reach")
			}
		})
	}
}
