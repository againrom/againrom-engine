package game

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/sim"
)

// overloadOrderFront is the fixture mission with its native hero carrying
// 6000 against a capacity of 431 and a speed modifier of 5 inside Speed 22.
// The claimed derive gives max(17 - 6000/431, 6) + 5 = 11.
func overloadOrderFront(t *testing.T) *FrontEnd {
	t.Helper()
	f := currentPoolFixtureFront(t, 91, 92)
	if err := f.App("overload order").OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	actors := f.live.world.Entities()
	h := &actors[2]
	if !h.Humanoid || h.ActorLoad.Source.Class != 0 || h.Capacity != 431 {
		t.Fatal("fixture hero is not the native Human this test expects", h.Humanoid, h.ActorLoad.Source.Class, h.Capacity)
	}
	h.Speed, h.SpeedModifier, h.Load, h.RotationSpeed = 22, 5, 6000, 11
	h.ActorLoad = sim.ActorLoad{Present: true, OwnWeight: 6000}
	// A worn item's speed effect moves the native basis word with the live
	// modifier (nativeItemEffects); this fixture states both.
	if !h.NativeBasis.ModifierByteKnown(4) || !h.NativeBasis.ModifierByteKnown(5) {
		t.Fatal("fixture hero's basis does not know its modifier speed word")
	}
	binary.LittleEndian.PutUint16(h.NativeBasis.Modifier[4:], 5)
	w, err := sim.NewWorld(1, f.live.world.Bounds(), sim.ModeCanonical, nil, actors)
	if err != nil {
		t.Fatal(err)
	}
	f.live.world, f.live.mission.state.World = w, w
	return f
}

func TestOverloadedNativeHumanSpeedSurvivesSAVE(t *testing.T) {
	f := overloadOrderFront(t)
	raw, doc, _ := saveCurrentEffect(t, f)
	found := 0
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Human" {
			continue
		}
		found++
		speed, err := savedStructureValue(r, "Speed")
		modifier, merr := savedActorRaw(r, "UD4", 64)
		mover, verr := savedActorRaw(r, "U154", 180)
		if err != nil || merr != nil || verr != nil || speed != 11 || binary.LittleEndian.Uint16(modifier[4:]) != 5 || mover[10] != 11 {
			t.Fatalf("SAV speed word %d, modifier word %d, mover byte %d; want 11, 5, 11", speed, binary.LittleEndian.Uint16(modifier[4:]), mover[10])
		}
	}
	if found != 1 {
		t.Fatal("Human records", found)
	}
	cold := openCurrentEffectSave(t, f, raw)
	assertCurrentWorldEqual(t, f.live.world, cold.live.world, "overloaded hero cold LOAD")
	for _, e := range cold.live.world.Entities() {
		if e.ID == 2 && (e.Speed != 22 || e.SpeedModifier != 5 || e.SpeedWord() != 11) {
			t.Fatalf("cold hero speed %d modifier %d word %d; want 22, 5, 11", e.Speed, e.SpeedModifier, e.SpeedWord())
		}
	}
	sim.Step(f.live.world, []sim.Command{sim.MoveTo(2, sim.CellPoint{X: 20, Y: 20})})
	sim.Step(cold.live.world, []sim.Command{sim.MoveTo(2, sim.CellPoint{X: 20, Y: 20})})
	for tick := range 40 {
		sim.Step(f.live.world, nil)
		sim.Step(cold.live.world, nil)
		if f.live.world.Hash() != cold.live.world.Hash() {
			t.Fatalf("tick %d: cold LOAD walks differently", tick)
		}
	}
}
