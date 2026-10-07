package game

import (
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

// TestTheDrawnOctantAdvancesAcrossAMultiTickTurn is P3's render-level witness
// and remaining-surface item 2's multi-tick reversal (adversarial return,
// section 7): a unit already mid-turn between two OPPOSITE octants (an arc of
// four octant-widths, the largest a turn can name) draws a different idle
// frame on every tick of the interval, never standing at the pre-turn octant
// for more than the interval's own first tick.
//
// Facing 0, DesiredFacing 128 (128 = facingStep*4, the opposite direction),
// RotationSpeed 32 (facingStep): arc 128, so TurnRemaining starts at
// ceil(128/32) = 4, matching what requestFacing itself would have set for
// this arc and rate (pkg/sim/facing.go).
func TestTheDrawnOctantAdvancesAcrossAMultiTickTurn(t *testing.T) {
	const startOct, endOct = 4, 0 // sheetOctant(0) and sheetOctant(128)
	ent := sim.Entity{ID: 1, X: 4, Y: 4, Class: 1, HP: 200, MaxHP: 200,
		Facing: 0, DesiredFacing: 128, TurnRemaining: 4, TurnTotal: 4, RotationSpeed: 32}
	mw := swingWorld(t, swingAnimDesc(), ent)
	art := mw.units.Classes[1]

	wantOct := []int{4, 5, 6, 7, 0} // before any tick, then after each of 4 ticks
	seenOct := make([]int, 0, 5)

	octOf := func() int {
		d := swingDraw(t, mw, 1)
		frame := d.Frame
		for oct := 0; oct < 8; oct++ {
			if frame == art.Frames[swingIdleBase+oct] {
				return oct
			}
		}
		t.Fatalf("drawn frame %p is not any idle octant frame", frame)
		return -1
	}

	seenOct = append(seenOct, octOf())
	for tick := 0; tick < 4; tick++ {
		mw.tick()
		seenOct = append(seenOct, octOf())
	}

	if len(seenOct) != len(wantOct) {
		t.Fatalf("recorded %d octants, want %d", len(seenOct), len(wantOct))
	}
	for i := range wantOct {
		if seenOct[i] != wantOct[i] {
			t.Errorf("step %d: drawn octant %d, want %d (sequence so far %v, want %v)",
				i, seenOct[i], wantOct[i], seenOct, wantOct)
		}
	}
	if seenOct[0] != startOct {
		t.Fatalf("step 0: drawn octant %d, want the pre-turn octant %d", seenOct[0], startOct)
	}
	if seenOct[len(seenOct)-1] != endOct {
		t.Fatalf("final step: drawn octant %d, want the post-turn octant %d", seenOct[len(seenOct)-1], endOct)
	}
	distinct := map[int]bool{}
	for _, o := range seenOct {
		distinct[o] = true
	}
	if len(distinct) < 3 {
		t.Fatalf("the turn drew only %d distinct octants over %d steps: %v; a static octant is exactly round 2's own defect (facing held at its pre-turn value for the whole interval)",
			len(distinct), len(seenOct), seenOct)
	}

	// The hashed Facing itself never advances mid-interval: it holds 0 until
	// the last tick's snap, per advanceTurns' own contract.
	if got := mw.world.Entities()[0].Facing; got != 128 {
		t.Fatalf("after the interval completes, Facing = %d, want 128 (DesiredFacing, written by advanceTurns' snap)", got)
	}
}

// TestEquipAndUnequipKeepTheRequestTimeTurnDuration is the pass-3 P1
// production witness. Each case applies a real item command, lets mapWorld.tick
// run sim.Step and rearm in their production order, then reads the idle frame
// through entityDraws. A positive RotationSpeed change may affect the next
// turn. It must not change the denominator of the active turn already paid for.
func TestEquipAndUnequipKeepTheRequestTimeTurnDuration(t *testing.T) {
	table := eqDefsTable(t)
	item := sim.ItemInstance{Code: eqSwordCode, Kind: 1,
		Effects: []sim.ItemEffect{{Kind: 18, Operand: 16}}}

	tests := []struct {
		name                string
		increase            bool
		remaining, total    uint8
		beforeOct, afterOct int
		wantRate            int32
	}{
		{"increase at first tick", true, 8, 8, 4, 5, 32},
		{"increase at middle tick", true, 4, 8, 6, 7, 32},
		{"increase at final tick", true, 1, 8, 0, 0, 32},
		{"decrease at first tick", false, 4, 4, 4, 5, 16},
		{"decrease at middle tick", false, 2, 4, 6, 7, 16},
		{"decrease at final tick", false, 1, 4, 7, 0, 16},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			const id = sim.EntityID(7)
			rate := int32(32)
			stock := sim.Stock{ID: id, EquippedItems: [sim.EquipSlots]sim.ItemInstance{item}}
			if tc.increase {
				rate = 16
				stock = sim.Stock{ID: id, ItemInstances: []sim.ItemInstance{item}}
			}
			ent := sim.Entity{ID: id, X: 4, Y: 4, Class: 1, HP: 200, MaxHP: 200,
				Facing: 0, DesiredFacing: 128, TurnRemaining: tc.remaining,
				TurnTotal: tc.total, RotationSpeed: rate}
			w, err := sim.NewStockedWorld(1, sim.Bounds{Width: swingW, Height: swingH},
				sim.ModeCanonical, sim.Terrain{}, []sim.Entity{ent}, nil,
				sim.Relations{}, nil, []sim.Stock{stock})
			if err != nil {
				t.Fatalf("NewStockedWorld: %v", err)
			}
			mw := equipMission(t, w, id, eqHero(), nil, table)
			art := worldFixtureArt(16, 16, 8, 14, 4, 4, 64)
			art.Anim, art.Corpse = swingAnimDesc(), art
			mw.units = &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{1: art}}

			octant := func() int {
				d := swingDraw(t, mw, id)
				for oct := 0; oct < 8; oct++ {
					if d.Frame == art.Frames[swingIdleBase+oct] {
						return oct
					}
				}
				t.Fatalf("drawn frame %p is not an idle octant frame", d.Frame)
				return -1
			}

			beforeHash := w.Hash()
			beforeProgress := int(gaEntity(t, w, id).DrawnFacing())
			if got := octant(); got != tc.beforeOct {
				t.Fatalf("before equipment change: octant %d, want %d", got, tc.beforeOct)
			}
			if w.Hash() != beforeHash {
				t.Fatal("the pre-command render call changed the canonical hash")
			}

			if tc.increase {
				mw.enqueueEquip(0)
			} else {
				mw.enqueueUnequip(0)
			}
			mw.tick()

			got := gaEntity(t, w, id)
			if got.RotationSpeed != tc.wantRate {
				t.Fatalf("RotationSpeed = %d, want %d after the equipment recompute", got.RotationSpeed, tc.wantRate)
			}
			wantRemaining := tc.remaining - 1
			wantTotal, wantFacing := tc.total, uint8(0)
			if wantRemaining == 0 {
				wantTotal, wantFacing = 0, 128
			}
			if got.TurnRemaining != wantRemaining || got.TurnTotal != wantTotal || got.Facing != wantFacing {
				t.Fatalf("canonical turn after equipment change = facing %d remaining/total %d/%d, want %d/%d/%d",
					got.Facing, got.TurnRemaining, got.TurnTotal, wantFacing, wantRemaining, wantTotal)
			}

			afterHash := w.Hash()
			afterProgress := int(got.DrawnFacing())
			if afterProgress < beforeProgress || afterProgress > 128 {
				t.Fatalf("drawn clockwise progress moved from %d to %d, want monotone progress in [0,128]",
					beforeProgress, afterProgress)
			}
			if gotOct := octant(); gotOct != tc.afterOct {
				t.Fatalf("after equipment change: octant %d, want %d", gotOct, tc.afterOct)
			}
			if w.Hash() != afterHash {
				t.Fatal("the post-command render call changed the canonical hash")
			}
		})
	}
}
