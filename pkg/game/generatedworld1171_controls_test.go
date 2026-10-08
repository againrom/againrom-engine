package game

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func generatedCampaignEntity(t *testing.T, f *FrontEnd, id sim.EntityID) sim.Entity {
	t.Helper()
	for _, e := range f.live.world.Entities() {
		if e.ID == id {
			return e
		}
	}
	t.Fatalf("current actor %d disappeared", id)
	return sim.Entity{}
}

func assertCurrentWorldEqual(t *testing.T, want, got *sim.World, cut string) {
	t.Helper()
	byID := map[sim.EntityID]sim.Entity{}
	for _, e := range got.Entities() {
		byID[e.ID] = e
	}
	if len(want.Entities()) != len(byID) {
		t.Fatalf("%s: actor population changed", cut)
	}
	for _, a := range want.Entities() {
		b, ok := byID[a.ID]
		if !ok {
			t.Fatalf("%s: current actor %d absent", cut, a.ID)
		}
		x, y := reflect.ValueOf(a), reflect.ValueOf(b)
		for i := 0; i < x.NumField(); i++ {
			if x.Field(i).CanInterface() && !reflect.DeepEqual(x.Field(i).Interface(), y.Field(i).Interface()) {
				t.Fatalf("%s actor%d %s: %v / %v", cut, a.ID, x.Type().Field(i).Name, x.Field(i).Interface(), y.Field(i).Interface())
			}
		}
	}
	if want.Hash() != got.Hash() {
		currentMenuWorldDiagnostics(t, want, got)
		t.Fatalf("%s: complete World hash changed: %x / %x", cut, want.Hash(), got.Hash())
	}
}

func generatedCurrentCold(t *testing.T, raw []byte) *FrontEnd {
	t.Helper()
	f := currentPoolFixtureFront(t, 91, 92)
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal("current ordinary LOAD", town, err)
	}
	if err := f.App("cold generated control").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestGeneratedCurrentWorldOrdinaryActorFields(t *testing.T) {
	for _, field := range []string{"unchanged", "Identity", "RuntimeID", "T0C", "T0E", "U4C"} {
		t.Run(field, func(t *testing.T) {
			f, s := currentCarrierFixture(t)
			id := poolEntity(t, f.live.world, 91).ID
			doc := generatedCurrentDocument(t, f, s)
			actor := generatedActorRecord(t, &doc, id)
			policy, _, err := sav.NativeActions(doc.State)
			if err != nil {
				t.Fatal(err)
			}
			want := map[string]uint32{}
			for _, name := range []string{"Identity", "RuntimeID", "T0C", "T0E", "U4C"} {
				want[name], err = savedStructureValue(actor, name)
				if err != nil {
					t.Fatal(err)
				}
			}
			if field != "unchanged" {
				next := want[field] + 1
				switch field {
				case "Identity":
					next = 0x6a123456
					old := want[field]
					for i := range doc.World.Cells {
						c := &doc.World.Cells[i]
						if c.GroundActor == old {
							c.GroundActor = next
						}
						if c.AirActor == old {
							c.AirActor = next
						}
					}
					for _, index := range doc.Players {
						if index != 0 {
							p := &doc.Objects[index-1]
							if hero, _ := savedStructureValue(p, "Hero"); hero == old {
								savedObjectSetValue(p, "Hero", next)
							}
						}
					}
				case "RuntimeID":
					next = 60123
				case "T0C":
					next = (want[field] + 1) % 2
				case "U4C":
					next = want[field] ^ 4
				}
				if next == want[field] {
					t.Fatal("ordinary control did not change its field")
				}
				savedObjectSetValue(actor, field, next)
				// An actor of an engine-written file loads without a source
				// binding. Its type is World state, and so is its key, which
				// the World's cell record holds; its runtime ID, row selector
				// and face/class come from constructor rules and are not
				// carried.
				if field == "T0E" || field == "Identity" {
					want[field] = next
				}
			}
			afterPolicy, _, err := sav.NativeActions(doc.State)
			if err != nil || !bytes.Equal(policy, afterPolicy) {
				t.Fatal("ordinary edit altered supplementary policy", err)
			}
			raw, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			current := generatedCurrentCold(t, raw)
			if field == "unchanged" {
				assertCurrentWorldEqual(t, f.live.world, current.live.world, "initial ordinary LOAD")
			}
			if e := generatedCampaignEntity(t, current, id); e.SourceBinding != (sim.SourceBinding{}) || field == "T0E" && e.TypeID != int32(want[field]) {
				t.Fatalf("ordinary edit changed native absence or lost current type: %+v", e)
			}
			for cycle := 0; cycle < 2; cycle++ {
				snapshot, _, err := current.Snapshot(true)
				if err != nil {
					t.Fatal(err)
				}
				doc = generatedCurrentDocument(t, current, snapshot)
				actor = generatedActorRecord(t, &doc, id)
				for name, value := range want {
					if got, err := savedStructureValue(actor, name); err != nil || got != value {
						t.Fatalf("cycle%d ordinary %s=%d, want%d: %v", cycle, name, got, value, err)
					}
				}
				raw, err = sav.EncodeDocumentData(doc)
				if err != nil {
					t.Fatal(err)
				}
				cold := generatedCurrentCold(t, raw)
				assertCurrentWorldEqual(t, current.live.world, cold.live.world, fmt.Sprintf("cycle%d checkpoint", cycle))
				before := generatedCampaignEntity(t, current, id).HP
				for tick := 0; tick < 20; tick++ {
					var commands []sim.Command
					if tick == 0 {
						commands = []sim.Command{sim.Damage(id, 1)}
					}
					sim.Step(current.live.world, commands)
					sim.Step(cold.live.world, commands)
					assertCurrentWorldEqual(t, current.live.world, cold.live.world, fmt.Sprintf("cycle%d successor%d", cycle, tick))
					if tick == 0 && generatedCampaignEntity(t, cold, id).HP != before-1 {
						t.Fatal("next real damage command did not execute")
					}
				}
				current = cold
			}
		})
	}
}

func TestGeneratedCurrentWorldMalformedAdmissionIsAtomic(t *testing.T) {
	f, snapshot := currentCarrierFixture(t)
	doc := generatedCurrentDocument(t, f, snapshot)
	before := f.live.world.Hash()
	live, town, units := f.live, f.Town, f.Units
	view := f.live.view.SaveApplication()
	for _, name := range []string{"truncated", "duplicate binding", "missing binding", "wrong object", "duplicate identity"} {
		t.Run(name, func(t *testing.T) {
			candidate, err := sav.CloneDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			a, err := readCurrentActions(&candidate)
			if err != nil || a == nil || len(a.Bindings) < 2 {
				t.Fatal("fixture lacks independent bindings", err)
			}
			switch name {
			case "duplicate binding":
				a.Bindings = append(a.Bindings, a.Bindings[0])
			case "missing binding":
				a.Bindings = a.Bindings[1:]
			case "wrong object":
				a.Bindings[0].Object = candidate.Players[0]
			case "duplicate identity":
				first := generatedActorRecord(t, &candidate, poolEntity(t, f.live.world, 91).ID)
				second := generatedActorRecord(t, &candidate, poolEntity(t, f.live.world, 92).ID)
				key, _ := savedStructureValue(first, "Identity")
				savedObjectSetValue(second, "Identity", key)
			}
			leaf, err := json.Marshal(a)
			if err != nil {
				t.Fatal(err)
			}
			if err := sav.SetNativeActions(&candidate.State, leaf); err != nil {
				t.Fatal(err)
			}
			raw, err := sav.EncodeDocumentData(candidate)
			if err != nil {
				t.Fatal("malformed candidate must reach LOAD", err)
			}
			if name == "truncated" {
				raw = raw[:8]
			}
			if _, _, err := f.RestoreOriginal(raw); err == nil {
				t.Fatal("malformed current document admitted")
			}
			if f.live != live || f.Town != town || f.Units != units || f.live.world.Hash() != before || !reflect.DeepEqual(view, f.live.view.SaveApplication()) {
				t.Fatal("malformed LOAD changed live state")
			}
		})
	}
}
