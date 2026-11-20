package sim

import "testing"

// A second-game computer unit below Wimpy flees a hostile within 10 cells (the
// block byte 5 plus 5) whatever its own scan range, and not one at 11. The
// first game keeps the mover's radius.
func TestSecondGameFleeRadiusIsTenWhateverTheScanRange(t *testing.T) {
	for _, tc := range []struct {
		name      string
		second    bool
		scanRange uint8
		gap       int32
		want      bool
	}{
		{"second game, scan range 8, hostile at 9", true, 8, 9, true},
		{"second game, scan range 8, hostile at 11", true, 8, 11, false},
		{"second game, scan range 3, hostile at 9", true, 3, 9, true},
		{"second game, scan range 3, hostile at 11", true, 3, 11, false},
		{"first game, scan range 8, hostile at 5", false, 8, 5, true},
		{"first game, scan range 8, hostile at 6", false, 8, 6, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := secondWithdrawWorld(t, tc.second, 30, 0, 30, tc.gap)
			w.entities[0].ScanRange = tc.scanRange
			regWithdrawalPass(w, nil)
			if got := w.entities[0].HasTarget; got != tc.want {
				t.Fatalf("flees = %t, want %t", got, tc.want)
			}
		})
	}
}

// The back-off radius is the constant 2 whatever the scan range.
func TestSecondGameBackOffRadiusIsTwoWhateverTheScanRange(t *testing.T) {
	for _, gap := range []int32{2, 3} {
		w := secondWithdrawWorld(t, true, 40, 40, 0, gap)
		w.entities[0].ScanRange = 9
		regWithdrawalPass(w, nil)
		if got := w.entities[0].HasTarget; got != (gap == 2) {
			t.Fatalf("hostile at %d: backs off = %t", gap, got)
		}
	}
}

// The Teleport arm lists hostiles within 5 cells. A booked, paid unit below
// Wimpy with its only hostile 6 to 10 cells away passes the 10-cell test and
// takes the idle route: it neither casts nor walks. A hostile at 5 is cast
// away from.
func TestSecondGameTeleportFleeListsFiveCells(t *testing.T) {
	for _, tc := range []struct {
		gap      int32
		wantCast bool
	}{{5, true}, {6, false}, {10, false}} {
		w := teleportWorld(t, true, true, 10, 30, 0, 30)
		w.entities[1].X = w.entities[0].X + tc.gap
		regWithdrawalPass(w, nil)
		got := w.entities[0]
		if (len(w.bookCasts) == 1) != tc.wantCast || (!tc.wantCast && (got.HasTarget || len(w.bookCasts) != 0)) {
			t.Fatalf("hostile at %d: casts %+v, target %t", tc.gap, w.bookCasts, got.HasTarget)
		}
	}
}

// A second-game computer mage with Heal in its book, below Wimpy with a hostile
// in reach and a wounded ally, meets withdraw and heal on one pass: the walk
// replaces the prepared heal, and no Heal is cast while the flee holds.
func TestSecondGameWithdrawOverwritesAPreparedHeal(t *testing.T) {
	w, heal := computerHealWorld(t, true, 1, false, woundedOf(3, healAlliedOwner, 4), woundedOf(5, healHostileOwner, 8))
	m := &w.entities[0]
	m.HP, m.MaxHP, m.Wimpy, m.Withdraw = 20, 100, 30, 0
	moved, casts := false, 0
	for range 40 {
		for _, event := range StepObserved(w, nil) {
			if event.Caster == 1 && event.Spell == heal {
				casts++
			}
		}
		moved = moved || w.entities[0].X != 2 || w.entities[0].Y != 2 || w.entities[0].HasTarget
	}
	t.Logf("moved=%t casts=%d", moved, casts)
	if !moved || casts != 0 {
		t.Fatalf("moved=%t casts=%d, want a walk and no Heal", moved, casts)
	}
}
