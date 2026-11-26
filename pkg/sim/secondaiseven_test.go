package sim

import (
	"fmt"
	"testing"
)

// secondProfileOf turns w into a second-game world.
func secondProfileOf(w *World) *World {
	w.rom2, w.game = &rom2ScriptState{}, newGameProfile(ScriptROM2)
	return w
}

var farSearchCost = []byte{1, 16, 8, 1, 8, 1, 1, 1, 16, 1, 16, 16, 16, 8, 1, 16, 1, 1, 1, 16, 8, 1, 16, 8, 16, 1, 16, 8, 1, 16, 16, 1, 16, 8, 8}

// The second game's far search of a one-cell unit tests its goal once per pair
// of generations: a goal first labelled in the first generation of a pair is
// still relaxed by the second, which can add a cheaper cell beside the goal.
func TestSecondFarSearchTestsTheGoalOncePerPairOfGenerations(t *testing.T) {
	b := Bounds{Width: 7, Height: 5}
	for _, tc := range []struct {
		name   string
		second bool
		want   string
	}{
		{"first game", false, "[{1 2} {2 2} {2 3} {3 4}]"},
		{"second game", true, "[{1 2} {2 2} {3 2} {4 3} {3 4}]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := csWorld(t, b, ModeCanonical, openGrid(b), farSearchCost, DomainGround, 0, 2)
			if tc.second {
				secondProfileOf(w)
			}
			got, ok := w.canonicalRoute(newRouteScratch(w), 0, terrainRelation, noWindow, staticBudget, exactGoal, 3, 4)
			if s := fmt.Sprint(got); !ok || s != tc.want {
				t.Fatalf("route %s ok %t, want %s", s, ok, tc.want)
			}
		})
	}
}

// The near search keeps a test before every generation in the second game too.
func TestSecondNearSearchKeepsTheTestBeforeEveryGeneration(t *testing.T) {
	b := Bounds{Width: 7, Height: 5}
	route := func(second bool) string {
		w := csWorld(t, b, ModeCanonical, openGrid(b), farSearchCost, DomainGround, 0, 2)
		if second {
			secondProfileOf(w)
		}
		got, _ := w.canonicalRoute(newRouteScratch(w), 0, unitRelation, dynamicWindow, scaledBudget, exactGoal, 3, 4)
		return fmt.Sprint(got)
	}
	if a, c := route(false), route(true); a != c {
		t.Fatalf("near search differs by game: %s and %s", a, c)
	}
}

// walkSubGoal is the cell the near search of a mover at (2,10) aims at along
// a stored route of n cells that starts at (headX,10).
func walkSubGoal(t *testing.T, second bool, headX int32, n int) cell {
	t.Helper()
	self := engFighter(1, 2, 2, 10)
	w := engWorld(t, Relations{}, self)
	if second {
		secondProfileOf(w)
	}
	var route []cell
	for k := 0; k < n; k++ {
		route = append(route, cell{x: headX + int32(k), y: 10})
	}
	last := route[n-1]
	w.entities[0].HasTarget, w.entities[0].TargetX, w.entities[0].TargetY = true, last.x, last.y
	w.routes[0] = route
	sub, ok := w.subGoal(newRouteScratch(w), 0)
	if !ok {
		t.Fatalf("route of %d from %d does not serve", n, headX)
	}
	return sub
}

// The second game's walk aims at the route end while five cells or fewer
// remain, at the head when the mover is more than three cells off it, else at
// the fourth cell; the first game's walk takes the same rule.
func TestSecondWalkAimsByTheRepathRule(t *testing.T) {
	for _, tc := range []struct {
		name          string
		headX         int32
		n             int
		first, second cell
	}{
		{"four cells", 3, 4, cell{6, 10}, cell{6, 10}},
		{"five cells", 3, 5, cell{7, 10}, cell{7, 10}},
		{"six cells near the head", 3, 6, cell{6, 10}, cell{6, 10}},
		{"six cells far from the head", 6, 6, cell{6, 10}, cell{6, 10}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := walkSubGoal(t, false, tc.headX, tc.n); got != tc.first {
				t.Fatalf("first game aims at %v, want %v", got, tc.first)
			}
			if got := walkSubGoal(t, true, tc.headX, tc.n); got != tc.second {
				t.Fatalf("second game aims at %v, want %v", got, tc.second)
			}
		})
	}
}

// hideEntity makes the entity of id invisible.
func hideEntity(t *testing.T, w *World, id EntityID) {
	t.Helper()
	if !w.attachEffect(id, id, SpellRule{ID: 15}, EffectInvisible, 1, 30, EffectDuration) {
		t.Fatal("invisibility did not attach")
	}
}

// A withdrawing second-game unit keeps an invisible hostile in its list while
// a member of its Group detects it, wherever the unit itself looks; the first
// game asks the same group.
func TestSecondWithdrawalListAsksTheGroupToDetect(t *testing.T) {
	for _, tc := range []struct {
		name         string
		second       bool
		mateDetects  uint8
		wantWithdraw bool
	}{
		{"second game, no detector", true, 0, false},
		{"second game, a member detects", true, 3, true},
		{"first game, a member detects", false, 3, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			self := withdrawalFighter(1, 2, 20, 20, 30)
			self.Wimpy = 30
			hero := withdrawalFighter(2, SelfSlot, 23, 20, 100)
			mate := withdrawalFighter(3, 2, 24, 22, 100)
			mate.SeeInvisible = tc.mateDetects
			w := engWorld(t, engRel(t, [3]uint32{2, SelfSlot, 1}), self, hero, mate)
			if tc.second {
				secondProfileOf(w)
			}
			hideEntity(t, w, 2)
			if arm, _ := w.withdrawalArm(0); (arm == withdrawFlee) != tc.wantWithdraw {
				t.Fatalf("arm %d, want withdraw %t", arm, tc.wantWithdraw)
			}
		})
	}
}

// coverTarget is the victim a defender at (20,21) takes while covering the
// subject at (20,20), among the enemies given, hidden being made invisible.
func coverTarget(t *testing.T, second bool, subjectReach uint8, hidden EntityID, enemies ...Entity) EntityID {
	t.Helper()
	defender := engFighter(1, 2, 20, 21)
	defender.Reach = 6
	subject := engFighter(2, 2, 20, 20)
	subject.Reach = subjectReach
	w := engWorld(t, engRel(t, [3]uint32{2, SelfSlot, 1}), append([]Entity{defender, subject}, enemies...)...)
	if second {
		secondProfileOf(w)
	}
	if hidden != 0 {
		hideEntity(t, w, hidden)
	}
	w.coverEngage(0, 1)
	return w.entities[0].AttackTarget
}

// A second-game defender whose defended unit has a range below 2 drops the
// flyers from its list; a longer range keeps them first. The first game does
// the same.
func TestSecondDefenderDropsFlyersForAShortRangeUnit(t *testing.T) {
	flyer := engFighter(3, SelfSlot, 22, 20)
	flyer.Domain = DomainAir
	ground := engFighter(4, SelfSlot, 24, 20)
	for _, tc := range []struct {
		name   string
		second bool
		reach  uint8
		want   EntityID
	}{
		{"first game, short range", false, 1, 4},
		{"second game, short range", true, 1, 4},
		{"second game, range 2", true, 2, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := coverTarget(t, tc.second, tc.reach, 0, flyer, ground); got != tc.want {
				t.Fatalf("victim %d, want %d", got, tc.want)
			}
		})
	}
}

// The defender's list ranks by edge distance: a wide enemy whose body is
// nearer than a narrow one wins although its anchor cell is farther.
func TestSecondDefenderRanksByEdgeDistance(t *testing.T) {
	wide := engFighter(3, SelfSlot, 15, 20)
	wide.TokenSize = 3
	narrow := engFighter(4, SelfSlot, 24, 20)
	for _, tc := range []struct {
		second bool
		want   EntityID
	}{{false, 3}, {true, 3}} {
		if got := coverTarget(t, tc.second, 1, 0, wide, narrow); got != tc.want {
			t.Fatalf("second %t: victim %d, want %d", tc.second, got, tc.want)
		}
	}
}

// An invisible enemy leaves the defender's list unless a unit of the defended
// unit's Group detects it, in both games.
func TestSecondDefenderListDropsAnInvisibleEnemyNoMemberDetects(t *testing.T) {
	near := engFighter(3, SelfSlot, 22, 20)
	far := engFighter(4, SelfSlot, 24, 20)
	for _, tc := range []struct {
		second bool
		want   EntityID
	}{{false, 4}, {true, 4}} {
		if got := coverTarget(t, tc.second, 1, 3, near, far); got != tc.want {
			t.Fatalf("second %t: victim %d, want %d", tc.second, got, tc.want)
		}
	}
}

// A second-game group keeps the first of two equal scores in the order of its
// candidate list, whichever way the list runs.
func TestSecondGroupTargetTieKeepsTheFirstOfTheList(t *testing.T) {
	member := engFighter(1, 2, 20, 20)
	a, b := engFighter(2, SelfSlot, 20, 19), engFighter(3, SelfSlot, 20, 21)
	w := secondProfileOf(engWorld(t, engRel(t, [3]uint32{2, SelfSlot, 1}), member, a, b))
	// A facing that turns equally to the two enemies, north and south of the member.
	for f := 0; f < 256; f++ {
		w.entities[0].Facing = uint8(f)
		if w.candidateCost(0, 1, orderGuard) == w.candidateCost(0, 2, orderGuard) {
			break
		}
	}
	if w.candidateCost(0, 1, orderGuard) != w.candidateCost(0, 2, orderGuard) || w.candidateCost(0, 1, orderGuard) == scoreSeed {
		t.Fatalf("fixture scores %d and %d are not equal and below the seed", w.candidateCost(0, 1, orderGuard), w.candidateCost(0, 2, orderGuard))
	}
	if got := w.chooseTarget(0, []int{1, 2}, orderGuard, targetGroupScore); got != 1 {
		t.Fatalf("list a,b chose %d, want 1", got)
	}
	if got := w.chooseTarget(0, []int{2, 1}, orderGuard, targetGroupScore); got != 2 {
		t.Fatalf("list b,a chose %d, want 2", got)
	}
}

// The defender looks five cells round the defended unit.
func TestSecondDefenderListReachesFiveCellsFromTheDefendedUnit(t *testing.T) {
	if got := coverTarget(t, true, 1, 0, engFighter(3, SelfSlot, 25, 20)); got != 3 {
		t.Fatalf("enemy 5 cells off: victim %d, want 3", got)
	}
	if got := coverTarget(t, true, 1, 0, engFighter(3, SelfSlot, 26, 20)); got != 0 {
		t.Fatalf("enemy 6 cells off: victim %d, want none", got)
	}
}

// A guarding group's notice base is floored at 8 cells in the second game.
func TestSecondGuardPostRadiusFloorIsEight(t *testing.T) {
	if minimalGuardRange != 8 {
		t.Fatalf("floor %d, want 8", minimalGuardRange)
	}
}

// A second-game mover walking the far search's route over a cost plane ends on
// its goal, and two runs of the same world hash alike tick for tick.
func TestSecondWalkOverACostPlaneIsDeterministic(t *testing.T) {
	b := Bounds{Width: 7, Height: 5}
	run := func() (hashes [3]uint64, at cell) {
		w, err := NewTerrainWorld(1, b, ModeCanonical, Terrain{Block: openGrid(b), Cost: farSearchCost},
			[]Entity{{ID: 1, X: 0, Y: 2, Domain: DomainGround, Owner: 2, HP: 10, MaxHP: 10, Speed: 40, TokenSize: 1}}, nil)
		if err != nil {
			t.Fatal(err)
		}
		secondProfileOf(w)
		w.attachMoveOrder(0, 3, 4, false)
		for k := 0; k < 1500; k++ {
			Step(w, nil)
			if k == 100 || k == 400 || k == 1499 {
				hashes[map[int]int{100: 0, 400: 1, 1499: 2}[k]] = w.Hash()
			}
		}
		return hashes, cell{x: w.entities[0].X, y: w.entities[0].Y}
	}
	h1, at := run()
	h2, _ := run()
	if h1 != h2 || at != (cell{x: 3, y: 4}) {
		t.Fatalf("hashes %v and %v, ends at %v, want equal and (3,4)", h1, h2, at)
	}
}
