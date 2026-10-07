package sim

import "testing"

// 0094: what the player's roster slot buys, and what it costs.
//
// Every world here is built in test code over the same two-slot relation shape:
// SelfSlot is the player's and the other is a map faction's. Nothing reads an
// install and nothing reads a map.

// psWorld is a world holding one entity on SelfSlot and one on SelfSlot+1, at the
// separation given, under the relation cells named.
//
// THE PLAYER'S ENTITY IS ALWAYS ID 1 and the faction's always id 2, so every
// assertion below names the same two ids and a reader never has to work out which
// way round a case put them.
func psWorld(t *testing.T, apart int32, cells ...[3]uint32) *World {
	t.Helper()
	return engWorld(t, engRel(t, cells...),
		engFighter(1, SelfSlot, 5, 5), engFighter(2, SelfSlot+1, 5+apart, 5))
}

// TestTheMatrixDecidesEachDirectionOnItsOwn is AC-4, AC-5 and AC-6 together,
// because the three are one table and splitting them would hide what the table is
// for: the relation is NOT symmetric, shipped content uses that, and a build that
// tested only the mutual case would pass with either direction wired to the other.
//
// The distances are chosen so that neither answer can come from geometry. The two
// stand adjacent, which is inside reach for the stand-ground scorer the player's
// slot selects and inside any circle the guard arm can compute, so what is left to
// decide either acquisition is the matrix.
func TestTheMatrixDecidesEachDirectionOnItsOwn(t *testing.T) {
	t.Parallel()

	const self, other = uint32(SelfSlot), uint32(SelfSlot + 1)
	for _, tc := range []struct {
		name                  string
		cells                 [][3]uint32
		playerHit, factionHit bool
	}{
		{"neither direction", nil, false, false},
		{"the faction hates the player", [][3]uint32{{other, self, 1}}, true, false},
		{"the player hates the faction", [][3]uint32{{self, other, 1}}, false, true},
		{"mutual", [][3]uint32{{other, self, 1}, {self, other, 1}}, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := psWorld(t, 1, tc.cells...)
			engRun(w, 1)
			if _, held := engVictim(w, 2); held != tc.playerHit {
				t.Errorf("the faction acquired the player = %v, want %v", held, tc.playerHit)
			}
			if _, held := engVictim(w, 1); held != tc.factionHit {
				t.Errorf("the player acquired the faction = %v, want %v", held, tc.factionHit)
			}
		})
	}
}

// TestThePlayerStandsHisGroundAndDoesNotChase is AC-7: the stance SelfSlot
// selects refuses every candidate past reach outright, so a party member with a
// hostile in plain sight and one cell too far is given nothing and stays put.
//
// The adjacent case is the discriminator. Without it, an implementation that had
// simply dropped the player out of the decision again would pass this test.
func TestThePlayerStandsHisGroundAndDoesNotChase(t *testing.T) {
	t.Parallel()

	const self, other = uint32(SelfSlot), uint32(SelfSlot + 1)
	for _, tc := range []struct {
		apart int32
		want  bool
	}{{groupScorerReach, true}, {groupScorerReach + 1, false}} {
		w := psWorld(t, tc.apart, [3]uint32{self, other, 1})
		before := entityAt(t, w, 1)
		engRun(w, 1)
		after := entityAt(t, w, 1)
		if _, held := engVictim(w, 1); held != tc.want {
			t.Errorf("at distance %d the player acquired = %v, want %v", tc.apart, held, tc.want)
		}
		if after.X != before.X || after.Y != before.Y {
			t.Errorf("at distance %d the player moved from (%d,%d) to (%d,%d) — "+
				"the stand-ground arm has no walk in it",
				tc.apart, before.X, before.Y, after.X, after.Y)
		}
	}
}

// entityAt is the entity id, failing rather than answering a zero value for one
// the world does not hold — absent is not "at the origin".
func entityAt(t *testing.T, w *World, id EntityID) Entity {
	t.Helper()
	for _, e := range w.Entities() {
		if e.ID == id {
			return e
		}
	}
	t.Fatalf("the world holds no entity %d", id)
	return Entity{}
}

func TestAPlayerOrderNeedsNoReissuePastAHostileThatCloses(t *testing.T) {
	t.Parallel()

	const self, other = uint32(SelfSlot), uint32(SelfSlot + 1)
	// The destination is straight past the hostile and well outside reach, so
	// arriving and being held are far apart on the map as well as in the state.
	const destX, destY = 25, 5
	// BOTH UNITS WALK, and that is the fixture rather than a detail: the
	// faction is on the guard arm and closes, so a player who broke contact for
	// a few ticks used to be back in reach by the next decision.
	walker := func(id EntityID, owner uint32, x int32) Entity {
		e := engFighter(id, owner, x, 5)
		e.Speed, e.AttackCharge, e.AttackRelax = 10, 4, 4
		// DEEP HEALTH, so that what this test measures is where a unit ENDS UP
		// and never which of the two fell first. At the fixture's own 20 the
		// player dies inside the window and every position below becomes a
		// statement about a corpse.
		e.HP, e.MaxHP = 4000, 4000
		return e
	}
	drive := func(owner uint32, reissueEvery int) Entity {
		w := engWorld(t, engRel(t, [3]uint32{self, other, 1}, [3]uint32{other, self, 1}),
			walker(1, owner, 5), walker(2, SelfSlot+1, 6))
		order := Command{Kind: KindMoveTo, Entity: 1, X: destX, Y: destY}
		Step(w, []Command{order})
		for i := 1; i < 12*scriptCycle; i++ {
			var cmds []Command
			if reissueEvery > 0 && i%reissueEvery == 0 {
				cmds = []Command{order}
			}
			Step(w, cmds)
		}
		return entityAt(t, w, 1)
	}

	// ONE order, no reissue, at the player's slot — the case FR-6a used to say
	// was overridden on contact — set against the same drive by a unit no
	// group can ever see.
	single := drive(SelfSlot, 0)
	unowned := drive(0, 0)
	if single.X != unowned.X || single.Y != unowned.Y {
		t.Errorf("a single order at the player's slot reached (%d,%d); the same drive at "+
			"slot 0 reached (%d,%d) — a unit under command is expected to cover the same "+
			"ground as one no group can see, with a single order and no re-issue at all",
			single.X, single.Y, unowned.X, unowned.Y)
	}
	if single.X == 5 || single.X == 6 {
		t.Errorf("a single order at the player's slot is still sitting at (%d,%d), the cell "+
			"the hostile held it at — the decision dropped its walk on contact, which is "+
			"exactly the defect FR-11 takes back", single.X, single.Y)
	}
	if single.HasAttackTarget {
		t.Error("a single order at the player's slot left the unit holding a victim — the " +
			"decision engaged it despite the destination it was still holding")
	}

	reissued := drive(SelfSlot, scriptCycle/2)
	if reissued.X != single.X || reissued.Y != single.Y {
		t.Errorf("re-issued, the player's unit reached (%d,%d); a single order reached "+
			"(%d,%d) — re-issuing is expected to be indistinguishable from ordering once, "+
			"now that no decision ever drops the walk to begin with",
			reissued.X, reissued.Y, single.X, single.Y)
	}
}
