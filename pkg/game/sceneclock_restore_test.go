package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

func TestRestoredMapStartsItsIdleFrameAtTheWorldTick(t *testing.T) {
	m := worldFixtureMap()
	units := worldFixtureUnitSet()
	open := func(w *sim.World) *mapWorld {
		return newMapWorld(w, nil, units, worldFixtureViewer(t, m))
	}
	frame := func(mw *mapWorld) *terrain.StaticFrame {
		t.Helper()
		for _, draw := range mw.entityDraws() {
			if draw.ID == 2 {
				if draw.Frame == nil || draw.Step.X != 0 || draw.Step.Y != 0 {
					t.Fatalf("idle actor frame=%p step=%v", draw.Frame, draw.Step)
				}
				return draw.Frame
			}
		}
		t.Fatal("idle actor absent")
		return nil
	}

	fresh := open(mapload.FromALM(m))
	if fresh.world.Tick() != 0 || fresh.scene != 0 {
		t.Fatalf("fresh map starts at world tick %d, scene %d", fresh.world.Tick(), fresh.scene)
	}
	first := frame(fresh)
	for range 9 {
		fresh.tick()
	}
	before := frame(fresh)
	if before == first {
		t.Fatal("idle fixture did not change frame before the save boundary")
	}
	if fresh.world.Tick() != 9 || fresh.scene != 9 {
		t.Fatalf("before save world tick %d, scene %d", fresh.world.Tick(), fresh.scene)
	}
	raw, err := fresh.world.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var world sim.World
	if err := world.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	cold := open(&world)
	if cold.world.Tick() != fresh.world.Tick() || cold.scene != fresh.scene {
		t.Fatalf("after cold load world tick %d, scene %d; running scene %d", cold.world.Tick(), cold.scene, fresh.scene)
	}
	if got := frame(cold); got != before {
		t.Fatal("cold load restarted the idle actor frame")
	}
	for range 5 {
		fresh.tick()
		cold.tick()
		if fresh.world.Hash() != cold.world.Hash() || frame(fresh) != frame(cold) || fresh.scene != cold.scene {
			t.Fatalf("continuation diverged at world tick %d: scenes %d/%d", fresh.world.Tick(), fresh.scene, cold.scene)
		}
	}
}
