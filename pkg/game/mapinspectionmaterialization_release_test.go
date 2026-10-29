package game

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestReleaseEditorEmptyMaterializationPreservesHumanItems(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	checkHumanMaterialization(t, installedInspector(t))
}

func checkHumanMaterialization(t *testing.T, x *MapInspector) {
	t.Helper()
	maps, humans, spells, npcs := 0, 0, 0, 0
	for _, entry := range x.Maps {
		key := "loose/" + entry.Source
		if entry.FromArchive {
			key = "scenario/" + entry.Source
		}
		doc, err := x.Open(key)
		if err != nil {
			t.Fatal(key, err)
		}
		s := doc.Edits.(*mapInspectionEdits)
		m := mustEditorMap(t, s)
		w0, err := mapload.FromALMWith(m, x.table, mapload.DifficultyNormal)
		if err != nil {
			t.Fatal(key, err)
		}
		maps++
		s.batch = s.model
		raw := s.model.Bytes()
		loot, err := m.Loot()
		if err != nil {
			t.Fatal(err)
		}
		for index, unit := range m.Units {
			r := mapload.Resolve(unit, x.table)
			if !r.Found() || r.Arm == mapload.ArmUnits {
				continue
			}
			if r.Arm == mapload.ArmNPC {
				npcs++
				if _, err := s.SetField("Unit", index, "wear7", 0); err == nil || !bytes.Equal(raw, s.model.Bytes()) {
					t.Fatal(key, index, "NPC Empty was accepted", err)
				}
				continue
			}
			if unit.Flags&0x80 != 0 {
				continue
			}
			humans++
			id := sim.EntityID(index)
			worn0, _ := w0.EquippedItems(id)
			for _, effect := range worn0[0].Effects {
				if effect.Kind == 41 {
					spells++
					break
				}
			}
			var explicit [sim.EquipSlots]bool
			for _, stock := range loot.Records {
				if stock.Owner != uint32(unit.UnitID) || stock.Ground() {
					continue
				}
				for _, element := range stock.Elements {
					if slot, ok := mapload.EquipTarget(data.ItemCode(element.ItemCode()), x.table); ok && element.Field04 != 0 {
						explicit[slot-1] = true
					}
				}
			}
			empty, inherit := 7, 1
			for explicit[empty-1] {
				empty++
				if empty > sim.EquipSlots {
					t.Fatal(key, index, "no non-weapon row slot")
				}
			}
			for explicit[inherit-1] || inherit == empty {
				inherit++
			}
			set := func(slot int, value int64) *sim.World {
				t.Helper()
				if _, err := s.SetField("Unit", index, fmt.Sprintf("wear%d", slot), value); err != nil {
					t.Fatal(key, index, slot, value, err)
				}
				w, err := mapload.FromALMWith(mustEditorMap(t, s), x.table, mapload.DifficultyNormal)
				if err != nil {
					t.Fatal(key, index, err)
				}
				return w
			}
			stage := "Empty"
			check := func(w, baseline *sim.World, cleared int, sameActor bool) {
				t.Helper()
				for _, actor := range baseline.Entities() {
					got, ok := w.Entity(actor.ID)
					if !ok || (actor.ID != id || sameActor) && !reflect.DeepEqual(actor, got) {
						a, b := reflect.ValueOf(actor), reflect.ValueOf(got)
						for i := 0; i < a.NumField(); i++ {
							if a.Field(i).CanInterface() && !reflect.DeepEqual(a.Field(i).Interface(), b.Field(i).Interface()) {
								t.Fatalf("%s %s unit %d actor %d %s changed: %v -> %v", key, stage, index, actor.ID, a.Type().Field(i).Name, a.Field(i), b.Field(i))
							}
						}
						t.Fatal(key, index, "actor changed")
					}
					want, _ := baseline.EquippedItems(actor.ID)
					if actor.ID == id && cleared > 0 {
						want[cleared-1] = sim.ItemInstance{}
					}
					held, _ := w.EquippedItems(actor.ID)
					pack0, _ := baseline.CarriedItems(actor.ID)
					pack, _ := w.CarriedItems(actor.ID)
					if !equalNativeItems(want[:], held[:]) || !equalNativeItems(pack0, pack) {
						t.Fatalf("%s unit %d actor %d holdings changed: before %+v/%+v after %+v/%+v", key, index, actor.ID, want, pack0, held, pack)
					}
				}
			}
			w1 := set(empty, 0)
			check(w1, w0, empty, false)
			w2 := set(inherit, -1)
			stage = "retained Inherit"
			check(w2, w1, 0, true)
			w3 := set(empty, -1)
			stage = "restored Inherit"
			check(w3, w0, 0, true)
			for range 3 {
				if !s.model.CanUndo() {
					break
				}
				if !s.model.Undo() {
					t.Fatal(key, index, "Undo unavailable")
				}
			}
			if s.model.CanUndo() || !bytes.Equal(raw, s.model.Bytes()) {
				t.Fatal(key, index, "Undo changed bytes")
			}
		}
		t.Logf("map=%s ordinary=%d spellWeapons=%d", key, humans, spells)
	}
	t.Logf("maps=%d ordinary=%d spellWeapons=%d NPCEmptyRefused=%d", maps, humans, spells, npcs)
}
