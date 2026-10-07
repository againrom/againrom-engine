package main

import (
	"fmt"
	"strings"
	"testing"

	"againrom/pkg/game"
	"againrom/pkg/sim"
)

func TestTakeWaitsForArrivalWithinTheRequestedTickBudget(t *testing.T) {
	for _, ticks := range []int{15, 16} {
		for _, alreadyClaimed := range []bool{false, true} {
			t.Run(fmt.Sprintf("ticks-%d/claimed-%t", ticks, alreadyClaimed), func(t *testing.T) {
				w, err := sim.NewLootWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, sim.Terrain{},
					[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 20, MaxHP: 20, Owner: sim.SelfSlot, Speed: 16, Facing: 64, DesiredFacing: 64}}, nil, sim.Relations{},
					[]sim.Sack{{X: 4, Y: 3, Gold: 500}})
				if err != nil {
					t.Fatal(err)
				}
				budget := ticks
				if alreadyClaimed {
					sim.Step(w, []sim.Command{sim.MoveTo(7, sim.CellPoint{X: 4, Y: 3})})
					budget--
				}
				got := pickUpAt(&tracer{}, &game.Mission{World: w}, 7, 4, 3, budget)
				if w.Tick() != uint64(ticks) {
					t.Fatalf("requested %d total payments, spent %d: %s", ticks, w.Tick(), got)
				}
				if ticks == 15 {
					if !strings.HasPrefix(got, "stopped at ") || len(w.Sacks()) != 1 || w.Purse(sim.SelfSlot) != 0 {
						t.Fatal("short budget took a sack or lost its bounded refusal", got, w.Sacks(), w.Purse(sim.SelfSlot))
					}
				} else if got != "took it" || len(w.Sacks()) != 0 || w.Purse(sim.SelfSlot) != 500 {
					t.Fatal("complete budget failed to take exactly once", got, w.Sacks(), w.Purse(sim.SelfSlot))
				}
			})
		}
	}
}
