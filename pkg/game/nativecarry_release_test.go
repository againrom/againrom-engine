package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestReleaseNativeCarryMissionTownColdAndNextMission(t *testing.T) {
	var want mapload.NativeCarryHistory
	var memberID string
	f := currentTown(t, nil, func(f *FrontEnd, id sim.EntityID) {
		e, ok := f.live.world.Entity(id)
		if !ok || e.ActorLoad.Source.Class != 0 {
			t.Fatal("actual hero is not native")
		}
		b := sim.NativeActorBasis{BasePresent: true, BaseKnown: 3, ModifierPresent: true, ModifierKnown: uint64(3) << 44, BodyPresent: true, BodyKnown: true, Body: 37}
		b.Base[0], b.Base[1], b.Base[23] = 17, 9, 201
		b.Modifier[44], b.Modifier[45], b.Modifier[63] = 21, 3, 203
		if err := f.live.world.RestoreNativeActorBases([]sim.NativeActorBasisRecord{{ID: id, Basis: b}}); err != nil {
			t.Fatal(err)
		}
		want = mapload.NativeCarryHistory{Basis: b, Class: e.NativeClass}
		for i, actor := range f.live.mission.ids {
			if actor == id {
				memberID = f.live.mission.party[i].ID
			}
		}
	})
	assertMember := func(f *FrontEnd) {
		t.Helper()
		for _, m := range f.Carried {
			if m.ID != memberID {
				continue
			}
			if m.Carry == nil || m.Carry.NativeHistory == nil || *m.Carry.NativeHistory != want {
				t.Fatal("town lost exact current native history")
			}
			return
		}
		t.Fatal("town lost hero identity")
	}
	assertMember(f)
	cold := currentTownReload(t, currentTownSave(t, f))
	assertMember(cold)
	for _, next := range []*FrontEnd{f, cold} {
		if err := next.App("native carry").OpenMission(next.MissionOpener(next.Town.Chapter())); err != nil {
			t.Fatal(err)
		}
		found := false
		for i, m := range next.live.mission.party {
			if m.ID != memberID {
				continue
			}
			e, ok := next.live.world.Entity(next.live.mission.ids[i])
			if !ok || e.NativeBasis != want.Basis || e.NativeClass != want.Class || e.ActorLoad.Source.Class != 0 {
				t.Fatal("next actual mission changed native history")
			}
			found = true
		}
		if !found {
			t.Fatal("next mission lost hero")
		}
		for n := 0; n < 3; n++ {
			sim.Step(next.live.world, nil)
		}
		raw, _, _ := saveCurrentEffect(t, next)
		if len(raw) == 0 {
			t.Fatal("next SAVE is empty")
		}
		last := releaseFront(t)
		open, town, err := last.RestoreOriginal(raw)
		if err == nil && !town {
			err = last.App("native carry second cold").OpenMission(open)
		}
		if err != nil || town {
			t.Fatal("next mission SAV cold LOAD", err)
		}
		for i, m := range last.live.mission.party {
			if m.ID != memberID {
				continue
			}
			e, ok := last.live.world.Entity(last.live.mission.ids[i])
			if !ok || e.NativeBasis != want.Basis || e.NativeClass != want.Class {
				t.Fatal("second mission cold LOAD lost carried history")
			}
		}
	}
}
