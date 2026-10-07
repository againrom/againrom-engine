package sim

import (
	"testing"
)

func destructibleWorld(t *testing.T, structures []Structure) *World {
	t.Helper()
	w, err := NewStructuredWorld(1052, Bounds{Width: 16, Height: 16}, ModeCanonical,
		Terrain{}, nil, nil, Relations{}, nil, nil, nil, GhostTemplate{}, structures)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func TestBlastAndRingSpellsDamageSingleCellStructures(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   uint16
		mode uint8
	}{
		{"Fire Ball blast", 2, areaModeBlast},
		{"Hail ring", 21, areaModeRing},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := destructibleWorld(t, []Structure{{
				ID: 7, Field42: 9, MaxHealth: 9, Col: 5, Row: 6,
				Width: 1, Height: 1, Attach: 1,
			}})
			rule := SpellRule{ID: tc.id, DamageMin: 9, DamageMax: 10, Damaging: true, Area: true}
			w.applyAreaCells(cellEffect{Spell: tc.id, Mode: tc.mode}, rule, []uint16{cellKey(5, 6)})
			if got := w.structures[0].Field42; got != 4 && got != 5 {
				t.Fatalf("health after first hit = %d, want 4 or 5", got)
			}
			w.applyAreaCells(cellEffect{Spell: tc.id, Mode: tc.mode}, rule, []uint16{cellKey(5, 6)})
			w.applyAreaCells(cellEffect{Spell: tc.id, Mode: tc.mode}, rule, []uint16{cellKey(5, 6)})
			if got := w.structures[0].Field42; got != 0 {
				t.Fatalf("health after lethal hits = %d, want clamped zero", got)
			}
		})
	}
}

func TestCloudDoesNotDamageStructures(t *testing.T) {
	structures := []Structure{
		{ID: 1, Field42: 9, MaxHealth: 9, Col: 5, Row: 6, Width: 1, Height: 1, Attach: 1},
		{ID: 2, Field42: 9, MaxHealth: 9, Col: 5, Row: 6, Width: 2, Height: 1, Attach: 3},
		{ID: 3, Field42: 9, MaxHealth: 9, Col: 8, Row: 8, Width: 1, Height: 1, Attach: 1},
		{ID: 4, Field42: 0xffff, MaxHealth: 9, Col: 5, Row: 6, Width: 1, Height: 1, Attach: 1},
	}
	rule := SpellRule{ID: 8, DamageMin: 9, DamageMax: 9, Damaging: true, Area: true}

	cloud := destructibleWorld(t, structures)
	cloud.applyAreaCells(cellEffect{Spell: 8, Mode: areaModeCloud}, rule, []uint16{cellKey(5, 6)})
	for i, s := range cloud.structures {
		if s.Field42 != structures[i].Field42 {
			t.Errorf("cloud changed structure %d to %d", i, s.Field42)
		}
	}
}

func TestStructureHealthShapeRoundTripsAndMovesTheDigest(t *testing.T) {
	w := destructibleWorld(t, []Structure{{
		ID: 7, Field42: 5, MaxHealth: 9, Col: -2, Row: 6,
		Width: 1, Height: 1, Attach: 1,
	}})
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if got := back.Structures(); len(got) != 1 || got[0] != w.structures[0] {
		t.Fatalf("round trip = %+v, want %+v", got, w.structures)
	}

	other := destructibleWorld(t, []Structure{{
		ID: 7, Field42: 4, MaxHealth: 9, Col: -2, Row: 6,
		Width: 1, Height: 1, Attach: 1,
	}})
	if other.Hash() == w.Hash() {
		t.Fatal("changing structure health did not move the digest")
	}
}
