package game

import (
	"bytes"
	"reflect"
	"testing"

	"againrom/pkg/sim"
)

// LiveAdvance projects to the viewer on its last tick only. The run it makes
// must equal the run that projects every tick: same World hash, same written
// SAV, same first-seen-dead stamps, same viewer population.
func TestReleaseLiveAdvanceProjectsOnceAndKeepsEveryTickState(t *testing.T) {
	_, source := groundCorpusFile(t, "2026-08-02/game0000.sav", "3407c9fce2ac9ce091871a17b0c08013cf2d1e1849396c6142b1a2257aec707d")
	run := func(batched bool) *FrontEnd {
		f := releaseFront(t)
		f.SetDeterministicFrames(true)
		open, town, err := f.RestoreOriginal(source)
		if err != nil || town {
			t.Fatalf("source LOAD town=%t error=%v", town, err)
		}
		if err := f.App("projection").OpenMission(open); err != nil {
			t.Fatal(err)
		}
		advance := func(n int) {
			if batched {
				f.LiveAdvance(n)
				return
			}
			for i := 0; i < n; i++ {
				f.LiveAdvance(1)
			}
		}
		advance(60)
		party := map[sim.EntityID]bool{}
		for _, id := range f.live.mission.ids {
			party[id] = true
		}
		killed := 0
		for _, e := range f.live.world.Entities() {
			if killed == 5 {
				break
			}
			if party[e.ID] || e.Owner == sim.SelfSlot || e.HP <= 0 || e.Decay != sim.DecayNone {
				continue
			}
			f.LiveKill(uint32(e.ID))
			killed++
		}
		if killed != 5 {
			t.Fatalf("killed %d actors, want 5", killed)
		}
		advance(700)
		return f
	}
	every, batched := run(false), run(true)
	if len(every.live.died) == 0 {
		t.Fatal("no first-seen-dead stamp recorded: the witness exercises nothing")
	}
	if a, b := every.live.world.Hash(), batched.live.world.Hash(); a != b {
		t.Fatalf("World hash %016x with a projection every tick, %016x batched", a, b)
	}
	if !reflect.DeepEqual(every.live.died, batched.live.died) {
		t.Fatalf("first-seen-dead stamps differ: %v batched %v", every.live.died, batched.live.died)
	}
	if every.live.scene != batched.live.scene {
		t.Fatalf("scene clock %d batched %d", every.live.scene, batched.live.scene)
	}
	if !bytes.Equal(saveSourceFlagsMission(t, every), saveSourceFlagsMission(t, batched)) {
		t.Fatal("written SAV differs between a projection every tick and a batched run")
	}
	ae := every.live.view.EntityMarkers()
	be := batched.live.view.EntityMarkers()
	as, _ := every.live.view.SackMarkers()
	bs, _ := batched.live.view.SackMarkers()
	if ae != be || as != bs || every.live.view.SpellBolts() != batched.live.view.SpellBolts() {
		t.Fatalf("viewer entities %d sacks %d bolts %d, batched %d, %d and %d", ae, as, every.live.view.SpellBolts(), be, bs, batched.live.view.SpellBolts())
	}
}
