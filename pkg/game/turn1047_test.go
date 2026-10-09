package game

import (
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

func TestTheDrawnOctantAdvancesAcrossAMultiTickTurn(t *testing.T) {
	actor := sim.Entity{ID: 1, X: 4, Y: 4, Class: 1, HP: 200, MaxHP: 200,
		Speed: 16, RotationSpeed: 32}
	mw := swingWorld(t, turnStandingAnim(), actor)
	mw.enqueue(1, 4, 5)
	for tick, frame := range []int{10, 12, 14, 0} {
		mw.tick()
		draw := swingDraw(t, mw, actor.ID)
		if draw.Frame != draw.Art.Frames[frame] || draw.Mirror {
			t.Errorf("tick %d: drawn frame/mirror %d/%v, want standing %d/false", tick+1,
				turnFrameIndex(t, draw.Art, draw.Frame), draw.Mirror, frame)
		}
	}
	if got := gaEntity(t, mw.world, actor.ID).Facing; got != 128 {
		t.Fatalf("after four sub-ticks: facing %d, want 128", got)
	}
}

func turnStandingAnim() terrain.UnitAnim {
	return terrain.UnitAnim{S: 16, D: 8, MoveBase: 16, MoveSlot: 1,
		MoveTrack: []int{0}, MoveOK: true, TailBase: 24, IdleSlot: 1,
		IdleTrack: []int{0}, IdleOK: true}
}

func TestEquipAndUnequipKeepTheTurnMessageFrameProgress(t *testing.T) {
	table := eqDefsTable(t)
	item := sim.ItemInstance{Code: eqSwordCode, Kind: 1,
		Effects: []sim.ItemEffect{{Kind: 18, Operand: 16}}}
	for _, tc := range []struct {
		name                    string
		increase                bool
		remaining, total        uint8
		beforeFrame, afterFrame int
		wantRate                int32
	}{
		{"increase at first tick", true, 8, 8, 8, 9, 32},
		{"increase at middle tick", true, 4, 8, 12, 13, 32},
		{"increase at final tick", true, 1, 8, 15, 0, 32},
		{"decrease at first tick", false, 4, 4, 8, 10, 16},
		{"decrease at middle tick", false, 2, 4, 12, 14, 16},
		{"decrease at final tick", false, 1, 4, 14, 0, 16},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const id = sim.EntityID(7)
			rate := int32(32)
			stock := sim.Stock{ID: id, EquippedItems: [sim.EquipSlots]sim.ItemInstance{item}}
			if tc.increase {
				rate = 16
				stock = sim.Stock{ID: id, ItemInstances: []sim.ItemInstance{item}}
			}
			drawn := uint8(128 * int(tc.total-tc.remaining) / int(tc.total))
			ent := sim.Entity{ID: id, X: 4, Y: 4, Class: 1, HP: 200, MaxHP: 200,
				Facing: drawn, DesiredFacing: 128, TurnRemaining: tc.remaining,
				TurnTotal: tc.total, RotationSpeed: rate,
				TurnState: sim.TurnState{Present: true, Active: true, Drawn: drawn,
					DrawTarget: 8, DrawRemaining: tc.remaining}}
			w, err := sim.NewStockedWorld(1, sim.Bounds{Width: swingW, Height: swingH},
				sim.ModeCanonical, sim.Terrain{}, []sim.Entity{ent}, nil,
				sim.Relations{}, nil, []sim.Stock{stock})
			if err != nil {
				t.Fatal(err)
			}
			mw := equipMission(t, w, id, eqHero(), nil, table)
			art := worldFixtureArt(16, 16, 8, 14, 4, 4, 64)
			art.Anim, art.Corpse = turnStandingAnim(), art
			mw.units = &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{1: art}}
			assertFrame := func(frame int) {
				before := w.Hash()
				draw := swingDraw(t, mw, id)
				if draw.Frame != art.Frames[frame] || draw.Mirror {
					t.Fatalf("drawn frame/mirror %d/%v, want standing %d/false",
						turnFrameIndex(t, art, draw.Frame), draw.Mirror, frame)
				}
				if w.Hash() != before {
					t.Fatal("drawing changed the simulation hash")
				}
			}
			assertFrame(tc.beforeFrame)
			if tc.increase {
				mw.enqueueEquip(0)
			} else {
				mw.enqueueUnequip(0)
			}
			mw.tick()
			got := gaEntity(t, w, id)
			if got.RotationSpeed != tc.wantRate {
				t.Fatalf("rotation speed %d, want %d", got.RotationSpeed, tc.wantRate)
			}
			if got.TurnState.DrawRemaining != tc.remaining-1 || got.TurnState.DrawTarget != 8 || got.TurnTotal != tc.total {
				t.Fatalf("turn message remaining/target/total %d/%d/%d, want %d/8/%d",
					got.TurnState.DrawRemaining, got.TurnState.DrawTarget, got.TurnTotal, tc.remaining-1, tc.total)
			}
			assertFrame(tc.afterFrame)
		})
	}
}
