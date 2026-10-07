package sim

import (
	"bytes"
	"testing"
)

func secondGameActivityWorld(t *testing.T, controlledX, controlledY int32) *World {
	t.Helper()
	s := secondGameScript(t, nil, []ScriptInstant{{Op: 39, Group: 9, HasGroup: true}}, nil)
	enemy := engFighter(2, 2, 96, 96)
	enemy.Group = 9
	victim := engFighter(3, 3, 97, 96)
	w, err := NewRelatedWorld(1, Bounds{Width: 128, Height: 128}, ModeCanonical, Terrain{}, []Entity{{ID: 1, Owner: SelfSlot, X: controlledX, Y: controlledY, HP: 20, MaxHP: 20}, enemy, victim}, s, engRel(t, [3]uint32{2, 3, 1}))
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func TestSecondGameClearForcedActivityGatesOrdinaryCombat(t *testing.T) {
	w := secondGameActivityWorld(t, 1, 1)
	w.runInstant(ScriptInstant{Op: 39, Group: 9, HasGroup: true})
	scriptTicks(w, 40, nil)
	victim, _ := w.Entity(3)
	if victim.HP != 20 {
		t.Fatalf("inactive ordinary group damaged victim: HP%d", victim.HP)
	}
}

func TestSecondGameActivityKeepsForceProximityAndDamageBranches(t *testing.T) {
	for _, tc := range []struct {
		name          string
		x, y          int32
		clear, damage bool
	}{{"force", 1, 1, false, false}, {"coarse corner", 80, 80, true, false}, {"damaged", 1, 1, true, true}} {
		t.Run(tc.name, func(t *testing.T) {
			w := secondGameActivityWorld(t, tc.x, tc.y)
			if tc.clear {
				w.runInstant(ScriptInstant{Op: 39, Group: 9, HasGroup: true})
			}
			if tc.damage {
				w.entities[indexOfEntity(w.entities, 2)].HP--
			}
			scriptTicks(w, 40, nil)
			victim, _ := w.Entity(3)
			if victim.HP == 20 {
				t.Fatal("active group did not attack")
			}
		})
	}
}

func TestSecondGameInactiveUnitStillExecutesExplicitMove(t *testing.T) {
	w := secondGameActivityWorld(t, 1, 1)
	w.runInstant(ScriptInstant{Op: 39, Group: 9, HasGroup: true})
	before, _ := w.Entity(2)
	Step(w, []Command{MoveTo(2, CellPoint{X: 90, Y: 96})})
	scriptTicks(w, 40, nil)
	after, _ := w.Entity(2)
	if after.X == before.X || after.CommandGroup == 0 {
		t.Fatalf("explicit move lost to inactivity: %+v", after)
	}
}

func TestSecondGameActivityClearStopsAnExistingOrdinaryAttack(t *testing.T) {
	w := secondGameActivityWorld(t, 1, 1)
	scriptTicks(w, 7, nil)
	victim, _ := w.Entity(3)
	if victim.HP == 20 {
		t.Fatal("forced actor did not begin attack")
	}
	w.runInstant(ScriptInstant{Op: 39, Group: 9, HasGroup: true})
	before := victim.HP
	scriptTicks(w, 8, nil)
	victim, _ = w.Entity(3)
	if victim.HP != before {
		t.Fatalf("inactive existing ordinary attack continued: %d -> %d", before, victim.HP)
	}
}

func TestSecondGameActivityClearSurvivesColdWorldAndReportParity(t *testing.T) {
	w := secondGameActivityWorld(t, 1, 1)
	w.runInstant(ScriptInstant{Op: 39, Group: 9, HasGroup: true})
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	if err := cold.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 40; n++ {
		Step(w, nil)
		StepReported(&cold, nil)
		a, err := w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		b, err := cold.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(a, b) || w.Hash() != cold.Hash() {
			t.Fatalf("group activity differed after restore at%d", n)
		}
	}
	victim, _ := cold.Entity(3)
	if victim.HP != 20 {
		t.Fatal("cold inactive group attacked")
	}
}

func TestSecondGameActivityDoesNotChangeFirstGameGroups(t *testing.T) {
	w := secondGameActivityWorld(t, 1, 1)
	w.script = nil
	w.rom2 = nil
	scriptTicks(w, 40, nil)
	victim, _ := w.Entity(3)
	if victim.HP == 20 {
		t.Fatal("first-game guard inherited second-game activity gate")
	}
}
