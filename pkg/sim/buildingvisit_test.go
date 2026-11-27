package sim

import (
	"reflect"
	"slices"
	"testing"
)

func init() {
	worldMethods = append(worldMethods, "SetBuildingVisits", "VisitsBuildings", "StructureUsable")
	slices.Sort(worldMethods)
	worldWriters = append(worldWriters, "SetBuildingVisits")
	worldArgReaders["StructureUsable"] = []reflect.Value{reflect.ValueOf(Structure{})}
}

func visitWorld(t *testing.T, d ScriptDialect, kind uint16, visits bool) *World {
	t.Helper()
	w := structureUseWorld(t, kind)
	w.game.buildings = newGameProfile(d).buildings
	w.SetBuildingVisits(visits, []EntityID{0})
	return w
}

func TestOnlyAVisitorEntersAShopOrAnInn(t *testing.T) {
	for _, tc := range []struct {
		d    ScriptDialect
		kind uint16
	}{{ScriptROM1, 34}, {ScriptROM2, 34}, {ScriptROM2, 67}} {
		w := visitWorld(t, tc.d, tc.kind, true)
		w.SetBuildingVisits(true, []EntityID{7})
		if got := visitUse(t, w); len(got) != 0 || w.entities[0].OffMap {
			t.Fatalf("dialect %d kind %d: a unit that is no visitor entered: visits %+v off map %v", tc.d, tc.kind, got, w.entities[0].OffMap)
		}
	}
}

func visitUse(t *testing.T, w *World) []BuildingVisit {
	t.Helper()
	var seen []BuildingVisit
	seen = append(seen, StepReported(w, []Command{UseStructure(0, 0)}).Visits...)
	for range 400 {
		if len(seen) != 0 || len(w.StructureUses()) == 0 && w.entities[0].OffMap {
			return seen
		}
		seen = append(seen, StepReported(w, nil).Visits...)
	}
	return seen
}

func reach(a, b int32) int32 {
	if a < 0 {
		a = -a
	}
	if b < 0 {
		b = -b
	}
	return max(a, b)
}

func TestVisitTakesTheUserOffTheMapAndLeaveBringsItBack(t *testing.T) {
	for _, tc := range []struct {
		name  string
		d     ScriptDialect
		kind  uint16
		class VisitClass
		town  int
	}{
		{"first game Shop", ScriptROM1, 34, VisitShop, 0},
		{"first game Shop 2", ScriptROM1, 35, VisitShop, 0},
		{"second game Shop", ScriptROM2, 34, VisitShop, 1},
		{"second game Kaarg Shop", ScriptROM2, 93, VisitShop, 2},
		{"second game Druid Shop", ScriptROM2, 107, VisitShop, 3},
		{"second game Inn", ScriptROM2, 67, VisitInn, 1},
		{"second game Kaarg Inn", ScriptROM2, 100, VisitInn, 2},
		{"second game Druid Inn", ScriptROM2, 113, VisitInn, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := visitWorld(t, tc.d, tc.kind, true)
			got := visitUse(t, w)
			if len(got) != 1 || got[0] != (BuildingVisit{Entity: 0, Structure: 0, Class: tc.class, Town: tc.town}) {
				t.Fatalf("visits %+v, want one of class %d", got, tc.class)
			}
			e := w.entities[0]
			if !e.OffMap {
				t.Fatal("the visitor is still on the map")
			}
			at := [2]int32{e.X, e.Y}
			Step(w, []Command{LeaveBuilding(0)})
			if e = w.entities[0]; e.OffMap || e.X != at[0] || e.Y != at[1] {
				t.Fatalf("left at %d,%d off map %v, want its own cell %v", e.X, e.Y, e.OffMap, at)
			}
		})
	}
}

func TestLeaveStandsOnAFreeCellWithinThreeWhenItsCellIsTaken(t *testing.T) {
	for _, d := range []ScriptDialect{ScriptROM1, ScriptROM2} {
		w := visitWorld(t, d, 34, true)
		if len(visitUse(t, w)) != 1 {
			t.Fatal("no visit")
		}
		e := w.entities[0]
		if _, err := w.CheatSummon(Entity{Owner: 3, HP: 50, MaxHP: 50}, nil, [EquipSlots]ItemInstance{}, e.X, e.Y); err != nil {
			t.Fatalf("the visitor's cell blocks a summon while it is inside: %v", err)
		}
		Step(w, []Command{LeaveBuilding(0)})
		after := w.entities[0]
		if after.OffMap || after.X == e.X && after.Y == e.Y || reach(after.X-e.X, after.Y-e.Y) > 3 {
			t.Fatalf("dialect %d: left at %d,%d off map %v, want a free cell within 3 of %d,%d", d, after.X, after.Y, after.OffMap, e.X, e.Y)
		}
	}
}

func TestChurchAndTheFirstGamesNoInnBuildNoVisit(t *testing.T) {
	for _, tc := range []struct {
		d    ScriptDialect
		kind uint16
	}{{ScriptROM2, 66}, {ScriptROM1, 66}, {ScriptROM1, 67}, {ScriptROM1, 93}} {
		w := visitWorld(t, tc.d, tc.kind, true)
		if got := visitUse(t, w); len(got) != 0 || w.entities[0].OffMap {
			t.Fatalf("dialect %d kind %d: visits %+v off map %v", tc.d, tc.kind, got, w.entities[0].OffMap)
		}
	}
}

func TestVisitsOffLeaveShopsAndInnsAsTheyWere(t *testing.T) {
	w := visitWorld(t, ScriptROM1, 34, false)
	var uses []StructureUse
	var visits []BuildingVisit
	cmds := []Command{UseStructure(0, 0)}
	for range 400 {
		r := StepReported(w, cmds)
		cmds = nil
		uses, visits = append(uses, r.ShopUses...), append(visits, r.Visits...)
	}
	if len(uses) != 1 || len(visits) != 0 || w.entities[0].OffMap {
		t.Fatalf("shop uses %d visits %d off map %v, want one use and no visit", len(uses), len(visits), w.entities[0].OffMap)
	}
	inn := visitWorld(t, ScriptROM2, 67, false)
	Step(inn, []Command{UseStructure(0, 0)})
	if len(inn.StructureUses()) != 0 || inn.StructureUsable(inn.structures[0]) {
		t.Fatal("an Inn is usable on a map whose entry sets no visits")
	}
}
