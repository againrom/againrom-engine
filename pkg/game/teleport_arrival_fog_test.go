package game

import (
	"testing"

	"againrom/pkg/sim"
)

func TestTeleportArrivalRevealsDestinationInTheArrivalTick(t *testing.T) {
	for _, blocked := range []bool{false, true} {
		t.Run(map[bool]string{false: "arrival", true: "blocked"}[blocked], func(t *testing.T) {
			caster := sim.Entity{ID: 1, X: 12, Y: 22, HP: 100, MaxHP: 100, Owner: sim.SelfSlot,
				TokenSize: 1, Mind: 100, Mana: 200, MaxMana: 200, KnownSpells: 1 << 26,
				ScanRange: 4, AttackCharge: 1, AttackRelax: 1}
			caster.Skill[5] = 30
			block := make([]byte, 45*45)
			if blocked {
				block[22*45+32] = 1
			}
			w, err := sim.NewStockedSpelledWorld(1, fogTestBounds, sim.ModeCanonical,
				sim.Terrain{Block: block}, []sim.Entity{caster}, nil, sim.Relations{}, nil, nil,
				[]sim.SpellRule{{ID: 26, ManaCost: 60, School: 5, MaxRange: 8, Defensive: true}})
			if err != nil {
				t.Fatal(err)
			}
			mw := newMapWorld(w, nil, nil, fogTestViewer(t))
			mw.attackOrCast(1, 0, 26, 32, 22, true)
			paid := false
			for n := 0; n < fogPeriod-1; n++ {
				mw.tick()
				after := teleportCaster(t, w)
				if after.Mana == caster.Mana {
					continue
				}
				paid = true
				wantX := int32(32)
				if blocked {
					wantX = caster.X
				}
				if after.X != wantX || after.Y != 22 || after.Mana != 140 {
					t.Fatal("unexpected paid cast result", after)
				}
				if !blocked {
					for i, lit := range w.Sight(sim.SelfSlot) {
						if (mw.fog.visible[i] != 0) != (lit != 0) || (lit != 0 && mw.fog.explored[i] == 0) {
							t.Fatalf("tick%d: arrival left cell%d visibility/exploration stale", w.Tick(), i)
						}
					}
				} else if mw.fog.visible[22*45+32] != 0 || mw.fog.explored[22*45+32] != 0 {
					t.Fatal("blocked Teleport revealed its rejected destination")
				}
				break
			}
			if !paid {
				t.Fatal("fixture did not cast before the periodic fog refresh")
			}
		})
	}
}
