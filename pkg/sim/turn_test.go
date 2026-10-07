package sim

// An attacker turns toward the unit it is hitting, and a facing gates nothing.

import (
	"reflect"
	"testing"
)

// turnBounds is roomy enough that nothing below reaches an edge.
var turnBounds = Bounds{Width: 11, Height: 11}

// armed is a unit that can hold a fight: alive, and with a cycle long enough
// that the cases below run inside one charge rather than through a blow.
func armed(id EntityID, x, y int32, facing uint8) Entity {
	return Entity{ID: id, X: x, Y: y, HP: 100, MaxHP: 100, Facing: facing,
		AttackCharge: 50, AttackRelax: 50}
}

// TestAnAttackerFacesAnAdjacentVictim is AC-4, over all eight relative
// positions: the victim stands on one of the eight cells around the attacker and
// the attacker points at it after a single advance.
//
// The attacker is built facing a direction the case cannot produce, so a case
// that turned nothing would fail rather than pass on the value it started with.
func TestAnAttackerFacesAnAdjacentVictim(t *testing.T) {
	for _, c := range theEight {
		// The opposite direction, which no correct turn here can reach.
		start := facingOfDir((c.dir + 4) % 8)
		w := mustWorldGrid(t, 1, turnBounds, ModeCanonical, openGrid(turnBounds), []Entity{
			armed(1, 5, 5, start),
			armed(2, 5+c.dx, 5+c.dy, 0),
		})
		Step(w, []Command{{Kind: KindAttack, Entity: 1, X: 2}})

		a := w.Entities()[0]
		if a.X != 5 || a.Y != 5 {
			t.Fatalf("%s: the attacker moved to (%d,%d); an adjacent victim is in reach", c.name, a.X, a.Y)
		}
		if got := FacingDir(a.Facing); got != c.dir {
			t.Errorf("%s: the attacker faces direction %d, want %d", c.name, got, c.dir)
		}
	}
}

// TestAnAttackerOutOfReachKeepsItsWalksFacing is AC-4's second clause. A victim
// far away is WALKED toward, and the facing is the walk's own — one step at a
// time — never a bearing over the whole distance.
//
// The victim sits well east and one cell south, so the bearing is south-east
// while every step of the walk but the first is due east. A build that faced the
// attacker at its victim from a distance would answer south-east on every tick;
// this asserts it does not.
func TestAnAttackerOutOfReachKeepsItsWalksFacing(t *testing.T) {
	w := mustWorldGrid(t, 1, turnBounds, ModeCanonical, openGrid(turnBounds), []Entity{
		armed(1, 1, 4, 0),
		armed(2, 9, 5, 0),
	})
	Step(w, []Command{{Kind: KindAttack, Entity: 1, X: 2}})

	// Walk it in, checking on every tick that the facing is the step just taken
	// and not the victim's bearing.
	prev := cell{1, 4}
	for tick := 0; tick < 20; tick++ {
		a := w.Entities()[0]
		if inReach(a, w.Entities()[1]) {
			if tick == 0 {
				t.Fatal("the attacker began in reach; this case needs a walk")
			}
			return
		}
		if a.X != prev.x || a.Y != prev.y {
			f, ok := facingToward(a.X-prev.x, a.Y-prev.y)
			if !ok {
				t.Fatalf("tick %d: a step with no direction", tick)
			}
			if a.Facing != f {
				t.Errorf("tick %d: the attacker stepped (%d,%d)->(%d,%d) and faces %d, want the step's %d",
					tick, prev.x, prev.y, a.X, a.Y, a.Facing, f)
			}
			prev = cell{a.X, a.Y}
		}
		Step(w, nil)
	}
	t.Fatal("the attacker never came into reach")
}

// TestAVictimOnTheAttackersOwnCellLeavesTheFacingAlone is AC-4's last clause.
// Two units may share a cell — a world holding such a pair is advanced, not
// repaired — and the delta between them names no direction.
func TestAVictimOnTheAttackersOwnCellLeavesTheFacingAlone(t *testing.T) {
	const mark = 0xa0
	w := mustWorldGrid(t, 1, turnBounds, ModeCanonical, openGrid(turnBounds), []Entity{
		armed(1, 5, 5, mark),
		armed(2, 5, 5, 0),
	})
	Step(w, []Command{{Kind: KindAttack, Entity: 1, X: 2}})
	if got := w.Entities()[0].Facing; got != mark {
		t.Errorf("the attacker faces %d, want the %d it held: a victim on its own cell names no direction",
			got, mark)
	}
}

// TestHealthAloneDoesNotEndOrBypassTheApproach pins the retained target rule at
// the approach: a linked target at negative health is still faced and acted on.
func TestHealthAloneDoesNotEndOrBypassTheApproach(t *testing.T) {
	const mark = 0xa0
	w := mustWorldGrid(t, 1, turnBounds, ModeCanonical, openGrid(turnBounds), []Entity{
		armed(1, 5, 5, mark),
		armed(2, 6, 5, 0),
	})
	Step(w, []Command{{Kind: KindAttack, Entity: 1, X: 2}, {Kind: KindKill, Entity: 2}})

	a := w.Entities()[0]
	if !a.HasAttackTarget {
		t.Fatal("negative target health ended the retained order")
	}
	if a.Facing == mark {
		t.Errorf("the retained approach left facing at %d instead of facing its linked target", mark)
	}
}

func TestAFacingGatesNothing(t *testing.T) {
	build := func(f uint8) *World {
		return mustWorldGrid(t, 9, turnBounds, ModeCanonical, openGrid(turnBounds), []Entity{
			// An attacker that can miss, so the roll is drawn and consumed; a
			// victim that can be walked to and felled.
			{ID: 1, X: 1, Y: 1, HP: 100, MaxHP: 100, Facing: f,
				AttackCharge: 2, AttackRelax: 1, ToHit: 40, DamageBase: 9, DamageSpread: 5},
			{ID: 2, X: 7, Y: 6, HP: 60, MaxHP: 60, Defence: 20},
		})
	}
	masked := func(w *World) ([]Entity, [][2]int32, [][2]int32) {
		ents := w.Entities()
		for i := range ents {
			ents[i].Facing = 0
		}
		return ents, w.Route(1), w.Route(2)
	}

	a, b := build(0), build(0xff)
	// The two DO differ where they must, and it is asserted HERE rather than at
	// the end: the field is canonical, so two worlds differing in it are two
	// worlds — but the schedule below walks the attacker in and turns it, so by
	// the last tick both hold the same facing and the difference is gone. That
	// convergence is the story working, not the field failing to reach the form.
	if a.Hash() == b.Hash() {
		t.Fatal("two worlds differing in a facing hash alike")
	}
	cmds := []Command{{Kind: KindAttack, Entity: 1, X: 2}}
	for tick := 0; tick < 60; tick++ {
		Step(a, cmds)
		Step(b, cmds)
		cmds = nil

		ae, ar1, ar2 := masked(a)
		be, br1, br2 := masked(b)
		if !reflect.DeepEqual(ae, be) {
			t.Fatalf("tick %d: the two worlds diverged in something other than a facing:\n %+v\n %+v",
				tick, ae, be)
		}
		if !reflect.DeepEqual(ar1, br1) || !reflect.DeepEqual(ar2, br2) {
			t.Fatalf("tick %d: the two worlds hold different routes", tick)
		}
	}
	// And the schedule has to have DONE something, or the case above would hold
	// over two worlds in which nothing happened.
	if got := a.Entities()[1].HP; got == 60 {
		t.Fatal("the victim took no damage; this case measures nothing")
	}
}
