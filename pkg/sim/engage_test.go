package sim

import "testing"

// engBounds is wide enough for every distance these tests reach out to and for
// the notice radius to have somewhere to fall short of.
var engBounds = Bounds{Width: 48, Height: 48}

// engRel is a relation from a list of {from, to, byte} triples.
func engRel(t *testing.T, cells ...[3]uint32) Relations {
	t.Helper()
	var r Relations
	for _, c := range cells {
		r.Set(c[0], c[1], byte(c[2]))
	}
	out, err := NewRelations(r.cells)
	if err != nil {
		t.Fatalf("NewRelations: %v", err)
	}
	return out
}

// engWorld is a world over engBounds carrying ents and rel.
func engWorld(t *testing.T, rel Relations, ents ...Entity) *World {
	t.Helper()
	w, err := NewRelatedWorld(1, engBounds, ModeCanonical, Terrain{}, ents, nil, rel)
	if err != nil {
		t.Fatalf("NewRelatedWorld: %v", err)
	}
	return w
}

// engFighter is a unit that can be hit and can hit back, at the owner and group
// named, standing at (x, y). Its blow always lands so that a fight's outcome is
// decided by whether it was ever ordered rather than by a roll.
func engFighter(id EntityID, owner uint32, x, y int32) Entity {
	return Entity{ID: id, X: x, Y: y, Owner: owner, HP: 20, MaxHP: 20,
		ScanRange: engSight, DamageBase: 3, AlwaysHits: true}
}

// engSight is the range every fighter this file builds is given, and it is THIS
// FILE'S OWN NUMBER rather than anything production carries.
//
// Until 0091 these distances were written against the package constant that
// seeded every march, so a test placing a candidate at "the edge of sight" moved
// with the constant and pinned nothing about it. There is no constant now — the
// range is the entity's — but the same hazard survives a rename, so the fixture
// states a range and the distances are laid out against that.
const engSight = 5

// engRun advances w through n decision phases and answers where the decision
// left things. The phase is one of sixteen and Step reads the tick before it
// raises it, so the first decision falls on the seventh call.
func engRun(w *World, decisions int) {
	for i := 0; i <= scriptPassPhase+scriptCycle*(decisions-1); i++ {
		Step(w, nil)
	}
}

// engVictim is the victim id entity id holds, and whether it holds one.
func engVictim(w *World, id EntityID) (EntityID, bool) {
	for _, e := range w.Entities() {
		if e.ID == id {
			return e.AttackTarget, e.HasAttackTarget
		}
	}
	return 0, false
}

// TestAHostileGroupStartsAFightNobodyOrdered is AC-4, and it is the whole story
// in one test: two units that have never been commanded, a relation that says one
// hates the other, and no commands at any tick. The guard closes, strikes, and
// kills.
//
// 0108 MADE THE REVERSE CELL AUTHORED RATHER THAN ABSENT. Until that story a
// blank cell stayed blank for the life of the mission, so "one way" was the
// free consequence of naming one direction; now a connecting blow flips the
// cell it was struck through, and this fixture's victim would fight back on
// tick one. [3][2] is therefore LOCKED (bit 1) rather than left clear — the
// one thing that declines a flip — so the one-wayness this test is about is
// stated by the fixture instead of being an accident of the rules. The story
// that made the flip is not being spared here: the victim's own retaliation
// is what TestAStruckFlippablePairEndsHostileAndTheVictimsGroupAcquiresTheStriker
// below exists to show.
func TestAHostileGroupStartsAFightNobodyOrdered(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}, [3]uint32{3, 2, 2}),
		engFighter(1, 2, 5, 5), engFighter(2, 3, 5+engSight, 5))
	engRun(w, 1)
	if v, held := engVictim(w, 1); !held || v != 2 {
		t.Fatalf("after one decision the hostile unit holds victim %v/%v, want entity 2", v, held)
	}
	for i := 0; i < 400; i++ {
		Step(w, nil)
	}
	got := w.Entities()
	if got[1].Alive() {
		t.Errorf("the victim is at %d of %d health and still standing", got[1].HP, got[1].MaxHP)
	}
	// The relation is one way, so the victim never fought back — it was killed by
	// something it was never hostile to and never ordered against.
	if got[1].HasAttackTarget {
		t.Errorf("the unit nobody is hostile to acquired %d", got[1].AttackTarget)
	}
	if got[0].HP != got[0].MaxHP {
		t.Errorf("the attacker took %d damage from a unit holding no order",
			got[0].MaxHP-got[0].HP)
	}
	if got[1].HP > -10 {
		t.Fatalf("fixture stopped at %d, want the target through the -10 boundary", got[1].HP)
	}
	if got[0].HasAttackTarget {
		t.Error("the AI retained a target beyond the -10 boundary")
	}
	vi := indexOfEntity(w.entities, got[1].ID)
	w.entities[vi].HP, w.entities[vi].Dwell = decayGoneHP-1, 0
	Step(w, nil)
	if got := w.Entities(); len(got) != 1 || got[0].HasAttackTarget {
		t.Errorf("target teardown left entities %+v", got)
	}
}

// TestAStruckFlippablePairEndsHostileAndTheVictimsGroupAcquiresTheStriker is
// AC-2: two entities on slots whose pair is flippable in both directions, one
// striking the other. Both cells carry bit 0 after the blow, every other byte
// of the matrix is unchanged, and the struck slot's group — which saw nothing
// before, because no relation named either direction hostile — then acquires
// the striker at its next decision.
func TestAStruckFlippablePairEndsHostileAndTheVictimsGroupAcquiresTheStriker(t *testing.T) {
	t.Parallel()

	const attackerOwner, victimOwner = uint32(2), uint32(3)
	a := engFighter(1, attackerOwner, 5, 5)
	v := engFighter(2, victimOwner, 6, 5)
	w := engWorld(t, Relations{}, a, v)

	// Before any blow: no relation names either direction, so a decision
	// finds nothing hostile either way.
	w.engagementPass()
	if _, held := engVictim(w, 1); held {
		t.Fatal("the attacker's group acquired a candidate with no relation naming it hostile")
	}
	if _, held := engVictim(w, 2); held {
		t.Fatal("the victim's group acquired a candidate before any blow, with no relation naming it hostile")
	}

	// Entity 1 strikes entity 2 directly: engFighter always hits, and a
	// default charge of 0 floors to 1, so the order's own advance resolves
	// the blow.
	Step(w, []Command{{Kind: KindAttack, Entity: 1, X: 2}})

	rel := w.Relations()
	if !rel.Hostile(attackerOwner, victimOwner) || !rel.Hostile(victimOwner, attackerOwner) {
		t.Fatalf("after the blow [%d][%d]=%v [%d][%d]=%v, want both hostile",
			attackerOwner, victimOwner, rel.Hostile(attackerOwner, victimOwner),
			victimOwner, attackerOwner, rel.Hostile(victimOwner, attackerOwner))
	}
	// Every other byte of the matrix is unchanged.
	for from := uint32(0); from < relationSlots; from++ {
		for to := uint32(0); to < relationSlots; to++ {
			if (from == attackerOwner && to == victimOwner) || (from == victimOwner && to == attackerOwner) {
				continue
			}
			if got := rel.Byte(from, to); got != 0 {
				t.Errorf("relation[%d][%d] = %#02x, want 0 — untouched by this blow", from, to, got)
			}
		}
	}

	// The struck slot's group then acquires the striker where before it saw
	// nothing.
	w.engagementPass()
	if got, held := engVictim(w, 2); !held || got != 1 {
		t.Errorf("after the blow entity 2 holds victim %v/%v, want entity 1 acquired", got, held)
	}
}

func TestNoDecisionIsTakenOffTheDecisionPhase(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}),
		engFighter(1, 2, 5, 5), engFighter(2, 3, 7, 5))
	for i := 0; i < scriptPassPhase; i++ {
		Step(w, nil)
		if _, held := engVictim(w, 1); held {
			t.Fatalf("a decision was taken at tick %d, before the decision phase", w.Tick())
		}
	}
	Step(w, nil)
	if _, held := engVictim(w, 1); !held {
		t.Fatal("no decision was taken on the decision phase")
	}
}

// TestTheRelationDecidesAcquisitionInOneDirection is AC-3. The pair is authored
// hostile one way and peaceful the other, and only one of them ever acquires —
// which is the shape 102 of 866 ordered pairs on shipped content have.
func TestTheRelationDecidesAcquisitionInOneDirection(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		rel     [3]uint32
		acquire EntityID
	}{
		{"guards attack monsters", [3]uint32{2, 3, 1}, 1},
		{"monsters attack guards", [3]uint32{3, 2, 1}, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := engWorld(t, engRel(t, tc.rel),
				engFighter(1, 2, 5, 5), engFighter(2, 3, 7, 5))
			engRun(w, 1)
			for _, id := range []EntityID{1, 2} {
				_, held := engVictim(w, id)
				if want := id == tc.acquire; held != want {
					t.Errorf("entity %d holds a victim=%v, want %v", id, held, want)
				}
			}
		})
	}
}

// TestSightBoundsTheCandidatePopulation is AC-6. A candidate one cell outside
// every member's sight is invisible however hostile the relation is, and one cell
// inside it is taken — so the population really is bounded by the radius and not
// by the notice circle that is measured from somewhere else.
func TestSightBoundsTheCandidatePopulation(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		dx   int32
		want bool
	}{
		{"at the edge of sight", engSight, true},
		{"one cell past sight", engSight + 1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}),
				engFighter(1, 2, 5, 5), engFighter(2, 3, 5+tc.dx, 5))
			// One decision only: a unit that acquires begins to walk, and a second
			// decision would be taken from wherever the walk left it.
			engRun(w, 1)
			if _, held := engVictim(w, 1); held != tc.want {
				t.Errorf("a candidate %d cell(s) away was acquired=%v, want %v", tc.dx, held, tc.want)
			}
		})
	}
}

// TestTheNoticeRadiusIsTheArithmeticTheLawNames is AC-7.
//
// Both halves are still worth having. The radius has to be the law's expression
// or the story that freezes it freezes the wrong number, and the clip has to drop
// what lies beyond whatever radius it is handed or freezing the right number
// would still not bite.
func TestTheNoticeRadiusIsTheArithmeticTheLawNames(t *testing.T) {
	t.Parallel()

	// One member: its distance from its own centroid is 0, so the geometry term is
	// its own range alone and the floor is what wins.
	lone := []Entity{{ID: 1, X: 9, Y: 9, ScanRange: engSight}}
	if got, want := noticeBase(lone, []int{0}, 9, 9), uint8(minimalGuardRange); got != want {
		t.Errorf("a lone member's notice base is %d, want the floor %d", got, want)
	}
	// Two members far apart: the geometry beats the floor, and the term is the
	// widest member's own distance from the centroid plus THAT MEMBER'S range.
	spread := []Entity{{ID: 1, X: 0, Y: 0, ScanRange: engSight}, {ID: 2, X: 40, Y: 0, ScanRange: engSight}}
	cx, cy := groupCentroid(spread, []int{0, 1})
	if got, want := noticeBase(spread, []int{0, 1}, cx, cy), uint8(20+engSight); got != want {
		t.Errorf("a group spread over 40 cells has notice base %d, want %d", got, want)
	}
	// And the reader widens whatever base it is given by exactly the arm's
	// margin, as a byte — the ONLY place the margin is added.
	if got, want := noticeRadius(uint8(20+engSight)), uint8(20+engSight+noticeMargin); got != want {
		t.Errorf("the widened radius is %d, want the base plus the margin, %d", got, want)
	}

	// And the clip keeps exactly what is at or inside whatever radius it is given.
	ents := []Entity{{ID: 1, X: 10, Y: 10}, {ID: 2, X: 13, Y: 10}, {ID: 3, X: 14, Y: 10}}
	got := clipToNotice([]int{0, 1, 2}, ents, 10, 10, 3)
	if len(got) != 2 || got[0] != 0 || got[1] != 1 {
		t.Errorf("clipping to radius 3 about (10,10) kept %v, want the two at distance 0 and 3", got)
	}
}

func TestNoCandidateAGroupCanSeeIsEverClipped(t *testing.T) {
	t.Parallel()

	members := []int{0, 1}
	for ax := int32(0); ax < 12; ax++ {
		for ay := int32(0); ay < 12; ay++ {
			for bx := int32(0); bx < 30; bx += 3 {
				ents := []Entity{{ID: 1, X: ax, Y: ay, ScanRange: 4}, {ID: 2, X: bx, Y: 7, ScanRange: 9}}
				w := engWorld(t, Relations{}, ents...)
				cx, cy := groupCentroid(ents, members)
				r := int64(noticeBase(ents, members, cx, cy))
				stamp := w.groupSight(aiSight, members)
				for px := int32(-3); px < 40; px++ {
					for py := int32(-3); py < 20; py++ {
						c := cell{x: px, y: py}
						if !w.sightShows(stamp, c) {
							continue
						}
						if d := c.chebyshevTo(cell{x: cx, y: cy}); d > r {
							t.Fatalf("members (%d,%d) and (%d,7) clipped a candidate at (%d,%d): "+
								"distance %d from the centre against a radius of %d",
								ax, ay, bx, px, py, d, r)
						}
					}
				}
			}
		}
	}
}

// TestAGuardingGroupClipsAtItsFrozenBaseNotLiveGeometry is 0095 AC-4 and AC-5,
// and it is the story: a test about TIME rather than about arithmetic. Two
// members are sent walking apart, far enough that a freshly recomputed notice
// base would end up wider than the one the world was built with, and the
// third member dies along the way — "a group ... walk[s] ... spread[s], and
// lose[s] a member," in the spec's own words. What the clip uses at the end is
// still the base the world was built with.
//
// THE DISCRIMINATION is a candidate standing where the two readings disagree:
// inside the sight of the member that walked toward it, outside the FROZEN
// circle, and inside what a fresh recompute over the settled geometry would
// be. A build that still recomputed live on a decision path would acquire it;
// this one does not.
func TestAGuardingGroupClipsAtItsFrozenBaseNotLiveGeometry(t *testing.T) {
	t.Parallel()

	const (
		owner, group   = uint32(5), uint32(7)
		startX, startY = int32(100), int32(40)
		spreadBy       = int32(60)
	)
	bounds := Bounds{Width: 260, Height: 80}

	a := Entity{ID: 1, X: startX, Y: startY, Owner: owner, Group: group,
		HP: 10, MaxHP: 10, ScanRange: 25, Speed: 60}
	b := Entity{ID: 2, X: startX, Y: startY, Owner: owner, Group: group,
		HP: 10, MaxHP: 10, ScanRange: 25, Speed: 60}
	c := Entity{ID: 3, X: startX, Y: startY, Owner: owner, Group: group,
		HP: 1, MaxHP: 1}
	// The discriminating candidate: within entity 1's own sight once it has
	// arrived, and far past where the group started. The march's own budget
	// falls short of a bare Chebyshev disk (AI-LOS-081), so this offset is
	// comfortably inside the sight range's OWN measured reach along a
	// cardinal line rather than assumed equal to ScanRange.
	cand := Entity{ID: 4, X: startX + spreadBy + 15, Y: startY, Owner: 9, HP: 5, MaxHP: 5}

	rel := engRel(t, [3]uint32{owner, 9, 1})
	w, err := NewRelatedWorld(1, bounds, ModeCanonical, Terrain{}, []Entity{a, b, c, cand}, nil, rel)
	if err != nil {
		t.Fatalf("NewRelatedWorld: %v", err)
	}

	_, base, _ := w.groupState(owner, group)
	if base != 25 {
		t.Fatalf("the fixture's own frozen base is %d, want 25 (0+25 from either A or B, above "+
			"the floor) — this test's arithmetic assumes that number", base)
	}
	frozen := int64(noticeRadius(base))

	Step(w, []Command{
		{Kind: KindMoveTo, Entity: 1, X: startX + spreadBy, Y: startY},
		{Kind: KindMoveTo, Entity: 2, X: startX - spreadBy, Y: startY},
		{Kind: KindKill, Entity: 3},
	})
	arrived := false
	for i := 0; i < 20000; i++ {
		Step(w, nil)
		e1 := w.entities[indexOfEntity(w.entities, 1)]
		e2 := w.entities[indexOfEntity(w.entities, 2)]
		if !e1.HasTarget && !e2.HasTarget {
			arrived = true
			break
		}
	}
	if !arrived {
		t.Fatalf("entities 1 and 2 did not finish walking within the tick budget")
	}
	// Anchor each member's post at its own walked-to cell, as a stance command
	// would — this test is about the frozen NOTICE BASE, not about the walk
	// home, and leaving the construction-time post in place would have an
	// off-post guard walk home the moment it arrives here, drifting before this
	// test's own snapshot below for a reason this test is not about.
	for _, id := range []EntityID{1, 2} {
		i := indexOfEntity(w.entities, id)
		w.entities[i].PostX, w.entities[i].PostY = w.entities[i].X, w.entities[i].Y
	}
	// One more full decision cycle at the settled positions, so a natural
	// decision has certainly been taken from where they now stand and not
	// merely mid-stride.
	for i := 0; i <= scriptCycle; i++ {
		Step(w, nil)
	}

	e1 := w.entities[indexOfEntity(w.entities, 1)]
	e2 := w.entities[indexOfEntity(w.entities, 2)]
	e3 := w.entities[indexOfEntity(w.entities, 3)]
	if e1.X != startX+spreadBy || e2.X != startX-spreadBy {
		t.Fatalf("entity 1 ended at %d and entity 2 at %d, want %d and %d",
			e1.X, e2.X, startX+spreadBy, startX-spreadBy)
	}
	if e3.Alive() {
		t.Fatalf("the third member is still alive; this test needs it lost")
	}

	// The frozen base did not move.
	if _, got, _ := w.groupState(owner, group); got != base {
		t.Fatalf("the frozen base moved from %d to %d after the group walked, spread and lost a "+
			"member", base, got)
	}

	// A fresh recompute, over the members now living (1 and 2 — 3 is dead),
	// gives a wider radius: the arithmetic this build would run on a decision
	// path if the freeze were reverted.
	cx, cy := groupCentroid(w.entities, []int{indexOfEntity(w.entities, 1), indexOfEntity(w.entities, 2)})
	if cx != startX || cy != startY {
		t.Fatalf("the live centroid is (%d,%d), want the symmetric (%d,%d) this test's arithmetic "+
			"assumes", cx, cy, startX, startY)
	}
	liveBase := noticeBase(w.entities, []int{indexOfEntity(w.entities, 1), indexOfEntity(w.entities, 2)}, cx, cy)
	live := int64(noticeRadius(liveBase))
	if live <= frozen {
		t.Fatalf("a fresh recompute (%d) is not wider than the frozen radius (%d); this test needs "+
			"the two to disagree", live, frozen)
	}

	distFromCentroid := int64(cand.X) - int64(cx)
	if distFromCentroid <= frozen || distFromCentroid > live {
		t.Fatalf("the candidate sits %d from the centroid; this test needs it past the frozen "+
			"radius %d and inside what a fresh recompute would be, %d", distFromCentroid, frozen, live)
	}

	for _, id := range []EntityID{1, 2} {
		if v, held := engVictim(w, id); held {
			t.Errorf("entity %d acquired candidate %d at distance %d from the centroid, past the "+
				"frozen radius %d — the clip used live geometry (a fresh recompute would be %d) "+
				"instead of the base the world was built with", id, v, distFromCentroid, frozen, live)
		}
	}
}

func TestAPairNoRecordNamesDecidesNothing(t *testing.T) {
	t.Parallel()

	const (
		fromOwner, toOwner, grp = uint32(2), uint32(9), uint32(42)
		mx, my                  = int32(20), int32(20)
	)
	member := Entity{ID: 1, X: mx, Y: my, Owner: fromOwner, Group: grp,
		HP: 5, MaxHP: 5, ScanRange: 10}
	cand := Entity{ID: 2, X: mx + 1, Y: my, Owner: 3, HP: 5, MaxHP: 5}
	// The control: an ordinary owned group the hand-over never touches,
	// standing beside its own hostile — AC-19's "every other group" made
	// concrete. Kept well inside engBounds (48x48), unlike mx+30 would be.
	control := Entity{ID: 3, X: 5, Y: 40, Owner: fromOwner, Group: 99,
		HP: 5, MaxHP: 5, ScanRange: 10}
	controlCand := Entity{ID: 4, X: 6, Y: 40, Owner: 3, HP: 5, MaxHP: 5}

	s := fireOnce(t, []ScriptInstant{
		{Op: ScriptInstantGiveGroup, Group: grp, HasGroup: true, Player: toOwner, HasPlayer: true},
	}, 0)
	rel := engRel(t, [3]uint32{toOwner, 3, 1}, [3]uint32{fromOwner, 3, 1})
	w, err := NewRelatedWorld(1, engBounds, ModeCanonical, Terrain{}, []Entity{member, cand, control, controlCand}, s, rel)
	if err != nil {
		t.Fatalf("NewRelatedWorld: %v", err)
	}
	// Before the hand-over the constructor's own freeze left a REAL record for
	// (fromOwner, grp) — the record the hand-over is about to strand.
	if _, _, ok := w.groupState(fromOwner, grp); !ok {
		t.Fatalf("the fixture's own construction-time record is missing; this test needs a " +
			"real record left behind")
	}
	runPass(t, w)
	if got := owners(w)[1]; got != toOwner {
		t.Fatalf("the hand-over left entity 1 at owner %d, want %d", got, toOwner)
	}
	// The hand-over builds a record for the member's new group at order 0
	// (AI-CMD-033). The pair under test is the one no record names, so that
	// record is removed here.
	nowGrp := entityAt(t, w, 1).Group
	if order, _, ok := w.groupState(toOwner, nowGrp); !ok || order != orderNone {
		t.Fatalf("(%d,%d) holds order %d, record %v after the hand-over, want a record at order 0",
			toOwner, nowGrp, order, ok)
	}
	for i := len(w.groups) - 1; i >= 0; i-- {
		if w.groups[i].owner == toOwner && w.groups[i].group == nowGrp {
			w.groups = append(w.groups[:i], w.groups[i+1:]...)
		}
	}
	// The member engaged under that record during the pass above; stand it
	// down so that only the next decision can start a fight.
	at := indexOfEntity(w.entities, 1)
	w.clearOrder(at)
	w.entities[at].releaseBetweenCycles(PendingRelease)
	w.entities[at].HasAttackTarget, w.entities[at].AttackTarget = false, 0

	before := entityAt(t, w, 1)
	w.engagementPass()
	after := entityAt(t, w, 1)

	if after.HasAttackTarget {
		t.Errorf("the hand-over survivor acquired victim %d though no record names its "+
			"(owner, group) pair", after.AttackTarget)
	}
	if after != before {
		t.Errorf("the hand-over survivor's fields moved though no decision was taken for it:\n"+
			" before %+v\n after  %+v", before, after)
	}
	if v, held := engVictim(w, 3); !held || v != 4 {
		t.Errorf("the untouched control group holds victim %v/%v, want entity 4 acquired — "+
			"every other group must still decide", v, held)
	}
}

func TestOnFlatGroundSightIsInsideTheDisk(t *testing.T) {
	t.Parallel()

	w := engWorld(t, Relations{}, Entity{ID: 1, X: 20, Y: 20, ScanRange: engSight})
	stamp := w.groupSight(aiSight, []int{0})
	for x := int32(0); x < engBounds.Width; x++ {
		for y := int32(0); y < engBounds.Height; y++ {
			c := cell{x: x, y: y}
			if w.sightShows(stamp, c) && c.chebyshevTo(cell{x: 20, y: 20}) > engSight {
				t.Fatalf("(%d,%d) is visible at Chebyshev %d on flat ground, past a range of %d",
					x, y, c.chebyshevTo(cell{x: 20, y: 20}), engSight)
			}
		}
	}
}

func TestAGroupStandingItsGroundTakesOnlyWhatItCanStrike(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		at    int32
		stand bool
		guard bool
	}{
		{"adjacent", groupScorerReach, true, true},
		{"one cell past reach", groupScorerReach + 1, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// The same geometry twice, once under each stance, so the only
			// difference between the two answers is the owner slot.
			for _, s := range []struct {
				owner uint32
				want  bool
			}{{SelfSlot, tc.stand}, {SelfSlot + 1, tc.guard}} {
				rel := engRel(t, [3]uint32{s.owner, 9, 1})
				w := engWorld(t, rel, engFighter(1, s.owner, 5, 5), engFighter(2, 9, 5+tc.at, 5))
				engRun(w, 1)
				if _, held := engVictim(w, 1); held != s.want {
					t.Errorf("owner %d at distance %d acquired=%v, want %v",
						s.owner, tc.at, held, s.want)
				}
			}
		})
	}
}

// TestAGroundMemberNeverTakesAFlier is AC-9 and the reachable half of the
// preference matrix: two cells of it are an absolute veto, and no distance, no
// relation and no absence of alternatives can get past one.
func TestAGroundMemberNeverTakesAFlier(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		member   Domain
		want     bool
		nameCand Domain
	}{
		{"ground member, air candidate", DomainGround, false, DomainAir},
		{"ghost member, air candidate", DomainGhost, false, DomainAir},
		{"air member, air candidate", DomainAir, true, DomainAir},
		{"ground member, ground candidate", DomainGround, true, DomainGround},
		{"ground member, ghost candidate", DomainGround, true, DomainGhost},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, c := engFighter(1, 2, 5, 5), engFighter(2, 3, 7, 5)
			m.Domain, c.Domain = tc.member, tc.nameCand
			w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), m, c)
			engRun(w, 1)
			if _, held := engVictim(w, 1); held != tc.want {
				t.Errorf("acquired=%v, want %v", held, tc.want)
			}
		})
	}
}

func TestTheCheapestCandidateWinsAndTheFirstBreaksATie(t *testing.T) {
	t.Parallel()

	rel := engRel(t, [3]uint32{2, 3, 1})
	t.Run("the nearer of two", func(t *testing.T) {
		// Entity 2 is the FARTHER of the pair and the lower id, so a build that
		// took the first candidate rather than the cheapest would take it.
		w := engWorld(t, rel, engFighter(1, 2, 5, 5), engFighter(2, 3, 9, 5), engFighter(3, 3, 7, 5))
		engRun(w, 1)
		if v, held := engVictim(w, 1); !held || v != 3 {
			t.Errorf("took %v/%v, want the nearer entity 3", v, held)
		}
	})
	t.Run("the first of two equals", func(t *testing.T) {
		w := engWorld(t, rel, engFighter(1, 2, 5, 5), engFighter(2, 3, 7, 5), engFighter(3, 3, 3, 5))
		engRun(w, 1)
		if v, held := engVictim(w, 1); !held || v != 2 {
			t.Errorf("took %v/%v, want the first of the two equals, entity 2", v, held)
		}
	})
}

func TestAGroupWithOnlyCorpsesInSightAttacksACorpse(t *testing.T) {
	t.Parallel()

	rel := engRel(t, [3]uint32{2, 3, 1})
	// Both bodies are tried: DEAD, which the attack cycle drops within the tick
	// that issues it (0086 D-8), and DOWNED, which it does not. The decision is
	// asked directly so that the selection this story owns is measured whichever
	// of the two the body is.
	for _, hp := range []int32{-1, 0} {
		body := engFighter(2, 3, 7, 5)
		body.HP = hp
		w := engWorld(t, rel, engFighter(1, 2, 5, 5), body)
		w.engagementPass()
		if v, held := engVictim(w, 1); !held || v != 2 {
			t.Errorf("a body at %d health was selected as %v/%v, want entity 2", hp, v, held)
		}
	}
	// A DOWNED body is nearer than the living unit and loses to it anyway, which
	// is the half of the park rule a single body cannot witness. It is run through
	// Step, because a downed victim is one the cycle keeps.
	body := engFighter(2, 3, 6, 5)
	body.HP = 0
	w := engWorld(t, rel, engFighter(1, 2, 5, 5), body, engFighter(3, 3, 9, 5))
	engRun(w, 1)
	if v, held := engVictim(w, 1); !held || v != 3 {
		t.Errorf("took %v/%v, want the living unit even though the body is nearer", v, held)
	}
}

func TestReEngagingTheSameVictimDoesNotRestartTheCycle(t *testing.T) {
	t.Parallel()

	// A charge longer than the decision's own period, so a reload would fall due
	// before the blow ever did.
	build := func(owner uint32, rel Relations) *World {
		a, b := engFighter(1, owner, 5, 5), engFighter(2, 3, 5, 6)
		a.AttackCharge = scriptCycle * 3
		return engWorld(t, rel, a, b)
	}
	decided := build(2, engRel(t, [3]uint32{2, 3, 1}))
	ordered := build(SelfSlot, engRel(t))
	for i := 0; i < 200; i++ {
		Step(decided, nil)
		// The commanded fight is started on the tick the decision starts its own,
		// so what is compared is the cadence and not the offset between them. It
		// is then re-issued every tick against the decision's every sixteenth,
		// which is the harsher test of the same rule.
		var cmds []Command
		if i >= scriptPassPhase {
			cmds = []Command{{Kind: KindAttack, Entity: 1, X: 2}}
		}
		Step(ordered, cmds)
	}
	d, o := decided.Entities()[1], ordered.Entities()[1]
	if d.HP == d.MaxHP {
		t.Fatalf("the decided fight landed nothing in 200 ticks — HP %d of %d", d.HP, d.MaxHP)
	}
	if d.HP != o.HP {
		t.Errorf("the decided fight left the victim at %d and the commanded one at %d — "+
			"the re-issue is restarting the cycle", d.HP, o.HP)
	}
}

func TestAFriendlyRelationDoesNotEndAnAttackAlreadyIssued(t *testing.T) {
	t.Parallel()

	victim := engFighter(2, 3, 5, 6)
	victim.HP, victim.MaxHP = 1<<20, 1<<20 // outlives the run, so its death cannot end the order
	w := engWorld(t, engRel(t, [3]uint32{SelfSlot, 3, 2}, [3]uint32{3, SelfSlot, 2}),
		engFighter(1, SelfSlot, 5, 5), victim)
	Step(w, []Command{{Kind: KindAttack, Entity: 1, X: 2}})
	for i := 0; i < 100; i++ {
		Step(w, nil)
	}
	got := w.Entities()
	if !got[0].HasAttackTarget {
		t.Error("a decision under a peaceful relation ended an attack already issued")
	}
	if got[1].HP >= got[1].MaxHP {
		t.Error("the attacker stopped striking")
	}
}

func TestASlotTheRosterDoesNotNameNeitherAcquiresNorIsAcquired(t *testing.T) {
	t.Parallel()

	// Row 0 and column 0 are written as hostile as a byte array allows, so what
	// keeps slot 0 out is the index law and not an empty matrix.
	rel := engRel(t, [3]uint32{0, 3, 1}, [3]uint32{3, 0, 1}, [3]uint32{0, 0, 1})
	w := engWorld(t, rel, engFighter(1, 0, 5, 5), engFighter(2, 3, 7, 5))
	engRun(w, 1)
	for _, e := range w.Entities() {
		if e.HasAttackTarget {
			t.Errorf("entity %d (owner %d) acquired %d", e.ID, e.Owner, e.AttackTarget)
		}
	}
}

func TestADecisionIsIdempotentOnAWorldNothingElseMoved(t *testing.T) {
	t.Parallel()

	// The bystander is owner 4: hostile to nobody and nobody's enemy, standing in
	// everybody's sight.
	w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}),
		engFighter(1, 2, 5, 5), engFighter(2, 3, 7, 5), engFighter(3, 4, 6, 6))
	before := w.Entities()[2]
	w.engagementPass()
	first := w.Entities()
	w.engagementPass()
	second := w.Entities()
	for i := range first {
		if first[i] != second[i] {
			t.Errorf("entity %d differs between two decisions:\n first %+v\nsecond %+v",
				first[i].ID, first[i], second[i])
		}
	}
	if first[2] != before {
		t.Errorf("the bystander was written to:\n before %+v\n after  %+v", before, first[2])
	}
	if !first[0].HasAttackTarget {
		t.Fatal("nothing was acquired — this test would pass for the wrong reason")
	}
	// And an attacker holds no destination of its own: the engage ends the walk.
	if first[0].HasTarget {
		t.Error("the engaged member kept a destination")
	}
}

func TestTheDecisionIsDeterministic(t *testing.T) {
	t.Parallel()

	build := func() *World {
		return engWorld(t, engRel(t, [3]uint32{2, 3, 1}, [3]uint32{3, 2, 1}),
			engFighter(1, 2, 5, 5), engFighter(2, 3, 9, 8),
			engFighter(3, 2, 6, 7), engFighter(4, 3, 10, 5))
	}
	a, b := build(), build()
	var cut *World
	for i := 0; i < 300; i++ {
		Step(a, nil)
		Step(b, nil)
		if a.Hash() != b.Hash() {
			t.Fatalf("tick %d: two identical worlds diverged", a.Tick())
		}
		if i == 150 {
			cut = &World{}
			if err := cut.UnmarshalBinary(mustMarshal(t, a)); err != nil {
				t.Fatalf("UnmarshalBinary: %v", err)
			}
		}
	}
	for i := 151; i < 300; i++ {
		Step(cut, nil)
	}
	if cut.Hash() != a.Hash() {
		t.Error("a world resumed from its bytes mid-fight stepped on to a different digest")
	}
	if a.Entities()[1].HP == a.Entities()[1].MaxHP {
		t.Error("nothing happened in 300 ticks — this test would pass for the wrong reason")
	}
}

func TestAGroupSeesAsOneAnimal(t *testing.T) {
	t.Parallel()

	// Two members of one group, far enough apart that neither march reaches the
	// other's quadrant, and a hostile unit beside the second of them.
	rel := engRel(t, [3]uint32{2, 3, 1})
	blind := engFighter(1, 2, 10, 20)
	scout := engFighter(2, 2, 30, 20)
	prey := engFighter(3, 3, 30+engSight, 20)
	w := engWorld(t, rel, blind, scout, prey)
	engRun(w, 1)

	// The blind member takes the same candidate, at a distance no march of its own
	// could ever have lit.
	v, held := engVictim(w, 1)
	if !held || v != 3 {
		t.Fatalf("the member out of range holds victim %v/%v, want entity 3", v, held)
	}
	if _, held := engVictim(w, 2); !held {
		t.Error("the member that can see the candidate did not take it")
	}

	// And a candidate no member can see is a candidate for none.
	far := engFighter(4, 3, 30+engSight+3, 40)
	w2 := engWorld(t, rel, blind, scout, far)
	engRun(w2, 1)
	for _, id := range []EntityID{1, 2} {
		if _, held := engVictim(w2, id); held {
			t.Errorf("entity %d acquired a candidate no member of its group can see", id)
		}
	}
}

// TestARidgeStopsAnEngagementFlatGroundMakes is AC-5 at the decision, which is
// where the story's title is: two hostile units at a distance the flat-ground
// build acquires across, with one column of raised ground between them.
func TestARidgeStopsAnEngagementFlatGroundMakes(t *testing.T) {
	t.Parallel()

	rel := engRel(t, [3]uint32{2, 3, 1})
	ents := []Entity{engFighter(1, 2, 20, 20), engFighter(2, 3, 25, 20)}

	h := make([]byte, int(engBounds.Width)*int(engBounds.Height))
	for y := int32(0); y < engBounds.Height; y++ {
		h[int(y)*int(engBounds.Width)+22] = 100
	}
	for _, tc := range []struct {
		name string
		ter  Terrain
		want bool
	}{
		{"flat ground", Terrain{}, true},
		{"a ridge between them", Terrain{Height: h}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, err := NewRelatedWorld(1, engBounds, ModeCanonical, tc.ter, ents, nil, rel)
			if err != nil {
				t.Fatalf("NewRelatedWorld: %v", err)
			}
			engRun(w, 1)
			if _, held := engVictim(w, 1); held != tc.want {
				t.Errorf("acquired=%v, want %v", held, tc.want)
			}
		})
	}
}

// TestTheSightPredicateReachesNeitherTheBytesNorTheDigest is AC-11. The window
// tables are derived from a constant and the stamp is scratch, so nothing this
// story adds is state: a world round trips unchanged, and two worlds built alike
// hash alike.
//
// AC-11 ORIGINALLY ALSO ASSERTED formatVersion == 16, and that clause was dropped
// where 0089 merged into 0090. It was the wrong witness for its own claim. The
// claim is "this story adds no field"; a literal version number cannot say that —
// it would hold equally if some OTHER story added a field and forgot to bump, and
// it fails when a story legitimately takes the next version, which is exactly what
// 0089 did with 17. What witnesses the claim is the pair below plus the empty
// pkg/sim field-set diff at the story's own merge. This is the same defect shape
// 0090's own mutation audit caught in M4: a test that reads its bound off the very
// constant it is pinning moves with whatever moves that constant.
func TestTheSightPredicateReachesNeitherTheBytesNorTheDigest(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}),
		engFighter(1, 2, 20, 20), engFighter(2, 3, 24, 20))
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(b); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	again, err := back.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if string(again) != string(b) {
		t.Error("a world does not round-trip through its byte form")
	}
	if back.Hash() != w.Hash() {
		t.Error("a round-tripped world hashes differently")
	}
}

func TestTwoDecisionsOnAnUnchangedWorldAgree(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}),
		engFighter(1, 2, 20, 20), engFighter(2, 3, 24, 20))
	a := w.groupSight(aiSight, []int{0, 1})
	b := w.groupSight(aiSight, []int{0, 1})
	if string(a) != string(b) {
		t.Error("two marches over an unchanged world light different cells")
	}
	engRun(w, 1)
	first, _ := engVictim(w, 1)
	w.decide(w.aiGroups()[0])
	second, _ := engVictim(w, 1)
	if first != second {
		t.Errorf("a second decision moved the assignment from %d to %d", first, second)
	}
}
