package game

import (
	"bytes"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestCurrentDyingAdmissionKeepsOrdinaryTimer(t *testing.T) {
	for _, current := range []bool{false, true} {
		f := currentPoolFixtureFront(t, 91)
		if err := f.App("dying admission").OpenMission(f.MissionOpener(10)); err != nil {
			t.Fatal(err)
		}
		actor := &poolFixtureActor{mapID: 91, cell: 0x100f, stage: 1, hp: 65528, maxHP: 31, timer: 7}
		file, err := sav.Open(savedContainer(poolFixtureBody([]*poolFixturePlayer{{groups: [][]*poolFixtureActor{{actor}}}}, nil)))
		if err != nil {
			t.Fatal(err)
		}
		graph, err := file.ActorGraph()
		if err != nil || len(graph.Actors) != 1 || !graph.Actors[0].Dying() {
			t.Fatal("ordinary dying fixture", graph, err)
		}
		a := graph.Actors[0]
		a.CurrentEntity = current
		registry := &originalActorRegistry{actors: []originalActorBinding{{Source: a, ID: 0}}, groups: graph.Groups,
			sources: graph.Actors, byOff: map[int]int{a.Off: 0}}
		ms := f.live.mission.state
		if err := admitOriginalActorRegistry(ms, registry, f.Table); err != nil {
			t.Fatal("dying admission", current, err)
		}
		e := ms.World.Entities()[0]
		if e.ID != 0 || e.HP != -8 || e.MaxHP != 31 || e.Decay != sim.DecayFallen || e.Dwell != 7 {
			t.Fatalf("current=%t changed ordinary dying tuple: %+v", current, e)
		}
		if err := ms.World.ImportOriginalDyingActors([]sim.OriginalDyingActor{{ID: 0, HP: -8, Timer: 7}}); err != nil {
			t.Fatal("ordinary cleanup no longer accepts its construction tuple", current, err)
		}
	}
}

func TestCurrentAdmissionKeepsNativeIDsAcrossUnjoinedALM(t *testing.T) {
	for _, kind := range []string{"unique", "duplicate", "zero", "missing"} {
		t.Run(kind, func(t *testing.T) {
			ids := []uint16{91, 92}
			if kind == "duplicate" {
				ids[1] = ids[0]
			} else if kind == "zero" {
				ids[0], ids[1] = 0, 0
			}
			f := currentPoolFixtureFront(t, ids...)
			if err := f.App("native admission").OpenMission(f.MissionOpener(10)); err != nil {
				t.Fatal(err)
			}
			before := f.live.world.Entities()
			if len(before) != 3 || before[0].ID != 0 || before[0].TokenSize != 0 || before[1].TokenSize != 0 {
				t.Fatal("fixture did not reach native zero-footprint map actors", before)
			}
			s, label, err := f.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := f.ExportCurrentSave(s, label)
			if err != nil {
				t.Fatal(err)
			}
			for _, footprint := range []uint32{0, 3} {
				doc, err := sav.DecodeDocumentData(raw)
				if err != nil {
					t.Fatal(err)
				}
				a, err := readCurrentActions(&doc)
				if err != nil {
					t.Fatal(err)
				}
				var object uint16
				for _, b := range a.Bindings {
					if !b.Structure && !b.Missing && b.ID == 0 {
						object = b.Object
					}
				}
				if object == 0 {
					t.Fatal("native actor zero lacks exact ordinary binding")
				}
				leaf, _, _ := sav.NativeActions(doc.State)
				mustSetValue(&doc.Objects[object-1], "U49", footprint)
				changed, err := sav.EncodeDocumentData(doc)
				if err != nil {
					t.Fatal(err)
				}
				back, err := sav.DecodeDocumentData(changed)
				if err != nil {
					t.Fatal(err)
				}
				unchanged, _, _ := sav.NativeActions(back.State)
				if !bytes.Equal(leaf, unchanged) {
					t.Fatal("ordinary footprint edit changed current policy")
				}
				for cycle := 0; cycle < 2; cycle++ {
					coldIDs := ids
					if kind == "missing" {
						coldIDs = []uint16{801, 802}
					}
					cold := currentPoolFixtureFront(t, coldIDs...)
					open, town, err := cold.RestoreOriginal(changed)
					if err == nil && !town {
						err = cold.App("cold native admission").OpenMission(open)
					}
					if err != nil || town {
						t.Fatalf("footprint=%d cycle=%d: %v", footprint, cycle, err)
					}
					after := cold.live.world.Entities()
					if len(after) != len(before) {
						t.Fatal("native actor population changed", len(after), len(before))
					}
					for i, want := range before {
						got := after[i]
						size := want.TokenSize
						if want.ID == sim.EntityID(0) {
							size = uint8(footprint)
						}
						if got.ID != want.ID || got.MapUnitID != want.MapUnitID || got.TokenSize != size || got.Owner != want.Owner ||
							got.X != want.X || got.Y != want.Y || got.HP != want.HP || got.MaxHP != want.MaxHP {
							t.Fatalf("footprint=%d cycle=%d native actor%d lost exact ID/map ID/size/owner/position/pool", footprint, cycle, want.ID)
						}
					}
					if cycle == 0 {
						next, label, err := cold.Snapshot(true)
						if err != nil {
							t.Fatal(err)
						}
						changed, err = cold.ExportCurrentSave(next, label)
						if err != nil {
							t.Fatal(err)
						}
					}
				}
			}
		})
	}
}
