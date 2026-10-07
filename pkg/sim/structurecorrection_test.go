package sim

import "testing"

func TestStructureAttackSearchRadiusIncludesTokenAndRounding(t *testing.T) {
	for _, tc := range []struct {
		token, reach uint8
		radius       int64
	}{
		{0, 1, 1}, {1, 1, 1}, {2, 1, 2}, {3, 1, 2},
		{4, 1, 3}, {2, 2, 3}, {3, 2, 3}, {255, 255, 382},
	} {
		a := Entity{X: 500, Y: 500, TokenSize: tc.token, Reach: tc.reach}
		if got := structureAttackRadius(a); got != tc.radius {
			t.Fatalf("token%d reach%d radius%d want%d", tc.token, tc.reach, got, tc.radius)
		}
		for _, sign := range []int32{-1, 1} {
			s := Structure{Col: 500 + sign*int32(tc.radius), Row: 500}
			if !InStructureReach(a, s) {
				t.Fatal("search boundary omitted a reachable cell")
			}
			s.Col += sign
			if InStructureReach(a, s) {
				t.Fatal("a reachable cell lies outside the search bounds")
			}
		}
	}
}

func structureCorrectionWorld(t *testing.T, token, side uint8, extraBlock bool) *World {
	t.Helper()
	grid := make([]byte, 20*20)
	for y := 10; y < 10+int(side); y++ {
		for x := 10; x < 10+int(side); x++ {
			grid[y*20+x] = 1
		}
	}
	if extraBlock {
		grid[9*20+8] = 1 // rejects (8,8)'s footprint despite its free anchor
	}
	a := structureCombatActor()
	a.ID, a.Owner, a.X, a.Y = 4, SelfSlot, 2, 10
	a.TokenSize, a.Speed = token, 20
	s := structureCombatTarget()
	s.ID, s.Col, s.Row, s.Width, s.Height = 4, 10, 10, side, side
	w, err := NewStructuredWorld(0, Bounds{Width: 20, Height: 20}, ModeCanonical,
		Terrain{Block: grid}, []Entity{a}, nil, Relations{}, nil, nil, nil, GhostTemplate{}, []Structure{s})
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func TestStructureAttackTwoCellApproachSearchesBeyondNominalReach(t *testing.T) {
	for _, extraBlock := range []bool{false, true} {
		w := structureCorrectionWorld(t, 2, 3, extraBlock)
		// At (8,10), Chebyshev distance 2 gives 2*256-128=384:
		// strike distance one. Its 2x2 body ends at x9, before the building.
		Step(w, []Command{{Kind: KindAttackStructure, Entity: 4, X: 4}})
		for n := 0; n < 1000 && w.structures[0].Field42 == 100; n++ {
			if !w.entities[0].HasAttackTarget {
				t.Fatal("reachable large attacker cancelled its order")
			}
			Step(w, nil)
			e := w.entities[0]
			// Independent full-footprint check of every committed position.
			for dy := int32(0); dy < 2; dy++ {
				for dx := int32(0); dx < 2; dx++ {
					x, y := e.X+dx, e.Y+dy
					if x < 0 || y < 0 || x >= 20 || y >= 20 || w.grid[y*20+x]&1 != 0 {
						t.Fatalf("approach body entered blocked cell (%d,%d)", x, y)
					}
				}
			}
		}
		if w.structures[0].Field42 == 100 {
			t.Fatal("reachable structure never struck")
		}
		if extraBlock && (w.entities[0].X != 8 || w.entities[0].Y != 10) {
			t.Fatalf("did not use the free full-body firing cell: (%d,%d)", w.entities[0].X, w.entities[0].Y)
		}
	}
}

func TestStructureSpellRuinClearsPursuitButPaysCurrentCrossing(t *testing.T) {
	w := structureCorrectionWorld(t, 1, 1, false)
	Step(w, []Command{{Kind: KindAttackStructure, Entity: 4, X: 4}})
	before := w.entities[0]
	if !before.HasTarget || before.Transit != 18 || len(w.routes[0]) == 0 {
		t.Fatalf("expected active crossing with remaining route: %+v", before)
	}
	// 1086 uses the Building resolver: a positive spread is required even
	// for this otherwise-lethal base. Zero spread is a separate no-damage arm.
	if !w.applyStructureSpellAt(10, 10, SpellRule{DamageMin: 150, DamageMax: 151}, 0) || w.structures[0].Field42 != 0 {
		t.Fatal("production structure spell consumer did not ruin target")
	}
	Step(w, nil)
	after := w.entities[0]
	if after.HasAttackTarget || after.AttackTargetKind != AttackTargetUnit || after.AttackTarget != 0 ||
		after.AttackCountdown != 0 || after.HasTarget || after.TargetX != 0 || after.TargetY != 0 ||
		after.Stall != 0 || len(w.routes[0]) != 0 {
		t.Fatalf("ruin left pursuit state: %+v route=%v", after, w.routes[0])
	}
	if after.X != before.X || after.Y != before.Y || after.Transit != before.Transit-1 || after.TransitTotal != before.TransitTotal {
		t.Fatal("cancellation discarded or moved the already-started crossing")
	}
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var restored World
	if err := restored.UnmarshalBinary(b); err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 200; n++ {
		Step(w, nil)
		Step(&restored, nil)
		e := w.entities[0]
		if w.Hash() != restored.Hash() || e.X != before.X || e.Y != before.Y || e.HasTarget || e.HasAttackTarget {
			t.Fatalf("cancelled pursuit moved or lost save continuity at tick %d", n)
		}
	}
	if w.entities[0].Transit != 0 {
		t.Fatal("cancelled pursuit stopped paying the existing crossing")
	}
}
