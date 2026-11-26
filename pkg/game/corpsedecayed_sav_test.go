package game

import (
	"encoding/json"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func corpseDecayedFixtureFront(t *testing.T, second bool, extra ...sim.Entity) *FrontEnd {
	t.Helper()
	if !second {
		f := castOrderFront(t)
		actors := append(f.live.world.Entities()[:1], extra...)
		for i := range actors {
			if actors[i].Capacity == 0 {
				actors[i].Capacity = data.UnitCapacity()
			}
			if actors[i].TokenSize == 0 {
				actors[i].TokenSize = 1
			}
		}
		w, err := sim.NewSpelledWorld(1, f.live.world.Bounds(), sim.ModeCanonical, nil, actors, nil, mapload.SpellRules(f.Table))
		if err != nil {
			t.Fatal(err)
		}
		f.live.world, f.live.mission.state.World = insertFixtureRuntimeIDs(w), w
		return f
	}
	f, app, screen, _ := secondBagFrontOf(t, nil, secondMissionContent{})
	secondBagEnter(t, f, app, screen)
	script, err := sim.NewROM2Script(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	var stock []sim.Stock
	baseEnts := f.live.world.Entities()
	for _, e := range baseEnts {
		pack, _ := f.live.world.CarriedStacks(e.ID)
		items, _ := f.live.world.CarriedItems(e.ID)
		worn, _ := f.live.world.EquippedItems(e.ID)
		for i := range pack {
			pack[i].ObjectID = 0
		}
		for i := range items {
			items[i].ObjectID = 0
		}
		for i := range worn {
			worn[i].ObjectID = 0
		}
		stock = append(stock, sim.Stock{ID: e.ID, OrderedStacks: pack, ItemInstances: items, EquippedItems: worn})
	}
	actors := append(baseEnts, extra...)
	w, err := sim.NewStructuredWorld(53, f.live.world.Bounds(), sim.ModeCanonical, sim.Terrain{}, actors, script, sim.Relations{}, nil, stock, nil, sim.GhostTemplate{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	mapload.BindSourceDerive(w)
	w = insertFixtureRuntimeIDs(w)
	f.live.world, f.live.mission.state.World = w, w
	f.live.mission.state.savedDocument = nil
	return f
}

func exportAndReloadCorpseSAV(t *testing.T, f *FrontEnd, second bool, mutate func(*currentActionData)) (*currentActionData, *FrontEnd, error) {
	t.Helper()
	snapshot, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	if second {
		snapshot.SavedDocument = nil
		snapshot.Residue.VisualIdentities, snapshot.Residue.VisualNext = nil, 0
	}
	raw, err := f.ExportCurrentSave(snapshot, "corpse decayed SAV")
	if err != nil {
		t.Fatal(err)
	}
	doc := engineSaveDocument(t, raw)
	a, err := readCurrentActions(&doc)
	if err != nil || a == nil {
		t.Fatalf("readCurrentActions: a=%v err=%v", a, err)
	}
	if mutate != nil {
		mutate(a)
		encoded, err := json.Marshal(a)
		if err != nil {
			t.Fatal(err)
		}
		if err := sav.SetNativeActions(&doc.State, encoded); err != nil {
			t.Fatal(err)
		}
		engineBytes, err := sav.EncodeDocumentData(doc)
		if err != nil {
			t.Fatal(err)
		}
		if second {
			outer, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			outer, err = sav.StoreEngineDocument(outer, engineBytes)
			if err != nil {
				t.Fatal(err)
			}
			raw, err = sav.EncodeDocumentData(outer)
			if err != nil {
				t.Fatal(err)
			}
		} else {
			raw = engineBytes
		}
	}
	var cold *FrontEnd
	if second {
		cold = secondMissionFront(t)
		cold.Table = f.Table
	} else {
		cold = currentPoolFixtureFront(t, 91, 92)
		cold.Table, cold.Humans = f.Table, f.Humans
	}
	opener, town, err := cold.RestoreOriginal(raw)
	if err == nil && !town {
		err = cold.App("corpse decayed cold LOAD").OpenMission(opener)
	}
	if town {
		t.Fatal("unexpected town restore")
	}
	return a, cold, err
}

func totalSackGold(w *sim.World) (total uint32) {
	for _, s := range w.Sacks() {
		total += s.Gold
	}
	return total
}

// A body felled to exactly -10 writes CorpseDecayed into SAV through the one
// SAV producer, reloads with the same state and strict decay ladder, and drops
// loot once with no second gold roll after load in both games.
func TestSAVCorpseDecayedAtMinusTenDropsLootOnce(t *testing.T) {
	for _, second := range []bool{false, true} {
		name := "first"
		if second {
			name = "second"
		}
		t.Run(name, func(t *testing.T) {
			creature := sim.Entity{
				ID: 101, X: 14, Y: 14, HP: 10, MaxHP: 10, Class: 35, TypeID: 0x41,
				Owner: 2, TokenSize: 1, Speed: 10,
				GoldChance: 101, TreasureMin: 37, TreasureMax: 0,
			}
			f := corpseDecayedFixtureFront(t, second, creature)
			id := creature.ID
			sim.Step(f.live.world, []sim.Command{sim.Damage(id, 20)})
			body, ok := f.live.world.Entity(id)
			if !ok || body.HP != -10 || body.Decay != sim.DecayFallen || !body.CorpseDecayed {
				t.Fatalf("after lethal blow to -10: ok=%v hp=%d decay=%d decayed=%v, want -10/1/true", ok, body.HP, body.Decay, body.CorpseDecayed)
			}
			if g := totalSackGold(f.live.world); g != 37 {
				t.Fatalf("gold after first drop = %d, want 37", g)
			}
			a, cold, err := exportAndReloadCorpseSAV(t, f, second, nil)
			if err != nil {
				t.Fatalf("SAV reload failed: %v", err)
			}
			if !a.Values[id].CorpseDecayed {
				t.Fatalf("SAV AgainromActions did not write CorpseDecayed=true for actor %d", id)
			}
			loaded, ok := cold.live.world.Entity(id)
			if !ok || loaded.HP != -10 || loaded.Decay != sim.DecayFallen || !loaded.CorpseDecayed {
				t.Fatalf("reloaded body: ok=%v hp=%d decay=%d decayed=%v, want -10/1/true", ok, loaded.HP, loaded.Decay, loaded.CorpseDecayed)
			}
			for range 64 {
				sim.Step(cold.live.world, nil)
			}
			if e, ok := cold.live.world.Entity(id); !ok || e.HP >= -10 || e.Decay != sim.DecayBones || !e.CorpseDecayed {
				t.Fatal("loaded body did not advance to bones")
			}
			if g := totalSackGold(cold.live.world); g != 37 {
				t.Errorf("gold after reload and decay = %d, want 37 (no second roll)", g)
			}
			// The saved flag alone must guard the drop even with payable roll columns.
			_, cold, err = exportAndReloadCorpseSAV(t, f, second, func(a *currentActionData) {
				v := a.Values[id]
				v.SuppressCorpseLoot = false
				v.GoldChance, v.TreasureMin, v.TreasureMax = 101, 37, 0
				a.Values[id] = v
			})
			if err != nil {
				t.Fatal(err)
			}
			for range 64 {
				sim.Step(cold.live.world, nil)
			}
			if g := totalSackGold(cold.live.world); g != 37 {
				t.Fatalf("saved corpse flag allowed another drop: gold=%d", g)
			}
		})
	}
}

// An old-style SAV without CorpseDecayed holding dead units at -20 (stage 3)
// and -40 (stage 4) loads in both games with CorpseDecayed derived from Stage.
func TestSAVOldSaveDeadUnitsAtMinusTwentyAndMinusFortyLoad(t *testing.T) {
	for _, second := range []bool{false, true} {
		name := "first"
		if second {
			name = "second"
		}
		t.Run(name, func(t *testing.T) {
			dead20 := sim.Entity{
				ID: 101, X: 14, Y: 14, HP: -20, MaxHP: 50, Class: 35, TypeID: 0x41,
				Owner: 2, TokenSize: 1, Decay: 3, GoldChance: 101, TreasureMin: 37,
			}
			dead40 := sim.Entity{
				ID: 102, X: 16, Y: 14, HP: -40, MaxHP: 50, Class: 35, TypeID: 0x41,
				Owner: 2, TokenSize: 1, Decay: 4, GoldChance: 101, TreasureMin: 53,
			}
			f := corpseDecayedFixtureFront(t, second, dead20, dead40)
			_, cold, err := exportAndReloadCorpseSAV(t, f, second, func(a *currentActionData) {
				for id, v := range a.Values {
					v.CorpseDecayed = false
					a.Values[id] = v
				}
			})
			if err != nil {
				t.Fatalf("old-style SAV load failed: %v", err)
			}
			e20, ok20 := cold.live.world.Entity(101)
			e40, ok40 := cold.live.world.Entity(102)
			if !ok20 || e20.HP != -20 || e20.Decay != 3 || !e20.CorpseDecayed {
				t.Fatalf("dead -20 actor: ok=%v hp=%d decay=%d decayed=%v, want -20/3/true", ok20, e20.HP, e20.Decay, e20.CorpseDecayed)
			}
			if !ok40 || e40.HP != -40 || e40.Decay != 4 || !e40.CorpseDecayed {
				t.Fatalf("dead -40 actor: ok=%v hp=%d decay=%d decayed=%v, want -40/4/true", ok40, e40.HP, e40.Decay, e40.CorpseDecayed)
			}
			for range 64 {
				sim.Step(cold.live.world, nil)
			}
			if g := totalSackGold(cold.live.world); g != 0 {
				t.Errorf("old-save corpses dropped loot on continuation: gold=%d", g)
			}
		})
	}
}

// A corrupt SAV naming CorpseDecayed on a living actor is refused on load.
func TestSAVCorruptCorpseDecayedOnLivingActorRefused(t *testing.T) {
	creature := sim.Entity{ID: 101, X: 14, Y: 14, HP: 10, MaxHP: 10, Class: 35, TypeID: 0x41, Owner: 2, TokenSize: 1}
	f := corpseDecayedFixtureFront(t, false, creature)
	_, _, err := exportAndReloadCorpseSAV(t, f, false, func(a *currentActionData) {
		v := a.Values[creature.ID]
		v.CorpseDecayed = true
		a.Values[creature.ID] = v
	})
	if err == nil {
		t.Fatal("accepted CorpseDecayed=true on a living actor in SAV")
	}
}
