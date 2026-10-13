package sim

import "testing"

const (
	owFighter = EntityID(1)
	owArcher  = EntityID(2)
	owMage    = EntityID(3)
)

// owWorld is a party of three on the participant's side, a melee fighter, an
// archer and a mage with Fire Arrow armed, beside hostiles at foes.
func owWorld(t *testing.T, foes ...[2]int32) *World {
	t.Helper()
	fighter := swarmFighter(owFighter, SelfSlot, 5, 10)
	fighter.AttackCharge, fighter.AttackRelax, fighter.Speed = 10, 10, 21
	archer := lcArcher(owArcher, SelfSlot, 5, 11)
	archer.HP, archer.MaxHP, archer.Speed = 5000, 5000, 21
	archer.AttackCharge, archer.AttackRelax = 12, 10
	mage := effectMage(owMage, 5, 9, 1<<1)
	mage.HP, mage.MaxHP, mage.Speed, mage.Capacity = 5000, 5000, 21, 1411
	mage.MaxMana, mage.Mana, mage.AttackCharge, mage.AttackRelax = 5000, 5000, 8, 12
	mage.ScanRange, mage.AutoSpell = engSight, 1
	ents := []Entity{fighter, archer, mage}
	for k, at := range foes {
		ents = append(ents, engFighter(EntityID(10+k), 2, at[0], at[1]))
	}
	rel := engRel(t, [3]uint32{SelfSlot, 2, relationHostile}, [3]uint32{2, SelfSlot, relationHostile})
	w, err := NewStructuredWorld(1, engBounds, ModeCanonical, Terrain{}, ents, nil, rel, nil, nil,
		[]SpellRule{hlArrow()}, GhostTemplate{}, nil)
	if err != nil {
		t.Fatalf("NewStructuredWorld: %v", err)
	}
	wakeGroups(w)
	return w
}

// owStrikes counts the strike releases of id on this tick: the cycle leaves its
// charge for its recovery. A release in a step fails the test.
type owStrikes struct {
	phase map[EntityID]AttackPhase
	n     int
}

func (s *owStrikes) observe(t *testing.T, w *World, tick int) {
	t.Helper()
	for _, e := range w.entities {
		before := s.phase[e.ID]
		if (before == AttackCharging || before == AttackCasting) && e.AttackPhase == AttackRelaxing {
			s.n++
			if e.Transit != 0 || w.motionActive(e.ID) {
				t.Fatalf("tick %d: entity %d released a strike in a step (transit %d)", tick, e.ID, e.Transit)
			}
		}
		s.phase[e.ID] = e.AttackPhase
	}
}

// Units under the player's Swarm walk to the commanded cell, stop at a cell to
// strike or cast at what they see, and walk on once the hostiles are dead. No
// strike and no cast is released in a step, and no cast charges on a walk.
func TestSwarmStopsStrikesCastsAndWalksOn(t *testing.T) {
	w := owWorld(t, [2]int32{15, 12}, [2]int32{19, 8})
	for k := range w.entities[3:] {
		w.entities[3+k].HP, w.entities[3+k].MaxHP = 120, 120
	}
	to := cell{x: 30, y: 10}
	var cmds []Command
	for _, id := range []EntityID{owFighter, owArcher, owMage} {
		cmds = append(cmds, GroupSwarmTo(id, CellPoint{X: to.x, Y: to.y}, 1))
	}
	watch := &osCastWatch{id: owMage}
	strikes := &owStrikes{phase: map[EntityID]AttackPhase{}}
	for tick := range 1500 {
		events := StepObserved(w, cmds)
		cmds = nil
		watch.observe(t, w, tick, events)
		strikes.observe(t, w, tick)
	}
	for _, e := range w.entities[3:] {
		if e.Alive() {
			t.Fatalf("hostile %d still lives with %d health", e.ID, e.HP)
		}
	}
	if watch.released == 0 || strikes.n == 0 {
		t.Fatalf("fixture: %d casts and %d strikes released", watch.released, strikes.n)
	}
	for _, e := range w.entities[:3] {
		if d := cellOf(&e).chebyshevTo(to); d > formationSpread {
			t.Errorf("member %d stands at (%d,%d), %d cells from the commanded cell", e.ID, e.X, e.Y, d)
		}
	}
}

// A melee fighter and an archer each pursue a hostile across the map; over
// every tick no release happens in a step.
func TestAttackersNeverReleaseInAStep(t *testing.T) {
	for _, id := range []EntityID{owFighter, owArcher} {
		w := owWorld(t, [2]int32{25, 14})
		w.entities[3].HP, w.entities[3].MaxHP = 400, 400
		w.entities[3].Speed = 21
		Step(w, []Command{Attack(id, 10)})
		strikes := &owStrikes{phase: map[EntityID]AttackPhase{}}
		for tick := range 800 {
			Step(w, nil)
			strikes.observe(t, w, tick)
		}
		if strikes.n == 0 {
			t.Fatalf("fixture: entity %d never struck", id)
		}
	}
}
