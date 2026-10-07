package sim

import "testing"

// This file covers 0097: the predicate underCommand and its one clause inside
// aiGroups. It builds on engage_test.go's own synthetic-world helpers (engWorld,
// engRel, engFighter, engVictim) rather than inventing a second set, and no test
// here reads a game install.

// cmdToDecision steps w up to, but not including, the tick its next decision
// runs on. A caller that wants to observe exactly one decision follows it with
// one more Step — which, because the move phase runs after the decision inside
// one Step call, also shows whether the decided-over (or skipped) unit advanced
// across that same tick.
func cmdToDecision(w *World) {
	for w.Tick()%scriptCycle != scriptPassPhase {
		Step(w, nil)
	}
}

// cmdEntity is id's current state, or a fatal test failure if the world holds
// none such.
func cmdEntity(t *testing.T, w *World, id EntityID) Entity {
	t.Helper()
	for _, e := range w.Entities() {
		if e.ID == id {
			return e
		}
	}
	t.Fatalf("the world holds no entity %d", id)
	return Entity{}
}

// TestACommandedUnitKeepsWalkingPastAHostileInReach is AC-1, with AC-3 run
// beside it as the control over the identical geometry: the same fixture,
// minus the order, is what today's engagement decision already does to a unit
// beside a hostile. AC-1 is that same world WITH the order, and the whole
// story is the difference the order makes.
func TestACommandedUnitKeepsWalkingPastAHostileInReach(t *testing.T) {
	t.Parallel()

	// The command is issued in the SAME Step call as the decision — commands
	// apply before the engagement pass in one tick — so the fixture never has
	// to account for six ticks of drift between "ordered" and "decided", and
	// AC-1 and AC-2 read off literally one shared moment.
	build := func(t *testing.T) *World {
		t.Helper()
		w := engWorld(t, engRel(t, [3]uint32{SelfSlot, 9, 1}, [3]uint32{9, SelfSlot, 1}),
			engFighter(1, SelfSlot, 5, 5), engFighter(2, 9, 6, 5))
		cmdToDecision(w)
		return w
	}

	t.Run("AC-3: unordered, it still fights what is within reach", func(t *testing.T) {
		t.Parallel()
		w := build(t)
		Step(w, nil)
		if v, held := engVictim(w, 1); !held || v != 2 {
			t.Fatalf("an unordered unit beside a hostile holds victim %v/%v, want entity 2 taken — "+
				"the fixture itself is not producing a fight", v, held)
		}
	})

	t.Run("AC-1: commanded, it keeps walking and takes nothing", func(t *testing.T) {
		t.Parallel()
		w := build(t)
		before := cmdEntity(t, w, 1)
		Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 5, Y: 20}})
		after := cmdEntity(t, w, 1)

		if !after.HasTarget || after.TargetX != 5 || after.TargetY != 20 {
			t.Errorf("a commanded unit's destination is held=%v (%d,%d) after a decision, want held "+
				"at (5,20)", after.HasTarget, after.TargetX, after.TargetY)
		}
		if after.HasAttackTarget {
			t.Errorf("a commanded unit was given victim %d by a decision", after.AttackTarget)
		}
		if after.X == before.X && after.Y == before.Y {
			t.Errorf("a commanded unit did not advance across the tick its decision ran on: "+
				"stayed at (%d,%d)", after.X, after.Y)
		}
	})
}

func TestACommandedUnitStaysACandidateForOthers(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t, [3]uint32{SelfSlot, 9, 1}, [3]uint32{9, SelfSlot, 1}),
		engFighter(1, SelfSlot, 5, 5), engFighter(2, 9, 6, 5))
	cmdToDecision(w)
	Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 5, Y: 20}})

	if v, held := engVictim(w, 2); !held || v != 1 {
		t.Errorf("the hostile's own group holds victim %v/%v, want the commanded unit (1) acquired — "+
			"FR-4 says it is still a candidate for everyone else", v, held)
	}
}

// TestACommandedFighterStillDecidesForItsGroup is AC-4: a unit that holds a
// victim, and a destination the approach wrote toward it, is a DECIDER — not
// under command, because HasAttackTarget is set — and the decision proves it
// by re-scoring the unit mid-pursuit: a nearer hostile comes into sight and the
// victim changes out from under the one issued by hand. If underCommand read
// HasTarget alone this unit would be wrongly skipped and the victim would
// never move off the one the attack command named.
func TestACommandedFighterStillDecidesForItsGroup(t *testing.T) {
	t.Parallel()

	member := engFighter(1, 2, 5, 5)
	far := engFighter(2, 3, 5, 15)  // the victim the attack command names
	near := engFighter(3, 3, 6, 11) // within sight, and nearer once the chase closes
	w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), member, far, near)

	Step(w, []Command{{Kind: KindAttack, Entity: 1, X: 2}})
	got := cmdEntity(t, w, 1)
	if !got.HasAttackTarget || got.AttackTarget != 2 || !got.HasTarget {
		t.Fatalf("the attack order did not leave entity 1 chasing entity 2 with a destination: %+v", got)
	}

	cmdToDecision(w)
	before := cmdEntity(t, w, 1)
	if !before.HasTarget || !before.HasAttackTarget {
		t.Fatalf("entity 1 is not mid-pursuit at the decision tick (destination held=%v, victim "+
			"held=%v) — this test's own timing assumption is wrong", before.HasTarget, before.HasAttackTarget)
	}

	Step(w, nil)
	got = cmdEntity(t, w, 1)
	if got.AttackTarget != 3 {
		t.Errorf("a unit mid-pursuit was not re-scored: it still holds victim %d, want the nearer "+
			"candidate (3) taken instead — which is what proves it was decided over rather than "+
			"skipped as under command", got.AttackTarget)
	}
}

// TestArrivalEndsCommandAndTheNextDecisionEngages is AC-5: the tick a commanded
// unit arrives, its destination ends with no victim taken on arrival alone; the
// next decision after that finds a decider again and gives it the hostile
// standing beside where it stopped.
func TestArrivalEndsCommandAndTheNextDecisionEngages(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t, [3]uint32{SelfSlot, 9, 1}),
		engFighter(1, SelfSlot, 5, 5), engFighter(2, 9, 7, 5))
	Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 6, Y: 5}})

	arrived := cmdEntity(t, w, 1)
	if arrived.HasTarget {
		t.Fatalf("the unit still holds a destination on the tick it arrived: %+v", arrived)
	}
	if arrived.X != 6 || arrived.Y != 5 {
		t.Fatalf("the unit did not arrive at (6,5): got (%d,%d)", arrived.X, arrived.Y)
	}
	if arrived.HasAttackTarget {
		t.Fatalf("arrival alone gave the unit a victim: %d", arrived.AttackTarget)
	}

	cmdToDecision(w)
	Step(w, nil)

	if v, held := engVictim(w, 1); !held || v != 2 {
		t.Errorf("the next decision after arrival holds victim %v/%v, want the adjacent hostile "+
			"(2) taken", v, held)
	}
}

func TestBeingStruckDoesNotEndCommand(t *testing.T) {
	t.Parallel()

	w := engWorld(t, Relations{}, engFighter(1, SelfSlot, 5, 5))
	Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 5, Y: 20}})
	Step(w, []Command{{Kind: KindDamage, Entity: 1, X: 5}})

	got := cmdEntity(t, w, 1)
	if !got.HasTarget || got.TargetX != 5 || got.TargetY != 20 {
		t.Fatalf("a struck commanded unit's destination is held=%v (%d,%d), want held at (5,20)",
			got.HasTarget, got.TargetX, got.TargetY)
	}
	if got.HasAttackTarget {
		t.Fatalf("a struck commanded unit acquired victim %d", got.AttackTarget)
	}

	before := got
	Step(w, nil)
	after := cmdEntity(t, w, 1)
	if after.X == before.X && after.Y == before.Y {
		t.Errorf("a struck commanded unit did not keep advancing: stayed at (%d,%d)", after.X, after.Y)
	}
}

func TestACommandedGroupContributesNoSight(t *testing.T) {
	t.Parallel()

	t.Run("AC-7: neither commanded member is given a victim", func(t *testing.T) {
		t.Parallel()
		a := engFighter(1, 5, 10, 10)
		a.ScanRange = 8
		b := engFighter(2, 5, 30, 10)
		b.ScanRange = 8
		hostile := engFighter(3, 9, 10, 15) // 5 from A, 20 from B — only A can see it alone
		w := engWorld(t, engRel(t, [3]uint32{5, 9, 1}), a, b, hostile)

		Step(w, []Command{
			{Kind: KindGroupMoveTo, Entity: 1, X: 10, Y: 40, Group: 1},
			{Kind: KindGroupMoveTo, Entity: 2, X: 10, Y: 40, Group: 1},
		})
		cmdToDecision(w)
		Step(w, nil)

		for _, id := range []EntityID{1, 2} {
			if v, held := engVictim(w, id); held {
				t.Errorf("commanded group member %d holds victim %v", id, v)
			}
		}
	})

	t.Run("AC-7/FR-5: a commanded member's sight does not reach its old group", func(t *testing.T) {
		t.Parallel()
		a := engFighter(1, 6, 10, 10)
		a.ScanRange = 8
		b := engFighter(2, 6, 30, 10)
		b.ScanRange = 8
		hostile := engFighter(3, 9, 10, 15) // 5 from A, 20 from B
		w := engWorld(t, engRel(t, [3]uint32{6, 9, 1}), a, b, hostile)

		// Only the sighted member (A) is commanded; B is left free and is the
		// group's only surviving decider.
		Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 10, Y: 40}})
		cmdToDecision(w)
		Step(w, nil)

		if v, held := engVictim(w, 2); held {
			t.Errorf("the surviving group member holds victim %v — it can only have seen the "+
				"hostile through the commanded member's sight, which FR-5 says it must not", v)
		}
		if v, held := engVictim(w, 1); held {
			t.Errorf("the commanded member itself was given victim %v", v)
		}
	})
}

// TestARepeatedMoveOrderStaysUnderCommand is AC-8: a second move order onto a
// commanded unit leaves it commanded still, now toward the new cell.
func TestARepeatedMoveOrderStaysUnderCommand(t *testing.T) {
	t.Parallel()

	w := engWorld(t, Relations{}, engFighter(1, 2, 5, 5))
	Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 5, Y: 20}})
	Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 40, Y: 5}})

	got := cmdEntity(t, w, 1)
	if !got.HasTarget || got.TargetX != 40 || got.TargetY != 5 {
		t.Errorf("a second move order left the destination held=%v (%d,%d), want held at (40,5)",
			got.HasTarget, got.TargetX, got.TargetY)
	}
	if got.HasAttackTarget {
		t.Errorf("a second move order left a victim held: %d", got.AttackTarget)
	}
	if !underCommand(got) {
		t.Errorf("the unit is no longer read as under command after a second move order: %+v", got)
	}
}

// TestAnAttackCommandEndsCommand is AC-9: an attack command onto a commanded
// unit ends the state and gives it that victim. The victim starts out of reach
// so the attack command also leaves the unit mid-pursuit — holding a
// destination the approach wrote, exactly as AC-4's fixture does — which is
// what makes the underCommand check below discriminate the predicate's
// "and no victim" half rather than passing merely because no destination is
// held at all.
func TestAnAttackCommandEndsCommand(t *testing.T) {
	t.Parallel()

	w := engWorld(t, Relations{}, engFighter(1, 2, 5, 5), engFighter(2, 3, 5, 9))
	Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 5, Y: 20}})
	Step(w, []Command{{Kind: KindAttack, Entity: 1, X: 2}})

	got := cmdEntity(t, w, 1)
	if !got.HasAttackTarget || got.AttackTarget != 2 {
		t.Fatalf("an attack command left victim held=%v/%v, want entity 2 held",
			got.HasAttackTarget, got.AttackTarget)
	}
	if !got.HasTarget {
		t.Fatalf("the victim is not out of reach — this test needs the unit mid-pursuit, holding "+
			"both a destination and a victim, for the check below to mean anything: %+v", got)
	}
	if underCommand(got) {
		t.Errorf("the unit is still read as under command after an attack order, despite holding "+
			"a victim: %+v", got)
	}
}

// TestFallingEndsCommand is AC-10: a commanded unit that falls holds neither a
// destination nor a victim, and is under command no longer.
func TestFallingEndsCommand(t *testing.T) {
	t.Parallel()

	w := engWorld(t, Relations{}, engFighter(1, 2, 5, 5))
	Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 5, Y: 20}})
	Step(w, []Command{{Kind: KindKill, Entity: 1}})

	got := cmdEntity(t, w, 1)
	if got.HasTarget {
		t.Errorf("a felled commanded unit still holds a destination: (%d,%d)", got.TargetX, got.TargetY)
	}
	if got.HasAttackTarget {
		t.Errorf("a felled commanded unit holds a victim: %d", got.AttackTarget)
	}
	if underCommand(got) {
		t.Errorf("a felled unit is still read as under command: %+v", got)
	}
}

func TestACommandedWorldRoundTripsAndHashesToTheSameDigest(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t, [3]uint32{SelfSlot, 9, 1}),
		engFighter(1, SelfSlot, 5, 5), engFighter(2, 9, 20, 20))
	Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 5, Y: 20}})

	if got := cmdEntity(t, w, 1); !underCommand(got) {
		t.Fatalf("the fixture does not hold a commanded unit: %+v", got)
	}

	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	// NO VERSION LITERAL, since 0166. This spelled the live version number and
	// had to be re-spelled by every story that bumped it, which is the shape
	// that has gone stale and been repaired after the fact three times in this
	// package. What this test witnesses is that a COMMANDED world round trips
	// and hashes to the same digest; the version it does that at is whatever
	// the constant says, and the version's own tripwire is the pinned form and
	// digest in binary_test.go and hash_test.go, which no unexamined bump can
	// satisfy.
	if b[0] != formatVersion {
		t.Fatalf("the form's version byte is %d, want the constant %d", b[0], formatVersion)
	}

	var back World
	if err := back.UnmarshalBinary(b); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	again, err := back.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary (round trip): %v", err)
	}
	if string(again) != string(b) {
		t.Fatal("a world holding a commanded unit does not round-trip through its byte form")
	}
	if back.Hash() != w.Hash() {
		t.Fatal("a round-tripped commanded world hashes differently from the original")
	}

	// 1029 moves it: this world is built rather than loaded from a map and
	// compiles no script, so the whole of the version-57 bump here is the
	// version byte and two zero bytes at the tail of each entity record. The
	// version-56 value this replaces was 0x2d9d41f57762fc01.
	//
	// 1033 moves it again (B3): this world declares no structure, so the
	// whole of the version-58 bump here is the version byte and the new
	// structure section's own bare four-byte zero count. The version-57
	// value this replaces was 0xc546eb680c4a4316.
	//
	// 1039 appends five zero resistance bytes to each entity. The version-58
	// value this replaces was 0xe25d460128b1278b.
	//
	// 1047 appends two zero-remainder inactive-turn bytes to each entity. The
	// version-62 value this replaces was 0x9f2db1d0f3f0d645.
	// 1052 moves the form version; this fixture has no structure records. 1063
	// moves it again for raw cloud counters; this fixture has no area record.
	requireStructureLegacyDigest(t, w, 0x3f39c912b3932741)
	requireOriginalDeadLegacyDigest(t, w, 0x9de05436bbfafd74)
	requireSpellbookLegacyDigest(t, w, 0xeb5ca5cb437d67ed)
	const wantDigest uint64 = 0x4cc680b6ab9204f6
	requireHumanMovementLegacyDigest(t, w, 0xab0f9871795a63bf)
	if got := fnv1a(strippedWorldOfSecondPhysical(mustMarshal(t, w))); got != wantDigest {
		t.Errorf("a commanded world's digest is %#x, want the literal %#x pinned from this fixture "+
			"— a later change to the byte form or the digest must fail this loudly rather than "+
			"pass by reading itself back", got, wantDigest)
	}
}

func TestNoCommandedUnitLeavesTheWorldUnchanged(t *testing.T) {
	t.Parallel()

	build := func(t *testing.T) *World {
		t.Helper()
		return engWorld(t, engRel(t, [3]uint32{2, 3, 1}, [3]uint32{3, 2, 1}),
			engFighter(1, 2, 5, 5), engFighter(2, 3, 9, 8),
			engFighter(3, 2, 6, 7), engFighter(4, 3, 10, 5))
	}
	a, b := build(t), build(t)
	for i := 0; i < 64; i++ {
		Step(a, nil)
		Step(b, nil)
		for _, e := range a.Entities() {
			if underCommand(e) {
				t.Fatalf("tick %d: entity %d is under command in a fixture that issues no move "+
					"orders at all — AC-13 needs none for the comparison below to mean anything",
					a.Tick(), e.ID)
			}
		}
	}
	ea, eb := a.Entities(), b.Entities()
	if len(ea) != len(eb) {
		t.Fatalf("entity counts differ after 64 ticks: %d vs %d", len(ea), len(eb))
	}
	for i := range ea {
		if ea[i] != eb[i] {
			t.Errorf("entity %d differs after 64 ticks with no commanded unit:\n a %+v\n b %+v",
				ea[i].ID, ea[i], eb[i])
		}
	}
}
