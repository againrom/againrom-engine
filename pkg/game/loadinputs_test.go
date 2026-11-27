package game

import (
	"reflect"
	"slices"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func loadInputsTable(t *testing.T) (*mapload.Table, sim.GhostTemplate) {
	t.Helper()
	defs := eqDefsTable(t)
	base := actorRegistryTable()
	units := base.Units.(dbCollection)
	creatureParams := append([]int32(nil), units[1].params...)
	creatureParams[4], creatureParams[29], creatureParams[30], creatureParams[31] = 25, 1, 1, 1
	creatureRow := dbEntry{name: units[1].name, params: creatureParams}
	ghostParams := make([]int32, 41)
	for i := range ghostParams {
		ghostParams[i] = -1
	}
	ghostParams[4], ghostParams[8], ghostParams[9], ghostParams[10] = 25, 10, 16, 6
	ghostParams[29], ghostParams[31], ghostParams[32], ghostParams[33] = 69, 1, 2, 12
	ghostRow := dbEntry{name: "Ghost", params: ghostParams}
	table := &mapload.Table{
		Units:     dbCollection{creatureRow, creatureRow, ghostRow},
		Humans:    append(fourBaseHumans(), dbEntry{name: "actor runtime", params: humansParams(50, 0, 5)}),
		Shapes:    defs.Shapes,
		Materials: defs.Materials,
		Armors:    defs.Armors,
		Shields:   defs.Shields,
		Weapons:   defs.Weapons,
		FlightInputs: &sim.FlightInputs{
			Pictures: map[int32]sim.FlightPicture{
				13: {Present: true, Phases: 11},
				20: {Present: true, Phases: 4, Homes: true, LifeTicks: 2},
			},
			Classes: map[int32]sim.FlightClass{
				1: {Projectile: 20, ShootDelay: 2, TileSize: 1},
			},
			ClassRule: flightClassRule,
		},
	}
	probe, err := mapload.FromALMWith(&alm.Map{Width: 4, Height: 4}, table, mapload.DifficultyNormal)
	if err != nil {
		t.Fatal(err)
	}
	ghost := probe.Ghost()
	if !ghost.Raisable() {
		t.Fatalf("fixture Ghost row did not produce a raisable template: %+v", ghost)
	}
	return table, ghost
}

func loadInputsFront(t *testing.T, ids ...uint16) (*FrontEnd, sim.GhostTemplate) {
	t.Helper()
	units := make([]synth.ALMUnit, len(ids))
	for i, id := range ids {
		units[i] = synth.ALMUnit{X: uint32(15+i)<<8 | 128, Y: 16<<8 | 128, ClassID: 1, ClassSubID: 1, UnitID: id}
	}
	f := poolFixtureFrontMap(t, synth.ALM(synth.ALMOptions{Width: 40, Height: 40, Units: units}))
	table, ghost := loadInputsTable(t)
	f.Table = table
	f.Humans = fourBaseHumans()
	f.Campaign = resolved(Campaign{Main: []int{10}, Offered: []int{10}, Chapters: map[int]Chapter{10: {Mission: 10}}}, nil)
	return f, ghost
}

type mergeOperandSummary struct {
	code, weight, class uint64
}

func worldMergeOperands(w *sim.World) []mergeOperandSummary {
	v := reflect.ValueOf(w).Elem().FieldByName("mergeOperands")
	out := make([]mergeOperandSummary, v.Len())
	for i := range v.Len() {
		row := v.Index(i)
		out[i] = mergeOperandSummary{
			code:   row.FieldByName("Code").Uint(),
			weight: uint64(row.FieldByName("Weight").Int()),
			class:  row.FieldByName("Constructor").FieldByName("Class").Uint(),
		}
	}
	return out
}

func worldBurstPhases(w *sim.World) uint64 {
	return reflect.ValueOf(w).Elem().FieldByName("burst").FieldByName("phases").Uint()
}

func assertEffectiveMerge(t *testing.T, w *sim.World, id sim.EntityID, table *mapload.Table) {
	t.Helper()
	plain := sim.PlainItem(eqShieldCode)
	built := mapload.SourceConstructedItem(plain, table)
	if built.SourceEquipment.Class == 0 {
		t.Fatal("fixture eqShieldCode has no source constructor")
	}
	if !w.ReplaceStock(sim.Stock{ID: id, ItemInstances: []sim.ItemInstance{built, plain}}) {
		t.Fatal("ReplaceStock refused test stacks")
	}
	before, ok := w.CarriedStacks(id)
	if !ok || len(before) != 2 {
		t.Fatalf("pre-merge stacks = %+v (ok=%v), want 2", before, ok)
	}
	sim.Step(w, []sim.Command{sim.RearrangeCarried(id, 1, 0, true)})
	after, _ := w.CarriedStacks(id)
	if len(after) != 1 || after[0].Count != 2 {
		t.Errorf("mergeOperands missing: RearrangeCarried did not merge plain and source-constructed stacks: %+v", after)
	}
}

func TestHoldingsStagingCarriesCreatureInputs(t *testing.T) {
	table, ghost := loadInputsTable(t)
	w, err := sim.NewStructuredWorld(1, sim.Bounds{Width: 40, Height: 40}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{
			{ID: 1, MapUnitID: 91, X: 5, Y: 6, HP: 20, MaxHP: 20},
			{ID: 2, MapUnitID: 92, X: 6, Y: 6, HP: 20, MaxHP: 20},
		}, nil, sim.Relations{}, nil, nil, nil, ghost, nil)
	if err != nil {
		t.Fatal(err)
	}
	mapload.DeclareCodeWeights(w, table, []uint16{eqShieldCode})
	mapload.BindFlightInputs(w, table)
	w.SetLoadedRecord(sim.LoadedDocument)
	wantMerge := worldMergeOperands(w)
	if len(wantMerge) == 0 {
		t.Fatal("fixture world has no mergeOperands before staging")
	}
	ms := &Mission{Map: &alm.Map{Width: 40, Height: 40}, World: w}

	var report OriginalSaveResume
	if err := restoreOriginalActorStock(ms, nil, table, &report); err != nil {
		t.Fatal(err)
	}
	if got := ms.World.Ghost(); got != ghost {
		t.Errorf("ghost after holdings staging = %+v, want %+v", got, ghost)
	}
	if got := worldMergeOperands(ms.World); !slices.Equal(got, wantMerge) {
		t.Errorf("mergeOperands after holdings staging = %+v, want %+v", got, wantMerge)
	}
	if got := worldBurstPhases(ms.World); got != 11 {
		t.Errorf("burst.phases after holdings staging = %d, want 11", got)
	}
	if !ms.World.HasFlightInputs() {
		t.Errorf("flightInputs dropped after holdings staging")
	}
	if got := ms.World.LoadedRecord(); got != sim.LoadedDocument {
		t.Errorf("loadedRecord after holdings staging = %v, want %v", got, sim.LoadedDocument)
	}
	assertEffectiveMerge(t, ms.World, 1, table)
}

func TestLoadSAVPreservesCreatureInputs(t *testing.T) {
	f, wantGhost := loadInputsFront(t, 91, 92)
	party := f.NextParty()
	if len(party) == 0 {
		t.Fatal("fixture has no fresh party")
	}
	party[0].Carry = nil
	party[0].Saved = &mapload.Saved{Cell: mapload.Cell{X: 15, Y: 15}, HP: 50, MaxHP: 50}
	party[0].CarriedItems = []sim.ItemInstance{sim.PlainItem(eqShieldCode)}
	party[0].Carried = []uint16{eqShieldCode}
	if err := f.App("start").OpenMission(f.MissionOpenerWith(10, party)); err != nil {
		t.Fatal(err)
	}
	startWorld := f.live.world
	heroID := f.live.mission.state.Start.IDs[0]
	sim.Step(startWorld, []sim.Command{sim.DropCarriedStack(heroID, 0, sim.CellPoint{X: 15, Y: 15})})
	if len(startWorld.Sacks()) == 0 {
		t.Fatal("dropping eqShieldCode produced no ground sack")
	}
	wantMerge := worldMergeOperands(startWorld)
	if len(wantMerge) == 0 {
		t.Fatal("mission start lacks mergeOperands")
	}
	if startWorld.Ghost() != wantGhost {
		t.Fatalf("mission start ghost = %+v, want %+v", startWorld.Ghost(), wantGhost)
	}
	if !startWorld.HasFlightInputs() {
		t.Fatal("mission start lacks flightInputs")
	}
	if phases := worldBurstPhases(startWorld); phases != 11 {
		t.Fatalf("mission start burst.phases = %d, want 11", phases)
	}

	snap, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	engineSAV, err := f.ExportCurrentSave(snap, label)
	if err != nil {
		t.Fatal(err)
	}

	doc, err := sav.DecodeDocumentData(engineSAV)
	if err != nil {
		t.Fatal(err)
	}
	doc.State.ValueRecords = slices.DeleteFunc(doc.State.ValueRecords, func(r sav.CityStateRecordData) bool {
		return r.Path == sav.NativeActionsPath
	})
	for i := range doc.Objects {
		if r := &doc.Objects[i]; r.Class == "Unit" || r.Class == "Human" || r.Class == "Humanoid" {
			if v, err := savedStructureValue(r, "U49"); err == nil && v == 0 {
				mustSetValue(r, "U49", 1)
			}
		}
	}
	origSAV, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}

	checkLoaded := func(t *testing.T, loaded *FrontEnd, wantRecord sim.LoadedRecord, checkMerge bool) {
		t.Helper()
		w := loaded.live.world
		if got := w.Ghost(); got != wantGhost {
			t.Errorf("ghost after LOAD = %+v, want %+v", got, wantGhost)
		}
		if checkMerge {
			if got := worldMergeOperands(w); !slices.Equal(got, wantMerge) {
				t.Errorf("mergeOperands after LOAD = %+v, want %+v", got, wantMerge)
			}
		}
		if phases := worldBurstPhases(w); phases != 11 {
			t.Errorf("burst.phases after LOAD = %d, want 11", phases)
		}
		if !w.HasFlightInputs() {
			t.Errorf("flightInputs dropped after LOAD")
		}
		if got := w.LoadedRecord(); got != wantRecord {
			t.Errorf("loadedRecord after LOAD = %v, want %v", got, wantRecord)
		}
	}

	t.Run("original_sav", func(t *testing.T) {
		fresh, _ := loadInputsFront(t, 91, 92)
		loaded, err := loadOriginalFront(fresh, origSAV)
		if err != nil {
			t.Fatal(err)
		}
		checkLoaded(t, loaded, sim.LoadedDocument, true)
	})

	t.Run("engine_sav", func(t *testing.T) {
		fresh, _ := loadInputsFront(t, 91, 92)
		loaded, err := loadOriginalFront(fresh, engineSAV)
		if err != nil {
			t.Fatal(err)
		}
		checkLoaded(t, loaded, sim.LoadedDocument, true)
	})

	t.Run("partial_original_sav", func(t *testing.T) {
		a := &poolFixtureActor{mapID: 91, cell: 0x0605, hp: 7, maxHP: 31, holdings: &holdingFixture{}}
		b := &poolFixtureActor{mapID: 92, cell: 0x0606, hp: 9, maxHP: 41, holdings: &holdingFixture{}}
		fresh, _ := loadInputsFront(t, 91, 92)
		loaded, err := loadOriginalFront(fresh, poolFixtureSave(a, b))
		if err != nil {
			t.Fatal(err)
		}
		checkLoaded(t, loaded, sim.LoadedWithoutDocument, false)
	})
}
