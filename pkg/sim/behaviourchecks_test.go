package sim

import "testing"

func TestTheNearSearchAimsAtTheRouteEndHeadOrFourthCell(t *testing.T) {
	line := func(from, n int32) []cell {
		route := make([]cell, n)
		for k := range route {
			route[k] = cell{x: from + int32(k), y: 0}
		}
		return route
	}
	for _, tc := range []struct {
		name  string
		route []cell
		want  cell
	}{
		{"five cells aim at the end", line(1, 5), cell{5, 0}},
		{"six cells aim at the fourth cell", line(1, 6), cell{4, 0}},
		{"a head three off is near", line(3, 6), cell{6, 0}},
		{"a head four off is aimed at", line(4, 6), cell{4, 0}},
		{"five cells with a far head aim at the end", line(4, 5), cell{8, 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			end := tc.route[len(tc.route)-1]
			w := sgWorld(t, []Entity{{ID: 1, X: 0, Y: 0, TargetX: end.x, TargetY: end.y, HasTarget: true}}, tc.route)
			sub, serves := w.subGoal(newRouteScratch(w), 0)
			if !serves || sub != tc.want {
				t.Fatalf("aimed at %v (serves %v), want %v", sub, serves, tc.want)
			}
		})
	}
}

func postGuardWorld(t *testing.T, dx int32) *World {
	t.Helper()
	return engWorld(t, engRel(t, [3]uint32{2, 3, 1}, [3]uint32{3, 2, 1}),
		withdrawalFighter(1, 2, 20, 20, 100), withdrawalFighter(2, 3, 20+dx, 20, 100))
}

func TestAGuardPostScansItsOwnMoverByte(t *testing.T) {
	for _, tc := range []struct {
		name   string
		dx     int32
		radius byte // 5 leaves the constructed byte
		want   int
	}{
		{"constructed 5 reaches 5", 5, 5, 1},
		{"constructed 5 stops short of 6", 6, 5, 0},
		{"a restored 6 reaches 6", 6, 6, 1},
		{"a restored 0 scans the centre only", 1, 0, 0},
		{"a restored 1 reaches 1", 1, 1, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := postGuardWorld(t, tc.dx)
			if tc.radius != ConstructorSearchRadius {
				setFleeSearchRadius(t, w, 1, tc.radius)
			}
			if got := len(w.postCandidates(0, cell{x: 20, y: 20})); got != tc.want {
				t.Fatalf("block holds %d candidates, want %d", got, tc.want)
			}
		})
	}
}

func coverWorld(t *testing.T, dy int32) *World {
	t.Helper()
	return engWorld(t, engRel(t, [3]uint32{2, 3, 1}, [3]uint32{3, 2, 1}),
		withdrawalFighter(1, 2, 10, 10, 100), withdrawalFighter(2, 2, 10, 14, 100), withdrawalFighter(3, 3, 10, 14+dy, 100))
}

func TestACoverScansTheProtectedUnitsMoverByte(t *testing.T) {
	for _, tc := range []struct {
		name                string
		dy                  int32
		protected, defender byte
		want                bool
	}{
		{"constructed 5 reaches 5", 5, 5, 5, true},
		{"constructed 5 stops short of 6", 6, 5, 5, false},
		{"the protected unit's 6 reaches 6", 6, 6, 5, true},
		{"the defender's own 0 is not read", 5, 5, 0, true},
		{"the defender's own 9 is not read", 6, 5, 9, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := coverWorld(t, tc.dy)
			if tc.protected != ConstructorSearchRadius {
				setFleeSearchRadius(t, w, 2, tc.protected)
			}
			if tc.defender != ConstructorSearchRadius {
				setFleeSearchRadius(t, w, 1, tc.defender)
			}
			w.coverEngage(0, 1)
			if got := w.entities[0].HasAttackTarget && w.entities[0].AttackTarget == 3; got != tc.want {
				t.Fatalf("defender engaged the hostile = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestACoverRanksByEdgeDistance(t *testing.T) {
	near := withdrawalFighter(3, 3, 24, 20, 100)
	wide := withdrawalFighter(4, 3, 15, 20, 100)
	wide.TokenSize = 3
	defender := withdrawalFighter(1, 2, 20, 20, 100)
	defender.Reach = 4
	subject := withdrawalFighter(2, 2, 20, 18, 100)
	w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}, [3]uint32{3, 2, 1}), defender, subject, near, wide)
	d, n := w.targetEdgeDistance(w.entities[0], w.entities[3]), w.targetEdgeDistance(w.entities[0], w.entities[2])
	if cell := (cell{x: 20, y: 20}); cell.chebyshevTo(cellOf(&w.entities[2])) >= cell.chebyshevTo(cellOf(&w.entities[3])) || d >= n {
		t.Fatalf("fixture: edge distances wide %d near %d, want wide the smaller while its anchor is the farther", d, n)
	}
	w.coverEngage(0, 1)
	if e := w.entities[0]; !e.HasAttackTarget || e.AttackTarget != 4 {
		t.Fatalf("defender engaged %d (%v), want the edge-nearer wide candidate 4", e.AttackTarget, e.HasAttackTarget)
	}
}

func TestACoverDropsFliersForAProtectedUnitOfReachBelowTwo(t *testing.T) {
	for _, tc := range []struct {
		name  string
		reach uint8
		want  EntityID
	}{
		{"reach 1 drops the flier", 1, 3},
		{"reach 2 prefers the flier", 2, 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defender := withdrawalFighter(1, 2, 20, 20, 100)
			defender.Reach = 4
			subject := withdrawalFighter(2, 2, 20, 18, 100)
			subject.Reach = tc.reach
			ground := withdrawalFighter(3, 3, 21, 20, 100)
			flier := withdrawalFighter(4, 3, 24, 20, 100)
			flier.Domain = DomainAir
			w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}, [3]uint32{3, 2, 1}), defender, subject, ground, flier)
			w.coverEngage(0, 1)
			if e := w.entities[0]; !e.HasAttackTarget || e.AttackTarget != tc.want {
				t.Fatalf("defender engaged %d (%v), want %d", e.AttackTarget, e.HasAttackTarget, tc.want)
			}
		})
	}
}

func retreatWorld(t *testing.T, detect, ownDetect uint8, host cell, size uint8, mateGroup uint32) *World {
	t.Helper()
	self := withdrawalFighter(1, 2, 20, 20, 8)
	self.Wimpy, self.Group, self.SeeInvisible = 8, 7, ownDetect
	mate := withdrawalFighter(2, 2, 20, 21, 100)
	mate.Group, mate.SeeInvisible = mateGroup, detect
	hostile := withdrawalFighter(3, 3, host.x, host.y, 100)
	hostile.TokenSize = size
	w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), self, mate, hostile)
	if !w.attachEffect(3, 3, SpellRule{ID: 15}, EffectInvisible, 1, 30, EffectDuration) {
		t.Fatal("invisibility did not attach")
	}
	return w
}

func TestARetreatSeesAnInvisibleHostileByTheGroupsEdgeDistance(t *testing.T) {
	for _, tc := range []struct {
		name      string
		detect    uint8
		ownDetect uint8
		host      cell
		size      uint8
		mateGroup uint32
		want      bool
	}{
		{"a groupmate at the edge distance", 3, 0, cell{23, 20}, 1, 7, true},
		{"a groupmate one short", 2, 0, cell{23, 20}, 1, 7, false},
		{"a wide hostile is measured edge to edge", 3, 0, cell{15, 20}, 3, 7, true},
		{"a wide hostile one short", 2, 0, cell{15, 20}, 3, 7, false},
		{"a unit of another group does not count", 5, 0, cell{23, 20}, 1, 8, false},
		{"the unit's own byte counts as a member's", 0, 3, cell{23, 20}, 1, 7, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := retreatWorld(t, tc.detect, tc.ownDetect, tc.host, tc.size, tc.mateGroup)
			regWithdrawalPass(w, nil)
			if got := w.entities[0].HasTarget; got != tc.want {
				t.Fatalf("retreat started = %v, want %v: %+v", got, tc.want, w.entities[0])
			}
		})
	}
}
