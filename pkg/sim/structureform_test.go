package sim

import (
	"testing"
)

func requireStructureLegacyDigest(t *testing.T, w *World, want uint64) {
	t.Helper()
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	b = strippedWorldOfConsumables(b, w)
	b[0] = 66
	if got := fnv1a(b); got != want {
		t.Fatalf("unit-only legacy digest=%x want %x", got, want)
	}
}

func TestStructureAttackStaticFootprintAndUnreachableStop(t *testing.T) {
	a, s := structureCombatActor(), structureCombatTarget()
	s.Col, s.Width, s.Height = 8, 3, 3
	grid := make([]byte, 16*16)
	for y := 3; y < 6; y++ {
		for x := 8; x < 11; x++ {
			grid[y*16+x] = 1
		}
	}
	w, err := NewStructuredWorld(0, Bounds{Width: 16, Height: 16}, ModeCanonical, Terrain{Block: grid}, []Entity{a}, nil, Relations{}, nil, nil, nil, GhostTemplate{}, []Structure{s})
	if err != nil {
		t.Fatal(err)
	}
	Step(w, []Command{{Kind: KindAttackStructure, Entity: 0, X: 0}})
	for n := 0; n < 1000 && w.structures[0].Field42 == 100; n++ {
		Step(w, nil)
	}
	if w.structures[0].Field42 == 100 {
		t.Fatal("blocked anchor prevented a reachable attack")
	}
	if e := w.entities[0]; e.X >= 8 && e.X < 11 && e.Y >= 3 && e.Y < 6 {
		t.Fatal("attacker entered the obstruction")
	}
	for i := range grid {
		grid[i] = 1
	}
	grid[3*16+2] = 0
	w, err = NewStructuredWorld(0, Bounds{Width: 16, Height: 16}, ModeCanonical, Terrain{Block: grid}, []Entity{a}, nil, Relations{}, nil, nil, nil, GhostTemplate{}, []Structure{s})
	if err != nil {
		t.Fatal(err)
	}
	Step(w, []Command{{Kind: KindAttackStructure, Entity: 0, X: 0}})
	if w.entities[0].HasAttackTarget || w.entities[0].HasTarget {
		t.Fatal("unreachable structure left a spinning order")
	}
}

func TestStructureAttackRangeUsesCombatTokenNotRectangle(t *testing.T) {
	a, s := structureCombatActor(), structureCombatTarget()
	s.Col, s.Width, s.Height = 4, 9, 9
	if InStructureReach(a, s) {
		t.Fatal("rectangle widened combat reach")
	}
	a.TokenSize = 2
	if !InStructureReach(a, s) {
		t.Fatal("attacker token missing from strike distance")
	}
	s.Col = 2147483647
	a.X = -2147483647
	if InStructureReach(a, s) {
		t.Fatal("coordinate overflow made a distant structure reachable")
	}
}
