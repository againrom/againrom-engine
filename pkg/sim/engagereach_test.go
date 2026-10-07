package sim

import "testing"

// engReachFighter is engFighter with a chosen domain and reach — the two
// fields this story's tests need and engFighter itself never touches, since
// every other test in this package that calls it is silent on both.
func engReachFighter(id EntityID, owner uint32, x, y int32, domain Domain, reach uint8) Entity {
	e := engFighter(id, owner, x, y)
	e.Domain, e.Reach = domain, reach
	return e
}

// TestAGroundMemberOfReachAboveOneScoresAFlier is AC-2. Row 1 — the ground
// row — holds a zero in the air column, so a reach-1 ground member vetoes a
// flier at any distance; row 0 holds no zero there, so a reach-4 member of
// the same domain scores it, and the value scored is row 0's own cell for a
// flier, which row 1 could never have produced at all.
func TestAGroundMemberOfReachAboveOneScoresAFlier(t *testing.T) {
	t.Parallel()

	w := engWorld(t, Relations{},
		engReachFighter(1, 2, 5, 5, DomainGround, 1),
		engReachFighter(2, 3, 9, 5, DomainAir, 1))
	mi, ci := indexOfEntity(w.entities, 1), indexOfEntity(w.entities, 2)

	if got := w.candidateCost(mi, ci, orderGuard); got != scoreSeed {
		t.Fatalf("a reach-1 ground member scored a flier at %d, want the seed %d", got, scoreSeed)
	}

	w.entities[mi].Reach = 4
	got := w.candidateCost(mi, ci, orderGuard)
	if got == scoreSeed {
		t.Fatal("a reach-4 ground member still scored the seed against a flier")
	}
	// The member's own row (ground) holds a zero in the air column and could
	// never have produced a finite value; row 0's cell for a flier is 4, and
	// at distance 4 (within reach) the rewritten distance term is 1 — pref 4
	// under the ordinary order takes the cost -= cost/4 arm.
	pref := preference[0][lawDomain(DomainAir)]
	if pref != 4 {
		t.Fatalf("fixture assumption broken: row 0's air cell is %d, want 4", pref)
	}
	want := int32(1)<<8 + turnCost(w.entities[mi], w.entities[ci])
	want -= want / 4
	if got != want {
		t.Errorf("a reach-4 ground member scored a flier at %d, want the row-0 value %d", got, want)
	}
}

func TestTheRewrittenDistanceTermIsFlatThenRises(t *testing.T) {
	t.Parallel()

	const reach = 4
	if pref := preference[0][lawDomain(DomainGhost)]; pref == 1 || pref == 4 {
		t.Fatalf("fixture assumption broken: row 0's ghost cell is %d, want neither 1 nor 4 "+
			"(so no modifier arm touches the cost read below)", pref)
	}

	var terms []int32
	for sep := int32(0); sep <= 2*reach; sep++ {
		w := engWorld(t, Relations{},
			engReachFighter(1, 2, 5, 5, DomainGround, reach),
			engReachFighter(2, 3, 5+sep, 5, DomainGhost, 1))
		mi, ci := indexOfEntity(w.entities, 1), indexOfEntity(w.entities, 2)
		term := w.candidateCost(mi, ci, orderGuard) >> 8
		terms = append(terms, term)

		want := int32(1)
		if sep > reach {
			want = sep + 1 - reach
		}
		if term != want {
			t.Errorf("separation %d at reach %d: distance term %d, want %d", sep, reach, term, want)
		}
	}
	for i := 1; i < len(terms); i++ {
		if terms[i] < terms[i-1] {
			t.Errorf("the term fell from %d to %d between separations %d and %d — P-1 requires "+
				"non-decreasing", terms[i-1], terms[i], i-1, i)
		}
	}
	for i, term := range terms {
		if int32(i) > reach && term < 2 {
			t.Errorf("separation %d: term %d is below the floor of 2 past the reach", i, term)
		}
	}
}

// TestStandingGroundWithReachAboveOneTakesUpToItsReach is AC-4: the refusal
// now reads the REWRITTEN term, so a reach-4 member takes a candidate
// standing at exactly that reach and refuses one a cell further — where a
// reach-1 member, whose distance term is never rewritten, still takes only
// what stands adjacent, exactly as it did before this story.
func TestStandingGroundWithReachAboveOneTakesUpToItsReach(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		reach uint8
		at    int32
		want  bool
	}{
		{"reach 4, candidate at reach", 4, 4, true},
		{"reach 4, candidate one past reach", 4, 5, false},
		{"reach 1, candidate adjacent", 1, 1, true},
		{"reach 1, candidate one past adjacent", 1, 2, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rel := engRel(t, [3]uint32{SelfSlot, 9, 1})
			w := engWorld(t, rel, engFighter(1, SelfSlot, 5, 5), engFighter(2, 9, 5+tc.at, 5))
			w.entities[indexOfEntity(w.entities, 1)].Reach = tc.reach
			engRun(w, 1)
			if _, held := engVictim(w, 1); held != tc.want {
				t.Errorf("reach %d, candidate at distance %d: acquired=%v, want %v",
					tc.reach, tc.at, held, tc.want)
			}
		})
	}
}

func TestAReachAboveOneGroundCandidateFoldsOntoTheImmobileColumn(t *testing.T) {
	t.Parallel()

	if folded, unfolded := preference[1][0], preference[1][1]; folded == unfolded {
		t.Fatalf("fixture assumption broken: row 1's immobile cell (%d) and ground "+
			"cell (%d) must differ for this test to tell the two columns apart", folded, unfolded)
	}

	// Member reach is 1, so the row choice stays at the member's own domain
	// (ground, row 1) and only the column choice under test moves.
	w := engWorld(t, Relations{},
		engReachFighter(1, 2, 5, 5, DomainGround, 1),
		engReachFighter(2, 3, 8, 5, DomainGround, 4))
	mi, ci := indexOfEntity(w.entities, 1), indexOfEntity(w.entities, 2)

	got := w.candidateCost(mi, ci, orderGuard)
	want := int32(3)<<8 + turnCost(w.entities[mi], w.entities[ci])
	want -= want / 4 // preference[1][0] is 4, the pref==4 ordinary-order arm
	if got != want {
		t.Errorf("a reach-4 ground candidate scored %d, want the immobile-column value %d", got, want)
	}
}

func TestAReachAboveOneFlierKeepsItsOwnColumnUnderTheOrdinaryOrder(t *testing.T) {
	t.Parallel()

	if pref := preference[1][3]; pref != 0 {
		t.Fatalf("fixture assumption broken: row 1's air cell is %d, want the veto 0", pref)
	}

	w := engWorld(t, Relations{},
		engReachFighter(1, 2, 5, 5, DomainGround, 1),
		engReachFighter(2, 3, 6, 5, DomainAir, 4))
	mi, ci := indexOfEntity(w.entities, 1), indexOfEntity(w.entities, 2)

	if got := w.candidateCost(mi, ci, orderGuard); got != scoreSeed {
		t.Errorf("a reach-4 flier under the ordinary order scored %d, want the seed %d "+
			"(its column stays air, not the immobile one)", got, scoreSeed)
	}
}

func TestTheSameFlierFoldsOntoTheImmobileColumnUnderStandGround(t *testing.T) {
	t.Parallel()

	// Distance 1 keeps the stand-ground refusal from firing: the member's own
	// reach is 1 here, so its distance term is never rewritten, and the
	// refusal's ceiling is the literal 1 — one cell further and the refusal
	// would return the seed regardless of the column choice under test.
	w := engWorld(t, Relations{},
		engReachFighter(1, 2, 5, 5, DomainGround, 1),
		engReachFighter(2, 3, 6, 5, DomainAir, 4))
	mi, ci := indexOfEntity(w.entities, 1), indexOfEntity(w.entities, 2)

	got := w.candidateCost(mi, ci, orderStandGround)
	if got == scoreSeed {
		t.Fatal("a reach-4 flier still scored the seed under stand ground")
	}
	want := int32(1)<<8 + turnCost(w.entities[mi], w.entities[ci])
	want >>= 1 // preference[1][0] is 4, the pref==4 stand-ground arm
	if got != want {
		t.Errorf("a reach-4 flier under stand ground scored %d, want the immobile-column value %d", got, want)
	}
}

func TestAReachOneMemberReadsThePlainDistanceAtZeroSeparation(t *testing.T) {
	t.Parallel()

	if pref := preference[3][1]; pref == 0 {
		t.Fatalf("fixture assumption broken: row 3's ground cell is a veto (%d), so this "+
			"pair could never reach the distance term at all", pref)
	}

	// The two stand on ONE cell, which is what makes the separation zero.
	w := engWorld(t, Relations{},
		engReachFighter(1, 2, 5, 5, DomainAir, 1),
		engReachFighter(2, 3, 5, 5, DomainGround, 1))
	mi, ci := indexOfEntity(w.entities, 1), indexOfEntity(w.entities, 2)
	if d := cellOf(w.entities[mi]).chebyshevTo(cellOf(w.entities[ci])); d != 0 {
		t.Fatalf("fixture assumption broken: the two are %d cells apart, want 0", d)
	}

	got := w.candidateCost(mi, ci, orderGuard)
	want := int32(0)<<8 + turnCost(w.entities[mi], w.entities[ci])
	want += want / 2 // preference[3][1] is 1, the pref==1 ordinary-order arm
	if got != want {
		t.Errorf("a reach-1 member scored a candidate on its own cell at %d, want the plain "+
			"distance term %d — the rewrite must not fire at a reach of 1", got, want)
	}
}
