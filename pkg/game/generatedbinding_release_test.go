package game

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func requireGeneratedTombstonesSAV(t *testing.T, f *FrontEnd, app *ui.App) (*FrontEnd, *ui.App) {
	t.Helper()
	type tombstone struct {
		Class     uint8
		MapUnitID uint16
		State     sim.DeadActorState
		Diary     *sim.SavedDiary
	}
	observe := func(w *sim.World, state *SnapshotSAVDocument) map[uint32]tombstone {
		out := map[uint32]tombstone{}
		byActor := map[sim.EntityID]uint32{}
		for _, d := range w.OriginalDeadActors() {
			if d.Source.Identity == 0 || out[d.Source.Identity].Class != 0 || d.Current.Stage != 5 {
				t.Fatal("fixture lacks unique terminal object identities", d)
			}
			out[d.Source.Identity] = tombstone{Class: (sim.SourceBinding{Class: d.Source.Class}).ActorClass(), MapUnitID: d.Source.MapUnitID, State: d.Current}
			byActor[d.ID] = d.Source.Identity
		}
		terminalBindings, err := currentTerminalActorBindings(state, w)
		if err != nil {
			t.Fatal("current terminal bindings", err)
		}
		for _, terminal := range w.CurrentTerminalActors() {
			object := terminalBindings[terminal.ID]
			if object == 0 || int(object) > len(state.Document.Objects) {
				t.Fatal("current terminal actor lacks its exact SAV object", terminal.ID)
			}
			record := &state.Document.Objects[object-1]
			identity, err := currentTerminalObjectIdentity(state, object)
			if err != nil || identity == 0 || out[identity].Class != 0 {
				t.Fatal("current terminal identity is not unique", err)
			}
			class := uint8(0)
			switch record.Class {
			case "Unit":
				class = 1
			case "Human":
				class = 2
			case "Humanoid":
				class = 3
			}
			if class == 0 {
				t.Fatal("current terminal object has no actor class", record.Class)
			}
			mapUnit, err := savedStructureValue(record, "T08")
			if err != nil {
				t.Fatal("current terminal object has no MapUnitID", err)
			}
			if terminal.HP < -32768 || terminal.HP > 32767 {
				t.Fatal("current terminal health exceeds SAV width", terminal.HP)
			}
			out[identity] = tombstone{Class: class, MapUnitID: uint16(mapUnit), State: sim.DeadActorState{Cell: terminal.Cell, HP: int16(terminal.HP), Stage: terminal.Stage}}
			byActor[terminal.ID] = identity
		}
		for _, d := range w.SavedDiaries() {
			if key := byActor[d.Owner.Actor]; !d.Owner.Player && key != 0 {
				body := out[key]
				d.Owner = sim.SavedDiaryOwner{}
				body.Diary = &d
				out[key] = body
			}
		}
		if len(out) == 0 {
			t.Fatal("no current retired actors")
		}
		for _, e := range w.Entities() {
			if _, retired := out[e.SourceBinding.Identity]; retired {
				t.Fatal("retired actor resurrected", e.ID)
			}
		}
		return out
	}
	tick := f.live.world.Tick()
	for cut := range 2 {
		snapshot, _, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		want := observe(f.live.world, snapshot.SavedDocument)
		if _, err = f.ExportCurrentWorldSave(snapshot, "Terminal projection"); err != nil {
			t.Fatal("current terminal producer", err)
		}
		current, err := currentWorldDocument(snapshot, false, f.Table)
		if err != nil {
			t.Fatal(err)
		}
		path := generatedMissionSave(t, f, app)
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := sav.DecodeDocumentData(raw)
		if err != nil || len(current.Objects) != len(encoded.Objects) {
			t.Fatal("terminal SAV graph population changed", err)
		}
		// Remint changes foreign keys, never the document object indices.
		// Only identity is joined here; expected values came from World above.
		reminted := map[uint32]tombstone{}
		for i := range current.Objects {
			key, _ := savedStructureValue(&current.Objects[i], "Identity")
			if body, ok := want[key]; ok {
				key, err = savedStructureValue(&encoded.Objects[i], "Identity")
				if err != nil || key == 0 || reminted[key].Class != 0 || current.Objects[i].Class != encoded.Objects[i].Class {
					t.Fatal("terminal identity remap is not unique", err)
				}
				reminted[key] = body
			}
		}
		if len(reminted) != len(want) {
			t.Fatal("terminal object lost its current binding")
		}
		cold := releaseFront(t)
		cold.SetDeterministicFrames(true)
		open, town, err := cold.RestoreOriginal(raw)
		if err != nil || town || open == nil {
			t.Fatal("retired mission SAV did not prepare", err)
		}
		next := cold.App("retired generated actors")
		if err = next.OpenMission(open); err != nil {
			t.Fatal(err)
		}
		next.Layout(1024, 768)
		coldSnapshot, _, err := cold.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		if cold.live.world.Tick() != tick || !reflect.DeepEqual(observe(cold.live.world, coldSnapshot.SavedDocument), reminted) {
			t.Fatal("SAV changed the current terminal population", cut)
		}
		var continued sim.World
		worldBytes, err := cold.live.world.MarshalBinary()
		if err != nil {
			t.Fatal("could not marshal the cold terminal world", err)
		}
		if err := continued.UnmarshalBinary(worldBytes); err != nil {
			t.Fatal("could not copy the cold terminal world", err)
		}
		for range 32 {
			sim.Step(&continued, nil)
		}
		if continued.Tick() != tick+32 || !reflect.DeepEqual(observe(&continued, coldSnapshot.SavedDocument), reminted) {
			t.Fatal("terminal population changed across 32 simulation steps", cut)
		}
		tick = cold.live.world.Tick()
		f, app = cold, next
	}
	return f, app
}

func TestReleaseGeneratedBindingsPreserveCurrentState(t *testing.T) {
	for _, mission := range []int{20, 80, 81} {
		t.Run(fmt.Sprint(mission), func(t *testing.T) {
			f := releaseFront(t)
			if mission == 20 {
				f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Campaign witness", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
			} else {
				prepareAcceptedCampaignMission(t, f, mission)
			}
			if err := f.App("current state bindings").OpenMission(f.MissionOpenerWith(mission, f.Carried)); err != nil {
				t.Fatal(err)
			}
			ms := f.live.mission.state
			type current struct {
				ID            sim.EntityID
				Class, TypeID int32
				Profile       sim.CurrentProfileBasis
			}
			observe := func(w *sim.World) []current {
				var actors []current
				for _, e := range w.Entities() {
					actors = append(actors, current{e.ID, e.Class, e.TypeID, e.CurrentProfileBasis})
				}
				return actors
			}
			before, formation := observe(ms.World), ms.World.FormationMode(sim.SelfSlot)
			hash := ms.World.Hash()
			ghost := ms.World.Ghost()
			snapshot, label, err := f.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := f.ExportCurrentSave(snapshot, label); err != nil {
				t.Fatal("current SAVE", err)
			}
			if ms.savedDocument != nil || ms.World.Hash() != hash {
				t.Fatal("SAVE fabricated live provenance or changed current state")
			}
			if !ghost.Raisable() || ms.World.Ghost() != ghost {
				t.Fatal("binding construction lost the installed summon template")
			}
			if got, known := ms.World.CommandFormationMode(sim.SelfSlot); !known || got != formation || !reflect.DeepEqual(observe(ms.World), before) {
				t.Fatal("binding installation replaced current settings, appearance or profile")
			}
			t.Logf("mission%d: before/after formation%d, %d actor class/type/profile values exact; hero %+v", mission, formation, len(before), before[len(before)-1])
		})
	}
}
