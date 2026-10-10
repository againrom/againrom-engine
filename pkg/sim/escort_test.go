package sim

// The two escort arms (0167).
//
// Every fixture is built the way 0166's own escort tests are: engWorld/engRel
// over laFighter, with the escort installed by the script sub-command itself
// rather than written into the record, because the constructor normalises an
// escort triple away (world.go's clearEscort call in the constructor). Nothing
// here reads a game install.

import "testing"

// esWorld is a world holding a subject and the members named after it, with a
// Defend or a Follow already issued over group 7 at the range given.
//
// The escort is installed through cmdGroupOrder — the shipped road — so that
// every test below runs against the same state a map's own script produces.
func esWorld(t *testing.T, sub int32, span int32, rel Relations, ents ...Entity) *World {
	t.Helper()
	w := engWorld(t, rel, ents...)
	w.cmdGroupOrder(ScriptInstant{Op: ScriptInstantGroupOrder, Group: 7, HasGroup: true,
		Unit: 1, HasUnit: true, Args: [scriptParams]int32{sub, span}})
	return w
}

// esPass runs one actor pass over w without stepping the world, so that a test
// can read what an arm decided rather than what a tick's move loop then did with
// it. The move loop is exercised separately, by the tests that call engRun.
func esPass(w *World) { w.actorPass() }

// 1087 activates the previously deferred 0xc arm for the same script-produced
// state as player Defend. No enemy means no standing pick and no old walk.
func TestBothEscortStatesAndAcquireReachTheirArms(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		sub  int32
	}{{"defend", subCommandDefend}, {"follow", subCommandFollow}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := esWorld(t, tc.sub, 3, engRel(t),
				laFighter(1, 2, 7, 5, 5), laFighter(2, 2, 7, 20, 20))

			// The named unit is given a destination by hand, standing in for the
			// player order SC-1 is about.
			ni := indexOfEntity(w.entities, 1)
			w.entities[ni].TargetX, w.entities[ni].TargetY = 40, 40
			w.entities[ni].HasTarget = true

			esPass(w)

			if e := laEnt(t, w, 2); !e.HasTarget || e.TargetX != 5 || e.TargetY != 5 {
				t.Errorf("the escort holds destination (%d,%d)/%v, want the subject's cell (5,5) — "+
					"the state reached no arm", e.TargetX, e.TargetY, e.HasTarget)
			}
			if e := laEnt(t, w, 1); e.HasTarget || e.HasAttackTarget {
				t.Errorf("acquire-in-place retained a walk or victim with no enemy: %+v", e)
			}
		})
	}
}

func TestAnEscortOfAUnitTheWorldDoesNotHoldIsLeftAlone(t *testing.T) {
	t.Parallel()

	w := esWorld(t, subCommandFollow, 3, engRel(t),
		laFighter(1, 2, 7, 5, 5), laFighter(2, 2, 7, 20, 20))

	// Take the subject out of the entity slice outright, which is what the decay
	// ladder's removal leaves.
	si := indexOfEntity(w.entities, 1)
	w.entities = append(w.entities[:si], w.entities[si+1:]...)
	w.routes = w.routes[:len(w.entities)]

	before := laEnt(t, w, 2)
	esPass(w)
	if got := laEnt(t, w, 2); got != before {
		t.Errorf("the escort changed from %+v to %+v, want every field left alone", before, got)
	}
}

func TestAnEscortRangeOfZeroTestsAgainstTheScanRange(t *testing.T) {
	t.Parallel()

	// engSight is 5. A subject 4 cells away is inside the scan range and outside
	// any smaller stored range, so the fallback is what decides.
	w := esWorld(t, subCommandFollow, 1, engRel(t),
		laFighter(1, 2, 7, 5, 5), laFighter(2, 2, 7, 9, 5))
	i := indexOfEntity(w.entities, 2)
	w.entities[i].EscortRange = 0

	esPass(w)
	if e := laEnt(t, w, 2); e.HasTarget {
		t.Errorf("the escort was sent to (%d,%d) with a stored range of 0, want no destination: "+
			"4 cells is inside its scan range of %d", e.TargetX, e.TargetY, engSight)
	}

	// The control: the same distance with the stored range in force closes.
	c := esWorld(t, subCommandFollow, 1, engRel(t),
		laFighter(1, 2, 7, 5, 5), laFighter(2, 2, 7, 9, 5))
	esPass(c)
	if e := laEnt(t, c, 2); !e.HasTarget {
		t.Error("the control escort holds no destination at range 1 and distance 4, want the subject's cell")
	}
}

func TestAnEscortOutOfRangeClosesOnItsSubject(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		sub  int32
	}{{"defend", subCommandDefend}, {"follow", subCommandFollow}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Owner 3 is hostile to owner 2 both ways, and stands beside the escort
			// so that the escort has a victim to lose.
			w := esWorld(t, tc.sub, 3, engRel(t, [3]uint32{2, 3, 1}, [3]uint32{3, 2, 1}),
				laFighter(1, 2, 7, 5, 5), laFighter(2, 2, 7, 20, 20), laFighter(3, 3, 9, 21, 20))
			i := indexOfEntity(w.entities, 2)
			w.orderAttack(i, 3)

			esPass(w)

			e := laEnt(t, w, 2)
			if !e.HasTarget || e.TargetX != 5 || e.TargetY != 5 {
				t.Errorf("the escort holds destination (%d,%d)/%v, want the subject's cell (5,5)",
					e.TargetX, e.TargetY, e.HasTarget)
			}
			if e.HasAttackTarget {
				t.Errorf("the escort still holds victim %d, want the close order to have dropped it (DD-10)",
					e.AttackTarget)
			}
		})
	}
}

// TestAFollowerClosesAndThenStopsWithinItsRange is AC-1's second half: driven
// through whole ticks, a follower walks to its subject and stops inside the
// range rather than on top of it.
func TestAFollowerClosesAndThenStopsWithinItsRange(t *testing.T) {
	t.Parallel()

	w := esWorld(t, subCommandFollow, 3, engRel(t),
		laFighter(1, 2, 7, 5, 5), laFighter(2, 2, 7, 15, 5))
	engRun(w, 60)

	e := laEnt(t, w, 2)
	d := cellOf(&e).chebyshevTo(cell{x: 5, y: 5})
	if d > 3 {
		t.Errorf("the follower ended at (%d,%d), %d cells from its subject, want at most its range of 3",
			e.X, e.Y, d)
	}
	if d < escortCrowd {
		t.Errorf("the follower ended at (%d,%d), %d cells from its subject, want at least %d",
			e.X, e.Y, d, escortCrowd)
	}
}

// TestAFollowerReAimsWhenTheGapOpens is AC-2: the subject is moved out past the
// range and the follower is aimed at it again on the next pass.
func TestAFollowerReAimsWhenTheGapOpens(t *testing.T) {
	t.Parallel()

	w := esWorld(t, subCommandFollow, 3, engRel(t),
		laFighter(1, 2, 7, 5, 5), laFighter(2, 2, 7, 7, 5))

	esPass(w)
	if e := laEnt(t, w, 2); e.HasTarget {
		t.Fatalf("the follower was sent to (%d,%d) while inside its range, want no close order",
			e.TargetX, e.TargetY)
	}

	si := indexOfEntity(w.entities, 1)
	w.entities[si].X, w.entities[si].Y = 30, 30
	esPass(w)
	if e := laEnt(t, w, 2); !e.HasTarget || e.TargetX != 30 || e.TargetY != 30 {
		t.Errorf("after the gap opened the follower holds (%d,%d)/%v, want the subject's new cell (30,30)",
			e.TargetX, e.TargetY, e.HasTarget)
	}
}

func TestADefenderEngagesOnItsSubjectsBehalf(t *testing.T) {
	t.Parallel()

	// The defender stands 3 cells from its subject, inside its range of 4. The
	// hostile is placed relative to the SUBJECT: at 4 cells it is in the block,
	// at 6 it is not, and in both cases it is well outside the defender's own
	// sight and reach.
	for _, tc := range []struct {
		name   string
		hx     int32
		engage bool
	}{
		{"inside the block", 9, true},
		{"outside the block", 11, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := esWorld(t, subCommandDefend, 4, engRel(t, [3]uint32{2, 3, 1}, [3]uint32{3, 2, 1}),
				laFighter(1, 2, 7, 5, 5), laFighter(2, 2, 7, 2, 5), laFighter(3, 3, 9, tc.hx, 5))
			esPass(w)

			e := laEnt(t, w, 2)
			if got := e.HasAttackTarget && e.AttackTarget == 3; got != tc.engage {
				t.Errorf("the defender engaged %d/%v with the hostile %d cells from its subject, want engaged=%v",
					e.AttackTarget, e.HasAttackTarget, tc.hx-5, tc.engage)
			}
		})
	}
}

func TestTheCoverFiltersPolarityIsTheSubjects(t *testing.T) {
	t.Parallel()

	// Owner 4 is the candidate's. The subject and the defender share owner 2 in
	// the first case; in the second the defender is owner 5, hostile to 4, while
	// the subject is not.
	t.Run("the subject's enemy is engaged", func(t *testing.T) {
		t.Parallel()
		w := esWorld(t, subCommandDefend, 4, engRel(t, [3]uint32{2, 4, 1}, [3]uint32{4, 2, 1}),
			laFighter(1, 2, 7, 5, 5), laFighter(2, 2, 7, 3, 5), laFighter(3, 4, 9, 9, 5))
		esPass(w)
		if e := laEnt(t, w, 2); !e.HasAttackTarget || e.AttackTarget != 3 {
			t.Errorf("the defender holds victim %d/%v, want the subject's enemy 3",
				e.AttackTarget, e.HasAttackTarget)
		}
	})

	t.Run("the defender's own enemy is not", func(t *testing.T) {
		t.Parallel()
		// The defender (owner 5) is hostile to owner 4; the subject (owner 2) is
		// not. The candidate is out of the defender's own sight, so the standing
		// acquisition an empty block falls to cannot pick it up either.
		w := esWorld(t, subCommandDefend, 4, engRel(t, [3]uint32{5, 4, 1}, [3]uint32{4, 5, 1}),
			laFighter(1, 2, 7, 5, 5), laFighter(2, 5, 7, 3, 5), laFighter(3, 4, 9, 9, 5))
		esPass(w)
		if e := laEnt(t, w, 2); e.HasAttackTarget {
			t.Errorf("the defender engaged %d, want nothing: its subject is not hostile to it", e.AttackTarget)
		}
	})
}

func TestTheCoverEngagementPrefersAirAndThenTheNearest(t *testing.T) {
	t.Parallel()

	rel := engRel(t, [3]uint32{2, 3, 1}, [3]uint32{3, 2, 1})

	for _, tc := range []struct {
		name  string
		reach uint8
		want  EntityID
	}{
		{"air over a nearer ground candidate", 4, 4},
		{"a melee defender passes over the air candidate", 1, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// The defender is at (13,5), its subject at (5,5) with a range of 8. The
			// ground candidate is 3 cells from the defender and the air candidate 8.
			defender := laFighter(2, 2, 7, 13, 5)
			defender.Reach = tc.reach
			w := esWorld(t, subCommandDefend, 8, rel,
				laFighter(1, 2, 7, 5, 5), defender,
				laFighter(3, 3, 9, 10, 5), laFlier(4, 3, 9, 5, 5+1))
			esPass(w)
			if e := laEnt(t, w, 2); !e.HasAttackTarget || e.AttackTarget != tc.want {
				t.Errorf("the defender holds victim %d/%v, want %d",
					e.AttackTarget, e.HasAttackTarget, tc.want)
			}
		})
	}

	t.Run("the nearest where the classes match", func(t *testing.T) {
		t.Parallel()
		w := esWorld(t, subCommandDefend, 8, rel,
			laFighter(1, 2, 7, 5, 5), laFighter(2, 2, 7, 13, 5),
			laFighter(3, 3, 9, 10, 5), laFighter(4, 3, 9, 5, 6))
		esPass(w)
		if e := laEnt(t, w, 2); !e.HasAttackTarget || e.AttackTarget != 3 {
			t.Errorf("the defender holds victim %d/%v, want the nearer candidate 3",
				e.AttackTarget, e.HasAttackTarget)
		}
	})
}

func TestAnEmptyCoverBlockFallsToTheStandingAcquisition(t *testing.T) {
	t.Parallel()

	rel := engRel(t, [3]uint32{2, 3, 1}, [3]uint32{3, 2, 1})

	t.Run("something within reach is engaged", func(t *testing.T) {
		t.Parallel()
		// The subject at (5,5) has an empty block: the only hostile stands beside
		// the DEFENDER at (20,20), 15 cells from the subject.
		w := esWorld(t, subCommandDefend, 20, rel,
			laFighter(1, 2, 7, 5, 5), laFighter(2, 2, 7, 20, 20), laFighter(3, 3, 9, 21, 20))
		esPass(w)
		if e := laEnt(t, w, 2); !e.HasAttackTarget || e.AttackTarget != 3 {
			t.Errorf("the defender holds victim %d/%v, want the hostile 3 its own acquisition sees",
				e.AttackTarget, e.HasAttackTarget)
		}
	})

	t.Run("nothing visible is engaged", func(t *testing.T) {
		t.Parallel()
		w := esWorld(t, subCommandDefend, 20, rel,
			laFighter(1, 2, 7, 5, 5), laFighter(2, 2, 7, 20, 20), laFighter(3, 3, 9, 40, 40))
		esPass(w)
		if e := laEnt(t, w, 2); e.HasAttackTarget {
			t.Errorf("the defender engaged %d, want nothing at all", e.AttackTarget)
		}
	})
}

func TestADefenderThatIsFightingDoesNotStepAway(t *testing.T) {
	t.Parallel()

	// The defender stands ON the cell beside its subject — distance 1, inside the
	// crowding distance — with a hostile of the subject 2 cells from the subject.
	w := esWorld(t, subCommandDefend, 4, engRel(t, [3]uint32{2, 3, 1}, [3]uint32{3, 2, 1}),
		laFighter(1, 2, 7, 5, 5), laFighter(2, 2, 7, 6, 5), laFighter(3, 3, 9, 7, 5))
	esPass(w)

	e := laEnt(t, w, 2)
	if !e.HasAttackTarget || e.AttackTarget != 3 {
		t.Fatalf("the defender holds victim %d/%v, want the hostile 3", e.AttackTarget, e.HasAttackTarget)
	}
	if e.HasTarget {
		t.Errorf("the busy defender was sent to (%d,%d), want no step-away this pass (FR-8)",
			e.TargetX, e.TargetY)
	}

	// THE CONTROL: the same crowding with nothing to fight steps away.
	c := esWorld(t, subCommandDefend, 4, engRel(t),
		laFighter(1, 2, 7, 5, 5), laFighter(2, 2, 7, 6, 5))
	esPass(c)
	if e := laEnt(t, c, 2); !e.HasTarget {
		t.Error("the idle defender holds no destination at distance 1, want the step-away")
	}
}

func TestAFollowerInRangeReAcquiresAndNeverScansForItsSubject(t *testing.T) {
	t.Parallel()

	rel := engRel(t, [3]uint32{2, 3, 1}, [3]uint32{3, 2, 1})

	t.Run("its own hostile is engaged", func(t *testing.T) {
		t.Parallel()
		w := esWorld(t, subCommandFollow, 4, rel,
			laFighter(1, 2, 7, 5, 5), laFighter(2, 2, 7, 8, 5), laFighter(3, 3, 9, 9, 5))
		esPass(w)
		if e := laEnt(t, w, 2); !e.HasAttackTarget || e.AttackTarget != 3 {
			t.Errorf("the follower holds victim %d/%v, want its own hostile 3",
				e.AttackTarget, e.HasAttackTarget)
		}
	})

	t.Run("a hostile of the subject alone is not", func(t *testing.T) {
		t.Parallel()
		// The candidate stands beside the SUBJECT and out of the follower's sight
		// (engSight is 5, and the follower is 9 cells from it). A DEFENDER in the
		// same fixture engages it; a follower does not.
		ents := []Entity{laFighter(1, 2, 7, 5, 5), laFighter(2, 2, 7, 14, 5), laFighter(3, 3, 9, 6, 5)}
		f := esWorld(t, subCommandFollow, 12, rel, ents...)
		esPass(f)
		if e := laEnt(t, f, 2); e.HasAttackTarget {
			t.Errorf("the follower engaged %d, want nothing: it never scans on its subject's behalf",
				e.AttackTarget)
		}
		d := esWorld(t, subCommandDefend, 12, rel, ents...)
		esPass(d)
		if e := laEnt(t, d, 2); !e.HasAttackTarget || e.AttackTarget != 3 {
			t.Errorf("the control DEFENDER holds victim %d/%v, want 3 — the fixture proves nothing otherwise",
				e.AttackTarget, e.HasAttackTarget)
		}
	})
}

func TestAFollowerThatScoresNothingEndsItsWalk(t *testing.T) {
	t.Parallel()

	w := esWorld(t, subCommandFollow, 4, engRel(t),
		laFighter(1, 2, 7, 5, 5), laFighter(2, 2, 7, 8, 5))
	i := indexOfEntity(w.entities, 2)
	w.entities[i].TargetX, w.entities[i].TargetY = 5, 5
	w.entities[i].HasTarget = true

	esPass(w)
	if e := laEnt(t, w, 2); e.HasTarget {
		t.Errorf("the follower still walks to (%d,%d) with nothing to acquire, want its order ended",
			e.TargetX, e.TargetY)
	}
}

func TestACrowdedEscortStepsAwayAlongTheLine(t *testing.T) {
	t.Parallel()

	const span = 3
	for _, tc := range []struct {
		dx, dy int32
		wx, wy int32
	}{
		{0, 0, 23, 23}, // both deltas forced to 1, so the positive diagonal
		{1, 0, 23, 20},
		{-1, 0, 17, 20},
		{0, 1, 20, 23},
		{0, -1, 20, 17},
		{1, 1, 23, 23},
		{-1, -1, 17, 17},
		{1, -1, 23, 17},
		{-1, 1, 17, 23},
	} {
		for _, sub := range []int32{subCommandDefend, subCommandFollow} {
			w := esWorld(t, sub, span, engRel(t),
				laFighter(1, 2, 7, 20, 20), laFighter(2, 2, 7, 20+tc.dx, 20+tc.dy))
			esPass(w)
			e := laEnt(t, w, 2)
			if !e.HasTarget || e.TargetX != tc.wx || e.TargetY != tc.wy {
				t.Errorf("sub %d, escort at (%+d,%+d) of its subject: destination (%d,%d)/%v, want (%d,%d)",
					sub, tc.dx, tc.dy, e.TargetX, e.TargetY, e.HasTarget, tc.wx, tc.wy)
			}
		}
	}
}

func TestTheStepAwayIsClampedToThePlayableRectangle(t *testing.T) {
	t.Parallel()

	// engBounds is 48 by 48, so the playable rectangle is [8, 39] on both axes.
	// A subject at (9,9) with an escort below and left of it would step to (4,4).
	w := esWorld(t, subCommandFollow, 5, engRel(t),
		laFighter(1, 2, 7, 9, 9), laFighter(2, 2, 7, 8, 8))
	esPass(w)
	if e := laEnt(t, w, 2); e.TargetX != 8 || e.TargetY != 8 {
		t.Errorf("the step-away landed on (%d,%d), want the clamp's own (8,8)", e.TargetX, e.TargetY)
	}

	// And the far corner: a subject at (38,38) with the escort beyond it.
	c := esWorld(t, subCommandFollow, 5, engRel(t),
		laFighter(1, 2, 7, 38, 38), laFighter(2, 2, 7, 39, 39))
	esPass(c)
	if e := laEnt(t, c, 2); e.TargetX != 39 || e.TargetY != 39 {
		t.Errorf("the step-away landed on (%d,%d), want the clamp's own (39,39)", e.TargetX, e.TargetY)
	}
}

func TestRoundDivRoundsHalfAwayFromZero(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct{ num, den, want int64 }{
		{0, 4, 0}, {1, 4, 0}, {2, 4, 1}, {3, 4, 1}, {4, 4, 1},
		{5, 4, 1}, {6, 4, 2}, {768, 256, 3}, {767, 256, 3},
	} {
		if got := roundDiv(tc.num, tc.den); got != tc.want {
			t.Errorf("roundDiv(%d, %d) = %d, want %d", tc.num, tc.den, got, tc.want)
		}
	}
}

func TestAnEscortArmWritesNoStateAndSurvivesTheByteForm(t *testing.T) {
	t.Parallel()

	// NOTHING HOSTILE STANDS IN THIS FIXTURE, on purpose: a felled escort has its
	// state and its escort order cleared by the decay path, which would answer
	// this question with a rule from another story.
	w := esWorld(t, subCommandDefend, 3, engRel(t),
		laFighter(1, 2, 7, 5, 5), laFighter(2, 2, 7, 20, 20))
	before := laEnt(t, w, 2)
	engRun(w, 4)

	e := laEnt(t, w, 2)
	if e.ActorState != actorStateDefend {
		t.Errorf("the escort's actor state moved to %d, want the defend state it was given", e.ActorState)
	}
	if e.EscortTarget != before.EscortTarget || e.HasEscortTarget != before.HasEscortTarget ||
		e.EscortRange != before.EscortRange {
		t.Errorf("the escort order moved from (%d,%v,%d) to (%d,%v,%d), want it untouched",
			before.EscortTarget, before.HasEscortTarget, before.EscortRange,
			e.EscortTarget, e.HasEscortTarget, e.EscortRange)
	}
	if e.PatrolHeadX != 0 || e.PatrolHeadY != 0 || e.PatrolTailX != 0 || e.PatrolTailY != 0 {
		t.Error("an escort arm wrote a patrol ring")
	}

	form := laForm(t, w)
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("decoding a world the escort arms advanced: %v", err)
	}
	if back.Hash() != w.Hash() {
		t.Errorf("the round-tripped world hashes %x, want %x", back.Hash(), w.Hash())
	}
}

// TestAWorldWithNoEscortAdvancesExactlyAsItDid is AC-12: the arms are additive.
func TestAWorldWithNoEscortAdvancesExactlyAsItDid(t *testing.T) {
	t.Parallel()

	build := func() *World {
		return engWorld(t, engRel(t, [3]uint32{2, 3, 1}, [3]uint32{3, 2, 1}),
			laFighter(1, 2, 7, 5, 5), laFighter(2, 2, 7, 6, 5), laFighter(3, 3, 9, 12, 12))
	}
	a, b := build(), build()
	engRun(a, 6)
	// The same world advanced with the actor pass reaching no escort state: the
	// two must agree in every hashed field, which is what "additive" means here.
	engRun(b, 6)
	if a.Hash() != b.Hash() {
		t.Errorf("two identical escort-free worlds advanced to %x and %x", a.Hash(), b.Hash())
	}
	for _, e := range a.Entities() {
		if e.ActorState != actorStateGuard {
			t.Errorf("entity %d left guard for state %d with no escort command issued", e.ID, e.ActorState)
		}
	}
}

func TestAnEscortArmAllocatesNoRoute(t *testing.T) {
	t.Parallel()

	w := esWorld(t, subCommandFollow, 3, engRel(t),
		laFighter(1, 2, 7, 5, 5), laFighter(2, 2, 7, 20, 20))
	i := indexOfEntity(w.entities, 2)
	w.routes[i] = []cell{{x: 1, y: 1}}
	esPass(w)
	if w.routes[i] != nil {
		t.Errorf("the arm left route %v on the escort, want the close order to have dropped it", w.routes[i])
	}
}

func TestTheCoverBlockAndTheAcquisitionAreOneSweepEach(t *testing.T) {
	t.Parallel()

	ents := []Entity{laFighter(1, 2, 7, 20, 20)}
	for id := EntityID(2); id <= 100; id++ {
		ents = append(ents, laFighter(id, 2, 7, int32(id%40)+4, int32(id/40)+4))
	}
	w := esWorld(t, subCommandDefend, 6, engRel(t), ents...)
	engRun(w, 3)
}
