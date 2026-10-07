package game

import (
	"reflect"
	"testing"

	"againrom/pkg/sim"
)

func TestPlayerRetreat1089PressPartitionAndNativeNextQueue(t *testing.T) {
	for _, separate := range []bool{false, true} {
		var rel sim.Relations
		rel.Set(sim.SelfSlot, 3, 1)
		w, err := sim.NewRelatedWorld(891, sim.Bounds{Width: 40, Height: 40}, sim.ModeCanonical, sim.Terrain{}, []sim.Entity{
			{ID: 10, Owner: sim.SelfSlot, X: 14, Y: 18, HP: 0, MaxHP: 100, DyingTime: 200},
			{ID: 20, Owner: sim.SelfSlot, X: 18, Y: 18, HP: 100, MaxHP: 100, ScanRange: 6},
			{ID: 30, Owner: sim.SelfSlot, X: 18, Y: 20, HP: 100, MaxHP: 100, ScanRange: 6},
			{ID: 40, Owner: 3, X: 22, Y: 18, HP: 100, MaxHP: 100},
		}, nil, rel)
		if err != nil {
			t.Fatal(err)
		}
		mw := &mapWorld{world: w, commanded: make(map[sim.EntityID]bool)}
		want := []sim.Command{
			{Kind: sim.KindGroupRetreat, Entity: 10, Player: sim.SelfSlot, Group: 1},
			{Kind: sim.KindGroupRetreat, Entity: 20, Player: sim.SelfSlot, Group: 1},
			{Kind: sim.KindGroupRetreat, Entity: 30, Player: sim.SelfSlot, Group: 1},
		}
		if separate {
			mw.playerRetreat([]uint32{10})
			mw.playerRetreat([]uint32{20, 30})
			want[1].Group, want[2].Group = 2, 2
		} else {
			mw.playerRetreat([]uint32{10, 20, 30})
		}
		mw.playerRetreat(nil)
		if !reflect.DeepEqual(mw.pending, want) || !mw.commanded[20] || !mw.commanded[30] || w.Tick() != 0 {
			t.Fatalf("wrong press partition separate=%v: %+v", separate, mw.pending)
		}
		raw, err := w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		var back sim.World
		if err := back.UnmarshalBinary(raw); err != nil {
			t.Fatal(err)
		}
		sim.Step(w, mw.pending)
		sim.Step(&back, want)
		for n := 0; n < 23; n++ {
			if w.Hash() != back.Hash() {
				t.Fatal("native next queue/continuation", separate, n)
			}
			sim.Step(w, nil)
			sim.Step(&back, nil)
		}
		for _, e := range w.Entities() {
			if e.ID != 20 && e.ID != 30 {
				continue
			}
			if (e.ActorState == 0x16) != separate || separate && e.X >= 18 {
				t.Fatalf("first-member refusal crossed a press: separate=%v actor=%+v", separate, e)
			}
		}
	}
}
