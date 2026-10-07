package sim

import "testing"

// pxPatroller is a patroller of its own: entity 1, owner 2, group 1, standing at
// (x, y) and commanded to patrol to (tx, ty) through the script's own
// sub-command 14. The relation row makes owner 2 and owner 3 mutually
// hostile, which is the pair every candidate below is placed on.
//
// The bounds are this file's own 128x128 rather than engBounds, because the
// mission-80-shaped case needs cells past 48 and one bounds for the file
// keeps every distance in it comparable.
var pxBounds = Bounds{Width: 128, Height: 128}

func pxPatroller(t *testing.T, x, y, tx, ty int32, others ...Entity) *World {
	t.Helper()
	a := engFighter(1, 2, x, y)
	a.Group = 1
	rel := engRel(t, [3]uint32{2, 3, relationHostile}, [3]uint32{3, 2, relationHostile})
	w, err := NewRelatedWorld(1, pxBounds, ModeCanonical, Terrain{}, append([]Entity{a}, others...), nil, rel)
	if err != nil {
		t.Fatalf("NewRelatedWorld: %v", err)
	}
	w.runInstant(ScriptInstant{Op: ScriptInstantGroupOrder, Group: 1, HasGroup: true,
		Args: [scriptParams]int32{subCommandPatrol, tx, ty}})
	if got := entityAt(t, w, 1); got.ActorState != actorStatePatrol {
		t.Fatalf("fixture: entity 1 is in actor state %d, want patrol — the command did not land", got.ActorState)
	}
	return w
}

// pxHostile is a candidate on owner 3, which pxPatroller's relation row makes
// hostile to the patroller, standing at (x, y).
func pxHostile(id EntityID, x, y int32) Entity { return engFighter(id, 3, x, y) }

// ------------------------------------------------------- the player result

// TestAPatrollerFightsWhatStandsInsideItsBlock is the story's whole point: a
// patrolling actor with a hostile inside the five-cell block around its post
// engages it and does NOT walk on.
//
// THE ENGAGEMENT IS PROVABLY THE BLOCK'S AND NOT THE STANDING PICKER'S. The
// candidate stands three cells away and the patroller's reach is 1;
// acquireStanding scores under stand ground, which refuses every candidate
// past reach (`AI-STAND-076`), so nothing but the post block can have
// produced this victim.
//
// AI-PATROL-013
func TestAPatrollerFightsWhatStandsInsideItsBlock(t *testing.T) {
	t.Parallel()

	w := pxPatroller(t, 10, 10, 20, 10, pxHostile(2, 13, 10))
	w.actorPass()

	got := entityAt(t, w, 1)
	if !got.HasAttackTarget || got.AttackTarget != 2 {
		t.Fatalf("the patroller holds victim %d (present %v), want entity 2 — it walked past a hostile "+
			"three cells from its post", got.AttackTarget, got.HasAttackTarget)
	}
	if got.HasTarget {
		t.Errorf("the patroller also holds destination (%d,%d) — guard engaged, so the ring's own "+
			"advance must not have run (AI-PATROL-013's 0/0xb gate)", got.TargetX, got.TargetY)
	}
	if got.PatrolLeg != patrolLegTail {
		t.Errorf("the ring advanced to leg %d while guard was engaging; the gate did not hold", got.PatrolLeg)
	}
}

// TestAPatrollerWalksOnPastANonHostileNeighbour is the first negative: a
// neighbour the patroller is not hostile to is not in the block at all, so
// the ring advance runs exactly as it did before this story.
func TestAPatrollerWalksOnPastANonHostileNeighbour(t *testing.T) {
	t.Parallel()

	// Owner 2 is the patroller's own slot and no relation row makes it
	// hostile to itself, so this neighbour standing one cell away is a
	// candidate the block filter drops.
	w := pxPatroller(t, 10, 10, 20, 10, engFighter(2, 2, 11, 10))
	w.actorPass()

	got := entityAt(t, w, 1)
	if got.HasAttackTarget {
		t.Fatalf("the patroller engaged entity %d, which it is not hostile to", got.AttackTarget)
	}
	if !got.HasTarget || got.TargetX != 20 || got.TargetY != 10 {
		t.Errorf("the patroller holds destination (%d,%d) present %v, want its tail waypoint (20,10) — "+
			"a patrol that stops patrolling is not a patrol", got.TargetX, got.TargetY, got.HasTarget)
	}
}

// TestAPatrollerWalksOnPastAHostileOutsideTheBlock is the second negative
// and the one that pins the distance: the same hostile, moved to six cells
// from the post, is outside `AI-BREAK-041`'s five-cell block and changes
// nothing. Five is `mover+0x08`, not sight — the patroller's own scan range
// here is 5 as well, so a build that had scanned instead of blocked would
// have engaged at six only if it used sight; the case that discriminates the
// two is the boundary pair below.
func TestAPatrollerWalksOnPastAHostileOutsideTheBlock(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		at       int32
		engaging bool
	}{
		{"five cells from the post, inside the block", 15, true},
		{"six cells from the post, outside it", 16, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := pxPatroller(t, 10, 10, 20, 10, pxHostile(2, tc.at, 10))
			w.actorPass()

			got := entityAt(t, w, 1)
			if got.HasAttackTarget != tc.engaging {
				t.Fatalf("at Chebyshev %d from the post the patroller %s, want the opposite — "+
					"the block radius is 5 (AI-BREAK-041, mover+0x08)", tc.at-10,
					map[bool]string{true: "engaged", false: "did not engage"}[got.HasAttackTarget])
			}
			if !tc.engaging && (!got.HasTarget || got.TargetX != 20) {
				t.Errorf("with nothing in the block the patroller holds destination (%d,%d) present %v, "+
					"want its tail waypoint (20,10)", got.TargetX, got.TargetY, got.HasTarget)
			}
		})
	}
}

// ------------------------------------------------------------------ the post

// TestPatrolCommandAnchorsThePost is `AI-PATROL-018`'s `L00415`: the setter
// that builds the ring writes the post to the actor's current cell in the
// same body. Both of this build's patrol setters are asked — the script's
// sub-command 14 and the player's own commandPatrol — because the law has
// one setter and this build has two.
//
// It is the positive assertion TestOnlyTheTwoStancesWriteAPost
// (stanceanchor_test.go) gave up when Patrol left its negative set.
func TestPatrolCommandAnchorsThePost(t *testing.T) {
	t.Parallel()

	t.Run("the script's sub-command 14", func(t *testing.T) {
		t.Parallel()
		a := engFighter(1, 2, 5, 5)
		a.Group = 1
		w := engWorld(t, engRel(t), a)
		i := indexOfEntity(w.entities, 1)
		w.entities[i].X, w.entities[i].Y = 12, 13 // moved off the constructed post
		w.runInstant(ScriptInstant{Op: ScriptInstantGroupOrder, Group: 1, HasGroup: true,
			Args: [scriptParams]int32{subCommandPatrol, 20, 13}})
		if got := entityAt(t, w, 1); got.PostX != 12 || got.PostY != 13 {
			t.Errorf("post is (%d,%d), want the cell the member stands on (12,13)", got.PostX, got.PostY)
		}
	})

	t.Run("the player's commandPatrol", func(t *testing.T) {
		t.Parallel()
		a := engFighter(1, 2, 5, 5)
		a.Group = 1
		w := engWorld(t, engRel(t), a)
		i := indexOfEntity(w.entities, 1)
		w.entities[i].X, w.entities[i].Y = 12, 13
		w.commandPatrol([]int{i}, cell{x: 20, y: 13})
		if got := entityAt(t, w, 1); got.PostX != 12 || got.PostY != 13 {
			t.Errorf("post is (%d,%d), want the cell the member stands on (12,13)", got.PostX, got.PostY)
		}
	})
}

// TestGuardAnchorsAZeroPostOnItsOwnCell is the arm's first clause,
// `AI-POST-042`/`AI-GUARD-012` at `L00327`: `ord+0x00` is a packed cell and
// a zero one is set to the actor's current cell the first time guard runs.
func TestGuardAnchorsAZeroPostOnItsOwnCell(t *testing.T) {
	t.Parallel()

	w := pxGuard(t, 10, 10)
	i := indexOfEntity(w.entities, 1)
	w.entities[i].PostX, w.entities[i].PostY = 0, 0

	w.actorPass()

	if got := entityAt(t, w, 1); got.PostX != 10 || got.PostY != 10 {
		t.Errorf("post is (%d,%d) after the first guard pass, want the actor's own cell (10,10)",
			got.PostX, got.PostY)
	}
}

// TestThePostFollowsAPatrollerAndStopsWhereGuardInterruptsIt is the
// re-anchor, `AI-PATROL-018`'s latch read through armPatrol's derived form.
// Two facts, and the second is the one that matters:
//
//   - while the patrol tail keeps running, the post is the cell the actor
//     stands on at each entry, so the block travels with it and the leash
//     never walks it back (`AI-POST-042`: "patrol is guard with a moving
//     post");
//   - the entry AFTER guard engaged does not re-anchor, so the post stays
//     where the patrol broke off and leashes the pursuit to it
//     (`AI-BREAK-041`).
func TestThePostFollowsAPatrollerAndStopsWhereGuardInterruptsIt(t *testing.T) {
	t.Parallel()

	w := pxPatroller(t, 10, 10, 20, 10)
	i := indexOfEntity(w.entities, 1)

	w.actorPass()
	// The mover is not run here; the actor is placed where a tick of walking
	// would have put it, which is what the next entry re-anchors on.
	w.entities[i].X = 11
	w.actorPass()
	if got := entityAt(t, w, 1); got.PostX != 11 || got.PostY != 10 {
		t.Fatalf("post is (%d,%d) after a patrolling step, want the actor's cell (11,10) — "+
			"the post did not follow the patroller", got.PostX, got.PostY)
	}

	// Now a hostile arrives inside the block. Guard engages, the tail does
	// not run, and the post must stop moving with the actor.
	w.entities = append(w.entities, pxHostile(2, 13, 10))
	w.routes = append(w.routes, nil)
	w.actorPass()
	if got := entityAt(t, w, 1); !got.HasAttackTarget {
		t.Fatalf("the patroller did not engage the hostile three cells away")
	}
	i = indexOfEntity(w.entities, 1)
	w.entities[i].X = 12 // the pursuit carries it one cell on
	w.actorPass()
	if got := entityAt(t, w, 1); got.PostX != 11 || got.PostY != 10 {
		t.Errorf("post is (%d,%d) after a pursuing step, want it left at the break-off cell (11,10) — "+
			"a post that follows a pursuit is a leash that never breaks", got.PostX, got.PostY)
	}
}

// ------------------------------------------------------------------ the leash

// TestAPatrollersPursuitBreaksOffWhenTheTargetLeavesTheBlock is
// `AI-BREAK-041`'s headline, and the consequence it says a consumer must not
// invert: how far the actor has chased is not measured. The input is the
// distance from the POST to the TARGET, so the same pursuit is kept at one
// distance and dropped at the next.
func TestAPatrollersPursuitBreaksOffWhenTheTargetLeavesTheBlock(t *testing.T) {
	t.Parallel()

	w := pxPatroller(t, 10, 10, 20, 10, pxHostile(2, 13, 10))
	w.actorPass()
	if got := entityAt(t, w, 1); !got.HasAttackTarget {
		t.Fatalf("fixture: the patroller did not engage at all")
	}

	i := indexOfEntity(w.entities, 1)
	vi := indexOfEntity(w.entities, 2)
	w.entities[i].X = 12  // it has chased two cells
	w.entities[vi].X = 17 // and the target has passed 5 from the post (10,10)
	w.actorPass()

	got := entityAt(t, w, 1)
	if got.HasAttackTarget {
		t.Fatalf("the patroller still holds victim %d, whose distance from the post is 7", got.AttackTarget)
	}
	if !got.HasTarget || got.TargetX != 10 || got.TargetY != 10 {
		t.Errorf("the patroller holds destination (%d,%d) present %v, want the post (10,10) — "+
			"an empty block away from home is the walk back (AI-GUARD-012, L00334)",
			got.TargetX, got.TargetY, got.HasTarget)
	}
}

func TestAGuardAtHomeDropsAVictimThatLeftItsBlock(t *testing.T) {
	t.Parallel()

	w := pxGuard(t, 10, 10, pxHostile(2, 20, 10))
	i := indexOfEntity(w.entities, 1)
	w.entities[i].AttackTarget, w.entities[i].HasAttackTarget = 2, true

	w.actorPass()

	if got := entityAt(t, w, 1); got.HasAttackTarget {
		t.Errorf("a guard standing on its post kept victim %d, ten cells away", got.AttackTarget)
	}
}

// ------------------------------------------------------------------ the ring

// TestTheRingStillAdvancesWhenGuardFindsNothing is the whole reason the gate
// is a gate and not a return: with an empty block and nothing to acquire,
// guard leaves the order idle and the tail runs — the flip on arrival and
// the fresh destination, exactly as they ran before this story.
func TestTheRingStillAdvancesWhenGuardFindsNothing(t *testing.T) {
	t.Parallel()

	w := pxPatroller(t, 10, 10, 20, 10)
	i := indexOfEntity(w.entities, 1)
	w.entities[i].X, w.entities[i].Y = 20, 10 // arrived at the tail

	w.actorPass()

	got := entityAt(t, w, 1)
	if got.PatrolLeg != patrolLegHead {
		t.Fatalf("the leg is %d after arriving at the tail, want head (%d)", got.PatrolLeg, patrolLegHead)
	}
	if !got.HasTarget || got.TargetX != 10 || got.TargetY != 10 {
		t.Errorf("the destination is (%d,%d) present %v, want the head waypoint (10,10)",
			got.TargetX, got.TargetY, got.HasTarget)
	}
}

// TestAPatrollerKeepsItsTurnWhileWalkingTheSameLeg pins the one thing the
// arm's re-ordering could silently have broken. The turn cancel used to run
// at the top of the arm, before the order was cleared; it now runs at the
// advance, from three fields captured before that clear. The decision is
// unchanged — an unchanged destination keeps the turn — and a turn cleared
// on every tick is a turn that never completes.
func TestAPatrollerKeepsItsTurnWhileWalkingTheSameLeg(t *testing.T) {
	t.Parallel()

	w := pxPatroller(t, 10, 10, 20, 10)
	i := indexOfEntity(w.entities, 1)
	w.actorPass() // gives it the tail as a destination
	w.entities[i].TurnRemaining, w.entities[i].TurnTotal = 2, 4
	w.actorPass()

	if got := entityAt(t, w, 1); got.TurnRemaining != 2 {
		t.Errorf("the turn remaining is %d after a pass that changed no destination, want 2 kept",
			got.TurnRemaining)
	}
}

// ------------------------------------------------------------- the dispatch

// pxGuard is a guard-state actor under a group whose order is 0 — the
// one population the new dispatch case runs for.
func pxGuard(t *testing.T, x, y int32, others ...Entity) *World {
	t.Helper()
	a := engFighter(1, 2, x, y)
	a.Group = 1
	rel := engRel(t, [3]uint32{2, 3, relationHostile}, [3]uint32{3, 2, relationHostile})
	w, err := NewRelatedWorld(1, pxBounds, ModeCanonical, Terrain{}, append([]Entity{a}, others...), nil, rel)
	if err != nil {
		t.Fatalf("NewRelatedWorld: %v", err)
	}
	for gi := range w.groups {
		if w.groups[gi].owner == 2 && w.groups[gi].group == 1 {
			w.groups[gi].order = orderNone
		}
	}
	return w
}

// TestTheGuardCaseRunsOnlyForAGroupUnderOrderZero is the gate `AI-ORDER-010`
// and `AI-POST-095` put on the whole per-actor machine. Guard is the
// constructor's default for every entity, so without it the arm would decide
// the entire map a second time, over the group layer's own guard and
// stand-ground arms.
func TestTheGuardCaseRunsOnlyForAGroupUnderOrderZero(t *testing.T) {
	t.Parallel()

	t.Run("order 0: the arm runs", func(t *testing.T) {
		t.Parallel()
		w := pxGuard(t, 10, 10, pxHostile(2, 13, 10))
		w.actorPass()
		if got := entityAt(t, w, 1); !got.HasAttackTarget {
			t.Errorf("a guard under group order 0 did not engage a hostile three cells from its post")
		}
	})

	t.Run("the group's own order: the arm does not", func(t *testing.T) {
		t.Parallel()
		w := pxGuard(t, 10, 10, pxHostile(2, 13, 10))
		for gi := range w.groups {
			w.groups[gi].order = orderGuard
		}
		i := indexOfEntity(w.entities, 1)
		before := w.entities[i]
		w.actorPass()
		if w.entities[i] != before {
			t.Errorf("a guard under group order 1 was decided by the actor layer as well:\n before %+v\n after  %+v",
				before, w.entities[i])
		}
	})
}

// TestAnOffMapGuardIsLeftInEveryField is the first of the arm's three
// refusals. They are this build's cast-ownership rule rather than a clause of
// the law's arm, and they are written where armAcquire writes its own.
func TestAnOffMapGuardIsLeftInEveryField(t *testing.T) {
	t.Parallel()

	w := pxGuard(t, 10, 10, pxHostile(2, 11, 10))
	i := indexOfEntity(w.entities, 1)
	w.entities[i].OffMap = true
	before := w.entities[i]

	w.actorPass()

	if w.entities[i] != before {
		t.Errorf("an off-map guard was changed by its own arm:\n before %+v\n after  %+v",
			before, w.entities[i])
	}
}

// ------------------------------------------------------- the mission-80 shape

// TestTheMission80TrollLeavesPatrolWhenAPartyUnitComesInside reproduces the
// shape the owner reported, with the map's own numbers stated rather than
// loaded: `80.alm` `Units[68]`, `UnitID = 112`, `GroupID = 3`, owner slot 4
// "Monsters", standing at (100,37), and the type-7 script's `"Start 2"`
// trigger issuing `Group Command : Patrol 3` toward (110,37). Slot 1 is the
// party and the roster's own diplomacy words put slots 1 and 4 mutually
// hostile in both directions.
//
// NOTHING HERE READS AN INSTALL. The coordinates, ids and slots are the
// fixture's own, and what the test measures is this package's behaviour on
// them: the troll patrols while the party is away, and stops patrolling and
// fights the moment a party unit stands inside its block.
func TestTheMission80TrollLeavesPatrolWhenAPartyUnitComesInside(t *testing.T) {
	t.Parallel()

	const monsters, party = uint32(4), uint32(1)
	troll := engFighter(112, monsters, 100, 37)
	troll.Group = 3
	hero := engFighter(1, party, 118, 37) // eighteen cells off, well outside the block
	rel := engRel(t, [3]uint32{party, monsters, relationHostile}, [3]uint32{monsters, party, relationHostile})
	w, err := NewRelatedWorld(1, pxBounds, ModeCanonical, Terrain{}, []Entity{hero, troll}, nil, rel)
	if err != nil {
		t.Fatalf("NewRelatedWorld: %v", err)
	}
	w.runInstant(ScriptInstant{Op: ScriptInstantGroupOrder, Group: 3, HasGroup: true,
		Args: [scriptParams]int32{subCommandPatrol, 110, 37}})

	w.actorPass()
	got := entityAt(t, w, 112)
	if got.HasAttackTarget || !got.HasTarget || got.TargetX != 110 {
		t.Fatalf("with the party eighteen cells away the troll holds victim %v and destination (%d,%d), "+
			"want no victim and its waypoint (110,37)", got.HasAttackTarget, got.TargetX, got.TargetY)
	}

	// The party walks up to it. Four cells is inside the block, and the
	// troll's reach is 1, so nothing but the block can engage here.
	hi := indexOfEntity(w.entities, 1)
	w.entities[hi].X = 104
	w.actorPass()

	got = entityAt(t, w, 112)
	if !got.HasAttackTarget || got.AttackTarget != 1 {
		t.Fatalf("the troll holds victim %d (present %v), want the party unit 1 — this is the defect "+
			"the owner reported: the troll beside the party's start walks its patrol and never attacks",
			got.AttackTarget, got.HasAttackTarget)
	}
	if got.HasTarget {
		t.Errorf("the troll also holds destination (%d,%d): it is still walking its ring while fighting",
			got.TargetX, got.TargetY)
	}
}

// TestTheSavedGuardScansTheSameBlock witnesses the shared body from the other
// side. `savedGuard` (savedgroupsai.go) is the original-runtime transcription
// of the same routine, and after this story its block scan IS `postEngage`.
// The witness is behavioural, not structural: the saved arm engages a hostile
// five cells from the post, reports it in `Raw[8]` as the engaged code 5, and
// refuses the same hostile one cell further out.
func TestTheSavedGuardScansTheSameBlock(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		at       int32
		engaging bool
	}{
		{"five cells from the post", 15, true},
		{"six cells from the post", 16, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := pxGuard(t, 10, 10, pxHostile(2, tc.at, 10))
			o := &SavedActorOrder{Entity: 1, State: 0xb}
			w.savedGuard(0, o)

			got := entityAt(t, w, 1)
			if got.HasAttackTarget != tc.engaging {
				t.Fatalf("the saved guard holds victim %d (present %v), want engaging=%v",
					got.AttackTarget, got.HasAttackTarget, tc.engaging)
			}
			if tc.engaging && o.Raw[8] != 5 {
				t.Errorf("the saved order reports state %d, want the engaged code 5", o.Raw[8])
			}
			if !tc.engaging && o.Raw[8] == 5 {
				t.Error("the saved order reports the engaged code 5 with no victim")
			}
			if o.Raw[0] != 10 || o.Raw[1] != 10 {
				t.Errorf("the saved post is (%d,%d), want the actor's own cell (10,10)", o.Raw[0], o.Raw[1])
			}
		})
	}
}

// TestAScriptAttackSurvivesTheNextActorPass is the case the paired release
// tests caught and the reason actor state 3 has a writer again. A member
// left in guard instead keeps an order the guard arm's own leash then breaks
// off on the very next pass, because the victim it holds is not a hostile
// inside its block.
func TestAScriptAttackSurvivesTheNextActorPass(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t),
		laFighter(1, 2, 7, 5, 5), laFighter(2, 2, 7, 6, 5), laFighter(3, 2, 7, 7, 5))
	w.cmdGroupOrder(ScriptInstant{Op: ScriptInstantGroupOrder, Group: 7, HasGroup: true,
		Unit: 1, HasUnit: true, Args: [scriptParams]int32{subCommandAttack}})

	for _, id := range []EntityID{2, 3} {
		if e := laEnt(t, w, id); e.ActorState != actorStateEngage {
			t.Fatalf("member %d is in actor state %d, want engage (%d)", id, e.ActorState, actorStateEngage)
		}
	}

	w.actorPass()

	for _, id := range []EntityID{2, 3} {
		e := laEnt(t, w, id)
		if !e.HasAttackTarget || e.AttackTarget != 1 {
			t.Errorf("member %d holds victim %d (present %v) after one actor pass, want the named unit 1 — "+
				"its order was broken off by an arm that does not own it", id, e.AttackTarget, e.HasAttackTarget)
		}
	}
}

// TestAPatrollerEndsAnUnownedTurnUnlessItIsCasting is the other half of the
// turn decision. A patroller that holds no destination of its own takes a
// fresh one from the ring, and that replacement ends the movement-owned turn
// — with one exception, a pending book cast, which owns the actor's facing
// for its whole wind-up.
//
// THE DECISION IS UNCHANGED BY THIS STORY, only its position: it used to run
// at the top of the arm and now runs at the advance, from three fields
// captured before the arm clears the order. A patroller arriving at a
// waypoint keeps its turn on both sides of the story, because the captured
// leg is the one it arrived at rather than the one the flip issues.
func TestAPatrollerEndsAnUnownedTurnUnlessItIsCasting(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		casting bool
		want    uint8
	}{
		{"walking", false, 0},
		{"casting", true, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := pxPatroller(t, 10, 10, 20, 10)
			i := indexOfEntity(w.entities, 1)
			w.entities[i].HasTarget = false
			w.entities[i].TurnRemaining, w.entities[i].TurnTotal = 2, 4
			if tc.casting {
				w.bookCasts = append(w.bookCasts, bookCast{Caster: 1, Spell: 1, Phase: bookCharging, Remaining: 3})
			}

			w.actorPass()

			got := entityAt(t, w, 1)
			if got.TurnRemaining != tc.want {
				t.Errorf("the turn remaining is %d after the ring issued a destination the actor did "+
					"not already hold, want %d", got.TurnRemaining, tc.want)
			}
			if !got.HasTarget || got.TargetX != 20 || got.TargetY != 10 {
				t.Fatalf("fixture: the destination is (%d,%d) present %v, want the tail waypoint "+
					"(20,10) — no destination was issued, so this tested nothing",
					got.TargetX, got.TargetY, got.HasTarget)
			}
		})
	}
}

// TestAPatrollersWalkHomeSurvivesTheNextPass runs ONE PASS FURTHER than the
// test above, which is where the leash is actually kept or lost (pass 1
// review, F-1). Breaking a pursuit off is worth nothing if the entry that
// follows it re-anchors the post to wherever the chase ended: the block
// travels with the actor, the same target is inside it again, and the walk
// home is dropped before a single step of it is taken.
//
// `AI-PATROL-018`'s latch is set only by the ring's own advance, so an entry
// that walked home leaves it unset and the next entry must NOT re-anchor.
// This is the assertion the derived latch has to earn.
func TestAPatrollersWalkHomeSurvivesTheNextPass(t *testing.T) {
	t.Parallel()

	w := pxPatroller(t, 10, 10, 20, 10, pxHostile(2, 13, 10))
	w.actorPass() // pass 1: engage
	i := indexOfEntity(w.entities, 1)
	vi := indexOfEntity(w.entities, 2)
	w.entities[i].X = 12  // chased two cells
	w.entities[vi].X = 17 // the target has passed 5 from the post
	w.actorPass()         // pass 2: break off, walk home
	if got := entityAt(t, w, 1); got.HasAttackTarget || !got.HasTarget {
		t.Fatalf("fixture: pass 2 did not produce a walk home (victim %v, destination %v)",
			got.HasAttackTarget, got.HasTarget)
	}

	w.actorPass() // pass 3: the one this test exists for

	got := entityAt(t, w, 1)
	if got.PostX != 10 || got.PostY != 10 {
		t.Errorf("the post moved to (%d,%d) on the entry after a break-off, want the post it broke "+
			"off from (10,10) — the leash is post-relative and this makes it actor-relative",
			got.PostX, got.PostY)
	}
	if got.HasAttackTarget {
		t.Errorf("the patroller re-acquired victim %d from the cell it chased to; its distance from "+
			"the standing post is 7", got.AttackTarget)
	}
	if !got.HasTarget || got.TargetX != 10 || got.TargetY != 10 {
		t.Errorf("the patroller holds destination (%d,%d) present %v, want the walk home to (10,10) "+
			"re-issued", got.TargetX, got.TargetY, got.HasTarget)
	}
}

// TestAKitedPatrollerStaysOnItsLeash is the same defect measured as a bound
// rather than as a single entry, and it is the discriminating case: a target
// that keeps withdrawing must not be able to tow a patroller across the map.
// The hostile retreats one cell every second pass; the patroller closes one
// cell per pass, so an actor-relative leash follows it forever while a
// post-relative one gives up within a few passes and walks back.
//
// The bound asserted is the ring itself: 120 passes must leave the patroller
// no further east than its own tail waypoint.
func TestAKitedPatrollerStaysOnItsLeash(t *testing.T) {
	t.Parallel()

	const passes = 120
	w := pxPatroller(t, 10, 10, 20, 10, pxHostile(2, 13, 10))
	i := indexOfEntity(w.entities, 1)
	vi := indexOfEntity(w.entities, 2)

	engaged, east := 0, int32(0)
	for p := 0; p < passes; p++ {
		w.actorPass()
		e := w.entities[i]
		if e.HasAttackTarget {
			engaged++
			if e.X < w.entities[vi].X { // close one cell per pass
				w.entities[i].X++
			}
		}
		if p%2 == 1 && w.entities[vi].X < pxBounds.Width-1 {
			w.entities[vi].X++ // and the target keeps withdrawing
		}
		if w.entities[i].X > east {
			east = w.entities[i].X
		}
	}

	got := entityAt(t, w, 1)
	t.Logf("after %d passes: actor=(%d,%d) post=(%d,%d) victim-held=%v engaged=%d/%d easternmost=%d",
		passes, got.X, got.Y, got.PostX, got.PostY, got.HasAttackTarget, engaged, passes, east)
	if east > 20 {
		t.Errorf("the patroller was towed to x=%d over %d passes, past its own ring tail at x=20",
			east, passes)
	}
	if got.PostX > 20 {
		t.Errorf("the post ended at (%d,%d), east of the ring the patrol was given",
			got.PostX, got.PostY)
	}
	if got.HasAttackTarget {
		t.Errorf("the patroller still holds victim %d after %d passes of withdrawal",
			got.AttackTarget, passes)
	}
	if engaged > passes/2 {
		t.Errorf("the patroller was engaged on %d of %d passes; a withdrawing target leaves the "+
			"block and stays out of it", engaged, passes)
	}
}
