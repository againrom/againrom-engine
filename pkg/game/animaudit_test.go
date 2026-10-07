package game_test

import (
	"slices"
	"testing"

	"againrom/pkg/game"
	"againrom/pkg/render/terrain"
)

// The fixture class ids, deliberately sparse and deliberately UNSORTED as map
// literals go: the audit must answer ascending by id whatever order the map
// iterates in.
const (
	auditFullID  = 2  // the spec's example class over a full 54-frame sheet
	auditShortID = 5  // the same descriptor over a DELIBERATELY SHORT 20-frame sheet
	auditBothID  = 8  // Flip 0, both gates on, an exact-fit 64-frame sheet
	auditBareID  = 11 // a frameless entry with the absent-everywhere descriptor
	auditNilID   = 13 // an entry present but nil: no class, no row
)

// auditFrames is a sheet as the audit sees one — its count. Every element is
// nil so a sweep that dereferenced a frame would panic.
func auditFrames(n int) []*terrain.StaticFrame { return make([]*terrain.StaticFrame, n) }

// specAnim is the spec's worked-example descriptor as hand literals (0024
// "I/O example"): Flip 1 -> (S, D) = (9, 5); MB 1, MV 2, AT 2, DY 2, BN 2,
// ID 0 -> MoveBase 9, AttackBase 9+5*3 = 24, DyingBase 9+5*5 = 34, TailBase
// 9+5*7 = 44, MoveSlot 3, MoveWind 1, IdleSlot 0, Total 9+5*(7+max(2,0)) =
// 54; Move Time [2,1] / Frame [0,1] -> track [0,0,1], period 3; MV 2 gates
// move on, ID 0 gates idle off.
func specAnim() terrain.UnitAnim {
	return terrain.UnitAnim{
		S: 9, D: 5,
		MoveBase: 9, AttackBase: 24, DyingBase: 34, TailBase: 44,
		MoveSlot: 3, MoveWind: 1, IdleSlot: 0,
		Total:     54,
		MoveTrack: []int{0, 0, 1},
		MoveOK:    true,
	}
}

// bothAnim is the Flip 0 layout with BOTH gates on, as hand literals:
// (S, D) = (16, 8); MB 0, MV 2, AT 1, DY 1, BN 0, ID 2 -> MoveBase 16,
// AttackBase 16+8*2 = 32, DyingBase 16+8*3 = 40, TailBase 16+8*4 = 48,
// MoveSlot 2, MoveWind 0, IdleSlot 2, Total 16+8*(4+max(0,2)) = 64; move
// Time [1,1] / Frame [0,1] -> track [0,1], period 2; idle Time [1,2] /
// Frame [0,1] -> track [0,1,1], period 3.
func bothAnim() terrain.UnitAnim {
	return terrain.UnitAnim{
		S: 16, D: 8,
		MoveBase: 16, AttackBase: 32, DyingBase: 40, TailBase: 48,
		MoveSlot: 2, MoveWind: 0, IdleSlot: 2,
		Total:     64,
		MoveTrack: []int{0, 1},
		IdleTrack: []int{0, 1, 1},
		MoveOK:    true,
		IdleOK:    true,
	}
}

// bareAnim is the descriptor of a class whose chain sets no anim scalar, as
// hand literals over the T1 defaults at Flip 1: MB = MV = AT = DY = BN = -1,
// ID = 0 -> MoveBase 9, AttackBase 9+5*(-2) = -1, DyingBase 9+5*(-3) = -6,
// TailBase 9+5*(-4) = -11, MoveSlot -2, MoveWind -1, IdleSlot 0, Total
// 9+5*(-4+max(-1,0)) = -11 — nothing clamped — both tracks empty, both gates
// off.
func bareAnim() terrain.UnitAnim {
	return terrain.UnitAnim{
		S: 9, D: 5,
		MoveBase: 9, AttackBase: -1, DyingBase: -6, TailBase: -11,
		MoveSlot: -2, MoveWind: -1, IdleSlot: 0,
		Total: -11,
	}
}

// auditSet is the hand-assembled bundle: two classes sharing the spec-example
// descriptor over sheets of different lengths, one class with both gates on,
// one frameless, one nil.
func auditSet() *terrain.UnitSet {
	return &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{
		auditFullID:  {Width: 32, Height: 40, CenterX: 16, CenterY: 36, Frames: auditFrames(54), Anim: specAnim()},
		auditShortID: {Width: 32, Height: 40, CenterX: 16, CenterY: 36, Frames: auditFrames(20), Anim: specAnim()},
		auditBothID:  {Width: 48, Height: 56, CenterX: 24, CenterY: 50, Frames: auditFrames(64), Anim: bothAnim()},
		auditBareID:  {Width: 16, Height: 16, CenterX: 8, CenterY: 14, Anim: bareAnim()},
		auditNilID:   nil,
	}}
}

// TestUnitAnimAudit — SC-9's unit half: the rows over both layouts, both
// states, all 8 octants and every track step, the in-range/guarded split as
// hand literals.
func TestUnitAnimAudit(t *testing.T) {
	t.Run("the rows, ascending by id, the split hand-computed", func(t *testing.T) {
		got := game.UnitAnimAudit(auditSet())

		// Each row below is worked out BY HAND from the descriptor literals
		// above; the domain is 2 states * 8 octants * one full period of the
		// longer track.
		//
		// auditFullID — period 3, domain 48. Moving: index 9 + slot*3 + 1 +
		// track[step], slots per octant at D 5 are 0,1,2,3,4,3,2,1, track
		// values 0,0,1 -> per-slot indices {10,10,11} up to {22,22,23}, all
		// under 54: 24 in range. Idle fails its gate -> standing at S 9:
		// frames 0,2,4,6,8,6,4,2, all under 54: 24 in range. 48/0.
		//
		// auditShortID — the same 24 + 24 selections against 20 frames.
		// Moving per octant: oct 0 {10,10,11}, oct 1 {13,13,14}, oct 2
		// {16,16,17}, oct 3 {19,19,20}, oct 4 {22,22,23}, oct 5 {19,19,20},
		// oct 6 {16,16,17}, oct 7 {13,13,14}; an index of 20 or more guards,
		// so octs 3 and 5 lose one each and oct 4 all three: 19 in range, 5
		// guarded. Standing: 0..8 all under 20, 24 in range. 43/5.
		//
		// auditBothID — period 3 (the idle track's), domain 48. Moving: 16 +
		// oct*2 + track[tick mod 2], ticks 0,1,2 -> values 0,1,0, top index
		// 16+14+1 = 31 < 64: 24 in range. Idle: 48 + oct*2 + track[tick mod
		// 3] -> values 0,1,1, top index 48+14+1 = 63 — exactly the last
		// frame of the 64-sheet: 24 in range. 48/0.
		//
		// auditBareID — no track, period 1, domain 16. Every selection is
		// standing (frames 0..8), and a sheet of NO frames guards them all:
		// [0, 0) contains nothing. 0/16.
		//
		// auditNilID contributes NO row: a nil entry has no class to sweep.
		want := []game.UnitAnimRow{
			{ID: auditFullID, Predicted: 54, Frames: 54, InRange: 48, Guarded: 0},
			{ID: auditShortID, Predicted: 54, Frames: 20, InRange: 43, Guarded: 5},
			{ID: auditBothID, Predicted: 64, Frames: 64, InRange: 48, Guarded: 0},
			{ID: auditBareID, Predicted: -11, Frames: 0, InRange: 0, Guarded: 16},
		}
		if !slices.Equal(got, want) {
			t.Errorf("UnitAnimAudit rows:\n got %+v\nwant %+v", got, want)
		}
	})

	t.Run("with no bundle there are no rows", func(t *testing.T) {
		if got := game.UnitAnimAudit(nil); len(got) != 0 {
			t.Errorf("a nil set audits to %+v, want no rows", got)
		}
		if got := game.UnitAnimAudit(&terrain.UnitSet{}); len(got) != 0 {
			t.Errorf("a zero set audits to %+v, want no rows", got)
		}
	})
}
