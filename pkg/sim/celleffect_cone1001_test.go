package sim

import (
	"reflect"
	"sort"
	"testing"
)

func acidStreamRule() SpellRule {
	return SpellRule{ID: 9, School: 2, MaxRange: 3, Area: true, Distribution: distributionStaged}
}

// TestAcidStreamConeOriginatesAtTheAimedCell casts Acid Stream through the
// production path at eight click offsets — one per compass direction, so
// both even and odd orientations (MAGIC-RING-048's two transform families)
// are exercised — and asserts the landed record's own cell and its
// stage-zero footprint sit at the CLICKED cell, not the caster's.
func TestAcidStreamConeOriginatesAtTheAimedCell(t *testing.T) {
	rule := acidStreamRule()
	for _, tc := range []struct {
		name   string
		dx, dy int32
	}{
		{"north", 0, -2},
		{"north-east", 2, -2},
		{"east", 2, 0},
		{"south-east", 2, 2},
		{"south", 0, 2},
		{"south-west", -2, 2},
		{"west", -2, 0},
		{"north-west", -2, -2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			caster := effectMage(1, 20, 20, 1<<9)
			w, err := NewSpelledWorld(0x1001, Bounds{Width: 40, Height: 40}, ModeCanonical, nil,
				[]Entity{caster}, nil, []SpellRule{rule})
			if err != nil {
				t.Fatalf("NewSpelledWorld: %v", err)
			}
			aimX, aimY := 20+tc.dx, 20+tc.dy
			spRunCast(w, Command{Kind: KindCastAt, Entity: 1, X: aimX, Y: aimY, Spell: 9})

			got := w.CellEffects()
			if len(got) != 1 {
				t.Fatalf("Acid Stream landed %d records, want 1", len(got))
			}
			if got[0].X != aimX || got[0].Y != aimY {
				t.Errorf("Acid Stream's own cell = (%d,%d), want the clicked cell (%d,%d) — it landed on the caster's cell (20,20) instead",
					got[0].X, got[0].Y, aimX, aimY)
			}

			facing, ok := facingToward(tc.dx, tc.dy)
			if !ok {
				t.Fatalf("test offset (%d,%d) names no direction", tc.dx, tc.dy)
			}
			want := canonicalCells(w.ringStageCells(cellEffect{Key: cellKey(aimX, aimY), Spell: 9, Direction: facing >> 5}, 0))
			gotCells := make([]uint16, len(got[0].Cells))
			for i, c := range got[0].Cells {
				gotCells[i] = cellKey(c[0], c[1])
			}
			sort.Slice(gotCells, func(i, j int) bool { return gotCells[i] < gotCells[j] })
			if !reflect.DeepEqual(gotCells, want) {
				t.Errorf("Acid Stream stage-zero cells = %v, want %v (offsets from the clicked cell (%d,%d))",
					gotCells, want, aimX, aimY)
			}
		})
	}
}

// TestAcidStreamDoesNotHitItsOwnCaster is the owner's own reported symptom:
// with round 3's anchor on the caster's own cell, stage 0's offset (0,0)
// painted that cell on every cast, so a damaging Acid Stream applied to the
// caster who threw it. The caster here stands two cells west of the aimed
// cell — outside the cone, which points away from him — so a correct
// anchor leaves his health untouched. Reverting the anchor to the caster's
// cell (celleffect.go's landAreaFacing) reddens this test.
func TestAcidStreamDoesNotHitItsOwnCaster(t *testing.T) {
	rule := acidStreamRule()
	rule.Damaging = true
	rule.DamageMin, rule.DamageMax = 10, 10

	caster := effectMage(1, 20, 20, 1<<9)
	w, err := NewSpelledWorld(0x1003, Bounds{Width: 40, Height: 40}, ModeCanonical, nil,
		[]Entity{caster}, nil, []SpellRule{rule})
	if err != nil {
		t.Fatalf("NewSpelledWorld: %v", err)
	}
	spRunCast(w, Command{Kind: KindCastAt, Entity: 1, X: 22, Y: 20, Spell: 9})

	ci := indexOfEntity(w.entities, 1)
	if ci < 0 {
		t.Fatal("caster is gone from the roster")
	}
	if got := w.entities[ci].HP; got != 100 {
		t.Errorf("caster HP = %d after casting Acid Stream at a cell within range but outside the cone's own footprint, want 100 (unchanged) — the caster took its own cone", got)
	}
}

// TestMeteorStormStillAnchorsAtTheAimedCell is the round-3 fix's own negative
// control, unaffected by round 4: Meteor Storm falls on the cell the player
// designates and must stay anchored there. If the anchor rule in
// landAreaFacing were written to move any row off the aimed cell, this would
// redden.
func TestMeteorStormStillAnchorsAtTheAimedCell(t *testing.T) {
	rule := SpellRule{ID: 21, School: 4, MaxRange: 10, Area: true, Distribution: distributionStaged}
	caster := effectMage(1, 20, 20, 1<<21)
	w, err := NewSpelledWorld(0x1002, Bounds{Width: 40, Height: 40}, ModeCanonical, nil,
		[]Entity{caster}, nil, []SpellRule{rule})
	if err != nil {
		t.Fatalf("NewSpelledWorld: %v", err)
	}
	spRunCast(w, Command{Kind: KindCastAt, Entity: 1, X: 24, Y: 18, Spell: 21})
	got := w.CellEffects()
	if len(got) != 1 {
		t.Fatalf("Meteor Storm landed %d records, want 1", len(got))
	}
	if got[0].X != 24 || got[0].Y != 18 {
		t.Errorf("Meteor Storm's own cell = (%d,%d), want the clicked cell (24,18) unchanged by the cone fix", got[0].X, got[0].Y)
	}
}
