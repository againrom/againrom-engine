package sim

import (
	"bytes"
	"testing"
)

// puWorld builds a world over an open 24x24 grid — wider than the combat tests'
// 16x16, because an approach needs room to be an approach.
func puWorld(t *testing.T, seed uint64, ents ...Entity) *World {
	t.Helper()
	w, err := NewWorld(seed, Bounds{Width: 24, Height: 24}, ModeCanonical, nil, ents)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	return w
}

// puFighter is an attacker that always hits for a fixed amount, with a one-tick
// charge and no relax, so every advance it spends in reach costs the victim
// something and the arithmetic of a case is the distance alone.
func puFighter(id EntityID, x, y int32) Entity {
	e := cbEnt(id, x, y)
	e.AttackCharge, e.AttackRelax = 1, 0
	e.DamageBase, e.AlwaysHits = 10, true
	return e
}

func TestAnAttackerWalksToItsVictim(t *testing.T) {
	w := puWorld(t, 7, puFighter(1, 0, 0), cbEnt(2, 8, 0))
	Step(w, []Command{cbOrder(1, 2)})

	// It set out on the ordering advance itself: the approach runs in the same
	// tick the command arrives in, so the first step is not paid for twice.
	if e := cbAt(t, w, 1); e.X == 0 && e.Y == 0 {
		t.Errorf("the attacker did not set out on the ordering advance, still at %d,%d", e.X, e.Y)
	}
	for n := 0; n < 40; n++ {
		if cbAt(t, w, 2).HP < 100 {
			break
		}
		Step(w, nil)
	}
	v, a := cbAt(t, w, 2), cbAt(t, w, 1)
	if v.HP >= 100 {
		t.Fatalf("no blow landed in 40 advances; the attacker stands at %d,%d and the victim at %d,%d",
			a.X, a.Y, v.X, v.Y)
	}
	// And it stopped where it could strike rather than on top of what it was
	// walking at: the stop distance is the reach.
	if !inReach(a, v) {
		t.Errorf("the attacker landed a blow from %d,%d on a victim at %d,%d, which is out of reach",
			a.X, a.Y, v.X, v.Y)
	}
	if a.HasTarget {
		t.Errorf("the attacker still holds destination %d,%d after arriving", a.TargetX, a.TargetY)
	}
}

func TestAnAdjacentVictimIsNotWalkedTo(t *testing.T) {
	w := puWorld(t, 11, puFighter(1, 4, 4), cbEnt(2, 5, 4))
	for n := 0; n < 6; n++ {
		Step(w, []Command{cbOrder(1, 2)})
		if a := cbAt(t, w, 1); a.X != 4 || a.Y != 4 || a.HasTarget {
			t.Fatalf("advance %d: the attacker moved to %d,%d or holds a destination (%v)",
				n, a.X, a.Y, a.HasTarget)
		}
	}
	if cbAt(t, w, 2).HP >= 100 {
		t.Error("standing still, it never struck")
	}
}

func TestAVictimThatMovesIsFollowed(t *testing.T) {
	// Both move a cell a tick — neither carries a rate — so the victim keeps
	// walking for the whole run and the attacker never closes on it. That is the
	// state this case needs: a pursuit that is still a pursuit at the end.
	w := puWorld(t, 13, puFighter(1, 0, 0), cbEnt(2, 6, 0))
	Step(w, []Command{cbOrder(1, 2), {Kind: KindMoveTo, Entity: 2, X: 20, Y: 20}})

	seen := map[[2]int32]bool{}
	for n := 0; n < 12; n++ {
		// Where the victim stands as this advance BEGINS, which is what the
		// attacker's turn sees: the loop resolves entities in ascending id, so
		// the attacker is aimed before the victim of a higher id has moved.
		was := cbAt(t, w, 2)
		Step(w, nil)
		a := cbAt(t, w, 1)
		if !a.HasAttackTarget || !a.HasTarget {
			continue
		}
		seen[[2]int32{a.TargetX, a.TargetY}] = true
		if a.TargetX != was.X || a.TargetY != was.Y {
			t.Fatalf("advance %d: the attacker aims at %d,%d; its victim began the advance at %d,%d",
				n, a.TargetX, a.TargetY, was.X, was.Y)
		}
	}
	if len(seen) < 2 {
		t.Errorf("the attacker aimed at %d distinct cell(s) while its victim walked; a re-aim visits more "+
			"than one", len(seen))
	}
}

// TestTargetHealthAloneDoesNotEndTheWalk keeps the pursuit on a target that is
// still linked. The removal path owns teardown; health does not duplicate it.
func TestTargetHealthAloneDoesNotEndTheWalk(t *testing.T) {
	w := puWorld(t, 17, puFighter(1, 0, 0), cbEnt(2, 10, 0))
	Step(w, []Command{cbOrder(1, 2)})
	if !cbAt(t, w, 1).HasTarget {
		t.Fatal("the fixture needs an attacker under way")
	}
	Step(w, []Command{{Kind: KindKill, Entity: 2}})
	a := cbAt(t, w, 1)
	if !a.HasTarget || a.TargetX != 10 || a.TargetY != 0 || a.Stall != 0 {
		t.Errorf("the linked target's health ended pursuit: target %v (%d,%d), stall %d",
			a.HasTarget, a.TargetX, a.TargetY, a.Stall)
	}
	if !a.HasAttackTarget {
		t.Error("the linked target's health ended the attack order")
	}
}

func TestAMoveOrderStillEndsAnApproach(t *testing.T) {
	w := puWorld(t, 19, puFighter(1, 0, 0), cbEnt(2, 12, 0))
	Step(w, []Command{cbOrder(1, 2)})
	Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 0, Y: 9}})
	a := cbAt(t, w, 1)
	if a.HasAttackTarget || a.AttackPhase != AttackReady || a.AttackCountdown != 0 {
		t.Errorf("the fight outlived the move order: victim %d (%v), phase %d, %d owed",
			a.AttackTarget, a.HasAttackTarget, a.AttackPhase, a.AttackCountdown)
	}
	if !a.HasTarget || a.TargetX != 0 || a.TargetY != 9 {
		t.Errorf("the mover holds %v (%d,%d), want the move order's own 0,9",
			a.HasTarget, a.TargetX, a.TargetY)
	}
}

func TestAnApproachRoundTrips(t *testing.T) {
	w := puWorld(t, 23, puFighter(1, 0, 0), cbEnt(2, 14, 3))
	Step(w, []Command{cbOrder(1, 2)})
	a := cbAt(t, w, 1)
	if !a.HasAttackTarget || !a.HasTarget {
		t.Fatalf("the fixture needs both orders at once, got victim %v and destination %v",
			a.HasAttackTarget, a.HasTarget)
	}

	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	// 12 while this story was written, 13 since 0076 gave a world its cost and
	// height planes, 14 since 0081 gave an entity a facing, 15 since 0086 gave a
	// world its relation. The claim was never about the NUMBER — it is that a
	// pursuit round-trips at whatever version the tree writes, and that this story
	// added no field of its own to make it move. The message said 13 for two
	// bumps after the literal said 14, which is the same thing the version-refusal
	// test's own doc block is on record about: a number carried by prose drifts.
	if b[0] != formatVersion {
		t.Errorf("the byte form reads version %d, want the %d this tree writes", b[0], formatVersion)
	}
	var got World
	if err := got.UnmarshalBinary(b); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	again, err := got.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary (again): %v", err)
	}
	if !bytes.Equal(b, again) {
		t.Error("a world mid-approach did not survive the round trip")
	}
	if got.Hash() != w.Hash() {
		t.Errorf("the decoded world hashes %016x, the original %016x", got.Hash(), w.Hash())
	}
}
